package middleware

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/xynoss/relay/services/api/internal/logger"
)

var validRequestIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)

// RequestID middleware ensures every request has a validated or newly generated X-Request-ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		incomingID := r.Header.Get("X-Request-ID")
		var requestID string

		if incomingID != "" && validRequestIDPattern.MatchString(incomingID) {
			requestID = incomingID
		} else {
			requestID = uuid.NewString()
		}

		ctx := logger.WithRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
