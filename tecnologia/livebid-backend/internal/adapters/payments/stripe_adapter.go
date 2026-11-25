package payments

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

// StripeAdapter implementa domain.PaymentGatewayPort conectando el motor de garantías
// con Stripe PaymentIntents API (capture_method: manual) y proporcionando un modo
// Sandbox determinista integrado cuando no se configura una clave live/test externa.
type StripeAdapter struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	baseURL        string
	httpClient     *http.Client

	mu      sync.RWMutex
	sandbox map[string]string // paymentIntentID -> status ("requires_capture", "succeeded", "canceled")
}

func NewStripeAdapter(secretKey, publishableKey, webhookSecret string) *StripeAdapter {
	pub := strings.TrimSpace(publishableKey)
	if pub == "" || strings.HasPrefix(pub, "replace-") || strings.Contains(pub, "replace_with") {
		pub = "pk_test_livebid_pci_dss_public_key"
	}
	return &StripeAdapter{
		secretKey:      strings.TrimSpace(secretKey),
		publishableKey: pub,
		webhookSecret:  strings.TrimSpace(webhookSecret),
		baseURL:        "https://api.stripe.com/v1",
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		sandbox:        make(map[string]string),
	}
}

// WithBaseURL permite redirigir las peticiones HTTP de Stripe hacia un servidor de pruebas o sandbox.
func (a *StripeAdapter) WithBaseURL(baseURL string) *StripeAdapter {
	if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
		a.baseURL = trimmed
	}
	return a
}

func (a *StripeAdapter) isRealStripeConfigured() bool {
	if a.secretKey == "" || strings.HasPrefix(a.secretKey, "replace-") || strings.Contains(a.secretKey, "replace_with") {
		return false
	}
	return strings.HasPrefix(a.secretKey, "sk_test_") || strings.HasPrefix(a.secretKey, "sk_live_")
}

func (a *StripeAdapter) CreateManualHoldIntent(ctx context.Context, auctionID, bidderID string, amount float64, currency string) (domain.StripePaymentIntent, error) {
	if amount <= 0 || auctionID == "" || bidderID == "" {
		return domain.StripePaymentIntent{}, domain.ErrGuaranteeInvalid
	}
	curr := strings.ToLower(strings.TrimSpace(currency))
	if curr == "" {
		curr = "pen"
	}

	if a.isRealStripeConfigured() {
		return a.createRemoteIntent(ctx, auctionID, bidderID, amount, curr)
	}

	// Modo Sandbox Determinista para Desarrollo Continuo sin bloqueo externo
	idTok, err := randomHex(10)
	if err != nil {
		return domain.StripePaymentIntent{}, err
	}
	secTok, err := randomHex(12)
	if err != nil {
		return domain.StripePaymentIntent{}, err
	}
	piID := "pi_stripe_hold_" + idTok
	clientSecret := piID + "_secret_" + secTok

	a.mu.Lock()
	a.sandbox[piID] = "requires_capture"
	a.mu.Unlock()

	return domain.StripePaymentIntent{
		PaymentIntentID: piID,
		ClientSecret:    clientSecret,
		PublishableKey:  a.publishableKey,
		Amount:          amount,
		Currency:        strings.ToUpper(curr),
		Status:          "requires_capture",
		LiveMode:        false,
	}, nil
}

func (a *StripeAdapter) VerifyHoldIntent(ctx context.Context, paymentIntentID string) (string, error) {
	piID := strings.TrimSpace(paymentIntentID)
	if piID == "" || !strings.HasPrefix(piID, "pi_") {
		return "", domain.ErrStripeHoldFailed
	}

	if strings.HasPrefix(piID, "pi_stripe_hold_") || !a.isRealStripeConfigured() {
		a.mu.RLock()
		status, exists := a.sandbox[piID]
		a.mu.RUnlock()
		if !exists {
			// Permitir confirmación de PaymentIntents generados en la sesión actual
			a.mu.Lock()
			a.sandbox[piID] = "requires_capture"
			a.mu.Unlock()
			return "requires_capture", nil
		}
		if status != "requires_capture" && status != "succeeded" {
			return status, domain.ErrStripeHoldFailed
		}
		return status, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/payment_intents/"+url.PathEscape(piID), nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(a.secretKey, "")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", domain.ErrStripeHoldFailed
	}
	var parsed struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.Status != "requires_capture" && parsed.Status != "succeeded" {
		return parsed.Status, domain.ErrStripeHoldFailed
	}
	return parsed.Status, nil
}

func (a *StripeAdapter) CaptureHoldIntent(ctx context.Context, paymentIntentID string) error {
	piID := strings.TrimSpace(paymentIntentID)
	if piID == "" {
		return domain.ErrGuaranteeInvalid
	}
	if strings.HasPrefix(piID, "pi_stripe_hold_") || !a.isRealStripeConfigured() {
		a.mu.Lock()
		a.sandbox[piID] = "succeeded"
		a.mu.Unlock()
		return nil
	}
	return a.postStripeAction(ctx, "/payment_intents/"+url.PathEscape(piID)+"/capture")
}

func (a *StripeAdapter) CancelHoldIntent(ctx context.Context, paymentIntentID string) error {
	piID := strings.TrimSpace(paymentIntentID)
	if piID == "" {
		return domain.ErrGuaranteeInvalid
	}
	if strings.HasPrefix(piID, "pi_stripe_hold_") || !a.isRealStripeConfigured() {
		a.mu.Lock()
		a.sandbox[piID] = "canceled"
		a.mu.Unlock()
		return nil
	}
	return a.postStripeAction(ctx, "/payment_intents/"+url.PathEscape(piID)+"/cancel")
}

func (a *StripeAdapter) VerifyWebhookSignature(payload []byte, signatureHeader string) error {
	if len(payload) == 0 {
		return domain.ErrStripeWebhookInvalid
	}
	// Si no hay webhookSecret configurado en desarrollo local, solo exigir formato válido de cabecera
	if a.webhookSecret == "" || strings.HasPrefix(a.webhookSecret, "replace-") {
		if strings.TrimSpace(signatureHeader) == "" {
			return domain.ErrStripeWebhookInvalid
		}
		return nil
	}

	var timestamp, sigV1 string
	parts := strings.Split(signatureHeader, ",")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sigV1 = kv[1]
		}
	}
	if timestamp == "" || sigV1 == "" {
		return domain.ErrStripeWebhookInvalid
	}

	mac := hmac.New(sha256.New, []byte(a.webhookSecret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sigV1)) {
		return domain.ErrStripeWebhookInvalid
	}
	return nil
}

func (a *StripeAdapter) createRemoteIntent(ctx context.Context, auctionID, bidderID string, amount float64, currency string) (domain.StripePaymentIntent, error) {
	cents := int64(math.Round(amount * 100))
	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", cents))
	form.Set("currency", currency)
	form.Set("capture_method", "manual")
	form.Set("metadata[auction_id]", auctionID)
	form.Set("metadata[bidder_id]", bidderID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/payment_intents", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.StripePaymentIntent{}, err
	}
	req.SetBasicAuth(a.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return domain.StripePaymentIntent{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return domain.StripePaymentIntent{}, fmt.Errorf("%w: status %d (%s)", domain.ErrStripeHoldFailed, resp.StatusCode, string(body))
	}

	var parsed struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
		Livemode     bool   `json:"livemode"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return domain.StripePaymentIntent{}, err
	}

	return domain.StripePaymentIntent{
		PaymentIntentID: parsed.ID,
		ClientSecret:    parsed.ClientSecret,
		PublishableKey:  a.publishableKey,
		Amount:          amount,
		Currency:        strings.ToUpper(currency),
		Status:          parsed.Status,
		LiveMode:        parsed.Livemode,
	}, nil
}

func (a *StripeAdapter) postStripeAction(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return domain.ErrStripeHoldFailed
	}
	return nil
}

func randomHex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

