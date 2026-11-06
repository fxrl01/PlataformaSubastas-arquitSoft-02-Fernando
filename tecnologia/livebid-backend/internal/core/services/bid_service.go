package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type MandateValidatorFunc func(mandateID, auctionID, bidderID string, amount float64, now time.Time) error

type AuctionBidService struct {
	auctions         domain.SubastaRepository
	bids             domain.BidRepository
	lock             domain.BidLockPort
	notifier         domain.BidNotifierPort
	guarantees       domain.GuaranteeRepository
	web3Verifier     domain.Web3SignatureVerifierPort
	fraudTelemetry   domain.FraudTelemetryPort
	mandateValidator MandateValidatorFunc
	now              func() time.Time
}

type PlaceBidCmd struct {
	SubastaID      string
	PostorID       string
	Monto          float64
	IdempotencyKey string
	WalletAddress  string
	Signature      string
	Nonce          string
	Deadline       int64
	AgentMandateID string
}

type BidResponse struct {
	PujaID          string               `json:"puja_id"`
	SubastaID       string               `json:"subasta_id"`
	PostorLiderID   string               `json:"postor_lider_id"`
	PrecioActual    float64              `json:"precio_actual"`
	TiempoExtendido bool                 `json:"tiempo_extendido"`
	FechaCierre     time.Time            `json:"fecha_cierre"`
	Estado          domain.EstadoSubasta `json:"estado"`
	HashAuditoria   string               `json:"hash_auditoria,omitempty"`
	EIP712Digest    string               `json:"eip712_digest,omitempty"`
	AgentMandateID  string               `json:"mandato_agente_id,omitempty"`
}

func NewAuctionBidService(auctions domain.SubastaRepository, bids domain.BidRepository, locks ...domain.BidLockPort) *AuctionBidService {
	var lock domain.BidLockPort
	if len(locks) > 0 {
		lock = locks[0]
	}
	return &AuctionBidService{auctions: auctions, bids: bids, lock: lock, now: time.Now}
}

func NewAuctionBidServiceWithNotifier(auctions domain.SubastaRepository, bids domain.BidRepository, lock domain.BidLockPort, notifier domain.BidNotifierPort) *AuctionBidService {
	return &AuctionBidService{auctions: auctions, bids: bids, lock: lock, notifier: notifier, now: time.Now}
}

func NewAuctionBidServiceWithNotifierAndGuarantee(auctions domain.SubastaRepository, bids domain.BidRepository, lock domain.BidLockPort, notifier domain.BidNotifierPort, guarantees domain.GuaranteeRepository) *AuctionBidService {
	return &AuctionBidService{auctions: auctions, bids: bids, lock: lock, notifier: notifier, guarantees: guarantees, now: time.Now}
}

func (s *AuctionBidService) WithWeb3Verifier(verifier domain.Web3SignatureVerifierPort) *AuctionBidService {
	s.web3Verifier = verifier
	return s
}

func (s *AuctionBidService) WithFraudTelemetry(telemetry domain.FraudTelemetryPort) *AuctionBidService {
	s.fraudTelemetry = telemetry
	return s
}

func (s *AuctionBidService) WithMandateValidator(validator MandateValidatorFunc) *AuctionBidService {
	s.mandateValidator = validator
	return s
}

func (s *AuctionBidService) PlaceBid(ctx context.Context, cmd PlaceBidCmd) (BidResponse, error) {
	now := s.now()
	if cmd.AgentMandateID != "" && s.mandateValidator != nil {
		if err := s.mandateValidator(cmd.AgentMandateID, cmd.SubastaID, cmd.PostorID, cmd.Monto, now); err != nil {
			return BidResponse{}, err
		}
	}
	var eip712Digest string
	if cmd.WalletAddress != "" || cmd.Signature != "" {
		if s.web3Verifier != nil {
			env := domain.EIP712BidEnvelope{
				AuctionID:     cmd.SubastaID,
				BidderID:      cmd.PostorID,
				WalletAddress: cmd.WalletAddress,
				Amount:        cmd.Monto,
				Nonce:         cmd.Nonce,
				Deadline:      cmd.Deadline,
				Signature:     cmd.Signature,
			}
			if err := s.web3Verifier.VerifyBidSignature(ctx, env, now); err != nil {
				return BidResponse{}, err
			}
			eip712Digest = s.web3Verifier.ComputeTypedDataDigest(env)
		}
	}

	var unlock domain.UnlockFunc
	var err error
	if s.lock != nil {
		unlock, err = s.lock.AdquirirLock(ctx, cmd.SubastaID, 1500*time.Millisecond)
		if err != nil {
			return BidResponse{}, domain.ErrLockTimeout
		}
		defer func() { _ = unlock() }()
	}
	subasta, err := s.auctions.ObtenerPorID(ctx, cmd.SubastaID)
	if err != nil {
		return BidResponse{}, err
	}
	if s.guarantees != nil {
		if _, err := s.guarantees.FindActive(ctx, cmd.SubastaID, cmd.PostorID); err != nil {
			return BidResponse{}, domain.ErrGarantiaInsuficiente
		}
	}
	if cmd.IdempotencyKey != "" {
		existing, err := s.bids.BuscarPorIdempotencyKey(ctx, cmd.SubastaID, cmd.PostorID, cmd.IdempotencyKey)
		if err != nil {
			return BidResponse{}, err
		}
		if existing != nil {
			if existing.Monto != cmd.Monto {
				return BidResponse{}, domain.ErrIdempotencyKeyConflict
			}
			return BidResponse{
				PujaID: existing.ID, SubastaID: subasta.ID, PostorLiderID: subasta.PostorLiderID,
				PrecioActual: subasta.PrecioActual, FechaCierre: subasta.FechaCierre,
				Estado: subasta.Estado, EIP712Digest: eip712Digest,
			}, nil
		}
	}
	precioAnterior := subasta.PrecioActual
	pujaID, err := newID("bid_")
	if err != nil {
		return BidResponse{}, err
	}
	puja, extended, err := subasta.ValidarYRegistrarPuja(pujaID, cmd.PostorID, cmd.Monto, now)
	if err != nil {
		return BidResponse{}, err
	}
	if err := s.bids.RegistrarPujaAtomica(ctx, subasta.ID, *puja, precioAnterior, subasta.FechaCierre, extended, cmd.IdempotencyKey); err != nil {
		return BidResponse{}, err
	}
	if s.notifier != nil {
		if err := s.notifier.BroadcastNuevaPuja(ctx, subasta.ID, *puja, extended, subasta.FechaCierre); err != nil {
			return BidResponse{}, err
		}
	}
	if s.fraudTelemetry != nil {
		s.fraudTelemetry.PublishBidTelemetry(domain.BidTelemetryEvent{
			AuctionID:     subasta.ID,
			BidID:         puja.ID,
			BidderID:      puja.PostorID,
			Amount:        puja.Monto,
			PreviousPrice: precioAnterior,
			Timestamp:     puja.Timestamp,
			AgentMandate:  cmd.AgentMandateID,
		})
	}
	rawAudit := fmt.Sprintf("%s|%s|%s|%.2f|%d", subasta.ID, puja.ID, puja.PostorID, puja.Monto, puja.Timestamp.UTC().UnixNano())
	auditSum := sha256.Sum256([]byte(rawAudit))
	return BidResponse{
		PujaID: puja.ID, SubastaID: subasta.ID, PostorLiderID: subasta.PostorLiderID,
		PrecioActual: subasta.PrecioActual, TiempoExtendido: extended,
		FechaCierre: subasta.FechaCierre, Estado: subasta.Estado,
		HashAuditoria:  "0x" + hex.EncodeToString(auditSum[:]),
		EIP712Digest:   eip712Digest,
		AgentMandateID: cmd.AgentMandateID,
	}, nil
}

func newID(prefix string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}
