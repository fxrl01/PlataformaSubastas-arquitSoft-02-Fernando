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
}

// BidLockPort define el puerto para adquirir bloqueos atómicos distribuidos (ej. Redis SETNX).
type BidLockPort interface {
	AdquirirLock(ctx context.Context, subastaID string, ttl time.Duration) (UnlockFunc, error)
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
