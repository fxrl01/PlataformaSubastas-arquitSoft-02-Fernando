package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRefreshTokenStore struct {
	pool *pgxpool.Pool
}

func NewPostgresRefreshTokenStore(pool *pgxpool.Pool) *PostgresRefreshTokenStore {
	return &PostgresRefreshTokenStore{pool: pool}
}

func (s *PostgresRefreshTokenStore) Save(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)`, tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (s *PostgresRefreshTokenStore) Consume(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrRefreshTokenInvalid
	}
	if err != nil {
		return "", fmt.Errorf("consume refresh token: %w", err)
	}
	return userID, nil
}
