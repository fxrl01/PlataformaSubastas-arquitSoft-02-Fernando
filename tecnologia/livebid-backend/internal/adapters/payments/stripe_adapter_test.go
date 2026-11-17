package payments_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
)

func TestStripeAdapterCreatesVerifiesCapturesAndCancelsManualHoldIntent(t *testing.T) {
	adapter := payments.NewStripeAdapter("", "pk_test_demo", "whsec_test_secret")

	intent, err := adapter.CreateManualHoldIntent(context.Background(), "SUB-2026-001", "USR-VIP-01", 4500, "PEN")
	if err != nil {
		t.Fatalf("create hold intent failed: %v", err)
	}
	if intent.PaymentIntentID == "" || intent.ClientSecret == "" || intent.Status != "requires_capture" {
		t.Fatalf("unexpected intent: %+v", intent)
	}

	status, err := adapter.VerifyHoldIntent(context.Background(), intent.PaymentIntentID)
	if err != nil || status != "requires_capture" {
		t.Fatalf("verify hold intent failed: status=%s err=%v", status, err)
	}

	if err := adapter.CaptureHoldIntent(context.Background(), intent.PaymentIntentID); err != nil {
		t.Fatalf("capture hold intent failed: %v", err)
	}

	statusAfterCapture, err := adapter.VerifyHoldIntent(context.Background(), intent.PaymentIntentID)
	if err != nil || statusAfterCapture != "succeeded" {
		t.Fatalf("expected succeeded after capture, got status=%s err=%v", statusAfterCapture, err)
	}

	losingIntent, _ := adapter.CreateManualHoldIntent(context.Background(), "SUB-2026-001", "USR-LOSER", 4500, "PEN")
	if err := adapter.CancelHoldIntent(context.Background(), losingIntent.PaymentIntentID); err != nil {
		t.Fatalf("cancel hold intent failed: %v", err)
	}
	if _, err := adapter.VerifyHoldIntent(context.Background(), losingIntent.PaymentIntentID); err == nil {
		t.Fatal("expected canceled intent to fail verification as active hold")
	}
}

func TestStripeAdapterVerifiesWebhookHMACSignature(t *testing.T) {
	secret := "whsec_livebid_test_secret"
	adapter := payments.NewStripeAdapter("", "pk_test_demo", secret)

	payload := []byte(`{"id":"evt_1","type":"payment_intent.amount_capturable_updated","data":{"object":{"id":"pi_stripe_hold_123"}}}`)
	timestamp := "1730900000"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	header := "t=" + timestamp + ",v1=" + sig
	if err := adapter.VerifyWebhookSignature(payload, header); err != nil {
		t.Fatalf("expected valid webhook signature, got: %v", err)
	}

	if err := adapter.VerifyWebhookSignature(payload, "t="+timestamp+",v1=deadbeef"); err == nil {
		t.Fatal("expected tampered webhook signature to be rejected")
	}
}

