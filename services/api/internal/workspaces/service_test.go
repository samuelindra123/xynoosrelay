package workspaces

import (
	"strings"
	"testing"
)

func TestGenerateOpaqueCodeUsesExpectedAlphabetAndEntropyLength(t *testing.T) {
	first, err := GenerateOpaqueCode()
	if err != nil {
		t.Fatalf("GenerateOpaqueCode() error = %v", err)
	}
	if len(first) != 12 {
		t.Fatalf("code length = %d, want 12", len(first))
	}
	if normalizeInviteCode(first) != first {
		t.Fatalf("generated code %q did not validate", first)
	}

	for _, r := range first {
		if !strings.ContainsRune(inviteCharset, r) {
			t.Fatalf("code contains character outside invite alphabet: %q", r)
		}
	}
}

func TestNormalizeInviteCodeRejectsMalformedInput(t *testing.T) {
	tests := []string{
		"", "short", "abcdefghijkl", "0123456789AB", "valid-code!!", "abcdefghijk/",
	}
	for _, input := range tests {
		if got := normalizeInviteCode(input); got != "" {
			t.Errorf("normalizeInviteCode(%q) = %q, want empty", input, got)
		}
	}
}

func TestResolveBaseURLOnlyUsesConfiguredFrontendURL(t *testing.T) {
	service := NewService(nil, "https://app.xynoss.example/")
	if got, want := service.ResolveBaseURL(), "https://app.xynoss.example"; got != want {
		t.Fatalf("ResolveBaseURL() = %q, want %q", got, want)
	}
}
