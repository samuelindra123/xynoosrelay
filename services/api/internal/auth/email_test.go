package auth

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/xynoss/relay/services/api/internal/config"
)

func TestResendEmailSenderRequiresExplicitCredentialsAndSender(t *testing.T) {
	tests := []struct {
		name string
		key  string
		from string
	}{
		{name: "missing key", from: "Relay <noreply@example.com>"},
		{name: "missing sender", key: "test-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewResendEmailSender(tt.key, tt.from, nil)
			if err := sender.SendOTP(context.Background(), "person@example.com", "123456"); err == nil {
				t.Fatal("expected unconfigured sender to fail before making a request")
			}
		})
	}
}

func TestConfiguredEmailSenderSelectsSafeProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("sandbox when no provider is configured", func(t *testing.T) {
		sender := NewConfiguredEmailSender(&config.Config{}, logger)
		sandbox, ok := sender.(*TestDoubleEmailSender)
		if !ok {
			t.Fatalf("provider type = %T, want *TestDoubleEmailSender", sender)
		}
		if err := sandbox.SendOTP(context.Background(), " User@Example.com ", "123456"); err != nil {
			t.Fatalf("sandbox SendOTP() error = %v", err)
		}
		if got := sandbox.GetLastCode("user@example.com"); got != "123456" {
			t.Fatalf("sandbox code = %q, want 123456", got)
		}
	})

	t.Run("Resend takes precedence over SMTP", func(t *testing.T) {
		sender := NewConfiguredEmailSender(&config.Config{
			ResendAPIKey: "test-key",
			EmailFrom:    "Relay <noreply@example.com>",
			SMTPHost:     "smtp.example.com",
		}, logger)
		if _, ok := sender.(*ResendEmailSender); !ok {
			t.Fatalf("provider type = %T, want *ResendEmailSender", sender)
		}
	})

	t.Run("SMTP provider uses configured fallback values", func(t *testing.T) {
		sender := NewConfiguredEmailSender(&config.Config{
			SMTPHost: "smtp.example.com",
			SMTPUser: "relay@example.com",
			SMTPPass: "password",
		}, logger)
		smtpSender, ok := sender.(*SMTPEmailSender)
		if !ok {
			t.Fatalf("provider type = %T, want *SMTPEmailSender", sender)
		}
		if smtpSender.port != "587" || smtpSender.from != smtpSender.user {
			t.Fatalf("SMTP defaults = port %q, from %q; want 587 and user", smtpSender.port, smtpSender.from)
		}
	})
}
