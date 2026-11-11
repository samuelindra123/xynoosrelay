package session

import (
	"time"
)

// Session represents an active or revoked authenticated session in PostgreSQL.
type Session struct {
	Managed           bool       `json:"-"`
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	VerifierHash      string     `json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	LastSeenAt        time.Time  `json:"last_seen_at"`
	IdleExpiresAt     time.Time  `json:"idle_expires_at"`
	AbsoluteExpiresAt time.Time  `json:"absolute_expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	DeviceID          *string    `json:"device_id,omitempty"`
	DeviceLabel       string     `json:"label,omitempty"`
	PlatformHint      string     `json:"platform_hint,omitempty"`
	IsCurrent         bool       `json:"is_current"`
}

// IsActive returns true if the session has not been revoked and neither timeout has elapsed.
func (s *Session) IsActive(now time.Time) bool {
	if s.RevokedAt != nil {
		return false
	}
	if now.After(s.IdleExpiresAt) || now.After(s.AbsoluteExpiresAt) {
		return false
	}
	return true
}

// Device represents client device metadata.
type Device struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Label        string    `json:"label"`
	PlatformHint string    `json:"platform_hint"`
	UserAgent    string    `json:"user_agent"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
}

// TokenPair contains the plaintext session token returned to the client and its hash for DB.
type TokenPair struct {
	Token        string // Plaintext format: <session_id>.<verifier_secret>
	SessionID    string
	VerifierHash string
}
