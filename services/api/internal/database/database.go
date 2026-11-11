package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xynoss/relay/services/api/internal/config"
)

// DB defines the database interface required by handlers and repositories.
type DB interface {
	Ping(ctx context.Context) error
	Close()
	IsConnected() bool
	RawPool() *pgxpool.Pool
}

// Pool wraps pgxpool.Pool with Neon-friendly configuration and lifecycle management.
type Pool struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewPool initializes a PostgreSQL connection pool optimized for Neon serverless architecture.
func NewPool(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Pool, error) {
	if cfg.DatabaseURL == "" {
		if cfg.AppEnv == config.EnvProduction || cfg.AppEnv == config.EnvStaging {
			return nil, errors.New("DATABASE_URL is required in staging and production")
		}
		log.Warn("DATABASE_URL not set; running with disconnected database placeholder")
		return &Pool{pool: nil, log: log}, nil
	}

	poolConfig, err := ParsePoolConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection on startup (allow 15s for Neon serverless wake-up)
	pingCtx, pingCancel := context.WithTimeout(ctx, 15*time.Second)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping verification failed: %w", err)
	}

	log.Info("database connection pool successfully initialized",
		slog.Int("max_conns", int(cfg.DBMaxConns)),
		slog.Int("min_conns", int(cfg.DBMinConns)),
	)

	return &Pool{pool: pool, log: log}, nil
}

// ParsePoolConfig constructs a Neon-compatible pgxpool.Config with connection limits and lifecycle jitter.
func ParsePoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	parsedURL, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// Sanitize URL for logging if needed
	poolConfig, err := pgxpool.ParseConfig(parsedURL.String())
	if err != nil {
		return nil, err
	}

	// Neon / Serverless connection settings
	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns

	// Jitter conn lifetime around 25-35 minutes to prevent synchronized pool recycles
	jitter := time.Duration(rand.Intn(600)) * time.Second
	poolConfig.MaxConnLifetime = 25*time.Minute + jitter
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Safe connection timeouts
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second

	// Set simple protocol or default prepared statements compatible with PgBouncer transaction pooling
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	return poolConfig, nil
}

// Ping checks whether database connection is alive.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return errors.New("database not connected")
	}
	return p.pool.Ping(ctx)
}

// Close gracefully terminates the connection pool.
func (p *Pool) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
		p.log.Info("database connection pool closed")
	}
}

// IsConnected returns whether an underlying pool exists.
func (p *Pool) IsConnected() bool {
	return p != nil && p.pool != nil
}

// RawPool returns the underlying pgxpool.Pool (or nil).
func (p *Pool) RawPool() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.pool
}
