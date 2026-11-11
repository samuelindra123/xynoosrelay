package middleware

import (
	"net/http"
	"strings"

	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
	"github.com/xynoss/relay/services/api/internal/session"
)

// RequireCSRF validates the CSRF token on mutating requests if an authenticated session is present.
func RequireCSRF(sessionService *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check mutating HTTP methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// If caller is a verified internal service with API key, CSRF is not required
			if IsInternalService(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			// If request uses explicit Authorization header (Bearer token), CSRF is not required (OWASP)
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			actor, ok := authctx.GetActor(r.Context())
			if !ok || actor.SessionID == "" {
				// If not authenticated, let route handlers enforce their own auth requirements
				next.ServeHTTP(w, r)
				return
			}

			csrfToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if csrfToken == "" {
				response.RenderError(w, r, response.NewForbiddenError("missing required X-CSRF-Token header"))
				return
			}

			if !sessionService.VerifyCSRFToken(csrfToken, actor.SessionID) {
				response.RenderError(w, r, response.NewForbiddenError("invalid or expired CSRF token"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
