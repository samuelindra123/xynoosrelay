package session

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	pair, err := GenerateTokenPair()
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	sessionID, verifierSecret, err := ParseToken(pair.Token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if sessionID != pair.SessionID {
		t.Errorf("expected sessionID %s, got %s", pair.SessionID, sessionID)
	}

	if !VerifyTokenHash(verifierSecret, pair.VerifierHash) {
		t.Errorf("VerifyTokenHash failed for valid secret")
	}

	// Tampered verifier secret should fail
	if VerifyTokenHash("wrongsecret", pair.VerifierHash) {
		t.Errorf("VerifyTokenHash succeeded for invalid secret")
	}
}

func TestCSRFToken(t *testing.T) {
	hashKey := "test-session-secret-key-at-least-32-chars-long"
	sessionID := "11111111-2222-3333-4444-555555555555"

	token := GenerateCSRFToken(sessionID, hashKey)
	if token == "" {
		t.Fatalf("expected non-empty CSRF token")
	}

	if !VerifyCSRFToken(token, sessionID, hashKey) {
		t.Errorf("expected CSRF token to be valid")
	}

	// Tampered token should fail
	if VerifyCSRFToken("invalid-csrf-token", sessionID, hashKey) {
		t.Errorf("expected invalid CSRF token to fail")
	}

	// Wrong session ID should fail
	if VerifyCSRFToken(token, "other-session-id", hashKey) {
		t.Errorf("expected token for different session to fail")
	}
}

func TestParseDeviceHint(t *testing.T) {
	tests := []struct {
		ua           string
		wantPlatform string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15", "macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15", "iOS"},
		{"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/120.0.0.0 Mobile Safari/537.36", "Android"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		_, platform := ParseDeviceHint(tt.ua)
		if platform != tt.wantPlatform {
			t.Errorf("for UA %q: got platform %s, want %s", tt.ua, platform, tt.wantPlatform)
		}
	}
}

func TestSession_IsActive(t *testing.T) {
	now := time.Now().UTC()

	activeSess := &Session{
		IdleExpiresAt:     now.Add(1 * time.Hour),
		AbsoluteExpiresAt: now.Add(24 * time.Hour),
		RevokedAt:         nil,
	}
	if !activeSess.IsActive(now) {
		t.Errorf("expected active session to be active")
	}

	revokedSess := &Session{
		IdleExpiresAt:     now.Add(1 * time.Hour),
		AbsoluteExpiresAt: now.Add(24 * time.Hour),
		RevokedAt:         &now,
	}
	if revokedSess.IsActive(now) {
		t.Errorf("expected revoked session to not be active")
	}

	expiredIdle := &Session{
		IdleExpiresAt:     now.Add(-1 * time.Minute),
		AbsoluteExpiresAt: now.Add(24 * time.Hour),
		RevokedAt:         nil,
	}
	if expiredIdle.IsActive(now) {
		t.Errorf("expected idle-expired session to not be active")
	}
}
