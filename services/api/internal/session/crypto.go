package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidTokenFormat = errors.New("invalid session token format")
)

// GenerateTokenPair generates a new session ID, a 256-bit random verifier secret, and its SHA-256 hash.
func GenerateTokenPair() (*TokenPair, error) {
	sessionID := uuid.NewString()

	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate cryptographically secure random bytes: %w", err)
	}

	verifierSecret := hex.EncodeToString(verifierBytes)
	hash := sha256.Sum256([]byte(verifierSecret))
	verifierHash := hex.EncodeToString(hash[:])

	token := fmt.Sprintf("%s.%s", sessionID, verifierSecret)

	return &TokenPair{
		Token:        token,
		SessionID:    sessionID,
		VerifierHash: verifierHash,
	}, nil
}

// ParseToken extracts the session ID and verifier secret from a composite session token.
func ParseToken(rawToken string) (sessionID, verifierSecret string, err error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidTokenFormat
	}
	if _, err := uuid.Parse(parts[0]); err != nil {
		return "", "", ErrInvalidTokenFormat
	}
	return parts[0], parts[1], nil
}

// VerifyTokenHash checks if candidate verifierSecret matches storedHash in constant time.
func VerifyTokenHash(verifierSecret, storedHash string) bool {
	candidate := sha256.Sum256([]byte(verifierSecret))
	candidateHex := hex.EncodeToString(candidate[:])

	return subtle.ConstantTimeCompare([]byte(candidateHex), []byte(storedHash)) == 1
}

// GenerateCSRFToken generates a cryptographic CSRF token bound to the session ID.
func GenerateCSRFToken(sessionID, hashKey string) string {
	mac := hmac.New(sha256.New, []byte(hashKey))
	mac.Write([]byte("xynoss-csrf-token:" + sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyCSRFToken validates an incoming CSRF token in constant time.
func VerifyCSRFToken(candidateToken, sessionID, hashKey string) bool {
	if candidateToken == "" || sessionID == "" || hashKey == "" {
		return false
	}
	expected := GenerateCSRFToken(sessionID, hashKey)
	return subtle.ConstantTimeCompare([]byte(candidateToken), []byte(expected)) == 1
}

// ParseDeviceHint parses User-Agent into a friendly label and coarse platform hint.
func ParseDeviceHint(ua string) (label, platformHint string) {
	if ua == "" {
		return "Unknown Device", "Unknown"
	}

	lower := strings.ToLower(ua)

	// Platform (Check mobile iOS before macOS because iOS UA contains 'like Mac OS X')
	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		platformHint = "iOS"
	case strings.Contains(lower, "windows"):
		platformHint = "Windows"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		platformHint = "macOS"
	case strings.Contains(lower, "android"):
		platformHint = "Android"
	case strings.Contains(lower, "linux"):
		platformHint = "Linux"
	default:
		platformHint = "Web"
	}

	// Browser / App Label
	switch {
	case strings.Contains(lower, "edg/"):
		label = fmt.Sprintf("Microsoft Edge on %s", platformHint)
	case strings.Contains(lower, "chrome/"):
		label = fmt.Sprintf("Chrome on %s", platformHint)
	case strings.Contains(lower, "firefox/"):
		label = fmt.Sprintf("Firefox on %s", platformHint)
	case strings.Contains(lower, "safari/"):
		label = fmt.Sprintf("Safari on %s", platformHint)
	default:
		label = fmt.Sprintf("Browser on %s", platformHint)
	}

	return label, platformHint
}
