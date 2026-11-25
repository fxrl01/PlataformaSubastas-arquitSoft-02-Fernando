package payments_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

func TestStripeAdapterRemoteHTTPFlow(t *testing.T) {
	status := "requires_capture"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok || user != "sk_test_remote_key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/payment_intents":
			_, _ = w.Write([]byte(`{"id":"pi_remote_999","client_secret":"pi_remote_999_secret_abc","status":"requires_capture","livemode":false}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/payment_intents/pi_remote_999":
			_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/payment_intents/pi_remote_999/capture":
			status = "succeeded"
			_, _ = w.Write([]byte(`{"id":"pi_remote_999","status":"succeeded"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/payment_intents/pi_remote_999/cancel":
			status = "canceled"
			_, _ = w.Write([]byte(`{"id":"pi_remote_999","status":"canceled"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	adapter := payments.NewStripeAdapter("sk_test_remote_key", "pk_test_remote", "").WithBaseURL(srv.URL + "/v1")

	intent, err := adapter.CreateManualHoldIntent(context.Background(), "sub-r1", "bid-r1", 1200, "usd")
	if err != nil || intent.PaymentIntentID != "pi_remote_999" || intent.Currency != "USD" {
		t.Fatalf("unexpected remote intent: %+v err=%v", intent, err)
	}

	gotStatus, err := adapter.VerifyHoldIntent(context.Background(), "pi_remote_999")
	if err != nil || gotStatus != "requires_capture" {
		t.Fatalf("verify remote intent failed: status=%s err=%v", gotStatus, err)
	}

	if err := adapter.CaptureHoldIntent(context.Background(), "pi_remote_999"); err != nil {
		t.Fatalf("capture remote intent failed: %v", err)
	}
	if err := adapter.CancelHoldIntent(context.Background(), "pi_remote_999"); err != nil {
		t.Fatalf("cancel remote intent failed: %v", err)
	}
	if _, err := adapter.VerifyHoldIntent(context.Background(), "pi_remote_999"); err != domain.ErrStripeHoldFailed {
		t.Fatalf("expected ErrStripeHoldFailed for canceled remote intent, got %v", err)
	}
	if _, err := adapter.CreateManualHoldIntent(context.Background(), "", "b", 0, ""); err != domain.ErrGuaranteeInvalid {
		t.Fatalf("expected ErrGuaranteeInvalid, got %v", err)
	}
	if err := adapter.VerifyWebhookSignature([]byte(`{"type":"ping"}`), "t=123"); err != nil {
		t.Fatalf("expected local dev webhook header fallback to pass, got %v", err)
	}
	if err := adapter.VerifyWebhookSignature(nil, ""); err != domain.ErrStripeWebhookInvalid {
		t.Fatalf("expected ErrStripeWebhookInvalid on empty payload, got %v", err)
	}
	if !strings.HasPrefix(intent.ClientSecret, "pi_remote_999_secret_") {
		t.Fatalf("unexpected client secret: %s", intent.ClientSecret)
	}

	// Error & fallback branches
	defaultAdapter := payments.NewStripeAdapter("replace-me", "replace-me", "whsec_live_1")
	if _, err := defaultAdapter.CreateManualHoldIntent(context.Background(), "sub-d", "bid-d", 100, ""); err != nil {
		t.Fatalf("expected default currency sandbox intent to succeed, got %v", err)
	}
	if _, err := defaultAdapter.VerifyHoldIntent(context.Background(), "bad"); err != domain.ErrStripeHoldFailed {
		t.Fatalf("expected ErrStripeHoldFailed for bad pi_ id, got %v", err)
	}
	if err := defaultAdapter.CaptureHoldIntent(context.Background(), ""); err != domain.ErrGuaranteeInvalid {
		t.Fatalf("expected ErrGuaranteeInvalid on empty capture id, got %v", err)
	}
	if err := defaultAdapter.CancelHoldIntent(context.Background(), ""); err != domain.ErrGuaranteeInvalid {
		t.Fatalf("expected ErrGuaranteeInvalid on empty cancel id, got %v", err)
	}
	if err := defaultAdapter.VerifyWebhookSignature([]byte(`{"type":"ping"}`), "invalid_format"); err != domain.ErrStripeWebhookInvalid {
		t.Fatalf("expected ErrStripeWebhookInvalid on malformed signature header, got %v", err)
	}

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer errSrv.Close()
	errAdapter := payments.NewStripeAdapter("sk_test_remote_key", "", "").WithBaseURL(errSrv.URL + "/v1")
	if _, err := errAdapter.CreateManualHoldIntent(context.Background(), "s1", "b1", 100, "usd"); err == nil {
		t.Fatal("expected error from 502 remote stripe create")
	}
	if _, err := errAdapter.VerifyHoldIntent(context.Background(), "pi_remote_502"); err != domain.ErrStripeHoldFailed {
		t.Fatalf("expected ErrStripeHoldFailed on 502 verify, got %v", err)
	}
	if err := errAdapter.CaptureHoldIntent(context.Background(), "pi_remote_502"); err != domain.ErrStripeHoldFailed {
		t.Fatalf("expected ErrStripeHoldFailed on 502 capture, got %v", err)
	}
	if err := errAdapter.VerifyWebhookSignature([]byte(`{"type":"ping"}`), ""); err != domain.ErrStripeWebhookInvalid {
		t.Fatalf("expected ErrStripeWebhookInvalid on empty header in dev mode, got %v", err)
	}
}
