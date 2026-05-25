package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/config"
)

// New creates a *pgxpool.Pool from cfg and validates connectivity via Ping.
// The pool is ready to use on a nil-error return; it must be released with Close.
func New(ctx context.Context, cfg *config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}
	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.DBConnTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := Ping(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	log.Info("database connected", zap.String("db", cfg.DBHostPort()))
	return pool, nil
}

// Ping sends a SELECT 1 to verify the pool is alive.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "SELECT 1")
	return err
}

// Close gracefully closes the pool, waiting for all acquired connections to return.
func Close(pool *pgxpool.Pool) {
	pool.Close()
}
