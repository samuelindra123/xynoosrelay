package workspaces

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) WorkspaceRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListWorkspaces)
	r.Post("/", h.CreateWorkspace)
	r.Get("/{id}", h.GetWorkspace)
	r.Post("/{id}/channels", h.CreateChannel)
	r.Get("/{id}/invites", h.ListInvites)
	r.Post("/{id}/invites", h.CreateInvite)
	r.Delete("/{id}/invites/{inviteID}", h.RevokeInvite)

	return r
}

func (h *Handler) InviteRoutes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/{code}", h.GetInvitePreview)
	r.With(requireAuth).Post("/{code}/accept", h.AcceptInvite)

	return r
}

type createWorkspaceReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var req createWorkspaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RenderError(w, r, response.NewError(http.StatusBadRequest, response.CodeMalformedRequest, "invalid json body"))
		return
	}

	ws, _, err := h.service.CreateWorkspace(r.Context(), actor.UserID, req.Name, req.Description)
	if err != nil {
		if IsInvalidInput(err) {
			response.RenderError(w, r, response.NewValidationErrorSingle("name", "workspace name must contain 1–100 characters and description at most 1000"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	det, err := h.service.GetWorkspaceByID(r.Context(), ws.ID, actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusCreated, det)
}

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	list, err := h.service.GetUserWorkspaces(r.Context(), actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	if list == nil {
		list = []*WorkspaceDetail{}
	}

	response.JSON(w, r, http.StatusOK, list)
}

func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}
	det, err := h.service.GetWorkspaceByID(r.Context(), wsID, actor.UserID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("workspace not found"))
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("you are not a member of this workspace"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, det)
}

type createChannelReq struct {
	Name string `json:"name"`
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	wsID := chi.URLParam(r, "id")
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}
	var req createChannelReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RenderError(w, r, response.NewError(http.StatusBadRequest, response.CodeMalformedRequest, "invalid json body"))
		return
	}

	ch, err := h.service.CreateChannel(r.Context(), wsID, actor.UserID, req.Name)
	if err != nil {
		if IsInvalidInput(err) {
			response.RenderError(w, r, response.NewValidationErrorSingle("name", "channel name must contain 1–50 characters"))
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("you do not have permission to create channels"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusCreated, ch)
}

type createInviteReq struct {
	DurationHours int `json:"duration_hours"`
	MaxUses       int `json:"max_uses"`
}

type createInviteResp struct {
	Code      string    `json:"code"`
	InviteURL string    `json:"invite_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	wsID := chi.URLParam(r, "id")
	var req createInviteReq
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid invitation settings"))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.RenderError(w, r, response.NewBadRequestError("expected one JSON object"))
		return
	}
	if req.DurationHours < 0 || req.DurationHours > 720 || req.MaxUses < 0 || req.MaxUses > 10000 {
		response.RenderError(w, r, response.NewValidationErrorSingle("invitation", "invalid duration or usage limit"))
		return
	}

	dur := time.Duration(req.DurationHours) * time.Hour
	if dur <= 0 {
		dur = 7 * 24 * time.Hour
	}

	inv, fullURL, err := h.service.CreateInvitation(r.Context(), wsID, actor.UserID, dur, req.MaxUses)
	if err != nil {
		if IsInvalidInput(err) {
			response.RenderError(w, r, response.NewValidationError("invalid invitation settings", []response.ErrorDetail{{Field: "duration_hours", Code: "must be between 1 and 720"}, {Field: "max_uses", Code: "must be between 0 and 10000"}}))
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("you do not have permission to create invitations"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusCreated, createInviteResp{
		Code:      inv.Code,
		InviteURL: fullURL,
		ExpiresAt: inv.ExpiresAt,
	})
}

func (h *Handler) GetInvitePreview(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	prev, err := h.service.GetInvitationPreview(r.Context(), code)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("invitation not found or expired"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, prev)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}
	items, err := h.service.ListInvitations(r.Context(), chi.URLParam(r, "id"), actor.UserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("you do not have permission to manage invitations"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.JSON(w, r, http.StatusOK, items)
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}
	err := h.service.RevokeInvitation(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "inviteID"), actor.UserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("you do not have permission to manage invitations"))
			return
		}
		if errors.Is(err, ErrInviteNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("invitation not found"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}
	response.NoContent(w)
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok || actor.UserID == "" {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	code := chi.URLParam(r, "code")
	det, err := h.service.AcceptInvitation(r.Context(), code, actor.UserID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.RenderError(w, r, response.NewForbiddenError("membership cannot be restored by invitation"))
			return
		}
		if errors.Is(err, ErrInviteNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("invitation not found, expired, or max uses reached"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, det)
}
