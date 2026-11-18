package services_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	syncedAuthID     string
	syncedStatus     domain.GuaranteeStatus
}

func (r *mockGuaranteeRepository) FindActive(context.Context, string, string) (*domain.Guarantee, error) {
	if r.active == nil || (r.active.Status != "" && r.active.Status != domain.GuaranteeHeld) {
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

func (r *mockGuaranteeRepository) UpdateStatusByAuthorizationID(_ context.Context, authorizationID string, status domain.GuaranteeStatus) error {
	r.syncedAuthID = authorizationID
	r.syncedStatus = status
	if r.active != nil && r.active.AuthorizationID == authorizationID {
		r.active.Status = status
	}
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

func signStripePayload(secret string, payload []byte) string {
	ts := "1731888000"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	return fmt.Sprintf("t=%s,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestStripeWebhookSynchronizesGuaranteeStatus(t *testing.T) {
	now := time.Now()
	auction := domain.NuevaSubasta("sub-webhook-1", "Reloj Omega", "", 5000, now, now.Add(time.Hour))
	guarantees := &mockGuaranteeRepository{}
	webhookSecret := "whsec_webhook_sync_secret"
	stripeAdapter := payments.NewStripeAdapter("", "pk_test_demo", webhookSecret)
	service := services.NewGuaranteeService(
		&MockRepo{subastas: map[string]*domain.Subasta{"sub-webhook-1": auction}}, guarantees,
	).WithPaymentGateway(stripeAdapter)

	offer, err := service.PrepareStripeIntent(context.Background(), "sub-webhook-1", "postor-wh", 0)
	if err != nil {
		t.Fatalf("PrepareStripeIntent failed: %v", err)
	}
	if _, err := service.Create(context.Background(), services.CreateGuaranteeCmd{
		AuctionID:       "sub-webhook-1",
		BidderID:        "postor-wh",
		Amount:          offer.HoldAmount,
		PaymentIntentID: offer.PaymentIntentID,
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 1. Evento asíncrono de cancelación / fallo revoca la garantía a LIBERADA
	cancelPayload := []byte(fmt.Sprintf(`{"id":"evt_cancel","type":"payment_intent.canceled","data":{"object":{"id":"%s","status":"canceled"}}}`, offer.PaymentIntentID))
	eventType, err := service.HandleStripeWebhook(context.Background(), cancelPayload, signStripePayload(webhookSecret, cancelPayload))
	if err != nil {
		t.Fatalf("HandleStripeWebhook canceled failed: %v", err)
	}
	if eventType != "payment_intent.canceled" {
		t.Fatalf("expected payment_intent.canceled, got %s", eventType)
	}
	if guarantees.syncedAuthID != offer.PaymentIntentID || guarantees.syncedStatus != domain.GuaranteeReleased {
		t.Fatalf("expected guarantee status LIBERADA on cancel webhook, got auth=%s status=%s", guarantees.syncedAuthID, guarantees.syncedStatus)
	}

	// 2. Evento asíncrono de captura exitosa actualiza la garantía a CAPTURADA_COBRO
	succeedPayload := []byte(fmt.Sprintf(`{"id":"evt_succ","type":"payment_intent.succeeded","data":{"object":{"id":"%s","status":"succeeded"}}}`, offer.PaymentIntentID))
	eventType, err = service.HandleStripeWebhook(context.Background(), succeedPayload, signStripePayload(webhookSecret, succeedPayload))
	if err != nil {
		t.Fatalf("HandleStripeWebhook succeeded failed: %v", err)
	}
	if eventType != "payment_intent.succeeded" || guarantees.syncedStatus != domain.GuaranteeCaptured {
		t.Fatalf("expected CAPTURADA_COBRO on succeeded webhook, got type=%s status=%s", eventType, guarantees.syncedStatus)
	}

	// 3. Firma inválida es rechazada con ErrStripeWebhookInvalid
	if _, err := service.HandleStripeWebhook(context.Background(), succeedPayload, "t=1731888000,v1=invalid"); err != domain.ErrStripeWebhookInvalid {
		t.Fatalf("expected ErrStripeWebhookInvalid, got %v", err)
	}
}
