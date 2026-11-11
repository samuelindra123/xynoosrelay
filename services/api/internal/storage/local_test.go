package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalProvider_PathTraversalPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xynoss-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider, err := NewLocalProvider(tempDir, "http://localhost:8080/v1/media/content", "test-key-32-chars-long-for-hmac!!")
	if err != nil {
		t.Fatalf("failed to initialize local provider: %v", err)
	}

	traversalKeys := []string{
		"../secret.txt",
		"../../etc/passwd",
		"avatars/../../evil.txt",
		"/absolute/path/file.jpg",
		"avatars/../../../windows/system32/cmd.exe",
	}

	ctx := context.Background()
	for _, k := range traversalKeys {
		t.Run(k, func(t *testing.T) {
			_, err := provider.PutObject(ctx, k, strings.NewReader("evil"), 4, "text/plain")
			if err == nil {
				t.Errorf("expected path traversal error for key %s, got nil", k)
			}
		})
	}
}

func TestLocalProvider_PutGetDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xynoss-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider, err := NewLocalProvider(tempDir, "http://localhost:8080/v1/media/content", "test-key-32-chars-long-for-hmac!!")
	if err != nil {
		t.Fatalf("failed to initialize local provider: %v", err)
	}

	ctx := context.Background()
	testKey := "avatars/user-123.png"
	content := []byte("fake-png-content-bytes")

	url, err := provider.PutObject(ctx, testKey, bytes.NewReader(content), int64(len(content)), "image/png")
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}
	if !strings.HasSuffix(url, "avatars/user-123.png") {
		t.Errorf("expected url to end with avatars/user-123.png, got %s", url)
	}

	// Verify file is on disk
	expectedFile := filepath.Join(tempDir, "avatars", "user-123.png")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("file not found on disk at: %s", expectedFile)
	}

	// Retrieve object
	rc, ct, err := provider.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("expected image/png, got %s", ct)
	}
	_ = rc.Close() // Close before delete on Windows!

	// Delete object
	if err := provider.DeleteObject(ctx, testKey); err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// Verify deleted
	_, _, err = provider.GetObject(ctx, testKey)
	if err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}

func TestLocalProvider_SignedUploadVerification(t *testing.T) {
	provider, err := NewLocalProvider(t.TempDir(), "http://localhost:8080/v1/media/content", "super-secret-local-signing-key-123")
	if err != nil {
		t.Fatalf("failed to init provider: %v", err)
	}

	ctx := context.Background()
	key := "avatars/test-uuid.webp"
	contentType := "image/webp"
	size := int64(1024 * 50)
	ttl := 5 * time.Minute

	uploadURL, headers, err := provider.GenerateUploadURL(ctx, key, contentType, size, ttl)
	if err != nil {
		t.Fatalf("GenerateUploadURL failed: %v", err)
	}
	if !strings.Contains(uploadURL, "sig=") {
		t.Errorf("uploadURL missing sig parameter: %s", uploadURL)
	}
	if headers["Content-Type"] != contentType {
		t.Errorf("expected Content-Type header %s", contentType)
	}

	// Tampered signature or expired should fail verification
	expiresAt := time.Now().Add(ttl).Unix()
	valid := provider.VerifyUploadSignature(key, contentType, size, expiresAt, "invalid-tampered-sig")
	if valid {
		t.Errorf("expected invalid signature to fail")
	}

	// Expired should fail
	pastExpires := time.Now().Add(-1 * time.Minute).Unix()
	validExpired := provider.VerifyUploadSignature(key, contentType, size, pastExpires, "anysig")
	if validExpired {
		t.Errorf("expected expired signature to fail")
	}
}
