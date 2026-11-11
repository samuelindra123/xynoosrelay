package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Sensitive keys to be redacted in log attributes
var sensitiveKeyPrefixes = []string{
	"token", "secret", "otp", "code", "password",
	"authorization", "cookie", "key", "verifier",
}

// RedactLogValue redacts any string containing sensitive fields
func RedactLogValue(val string) string {
	if len(val) == 0 {
		return ""
	}
	return "[REDACTED]"
}

// NewJSONHandler creates a structured slog JSON handler that redacts sensitive attributes.
func NewJSONHandler(w io.Writer, minLevel slog.Level) *slog.JSONHandler {
	opts := &slog.HandlerOptions{
		Level: minLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			lowerKey := strings.ToLower(a.Key)
			for _, sensitive := range sensitiveKeyPrefixes {
				if strings.Contains(lowerKey, sensitive) {
					return slog.Attr{
						Key:   a.Key,
						Value: slog.StringValue("[REDACTED]"),
					}
				}
			}
			return a
		},
	}
	return slog.NewJSONHandler(w, opts)
}

// SetupLogger initializes the default slog logger.
func SetupLogger(appEnv string) *slog.Logger {
	level := slog.LevelInfo
	if appEnv == "development" || appEnv == "test" {
		level = slog.LevelDebug
	}

	handler := NewJSONHandler(os.Stdout, level)
	l := slog.New(handler)
	slog.SetDefault(l)
	return l
}

type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}
