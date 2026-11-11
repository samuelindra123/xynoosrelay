package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// MockProvider provides an in-memory StorageProvider for automated tests.
type MockProvider struct {
	mu      sync.RWMutex
	objects map[string][]byte
	types   map[string]string
}

// NewMockProvider creates an in-memory mock storage provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		objects: make(map[string][]byte),
		types:   make(map[string]string),
	}
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) GenerateUploadURL(ctx context.Context, key string, contentType string, sizeBytes int64, ttl time.Duration) (string, map[string]string, error) {
	return fmt.Sprintf("https://mock-storage.local/upload/%s", key), map[string]string{
		"Content-Type": contentType,
	}, nil
}

func (m *MockProvider) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	m.objects[key] = data
	m.types[key] = contentType
	return m.PublicURL(key), nil
}

func (m *MockProvider) GetObject(ctx context.Context, key string) (io.ReadCloser, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.objects[key]
	if !ok {
		return nil, "", errors.New("object not found")
	}

	contentType := m.types[key]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return io.NopCloser(bytes.NewReader(data)), contentType, nil
}

func (m *MockProvider) DeleteObject(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, key)
	delete(m.types, key)
	return nil
}

func (m *MockProvider) PublicURL(key string) string {
	return fmt.Sprintf("https://cdn.mock-storage.local/%s", key)
}
