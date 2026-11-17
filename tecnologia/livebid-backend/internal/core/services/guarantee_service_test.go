package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

type mockGuaranteeRepository struct {
	active           *domain.Guarantee
	saved            *domain.Guarantee
	settledAuctionID string
	settledWinnerID  string
}

func (r *mockGuaranteeRepository) FindActive(context.Context, string, string) (*domain.Guarantee, error) {
	if r.active == nil {
		return nil, domain.ErrUserNotFound
	}
	return r.active, nil
}

func (r *mockGuaranteeRepository) Save(_ context.Context, guarantee *domain.Guarantee) error {
	r.saved = guarantee
	r.active = guarantee
	return nil
}

func (r *mockGuaranteeRepository) SettleAuction(_ context.Context, auctionID, winnerID string) error {
	r.settledAuctionID = auctionID
	r.settledWinnerID = winnerID
	return nil
}

func TestGuaranteeRequiresTenPercentOfBasePrice(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{}
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, guarantees,
	)

	view, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "usr-1", Amount: 250,
	})
	if err != nil {
		t.Fatalf("create guarantee failed: %v", err)
	}
	if view.Amount != 250 || view.Status != string(domain.GuaranteeHeld) {
		t.Fatalf("unexpected guarantee: %+v", view)
	}
}

func TestGuaranteeRejectsDuplicateActiveGuarantee(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{active: &domain.Guarantee{ID: "gar-1"}}
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, guarantees,
	)

	_, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "usr-1", Amount: 250,
	})
	if err != domain.ErrGuaranteeAlreadyActive {
		t.Fatalf("expected duplicate guarantee error, got %v", err)
	}
}

type mockAIRiskEvaluator struct {
	eval domain.EvaluacionRiesgo
}

func (m mockAIRiskEvaluator) EvaluarRiesgo(context.Context, string, string, float64) (domain.NivelRiesgo, error) {
	return m.eval.Nivel, nil
}

func (m mockAIRiskEvaluator) EvaluarPostor(context.Context, string, string) (domain.EvaluacionRiesgo, error) {
	return m.eval, nil
}

func TestGuaranteeRequiresTwentyPercentForHighRiskBidder(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-1", "Laptop", "", 2500, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{}
	risk := mockAIRiskEvaluator{eval: domain.EvaluacionRiesgo{
		Score:              38,
		Nivel:              domain.RiesgoAlto,
		PorcentajeGarantia: 0.20,
	}}
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-1": auction}}, guarantees, risk,
	)

	// 10% (250) must be rejected for high risk bidder (requires 20% = 500)
	if _, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "USR-RIESGO_ALTO", Amount: 250,
	}); err != domain.ErrGuaranteeInsufficient {
		t.Fatalf("expected ErrGuaranteeInsufficient for 10%% on high-risk bidder, got %v", err)
	}

	view, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID: "sub-1", BidderID: "USR-RIESGO_ALTO", Amount: 500,
	})
	if err != nil {
		t.Fatalf("expected 20%% guarantee to succeed, got %v", err)
	}
	if view.TrustScore != 38 || view.RiskLevel != domain.RiesgoAlto {
		t.Fatalf("expected AI trust score 38 and HIGH risk, got %+v", view)
	}
}

func TestStripeIntentAndGuaranteeConfirmationFlow(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-stripe-1", "MacBook Pro M3", "", 3000, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{}
	stripeAdapter := payments.NewStripeAdapter("", "pk_test_livebid_demo", "whsec_demo")
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-stripe-1": auction}}, guarantees,
	).WithPaymentGateway(stripeAdapter)

	offer, err := service.PrepareStripeIntent(context.Background(), "sub-stripe-1", "postor-01", 0)
	if err != nil {
		t.Fatalf("PrepareStripeIntent failed: %v", err)
	}
	if offer.RequiredAmount != 300 || offer.CaptureMethod != "manual" {
		t.Fatalf("expected required amount 300 and manual capture, got %+v", offer)
	}
	if !strings.HasPrefix(offer.PaymentIntentID, "pi_stripe_hold_") {
		t.Fatalf("expected pi_stripe_hold_ prefix, got %s", offer.PaymentIntentID)
	}

	view, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID:       "sub-stripe-1",
		BidderID:        "postor-01",
		Amount:          offer.HoldAmount,
		PaymentIntentID: offer.PaymentIntentID,
	})
	if err != nil {
		t.Fatalf("Create with Stripe PaymentIntentID failed: %v", err)
	}
	if view.AuthorizationID != offer.PaymentIntentID {
		t.Fatalf("expected AuthorizationID %s, got %s", offer.PaymentIntentID, view.AuthorizationID)
	}
	if view.Status != string(domain.GuaranteeHeld) {
		t.Fatalf("expected status RETENIDA, got %s", view.Status)
	}
}
