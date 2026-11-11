package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/logger"
	"github.com/xynoss/relay/services/api/internal/response"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := logger.GetRequestID(r.Context())
		if reqID == "" {
			t.Errorf("expected request ID in context, got empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: no incoming header -> generates new ID
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	respID := rec.Header().Get("X-Request-ID")
	if respID == "" {
		t.Errorf("expected X-Request-ID header in response")
	}

	// Case 2: valid incoming header -> reuses it
	customID := "custom-req-id-12345"
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Request-ID", customID)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Request-ID") != customID {
		t.Errorf("expected reused request ID %s, got %s", customID, rec2.Header().Get("X-Request-ID"))
	}
}

func TestCORSMiddleware(t *testing.T) {
	cors := CORS([]string{"http://localhost:3000", "https://app.xynoss.relay"})
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Allowed Origin
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected allow origin http://localhost:3000, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected allow credentials true")
	}

	// Case 2: Disallowed Origin
	reqDisallowed := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqDisallowed.Header.Set("Origin", "http://evil.com")
	recDisallowed := httptest.NewRecorder()
	handler.ServeHTTP(recDisallowed, reqDisallowed)

	if recDisallowed.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no allow origin for evil.com, got %s", recDisallowed.Header().Get("Access-Control-Allow-Origin"))
	}

	// Case 3: OPTIONS Preflight
	reqOptions := httptest.NewRequest(http.MethodOptions, "/test", nil)
	reqOptions.Header.Set("Origin", "http://localhost:3000")
	recOptions := httptest.NewRecorder()
	handler.ServeHTTP(recOptions, reqOptions)

	if recOptions.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for options preflight, got %d", recOptions.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	sec := SecurityHeaders(true)
	dummyHandler := sec(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: GET request sets security headers
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	dummyHandler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff header")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("missing DENY header")
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Errorf("expected HSTS in production")
	}

	// Case 2: POST with non-JSON content-type rejected
	reqPost := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("text data"))
	reqPost.Header.Set("Content-Type", "text/plain")
	recPost := httptest.NewRecorder()
	dummyHandler.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 Unsupported Media Type, got %d", recPost.Code)
	}

	// Case 3: POST with application/json accepted
	reqPostJSON := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"key":"value"}`))
	reqPostJSON.Header.Set("Content-Type", "application/json")
	recPostJSON := httptest.NewRecorder()
	dummyHandler.ServeHTTP(recPostJSON, reqPostJSON)

	if recPostJSON.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid json body, got %d", recPostJSON.Code)
	}
}

func TestRecoverer(t *testing.T) {
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recov := Recoverer(nullLogger)

	panicHandler := recov(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected critical failure")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	panicHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 status code, got %d", rec.Code)
	}

	var errResp response.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if errResp.Error.Code != response.CodeInternalError {
		t.Errorf("expected INTERNAL_ERROR code, got %s", errResp.Error.Code)
	}
}

func TestRequireAuth(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Unauthenticated request -> 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)

	if recUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", recUnauth.Code)
	}

	// Case 2: Authenticated request with Actor -> 200
	reqAuth := httptest.NewRequest(http.MethodGet, "/protected", nil)
	ctx := authctx.WithActor(reqAuth.Context(), &authctx.Actor{UserID: "user-123", SessionID: "sess-456"})
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth.WithContext(ctx))

	if recAuth.Code != http.StatusOK {
		t.Errorf("expected 200 OK with actor, got %d", recAuth.Code)
	}
}

func TestRequireInternalAPIKey(t *testing.T) {
	keys := []string{"active-key-v1", "active-key-v2"}
	middleware := RequireInternalAPIKey(keys)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Missing key -> 401
	reqMissing := httptest.NewRequest(http.MethodGet, "/internal/data", nil)
	recMissing := httptest.NewRecorder()
	handler.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing key, got %d", recMissing.Code)
	}

	// Case 2: Invalid key -> 401
	reqInvalid := httptest.NewRequest(http.MethodGet, "/internal/data", nil)
	reqInvalid.Header.Set(HeaderInternalAPIKey, "wrong-key")
	recInvalid := httptest.NewRecorder()
	handler.ServeHTTP(recInvalid, reqInvalid)
	if recInvalid.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid key, got %d", recInvalid.Code)
	}

	// Case 3: Valid key-v1 -> 200
	reqV1 := httptest.NewRequest(http.MethodGet, "/internal/data", nil)
	reqV1.Header.Set(HeaderInternalAPIKey, "active-key-v1")
	recV1 := httptest.NewRecorder()
	handler.ServeHTTP(recV1, reqV1)
	if recV1.Code != http.StatusOK {
		t.Errorf("expected 200 for valid key-v1, got %d", recV1.Code)
	}

	// Case 4: Valid key-v2 (Key rotation) -> 200
	reqV2 := httptest.NewRequest(http.MethodGet, "/internal/data", nil)
	reqV2.Header.Set(HeaderInternalAPIKey, "active-key-v2")
	recV2 := httptest.NewRecorder()
	handler.ServeHTTP(recV2, reqV2)
	if recV2.Code != http.StatusOK {
		t.Errorf("expected 200 for rotated key-v2, got %d", recV2.Code)
	}
}
