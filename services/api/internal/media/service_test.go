package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xynoss/relay/services/api/internal/storage"
)

type memoryMediaStore struct {
	items map[string]*Media
}

func newMemoryMediaStore() *memoryMediaStore {
	return &memoryMediaStore{items: make(map[string]*Media)}
}

func (s *memoryMediaStore) Create(_ context.Context, item *Media) (*Media, error) {
	created := *item
	created.ID = "00000000-0000-4000-8000-000000000001"
	s.items[created.ID] = &created
	return &created, nil
}

func (s *memoryMediaStore) GetByID(_ context.Context, id string) (*Media, error) {
	item, ok := s.items[id]
	if !ok || item.Status == StatusDeleted {
		return nil, ErrMediaNotFound
	}
	copyOfItem := *item
	return &copyOfItem, nil
}

func (s *memoryMediaStore) UpdateStatus(_ context.Context, id string, status MediaStatus, detectedType string, size int64, checksum *string) error {
	item, ok := s.items[id]
	if !ok {
		return ErrMediaNotFound
	}
	item.Status = status
	item.DetectedType = &detectedType
	item.Bytes = size
	item.Checksum = checksum
	return nil
}

func (s *memoryMediaStore) MarkDeleted(_ context.Context, id string) error {
	item, ok := s.items[id]
	if !ok {
		return ErrMediaNotFound
	}
	item.Status = StatusDeleted
	return nil
}

func validPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 20, G: 80, B: 40, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return data.Bytes()
}

func TestMediaUploadLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newMemoryMediaStore()
	provider := storage.NewMockProvider()
	service := NewService(store, provider)

	if _, err := service.CreateUploadIntent(ctx, "", storage.PurposeAvatar, "image/png", 100, ""); err == nil {
		t.Fatal("expected actor validation error")
	}
	if _, err := service.CreateUploadIntent(ctx, "user-1", storage.PurposeAvatar, "image/svg+xml", 100, ""); !errors.Is(err, storage.ErrInvalidMimeType) {
		t.Fatalf("expected MIME validation error, got %v", err)
	}

	data := validPNG(t)
	intent, err := service.CreateUploadIntent(ctx, "user-1", storage.PurposeAvatar, "image/png", int64(len(data)), "checksum")
	if err != nil {
		t.Fatalf("CreateUploadIntent: %v", err)
	}
	if intent.Method != "PUT" || intent.UploadURL == "" || intent.RequiredHeaders["Content-Type"] != "image/png" {
		t.Fatalf("unexpected upload intent: %+v", intent)
	}
	if intent.ObjectKey[:7] != "avatars" {
		t.Fatalf("unexpected avatar key: %s", intent.ObjectKey)
	}
	if _, err := provider.PutObject(ctx, intent.ObjectKey, bytes.NewReader(data), int64(len(data)), "image/png"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	if _, err := service.FinalizeUpload(ctx, "another-user", intent.MediaID, ""); !errors.Is(err, ErrUnauthorizedMediaAccess) {
		t.Fatalf("expected ownership error, got %v", err)
	}
	finalized, err := service.FinalizeUpload(ctx, "user-1", intent.MediaID, "final-checksum")
	if err != nil {
		t.Fatalf("FinalizeUpload: %v", err)
	}
	if finalized.Status != StatusReady || finalized.DetectedType == nil || *finalized.DetectedType != "image/png" || finalized.URL == "" {
		t.Fatalf("unexpected finalized media: %+v", finalized)
	}
	if finalized.Checksum == nil || *finalized.Checksum != "final-checksum" {
		t.Fatalf("final checksum not stored: %+v", finalized.Checksum)
	}

	resolved, err := service.GetMedia(ctx, intent.MediaID)
	if err != nil || resolved.URL == "" {
		t.Fatalf("GetMedia URL resolution failed: media=%+v err=%v", resolved, err)
	}
	if err := service.DeleteMedia(ctx, "another-user", intent.MediaID); !errors.Is(err, ErrUnauthorizedMediaAccess) {
		t.Fatalf("expected delete ownership error, got %v", err)
	}
	if err := service.DeleteMedia(ctx, "user-1", intent.MediaID); err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if _, _, err := provider.GetObject(ctx, intent.ObjectKey); err == nil {
		t.Fatal("expected object cleanup after deletion")
	}
	if _, err := service.GetMedia(ctx, intent.MediaID); !errors.Is(err, ErrMediaNotFound) {
		t.Fatalf("expected deleted media to be hidden, got %v", err)
	}
}

func TestFinalizeRejectsSpoofedImage(t *testing.T) {
	ctx := context.Background()
	store := newMemoryMediaStore()
	provider := storage.NewMockProvider()
	service := NewService(store, provider)
	intent, err := service.CreateUploadIntent(ctx, "user-1", storage.PurposeBanner, "image/png", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.PutObject(ctx, intent.ObjectKey, bytes.NewReader([]byte("not an image")), 12, "image/png"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FinalizeUpload(ctx, "user-1", intent.MediaID, ""); !errors.Is(err, storage.ErrInvalidMimeType) {
		t.Fatalf("expected content validation failure, got %v", err)
	}
	if store.items[intent.MediaID].Status != StatusRejected {
		t.Fatalf("expected rejected state, got %s", store.items[intent.MediaID].Status)
	}
}
