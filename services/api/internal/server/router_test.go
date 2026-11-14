package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/response"
)

func TestServer_HealthAndRouting(t *testing.T) {
	cfg := &config.Config{
		AppEnv:         config.EnvTest,
		Port:           "8080",
		WebOrigins:     []string{"http://localhost:3000"},
		RequestTimeout: 5,
	}
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, nullLogger, nil)

	// Test 1: GET /health/live
	req1 := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec1 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("expected status 200 for /health/live, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-Request-ID") == "" {
		t.Errorf("expected X-Request-ID header to be present")
	}

	// Test 2: GET /v1/health
	req2 := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status 200 for /v1/health, got %d", rec2.Code)
	}

	// Test 3: GET /v1/nonexistent (Standard 404 envelope)
	req3 := httptest.NewRequest(http.MethodGet, "/v1/nonexistent", nil)
	rec3 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for nonexistent route, got %d", rec3.Code)
	}

	var errResp response.ErrorEnvelope
	if err := json.NewDecoder(rec3.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}

	if errResp.Error.Code != response.CodeNotFound {
		t.Errorf("expected code NOT_FOUND, got %s", errResp.Error.Code)
	}
	if errResp.Error.RequestID == "" {
		t.Errorf("expected non-empty request_id in error body")
	}
}

func TestServer_EndpointRouteMatrix(t *testing.T) {
	cfg := &config.Config{
		AppEnv:         config.EnvTest,
		Port:           "8080",
		FrontendURL:    "http://localhost:3000",
		APIBaseURL:     "http://localhost:8080",
		WebOrigins:     []string{"http://localhost:3000"},
		RequestTimeout: 5,
		SessionHashKey: "test-session-key-that-is-long-enough-for-local-tests",
	}
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, nullLogger, nil)

	protectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/auth/session"},
		{http.MethodPost, "/v1/auth/logout"},
		{http.MethodPost, "/v1/media/uploads"},
		{http.MethodPost, "/v1/media/00000000-0000-0000-0000-000000000001/finalize"},
		{http.MethodGet, "/v1/media/00000000-0000-0000-0000-000000000001"},
		{http.MethodDelete, "/v1/media/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/v1/me"},
		{http.MethodGet, "/v1/me/profile"},
		{http.MethodPatch, "/v1/me/profile"},
		{http.MethodPost, "/v1/me/onboarding/complete"},
		{http.MethodGet, "/v1/me/sessions"},
		{http.MethodDelete, "/v1/me/sessions/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/v1/users/search"},
		{http.MethodPost, "/v1/users/me/avatar"},
		{http.MethodDelete, "/v1/users/me/avatar"},
		{http.MethodPost, "/v1/users/me/banner"},
		{http.MethodDelete, "/v1/users/me/banner"},
		{http.MethodGet, "/v1/workspaces"},
		{http.MethodPost, "/v1/workspaces"},
		{http.MethodGet, "/v1/workspaces/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/v1/workspaces/00000000-0000-0000-0000-000000000001/channels"},
		{http.MethodGet, "/v1/workspaces/00000000-0000-0000-0000-000000000001/invites"},
		{http.MethodPost, "/v1/workspaces/00000000-0000-0000-0000-000000000001/invites"},
		{http.MethodDelete, "/v1/workspaces/00000000-0000-0000-0000-000000000001/invites/00000000-0000-0000-0000-000000000002"},
		{http.MethodPost, "/v1/invites/opaque-code/accept"},
		{http.MethodGet, "/v1/channels/00000000-0000-0000-0000-000000000001/read"},
		{http.MethodPost, "/v1/channels/00000000-0000-0000-0000-000000000001/read"},
		{http.MethodGet, "/v1/channels/00000000-0000-0000-0000-000000000001/access"},
		{http.MethodPut, "/v1/channels/00000000-0000-0000-0000-000000000001/access"},
		{http.MethodDelete, "/v1/channels/00000000-0000-0000-0000-000000000001/access/00000000-0000-0000-0000-000000000002"},
		{http.MethodGet, "/v1/channels/00000000-0000-0000-0000-000000000001/messages"},
		{http.MethodPost, "/v1/channels/00000000-0000-0000-0000-000000000001/messages"},
		{http.MethodPatch, "/v1/channels/00000000-0000-0000-0000-000000000001/messages/1"},
		{http.MethodDelete, "/v1/channels/00000000-0000-0000-0000-000000000001/messages/1"},
		{http.MethodPut, "/v1/channels/00000000-0000-0000-0000-000000000001/messages/1/reactions/heart"},
		{http.MethodDelete, "/v1/channels/00000000-0000-0000-0000-000000000001/messages/1/reactions/heart"},
		{http.MethodGet, "/v1/conversations"},
		{http.MethodPost, "/v1/conversations/direct"},
		{http.MethodGet, "/v1/conversations/00000000-0000-0000-0000-000000000001/read"},
		{http.MethodPost, "/v1/conversations/00000000-0000-0000-0000-000000000001/read"},
		{http.MethodGet, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages"},
		{http.MethodPost, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages"},
		{http.MethodPatch, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages/1"},
		{http.MethodDelete, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages/1"},
		{http.MethodPut, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages/1/reactions/heart"},
		{http.MethodDelete, "/v1/conversations/00000000-0000-0000-0000-000000000001/messages/1/reactions/heart"},
		{http.MethodGet, "/v1/internal/ping"},
	}

	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected registered protected route to reject unauthenticated request with 401, got %d", rec.Code)
			}
		})
	}

	publicRoutes := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/health", "", http.StatusOK},
		{http.MethodGet, "/health/live", "", http.StatusOK},
		{http.MethodGet, "/health/ready", "", http.StatusOK},
		{http.MethodGet, "/v1/health", "", http.StatusOK},
		{http.MethodGet, "/v1/health/live", "", http.StatusOK},
		{http.MethodGet, "/v1/health/ready", "", http.StatusOK},
		{http.MethodPost, "/v1/auth/otp/challenges", "{", http.StatusBadRequest},
		{http.MethodPost, "/v1/auth/otp/verifications", "{", http.StatusBadRequest},
		{http.MethodGet, "/v1/auth/google/start", "", http.StatusBadRequest},
		{http.MethodGet, "/v1/auth/google/callback", "", http.StatusFound},
		{http.MethodGet, "/v1/invites/invalid-code", "", http.StatusNotFound},
		{http.MethodPut, "/v1/media/content/upload", "", http.StatusBadRequest},
		{http.MethodGet, "/v1/media/content/not-found", "", http.StatusNotFound},
	}

	for _, route := range publicRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
			if route.method == http.MethodPost || route.method == http.MethodPut || route.method == http.MethodPatch {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)
			if rec.Code != route.status {
				t.Fatalf("expected status %d, got %d", route.status, rec.Code)
			}
		})
	}
}
