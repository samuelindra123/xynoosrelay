package health

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/response"
)

// Handler provides health check HTTP endpoints.
type Handler struct {
	db     database.DB
	appEnv string
}

// NewHandler constructs a health Handler.
func NewHandler(db database.DB, appEnv string) *Handler {
	return &Handler{
		db:     db,
		appEnv: appEnv,
	}
}

// Routes registers health endpoints on a Chi router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/live", h.Live)
	r.Get("/ready", h.Ready)
	r.Get("/", h.Status)
	return r
}

// Live handles GET /health/live (Liveness probe - checks if process is responsive).
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Ready handles GET /health/ready (Readiness probe - checks database dependencies).
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	deps := make(map[string]string)
	isReady := true

	if h.db != nil && h.db.IsConnected() {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.db.Ping(ctx); err != nil {
			deps["database"] = "down"
			isReady = false
		} else {
			deps["database"] = "up"
		}
	} else {
		deps["database"] = "unconfigured"
	}

	if !isReady {
		response.JSON(w, r, http.StatusServiceUnavailable, map[string]any{
			"status":       "not_ready",
			"dependencies": deps,
		})
		return
	}

	response.JSON(w, r, http.StatusOK, map[string]any{
		"status":       "ready",
		"dependencies": deps,
	})
}

// Status handles GET /v1/health (General service status).
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, map[string]any{
		"status":      "ok",
		"service":     "xynoss-relay-api",
		"environment": h.appEnv,
	})
}
