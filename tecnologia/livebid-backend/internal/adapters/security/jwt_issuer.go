package security

import (
	"errors"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/golang-jwt/jwt/v5"
)

var ErrEmptyJWTSecret = errors.New("jwt secret is required")

type JWTIssuer struct {
	secret []byte
}

type VerifiedClaims struct {
	UserID string
	Role   domain.UserRole
}

func NewJWTIssuer(secret string) (*JWTIssuer, error) {
	if secret == "" {
		return nil, ErrEmptyJWTSecret
	}
	return &JWTIssuer{secret: []byte(secret)}, nil
}

func (i *JWTIssuer) Issue(claims domain.TokenClaims, expiresAt time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  claims.UserID,
		"role": string(claims.Role),
		"exp":  expiresAt.Unix(),
		"iat":  time.Now().Unix(),
	})
	return token.SignedString(i.secret)
}

func (i *JWTIssuer) Verify(rawToken string) (VerifiedClaims, error) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return i.secret, nil
	})
	if err != nil || !token.Valid {
		return VerifiedClaims{}, errors.New("invalid jwt")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return VerifiedClaims{}, errors.New("invalid jwt claims")
	}
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return VerifiedClaims{}, errors.New("missing jwt subject")
	}
	role, ok := claims["role"].(string)
	if !ok || role == "" {
		return VerifiedClaims{}, errors.New("missing jwt role")
	}
	return VerifiedClaims{UserID: userID, Role: domain.UserRole(role)}, nil
}
