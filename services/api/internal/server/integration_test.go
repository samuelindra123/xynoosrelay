package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/database"
	"github.com/xynoss/relay/services/api/internal/response"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	testDB := os.Getenv("TEST_DATABASE_URL")
	if testDB == "" {
		t.Skip("TEST_DATABASE_URL is required for database integration tests")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}
	if configuredDB := strings.TrimSpace(cfg.DatabaseURL); configuredDB != "" && testDB == configuredDB {
		t.Fatal("refusing to run integration tests: TEST_DATABASE_URL matches configured DATABASE_URL")
	}
	cfg.DatabaseURL = testDB
	cfg.AppEnv = config.EnvTest
	cfg.ResendAPIKey = ""
	cfg.EmailFrom = ""
	cfg.SMTPHost = ""
	cfg.SMTPUser = ""
	cfg.SMTPPass = ""
	cfg.StorageProvider = "local"
	cfg.StorageLocalDir = t.TempDir()
	cfg.StoragePublicBaseURL = "http://integration.test/v1/media/content"
	if len(cfg.InternalAPIKeys) == 0 {
		cfg.InternalAPIKeys = []string{"integration-test-key"}
	}

	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg, nullLogger)
	if err != nil {
		t.Skipf("skipping test: database connection pool initialization failed: %v", err)
		return nil, func() {}
	}
	srv := New(cfg, nullLogger, pool)
	if srv.EmailSender() == nil {
		t.Fatal("integration server must use the in-memory email sandbox")
	}
	cleanup := func() { pool.Close() }
	return srv, cleanup
}

func integrationPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 20, G: 120, B: 220, A: 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func createIntegrationUser(t *testing.T, srv *Server, username string) (string, string, string) {
	t.Helper()
	ctx := context.Background()
	userID := uuid.NewString()
	uniqueUsername := username + "_" + uuid.NewString()[:8]
	email := uniqueUsername + "@example.com"
	_, err := srv.db.RawPool().Exec(ctx, `INSERT INTO users (id, username, username_normalized, onboarding_completed_at) VALUES ($1,$2,$2,NOW()); INSERT INTO profiles (user_id, display_name) VALUES ($1,$2); INSERT INTO identities (user_id, provider, provider_subject, verified_email_normalized) VALUES ($1,'integration',$3,$3)`, userID, uniqueUsername, email)
	if err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	tokens, _, csrf, err := srv.sessionService.CreateSession(ctx, userID, "integration-test")
	if err != nil {
		t.Fatalf("create integration session: %v", err)
	}
	return userID, tokens.Token, csrf
}

func integrationRequest(t *testing.T, client *http.Client, baseURL, token, method, path, body string, status int) []byte {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if method == http.MethodPost && strings.HasSuffix(path, "/messages") {
		req.Header.Set("Idempotency-Key", uuid.NewString())
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != status {
		t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, resp.StatusCode, string(data))
	}
	return data
}

func TestAPI_EndToEndRemainingAuthorizationMediaAndConcurrency(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := &http.Client{Timeout: 15 * time.Second}

	ownerID, ownerToken, _ := createIntegrationUser(t, srv, "audit-owner")
	memberID, memberToken, _ := createIntegrationUser(t, srv, "audit-member")
	outsiderID, outsiderToken, _ := createIntegrationUser(t, srv, "audit-outsider")
	defer func() {
		_, _ = srv.db.RawPool().Exec(context.Background(), `DELETE FROM workspaces WHERE owner_user_id IN ($1,$2,$3)`, ownerID, memberID, outsiderID)
		for _, id := range []string{ownerID, memberID, outsiderID} {
			_, _ = srv.db.RawPool().Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
		}
	}()

	workspaceBody := integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/workspaces", `{"name":"Audit Workspace `+uuid.NewString()[:8]+`"}`, http.StatusCreated)
	var workspace struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(workspaceBody, &workspace); err != nil || workspace.Data.ID == "" {
		t.Fatalf("workspace response missing id: %s", workspaceBody)
	}
	workspaceID := workspace.Data.ID
	if _, err := srv.db.RawPool().Exec(context.Background(), `INSERT INTO memberships (workspace_id,user_id,state) VALUES ($1,$2,'active')`, workspaceID, memberID); err != nil {
		t.Fatalf("add member: %v", err)
	}

	// Workspace retrieval and user search exercise persistence and authorization boundaries.
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodGet, "/v1/workspaces/"+workspaceID, "", http.StatusOK)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodGet, "/v1/workspaces/"+workspaceID, "", http.StatusOK)
	integrationRequest(t, client, ts.URL, outsiderToken, http.MethodGet, "/v1/workspaces/"+workspaceID, "", http.StatusForbidden)
	search := integrationRequest(t, client, ts.URL, ownerToken, http.MethodGet, "/v1/users/search?q=audit-member&limit=5", "", http.StatusOK)
	if !strings.Contains(string(search), memberID) {
		t.Fatalf("user search did not return persisted member: %s", search)
	}

	// Private channel access is denied before an ACL grant and allowed afterward.
	channelBody := integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/workspaces/"+workspaceID+"/channels", `{"name":"Private Audit","visibility":"private"}`, http.StatusCreated)
	var channel struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(channelBody, &channel); err != nil || channel.Data.ID == "" {
		t.Fatalf("channel response missing id: %s", channelBody)
	}
	channelID := channel.Data.ID
	integrationRequest(t, client, ts.URL, memberToken, http.MethodGet, "/v1/channels/"+channelID+"/messages", "", http.StatusNotFound)
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodPut, "/v1/channels/"+channelID+"/access", `{"user_id":"`+memberID+`"}`, http.StatusNoContent)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodGet, "/v1/channels/"+channelID+"/messages", "", http.StatusOK)
	integrationRequest(t, client, ts.URL, outsiderToken, http.MethodPut, "/v1/channels/"+channelID+"/access", `{"user_id":"`+outsiderID+`"}`, http.StatusNotFound)

	msg := integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/channels/"+channelID+"/messages", `{"text":"editable"}`, http.StatusCreated)
	var message struct {
		Data struct {
			Sequence int64 `json:"sequence"`
			Revision int   `json:"revision"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg, &message); err != nil || message.Data.Sequence == 0 {
		t.Fatalf("message response missing sequence: %s", msg)
	}
	seq := message.Data.Sequence
	editPath := "/v1/channels/" + channelID + "/messages/" + fmt.Sprint(seq)
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodPatch, editPath, `{"text":"edited","revision":1}`, http.StatusOK)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPatch, editPath, `{"text":"hijack","revision":2}`, http.StatusConflict)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPut, editPath+"/reactions/heart", `{}`, http.StatusNoContent)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPut, editPath+"/reactions/heart", `{}`, http.StatusNoContent)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPost, "/v1/channels/"+channelID+"/read", `{"last_read_sequence":1}`, http.StatusOK)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPost, "/v1/channels/"+channelID+"/read", `{"last_read_sequence":0}`, http.StatusOK)
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodDelete, editPath, `{}`, http.StatusNoContent)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPut, editPath+"/reactions/heart", `{}`, http.StatusNotFound)

	// Avatar upload/finalize/attach/remove verifies object storage and profile persistence.
	pngData := integrationPNG(t)
	intentBody := integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/media/uploads", `{"purpose":"avatar","content_type":"image/png","size_bytes":`+fmt.Sprint(len(pngData))+`}`, http.StatusCreated)
	var intent struct {
		Data struct {
			MediaID   string `json:"media_id"`
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(intentBody, &intent); err != nil || intent.Data.MediaID == "" {
		t.Fatalf("media intent response missing id: %s", intentBody)
	}
	if _, err := srv.storageProv.PutObject(context.Background(), intent.Data.ObjectKey, bytes.NewReader(pngData), int64(len(pngData)), "image/png"); err != nil {
		t.Fatalf("put avatar object: %v", err)
	}
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/media/"+intent.Data.MediaID+"/finalize", `{}`, http.StatusOK)
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodPost, "/v1/users/me/avatar", `{"media_id":"`+intent.Data.MediaID+`"}`, http.StatusOK)
	integrationRequest(t, client, ts.URL, memberToken, http.MethodPost, "/v1/users/me/avatar", `{"media_id":"`+intent.Data.MediaID+`"}`, http.StatusForbidden)
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodDelete, "/v1/users/me/avatar", `{}`, http.StatusOK)

	// Two independent sessions can be revoked concurrently; the target session is no longer valid.
	tokens, sess, _, err := srv.sessionService.CreateSession(context.Background(), ownerID, "second-device")
	if err != nil {
		t.Fatal(err)
	}
	integrationRequest(t, client, ts.URL, ownerToken, http.MethodDelete, "/v1/me/sessions/"+sess.ID, `{}`, http.StatusNoContent)
	if _, err := srv.sessionService.ValidateSessionToken(context.Background(), tokens.Token); err == nil {
		t.Fatal("explicit session revocation did not invalidate the target session")
	}

	tokens, sess, _, err = srv.sessionService.CreateSession(context.Background(), ownerID, "concurrent-device")
	if err != nil {
		t.Fatal(err)
	}
	var revokeErrs [2]error
	start := make(chan struct{})
	for i := range revokeErrs {
		go func(i int) {
			<-start
			_, revokeErrs[i] = srv.db.RawPool().Exec(context.Background(), `UPDATE sessions SET revoked_at=NOW() WHERE id=$1 AND revoked_at IS NULL`, sess.ID)
		}(i)
	}
	close(start)
	time.Sleep(50 * time.Millisecond)
	for _, err := range revokeErrs {
		if err != nil {
			t.Fatalf("concurrent session revocation: %v", err)
		}
	}
	if _, err := srv.sessionService.ValidateSessionToken(context.Background(), tokens.Token); err == nil {
		t.Fatal("revoked concurrent session remained valid")
	}
}

func cleanupTestAccount(t *testing.T, srv *Server, email string) {
	t.Helper()
	pool := srv.db.RawPool()
	userIDs := `SELECT user_id FROM identities WHERE verified_email_normalized = $1`
	if _, err := pool.Exec(context.Background(), `DELETE FROM workspaces WHERE owner_user_id IN (`+userIDs+`)`, email); err != nil {
		t.Errorf("cleanup test workspaces: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM conversations WHERE id IN (SELECT conversation_id FROM conversation_members WHERE user_id IN (`+userIDs+`))`, email); err != nil {
		t.Errorf("cleanup test conversations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id IN (`+userIDs+`)`, email); err != nil {
		t.Errorf("cleanup test account: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM otp_challenges WHERE email_normalized = $1`, email); err != nil {
		t.Errorf("cleanup test OTP challenges: %v", err)
	}
}

func TestAPI_EndToEndAuthAndProfileFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	if !srv.db.IsConnected() {
		t.Skip("skipping integration test: database is not connected")
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	testEmail := "e2e-user-" + uuid.NewString() + "@example.com"
	defer cleanupTestAccount(t, srv, testEmail)

	// 1. Request OTP challenge
	reqBody, _ := json.Marshal(map[string]string{"email": testEmail})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/auth/otp/challenges", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("challenge request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 202 Accepted, got %d: %s", resp.StatusCode, string(body))
	}

	var challengeEnvelope struct {
		Data struct {
			ChallengeID       string `json:"challenge_id"`
			RetryAfterSeconds int    `json:"retry_after_seconds"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&challengeEnvelope); err != nil {
		t.Fatalf("failed to parse challenge response: %v", err)
	}

	challengeID := challengeEnvelope.Data.ChallengeID
	code := srv.EmailSender().GetLastCode(testEmail)
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code in test emailer, got %s", code)
	}

	// 2. Verify OTP
	verifyBody, _ := json.Marshal(map[string]string{
		"challenge_id": challengeID,
		"code":         code,
	})
	reqVerify, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/auth/otp/verifications", bytes.NewReader(verifyBody))
	reqVerify.Header.Set("Content-Type", "application/json")
	respVerify, err := client.Do(reqVerify)
	if err != nil {
		t.Fatalf("verify request failed: %v", err)
	}
	defer respVerify.Body.Close()

	if respVerify.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respVerify.Body)
		t.Fatalf("expected 200 OK, got %d: %s", respVerify.StatusCode, string(body))
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, c := range respVerify.Cookies() {
		if strings.Contains(c.Name, "session") {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session cookie in Set-Cookie headers")
	}

	var verifyResult struct {
		Data struct {
			OnboardingRequired bool   `json:"onboarding_required"`
			CSRFToken          string `json:"csrf_token"`
			User               struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respVerify.Body).Decode(&verifyResult); err != nil {
		t.Fatalf("failed to decode verification response: %v", err)
	}

	csrfToken := verifyResult.Data.CSRFToken
	if csrfToken == "" {
		t.Fatalf("expected CSRF token")
	}

	// 3. Call GET /v1/me
	reqMe, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/me", nil)
	reqMe.AddCookie(sessionCookie)
	respMe, err := client.Do(reqMe)
	if err != nil {
		t.Fatalf("GET /v1/me failed: %v", err)
	}
	defer respMe.Body.Close()

	if respMe.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /v1/me, got %d", respMe.StatusCode)
	}

	// 4. Complete Onboarding (POST /v1/me/onboarding/complete)
	uniqueUsername := "user_" + time.Now().Format("150405")
	onboardBody, _ := json.Marshal(map[string]string{
		"display_name": "Test User",
		"username":     uniqueUsername,
	})
	reqOnboard, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/me/onboarding/complete", bytes.NewReader(onboardBody))
	reqOnboard.Header.Set("Content-Type", "application/json")
	reqOnboard.Header.Set("X-CSRF-Token", csrfToken)
	reqOnboard.AddCookie(sessionCookie)
	respOnboard, err := client.Do(reqOnboard)
	if err != nil {
		t.Fatalf("onboarding failed: %v", err)
	}
	defer respOnboard.Body.Close()

	if respOnboard.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respOnboard.Body)
		t.Fatalf("expected 200 OK for onboarding, got %d: %s", respOnboard.StatusCode, string(body))
	}

	// 5. Update Profile (PATCH /v1/me/profile)
	patchBody, _ := json.Marshal(map[string]string{
		"bio": "Building on XYNOSS Relay backend!",
	})
	reqPatch, _ := http.NewRequest(http.MethodPatch, ts.URL+"/v1/me/profile", bytes.NewReader(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("X-CSRF-Token", csrfToken)
	reqPatch.AddCookie(sessionCookie)
	respPatch, err := client.Do(reqPatch)
	if err != nil {
		t.Fatalf("patch profile failed: %v", err)
	}
	defer respPatch.Body.Close()

	if respPatch.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for patch profile, got %d", respPatch.StatusCode)
	}

	// 6. List active sessions (GET /v1/me/sessions)
	reqSessions, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/me/sessions", nil)
	reqSessions.AddCookie(sessionCookie)
	respSessions, err := client.Do(reqSessions)
	if err != nil {
		t.Fatalf("GET /v1/me/sessions failed: %v", err)
	}
	defer respSessions.Body.Close()

	if respSessions.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for sessions list, got %d", respSessions.StatusCode)
	}

	// Exercise workspace/channel/invitation routes against the real database.
	var requestKey string
	call := func(method, path, payload string, status int, authenticated bool) json.RawMessage {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+"/v1"+path, strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if method == http.MethodPost && strings.HasSuffix(path, "/messages") && requestKey != "" {
			req.Header.Set("Idempotency-Key", requestKey)
		}
		if authenticated {
			req.AddCookie(sessionCookie)
			req.Header.Set("X-CSRF-Token", csrfToken)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, res.StatusCode, status, body)
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if len(body) > 0 {
			_ = json.Unmarshal(body, &envelope)
		}
		return envelope.Data
	}
	var workspace struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(call("POST", "/workspaces", `{"name":"E2E workspace `+time.Now().Format("150405.000000000")+`"}`, 201, true), &workspace); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := srv.db.RawPool().Exec(context.Background(), "DELETE FROM workspaces WHERE id = $1", workspace.ID)
		if err != nil {
			t.Errorf("cleanup workspace: %v", err)
		}
	}()
	call("GET", "/workspaces", "", 200, true)
	var channel struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(call("POST", "/workspaces/"+workspace.ID+"/channels", `{"name":"engineering"}`, 201, true), &channel); err != nil {
		t.Fatal(err)
	}
	path := "/channels/" + channel.ID + "/messages"
	call("GET", path, "", 401, false)
	call("POST", path, `{"text":"   "}`, 422, true)
	call("POST", path, `{"text":"tanpa kunci"}`, 400, true)
	requestKey = uuid.NewString()
	first := call("POST", path, `{"text":"Halo dari server"}`, 201, true)
	replayed := call("POST", path, `{"text":"Halo dari server"}`, 201, true)
	if string(first) != string(replayed) {
		t.Fatal("retry returned a different message")
	}
	call("POST", path, `{"text":"payload berbeda"}`, 409, true)
	requestKey = uuid.NewString()
	call("POST", path, `{"text":"Pesan kedua"}`, 201, true)
	var sent []struct {
		Sequence int    `json:"sequence"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(call("GET", path, "", 200, true), &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || sent[0].Text != "Halo dari server" || sent[1].Text != "Pesan kedua" || sent[0].Sequence >= sent[1].Sequence {
		t.Fatalf("unexpected message order: %+v", sent)
	}
	call("POST", "/workspaces/"+workspace.ID+"/invites", `{"duration_hours":-1}`, 422, true)
	var invite struct {
		Code string `json:"code"`
		URL  string `json:"invite_url"`
	}
	if err := json.Unmarshal(call("POST", "/workspaces/"+workspace.ID+"/invites", `{"duration_hours":1,"max_uses":1}`, 201, true), &invite); err != nil {
		t.Fatal(err)
	}
	if len(invite.Code) != 12 || !strings.Contains(invite.URL, "/invite/"+invite.Code) {
		t.Fatal("invalid invite contract")
	}
	call("GET", "/invites/"+invite.Code, "", 200, false)
	call("POST", "/invites/"+invite.Code+"/accept", "", 401, false)
	call("POST", "/invites/"+invite.Code+"/accept", "", 200, true)
	call("POST", "/invites/"+invite.Code+"/accept", "", 200, true)
	var invitations []struct {
		ID   string `json:"id"`
		Uses int    `json:"uses"`
	}
	if err := json.Unmarshal(call("GET", "/workspaces/"+workspace.ID+"/invites", "", 200, true), &invitations); err != nil {
		t.Fatal(err)
	}
	if len(invitations) != 1 || invitations[0].Uses != 0 {
		t.Fatal("existing members must not consume invite quota")
	}
	call("DELETE", "/workspaces/"+workspace.ID+"/invites/"+invitations[0].ID, "", 204, true)
	call("GET", "/invites/"+invite.Code, "", 404, false)

	// A second, isolated account exercises recipient access and membership removal.
	var secondUser string
	username := "dm_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	err = srv.db.RawPool().QueryRow(context.Background(), `INSERT INTO users(username,username_normalized) VALUES($1,$1) RETURNING id`, username).Scan(&secondUser)
	if err != nil {
		t.Fatal(err)
	}
	var directID string
	defer func() {
		if directID != "" {
			if _, err := srv.db.RawPool().Exec(context.Background(), `DELETE FROM conversations WHERE id=$1`, directID); err != nil {
				t.Errorf("cleanup conversation: %v", err)
			}
		}
		if _, err := srv.db.RawPool().Exec(context.Background(), `DELETE FROM users WHERE id=$1`, secondUser); err != nil {
			t.Errorf("cleanup second user: %v", err)
		}
	}()
	secondToken, _, _, err := srv.SessionService().CreateSession(context.Background(), secondUser, "integration test")
	if err != nil {
		t.Fatal(err)
	}
	directBody, _ := json.Marshal(map[string]string{"user_id": secondUser})
	var direct struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(call("POST", "/conversations/direct", string(directBody), 200, true), &direct); err != nil {
		t.Fatal(err)
	}
	directID = direct.ID
	directPath := "/conversations/" + direct.ID + "/messages"
	requestKey = uuid.NewString()
	directFirst := call("POST", directPath, `{"text":"Halo teman"}`, 201, true)
	if string(directFirst) != string(call("POST", directPath, `{"text":"Halo teman"}`, 201, true)) {
		t.Fatal("DM retry returned different message")
	}
	secondCall := func(method, path, body string, status int) json.RawMessage {
		t.Helper()
		req, _ := http.NewRequest(method, ts.URL+"/v1"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+secondToken.Token)
		if method == http.MethodPost {
			req.Header.Set("Idempotency-Key", uuid.NewString())
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		payload, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("second user %s %s: got %d want %d: %s", method, path, res.StatusCode, status, payload)
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &envelope)
		}
		return envelope.Data
	}
	var inbox []struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(secondCall("GET", "/conversations", "", 200), &inbox); err != nil {
		t.Fatal(err)
	}
	if len(inbox) != 1 || inbox[0].ID != directID || inbox[0].UserID != verifyResult.Data.User.ID {
		t.Fatalf("recipient inbox missing conversation: %+v", inbox)
	}
	var received []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(secondCall("GET", directPath, "", 200), &received); err != nil {
		t.Fatal(err)
	}
	if len(received) != 1 || received[0].Text != "Halo teman" {
		t.Fatalf("recipient did not receive DM: %+v", received)
	}
	secondCall("POST", directPath, `{"text":"Halo kembali"}`, 201)
	received = nil
	if err := json.Unmarshal(call("GET", directPath, "", 200, true), &received); err != nil {
		t.Fatal(err)
	}
	if len(received) != 2 || received[1].Text != "Halo kembali" {
		t.Fatalf("sender did not receive reply: %+v", received)
	}
	if _, err := srv.db.RawPool().Exec(context.Background(), `UPDATE conversation_members SET left_at=NOW() WHERE conversation_id=$1 AND user_id=$2`, directID, secondUser); err != nil {
		t.Fatal(err)
	}
	secondCall("GET", directPath, "", 404)
	secondCall("POST", directPath, `{"text":"no access"}`, 404)

	// 7. Logout (POST /v1/auth/logout)
	reqLogout, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/auth/logout", nil)
	reqLogout.Header.Set("X-CSRF-Token", csrfToken)
	reqLogout.AddCookie(sessionCookie)
	respLogout, err := client.Do(reqLogout)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	defer respLogout.Body.Close()

	if respLogout.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for logout, got %d", respLogout.StatusCode)
	}

	// 8. Verify session revoked on subsequent request
	reqMeAfterLogout, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/me", nil)
	reqMeAfterLogout.AddCookie(sessionCookie)
	respAfterLogout, err := client.Do(reqMeAfterLogout)
	if err != nil {
		t.Fatalf("request after logout failed: %v", err)
	}
	defer respAfterLogout.Body.Close()

	if respAfterLogout.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized after logout, got %d", respAfterLogout.StatusCode)
	}
}

func TestAPI_CSRFProtectionEnforcement(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	if !srv.db.IsConnected() {
		t.Skip("skipping integration test: database is not connected")
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	testEmail := "csrf-user-" + uuid.NewString() + "@example.com"
	defer cleanupTestAccount(t, srv, testEmail)

	// 1. Request and verify OTP to obtain session
	reqBody, _ := json.Marshal(map[string]string{"email": testEmail})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/auth/otp/challenges", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	var challengeEnvelope struct {
		Data struct {
			ChallengeID string `json:"challenge_id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&challengeEnvelope)
	resp.Body.Close()

	code := srv.EmailSender().GetLastCode(testEmail)
	verifyBody, _ := json.Marshal(map[string]string{
		"challenge_id": challengeEnvelope.Data.ChallengeID,
		"code":         code,
	})
	reqVerify, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/auth/otp/verifications", bytes.NewReader(verifyBody))
	reqVerify.Header.Set("Content-Type", "application/json")
	respVerify, _ := client.Do(reqVerify)
	var sessionCookie *http.Cookie
	for _, c := range respVerify.Cookies() {
		if strings.Contains(c.Name, "session") {
			sessionCookie = c
			break
		}
	}
	respVerify.Body.Close()
	if sessionCookie == nil {
		t.Fatalf("expected session cookie in Set-Cookie headers from respVerify (status %d)", respVerify.StatusCode)
	}

	// 2. Try mutating request WITHOUT X-CSRF-Token header -> must be rejected with 403 Forbidden!
	patchBody, _ := json.Marshal(map[string]string{"bio": "CSRF attack attempt"})
	reqCSRFAttack, _ := http.NewRequest(http.MethodPatch, ts.URL+"/v1/me/profile", bytes.NewReader(patchBody))
	reqCSRFAttack.Header.Set("Content-Type", "application/json")
	reqCSRFAttack.AddCookie(sessionCookie)
	// Notice: No X-CSRF-Token header!
	respAttack, err := client.Do(reqCSRFAttack)
	if err != nil {
		t.Fatalf("attack request failed: %v", err)
	}
	defer respAttack.Body.Close()

	if respAttack.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when X-CSRF-Token is omitted, got %d", respAttack.StatusCode)
	}

	var errResp response.ErrorEnvelope
	_ = json.NewDecoder(respAttack.Body).Decode(&errResp)
	if errResp.Error.Code != response.CodeForbidden {
		t.Errorf("expected error code FORBIDDEN, got %s", errResp.Error.Code)
	}
}

func TestAPI_InternalAPIKeyProtection(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Request without X-Internal-API-Key -> 401
	req1, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/internal/ping", nil)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing internal key, got %d", resp1.StatusCode)
	}

	// 2. Request with invalid key -> 401
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/internal/ping", nil)
	req2.Header.Set("X-Internal-API-Key", "wrong-secret-key")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid internal key, got %d", resp2.StatusCode)
	}

	// 3. Request with valid internal key -> 200
	validKey := srv.cfg.InternalAPIKeys[0]
	req3, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/internal/ping", nil)
	req3.Header.Set("X-Internal-API-Key", validKey)
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for valid internal key, got %d", resp3.StatusCode)
	}
}
