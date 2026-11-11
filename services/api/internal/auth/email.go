package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/xynoss/relay/services/api/internal/config"
)

// EmailSender defines the email dispatch contract for authentication challenges.
type EmailSender interface {
	SendOTP(ctx context.Context, toEmail, code string) error
}

// ResendEmailSender dispatches real transactional emails using the Resend API.
type ResendEmailSender struct {
	apiKey     string
	from       string
	httpClient *http.Client
	log        *slog.Logger
}

// NewResendEmailSender creates a new Resend email dispatcher.
func NewResendEmailSender(apiKey, from string, log *slog.Logger) *ResendEmailSender {
	if from == "" {
		from = "XYNOSS Relay <auth@xynoss.app>"
	}
	return &ResendEmailSender{
		apiKey:     apiKey,
		from:       from,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        log,
	}
}

func (r *ResendEmailSender) SendOTP(ctx context.Context, toEmail, code string) error {
	payload := map[string]any{
		"from":    r.from,
		"to":      []string{toEmail},
		"subject": "Kode verifikasi XYNOSS Relay",
		"html": fmt.Sprintf(`
			<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; max-width: 480px; margin: 0 auto; padding: 28px; border: 1px solid #e0e4d8; border-radius: 16px; background: #fbfaf5; color: #273129;">
				<h2 style="margin: 0 0 12px; font-size: 20px; font-weight: 800; letter-spacing: -0.05em;">XYNOSS <span style="font-weight: 400; color: #6d7b6d;">Relay</span></h2>
				<p style="color: #637365; font-size: 14px; line-height: 1.6; margin: 0 0 20px;">Gunakan kode verifikasi berikut untuk masuk ke akun kamu. Tanpa kata sandi.</p>
				<div style="background: #253c31; color: #e9f0e5; font-size: 32px; font-weight: 800; letter-spacing: 8px; text-align: center; padding: 18px 24px; border-radius: 12px; margin: 20px 0; font-family: monospace;">
					%s
				</div>
				<p style="color: #8c998b; font-size: 12px; line-height: 1.6; margin: 20px 0 0;">Kode ini hanya berlaku selama 10 menit. Jika kamu tidak meminta kode ini, abaikan email ini.</p>
			</div>
		`, code),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to build resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		if r.log != nil {
			r.log.Error("failed to dispatch email via Resend", "err", err)
		}
		return fmt.Errorf("resend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		if r.log != nil {
			r.log.Error("Resend API rejected email", "status", resp.StatusCode)
		}
		return fmt.Errorf("resend returned status %d", resp.StatusCode)
	}

	if r.log != nil {
		r.log.Info("email successfully dispatched via Resend")
	}
	return nil
}

// SMTPEmailSender dispatches real transactional emails via standard SMTP.
type SMTPEmailSender struct {
	host string
	port string
	user string
	pass string
	from string
	log  *slog.Logger
}

func NewSMTPEmailSender(host, port, user, pass, from string, log *slog.Logger) *SMTPEmailSender {
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = user
	}
	return &SMTPEmailSender{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
		log:  log,
	}
}

func (s *SMTPEmailSender) SendOTP(ctx context.Context, toEmail, code string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	var auth smtp.Auth
	if s.user != "" && s.pass != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}

	subject := fmt.Sprintf("Subject: Kode Verifikasi XYNOSS Relay: %s\r\n", code)
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 24px; border: 1px solid #e0e4d8; border-radius: 12px; background: #fbfaf5;">
			<h2 style="color: #273129; margin-bottom: 8px;">XYNOSS Relay</h2>
			<p style="color: #6d7b6d; font-size: 14px;">Gunakan kode verifikasi berikut untuk masuk ke akun kamu:</p>
			<div style="background: #253c31; color: #e9f0e5; font-size: 28px; font-weight: bold; letter-spacing: 6px; text-align: center; padding: 16px; border-radius: 8px; margin: 20px 0;">
				%s
			</div>
			<p style="color: #8c998b; font-size: 12px;">Kode ini hanya berlaku selama 10 menit. Jangan bagikan kode ini kepada siapa pun.</p>
		</div>
	`, code)

	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\n%s%s%s", toEmail, s.from, subject, mime, body))

	err := smtp.SendMail(addr, auth, s.from, []string{toEmail}, msg)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to dispatch email via SMTP", "err", err)
		}
		return fmt.Errorf("smtp send failed: %w", err)
	}

	if s.log != nil {
		s.log.Info("email successfully dispatched via SMTP", "recipient", toEmail)
	}
	return nil
}

// TestDoubleEmailSender provides an in-memory test double that records OTPs for test verification.
type TestDoubleEmailSender struct {
	mu        sync.RWMutex
	lastCodes map[string]string
	log       *slog.Logger
}

// NewTestDoubleEmailSender creates a new test double email sender.
func NewTestDoubleEmailSender(log *slog.Logger) *TestDoubleEmailSender {
	return &TestDoubleEmailSender{
		lastCodes: make(map[string]string),
		log:       log,
	}
}

func (t *TestDoubleEmailSender) SendOTP(ctx context.Context, toEmail, code string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	normalized := strings.ToLower(strings.TrimSpace(toEmail))
	t.lastCodes[normalized] = code

	if t.log != nil {
		t.log.Info("test double: OTP challenge dispatched")
	}

	return nil
}

// GetLastCode retrieves the most recent code sent to an email (for automated test verification).
func (t *TestDoubleEmailSender) GetLastCode(toEmail string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	normalized := strings.ToLower(strings.TrimSpace(toEmail))
	return t.lastCodes[normalized]
}

// Clear flushes all recorded test codes.
func (t *TestDoubleEmailSender) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastCodes = make(map[string]string)
}

// NewConfiguredEmailSender creates the appropriate email sender based on environment config.
func NewConfiguredEmailSender(cfg *config.Config, log *slog.Logger) EmailSender {
	if cfg.ResendAPIKey != "" {
		if log != nil {
			log.Info("using real Resend email provider", "from", cfg.EmailFrom)
		}
		return NewResendEmailSender(cfg.ResendAPIKey, cfg.EmailFrom, log)
	}
	if cfg.SMTPHost != "" {
		if log != nil {
			log.Info("using real SMTP email provider", "host", cfg.SMTPHost, "from", cfg.EmailFrom)
		}
		return NewSMTPEmailSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.EmailFrom, log)
	}
	if log != nil {
		log.Info("no external email provider configured; using local dev sandbox")
	}
	return NewTestDoubleEmailSender(log)
}

func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return "unknown"
}
