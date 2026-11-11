package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
)

// Store defines operations on sessions and devices.
type Store interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	GetBetterAuthSession(ctx context.Context, token string) (*Session, error)
	TouchSession(ctx context.Context, id string, lastSeen, idleExpires time.Time) error
	RevokeSession(ctx context.Context, id string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	ListUserSessions(ctx context.Context, userID string) ([]*Session, error)
	RegisterOrUpdateDevice(ctx context.Context, d *Device) (string, error)
}

// PGStore implements Store backed by PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL-backed session store.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) CreateSession(ctx context.Context, sess *Session) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		INSERT INTO sessions (
			id, user_id, verifier_hash, created_at, last_seen_at,
			idle_expires_at, absolute_expires_at, device_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		);
	`

	_, err := s.pool.Exec(ctx, query,
		sess.ID, sess.UserID, sess.VerifierHash, sess.CreatedAt, sess.LastSeenAt,
		sess.IdleExpiresAt, sess.AbsoluteExpiresAt, sess.DeviceID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	return nil
}

func (s *PGStore) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	query := `
		SELECT s.id, s.user_id, s.verifier_hash, s.created_at, s.last_seen_at,
		       s.idle_expires_at, s.absolute_expires_at, s.revoked_at, s.device_id,
		       COALESCE(d.label, ''), COALESCE(d.platform_hint, '')
		FROM sessions s
		LEFT JOIN devices d ON d.id = s.device_id
		WHERE s.id = $1
		LIMIT 1;
	`

	var sess Session
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&sess.ID, &sess.UserID, &sess.VerifierHash, &sess.CreatedAt, &sess.LastSeenAt,
		&sess.IdleExpiresAt, &sess.AbsoluteExpiresAt, &sess.RevokedAt, &sess.DeviceID,
		&sess.DeviceLabel, &sess.PlatformHint,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	return &sess, nil
}

func (s *PGStore) GetBetterAuthSession(ctx context.Context, token string) (*Session, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	// 1. Try neon_auth schema (Better Auth uses camelCase quoted columns: "userId", "expiresAt")
	query := `
		SELECT s.id::text, s."userId"::text, s."expiresAt", COALESCE(u.name, 'Pengguna'), COALESCE(u.email, ''), COALESCE(u.image, '')
		FROM neon_auth.session s
		LEFT JOIN neon_auth.user u ON u.id = s."userId"
		WHERE s.token = $1 AND s."expiresAt" > NOW()
		LIMIT 1;
	`

	var sessID, userID, name, email, image string
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx, query, token).Scan(
		&sessID, &userID, &expiresAt, &name, &email, &image,
	)
	if err != nil {
		// 2. Try fallback to public schema if neon_auth is not prefixed
		altQuery := `
			SELECT s.id::text, COALESCE(s."userId", s.user_id)::text, COALESCE(s."expiresAt", s.expires_at), COALESCE(u.name, 'Pengguna'), COALESCE(u.email, ''), COALESCE(u.image, '')
			FROM session s
			LEFT JOIN "user" u ON u.id = COALESCE(s."userId", s.user_id)
			WHERE s.token = $1 AND COALESCE(s."expiresAt", s.expires_at) > NOW()
			LIMIT 1;
		`
		err2 := s.pool.QueryRow(ctx, altQuery, token).Scan(
			&sessID, &userID, &expiresAt, &name, &email, &image,
		)
		if err2 != nil {
			return nil, ErrSessionNotFound
		}
	}

	// Ensure domain user & profile exist in XYNOSS Relay tables
	s.ensureDomainUserAndProfile(ctx, userID, name, email, image)

	now := time.Now().UTC()
	return &Session{
		Managed:           true,
		ID:                sessID,
		UserID:            userID,
		CreatedAt:         now,
		LastSeenAt:        now,
		IdleExpiresAt:     expiresAt,
		AbsoluteExpiresAt: expiresAt,
		DeviceLabel:       "Neon Auth",
		PlatformHint:      "web",
	}, nil
}

func (s *PGStore) ensureDomainUserAndProfile(ctx context.Context, userID, name, email, avatarURL string) {
	username := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	if username == "" {
		if len(userID) >= 8 {
			username = "user_" + userID[:8]
		} else {
			username = "user_" + userID
		}
	}

	// Check if username is already taken by a different user
	var exists bool
	_ = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM public.users WHERE username_normalized = $1 AND id != $2::uuid)", username, userID).Scan(&exists)
	if exists {
		suffix := userID
		if len(suffix) > 6 {
			suffix = suffix[:6]
		}
		username = fmt.Sprintf("%s_%s", username, suffix)
	}

	// 1. users table
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO public.users (id, username, username_normalized, created_at)
		VALUES ($1::uuid, $2, $3, NOW())
		ON CONFLICT (id) DO NOTHING;
	`, userID, username, username)

	// 2. profiles table
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO public.profiles (user_id, display_name, bio, availability, revision, updated_at)
		VALUES ($1::uuid, $2, '', 'online', 1, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = CASE WHEN profiles.display_name = '' THEN EXCLUDED.display_name ELSE profiles.display_name END,
			updated_at = NOW();
	`, userID, name)

	// 3. identities table
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO public.identities (user_id, provider, provider_subject, verified_email_normalized, verified_at)
		VALUES ($1::uuid, 'neon_auth', $2, $3, NOW())
		ON CONFLICT (provider, provider_subject) DO NOTHING;
	`, userID, userID, strings.ToLower(email))

	// 4. preferences table
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO public.preferences (user_id, settings_version, updated_at)
		VALUES ($1::uuid, 1, NOW())
		ON CONFLICT (user_id) DO NOTHING;
	`, userID)
}

func (s *PGStore) TouchSession(ctx context.Context, id string, lastSeen, idleExpires time.Time) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE sessions
		SET last_seen_at = $2,
		    idle_expires_at = LEAST($3, absolute_expires_at)
		WHERE id = $1 AND revoked_at IS NULL;
	`

	_, err := s.pool.Exec(ctx, query, id, lastSeen, idleExpires)
	return err
}

func (s *PGStore) RevokeSession(ctx context.Context, id string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL;
	`

	res, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *PGStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL;
	`

	_, err := s.pool.Exec(ctx, query, userID)
	return err
}

func (s *PGStore) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	query := `
		SELECT s.id, s.user_id, s.created_at, s.last_seen_at,
		       s.idle_expires_at, s.absolute_expires_at, s.revoked_at, s.device_id,
		       COALESCE(d.label, 'Unknown Device'), COALESCE(d.platform_hint, 'Web')
		FROM sessions s
		LEFT JOIN devices d ON d.id = s.device_id
		WHERE s.user_id = $1 AND s.revoked_at IS NULL AND s.idle_expires_at > NOW() AND s.absolute_expires_at > NOW()
		ORDER BY s.last_seen_at DESC;
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(
			&sess.ID, &sess.UserID, &sess.CreatedAt, &sess.LastSeenAt,
			&sess.IdleExpiresAt, &sess.AbsoluteExpiresAt, &sess.RevokedAt, &sess.DeviceID,
			&sess.DeviceLabel, &sess.PlatformHint,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, &sess)
	}

	return sessions, rows.Err()
}

func (s *PGStore) RegisterOrUpdateDevice(ctx context.Context, d *Device) (string, error) {
	if s.pool == nil {
		return "", errors.New("database pool is disconnected")
	}

	query := `
		INSERT INTO devices (
			user_id, label, platform_hint, user_agent, first_seen, last_seen
		) VALUES (
			$1, $2, $3, $4, NOW(), NOW()
		) RETURNING id;
	`

	var deviceID string
	err := s.pool.QueryRow(ctx, query, d.UserID, d.Label, d.PlatformHint, d.UserAgent).Scan(&deviceID)
	if err != nil {
		return "", fmt.Errorf("failed to register device: %w", err)
	}

	return deviceID, nil
}
