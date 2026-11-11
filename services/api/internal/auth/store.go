package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrChallengeNotFound        = errors.New("authentication challenge not found")
	ErrChallengeExpired         = errors.New("authentication challenge has expired")
	ErrChallengeAlreadyConsumed = errors.New("authentication challenge has already been consumed")
	ErrChallengeInvalidated     = errors.New("authentication challenge has been superseded or invalidated")
	ErrTooManyAttempts          = errors.New("maximum verification attempts exceeded for this challenge")
	ErrInvalidCode              = errors.New("invalid verification code")
)

// Store defines persistence operations for OTP challenges.
type Store interface {
	CreateChallenge(ctx context.Context, emailNormalized, codeHMAC, keyVersion string, expiresAt time.Time) (string, error)
	GetLatestChallengeCooldown(ctx context.Context, emailNormalized string) (time.Duration, bool, error)
	VerifyAndConsume(ctx context.Context, challengeID, candidateCode, hmacKey string) (string, error)
}

// PGStore implements Store backed by PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL auth store.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) CreateChallenge(ctx context.Context, emailNormalized, codeHMAC, keyVersion string, expiresAt time.Time) (string, error) {
	if s.pool == nil {
		return "", errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Invalidate any existing unconsumed challenges for this email
	_, err = tx.Exec(ctx, `
		UPDATE otp_challenges
		SET invalidated_at = NOW()
		WHERE email_normalized = $1 AND consumed_at IS NULL AND invalidated_at IS NULL;
	`, emailNormalized)
	if err != nil {
		return "", fmt.Errorf("failed to invalidate existing challenges: %w", err)
	}

	// 2. Insert new challenge
	var challengeID string
	err = tx.QueryRow(ctx, `
		INSERT INTO otp_challenges (
			email_normalized, code_hmac, key_version, expires_at, created_at
		) VALUES (
			$1, $2, $3, $4, NOW()
		) RETURNING id;
	`, emailNormalized, codeHMAC, keyVersion, expiresAt).Scan(&challengeID)
	if err != nil {
		return "", fmt.Errorf("failed to insert challenge: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit challenge transaction: %w", err)
	}

	return challengeID, nil
}

func (s *PGStore) GetLatestChallengeCooldown(ctx context.Context, emailNormalized string) (time.Duration, bool, error) {
	if s.pool == nil {
		return 0, false, errors.New("database pool is disconnected")
	}

	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT created_at
		FROM otp_challenges
		WHERE email_normalized = $1
		ORDER BY created_at DESC
		LIMIT 1;
	`, emailNormalized).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}

	elapsed := time.Since(createdAt)
	cooldown := 60 * time.Second
	if elapsed < cooldown {
		return cooldown - elapsed, true, nil
	}

	return 0, false, nil
}

func (s *PGStore) VerifyAndConsume(ctx context.Context, challengeID, candidateCode, hmacKey string) (string, error) {
	if s.pool == nil {
		return "", errors.New("database pool is disconnected")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin verification transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Row lock with SELECT ... FOR UPDATE to avoid race conditions on simultaneous verification
	query := `
		SELECT id, email_normalized, code_hmac, expires_at, attempts, max_attempts, consumed_at, invalidated_at
		FROM otp_challenges
		WHERE id = $1
		FOR UPDATE;
	`

	var (
		id              string
		emailNormalized string
		codeHMAC        string
		expiresAt       time.Time
		attempts        int
		maxAttempts     int
		consumedAt      *time.Time
		invalidatedAt   *time.Time
	)

	err = tx.QueryRow(ctx, query, challengeID).Scan(
		&id, &emailNormalized, &codeHMAC, &expiresAt,
		&attempts, &maxAttempts, &consumedAt, &invalidatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrChallengeNotFound
		}
		return "", fmt.Errorf("failed to lock challenge: %w", err)
	}

	// 1. Check if invalidated
	if invalidatedAt != nil {
		return "", ErrChallengeInvalidated
	}

	// 2. Check if already consumed
	if consumedAt != nil {
		return "", ErrChallengeAlreadyConsumed
	}

	// 3. Check expiration
	if time.Now().After(expiresAt) {
		return "", ErrChallengeExpired
	}

	// 4. Increment attempts unconditionally
	attempts++
	_, err = tx.Exec(ctx, `UPDATE otp_challenges SET attempts = $2 WHERE id = $1;`, challengeID, attempts)
	if err != nil {
		return "", fmt.Errorf("failed to increment attempts: %w", err)
	}

	// 5. Check if attempts exceeded
	if attempts > maxAttempts {
		_ = tx.Commit(ctx) // Save attempt count before returning
		return "", ErrTooManyAttempts
	}

	// 6. Constant-time HMAC comparison
	if !VerifyOTPHMAC(candidateCode, emailNormalized, hmacKey, codeHMAC) {
		_ = tx.Commit(ctx) // Save attempt count before returning
		return "", ErrInvalidCode
	}

	// 7. Mark consumed
	_, err = tx.Exec(ctx, `UPDATE otp_challenges SET consumed_at = NOW() WHERE id = $1;`, challengeID)
	if err != nil {
		return "", fmt.Errorf("failed to mark challenge consumed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit consumption: %w", err)
	}

	return emailNormalized, nil
}
