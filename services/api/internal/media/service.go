package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xynoss/relay/services/api/internal/storage"
)

var (
	ErrUnauthorizedMediaAccess = errors.New("unauthorized to access or modify this media asset")
	ErrInvalidMediaState       = errors.New("media asset is not in a valid state for this operation")
)

// Service encapsulates media upload intents, content validation, and asset lifecycle.
type Service struct {
	store    Store
	provider storage.StorageProvider
}

// NewService constructs a new media service instance.
func NewService(store Store, provider storage.StorageProvider) *Service {
	return &Service{
		store:    store,
		provider: provider,
	}
}

// CreateUploadIntent validates request parameters and returns signed direct upload instructions.
func (s *Service) CreateUploadIntent(ctx context.Context, actorID string, purpose storage.Purpose, declaredMime string, sizeBytes int64, checksum string) (*storage.UploadIntent, error) {
	if actorID == "" {
		return nil, errors.New("actor ID is required")
	}

	if err := storage.ValidateMediaMetadata(purpose, declaredMime, sizeBytes); err != nil {
		return nil, err
	}

	ext := storage.AllowedMIMETypes[strings.ToLower(declaredMime)]
	objectKey := fmt.Sprintf("%ss/%s%s", purpose, uuid.NewString(), ext)

	m := &Media{
		OwnerID:      actorID,
		ObjectKey:    objectKey,
		Status:       StatusPending,
		DeclaredType: declaredMime,
		Bytes:        sizeBytes,
	}
	if checksum != "" {
		m.Checksum = &checksum
	}

	created, err := s.store.Create(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("failed to register upload intent in database: %w", err)
	}

	uploadTTL := 15 * time.Minute
	uploadURL, headers, err := s.provider.GenerateUploadURL(ctx, objectKey, declaredMime, sizeBytes, uploadTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed upload URL: %w", err)
	}

	return &storage.UploadIntent{
		MediaID:         created.ID,
		UploadURL:       uploadURL,
		Method:          "PUT",
		RequiredHeaders: headers,
		ObjectKey:       objectKey,
		ExpiresAt:       time.Now().Add(uploadTTL),
	}, nil
}

// FinalizeUpload verifies uploaded content against magic bytes, dimensions, and ownership, then marks ready.
func (s *Service) FinalizeUpload(ctx context.Context, actorID, mediaID, checksum string) (*Media, error) {
	m, err := s.store.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	// 1. Authorization: user must own the media
	if m.OwnerID != actorID {
		return nil, ErrUnauthorizedMediaAccess
	}

	// 2. Validate current state
	if m.Status == StatusReady {
		m.URL = s.provider.PublicURL(m.ObjectKey)
		return m, nil
	}
	if m.Status != StatusPending {
		return nil, fmt.Errorf("%w: status is %s", ErrInvalidMediaState, m.Status)
	}

	// 3. Retrieve object from storage to inspect bytes
	rc, _, err := s.provider.GetObject(ctx, m.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch uploaded file from storage: %w", err)
	}
	defer rc.Close()

	purpose := storage.PurposeAvatar
	if strings.HasPrefix(m.ObjectKey, "banners/") {
		purpose = storage.PurposeBanner
	}

	_, inspection, err := storage.ReadAndValidateStream(purpose, m.DeclaredType, rc)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, mediaID, StatusRejected, "", 0, nil)
		return nil, fmt.Errorf("media content validation failed: %w", err)
	}

	var cs *string
	if checksum != "" {
		cs = &checksum
	} else if m.Checksum != nil {
		cs = m.Checksum
	}

	if err := s.store.UpdateStatus(ctx, mediaID, StatusReady, inspection.DetectedMIME, inspection.Bytes, cs); err != nil {
		return nil, fmt.Errorf("failed to mark media ready in database: %w", err)
	}

	m.Status = StatusReady
	m.DetectedType = &inspection.DetectedMIME
	m.Bytes = inspection.Bytes
	m.Checksum = cs
	m.URL = s.provider.PublicURL(m.ObjectKey)

	return m, nil
}

// GetMedia retrieves media metadata and resolves public URL.
func (s *Service) GetMedia(ctx context.Context, mediaID string) (*Media, error) {
	m, err := s.store.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	m.URL = s.provider.PublicURL(m.ObjectKey)
	return m, nil
}

// GetProfileContent returns ready image bytes for the public profile-media proxy.
// The HTTP handler separately checks that the ID is currently attached to a profile.
func (s *Service) GetProfileContent(ctx context.Context, mediaID string) (io.ReadCloser, string, error) {
	m, err := s.store.GetByID(ctx, mediaID)
	if err != nil {
		return nil, "", err
	}
	if m.Status != StatusReady {
		return nil, "", ErrInvalidMediaState
	}
	rc, contentType, err := s.provider.GetObject(ctx, m.ObjectKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read profile media: %w", err)
	}
	if m.DetectedType != nil && contentType != *m.DetectedType {
		_ = rc.Close()
		return nil, "", errors.New("stored profile media type does not match its validated type")
	}
	return rc, contentType, nil
}

// DeleteMedia deletes media from database and purges object from storage provider.
func (s *Service) DeleteMedia(ctx context.Context, actorID, mediaID string) error {
	m, err := s.store.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	if m.OwnerID != actorID {
		return ErrUnauthorizedMediaAccess
	}

	if err := s.store.MarkDeleted(ctx, mediaID); err != nil {
		return fmt.Errorf("failed to mark media deleted: %w", err)
	}

	// Async/best-effort delete from storage
	_ = s.provider.DeleteObject(ctx, m.ObjectKey)
	return nil
}

// StorageProvider returns the active underlying storage provider.
func (s *Service) StorageProvider() storage.StorageProvider {
	return s.provider
}
