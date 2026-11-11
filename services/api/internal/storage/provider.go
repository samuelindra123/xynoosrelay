package storage

import (
	"context"
	"io"
	"time"
)

// Purpose defines the intended media usage.
type Purpose string

const (
	PurposeAvatar Purpose = "avatar"
	PurposeBanner Purpose = "banner"
)

// UploadIntent holds the information returned to the client to perform a direct upload.
type UploadIntent struct {
	MediaID         string            `json:"media_id"`
	UploadURL       string            `json:"upload_url"`
	Method          string            `json:"method"`
	RequiredHeaders map[string]string `json:"required_headers"`
	ObjectKey       string            `json:"object_key"`
	ExpiresAt       time.Time         `json:"expires_at"`
}

// StorageProvider defines the pluggable object storage contract.
type StorageProvider interface {
	// GenerateUploadURL creates a presigned or direct URL for uploading a file.
	GenerateUploadURL(ctx context.Context, key string, contentType string, sizeBytes int64, ttl time.Duration) (uploadURL string, headers map[string]string, err error)

	// PutObject stores an object directly in storage.
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (publicURL string, err error)

	// GetObject retrieves an object from storage.
	GetObject(ctx context.Context, key string) (io.ReadCloser, string, error)

	// DeleteObject removes an object from storage.
	DeleteObject(ctx context.Context, key string) error

	// PublicURL returns the public accessibility URL for a given key.
	PublicURL(key string) string

	// Name returns the provider identifier.
	Name() string
}
