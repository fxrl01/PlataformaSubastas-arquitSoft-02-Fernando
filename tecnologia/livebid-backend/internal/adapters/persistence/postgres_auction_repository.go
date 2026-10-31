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

type PostgresAuctionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuctionRepository(pool *pgxpool.Pool) *PostgresAuctionRepository {
	return &PostgresAuctionRepository{pool: pool}
}

func (r *PostgresAuctionRepository) ObtenerPorID(ctx context.Context, id string) (*domain.Subasta, error) {
	const query = `
		SELECT id, titulo, descripcion, precio_base, incremento_minimo, precio_actual,
			postor_lider_id, estado, fin_programado, created_at, inicio_programado, vendedor_id
		FROM subastas WHERE id = $1`
	subasta, err := scanAuction(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSubastaNoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("find auction by id: %w", err)
	}
	return subasta, nil
}

func (r *PostgresAuctionRepository) Listar(ctx context.Context) ([]*domain.Subasta, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, titulo, descripcion, precio_base, incremento_minimo, precio_actual,
			postor_lider_id, estado, fin_programado, created_at, inicio_programado, vendedor_id
		FROM subastas ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list auctions: %w", err)
	}
	defer rows.Close()

	var result []*domain.Subasta
	for rows.Next() {
		subasta, err := scanAuction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan auction: %w", err)
		}
		result = append(result, subasta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auctions: %w", err)
	}
	return result, nil
}

func (r *PostgresAuctionRepository) Guardar(ctx context.Context, subasta *domain.Subasta) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subastas
			(id, vendedor_id, titulo, descripcion, precio_base, incremento_minimo,
			 precio_actual, postor_lider_id, estado, inicio_programado, fin_programado)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			vendedor_id = EXCLUDED.vendedor_id,
			titulo = EXCLUDED.titulo,
			descripcion = EXCLUDED.descripcion,
			precio_actual = EXCLUDED.precio_actual,
			postor_lider_id = EXCLUDED.postor_lider_id,
			estado = EXCLUDED.estado,
			inicio_programado = EXCLUDED.inicio_programado,
			fin_programado = EXCLUDED.fin_programado`,
		subasta.ID, nullableString(subasta.VendedorID), subasta.Titulo, subasta.Descripcion,
		subasta.PrecioBase, subasta.CalcularIncrementoMinimo(), subasta.PrecioActual,
		nullableString(subasta.PostorLiderID), subasta.Estado, subasta.FechaInicio, subasta.FechaCierre)
	if err != nil {
		return fmt.Errorf("save auction: %w", err)
	}
	return nil
}

type auctionScanner interface {
	Scan(dest ...any) error
}

func scanAuction(row auctionScanner) (*domain.Subasta, error) {
	var subasta domain.Subasta
	var vendedorID, liderID, descripcion *string
	var precioBase, incrementoMinimo, precioActual string
	var createdAt time.Time
	err := row.Scan(
		&subasta.ID, &subasta.Titulo, &descripcion, &precioBase, &incrementoMinimo,
		&precioActual, &liderID, &subasta.Estado, &subasta.FechaCierre, &createdAt,
		&subasta.FechaInicio, &vendedorID,
	)
	if err != nil {
		return nil, err
	}
	if vendedorID != nil {
		subasta.VendedorID = *vendedorID
	}
	if descripcion != nil {
		subasta.Descripcion = *descripcion
	}
	if liderID != nil {
		subasta.PostorLiderID = *liderID
	}
	subasta.PrecioBase, err = strconv.ParseFloat(precioBase, 64)
	if err != nil {
		return nil, fmt.Errorf("parse base price: %w", err)
	}
	if _, err := strconv.ParseFloat(incrementoMinimo, 64); err != nil {
		return nil, fmt.Errorf("parse minimum increment: %w", err)
	}
	subasta.PrecioActual, err = strconv.ParseFloat(precioActual, 64)
	if err != nil {
		return nil, fmt.Errorf("parse current price: %w", err)
	}
	subasta.Pujas = make([]domain.Puja, 0)
	return &subasta, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
