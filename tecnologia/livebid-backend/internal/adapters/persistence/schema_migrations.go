package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureSchema aplica de forma idempotente las extensiones de esquema para RBAC,
// perfil de usuario, encargado oficial de subasta y ficha pericial del lote (JSONB).
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const ddl = `
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS kyc_aprobado BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS rol_solicitado VARCHAR(30) NOT NULL DEFAULT 'POSTOR';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS documento_identidad VARCHAR(64) NOT NULL DEFAULT '';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS telefono VARCHAR(40) NOT NULL DEFAULT '';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS wallet_address VARCHAR(80) NOT NULL DEFAULT '';

		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS martillero_id VARCHAR(64) REFERENCES usuarios(id);
		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS encargado_json JSONB NOT NULL DEFAULT '{}'::jsonb;
		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS detalles_lote_json JSONB NOT NULL DEFAULT '{}'::jsonb;
		CREATE INDEX IF NOT EXISTS idx_subastas_created_at ON subastas(created_at DESC);
	`
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("ensure schema migrations: %w", err)
	}
	return nil
}

