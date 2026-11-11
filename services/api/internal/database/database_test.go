package database

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/xynoss/relay/services/api/internal/config"
)

func TestParsePoolConfig_NeonSettings(t *testing.T) {
	cfg := &config.Config{
		DatabaseURL: "postgres://user:secretpass@ep-cool-sample.us-east-2.aws.neon.tech/neondb?sslmode=require",
		DBMaxConns:  3,
		DBMinConns:  1,
	}

	poolConfig, err := ParsePoolConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error parsing pool config: %v", err)
	}

	if poolConfig.MaxConns != 3 {
		t.Errorf("expected MaxConns 3, got %d", poolConfig.MaxConns)
	}
	if poolConfig.MinConns != 1 {
		t.Errorf("expected MinConns 1, got %d", poolConfig.MinConns)
	}
	if poolConfig.MaxConnIdleTime != 5*time.Minute {
		t.Errorf("expected MaxConnIdleTime 5m, got %v", poolConfig.MaxConnIdleTime)
	}
	if poolConfig.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
		t.Errorf("expected QueryExecModeSimpleProtocol for Neon pooler compatibility")
	}
}

func TestNewPool_DevDisconnected(t *testing.T) {
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		AppEnv:      config.EnvDevelopment,
		DatabaseURL: "",
	}

	p, err := NewPool(context.Background(), cfg, nullLogger)
	if err != nil {
		t.Fatalf("expected no error in development with empty DATABASE_URL, got %v", err)
	}
	if p.IsConnected() {
		t.Errorf("expected pool to be disconnected")
	}
	if err := p.Ping(context.Background()); err == nil {
		t.Errorf("expected ping to fail when disconnected")
	}
}

func TestNewPool_ProductionRequiresDB(t *testing.T) {
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		AppEnv:      config.EnvProduction,
		DatabaseURL: "",
	}

	_, err := NewPool(context.Background(), cfg, nullLogger)
	if err == nil {
		t.Fatalf("expected error in production when DATABASE_URL is empty")
	}
}
