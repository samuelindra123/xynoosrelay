package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	log := logger.SetupLogger(cfg.AppEnv)

	targetURL := cfg.DatabaseDirectURL
	if targetURL == "" {
		targetURL = cfg.DatabaseURL
	}

	if targetURL == "" {
		log.Error("cannot run migrations: neither DATABASE_DIRECT_URL nor DATABASE_URL is set")
		os.Exit(1)
	}

	log.Info("running database migrations against Neon PostgreSQL...")
	if err := migrations.Up(targetURL, log); err != nil {
		log.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("all database migrations applied successfully")
}
