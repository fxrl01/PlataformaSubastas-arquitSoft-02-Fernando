package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Save(ctx context.Context, user *User) error
	UpdateStatus(ctx context.Context, id string, status UserStatus) error
}

// AdminUserRepository extiende UserRepository con capacidades administrativas de RBAC y KYC.
type AdminUserRepository interface {
	UserRepository
	ListUsers(ctx context.Context, status UserStatus, role UserRole) ([]*User, error)
	UpdateRole(ctx context.Context, id string, role UserRole) error
	UpdateKYCAndStatus(ctx context.Context, id string, status UserStatus, kycApproved bool) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenClaims struct {
	UserID string
	Role   UserRole
}

type TokenIssuer interface {
	Issue(claims TokenClaims, expiresAt time.Time) (string, error)
}

type RefreshTokenStore interface {
	Save(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	Consume(ctx context.Context, tokenHash string) (string, error)
}

// UnlockFunc es un callback para liberar un lock atómico adquirido.
type UnlockFunc func() error

// SubastaRepository define el puerto (interfaz) para la persistencia de subastas.
type SubastaRepository interface {
	ObtenerPorID(ctx context.Context, id string) (*Subasta, error)
	Guardar(ctx context.Context, subasta *Subasta) error
	Listar(ctx context.Context) ([]*Subasta, error)
}

// ExpiredSubastaRepository define una capacidad optimizada para obtener únicamente
// las subastas activas cuyo tiempo de cierre ya venció, evitando escaneos completos.
type ExpiredSubastaRepository interface {
	ListarVencidas(ctx context.Context, now time.Time) ([]*Subasta, error)
}

type BidRepository interface {
	BuscarPorIdempotencyKey(ctx context.Context, subastaID, bidderID, key string) (*Puja, error)
	RegistrarPujaAtomica(ctx context.Context, subastaID string, puja Puja, precioAnterior float64, fechaCierre time.Time, tiempoExtendido bool, idempotencyKey string) error
	ListarPorSubasta(ctx context.Context, subastaID, bidderID string, minimumAmount float64, limit, offset int) ([]Puja, int, error)
}

// BidLockPort define el puerto para adquirir bloqueos atómicos distribuidos (ej. Redis SETNX).
type BidLockPort interface {
	AdquirirLock(ctx context.Context, subastaID string, ttl time.Duration) (UnlockFunc, error)
}

type BidNotifierPort interface {
	BroadcastNuevaPuja(ctx context.Context, subastaID string, puja Puja, tiempoExtendido bool, fechaCierre time.Time) error
	BroadcastSubastaCerrada(ctx context.Context, subastaID string, ganadora *Puja) error
}

type GuaranteeRepository interface {
	FindActive(ctx context.Context, auctionID, bidderID string) (*Guarantee, error)
	Save(ctx context.Context, guarantee *Guarantee) error
	SettleAuction(ctx context.Context, auctionID, winnerID string) error
}

// GuaranteeStatusSynchronizer permite sincronizar el estado de una garantía a partir
// de eventos asíncronos firmados de la pasarela bancaria (Stripe Webhooks) por su AuthorizationID (pi_...).
type GuaranteeStatusSynchronizer interface {
	UpdateStatusByAuthorizationID(ctx context.Context, authorizationID string, status GuaranteeStatus) error
}

// PaymentGatewayPort define el contrato hexagonal hacia la pasarela de pagos fiduciaria (Stripe).
// Divide estrictamente las responsabilidades Backend (creación de PaymentIntent Hold, verificación,
// captura al ganador y liberación al resto) de la captura de tarjeta en el Frontend.
type PaymentGatewayPort interface {
	CreateManualHoldIntent(ctx context.Context, auctionID, bidderID string, amount float64, currency string) (StripePaymentIntent, error)
	VerifyHoldIntent(ctx context.Context, paymentIntentID string) (string, error)
	CaptureHoldIntent(ctx context.Context, paymentIntentID string) error
	CancelHoldIntent(ctx context.Context, paymentIntentID string) error
	VerifyWebhookSignature(payload []byte, signatureHeader string) error
}

// PaymentEscrowPort define el puerto para validar y retener garantías de postores.
type PaymentEscrowPort interface {
	ValidarGarantiaActiva(ctx context.Context, postorID string, subastaID string) (bool, error)
	RetenerGarantia(ctx context.Context, postorID string, subastaID string, monto float64) error
	LiberarGarantia(ctx context.Context, postorID string, subastaID string) error
}

// RoomNotifierPort define el puerto para emitir eventos en tiempo real a las salas WebSocket.
type RoomNotifierPort interface {
	BroadcastNuevaPuja(ctx context.Context, subastaID string, puja Puja, tiempoExtendido bool) error
	BroadcastSubastaCerrada(ctx context.Context, subastaID string, ganadora *Puja) error
}

// NivelRiesgo representa la calificación de riesgo crediticio o conductual.
type NivelRiesgo string

const (
	RiesgoBajo  NivelRiesgo = "BAJO"
	RiesgoMedio NivelRiesgo = "MEDIO"
	RiesgoAlto  NivelRiesgo = "ALTO"
)

// AIRiskPort define el puerto para evaluación preventiva de riesgo y fraude con IA (Gemini).
type AIRiskPort interface {
	EvaluarRiesgo(ctx context.Context, postorID string, subastaID string, monto float64) (NivelRiesgo, error)
}

// EvaluacionRiesgo contiene el veredicto detallado del agente de IA o su Circuit Breaker.
type EvaluacionRiesgo struct {
	Score              int         `json:"score"`
	Nivel              NivelRiesgo `json:"nivel_riesgo"`
	PorcentajeGarantia float64     `json:"porcentaje_garantia"`
	Justificacion      string      `json:"justificacion"`
	ModoFallback       bool        `json:"modo_fallback"`
}

// AIRiskEvaluator extiende AIRiskPort con evaluación de perfil para garantías dinámicas.
type AIRiskEvaluator interface {
	AIRiskPort
	EvaluarPostor(ctx context.Context, postorID string, subastaID string) (EvaluacionRiesgo, error)
}

// Web3SignatureVerifierPort verifica firmas tipadas EIP-712 emitidas por billeteras Web3.
type Web3SignatureVerifierPort interface {
	ComputeTypedDataDigest(envelope EIP712BidEnvelope) string
	VerifyBidSignature(ctx context.Context, envelope EIP712BidEnvelope, now time.Time) error
}

// BlockchainEscrowPort define el puerto hacia el Smart Contract de custodia (Trustless Escrow) y anclaje Merkle.
type BlockchainEscrowPort interface {
	LockCollateralOnChain(ctx context.Context, auctionID, bidderID, walletAddress string, amount float64) (txHash string, err error)
	SettleEscrowAndAnchorMerkleRoot(ctx context.Context, auctionID, winnerID string, winningAmount float64, merkleRoot string) (OnChainSettlementReceipt, error)
}

// AuditLogEntry representa un eslabón de la cadena de auditoría forense de una subasta.
type AuditLogEntry struct {
	Event        string `json:"evento"`
	PreviousHash string `json:"hash_anterior"`
	CurrentHash  string `json:"hash_actual"`
}

// AuditTrailRepository consulta la cadena criptográfica de pujas y persiste recibos de anclaje on-chain.
type AuditTrailRepository interface {
	ObtenerHashesAuditoria(ctx context.Context, subastaID string) ([]string, error)
	ObtenerCadenaAuditoria(ctx context.Context, subastaID string) ([]AuditLogEntry, error)
	GuardarReciboOnChain(ctx context.Context, receipt OnChainSettlementReceipt) error
	ObtenerReciboOnChain(ctx context.Context, subastaID string) (*OnChainSettlementReceipt, error)
}

// AuditEventAppender permite anexar hitos del ciclo de vida de la subasta en la cadena SHA-256 de logs_auditoria.
type AuditEventAppender interface {
	AppendAuditEvent(ctx context.Context, subastaID, event string, payload map[string]any) error
}
