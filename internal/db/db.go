package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onelineai/hana-news-api/internal/config"
)

type DB struct {
	Silver *pgxpool.Pool
	Gold   *pgxpool.Pool
}

func New(ctx context.Context, cfg *config.Config) (*DB, error) {
	silver, err := connectPool(ctx, cfg.Silver, true)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to silver db: %w", err)
	}

	gold, err := connectPool(ctx, cfg.Gold, false)
	if err != nil {
		silver.Close()
		return nil, fmt.Errorf("failed to connect to gold db: %w", err)
	}

	return &DB{
		Silver: silver,
		Gold:   gold,
	}, nil
}

func connectPool(ctx context.Context, cfg config.DBConfig, readOnly bool) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return pool, nil
}

func (d *DB) Close() {
	if d.Silver != nil {
		d.Silver.Close()
	}
	if d.Gold != nil {
		d.Gold.Close()
	}
}

func (d *DB) HealthCheck(ctx context.Context) error {
	if err := d.Silver.Ping(ctx); err != nil {
		return fmt.Errorf("silver db unhealthy: %w", err)
	}
	if err := d.Gold.Ping(ctx); err != nil {
		return fmt.Errorf("gold db unhealthy: %w", err)
	}
	return nil
}
