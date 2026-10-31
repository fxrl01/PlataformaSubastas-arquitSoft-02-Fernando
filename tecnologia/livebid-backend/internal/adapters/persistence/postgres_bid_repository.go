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

func (r *PostgresBidRepository) RegistrarPujaAtomica(ctx context.Context, subastaID string, puja domain.Puja, precioAnterior float64, fechaCierre time.Time, tiempoExtendido bool) error {
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
			(id, subasta_id, postor_id, monto, correlativo, timestamp_oficial, anti_sniping_disparado)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		puja.ID, subastaID, puja.PostorID, puja.Monto, correlativo, puja.Timestamp, tiempoExtendido)
	if err != nil {
		return fmt.Errorf("insert bid: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bid transaction: %w", err)
	}
	return nil
}
