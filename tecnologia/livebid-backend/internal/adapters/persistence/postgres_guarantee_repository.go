package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresGuaranteeRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresGuaranteeRepository(pool *pgxpool.Pool) *PostgresGuaranteeRepository {
	return &PostgresGuaranteeRepository{pool: pool}
}

func (r *PostgresGuaranteeRepository) FindActive(ctx context.Context, auctionID, bidderID string) (*domain.Guarantee, error) {
	var g domain.Guarantee
	err := r.pool.QueryRow(ctx, `
		SELECT id, subasta_id, postor_id, monto_retenido, autorizacion_bancaria_id,
			estado, score_confianza_ia, fecha_retencion
		FROM garantias
		WHERE subasta_id = $1 AND postor_id = $2 AND estado = 'RETENIDA'
		ORDER BY fecha_retencion DESC LIMIT 1`, auctionID, bidderID).Scan(
		&g.ID, &g.AuctionID, &g.BidderID, &g.Amount, &g.AuthorizationID,
		&g.Status, &g.TrustScore, &g.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find active guarantee: %w", err)
	}
	return &g, nil
}

func (r *PostgresGuaranteeRepository) Save(ctx context.Context, g *domain.Guarantee) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO garantias
			(id, subasta_id, postor_id, monto_retenido, autorizacion_bancaria_id,
			 estado, score_confianza_ia, fecha_retencion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		g.ID, g.AuctionID, g.BidderID, g.Amount, g.AuthorizationID,
		g.Status, g.TrustScore, g.CreatedAt)
	if err != nil {
		return fmt.Errorf("save guarantee: %w", err)
	}
	return nil
}

func (r *PostgresGuaranteeRepository) SettleAuction(ctx context.Context, auctionID, winnerID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin settle guarantees: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if winnerID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE garantias
			SET estado = 'CAPTURADA_COBRO'
			WHERE subasta_id = $1 AND postor_id = $2 AND estado = 'RETENIDA'`,
			auctionID, winnerID); err != nil {
			return fmt.Errorf("capture winning guarantee: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE garantias
		SET estado = 'LIBERADA'
		WHERE subasta_id = $1 AND estado = 'RETENIDA'
		  AND ($2 = '' OR postor_id <> $2)`,
		auctionID, winnerID); err != nil {
		return fmt.Errorf("release losing guarantees: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit settled guarantees: %w", err)
	}
	return nil
}
