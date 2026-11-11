package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
	"github.com/xynoss/relay/services/api/internal/session"
	"github.com/xynoss/relay/services/api/internal/users"
)

// Handler exposes the HTTP routes for authentication.
type Handler struct {
	service        *Service
	sessionService *session.Service
	usersService   *users.Service
	isProduction   bool
	frontendURL    string
}

func (h *Handler) googleCallbackPath() string {
	uri, err := url.Parse(h.service.cfg.GoogleRedirectURI)
	if err == nil && strings.HasPrefix(uri.Path, "/") && uri.Path != "/" {
		return uri.Path
	}
	return "/v1/auth/google/callback"
}

// NewHandler constructs an auth HTTP handler.
func NewHandler(service *Service, sessionService *session.Service, usersService *users.Service, isProduction bool, frontendURL string) *Handler {
	return &Handler{
		service:        service,
		sessionService: sessionService,
		usersService:   usersService,
		isProduction:   isProduction,
		frontendURL:    frontendURL,
	}
}

// Routes returns the Chi router for /v1/auth.
func (h *Handler) Routes(requireAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Post("/otp/challenges", h.RequestOTPChallenge)
	r.Post("/otp/verifications", h.VerifyOTPCode)
	r.Get("/google/start", h.StartGoogle)
	r.Get("/google/callback", h.GoogleCallback)

	// Authenticated routes
	r.Group(func(authGroup chi.Router) {
		if requireAuth != nil {
			authGroup.Use(requireAuth)
		}
		authGroup.Get("/session", h.GetSessionBootstrap)
		authGroup.Post("/logout", h.Logout)
	})

	return r
}

type OTPChallengeRequestBody struct {
	Email string `json:"email"`
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func (h *Handler) RequestOTPChallenge(w http.ResponseWriter, r *http.Request) {
	var body OTPChallengeRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request body JSON"))
		return
	}

	clientIP := remoteIP(r.RemoteAddr)
	resp, err := h.service.RequestOTP(r.Context(), clientIP, body.Email)
	if err != nil {
		if errors.Is(err, ErrInvalidEmailFormat) {
			response.RenderError(w, r, response.NewValidationErrorSingle("email", "invalid email address format"))
			return
		}
		if errors.Is(err, ErrRateLimited) {
			response.RenderError(w, r, response.NewRateLimitedError(60, "too many challenge requests; please wait"))
			return
		}
		if errors.Is(err, ErrEmailDeliveryUnavailable) {
			response.RenderError(w, r, response.NewServiceUnavailableError("email delivery unavailable"))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	// 202 Accepted with generic anti-enumeration response
	response.JSON(w, r, http.StatusAccepted, resp)
}

type OTPVerificationRequestBody struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

func (h *Handler) VerifyOTPCode(w http.ResponseWriter, r *http.Request) {
	var body OTPVerificationRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.RenderError(w, r, response.NewBadRequestError("invalid request body JSON"))
		return
	}

	clientIP := remoteIP(r.RemoteAddr)
	userAgent := r.UserAgent()

	tokenPair, sess, csrfToken, userView, err := h.service.VerifyOTP(r.Context(), clientIP, body.ChallengeID, body.Code, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCode), errors.Is(err, ErrChallengeNotFound), errors.Is(err, ErrChallengeInvalidated):
			response.RenderError(w, r, response.NewUnauthorizedError("invalid or expired verification code"))
		case errors.Is(err, ErrChallengeExpired):
			response.RenderError(w, r, response.NewUnauthorizedError("verification code has expired"))
		case errors.Is(err, ErrTooManyAttempts):
			response.RenderError(w, r, response.NewUnauthorizedError("maximum verification attempts exceeded"))
		case errors.Is(err, ErrChallengeAlreadyConsumed):
			response.RenderError(w, r, response.NewUnauthorizedError("verification code has already been used"))
		case errors.Is(err, ErrRateLimited):
			response.RenderError(w, r, response.NewRateLimitedError(60, "too many verification attempts; please wait"))
		case errors.Is(err, ErrIdentityNotLinked):
			response.RenderError(w, r, response.NewUnauthorizedError("login method is not linked to this account"))
		default:
			response.RenderError(w, r, response.NewInternalError(err))
		}
		return
	}

	// Set session cookie
	session.SetSessionCookie(w, tokenPair.Token, sess.IdleExpiresAt, h.isProduction)

	response.JSON(w, r, http.StatusOK, &VerificationResponse{
		User:               userView,
		OnboardingRequired: userView.OnboardingRequired,
		CSRFToken:          csrfToken,
	})
}

func (h *Handler) StartGoogle(w http.ResponseWriter, r *http.Request) {
	returnPath := r.URL.Query().Get("return_path")
	authURL, state, verifier, err := h.service.StartGoogleOAuth(r.Context(), returnPath)
	if err != nil {
		if errors.Is(err, ErrGoogleOAuthNotConfigured) {
			response.RenderError(w, r, response.NewBadRequestError("Google OAuth belum dikonfigurasi pada server backend (GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET belum diisi di .env)"))
			return
		}
		if errors.Is(err, ErrInvalidReturnPath) {
			response.RenderError(w, r, response.NewBadRequestError(err.Error()))
			return
		}
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	for name, value := range map[string]string{"xynoss_oauth_state": state, "xynoss_oauth_verifier": verifier} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: h.googleCallbackPath(), MaxAge: 900, HttpOnly: true, Secure: h.isProduction, SameSite: http.SameSiteLaxMode})
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	userAgent := r.UserAgent()
	stateCookie, stateErr := r.Cookie("xynoss_oauth_state")
	verifierCookie, verifierErr := r.Cookie("xynoss_oauth_verifier")
	for _, name := range []string{"xynoss_oauth_state", "xynoss_oauth_verifier"} {
		http.SetCookie(w, &http.Cookie{Name: name, Path: h.googleCallbackPath(), MaxAge: -1, HttpOnly: true, Secure: h.isProduction, SameSite: http.SameSiteLaxMode})
	}
	if stateErr != nil || verifierErr != nil || subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie.Value)) != 1 {
		http.Redirect(w, r, h.frontendURL+"/masuk?error=oauth_failed", http.StatusFound)
		return
	}

	tokenPair, sess, _, returnPath, err := h.service.HandleGoogleCallback(r.Context(), code, state, verifierCookie.Value, userAgent)
	if err != nil {
		redirectErrURL := h.frontendURL + "/masuk?error=oauth_failed"
		if errors.Is(err, ErrIdentityNotLinked) {
			redirectErrURL = h.frontendURL + "/masuk?error=account_not_linked"
		}
		http.Redirect(w, r, redirectErrURL, http.StatusFound)
		return
	}

	// Set session cookie
	session.SetSessionCookie(w, tokenPair.Token, sess.IdleExpiresAt, h.isProduction)

	targetURL := h.frontendURL + returnPath
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (h *Handler) GetSessionBootstrap(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if !ok {
		response.RenderError(w, r, response.NewUnauthorizedError("no active session"))
		return
	}

	userView, err := h.usersService.GetCurrentUser(r.Context(), actor.UserID)
	if err != nil {
		response.RenderError(w, r, response.NewInternalError(err))
		return
	}

	csrfToken := h.sessionService.GetCSRFToken(actor.SessionID)

	response.JSON(w, r, http.StatusOK, map[string]any{
		"user":                userView,
		"onboarding_required": userView.OnboardingRequired,
		"csrf_token":          csrfToken,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	actor, ok := authctx.GetActor(r.Context())
	if ok && actor.SessionID != "" && !actor.ManagedSession {
		if err := h.sessionService.RevokeSession(r.Context(), actor.UserID, actor.SessionID); err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			response.RenderError(w, r, response.NewInternalError(err))
			return
		}
	}

	session.ClearSessionCookie(w, h.isProduction)
	w.WriteHeader(http.StatusNoContent)
}
