package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/session"
	"github.com/xynoss/relay/services/api/internal/users"
)

var (
	ErrInvalidEmailFormat       = errors.New("invalid email address format")
	ErrRateLimited              = errors.New("too many requests; please slow down")
	ErrEmailDeliveryUnavailable = errors.New("email delivery unavailable")
	ErrInvalidReturnPath        = errors.New("invalid return path: must be a relative path starting with /")
	ErrOAuthStateInvalid        = errors.New("invalid or expired OAuth state parameter")
	ErrGoogleEmailUnverified    = errors.New("google account email is not verified")
	ErrGoogleOAuthNotConfigured = errors.New("google oauth is not configured on the server")
	ErrIdentityNotLinked        = errors.New("login method is not linked to this account")
)

// ChallengeResponse holds the generic safe payload returned to the client.
type ChallengeResponse struct {
	ChallengeID       string `json:"challenge_id"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
}

// VerificationResponse holds the result of a successful OTP or OAuth verification.
type VerificationResponse struct {
	User               *users.CurrentUserView `json:"user"`
	OnboardingRequired bool                   `json:"onboarding_required"`
	CSRFToken          string                 `json:"csrf_token"`
}

// Service manages the entire authentication lifecycle including OTP and Google OAuth.
type Service struct {
	cfg            *config.Config
	store          Store
	usersStore     users.Store
	sessionService *session.Service
	emailSender    EmailSender
	rateLimiter    *RateLimiter
	httpClient     *http.Client
}

// NewService constructs a new auth service instance.
func NewService(
	cfg *config.Config,
	store Store,
	usersStore users.Store,
	sessionService *session.Service,
	emailSender EmailSender,
) *Service {
	return &Service{
		cfg:            cfg,
		store:          store,
		usersStore:     usersStore,
		sessionService: sessionService,
		emailSender:    emailSender,
		rateLimiter:    NewRateLimiter(),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// RequestOTP creates an anti-enumeration email OTP challenge and dispatches the code.
func (s *Service) RequestOTP(ctx context.Context, clientIP, email string) (*ChallengeResponse, error) {
	if s.cfg.AppEnv != config.EnvTest && !s.cfg.HasEmailProvider() {
		return nil, ErrEmailDeliveryUnavailable
	}
	emailNormalized := strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(emailNormalized, "@") || !strings.Contains(emailNormalized, ".") || len(emailNormalized) < 5 {
		return nil, ErrInvalidEmailFormat
	}

	// 1. IP Rate limit: max 10 challenges per 15 minutes per IP
	if allowed, _ := s.rateLimiter.Allow(HashKey("ip:otp_req", clientIP), 10, 15*time.Minute); !allowed {
		return nil, ErrRateLimited
	}

	// 2. Email Resend Cooldown (60 seconds)
	if remaining, inCooldown, err := s.store.GetLatestChallengeCooldown(ctx, emailNormalized); err == nil && inCooldown {
		return &ChallengeResponse{
			ChallengeID:       "cooldown",
			RetryAfterSeconds: int(remaining.Seconds()) + 1,
		}, nil
	}

	// 3. Generate uniform 6-digit CSPRNG code
	code, err := Generate6DigitCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	// 4. Compute HMAC
	keyVersion := "v1"
	hmacKey := s.cfg.OTPHMACKeys
	if hmacKey == "" {
		hmacKey = "xynoss-dev-otp-hmac-key-default-32-chars-long"
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	// Invert binding: HMAC = ComputeOTPHMAC(code, emailNormalized, hmacKey) or with temporary placeholder
	// To bind to challengeID, we pre-generate a challenge UUID or use email+timestamp
	codeHMAC := ComputeOTPHMAC(code, emailNormalized, hmacKey)

	challengeID, err := s.store.CreateChallenge(ctx, emailNormalized, codeHMAC, keyVersion, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge: %w", err)
	}

	// 5. Send Email via EmailSender
	if err := s.emailSender.SendOTP(ctx, emailNormalized, code); err != nil {
		return nil, ErrEmailDeliveryUnavailable
	}

	return &ChallengeResponse{
		ChallengeID:       challengeID,
		RetryAfterSeconds: 60,
	}, nil
}

// VerifyOTP checks the code, increments attempt counters, creates or loads the user, and starts a session.
func (s *Service) VerifyOTP(ctx context.Context, clientIP, challengeID, code, userAgent string) (*session.TokenPair, *session.Session, string, *users.CurrentUserView, error) {
	trimmedCode := strings.TrimSpace(code)
	if len(trimmedCode) != 6 {
		return nil, nil, "", nil, ErrInvalidCode
	}

	// 1. IP Rate limit: max 15 verification attempts per 15 minutes
	if allowed, _ := s.rateLimiter.Allow(HashKey("ip:otp_ver", clientIP), 15, 15*time.Minute); !allowed {
		return nil, nil, "", nil, ErrRateLimited
	}

	// 2. Verify and consume in database
	hmacKey := s.cfg.OTPHMACKeys
	if hmacKey == "" {
		hmacKey = "xynoss-dev-otp-hmac-key-default-32-chars-long"
	}

	emailNormalized, err := s.store.VerifyAndConsume(ctx, challengeID, trimmedCode, hmacKey)
	if err != nil {
		return nil, nil, "", nil, err
	}

	// 3. Find or create user
	u, err := s.usersStore.FindUserByEmail(ctx, emailNormalized)
	var p *users.Profile
	if errors.Is(err, users.ErrUserNotFound) {
		u, p, err = s.usersStore.CreateUserWithProfile(ctx, emailNormalized, "email_otp", emailNormalized)
		if err != nil {
			return nil, nil, "", nil, fmt.Errorf("failed to provision user: %w", err)
		}
	} else if err != nil {
		return nil, nil, "", nil, fmt.Errorf("failed to query user: %w", err)
	} else {
		identity, identityErr := s.usersStore.FindIdentity(ctx, "email_otp", emailNormalized)
		if errors.Is(identityErr, users.ErrIdentityNotFound) || (identityErr == nil && identity.UserID != u.ID) {
			return nil, nil, "", nil, ErrIdentityNotLinked
		}
		if identityErr != nil {
			return nil, nil, "", nil, fmt.Errorf("failed to query email identity: %w", identityErr)
		}
		p, err = s.usersStore.GetProfile(ctx, u.ID)
		if err != nil {
			return nil, nil, "", nil, fmt.Errorf("failed to query user profile: %w", err)
		}
	}

	// 4. Create session
	tokenPair, sess, csrfToken, err := s.sessionService.CreateSession(ctx, u.ID, userAgent)
	if err != nil {
		return nil, nil, "", nil, fmt.Errorf("failed to create session: %w", err)
	}

	userView := &users.CurrentUserView{
		ID:                 u.ID,
		Username:           u.Username,
		OnboardingRequired: !u.IsOnboardingCompleted(),
		Profile:            p,
	}

	return tokenPair, sess, csrfToken, userView, nil
}

// StartGoogleOAuth constructs the OAuth authorization URL with PKCE and state protection.
func (s *Service) StartGoogleOAuth(ctx context.Context, returnPath string) (string, string, string, error) {
	if !s.cfg.HasGoogleOAuth() || len(s.cfg.SessionHashKey) < 32 {
		return "", "", "", ErrGoogleOAuthNotConfigured
	}

	if returnPath == "" {
		returnPath = "/app"
	}
	if !strings.HasPrefix(returnPath, "/") || strings.HasPrefix(returnPath, "//") || strings.Contains(returnPath, "\\") {
		return "", "", "", ErrInvalidReturnPath
	}

	stateKey := s.cfg.SessionHashKey
	state := GenerateOAuthState(stateKey, returnPath)

	verifier, codeChallenge, err := GeneratePKCE()
	if err != nil {
		return "", "", "", err
	}

	redirectURI := s.cfg.GoogleRedirectURI
	if redirectURI == "" {
		redirectURI = s.cfg.APIBaseURL + "/v1/auth/google/callback"
	}

	clientID := s.cfg.GoogleClientID
	if clientID == "" {
		clientID = "mock-google-client-id"
	}

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid%%20email%%20profile&state=%s&code_challenge=%s&code_challenge_method=S256",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
	)

	return authURL, state, verifier, nil
}

// GoogleUserInfo represents claims returned by Google OpenID Connect.
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// HandleGoogleCallback processes Google OAuth authorization code exchange and session establishment.
func (s *Service) HandleGoogleCallback(ctx context.Context, code, state, verifier, userAgent string) (*session.TokenPair, *session.Session, string, string, error) {
	if !s.cfg.HasGoogleOAuth() || len(s.cfg.SessionHashKey) < 32 {
		return nil, nil, "", "", ErrGoogleOAuthNotConfigured
	}
	if verifier == "" || code == "" {
		return nil, nil, "", "", ErrOAuthStateInvalid
	}
	stateKey := s.cfg.SessionHashKey

	returnPath, valid := VerifyOAuthState(stateKey, state)
	if !valid {
		return nil, nil, "", "", ErrOAuthStateInvalid
	}

	var userInfo GoogleUserInfo
	{
		// Real Google Token Exchange
		tokenURL := "https://oauth2.googleapis.com/token"
		values := url.Values{}
		values.Set("code", code)
		values.Set("client_id", s.cfg.GoogleClientID)
		values.Set("client_secret", s.cfg.GoogleClientSecret)
		values.Set("redirect_uri", s.cfg.GoogleRedirectURI)
		values.Set("grant_type", "authorization_code")
		values.Set("code_verifier", verifier)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to build token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to exchange google code: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, nil, "", "", fmt.Errorf("google token exchange failed with status %d: %s", resp.StatusCode, string(body))
		}

		var tokenResp struct {
			AccessToken string `json:"access_token"`
			IDToken     string `json:"id_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to decode google token response: %w", err)
		}

		// Fetch User Info
		userinfoReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
		if err != nil {
			return nil, nil, "", "", err
		}
		userinfoReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

		userinfoResp, err := s.httpClient.Do(userinfoReq)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to fetch google userinfo: %w", err)
		}
		defer userinfoResp.Body.Close()

		if userinfoResp.StatusCode != http.StatusOK {
			return nil, nil, "", "", fmt.Errorf("google userinfo request returned %d", userinfoResp.StatusCode)
		}

		if err := json.NewDecoder(userinfoResp.Body).Decode(&userInfo); err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to decode google userinfo: %w", err)
		}
	}

	if !userInfo.EmailVerified {
		return nil, nil, "", "", ErrGoogleEmailUnverified
	}

	emailNormalized := strings.ToLower(strings.TrimSpace(userInfo.Email))

	// Find identity by Google subject
	ident, err := s.usersStore.FindIdentity(ctx, "google", userInfo.Sub)
	var u *users.User
	if err == nil {
		u, err = s.usersStore.GetByID(ctx, ident.UserID)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to load user for identity: %w", err)
		}
	} else if errors.Is(err, users.ErrIdentityNotFound) {
		// Identity not found: check if user exists by email to link account
		u, err = s.usersStore.FindUserByEmail(ctx, emailNormalized)
		if errors.Is(err, users.ErrUserNotFound) {
			// New user
			u, _, err = s.usersStore.CreateUserWithProfile(ctx, emailNormalized, "google", userInfo.Sub)
			if err != nil {
				return nil, nil, "", "", fmt.Errorf("failed to create user: %w", err)
			}
		} else if err == nil {
			return nil, nil, "", "", ErrIdentityNotLinked
		} else {
			return nil, nil, "", "", fmt.Errorf("failed to query user: %w", err)
		}
	} else {
		return nil, nil, "", "", fmt.Errorf("failed to query identity: %w", err)
	}

	// Create session
	tokenPair, sess, csrfToken, err := s.sessionService.CreateSession(ctx, u.ID, userAgent)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to create session: %w", err)
	}

	return tokenPair, sess, csrfToken, returnPath, nil
}
