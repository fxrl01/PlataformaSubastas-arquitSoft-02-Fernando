package persistence

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolOptions controla el dimensionamiento del pool de conexiones PostgreSQL
// y la compatibilidad con PgBouncer en modo transaction pooling.
type PoolOptions struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	SimpleProtocol    bool
}

// DefaultPoolOptionsFromEnv lee parámetros de escalabilidad de base de datos desde variables de entorno.
func DefaultPoolOptionsFromEnv() PoolOptions {
	return PoolOptions{
		MaxConns:          int32(readPositiveEnvInt("DB_MAX_CONNS", 25)),
		MinConns:          int32(readPositiveEnvInt("DB_MIN_CONNS", 4)),
		MaxConnLifetime:   time.Duration(readPositiveEnvInt("DB_MAX_CONN_LIFETIME_MIN", 30)) * time.Minute,
		MaxConnIdleTime:   time.Duration(readPositiveEnvInt("DB_MAX_CONN_IDLE_TIME_MIN", 5)) * time.Minute,
		HealthCheckPeriod: time.Duration(readPositiveEnvInt("DB_HEALTH_CHECK_PERIOD_SEC", 30)) * time.Second,
		SimpleProtocol:    strings.EqualFold(strings.TrimSpace(os.Getenv("DB_PGBOUNCER_MODE")), "true"),
	}
}

// NewPostgresPool crea un *pgxpool.Pool con parámetros de producción optimizados.
func NewPostgresPool(ctx context.Context, databaseURL string, opts PoolOptions) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}
	if opts.MinConns >= 0 && opts.MinConns <= cfg.MaxConns {
		cfg.MinConns = opts.MinConns
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	if opts.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = opts.HealthCheckPeriod
	}
	if opts.SimpleProtocol {
		// Requerido cuando se utiliza PgBouncer en pool_mode = transaction
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open configured database pool: %w", err)
	}
	return pool, nil
}

func readPositiveEnvInt(key string, fallback int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
