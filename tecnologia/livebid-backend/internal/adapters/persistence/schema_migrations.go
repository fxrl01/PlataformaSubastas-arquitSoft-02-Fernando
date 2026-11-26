package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureSchema aplica de forma idempotente las extensiones de esquema para RBAC,
// perfil de usuario editable, encargado oficial, ficha pericial (JSONB) y muro Social Feed.
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const ddl = `
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS kyc_aprobado BOOLEAN NOT NULL DEFAULT FALSE;
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS rol_solicitado VARCHAR(30) NOT NULL DEFAULT 'POSTOR';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS documento_identidad VARCHAR(64) NOT NULL DEFAULT '';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS telefono VARCHAR(40) NOT NULL DEFAULT '';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS wallet_address VARCHAR(80) NOT NULL DEFAULT '';
		ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '';

		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS martillero_id VARCHAR(64) REFERENCES usuarios(id);
		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS encargado_json JSONB NOT NULL DEFAULT '{}'::jsonb;
		ALTER TABLE subastas ADD COLUMN IF NOT EXISTS detalles_lote_json JSONB NOT NULL DEFAULT '{}'::jsonb;
		CREATE INDEX IF NOT EXISTS idx_subastas_created_at ON subastas(created_at DESC);

		CREATE TABLE IF NOT EXISTS social_posts (
			id VARCHAR(64) PRIMARY KEY,
			autor_id VARCHAR(64) NOT NULL DEFAULT '',
			autor_nombre VARCHAR(140) NOT NULL,
			autor_rol VARCHAR(30) NOT NULL DEFAULT 'POSTOR',
			contenido TEXT NOT NULL,
			imagen_url TEXT NOT NULL DEFAULT '',
			subasta_id VARCHAR(64) NOT NULL DEFAULT '',
			etiqueta VARCHAR(50) NOT NULL DEFAULT 'COMUNIDAD',
			likes_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_social_posts_created_at ON social_posts(created_at DESC);

		CREATE TABLE IF NOT EXISTS social_comments (
			id VARCHAR(64) PRIMARY KEY,
			post_id VARCHAR(64) NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
			autor_nombre VARCHAR(140) NOT NULL,
			comentario TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_social_comments_post_id ON social_comments(post_id, created_at ASC);

		INSERT INTO social_posts (id, autor_id, autor_nombre, autor_rol, contenido, subasta_id, etiqueta, likes_count, created_at)
		VALUES
			('soc_seed_01', 'usr_admin_root', 'Casa Aureum • Curaduría', 'MARTILLERO',
			 'Acabamos de completar la inspección técnica del lote principal en exhibición. Pueden revisar el cronograma oficial y activar su recordatorio antes del inicio de pujas.',
			 'SUB-2026-001', 'ADELANTO DE LOTE', 18, CURRENT_TIMESTAMP - INTERVAL '2 hours'),
			('soc_seed_02', 'usr_seed_coleccion', 'Valeria Montes • Coleccionista', 'VENDEDOR',
			 'Esta semana consignaremos piezas de alta relojería suiza con estuche y certificado original. ¿Qué modelos clásicos les gustaría ver en el próximo cronograma?',
			 '', 'COLECCIONISMO', 12, CURRENT_TIMESTAMP - INTERVAL '5 hours'),
			('soc_seed_03', 'usr_seed_guia', 'Dr. Carlos Mendoza • Martillero', 'MARTILLERO',
			 'Consejo para nuevos participantes: recuerden que si entra una oferta en los últimos 15 segundos, el reloj suma automáticamente +30 segundos para que todos compitan en igualdad.',
			 'SUB-2026-001', 'CONSEJO EN SALA', 25, CURRENT_TIMESTAMP - INTERVAL '1 day')
		ON CONFLICT (id) DO NOTHING;
	`
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("ensure schema migrations: %w", err)
	}
	return nil
}
