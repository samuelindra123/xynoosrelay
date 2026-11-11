package session

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
)

// Handler exposes HTTP routes for active device session management.
type Handler struct {
	service      *Service
	isProduction bool
}

// NewHandler constructs a session HTTP handler.
func NewHandler(service *Service, isProduction bool) *Handler {
	return &Handler{
		service:      service,
		isProduction: isProduction,
	}
}

// Routes returns the Chi router for /v1/me/sessions.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListSessions)
	r.Delete("/{session_id}", h.RevokeSession)

	return r
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	sessions, err := h.service.ListUserSessions(r.Context(), actor.UserID, actor.SessionID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, sessions)
}

func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	targetSessionID := chi.URLParam(r, "session_id")
	if targetSessionID == "" {
		response.RenderError(w, r, response.NewBadRequestError("session_id is required"))
		return
	}

	err := h.service.RevokeSession(r.Context(), actor.UserID, targetSessionID)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedSessionAccess) {
			response.RenderError(w, r, response.NewForbiddenError("you cannot revoke a session that does not belong to you"))
			return
		}
		if errors.Is(err, ErrSessionNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("session not found"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	// If the user revoked their own current session, clear their session cookie
	if targetSessionID == actor.SessionID {
		ClearSessionCookie(w, h.isProduction)
	}

	w.WriteHeader(http.StatusNoContent)
}
