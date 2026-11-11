package storage

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	_ "golang.org/x/image/webp"
)

var (
	ErrInvalidMimeType   = errors.New("unsupported media type; only JPEG, PNG, and WebP are allowed")
	ErrSpoofedMimeType   = errors.New("file content signature does not match declared MIME type")
	ErrFileTooLarge      = errors.New("file exceeds maximum allowed size")
	ErrCorruptImage      = errors.New("image decoding failed; file may be corrupt or malformed")
	ErrDimensionExceeded = errors.New("image dimensions exceed maximum allowed limits (4096 x 4096)")
	ErrDangerousContent  = errors.New("potentially dangerous content detected (e.g. SVG or script)")
)

const (
	MaxAvatarSize = 5 * 1024 * 1024  // 5 MB
	MaxBannerSize = 10 * 1024 * 1024 // 10 MB

	MaxImageWidth  = 4096
	MaxImageHeight = 4096
)

// AllowedMIMETypes maps permitted MIME types to canonical file extensions.
var AllowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ValidateMediaMetadata performs pre-flight checks on declared MIME type and size limit.
func ValidateMediaMetadata(purpose Purpose, declaredMime string, sizeBytes int64) error {
	ext, ok := AllowedMIMETypes[strings.ToLower(declaredMime)]
	if !ok || ext == "" {
		return ErrInvalidMimeType
	}

	maxSize := int64(MaxAvatarSize)
	if purpose == PurposeBanner {
		maxSize = MaxBannerSize
	}

	if sizeBytes <= 0 || sizeBytes > maxSize {
		return fmt.Errorf("%w: max allowed for %s is %d bytes, got %d", ErrFileTooLarge, purpose, maxSize, sizeBytes)
	}

	return nil
}

// MediaInspectionResult contains verified media details after inspecting content.
type MediaInspectionResult struct {
	DetectedMIME string
	Bytes        int64
	Width        int
	Height       int
}

// InspectAndValidateContent validates file bytes against MIME allowlist, magic bytes, dimensions, and bombs.
func InspectAndValidateContent(purpose Purpose, declaredMime string, data []byte) (*MediaInspectionResult, error) {
	if len(data) == 0 {
		return nil, errors.New("file content cannot be empty")
	}

	maxSize := int64(MaxAvatarSize)
	if purpose == PurposeBanner {
		maxSize = MaxBannerSize
	}
	if int64(len(data)) > maxSize {
		return nil, ErrFileTooLarge
	}

	// 1. Check for dangerous signatures (e.g. SVG, XML, HTML scripts)
	lowerPrefix := strings.ToLower(string(data[:min(len(data), 512)]))
	if strings.Contains(lowerPrefix, "<svg") ||
		strings.Contains(lowerPrefix, "<?xml") ||
		strings.Contains(lowerPrefix, "<html") ||
		strings.Contains(lowerPrefix, "<script") {
		return nil, ErrDangerousContent
	}

	// 2. Sniff MIME type from header bytes
	detectedMime := http.DetectContentType(data)

	// WebP detection: standard DetectContentType might return application/octet-stream for WebP
	isWebP := len(data) >= 12 &&
		string(data[0:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"

	isPNG := len(data) >= 8 &&
		bytes.Equal(data[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})

	isJPEG := len(data) >= 3 &&
		bytes.Equal(data[0:3], []byte{0xFF, 0xD8, 0xFF})

	var canonicalDetected string
	if isJPEG {
		canonicalDetected = "image/jpeg"
	} else if isPNG {
		canonicalDetected = "image/png"
	} else if isWebP {
		canonicalDetected = "image/webp"
	} else {
		// Fallback to detectedMime if allowed
		if _, ok := AllowedMIMETypes[detectedMime]; ok {
			canonicalDetected = detectedMime
		} else {
			return nil, ErrInvalidMimeType
		}
	}

	// Ensure declared matches detected
	if strings.ToLower(declaredMime) != canonicalDetected {
		return nil, fmt.Errorf("%w: declared '%s' but detected '%s'", ErrSpoofedMimeType, declaredMime, canonicalDetected)
	}

	// 3. Decode image config to verify dimensions and prevent decompression bombs
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptImage, err)
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("%w: invalid image dimensions", ErrCorruptImage)
	}

	if cfg.Width > MaxImageWidth || cfg.Height > MaxImageHeight {
		return nil, fmt.Errorf("%w: %dx%d exceeds %dx%d", ErrDimensionExceeded, cfg.Width, cfg.Height, MaxImageWidth, MaxImageHeight)
	}

	return &MediaInspectionResult{
		DetectedMIME: canonicalDetected,
		Bytes:        int64(len(data)),
		Width:        cfg.Width,
		Height:       cfg.Height,
	}, nil
}

// ReadAndValidateStream streams up to maxBytes, buffers for sniffing, and verifies content integrity.
func ReadAndValidateStream(purpose Purpose, declaredMime string, r io.Reader) ([]byte, *MediaInspectionResult, error) {
	maxSize := int64(MaxAvatarSize)
	if purpose == PurposeBanner {
		maxSize = MaxBannerSize
	}

	// Read up to maxSize + 1 to detect overflows
	lr := io.LimitReader(r, maxSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read media stream: %w", err)
	}

	if int64(len(data)) > maxSize {
		return nil, nil, ErrFileTooLarge
	}

	result, err := InspectAndValidateContent(purpose, declaredMime, data)
	if err != nil {
		return nil, nil, err
	}

	return data, result, nil
}
