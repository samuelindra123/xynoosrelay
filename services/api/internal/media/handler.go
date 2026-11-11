package media

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
	"github.com/xynoss/relay/services/api/internal/storage"
)

// Handler exposes the HTTP routes for media upload intents, finalization, and serving.
type Handler struct {
	service *Service
}

// NewHandler constructs a media HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Routes returns the Chi router for /v1/media.
func (h *Handler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Direct upload & content serving routes for local development provider
	r.Put("/content/upload", h.HandleLocalDirectUpload)
	r.Get("/content/*", h.ServeLocalContent)

	// Authenticated media management routes
	r.Group(func(authGroup chi.Router) {
		if requireAuth != nil {
			authGroup.Use(requireAuth)
		}

		authGroup.Post("/uploads", h.CreateUploadIntent)
		authGroup.Post("/{media_id}/finalize", h.FinalizeMedia)
		authGroup.Get("/{media_id}", h.GetMediaMetadata)
		authGroup.Delete("/{media_id}", h.DeleteMedia)
	})

	return r
}

func (h *Handler) CreateUploadIntent(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	var req MediaIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request body JSON"))
		return
	}

	purpose := storage.Purpose(strings.ToLower(strings.TrimSpace(req.Purpose)))
	if purpose != storage.PurposeAvatar && purpose != storage.PurposeBanner {
		response.RenderError(w, r, response.NewValidationErrorSingle("purpose", "purpose must be 'avatar' or 'banner'"))
		return
	}

	intent, err := h.service.CreateUploadIntent(r.Context(), actor.UserID, purpose, req.ContentType, req.SizeBytes, req.Checksum)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidMimeType) {
			response.RenderError(w, r, response.NewValidationErrorSingle("content_type", err.Error()))
			return
		}
		if errors.Is(err, storage.ErrFileTooLarge) {
			response.RenderError(w, r, response.NewError(http.StatusRequestEntityTooLarge, response.CodePayloadTooLarge, err.Error()))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusCreated, intent)
}

func (h *Handler) FinalizeMedia(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	mediaID := chi.URLParam(r, "media_id")
	if mediaID == "" {
		response.RenderError(w, r, response.NewBadRequestError("media_id is required"))
		return
	}

	var req FinalizeMediaRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	m, err := h.service.FinalizeUpload(r.Context(), actor.UserID, mediaID, req.Checksum)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorizedMediaAccess):
			response.RenderError(w, r, response.NewForbiddenError("you do not own this media asset"))
		case errors.Is(err, ErrMediaNotFound):
			response.RenderError(w, r, response.NewNotFoundError("media asset not found"))
		case errors.Is(err, storage.ErrSpoofedMimeType), errors.Is(err, storage.ErrInvalidMimeType), errors.Is(err, storage.ErrDangerousContent):
			response.RenderError(w, r, response.NewValidationErrorSingle("file", err.Error()))
		case errors.Is(err, storage.ErrCorruptImage), errors.Is(err, storage.ErrDimensionExceeded):
			response.RenderError(w, r, response.NewValidationErrorSingle("file", err.Error()))
		default:
			response.RenderError(w, r, response.NewInternalError(err))
		}
		return
	}

	response.JSON(w, r, http.StatusOK, m)
}

func (h *Handler) GetMediaMetadata(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "media_id")
	if mediaID == "" {
		response.RenderError(w, r, response.NewBadRequestError("media_id is required"))
		return
	}

	m, err := h.service.GetMedia(r.Context(), mediaID)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("media asset not found"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, m)
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("authentication required"))
		return
	}

	mediaID := chi.URLParam(r, "media_id")
	if mediaID == "" {
		response.RenderError(w, r, response.NewBadRequestError("media_id is required"))
		return
	}

	err := h.service.DeleteMedia(r.Context(), actor.UserID, mediaID)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedMediaAccess) {
			response.RenderError(w, r, response.NewForbiddenError("you do not own this media asset"))
			return
		}
		if errors.Is(err, ErrMediaNotFound) {
			response.RenderError(w, r, response.NewNotFoundError("media asset not found"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleLocalDirectUpload receives uploaded file streams when using local disk storage in development.
func (h *Handler) HandleLocalDirectUpload(w http.ResponseWriter, r *http.Request) {
	localProv, ok := h.service.StorageProvider().(*storage.LocalProvider)
	if !ok {
		response.RenderError(w, r, response.NewBadRequestError("direct uploads not supported on this storage provider"))
		return
	}

	key, contentType, sig, size, expires, err := storage.ParseUploadQueryParams(r.URL.Query())
	if err != nil {
		response.RenderError(w, r, response.NewBadRequestError(err.Error()))
		return
	}

	if !localProv.VerifyUploadSignature(key, contentType, size, expires, sig) {
		response.RenderError(w, r, response.NewForbiddenError("invalid or expired upload signature"))
		return
	}

	publicURL, err := localProv.PutObject(r.Context(), key, r.Body, size, contentType)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	response.JSON(w, r, http.StatusOK, map[string]string{
		"url": publicURL,
		"key": key,
	})
}

// ServeLocalContent serves public uploaded media files directly in development mode.
func (h *Handler) ServeLocalContent(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" {
		http.NotFound(w, r)
		return
	}

	rc, contentType, err := h.service.StorageProvider().GetObject(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, rc)
}
