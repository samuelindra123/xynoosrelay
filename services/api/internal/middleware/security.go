package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/xynoss/relay/services/api/internal/response"
)

// SecurityHeaders applies standard defensive HTTP response headers.
func SecurityHeaders(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			if isProduction {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Enforce Content-Type: application/json for mutations with request body
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				if r.ContentLength > 0 || r.Header.Get("Transfer-Encoding") == "chunked" {
					ct := r.Header.Get("Content-Type")
					mediaType, _, err := mime.ParseMediaType(ct)
					if err != nil || strings.ToLower(mediaType) != "application/json" {
						response.RenderError(w, r, response.NewError(
							http.StatusUnsupportedMediaType,
							response.CodeMalformedRequest,
							"Content-Type must be application/json",
						))
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
