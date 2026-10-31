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
