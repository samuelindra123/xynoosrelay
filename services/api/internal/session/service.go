package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultIdleTimeout     = 7 * 24 * time.Hour  // 7 days idle expiry
	DefaultAbsoluteTimeout = 30 * 24 * time.Hour // 30 days absolute expiry
)

var (
	ErrUnauthorizedSessionAccess = errors.New("unauthorized to access or modify this session")
	ErrInvalidVerifier           = errors.New("invalid session verifier secret")
)

// Service coordinates session lifecycle, token generation, cryptographic validation, and CSRF protection.
type Service struct {
	store          Store
	sessionHashKey string
}

// NewService constructs a new session service instance.
func NewService(store Store, sessionHashKey string) *Service {
	return &Service{
		store:          store,
		sessionHashKey: sessionHashKey,
	}
}

// CreateSession issues a new 256-bit opaque session token and persists the session.
func (s *Service) CreateSession(ctx context.Context, userID string, userAgent string) (*TokenPair, *Session, string, error) {
	if userID == "" {
		return nil, nil, "", errors.New("user ID is required")
	}

	tokenPair, err := GenerateTokenPair()
	if err != nil {
		return nil, nil, "", err
	}

	label, platformHint := ParseDeviceHint(userAgent)
	deviceID, _ := s.store.RegisterOrUpdateDevice(ctx, &Device{
		UserID:       userID,
		Label:        label,
		PlatformHint: platformHint,
		UserAgent:    userAgent,
	})

	now := time.Now().UTC()
	sess := &Session{
		ID:                tokenPair.SessionID,
		UserID:            userID,
		VerifierHash:      tokenPair.VerifierHash,
		CreatedAt:         now,
		LastSeenAt:        now,
		IdleExpiresAt:     now.Add(DefaultIdleTimeout),
		AbsoluteExpiresAt: now.Add(DefaultAbsoluteTimeout),
		DeviceLabel:       label,
		PlatformHint:      platformHint,
	}
	if deviceID != "" {
		sess.DeviceID = &deviceID
	}

	if err := s.store.CreateSession(ctx, sess); err != nil {
		return nil, nil, "", fmt.Errorf("failed to save session: %w", err)
	}

	csrfToken := GenerateCSRFToken(sess.ID, s.sessionHashKey)

	return tokenPair, sess, csrfToken, nil
}

// ValidateSessionToken parses, inspects, and validates a session token in constant time.
func (s *Service) ValidateSessionToken(ctx context.Context, rawToken string) (*Session, error) {
	// 1. If token contains '.', it's a native XYNOSS Relay token (sessionID.verifier)
	if strings.Contains(rawToken, ".") {
		sessionID, verifierSecret, err := ParseToken(rawToken)
		if err == nil {
			sess, err := s.store.GetSessionByID(ctx, sessionID)
			if err == nil {
				now := time.Now().UTC()
				if sess.RevokedAt != nil {
					return nil, ErrSessionRevoked
				}
				if now.After(sess.AbsoluteExpiresAt) || now.After(sess.IdleExpiresAt) {
					return nil, ErrSessionExpired
				}
				if VerifyTokenHash(verifierSecret, sess.VerifierHash) {
					// Extend idle timeout
					newIdle := now.Add(DefaultIdleTimeout)
					_ = s.store.TouchSession(ctx, sess.ID, now, newIdle)
					return sess, nil
				}
			}
		}
	}

	return nil, ErrSessionNotFound
}

// RevokeSession ensures the actor owns the target session before revoking.
func (s *Service) RevokeSession(ctx context.Context, actorID, sessionID string) error {
	sess, err := s.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if sess.UserID != actorID {
		return ErrUnauthorizedSessionAccess
	}

	return s.store.RevokeSession(ctx, sessionID)
}

// ListUserSessions returns active sessions for a user, highlighting the current session.
func (s *Service) ListUserSessions(ctx context.Context, userID, currentSessionID string) ([]*Session, error) {
	sessions, err := s.store.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, sess := range sessions {
		sess.IsCurrent = (sess.ID == currentSessionID)
	}

	return sessions, nil
}

// GetCSRFToken creates or retrieves the valid CSRF token for a given session ID.
func (s *Service) GetCSRFToken(sessionID string) string {
	return GenerateCSRFToken(sessionID, s.sessionHashKey)
}

// VerifyCSRFToken checks an incoming CSRF token against a session ID.
func (s *Service) VerifyCSRFToken(candidateToken, sessionID string) bool {
	return VerifyCSRFToken(candidateToken, sessionID, s.sessionHashKey)
}
