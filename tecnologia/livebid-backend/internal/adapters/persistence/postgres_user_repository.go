package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, nombre, email, password_hash, rol, estado, created_at, updated_at
		FROM usuarios WHERE email = $1`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash,
		&user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO usuarios
			(id, nombre, email, password_hash, rol, estado, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Name, user.Email,
		user.PasswordHash, user.Role, user.Status, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		var constraintErr *pgconn.PgError
		if errors.As(err, &constraintErr) && constraintErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}
