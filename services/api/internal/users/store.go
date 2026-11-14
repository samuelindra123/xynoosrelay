package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrProfileNotFound  = errors.New("profile not found")
	ErrUsernameConflict = errors.New("username is already taken")
	ErrRevisionMismatch = errors.New("profile revision mismatch; please refresh and try again")
	ErrIdentityNotFound = errors.New("identity not found")
)

// Store defines operations on users, profiles, and identities.
type Store interface {
	CreateUserWithProfile(ctx context.Context, emailNormalized, provider, subject string) (*User, *Profile, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetProfile(ctx context.Context, userID string) (*Profile, error)
	UpdateProfile(ctx context.Context, userID string, p *Profile, expectedRevision int) (*Profile, error)
	SetUsername(ctx context.Context, userID, username, usernameNormalized string) error
	SetAvatar(ctx context.Context, userID string, mediaID *string) error
	SetBanner(ctx context.Context, userID string, mediaID *string) error
	CompleteOnboarding(ctx context.Context, userID, displayName string, username *string) (*User, *Profile, error)
	FindIdentity(ctx context.Context, provider, subject string) (*Identity, error)
	FindUserByEmail(ctx context.Context, emailNormalized string) (*User, error)
	LinkIdentity(ctx context.Context, userID, provider, subject, emailNormalized string) (*Identity, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]*UserSearchResult, error)
}

// PGStore implements Store backed by PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL users store.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) CreateUserWithProfile(ctx context.Context, emailNormalized, provider, subject string) (*User, *Profile, error) {
	if s.pool == nil {
		return nil, nil, errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Create user
	var u User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (created_at) VALUES (NOW())
		RETURNING id, username, username_normalized, created_at, deleted_at, onboarding_completed_at;
	`).Scan(&u.ID, &u.Username, &u.UsernameNormalized, &u.CreatedAt, &u.DeletedAt, &u.OnboardingCompletedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// 2. Create initial profile
	initialDisplayName := strings.Split(emailNormalized, "@")[0]
	var p Profile
	err = tx.QueryRow(ctx, `
		INSERT INTO profiles (user_id, display_name, bio, availability, revision, updated_at)
		VALUES ($1, $2, '', 'online', 1, NOW())
		RETURNING user_id, display_name, bio, avatar_media_id, banner_media_id, availability, custom_status, revision, updated_at;
	`, u.ID, initialDisplayName).Scan(
		&p.UserID, &p.DisplayName, &p.Bio, &p.AvatarMediaID, &p.BannerMediaID,
		&p.Availability, &p.CustomStatus, &p.Revision, &p.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert profile: %w", err)
	}

	// 3. Link identity
	_, err = tx.Exec(ctx, `
		INSERT INTO identities (user_id, provider, provider_subject, verified_email_normalized, verified_at)
		VALUES ($1, $2, $3, $4, NOW());
	`, u.ID, provider, subject, emailNormalized)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to link identity: %w", err)
	}

	// 4. Initialize preferences
	_, err = tx.Exec(ctx, `
		INSERT INTO preferences (user_id, settings_version, updated_at)
		VALUES ($1, 1, NOW());
	`, u.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize preferences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit user creation: %w", err)
	}

	return &u, &p, nil
}

func (s *PGStore) GetByID(ctx context.Context, id string) (*User, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, username_normalized, created_at, deleted_at, onboarding_completed_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1;
	`, id).Scan(&u.ID, &u.Username, &u.UsernameNormalized, &u.CreatedAt, &u.DeletedAt, &u.OnboardingCompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &u, nil
}

func (s *PGStore) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, display_name, bio, avatar_media_id, banner_media_id, availability, custom_status, revision, updated_at
		FROM profiles
		WHERE user_id = $1
		LIMIT 1;
	`, userID).Scan(
		&p.UserID, &p.DisplayName, &p.Bio, &p.AvatarMediaID, &p.BannerMediaID,
		&p.Availability, &p.CustomStatus, &p.Revision, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to query profile: %w", err)
	}

	return &p, nil
}

func (s *PGStore) UpdateProfile(ctx context.Context, userID string, p *Profile, expectedRevision int) (*Profile, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	query := `
		UPDATE profiles
		SET display_name = $2,
		    bio = $3,
		    availability = $4,
		    custom_status = $5,
		    revision = revision + 1,
		    updated_at = NOW()
		WHERE user_id = $1 AND ($6 <= 0 OR revision = $6)
		RETURNING user_id, display_name, bio, avatar_media_id, banner_media_id, availability, custom_status, revision, updated_at;
	`

	var updated Profile
	err := s.pool.QueryRow(ctx, query,
		userID, p.DisplayName, p.Bio, p.Availability, p.CustomStatus, expectedRevision,
	).Scan(
		&updated.UserID, &updated.DisplayName, &updated.Bio, &updated.AvatarMediaID, &updated.BannerMediaID,
		&updated.Availability, &updated.CustomStatus, &updated.Revision, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if profile exists to distinguish not found from revision mismatch
			var currentRev int
			if checkErr := s.pool.QueryRow(ctx, `SELECT revision FROM profiles WHERE user_id = $1`, userID).Scan(&currentRev); checkErr == nil {
				return nil, ErrRevisionMismatch
			}
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return &updated, nil
}

func (s *PGStore) SetUsername(ctx context.Context, userID, username, usernameNormalized string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE users
		SET username = $2, username_normalized = $3
		WHERE id = $1 AND deleted_at IS NULL;
	`

	_, err := s.pool.Exec(ctx, query, userID, username, usernameNormalized)
	if err != nil {
		if strings.Contains(err.Error(), "idx_users_active_username_normalized") || strings.Contains(err.Error(), "unique") {
			return ErrUsernameConflict
		}
		return fmt.Errorf("failed to update username: %w", err)
	}

	return nil
}

func (s *PGStore) SetAvatar(ctx context.Context, userID string, mediaID *string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE profiles
		SET avatar_media_id = $2, revision = revision + 1, updated_at = NOW()
		WHERE user_id = $1;
	`

	_, err := s.pool.Exec(ctx, query, userID, mediaID)
	return err
}

func (s *PGStore) SetBanner(ctx context.Context, userID string, mediaID *string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE profiles
		SET banner_media_id = $2, revision = revision + 1, updated_at = NOW()
		WHERE user_id = $1;
	`

	_, err := s.pool.Exec(ctx, query, userID, mediaID)
	return err
}

func (s *PGStore) CompleteOnboarding(ctx context.Context, userID, displayName string, username *string) (*User, *Profile, error) {
	if s.pool == nil {
		return nil, nil, errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	// Update user onboarding timestamp and optional username
	var u User
	if username != nil && *username != "" {
		normalized := strings.ToLower(strings.TrimSpace(*username))
		err = tx.QueryRow(ctx, `
			UPDATE users
			SET username = $2, username_normalized = $3, onboarding_completed_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id, username, username_normalized, created_at, deleted_at, onboarding_completed_at;
		`, userID, strings.TrimSpace(*username), normalized).Scan(
			&u.ID, &u.Username, &u.UsernameNormalized, &u.CreatedAt, &u.DeletedAt, &u.OnboardingCompletedAt,
		)
	} else {
		err = tx.QueryRow(ctx, `
			UPDATE users
			SET onboarding_completed_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id, username, username_normalized, created_at, deleted_at, onboarding_completed_at;
		`, userID).Scan(
			&u.ID, &u.Username, &u.UsernameNormalized, &u.CreatedAt, &u.DeletedAt, &u.OnboardingCompletedAt,
		)
	}
	if err != nil {
		if strings.Contains(err.Error(), "idx_users_active_username_normalized") || strings.Contains(err.Error(), "unique") {
			return nil, nil, ErrUsernameConflict
		}
		return nil, nil, fmt.Errorf("failed to complete user onboarding: %w", err)
	}

	// Update display name in profile
	var p Profile
	err = tx.QueryRow(ctx, `
		UPDATE profiles
		SET display_name = $2, revision = revision + 1, updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, display_name, bio, avatar_media_id, banner_media_id, availability, custom_status, revision, updated_at;
	`, userID, displayName).Scan(
		&p.UserID, &p.DisplayName, &p.Bio, &p.AvatarMediaID, &p.BannerMediaID,
		&p.Availability, &p.CustomStatus, &p.Revision, &p.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update profile display name: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return &u, &p, nil
}

func (s *PGStore) FindIdentity(ctx context.Context, provider, subject string) (*Identity, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var ident Identity
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_subject, verified_email_normalized, verified_at
		FROM identities
		WHERE provider = $1 AND provider_subject = $2
		LIMIT 1;
	`, provider, subject).Scan(
		&ident.ID, &ident.UserID, &ident.Provider, &ident.ProviderSubject,
		&ident.VerifiedEmailNormalized, &ident.VerifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, fmt.Errorf("failed to query identity: %w", err)
	}

	return &ident, nil
}

func (s *PGStore) FindUserByEmail(ctx context.Context, emailNormalized string) (*User, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.username_normalized, u.created_at, u.deleted_at, u.onboarding_completed_at
		FROM users u
		JOIN identities i ON i.user_id = u.id
		WHERE i.verified_email_normalized = $1 AND u.deleted_at IS NULL
		ORDER BY i.verified_at DESC
		LIMIT 1;
	`, emailNormalized).Scan(
		&u.ID, &u.Username, &u.UsernameNormalized, &u.CreatedAt, &u.DeletedAt, &u.OnboardingCompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	return &u, nil
}

func (s *PGStore) LinkIdentity(ctx context.Context, userID, provider, subject, emailNormalized string) (*Identity, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	var ident Identity
	err := s.pool.QueryRow(ctx, `
		INSERT INTO identities (user_id, provider, provider_subject, verified_email_normalized, verified_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (provider, provider_subject) DO UPDATE
		SET verified_email_normalized = EXCLUDED.verified_email_normalized,
		    verified_at = NOW()
		RETURNING id, user_id, provider, provider_subject, verified_email_normalized, verified_at;
	`, userID, provider, subject, emailNormalized).Scan(
		&ident.ID, &ident.UserID, &ident.Provider, &ident.ProviderSubject,
		&ident.VerifiedEmailNormalized, &ident.VerifiedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to link identity: %w", err)
	}

	return &ident, nil
}

func (s *PGStore) SearchUsers(ctx context.Context, query string, limit int) ([]*UserSearchResult, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	pattern := "%" + strings.TrimSpace(query) + "%"

	sqlQuery := `
		SELECT u.id::text,
		       COALESCE(NULLIF(p.display_name, ''), u.username, 'Pengguna') AS display_name,
		       COALESCE(u.username, 'user') AS username,
		       p.avatar_media_id,
		       p.banner_media_id,
		       COALESCE(p.bio, '') AS bio,
		       COALESCE(p.availability, 'Tersedia') AS availability,
		       p.custom_status
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE u.deleted_at IS NULL AND (
			$1 = '%%' OR
			p.display_name ILIKE $1 OR
			u.username ILIKE $1
		)
		ORDER BY p.updated_at DESC NULLS LAST
		LIMIT $2;
	`

	rows, err := s.pool.Query(ctx, sqlQuery, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var results []*UserSearchResult
	for rows.Next() {
		var r UserSearchResult
		if err := rows.Scan(
			&r.ID, &r.DisplayName, &r.Username, &r.AvatarMediaID, &r.BannerMediaID, &r.Bio, &r.Availability, &r.CustomStatus,
		); err != nil {
			return nil, err
		}
		results = append(results, &r)
	}

	return results, rows.Err()
}
