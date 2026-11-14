package handler

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/internal/server"
)

var (
	once      sync.Once
	srv       *server.Server
	initError error
)

func initialize() {
	cfg, err := config.Load()
	if err != nil {
		initError = err
		slog.Error("serverless API configuration is invalid", "error", err)
		return
	}

	log := logger.SetupLogger(cfg.AppEnv)

	// In serverless cold start, initialize pool with background context
	dbPool, err := database.NewPool(context.Background(), cfg, log)
	if err != nil {
		log.Error("serverless database initialization failed", "error", err)
	}

	srv = server.New(cfg, log, dbPool)
}

// Handler is the Vercel Go Serverless Function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(initialize)
	if initError != nil || srv == nil {
		http.Error(w, "service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	srv.Router().ServeHTTP(w, r)
}
