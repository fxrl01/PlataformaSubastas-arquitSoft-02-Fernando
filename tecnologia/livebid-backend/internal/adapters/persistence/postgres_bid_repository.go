package persistence

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBidRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBidRepository(pool *pgxpool.Pool) *PostgresBidRepository {
	return &PostgresBidRepository{pool: pool}
}

func (r *PostgresBidRepository) BuscarPorIdempotencyKey(ctx context.Context, subastaID, bidderID, key string) (*domain.Puja, error) {
	var bid domain.Puja
	var amount string
	err := r.pool.QueryRow(ctx, `
		SELECT id, subasta_id, postor_id, monto::text, timestamp_oficial
		FROM pujas
		WHERE subasta_id = $1 AND postor_id = $2 AND idempotency_key = $3`,
		subastaID, bidderID, key).Scan(
		&bid.ID, &bid.SubastaID, &bid.PostorID, &amount, &bid.Timestamp,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find bid by idempotency key: %w", err)
	}
	bid.Monto, err = strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("parse idempotent bid amount: %w", err)
	}
	return &bid, nil
}

func (r *PostgresBidRepository) RegistrarPujaAtomica(ctx context.Context, subastaID string, puja domain.Puja, precioAnterior float64, fechaCierre time.Time, tiempoExtendido bool, idempotencyKey string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bid transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var estado string
	var precioActual string
	err = tx.QueryRow(ctx, `
		SELECT estado, precio_actual::text
		FROM subastas WHERE id = $1 FOR UPDATE`, subastaID).Scan(&estado, &precioActual)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSubastaNoEncontrada
	}
	if err != nil {
		return fmt.Errorf("lock auction: %w", err)
	}
	if estado != string(domain.SubastaActiva) && estado != string(domain.SubastaEnConteo) {
		return domain.ErrSubastaCerrada
	}
	precioActualValue, err := strconv.ParseFloat(precioActual, 64)
	if err != nil {
		return fmt.Errorf("parse auction price: %w", err)
	}
	if precioActualValue != precioAnterior {
		return domain.ErrPujaConcurrente
	}

	var correlativo int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(correlativo), 0) + 1
		FROM pujas WHERE subasta_id = $1`, subastaID).Scan(&correlativo)
	if err != nil {
		return fmt.Errorf("next bid sequence: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE subastas
		SET precio_actual = $1, postor_lider_id = $2, fin_programado = $3
		WHERE id = $4`,
		puja.Monto, puja.PostorID, fechaCierre, subastaID)
	if err != nil {
		return fmt.Errorf("update auction price: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO pujas
			(id, subasta_id, postor_id, monto, correlativo, timestamp_oficial, anti_sniping_disparado, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))`,
		puja.ID, subastaID, puja.PostorID, puja.Monto, correlativo, puja.Timestamp, tiempoExtendido, idempotencyKey)
	if err != nil {
		return fmt.Errorf("insert bid: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bid transaction: %w", err)
	}
	return nil
}

func (r *PostgresBidRepository) ListarPorSubasta(ctx context.Context, subastaID, bidderID string, minimumAmount float64, limit, offset int) ([]domain.Puja, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, subasta_id, postor_id, monto::text, timestamp_oficial
		FROM pujas
		WHERE subasta_id = $1
		  AND ($2 = '' OR postor_id = $2)
		  AND ($3 = 0 OR monto >= $3)
		ORDER BY correlativo ASC
		LIMIT $4 OFFSET $5`, subastaID, bidderID, minimumAmount, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list auction bids: %w", err)
	}
	defer rows.Close()

	bids := make([]domain.Puja, 0)
	for rows.Next() {
		var bid domain.Puja
		var amount string
		if err := rows.Scan(&bid.ID, &bid.SubastaID, &bid.PostorID, &amount, &bid.Timestamp); err != nil {
			return nil, 0, fmt.Errorf("scan auction bid: %w", err)
		}
		bid.Monto, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("parse bid amount: %w", err)
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate auction bids: %w", err)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pujas
		WHERE subasta_id = $1
		  AND ($2 = '' OR postor_id = $2)
		  AND ($3 = 0 OR monto >= $3)`,
		subastaID, bidderID, minimumAmount).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count auction bids: %w", err)
	}
	return bids, total, nil
}
