package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment constants
const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

// Config holds all validated configuration values for the service.
type Config struct {
	AppEnv                 string
	Port                   string
	APIBaseURL             string
	FrontendURL            string
	WebOrigins             []string
	DatabaseURL            string
	DatabaseDirectURL      string
	SessionHashKey         string
	OTPHMACKeys            string
	GoogleClientID         string
	GoogleClientSecret     string
	GoogleRedirectURI      string
	InternalAPIKeys        []string
	StorageProvider        string // "local", "s3", "neon", "r2"
	StorageLocalDir        string
	StorageBucket          string
	StorageEndpoint        string
	StorageRegion          string
	StorageAccessKeyID     string
	StorageSecretAccessKey string
	StoragePublicBaseURL   string
	DBMaxConns             int32
	DBMinConns             int32
	RequestTimeout         time.Duration
	NeonAuthBaseURL        string
	NeonAuthJWKSURL        string
	NeonRealtimeURL        string
	NeonRealtimeSecret     string
	ResendAPIKey           string
	EmailFrom              string
	SMTPHost               string
	SMTPPort               string
	SMTPUser               string
	SMTPPass               string
}

// loadDotEnv reads env files in precedence order, keeping values already set by the process or an earlier file.
func loadDotEnv(filenames ...string) {
	for _, fn := range filenames {
		file, err := os.Open(fn)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Strip surrounding quotes
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
				(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				if len(val) >= 2 {
					val = val[1 : len(val)-1]
				}
			}

			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		_ = file.Close()
	}
}

// Load loads configuration from environment variables, automatically loading .env if present.
func Load() (*Config, error) {
	loadDotEnv(".env", "../.env", "../../.env", "services/api/.env")
	return LoadFromLookup(os.LookupEnv)
}

// LoadFromLookup loads and validates configuration from a custom lookup function.
func LoadFromLookup(lookup func(string) (string, bool)) (*Config, error) {
	getEnv := func(key, defaultValue string) string {
		if val, exists := lookup(key); exists && strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val)
		}
		return defaultValue
	}

	appEnv := strings.ToLower(getEnv("APP_ENV", EnvDevelopment))
	switch appEnv {
	case EnvDevelopment, EnvStaging, EnvProduction, EnvTest:
	default:
		return nil, fmt.Errorf("invalid APP_ENV '%s': must be development, staging, production, or test", appEnv)
	}

	port := getEnv("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid PORT '%s': must be a valid port number", port)
	}

	apiBaseURL := getEnv("API_BASE_URL", "http://localhost:"+port)

	rawOrigins := getEnv("WEB_ORIGINS", "http://localhost:3000")
	var webOrigins []string
	for _, orig := range strings.Split(rawOrigins, ",") {
		trimmed := strings.TrimSpace(orig)
		if trimmed != "" {
			webOrigins = append(webOrigins, trimmed)
		}
	}

	dbMaxConns := int32(5)
	if val, exists := lookup("DB_MAX_CONNS"); exists && strings.TrimSpace(val) != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 32)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid DB_MAX_CONNS '%s': must be a positive integer", val)
		}
		dbMaxConns = int32(parsed)
	}

	dbMinConns := int32(1)
	if val, exists := lookup("DB_MIN_CONNS"); exists && strings.TrimSpace(val) != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 32)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("invalid DB_MIN_CONNS '%s': must be non-negative integer", val)
		}
		dbMinConns = int32(parsed)
	}

	timeoutSec := 15
	if val, exists := lookup("REQUEST_TIMEOUT_SECONDS"); exists && strings.TrimSpace(val) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("invalid REQUEST_TIMEOUT_SECONDS '%s': must be a positive integer", val)
		}
		timeoutSec = parsed
	}

	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")

	rawInternalKeys := getEnv("INTERNAL_API_KEYS", "")
	var internalAPIKeys []string
	if rawInternalKeys != "" {
		for _, k := range strings.Split(rawInternalKeys, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				internalAPIKeys = append(internalAPIKeys, k)
			}
		}
	}

	storageProvider := strings.ToLower(getEnv("STORAGE_PROVIDER", ""))
	if storageProvider == "" {
		if getEnv("AWS_ENDPOINT_URL_S3", "") != "" {
			storageProvider = "neon"
		} else {
			storageProvider = "local"
		}
	}
	storageLocalDir := getEnv("STORAGE_LOCAL_DIR", "uploads")
	storageBucket := getEnv("STORAGE_BUCKET", "")
	if storageProvider == "neon" && storageBucket == "" {
		storageBucket = "uploads"
	}

	cfg := &Config{
		AppEnv:                 appEnv,
		Port:                   port,
		APIBaseURL:             apiBaseURL,
		FrontendURL:            frontendURL,
		WebOrigins:             webOrigins,
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		DatabaseDirectURL:      getEnv("DATABASE_DIRECT_URL", getEnv("DATABASE_URL_UNPOOLED", "")),
		SessionHashKey:         getEnv("SESSION_HASH_KEY", ""),
		OTPHMACKeys:            getEnv("OTP_HMAC_KEYS", ""),
		GoogleClientID:         getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:     getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI:      getEnv("GOOGLE_REDIRECT_URI", apiBaseURL+"/v1/auth/google/callback"),
		InternalAPIKeys:        internalAPIKeys,
		StorageProvider:        storageProvider,
		StorageLocalDir:        storageLocalDir,
		StorageBucket:          storageBucket,
		StorageEndpoint:        getEnv("STORAGE_ENDPOINT", getEnv("AWS_ENDPOINT_URL_S3", "")),
		StorageRegion:          getEnv("STORAGE_REGION", getEnv("AWS_REGION", "auto")),
		StorageAccessKeyID:     getEnv("STORAGE_ACCESS_KEY_ID", getEnv("AWS_ACCESS_KEY_ID", "")),
		StorageSecretAccessKey: getEnv("STORAGE_SECRET_ACCESS_KEY", getEnv("AWS_SECRET_ACCESS_KEY", "")),
		StoragePublicBaseURL:   getEnv("STORAGE_PUBLIC_BASE_URL", apiBaseURL+"/v1/media/content"),
		DBMaxConns:             dbMaxConns,
		DBMinConns:             dbMinConns,
		RequestTimeout:         time.Duration(timeoutSec) * time.Second,
		NeonAuthBaseURL:        getEnv("NEON_AUTH_BASE_URL", ""),
		NeonAuthJWKSURL:        getEnv("NEON_AUTH_JWKS_URL", ""),
		NeonRealtimeURL:        getEnv("NEON_REALTIME_URL", ""),
		NeonRealtimeSecret:     getEnv("NEON_REALTIME_SECRET", ""),
		ResendAPIKey:           getEnv("RESEND_API_KEY", ""),
		EmailFrom:              getEnv("RESEND_FROM_EMAIL", getEnv("EMAIL_FROM", "")),
		SMTPHost:               getEnv("SMTP_HOST", ""),
		SMTPPort:               getEnv("SMTP_PORT", "587"),
		SMTPUser:               getEnv("SMTP_USER", ""),
		SMTPPass:               getEnv("SMTP_PASS", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// HasEmailProvider reports whether a real email delivery service (Resend or SMTP) is configured.
func (c *Config) HasEmailProvider() bool {
	return c.ResendAPIKey != "" || c.SMTPHost != ""
}

// HasGoogleOAuth reports whether Google OAuth credentials are fully provided.
func (c *Config) HasGoogleOAuth() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// validate ensures mandatory variables are set for staging and production.
func (c *Config) validate() error {
	switch c.StorageProvider {
	case "local", "s3", "r2", "neon":
	default:
		return errors.New("STORAGE_PROVIDER must be one of local, s3, r2, or neon")
	}
	if c.StorageProvider == "s3" || c.StorageProvider == "r2" || c.StorageProvider == "neon" {
		if c.StorageBucket == "" || c.StorageAccessKeyID == "" || c.StorageSecretAccessKey == "" {
			return errors.New("remote storage requires STORAGE_BUCKET, STORAGE_ACCESS_KEY_ID and STORAGE_SECRET_ACCESS_KEY")
		}
	}
	if c.StorageProvider == "neon" {
		if c.StorageEndpoint == "" {
			return errors.New("Neon Object Storage requires STORAGE_ENDPOINT")
		}
		if c.StorageRegion == "" || c.StorageRegion == "auto" {
			return errors.New("Neon Object Storage requires its explicit STORAGE_REGION")
		}
	}
	if c.AppEnv == EnvProduction || c.AppEnv == EnvStaging {
		if err := requireHTTPSURL("API_BASE_URL", c.APIBaseURL, false); err != nil {
			return err
		}
		if err := requireHTTPSURL("FRONTEND_URL", c.FrontendURL, false); err != nil {
			return err
		}
		if err := requireHTTPSURL("GOOGLE_REDIRECT_URI", c.GoogleRedirectURI, true); err != nil {
			return err
		}
		if len(c.WebOrigins) == 0 {
			return errors.New("WEB_ORIGINS must include the production frontend origin")
		}
		for _, origin := range c.WebOrigins {
			if err := requireHTTPSURL("WEB_ORIGINS", origin, false); err != nil {
				return err
			}
		}
		var missing []string
		if c.DatabaseURL == "" {
			missing = append(missing, "DATABASE_URL")
		}
		if c.SessionHashKey == "" {
			missing = append(missing, "SESSION_HASH_KEY")
		}
		if c.OTPHMACKeys == "" {
			missing = append(missing, "OTP_HMAC_KEYS")
		}
		if c.StorageProvider != "neon" {
			missing = append(missing, "STORAGE_PROVIDER=neon")
		}
		if !c.HasEmailProvider() {
			missing = append(missing, "RESEND_API_KEY (or SMTP_HOST)")
		}
		if c.HasEmailProvider() && c.EmailFrom == "" {
			missing = append(missing, "RESEND_FROM_EMAIL (or EMAIL_FROM)")
		}
		if len(missing) > 0 {
			return errors.New("missing required environment variables in " + c.AppEnv + ": " + strings.Join(missing, ", "))
		}
	}
	return nil
}

func requireHTTPSURL(name, raw string, allowPath bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || strings.Contains(u.Host, "*") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s must be an absolute HTTPS URL without credentials, wildcard host, query, or fragment", name)
	}
	if !allowPath && u.Path != "" && u.Path != "/" {
		return fmt.Errorf("%s must be an origin without a path", name)
	}
	return nil
}

// IsProduction returns true if current environment is production.
func (c *Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

// RedactedSummary returns safe non-sensitive configuration details for observability.
func (c *Config) RedactedSummary() map[string]any {
	hasDB := c.DatabaseURL != ""
	hasDirectDB := c.DatabaseDirectURL != ""
	hasSessionKey := c.SessionHashKey != ""
	hasOTPKeys := c.OTPHMACKeys != ""
	hasNeonAuth := c.NeonAuthBaseURL != ""
	hasNeonRealtime := c.NeonRealtimeURL != ""

	return map[string]any{
		"app_env":             c.AppEnv,
		"port":                c.Port,
		"api_base_url":        c.APIBaseURL,
		"frontend_url":        c.FrontendURL,
		"web_origins":         c.WebOrigins,
		"storage_provider":    c.StorageProvider,
		"has_database_url":    hasDB,
		"has_direct_db_url":   hasDirectDB,
		"has_session_key":     hasSessionKey,
		"has_otp_keys":        hasOTPKeys,
		"has_google_oauth":    c.GoogleClientID != "" && c.GoogleClientSecret != "",
		"has_internal_keys":   len(c.InternalAPIKeys) > 0,
		"has_neon_auth":       hasNeonAuth,
		"has_neon_realtime":   hasNeonRealtime,
		"db_max_conns":        c.DBMaxConns,
		"db_min_conns":        c.DBMinConns,
		"request_timeout_sec": int(c.RequestTimeout.Seconds()),
	}
}
