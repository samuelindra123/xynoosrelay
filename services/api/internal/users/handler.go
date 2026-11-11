package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/media"
	"github.com/xynoss/relay/services/api/internal/response"
)

// Handler exposes the HTTP routes for user profiles and onboarding.
type Handler struct {
	service *Service
}

// NewHandler constructs a users HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// MeRoutes returns the Chi router for /v1/me.
func (h *Handler) MeRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetCurrentUser)
	r.Get("/profile", h.GetCurrentProfile)
	r.Patch("/profile", h.UpdateProfile)
	r.Post("/onboarding/complete", h.CompleteOnboarding)

	return r
}

// UserMediaRoutes returns routes for /v1/users/me/avatar and /v1/users/me/banner.
func (h *Handler) UserMediaRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/search", h.SearchUsers)
	r.Post("/me/avatar", h.SetAvatar)
	r.Delete("/me/avatar", h.RemoveAvatar)
	r.Post("/me/banner", h.SetBanner)
	r.Delete("/me/banner", h.RemoveBanner)

	return r
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	results, err := h.service.SearchUsers(r.Context(), q, limit)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	if results == nil {
		results = []*UserSearchResult{}
	}

	response.JSON(w, r, http.StatusOK, results)
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	userView, err := h.service.GetCurrentUser(r.Context(), actor.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("user account not found"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, userView)
}

func (h *Handler) GetCurrentProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	userView, err := h.service.GetCurrentUser(r.Context(), actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, userView.Profile)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request body JSON"))
		return
	}

	// Check optional If-Match header for revision optimistic concurrency
	ifMatchRevision := 0
	ifMatchHeader := r.Header.Get("If-Match")
	if ifMatchHeader != "" {
		if rev, err := strconv.Atoi(ifMatchHeader); err == nil {
			ifMatchRevision = rev
		}
	}

	updated, err := h.service.UpdateProfile(r.Context(), actor.UserID, &req, ifMatchRevision)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameConflict):
			response.RenderError(w, r, response.NewConflictError("username is already taken"))
		case errors.Is(err, ErrRevisionMismatch):
			response.RenderError(w, r, response.NewError(http.StatusPreconditionFailed, "REVISION_MISMATCH", "profile revision mismatch"))
		case errors.Is(err, ErrInvalidUsername):
			response.RenderError(w, r, response.NewValidationErrorSingle("username", err.Error()))
		case errors.Is(err, ErrInvalidDisplayName):
			response.RenderError(w, r, response.NewValidationErrorSingle("display_name", err.Error()))
		default:
			response.RenderError(w, r, response.NewInternalError(err))
		}
		return
	}

	response.JSON(w, r, http.StatusOK, updated)
}

type SetMediaRequestBody struct {
	MediaID string `json:"media_id"`
}

func (h *Handler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var body SetMediaRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MediaID == "" {
		response.RenderError(w, r, response.NewBadRequestError("media_id is required"))
		return
	}

	profile, err := h.service.SetAvatar(r.Context(), actor.UserID, body.MediaID)
	if err != nil {
		if errors.Is(err, media.ErrUnauthorizedMediaAccess) {
			response.RenderError(w, r, response.NewForbiddenError("you do not own this media asset"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

func (h *Handler) RemoveAvatar(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	profile, err := h.service.RemoveAvatar(r.Context(), actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

func (h *Handler) SetBanner(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var body SetMediaRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MediaID == "" {
		response.RenderError(w, r, response.NewBadRequestError("media_id is required"))
		return
	}

	profile, err := h.service.SetBanner(r.Context(), actor.UserID, body.MediaID)
	if err != nil {
		if errors.Is(err, media.ErrUnauthorizedMediaAccess) {
			response.RenderError(w, r, response.NewForbiddenError("you do not own this media asset"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

func (h *Handler) RemoveBanner(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	profile, err := h.service.RemoveBanner(r.Context(), actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, profile)
}

func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var body CompleteOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request body JSON"))
		return
	}

	view, err := h.service.CompleteOnboarding(r.Context(), actor.UserID, body.DisplayName, body.Username)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameConflict):
			response.RenderError(w, r, response.NewConflictError("username is already taken"))
		case errors.Is(err, ErrInvalidUsername):
			response.RenderError(w, r, response.NewValidationErrorSingle("username", err.Error()))
		case errors.Is(err, ErrInvalidDisplayName):
			response.RenderError(w, r, response.NewValidationErrorSingle("display_name", err.Error()))
		default:
			response.RenderError(w, r, response.NewInternalError(err))
		}
		return
	}

	response.JSON(w, r, http.StatusOK, view)
}
