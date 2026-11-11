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
				"APP_ENV":          "production",
				"DATABASE_URL":     "postgres://user:pass@host/db",
				"SESSION_HASH_KEY": "supersecret256bitkeyrequiredhere",
				"OTP_HMAC_KEYS":    "key1:secretkeyversion1",
			},
			wantError: false,
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
