package media

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMediaNotFound = errors.New("media record not found")
)

// Store defines the storage operations for media entities.
type Store interface {
	Create(ctx context.Context, m *Media) (*Media, error)
	GetByID(ctx context.Context, id string) (*Media, error)
	UpdateStatus(ctx context.Context, id string, status MediaStatus, detectedType string, bytes int64, checksum *string) error
	MarkDeleted(ctx context.Context, id string) error
}

// PGStore implements Store backed by PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL media store.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// IsProfileMedia reports whether an image is currently attached to a public user profile.
func (s *PGStore) IsProfileMedia(ctx context.Context, mediaID string) (bool, error) {
	if s.pool == nil {
		return false, errors.New("database pool is disconnected")
	}
	var attached bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM profiles WHERE avatar_media_id = $1 OR banner_media_id = $1
	)`, mediaID).Scan(&attached)
	if err != nil {
		return false, fmt.Errorf("failed to check profile media visibility: %w", err)
	}
	return attached, nil
}

func (s *PGStore) Create(ctx context.Context, m *Media) (*Media, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	query := `
		INSERT INTO media (
			owner_id, object_key, status, declared_type, bytes, checksum
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING id, owner_id, object_key, status, declared_type, detected_type, bytes, checksum, created_at;
	`

	var created Media
	err := s.pool.QueryRow(ctx, query,
		m.OwnerID, m.ObjectKey, string(m.Status), m.DeclaredType, m.Bytes, m.Checksum,
	).Scan(
		&created.ID, &created.OwnerID, &created.ObjectKey, &created.Status,
		&created.DeclaredType, &created.DetectedType, &created.Bytes,
		&created.Checksum, &created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert media: %w", err)
	}

	return &created, nil
}

func (s *PGStore) GetByID(ctx context.Context, id string) (*Media, error) {
	if s.pool == nil {
		return nil, errors.New("database pool is disconnected")
	}

	query := `
		SELECT id, owner_id, object_key, status, declared_type, detected_type, bytes, checksum, created_at
		FROM media
		WHERE id = $1 AND status != 'deleted'
		LIMIT 1;
	`

	var m Media
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.OwnerID, &m.ObjectKey, &m.Status,
		&m.DeclaredType, &m.DetectedType, &m.Bytes,
		&m.Checksum, &m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMediaNotFound
		}
		return nil, fmt.Errorf("failed to query media: %w", err)
	}

	return &m, nil
}

func (s *PGStore) UpdateStatus(ctx context.Context, id string, status MediaStatus, detectedType string, bytes int64, checksum *string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `
		UPDATE media
		SET status = $2, detected_type = $3, bytes = $4, checksum = COALESCE($5, checksum)
		WHERE id = $1;
	`

	res, err := s.pool.Exec(ctx, query, id, string(status), detectedType, bytes, checksum)
	if err != nil {
		return fmt.Errorf("failed to update media status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrMediaNotFound
	}

	return nil
}

func (s *PGStore) MarkDeleted(ctx context.Context, id string) error {
	if s.pool == nil {
		return errors.New("database pool is disconnected")
	}

	query := `UPDATE media SET status = 'deleted' WHERE id = $1;`
	_, err := s.pool.Exec(ctx, query, id)
	return err
}
