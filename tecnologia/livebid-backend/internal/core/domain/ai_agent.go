package domain

import "time"

type AgentStrategy string

const (
	StrategyMinimumIncrement AgentStrategy = "INCREMENTO_MINIMO"
	StrategyConservative     AgentStrategy = "CONSERVADORA"
	StrategyAntiSniping      AgentStrategy = "ANTI_SNIPING_DEFENSIVO"
)

// AgentMandate representa un mandato Zero-Trust acotado por presupuesto mediante el cual
// un postor humano autoriza a un Agente Autónomo de IA a pujar en su nombre.
type AgentMandate struct {
	ID                  string        `json:"mandato_id"`
	AuctionID           string        `json:"subasta_id"`
	PrincipalBidderID   string        `json:"postor_principal_id"`
	AgentID             string        `json:"agente_id"`
	MaxAuthorizedAmount float64       `json:"monto_maximo_autorizado"`
	Strategy            AgentStrategy `json:"estrategia"`
	Revoked             bool          `json:"revocado"`
	ExpiresAt           time.Time     `json:"expira_en"`
	CreatedAt           time.Time     `json:"creado_en"`
}

func (m *AgentMandate) IsValidAt(now time.Time) bool {
	return !m.Revoked && now.Before(m.ExpiresAt)
}

func (m *AgentMandate) CanAuthorizeBid(amount float64, now time.Time) error {
	if !m.IsValidAt(now) {
		return ErrAgentMandateExpired
	}
	if amount > m.MaxAuthorizedAmount {
		return ErrAgentMandateExceeded
	}
	return nil
}

type FraudAlertType string

const (
	AlertShillPingPong FraudAlertType = "COLUSION_PING_PONG"
	AlertBotSpeedBurst FraudAlertType = "VELOCIDAD_SOBREHUMANA_BOT"
	AlertIrrationalJump FraudAlertType = "SALTO_IRRACIONAL_PRECIO"
)

// FraudAlert representa una anomalía conductual detectada asíncronamente en la sala de subasta.
type FraudAlert struct {
	ID          string         `json:"id"`
	AuctionID   string         `json:"subasta_id"`
	Type        FraudAlertType `json:"tipo"`
	Severity    NivelRiesgo    `json:"severidad"`
	SuspectIDs  []string       `json:"sospechosos_ids"`
	Description string         `json:"descripcion"`
	DetectedAt  time.Time      `json:"detectada_en"`
}

// BidTelemetryEvent contiene los metadatos de una puja aceptada para análisis forense asíncrono.
type BidTelemetryEvent struct {
	AuctionID     string
	BidID         string
	BidderID      string
	Amount        float64
	PreviousPrice float64
	Timestamp     time.Time
	AgentMandate  string
}

// FraudTelemetryPort recibe eventos de puja sin bloquear el camino crítico para detección asíncrona.
type FraudTelemetryPort interface {
	PublishBidTelemetry(event BidTelemetryEvent)
	ListAlertsByAuction(auctionID string) []FraudAlert
}

