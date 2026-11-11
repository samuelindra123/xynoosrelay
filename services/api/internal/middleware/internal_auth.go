package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/xynoss/relay/services/api/internal/response"
)

const HeaderInternalAPIKey = "X-Internal-API-Key"

type internalServiceContextKey struct{}

// InternalService represents a verified caller service identity.
type InternalService struct {
	Verified bool
}

// WithInternalService marks the request as originating from a trusted internal service.
func WithInternalService(ctx context.Context, s *InternalService) context.Context {
	return context.WithValue(ctx, internalServiceContextKey{}, s)
}

// IsInternalService returns true if the request was verified via valid internal API key.
func IsInternalService(ctx context.Context) bool {
	s, ok := ctx.Value(internalServiceContextKey{}).(*InternalService)
	return ok && s != nil && s.Verified
}

// RequireInternalAPIKey enforces that incoming requests carry a valid internal API key.
func RequireInternalAPIKey(validKeys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			candidateKey := strings.TrimSpace(r.Header.Get(HeaderInternalAPIKey))
			if candidateKey == "" {
				response.RenderError(w, r, response.NewUnauthorizedError("missing required internal API key"))
				return
			}

			matched := false
			for _, k := range validKeys {
				if len(k) > 0 && subtle.ConstantTimeCompare([]byte(candidateKey), []byte(k)) == 1 {
					matched = true
					break
				}
			}

			if !matched {
				response.RenderError(w, r, response.NewUnauthorizedError("invalid internal API key"))
				return
			}

			ctx := WithInternalService(r.Context(), &InternalService{Verified: true})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
