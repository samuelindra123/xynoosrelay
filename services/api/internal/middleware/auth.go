package middleware

import (
	"net/http"

	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/response"
	"github.com/xynoss/relay/services/api/internal/session"
)

// Authenticate extracts session tokens from cookie or Authorization header and attaches Actor to context.
func Authenticate(sessionService *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := session.ExtractSessionToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			sess, err := sessionService.ValidateSessionToken(r.Context(), token)
			if err != nil {
				// Invalid or expired token: continue as unauthenticated; endpoints with RequireAuth will reject
				next.ServeHTTP(w, r)
				return
			}

			actor := &authctx.Actor{
				UserID:         sess.UserID,
				SessionID:      sess.ID,
				ManagedSession: sess.Managed,
			}

			ctx := authctx.WithActor(r.Context(), actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth enforces that an authenticated actor is present in the request context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := authctx.GetActor(r.Context())
		if !ok || actor.UserID == "" {
			response.RenderError(w, r, response.NewUnauthorizedError("authentication required for this endpoint"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
