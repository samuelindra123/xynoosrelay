package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LocalProvider implements StorageProvider by persisting media to the local filesystem.
type LocalProvider struct {
	baseDir    string
	publicURL  string
	signingKey []byte
}

// NewLocalProvider creates a new local disk storage provider.
func NewLocalProvider(baseDir, publicURL, signingKey string) (*LocalProvider, error) {
	if baseDir == "" {
		baseDir = "uploads"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to initialize local storage directory: %w", err)
	}

	if signingKey == "" {
		signingKey = "xynoss-local-dev-signing-key-minimum-32-chars-long"
	}

	return &LocalProvider{
		baseDir:    filepath.Clean(baseDir),
		publicURL:  strings.TrimRight(publicURL, "/"),
		signingKey: []byte(signingKey),
	}, nil
}

func (l *LocalProvider) Name() string {
	return "local"
}

// GenerateUploadURL creates a signed URL allowing direct upload to this server in development.
func (l *LocalProvider) GenerateUploadURL(ctx context.Context, key string, contentType string, sizeBytes int64, ttl time.Duration) (string, map[string]string, error) {
	expiresAt := time.Now().Add(ttl).Unix()

	// Compute signature: HMAC-SHA256(key:contentType:sizeBytes:expiresAt)
	payload := fmt.Sprintf("%s:%s:%d:%d", key, contentType, sizeBytes, expiresAt)
	mac := hmac.New(sha256.New, l.signingKey)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	uploadURL := fmt.Sprintf("%s/upload?key=%s&content_type=%s&size=%d&expires=%d&sig=%s",
		l.publicURL,
		url.QueryEscape(key),
		url.QueryEscape(contentType),
		sizeBytes,
		expiresAt,
		sig,
	)

	headers := map[string]string{
		"Content-Type": contentType,
	}

	return uploadURL, headers, nil
}

// VerifyUploadSignature verifies that the given parameters match the HMAC signature and have not expired.
func (l *LocalProvider) VerifyUploadSignature(key, contentType string, sizeBytes, expiresAt int64, sig string) bool {
	if time.Now().Unix() > expiresAt {
		return false
	}

	payload := fmt.Sprintf("%s:%s:%d:%d", key, contentType, sizeBytes, expiresAt)
	mac := hmac.New(sha256.New, l.signingKey)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

func (l *LocalProvider) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	safePath, err := l.resolveSafePath(key)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	outFile, err := os.OpenFile(safePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open destination file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return "", fmt.Errorf("failed to write file content: %w", err)
	}

	return l.PublicURL(key), nil
}

func (l *LocalProvider) GetObject(ctx context.Context, key string) (io.ReadCloser, string, error) {
	safePath, err := l.resolveSafePath(key)
	if err != nil {
		return nil, "", err
	}

	file, err := os.Open(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", errors.New("object not found")
		}
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}

	// Sniff content type from first 512 bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	_, _ = file.Seek(0, io.SeekStart)

	detected := "application/octet-stream"
	if n > 0 {
		if strings.HasSuffix(key, ".jpg") || strings.HasSuffix(key, ".jpeg") {
			detected = "image/jpeg"
		} else if strings.HasSuffix(key, ".png") {
			detected = "image/png"
		} else if strings.HasSuffix(key, ".webp") {
			detected = "image/webp"
		}
	}

	return file, detected, nil
}

func (l *LocalProvider) DeleteObject(ctx context.Context, key string) error {
	safePath, err := l.resolveSafePath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(safePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (l *LocalProvider) PublicURL(key string) string {
	return fmt.Sprintf("%s/%s", l.publicURL, strings.TrimPrefix(key, "/"))
}

// resolveSafePath ensures that the requested key cannot traverse outside baseDir.
func (l *LocalProvider) resolveSafePath(key string) (string, error) {
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return "", errors.New("path traversal detected: absolute paths are forbidden")
	}

	cleanedKey := filepath.Clean(filepath.FromSlash(key))
	if strings.Contains(cleanedKey, "..") || filepath.IsAbs(cleanedKey) {
		return "", errors.New("path traversal detected: key contains invalid path segments")
	}

	fullPath := filepath.Join(l.baseDir, cleanedKey)
	rel, err := filepath.Rel(l.baseDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("path traversal detected: escaped storage root")
	}

	return fullPath, nil
}

// ParseUploadQueryParams parses query values from an incoming upload request.
func ParseUploadQueryParams(q url.Values) (key, contentType, sig string, size, expires int64, err error) {
	key = q.Get("key")
	contentType = q.Get("content_type")
	sig = q.Get("sig")

	if key == "" || contentType == "" || sig == "" {
		return "", "", "", 0, 0, errors.New("missing required upload query parameters")
	}

	size, err = strconv.ParseInt(q.Get("size"), 10, 64)
	if err != nil {
		return "", "", "", 0, 0, errors.New("invalid size parameter")
	}

	expires, err = strconv.ParseInt(q.Get("expires"), 10, 64)
	if err != nil {
		return "", "", "", 0, 0, errors.New("invalid expires parameter")
	}

	return key, contentType, sig, size, expires, nil
}
