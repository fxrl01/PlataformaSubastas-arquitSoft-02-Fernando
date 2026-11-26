package domain

import (
	"errors"
	"strings"
	"time"
)

type UserRole string

const (
	RoleBidder     UserRole = "POSTOR"
	RoleSeller     UserRole = "VENDEDOR"
	RoleAuctioneer UserRole = "MARTILLERO"
	RoleAdmin      UserRole = "ADMIN"
)

func IsValidRole(role UserRole) bool {
	switch role {
	case RoleBidder, RoleSeller, RoleAuctioneer, RoleAdmin:
		return true
	default:
		return false
	}
}

type UserStatus string

const (
	UserPending   UserStatus = "PENDIENTE"
	UserActive    UserStatus = "ACTIVO"
	UserSuspended UserStatus = "SUSPENDIDO"
	UserBlocked   UserStatus = "BLOQUEADO"
)

func IsValidUserStatus(status UserStatus) bool {
	switch status {
	case UserPending, UserActive, UserSuspended, UserBlocked:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidEmail        = errors.New("domain: correo inválido")
	ErrInvalidPassword     = errors.New("domain: contraseña inválida")
	ErrInvalidUser         = errors.New("domain: usuario inválido")
	ErrInvalidRole         = errors.New("domain: rol de usuario inválido")
	ErrInvalidUserStatus   = errors.New("domain: estado de usuario inválido")
	ErrUserAlreadyExists   = errors.New("domain: usuario ya registrado")
	ErrUserNotFound        = errors.New("domain: usuario no encontrado")
	ErrInvalidCredentials  = errors.New("domain: credenciales inválidas")
	ErrUserNotActive       = errors.New("domain: usuario no activo")
	ErrRefreshTokenInvalid = errors.New("domain: refresh token inválido")
)

type User struct {
	ID            string
	Name          string
	Email         string
	PasswordHash  string
	Role          UserRole
	RequestedRole UserRole
	Status        UserStatus
	KYCApproved   bool
	DocumentID    string
	Phone         string
	WalletAddress string
	Bio           string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewUser(id, name, email, passwordHash string, now time.Time) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(email) < 5 {
		return nil, ErrInvalidEmail
	}
	if strings.TrimSpace(name) == "" || passwordHash == "" {
		return nil, ErrInvalidUser
	}
	return &User{
		ID:            id,
		Name:          strings.TrimSpace(name),
		Email:         email,
		PasswordHash:  passwordHash,
		Role:          RoleBidder,
		RequestedRole: RoleBidder,
		Status:        UserPending,
		KYCApproved:   false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}
