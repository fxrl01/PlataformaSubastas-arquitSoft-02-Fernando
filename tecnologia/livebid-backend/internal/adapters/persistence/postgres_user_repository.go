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

const userSelectColumns = `
	SELECT id, nombre, email, password_hash, rol,
		COALESCE(rol_solicitado, rol), estado, COALESCE(kyc_aprobado, false),
		COALESCE(documento_identidad, ''), COALESCE(telefono, ''), COALESCE(wallet_address, ''),
		COALESCE(bio, ''),
		created_at, updated_at
	FROM usuarios`

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := userSelectColumns + ` WHERE email = $1`
	user, err := scanUser(r.pool.QueryRow(ctx, query, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := userSelectColumns + ` WHERE id = $1`
	user, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) ListUsers(ctx context.Context, status domain.UserStatus, role domain.UserRole) ([]*domain.User, error) {
	query := userSelectColumns + ` WHERE ($1 = '' OR estado = $1) AND ($2 = '' OR rol = $2) ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, string(status), string(role))
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var result []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	user := &domain.User{}
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.RequestedRole,
		&user.Status,
		&user.KYCApproved,
		&user.DocumentID,
		&user.Phone,
		&user.WalletAddress,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	requestedRole := user.RequestedRole
	if requestedRole == "" {
		requestedRole = user.Role
	}
	const query = `
		INSERT INTO usuarios
			(id, nombre, email, password_hash, rol, rol_solicitado, estado, kyc_aprobado,
			 documento_identidad, telefono, wallet_address, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Name, user.Email, user.PasswordHash,
		user.Role, requestedRole, user.Status, user.KYCApproved,
		user.DocumentID, user.Phone, user.WalletAddress, user.Bio,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var constraintErr *pgconn.PgError
		if errors.As(err, &constraintErr) && constraintErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) UpdateStatus(ctx context.Context, id string, status domain.UserStatus) error {
	kyc := status == domain.UserActive
	return r.UpdateKYCAndStatus(ctx, id, status, kyc)
}

func (r *PostgresUserRepository) UpdateKYCAndStatus(ctx context.Context, id string, status domain.UserStatus, kycApproved bool) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE usuarios SET estado = $1, kyc_aprobado = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
		status, kycApproved, id,
	)
	if err != nil {
		return fmt.Errorf("update user kyc and status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) UpdateRole(ctx context.Context, id string, role domain.UserRole) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE usuarios SET rol = $1, rol_solicitado = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		role, id,
	)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) UpdateProfile(ctx context.Context, id, name, phone, documentID, walletAddress, bio string) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE usuarios
		 SET nombre = $1, telefono = $2, documento_identidad = $3, wallet_address = $4, bio = $5, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $6`,
		name, phone, documentID, walletAddress, bio, id,
	)
	if err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
