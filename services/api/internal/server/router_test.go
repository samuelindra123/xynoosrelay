package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
