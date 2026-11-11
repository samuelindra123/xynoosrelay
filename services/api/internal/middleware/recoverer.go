package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/internal/response"
)

// Recoverer recovers from panics and returns a standardized 500 error envelope.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					reqID := logger.GetRequestID(r.Context())
					stack := string(debug.Stack())

					log.ErrorContext(r.Context(), "http_panic_recovered",
						slog.String("request_id", reqID),
						slog.Any("panic", rvr),
						slog.String("stack", stack),
					)

					response.RenderError(w, r, response.NewInternalError(fmt.Errorf("panic: %v", rvr)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
