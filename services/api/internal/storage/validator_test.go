package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestValidateMediaMetadata(t *testing.T) {
	tests := []struct {
		name         string
		purpose      Purpose
		mime         string
		size         int64
		expectErr    bool
		errSubstring string
	}{
		{"valid avatar JPEG", PurposeAvatar, "image/jpeg", 1024 * 500, false, ""},
		{"valid avatar PNG", PurposeAvatar, "image/png", 1024 * 1000, false, ""},
		{"valid avatar WebP", PurposeAvatar, "image/webp", 1024 * 2000, false, ""},
		{"invalid MIME SVG", PurposeAvatar, "image/svg+xml", 1024, true, "unsupported media type"},
		{"invalid MIME PDF", PurposeAvatar, "application/pdf", 1024, true, "unsupported media type"},
		{"avatar size exceeded (6MB)", PurposeAvatar, "image/jpeg", 6 * 1024 * 1024, true, "exceeds maximum allowed size"},
		{"banner size valid (8MB)", PurposeBanner, "image/jpeg", 8 * 1024 * 1024, false, ""},
		{"banner size exceeded (12MB)", PurposeBanner, "image/jpeg", 12 * 1024 * 1024, true, "exceeds maximum allowed size"},
		{"zero size", PurposeAvatar, "image/jpeg", 0, true, "exceeds maximum allowed size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMediaMetadata(tt.purpose, tt.mime, tt.size)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errSubstring, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInspectAndValidateContent_ValidPNGAndJPEG(t *testing.T) {
	pngData := createTestPNG(200, 200)
	res, err := InspectAndValidateContent(PurposeAvatar, "image/png", pngData)
	if err != nil {
		t.Fatalf("unexpected error for valid PNG: %v", err)
	}
	if res.Width != 200 || res.Height != 200 {
		t.Errorf("expected 200x200, got %dx%d", res.Width, res.Height)
	}
	if res.DetectedMIME != "image/png" {
		t.Errorf("expected image/png, got %s", res.DetectedMIME)
	}

	jpegData := createTestJPEG(300, 150)
	resJpg, err := InspectAndValidateContent(PurposeBanner, "image/jpeg", jpegData)
	if err != nil {
		t.Fatalf("unexpected error for valid JPEG: %v", err)
	}
	if resJpg.Width != 300 || resJpg.Height != 150 {
		t.Errorf("expected 300x150, got %dx%d", resJpg.Width, resJpg.Height)
	}
}

func TestInspectAndValidateContent_SpoofedContent(t *testing.T) {
	// Send PNG bytes with declared JPEG
	pngData := createTestPNG(100, 100)
	_, err := InspectAndValidateContent(PurposeAvatar, "image/jpeg", pngData)
	if err == nil {
		t.Fatalf("expected spoofing error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected mismatch error, got: %v", err)
	}
}

func TestInspectAndValidateContent_DangerousSVG(t *testing.T) {
	svgData := []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert('xss')</script></svg>")
	_, err := InspectAndValidateContent(PurposeAvatar, "image/jpeg", svgData)
	if err == nil {
		t.Fatalf("expected dangerous content error, got nil")
	}
}

func TestInspectAndValidateContent_DimensionExceeded(t *testing.T) {
	largeImg := createTestPNG(5000, 100)
	_, err := InspectAndValidateContent(PurposeBanner, "image/png", largeImg)
	if err == nil {
		t.Fatalf("expected dimension exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected dimension error, got: %v", err)
	}
}
