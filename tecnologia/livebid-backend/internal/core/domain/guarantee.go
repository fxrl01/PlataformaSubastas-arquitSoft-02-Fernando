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
)

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
