package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type GuaranteeService struct {
	auctions         domain.SubastaRepository
	guarantees       domain.GuaranteeRepository
	risk             domain.AIRiskEvaluator
	blockchainEscrow domain.BlockchainEscrowPort
	paymentGateway   domain.PaymentGatewayPort
	now              func() time.Time
}

type CreateGuaranteeCmd struct {
	AuctionID       string
	BidderID        string
	Amount          float64
	WalletAddress   string
	PaymentIntentID string
}

type StripeIntentOfferView struct {
	PaymentIntentID string             `json:"payment_intent_id"`
	ClientSecret    string             `json:"client_secret"`
	PublishableKey  string             `json:"publishable_key"`
	AuctionID       string             `json:"subasta_id"`
	BidderID        string             `json:"postor_id"`
	RequiredAmount  float64            `json:"monto_requerido"`
	HoldAmount      float64            `json:"monto_retencion"`
	GuaranteeRatio  float64            `json:"porcentaje_garantia"`
	TrustScore      int                `json:"score_confianza"`
	RiskLevel       domain.NivelRiesgo `json:"nivel_riesgo,omitempty"`
	FallbackMode    bool               `json:"modo_fallback,omitempty"`
	CaptureMethod   string             `json:"capture_method"`
	IntentStatus    string             `json:"estado_intent"`
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

func (s *GuaranteeService) WithPaymentGateway(gateway domain.PaymentGatewayPort) *GuaranteeService {
	s.paymentGateway = gateway
	return s
}

func (s *GuaranteeService) evaluateBidderRequirement(ctx context.Context, auction *domain.Subasta, bidderID string) (required float64, ratio float64, trustScore int, riskLevel domain.NivelRiesgo, fallbackMode bool) {
	trustScore = 75
	ratio = 0.10
	if s.risk != nil {
		if eval, err := s.risk.EvaluarPostor(ctx, bidderID, auction.ID); err == nil {
			trustScore = eval.Score
			if eval.PorcentajeGarantia > 0 {
				ratio = eval.PorcentajeGarantia
			}
			riskLevel = eval.Nivel
			fallbackMode = eval.ModoFallback
		}
	}
	required = domain.RequiredGuaranteeWithRatio(auction.PrecioBase, ratio)
	return
}

func (s *GuaranteeService) PrepareStripeIntent(ctx context.Context, auctionID, bidderID string, requestedAmount float64) (StripeIntentOfferView, error) {
	auction, err := s.auctions.ObtenerPorID(ctx, auctionID)
	if err != nil {
		return StripeIntentOfferView{}, err
	}
	if active, err := s.guarantees.FindActive(ctx, auctionID, bidderID); err == nil && active != nil {
		return StripeIntentOfferView{}, domain.ErrGuaranteeAlreadyActive
	}
	required, ratio, trustScore, riskLevel, fallbackMode := s.evaluateBidderRequirement(ctx, auction, bidderID)
	holdAmount := required
	if requestedAmount > required {
		holdAmount = requestedAmount
	}
	if s.paymentGateway == nil {
		return StripeIntentOfferView{}, domain.ErrStripeHoldFailed
	}
	intent, err := s.paymentGateway.CreateManualHoldIntent(ctx, auctionID, bidderID, holdAmount, "usd")
	if err != nil {
		return StripeIntentOfferView{}, err
	}
	return StripeIntentOfferView{
		PaymentIntentID: intent.PaymentIntentID,
		ClientSecret:    intent.ClientSecret,
		PublishableKey:  intent.PublishableKey,
		AuctionID:       auctionID,
		BidderID:        bidderID,
		RequiredAmount:  required,
		HoldAmount:      holdAmount,
		GuaranteeRatio:  ratio,
		TrustScore:      trustScore,
		RiskLevel:       riskLevel,
		FallbackMode:    fallbackMode,
		CaptureMethod:   "manual",
		IntentStatus:    intent.Status,
	}, nil
}

func (s *GuaranteeService) GetActive(ctx context.Context, auctionID, bidderID string) (GuaranteeView, error) {
	guarantee, err := s.guarantees.FindActive(ctx, auctionID, bidderID)
	if err != nil {
		return GuaranteeView{}, err
	}
	return toGuaranteeView(guarantee), nil
}

func (s *GuaranteeService) HandleStripeWebhook(ctx context.Context, payload []byte, signatureHeader string) (string, error) {
	if s.paymentGateway == nil {
		return "", domain.ErrStripeWebhookInvalid
	}
	if err := s.paymentGateway.VerifyWebhookSignature(payload, signatureHeader); err != nil {
		return "", err
	}
	var event struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &event); err != nil || strings.TrimSpace(event.Type) == "" {
		return "stripe.webhook.verified", nil
	}
	return event.Type, nil
}

func (s *GuaranteeService) Create(ctx context.Context, cmd CreateGuaranteeCmd) (GuaranteeView, error) {
	auction, err := s.auctions.ObtenerPorID(ctx, cmd.AuctionID)
	if err != nil {
		return GuaranteeView{}, err
	}
	required, _, trustScore, riskLevel, fallbackMode := s.evaluateBidderRequirement(ctx, auction, cmd.BidderID)
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
	var authorization string
	trimmedIntent := strings.TrimSpace(cmd.PaymentIntentID)
	if s.paymentGateway != nil && trimmedIntent != "" {
		if _, err := s.paymentGateway.VerifyHoldIntent(ctx, trimmedIntent); err != nil {
			return GuaranteeView{}, err
		}
		authorization = trimmedIntent
	} else if s.paymentGateway != nil {
		intent, err := s.paymentGateway.CreateManualHoldIntent(ctx, cmd.AuctionID, cmd.BidderID, cmd.Amount, "usd")
		if err != nil {
			return GuaranteeView{}, err
		}
		authorization = intent.PaymentIntentID
	} else if trimmedIntent != "" {
		authorization = trimmedIntent
	} else {
		authorization, err = newID("auth_")
		if err != nil {
			return GuaranteeView{}, err
		}
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
