package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/internal/server"
)

func main() {
	// 1. Load validated configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize structured logging
	log := logger.SetupLogger(cfg.AppEnv)
	log.Info("starting xynoss relay backend",
		slog.Any("config", cfg.RedactedSummary()),
	)

	// 3. Connect to PostgreSQL (Neon-friendly)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := database.NewPool(ctx, cfg, log)
	if err != nil {
		log.Error("fatal database connection error", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Assemble HTTP server with Chi router
	srv := server.New(cfg, log, dbPool)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Run server in background goroutine
	shutdownErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", httpServer.Addr), slog.String("env", cfg.AppEnv))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr <- err
		}
	}()

	// 6. Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("received shutdown signal", slog.String("signal", sig.String()))
	case err := <-shutdownErr:
		log.Error("http server failure", slog.Any("error", err))
	}

	// 7. Graceful shutdown with 10 second timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", slog.Any("error", err))
	}

	log.Info("server shutdown complete")
}
