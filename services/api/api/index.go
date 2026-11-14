package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
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
	restoreRewrittenPath(r)
	once.Do(initialize)
	if initError != nil || srv == nil {
		http.Error(w, "service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	srv.Router().ServeHTTP(w, r)
}

// restoreRewrittenPath restores the API path captured by vercel.json. Vercel
// invokes this function at /api, so the original /v1/... path is carried in a
// reserved query parameter and removed before the request reaches the router.
func restoreRewrittenPath(r *http.Request) bool {
	query := r.URL.Query()
	path := query.Get("__relay_path")
	if path != "/v1" && !strings.HasPrefix(path, "/v1/") {
		return false
	}

	r.URL.Path = path
	r.URL.RawPath = ""
	query.Del("__relay_path")
	r.URL.RawQuery = query.Encode()
	return true
}
