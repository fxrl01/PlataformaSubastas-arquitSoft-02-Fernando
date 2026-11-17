package domain

import (
	"errors"
	"time"
)

type GuaranteeStatus string

const (
	GuaranteeHeld     GuaranteeStatus = "RETENIDA"
	GuaranteeCaptured GuaranteeStatus = "CAPTURADA_COBRO"
	GuaranteeReleased GuaranteeStatus = "LIBERADA"
)

var (
	ErrGuaranteeAlreadyActive = errors.New("domain: ya existe una garantía activa")
	ErrGuaranteeInsufficient  = errors.New("domain: garantía insuficiente")
	ErrGuaranteeInvalid       = errors.New("domain: datos de garantía inválidos")
	ErrStripeHoldFailed       = errors.New("domain: no se pudo retener la garantía en Stripe")
	ErrStripeWebhookInvalid   = errors.New("domain: firma de webhook Stripe inválida")
)

// StripePaymentIntent encapsula los metadatos seguros de una pre-autorización (Hold) en Stripe.
// Cumple RNF-01 (PCI-DSS): nunca contiene PAN ni CVV de tarjeta.
type StripePaymentIntent struct {
	PaymentIntentID string  `json:"payment_intent_id"`
	ClientSecret    string  `json:"client_secret"`
	PublishableKey  string  `json:"publishable_key"`
	Amount          float64 `json:"monto"`
	Currency        string  `json:"moneda"`
	Status          string  `json:"estado_stripe"`
	LiveMode        bool    `json:"modo_produccion"`
}

type Guarantee struct {
	ID              string
	AuctionID       string
	BidderID        string
	Amount          float64
	AuthorizationID string
	Status          GuaranteeStatus
	TrustScore      int
	CreatedAt       time.Time
}

func RequiredGuarantee(basePrice float64) float64 {
	return basePrice * 0.10
}

func RequiredGuaranteeWithRatio(basePrice, ratio float64) float64 {
	if ratio <= 0 {
		return RequiredGuarantee(basePrice)
	}
	return basePrice * ratio
}
