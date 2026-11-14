package config

import (
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	mockEnv := map[string]string{}
	lookup := func(key string) (string, bool) {
		val, ok := mockEnv[key]
		return val, ok
	}

	cfg, err := LoadFromLookup(lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppEnv != EnvDevelopment {
		t.Errorf("expected AppEnv to be '%s', got '%s'", EnvDevelopment, cfg.AppEnv)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected Port to be '8080', got '%s'", cfg.Port)
	}
	if len(cfg.WebOrigins) != 1 || cfg.WebOrigins[0] != "http://localhost:3000" {
		t.Errorf("unexpected WebOrigins: %v", cfg.WebOrigins)
	}
	if cfg.DBMaxConns != 5 {
		t.Errorf("expected DBMaxConns to be 5, got %d", cfg.DBMaxConns)
	}
	if cfg.RequestTimeout != 15*time.Second {
		t.Errorf("expected RequestTimeout to be 15s, got %v", cfg.RequestTimeout)
	}
}

func TestConfig_UsesNeonCLIObjectStorageEnvironment(t *testing.T) {
	mockEnv := map[string]string{
		"AWS_ACCESS_KEY_ID":     "test-access-key",
		"AWS_SECRET_ACCESS_KEY": "test-storage-secret",
		"AWS_ENDPOINT_URL_S3":   "https://branch.storage.example.neon.tech",
		"AWS_REGION":            "us-east-2",
	}
	lookup := func(key string) (string, bool) {
		value, ok := mockEnv[key]
		return value, ok
	}

	cfg, err := LoadFromLookup(lookup)
	if err != nil {
		t.Fatalf("unexpected config error: %v", err)
	}
	if cfg.StorageProvider != "neon" {
		t.Errorf("expected Neon provider, got %q", cfg.StorageProvider)
	}
	if cfg.StorageBucket != "uploads" {
		t.Errorf("expected uploads bucket, got %q", cfg.StorageBucket)
	}
	if cfg.StorageEndpoint != mockEnv["AWS_ENDPOINT_URL_S3"] || cfg.StorageRegion != mockEnv["AWS_REGION"] {
		t.Error("Neon endpoint or region was not loaded from AWS_* environment variables")
	}
	if cfg.StorageAccessKeyID != mockEnv["AWS_ACCESS_KEY_ID"] || cfg.StorageSecretAccessKey != mockEnv["AWS_SECRET_ACCESS_KEY"] {
		t.Error("Neon credentials were not loaded from AWS_* environment variables")
	}
}

func TestConfig_ProductionValidation(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantError bool
	}{
		{
			name: "production missing all secrets",
			env: map[string]string{
				"APP_ENV": "production",
			},
			wantError: true,
		},
		{
			name: "production with all required secrets",
			env: map[string]string{
				"APP_ENV":                   "production",
				"API_BASE_URL":              "https://api.example.com",
				"FRONTEND_URL":              "https://app.example.com",
				"WEB_ORIGINS":               "https://app.example.com",
				"GOOGLE_REDIRECT_URI":       "https://api.example.com/v1/auth/google/callback",
				"DATABASE_URL":              "postgres://user:pass@host/db",
				"SESSION_HASH_KEY":          "supersecret256bitkeyrequiredhere",
				"OTP_HMAC_KEYS":             "key1:secretkeyversion1",
				"STORAGE_PROVIDER":          "neon",
				"STORAGE_ENDPOINT":          "https://storage.example.neon.tech",
				"STORAGE_REGION":            "us-east-2",
				"STORAGE_BUCKET":            "media",
				"STORAGE_ACCESS_KEY_ID":     "test-access-key",
				"STORAGE_SECRET_ACCESS_KEY": "test-storage-secret",
				"RESEND_API_KEY":            "test-resend-key",
				"RESEND_FROM_EMAIL":         "Relay <noreply@example.com>",
			},
			wantError: false,
		},
		{
			name: "production rejects insecure frontend origin",
			env: map[string]string{
				"APP_ENV":                   "production",
				"API_BASE_URL":              "https://api.example.com",
				"FRONTEND_URL":              "https://app.example.com",
				"WEB_ORIGINS":               "http://app.example.com",
				"GOOGLE_REDIRECT_URI":       "https://api.example.com/v1/auth/google/callback",
				"DATABASE_URL":              "postgres://user:pass@host/db",
				"SESSION_HASH_KEY":          "supersecret256bitkeyrequiredhere",
				"OTP_HMAC_KEYS":             "key1:secretkeyversion1",
				"STORAGE_PROVIDER":          "neon",
				"STORAGE_ENDPOINT":          "https://storage.example.neon.tech",
				"STORAGE_REGION":            "us-east-2",
				"STORAGE_BUCKET":            "media",
				"STORAGE_ACCESS_KEY_ID":     "test-access-key",
				"STORAGE_SECRET_ACCESS_KEY": "test-storage-secret",
				"RESEND_API_KEY":            "test-resend-key",
				"RESEND_FROM_EMAIL":         "Relay <noreply@example.com>",
			},
			wantError: true,
		},
		{
			name: "invalid environment",
			env: map[string]string{
				"APP_ENV": "invalid_env",
			},
			wantError: true,
		},
		{
			name: "invalid port",
			env: map[string]string{
				"PORT": "not_a_number",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				val, ok := tt.env[key]
				return val, ok
			}

			_, err := LoadFromLookup(lookup)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadFromLookup() error = %v, wantError = %v", err, tt.wantError)
			}
		})
	}
}

func TestConfig_RedactedSummary(t *testing.T) {
	mockEnv := map[string]string{
		"DATABASE_URL":     "postgres://user:pass@host/db",
		"SESSION_HASH_KEY": "secret",
		"OTP_HMAC_KEYS":    "otp_secret",
	}
	lookup := func(key string) (string, bool) {
		val, ok := mockEnv[key]
		return val, ok
	}

	cfg, err := LoadFromLookup(lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summary := cfg.RedactedSummary()
	if summary["has_database_url"] != true {
		t.Errorf("expected has_database_url to be true")
	}
	// Verify raw secret does not appear anywhere in summary
	for k, v := range summary {
		strVal := v
		if str, ok := strVal.(string); ok {
			if str == "secret" || str == "otp_secret" || str == "postgres://user:pass@host/db" {
				t.Errorf("sensitive value leaked in summary key: %s", k)
			}
		}
	}
}
