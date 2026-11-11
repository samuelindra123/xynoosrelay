package media

import (
	"time"
)

// MediaStatus defines the lifecycle status of an uploaded media object.
type MediaStatus string

const (
	StatusPending     MediaStatus = "pending"
	StatusQuarantined MediaStatus = "quarantined"
	StatusReady       MediaStatus = "ready"
	StatusRejected    MediaStatus = "rejected"
	StatusDeleted     MediaStatus = "deleted"
)

// Media represents a media asset record stored in PostgreSQL.
type Media struct {
	ID           string      `json:"id"`
	OwnerID      string      `json:"owner_id"`
	ObjectKey    string      `json:"object_key"`
	Status       MediaStatus `json:"status"`
	DeclaredType string      `json:"declared_type"`
	DetectedType *string     `json:"detected_type,omitempty"`
	Bytes        int64       `json:"bytes"`
	Checksum     *string     `json:"checksum,omitempty"`
	URL          string      `json:"url,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// MediaIntentRequest payload for requesting an upload intent.
type MediaIntentRequest struct {
	Purpose     string `json:"purpose"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Checksum    string `json:"checksum,omitempty"`
}

// FinalizeMediaRequest payload for finalizing an upload.
type FinalizeMediaRequest struct {
	Checksum string `json:"checksum,omitempty"`
}
