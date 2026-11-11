package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/session"
	"github.com/xynoss/relay/services/api/internal/users"
)

type failingEmailSender struct{}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResendOTPRequest(t *testing.T) {
	sender := NewResendEmailSender("test-key", "auth@example.com", nil)
	sender.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != "https://api.resend.com/emails" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("unexpected Resend request")
		}
		var body struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			HTML    string   `json:"html"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.From != "auth@example.com" || len(body.To) != 1 || body.To[0] != "recipient@example.com" || !strings.Contains(body.HTML, "123456") || strings.Contains(body.Subject, "123456") {
			t.Fatal("unexpected OTP email payload")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}
	if err := sender.SendOTP(context.Background(), "recipient@example.com", "123456"); err != nil {
		t.Fatal(err)
	}
}

func (failingEmailSender) SendOTP(context.Context, string, string) error {
	return errors.New("delivery failed")
}

func TestRequestOTPRejectsUnavailableDelivery(t *testing.T) {
	t.Run("production without email provider", func(t *testing.T) {
		svc := NewService(&config.Config{AppEnv: config.EnvProduction}, nil, nil, nil, NewTestDoubleEmailSender(nil))
		if _, err := svc.RequestOTP(context.Background(), "127.0.0.1", "test@example.com"); !errors.Is(err, ErrEmailDeliveryUnavailable) {
			t.Fatalf("expected delivery failure, got %v", err)
		}
	})
	t.Run("sender fails", func(t *testing.T) {
		cfg := &config.Config{AppEnv: config.EnvTest, OTPHMACKeys: "test-otp-hmac-key-32-chars-minimum!!"}
		svc := NewService(cfg, newMockAuthStore(), nil, nil, failingEmailSender{})
		if _, err := svc.RequestOTP(context.Background(), "127.0.0.1", "test@example.com"); !errors.Is(err, ErrEmailDeliveryUnavailable) {
			t.Fatalf("expected delivery failure, got %v", err)
		}
	})
}

func TestGoogleCallbackRequiresConfiguredProviderAndBrowserBinding(t *testing.T) {
	svc := NewService(&config.Config{}, nil, nil, nil, nil)
	_, _, _, _, err := svc.HandleGoogleCallback(context.Background(), "code", "state", "verifier", "test")
	if !errors.Is(err, ErrGoogleOAuthNotConfigured) {
		t.Fatalf("unconfigured provider accepted callback: %v", err)
	}
	h := NewHandler(svc, nil, nil, false, "http://localhost:3000")
	req := httptest.NewRequest("GET", "/v1/auth/google/callback?state=unexpected&code=code", nil)
	w := httptest.NewRecorder()
	h.GoogleCallback(w, req)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "oauth_failed") {
		t.Fatalf("callback without browser cookies must fail, status %d", w.Code)
	}
}

func TestGoogleDoesNotLinkExistingEmailAccount(t *testing.T) {
	ctx := context.Background()
	store := newMockUsersStore()
	_, _, err := store.CreateUserWithProfile(ctx, "same@example.com", "email_otp", "same@example.com")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{GoogleClientID: "test", GoogleClientSecret: "test", GoogleRedirectURI: "https://example.com/callback", SessionHashKey: "test-session-key-at-least-32-characters"}
	svc := NewService(cfg, nil, store, nil, nil)
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"access_token":"test"}`
		if r.URL.Host == "openidconnect.googleapis.com" {
			body = `{"sub":"google-sub","email":"same@example.com","email_verified":true}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	_, _, _, _, err = svc.HandleGoogleCallback(ctx, "code", GenerateOAuthState(cfg.SessionHashKey, "/app"), "verifier", "test")
	if !errors.Is(err, ErrIdentityNotLinked) {
		t.Fatalf("expected explicit link requirement, got %v", err)
	}
	if _, err := store.FindIdentity(ctx, "google", "google-sub"); !errors.Is(err, users.ErrIdentityNotFound) {
		t.Fatalf("unexpected Google identity created: %v", err)
	}
}

// In-Memory Test Stores for isolated unit testing without database
type mockAuthStore struct {
	challenges map[string]*challengeData
}

type challengeData struct {
	email       string
	codeHMAC    string
	expiresAt   time.Time
	attempts    int
	maxAttempts int
	consumed    bool
	invalidated bool
	createdAt   time.Time
}

func newMockAuthStore() *mockAuthStore {
	return &mockAuthStore{challenges: make(map[string]*challengeData)}
}

func (m *mockAuthStore) CreateChallenge(ctx context.Context, emailNormalized, codeHMAC, keyVersion string, expiresAt time.Time) (string, error) {
	for _, c := range m.challenges {
		if c.email == emailNormalized && !c.consumed && !c.invalidated {
			c.invalidated = true
		}
	}
	id := "test-challenge-" + time.Now().Format("150405.000000")
	m.challenges[id] = &challengeData{
		email:       emailNormalized,
		codeHMAC:    codeHMAC,
		expiresAt:   expiresAt,
		attempts:    0,
		maxAttempts: 5,
		consumed:    false,
		invalidated: false,
		createdAt:   time.Now(),
	}
	return id, nil
}

func (m *mockAuthStore) GetLatestChallengeCooldown(ctx context.Context, emailNormalized string) (time.Duration, bool, error) {
	var latest *challengeData
	for _, c := range m.challenges {
		if c.email == emailNormalized {
			if latest == nil || c.createdAt.After(latest.createdAt) {
				latest = c
			}
		}
	}
	if latest == nil {
		return 0, false, nil
	}
	elapsed := time.Since(latest.createdAt)
	if elapsed < 60*time.Second {
		return 60*time.Second - elapsed, true, nil
	}
	return 0, false, nil
}

func (m *mockAuthStore) VerifyAndConsume(ctx context.Context, challengeID, candidateCode, hmacKey string) (string, error) {
	c, ok := m.challenges[challengeID]
	if !ok {
		return "", ErrChallengeNotFound
	}
	if c.invalidated {
		return "", ErrChallengeInvalidated
	}
	if c.consumed {
		return "", ErrChallengeAlreadyConsumed
	}
	if time.Now().After(c.expiresAt) {
		return "", ErrChallengeExpired
	}
	c.attempts++
	if c.attempts > c.maxAttempts {
		return "", ErrTooManyAttempts
	}
	if !VerifyOTPHMAC(candidateCode, c.email, hmacKey, c.codeHMAC) {
		return "", ErrInvalidCode
	}
	c.consumed = true
	return c.email, nil
}

type mockUsersStore struct {
	users      map[string]*users.User
	profiles   map[string]*users.Profile
	identities map[string]*users.Identity
}

func newMockUsersStore() *mockUsersStore {
	return &mockUsersStore{
		users:      make(map[string]*users.User),
		profiles:   make(map[string]*users.Profile),
		identities: make(map[string]*users.Identity),
	}
}

func (m *mockUsersStore) CreateUserWithProfile(ctx context.Context, emailNormalized, provider, subject string) (*users.User, *users.Profile, error) {
	userID := "user-" + emailNormalized
	u := &users.User{
		ID:        userID,
		CreatedAt: time.Now(),
	}
	p := &users.Profile{
		UserID:       userID,
		DisplayName:  strings.Split(emailNormalized, "@")[0],
		Availability: "online",
		Revision:     1,
		UpdatedAt:    time.Now(),
	}
	m.users[userID] = u
	m.profiles[userID] = p
	m.identities[provider+":"+subject] = &users.Identity{
		ID:                      "ident-" + userID,
		UserID:                  userID,
		Provider:                provider,
		ProviderSubject:         subject,
		VerifiedEmailNormalized: &emailNormalized,
		VerifiedAt:              time.Now(),
	}
	return u, p, nil
}

func (m *mockUsersStore) GetByID(ctx context.Context, id string) (*users.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, users.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUsersStore) GetProfile(ctx context.Context, userID string) (*users.Profile, error) {
	p, ok := m.profiles[userID]
	if !ok {
		return nil, users.ErrProfileNotFound
	}
	return p, nil
}

func (m *mockUsersStore) UpdateProfile(ctx context.Context, userID string, p *users.Profile, expectedRevision int) (*users.Profile, error) {
	m.profiles[userID] = p
	return p, nil
}

func (m *mockUsersStore) SetUsername(ctx context.Context, userID, username, normalized string) error {
	if u, ok := m.users[userID]; ok {
		u.Username = &username
		u.UsernameNormalized = &normalized
	}
	return nil
}

func (m *mockUsersStore) SetAvatar(ctx context.Context, userID string, mediaID *string) error {
	if p, ok := m.profiles[userID]; ok {
		p.AvatarMediaID = mediaID
	}
	return nil
}

func (m *mockUsersStore) SetBanner(ctx context.Context, userID string, mediaID *string) error {
	if p, ok := m.profiles[userID]; ok {
		p.BannerMediaID = mediaID
	}
	return nil
}

func (m *mockUsersStore) CompleteOnboarding(ctx context.Context, userID, displayName string, username *string) (*users.User, *users.Profile, error) {
	u := m.users[userID]
	now := time.Now()
	u.OnboardingCompletedAt = &now
	if username != nil {
		u.Username = username
	}
	p := m.profiles[userID]
	p.DisplayName = displayName
	return u, p, nil
}

func (m *mockUsersStore) FindIdentity(ctx context.Context, provider, subject string) (*users.Identity, error) {
	ident, ok := m.identities[provider+":"+subject]
	if !ok {
		return nil, users.ErrIdentityNotFound
	}
	return ident, nil
}

func (m *mockUsersStore) FindUserByEmail(ctx context.Context, emailNormalized string) (*users.User, error) {
	for _, ident := range m.identities {
		if ident.VerifiedEmailNormalized != nil && *ident.VerifiedEmailNormalized == emailNormalized {
			return m.users[ident.UserID], nil
		}
	}
	return nil, users.ErrUserNotFound
}

func (m *mockUsersStore) LinkIdentity(ctx context.Context, userID, provider, subject, emailNormalized string) (*users.Identity, error) {
	ident := &users.Identity{
		ID:                      "ident-" + userID,
		UserID:                  userID,
		Provider:                provider,
		ProviderSubject:         subject,
		VerifiedEmailNormalized: &emailNormalized,
		VerifiedAt:              time.Now(),
	}
	m.identities[provider+":"+subject] = ident
	return ident, nil
}

func (m *mockUsersStore) SearchUsers(ctx context.Context, query string, limit int) ([]*users.UserSearchResult, error) {
	var results []*users.UserSearchResult
	for _, p := range m.profiles {
		results = append(results, &users.UserSearchResult{
			ID:           p.UserID,
			DisplayName:  p.DisplayName,
			Username:     "user_" + p.UserID,
			AvatarURL:    p.AvatarURL,
			Bio:          p.Bio,
			Availability: p.Availability,
		})
	}
	return results, nil
}

type mockSessionStore struct {
	sessions map[string]*session.Session
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*session.Session)}
}

func (m *mockSessionStore) CreateSession(ctx context.Context, s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) GetSessionByID(ctx context.Context, id string) (*session.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, session.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockSessionStore) GetBetterAuthSession(ctx context.Context, token string) (*session.Session, error) {
	return nil, session.ErrSessionNotFound
}

func (m *mockSessionStore) TouchSession(ctx context.Context, id string, lastSeen, idleExpires time.Time) error {
	if s, ok := m.sessions[id]; ok {
		s.LastSeenAt = lastSeen
		s.IdleExpiresAt = idleExpires
	}
	return nil
}

func (m *mockSessionStore) RevokeSession(ctx context.Context, id string) error {
	if s, ok := m.sessions[id]; ok {
		now := time.Now()
		s.RevokedAt = &now
	}
	return nil
}

func (m *mockSessionStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID {
			s.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockSessionStore) ListUserSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	var list []*session.Session
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *mockSessionStore) RegisterOrUpdateDevice(ctx context.Context, d *session.Device) (string, error) {
	return "mock-device-id", nil
}

func TestAuth_OTPFullLifecycleAndSecurity(t *testing.T) {
	cfg := &config.Config{
		AppEnv:         config.EnvTest,
		OTPHMACKeys:    "test-otp-hmac-key-32-chars-minimum!!",
		SessionHashKey: "test-session-key-32-chars-minimum!!",
	}

	authStore := newMockAuthStore()
	usersStore := newMockUsersStore()
	sessionStore := newMockSessionStore()
	sessionService := session.NewService(sessionStore, cfg.SessionHashKey)
	emailSender := NewTestDoubleEmailSender(nil)

	svc := NewService(cfg, authStore, usersStore, sessionService, emailSender)

	ctx := context.Background()
	clientIP := "192.168.1.100"
	testEmail := "alice@example.com"

	// 1. Request OTP
	resp, err := svc.RequestOTP(ctx, clientIP, testEmail)
	if err != nil {
		t.Fatalf("RequestOTP failed: %v", err)
	}
	if resp.ChallengeID == "" {
		t.Fatalf("expected valid challenge ID")
	}

	// Verify email sender received 6-digit code
	code := emailSender.GetLastCode(testEmail)
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	// 2. Cooldown check: second immediate request should trigger cooldown without sending a new code
	cooldownResp, err := svc.RequestOTP(ctx, clientIP, testEmail)
	if err != nil {
		t.Fatalf("cooldown request failed: %v", err)
	}
	if cooldownResp.ChallengeID != "cooldown" {
		t.Errorf("expected cooldown challenge ID, got %s", cooldownResp.ChallengeID)
	}

	// 3. Invalid code attempt
	_, _, _, _, err = svc.VerifyOTP(ctx, clientIP, resp.ChallengeID, "000000", "Test-Agent")
	if err != ErrInvalidCode {
		t.Errorf("expected ErrInvalidCode, got %v", err)
	}

	// 4. Successful verification with correct code
	tokenPair, sess, csrfToken, userView, err := svc.VerifyOTP(ctx, clientIP, resp.ChallengeID, code, "Test-Agent")
	if err != nil {
		t.Fatalf("VerifyOTP failed with correct code: %v", err)
	}
	if tokenPair.Token == "" || sess.ID == "" || csrfToken == "" {
		t.Errorf("expected non-empty tokens and session")
	}
	if userView.OnboardingRequired != true {
		t.Errorf("expected new user to require onboarding")
	}

	// 5. Replay attack: verifying again with the same code must fail (already consumed)
	_, _, _, _, err = svc.VerifyOTP(ctx, clientIP, resp.ChallengeID, code, "Test-Agent")
	if err != ErrChallengeAlreadyConsumed {
		t.Errorf("expected ErrChallengeAlreadyConsumed on replay, got %v", err)
	}

	// 6. Validate established session
	validSess, err := sessionService.ValidateSessionToken(ctx, tokenPair.Token)
	if err != nil {
		t.Fatalf("ValidateSessionToken failed: %v", err)
	}
	if validSess.UserID != userView.ID {
		t.Errorf("expected session user ID %s, got %s", userView.ID, validSess.UserID)
	}
}

func TestAuth_PKCEAndOAuthState(t *testing.T) {
	key := "test-state-secret-key-32-chars-long!!"

	// 1. Generate and verify valid state
	state := GenerateOAuthState(key, "/dashboard/custom")
	returnPath, valid := VerifyOAuthState(key, state)
	if !valid {
		t.Fatalf("expected valid state")
	}
	if returnPath != "/dashboard/custom" {
		t.Errorf("expected returnPath /dashboard/custom, got %s", returnPath)
	}

	// 2. Tampered state
	_, tamperedValid := VerifyOAuthState(key, state+"tampered")
	if tamperedValid {
		t.Errorf("expected tampered state to be invalid")
	}

	// 3. PKCE verification
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}
	if len(verifier) == 0 || len(challenge) == 0 {
		t.Errorf("empty PKCE verifier or challenge")
	}
}
