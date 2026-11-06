package services

import (
	"context"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type GuaranteeService struct {
	auctions         domain.SubastaRepository
	guarantees       domain.GuaranteeRepository
	risk             domain.AIRiskEvaluator
	blockchainEscrow domain.BlockchainEscrowPort
	now              func() time.Time
}

type CreateGuaranteeCmd struct {
	AuctionID     string
	BidderID      string
	Amount        float64
	WalletAddress string
}

type GuaranteeView struct {
	ID              string             `json:"id"`
	AuctionID       string             `json:"subasta_id"`
	BidderID        string             `json:"postor_id"`
	Amount          float64            `json:"monto"`
	AuthorizationID string             `json:"autorizacion_id"`
	Status          string             `json:"estado"`
	TrustScore      int                `json:"score_confianza"`
	RiskLevel       domain.NivelRiesgo `json:"nivel_riesgo,omitempty"`
	FallbackMode    bool               `json:"modo_fallback,omitempty"`
	OnChainTxHash   string             `json:"tx_hash_onchain,omitempty"`
}

func NewGuaranteeService(auctions domain.SubastaRepository, guarantees domain.GuaranteeRepository, risk ...domain.AIRiskEvaluator) *GuaranteeService {
	var evaluator domain.AIRiskEvaluator
	if len(risk) > 0 {
		evaluator = risk[0]
	}
	return &GuaranteeService{auctions: auctions, guarantees: guarantees, risk: evaluator, now: time.Now}
}

func (s *GuaranteeService) WithBlockchainEscrow(escrow domain.BlockchainEscrowPort) *GuaranteeService {
	s.blockchainEscrow = escrow
	return s
}

func (s *GuaranteeService) Create(ctx context.Context, cmd CreateGuaranteeCmd) (GuaranteeView, error) {
	auction, err := s.auctions.ObtenerPorID(ctx, cmd.AuctionID)
	if err != nil {
		return GuaranteeView{}, err
	}
	trustScore := 75
	ratio := 0.10
	var riskLevel domain.NivelRiesgo
	var fallbackMode bool
	if s.risk != nil {
		if eval, err := s.risk.EvaluarPostor(ctx, cmd.BidderID, cmd.AuctionID); err == nil {
			trustScore = eval.Score
			if eval.PorcentajeGarantia > 0 {
				ratio = eval.PorcentajeGarantia
			}
			riskLevel = eval.Nivel
			fallbackMode = eval.ModoFallback
		}
	}
	required := domain.RequiredGuaranteeWithRatio(auction.PrecioBase, ratio)
	if cmd.Amount < required {
		return GuaranteeView{}, domain.ErrGuaranteeInsufficient
	}
	if active, err := s.guarantees.FindActive(ctx, cmd.AuctionID, cmd.BidderID); err == nil && active != nil {
		return GuaranteeView{}, domain.ErrGuaranteeAlreadyActive
	}
	var onChainTxHash string
	if s.blockchainEscrow != nil {
		txHash, err := s.blockchainEscrow.LockCollateralOnChain(ctx, cmd.AuctionID, cmd.BidderID, cmd.WalletAddress, cmd.Amount)
		if err != nil {
			return GuaranteeView{}, err
		}
		onChainTxHash = txHash
	}
	id, err := newID("gar_")
	if err != nil {
		return GuaranteeView{}, err
	}
	authorization, err := newID("auth_")
	if err != nil {
		return GuaranteeView{}, err
	}
	guarantee := &domain.Guarantee{
		ID: id, AuctionID: cmd.AuctionID, BidderID: cmd.BidderID,
		Amount: cmd.Amount, AuthorizationID: authorization,
		Status: domain.GuaranteeHeld, TrustScore: trustScore, CreatedAt: s.now(),
	}
	if err := s.guarantees.Save(ctx, guarantee); err != nil {
		return GuaranteeView{}, err
	}
	view := toGuaranteeView(guarantee)
	view.RiskLevel = riskLevel
	view.FallbackMode = fallbackMode
	view.OnChainTxHash = onChainTxHash
	return view, nil
}

func toGuaranteeView(g *domain.Guarantee) GuaranteeView {
	return GuaranteeView{ID: g.ID, AuctionID: g.AuctionID, BidderID: g.BidderID,
		Amount: g.Amount, AuthorizationID: g.AuthorizationID, Status: string(g.Status),
		TrustScore: g.TrustScore}
}
