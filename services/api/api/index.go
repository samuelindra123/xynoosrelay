package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/internal/server"
)

var (
	once sync.Once
	srv  *server.Server
)

func initialize() {
	cfg, err := config.Load()
	if err != nil {
		// Fallback in case of misconfiguration in serverless environment
		cfg = &config.Config{
			AppEnv:         config.EnvProduction,
			Port:           "8080",
			RequestTimeout: 15,
		}
	}

	log := logger.SetupLogger(cfg.AppEnv)

	// In serverless cold start, initialize pool with background context
	dbPool, err := database.NewPool(context.Background(), cfg, log)
	if err != nil {
		log.Error("serverless database initialization warning", "error", err)
	}

	srv = server.New(cfg, log, dbPool)
}

// Handler is the Vercel Go Serverless Function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(initialize)
	srv.Router().ServeHTTP(w, r)
}
