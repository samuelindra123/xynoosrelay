package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xynoss/relay/services/api/internal/config"
	"github.com/xynoss/relay/services/api/internal/response"
)

type mockDB struct {
	connected bool
	pingErr   error
}

func (m *mockDB) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockDB) Close() {}

func (m *mockDB) IsConnected() bool {
	return m.connected
}

func (m *mockDB) RawPool() *pgxpool.Pool {
	return nil
}

func TestHealthHandler_Live(t *testing.T) {
	h := NewHandler(nil, config.EnvDevelopment)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp response.SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "ok" {
		t.Errorf("unexpected status value: %v", resp.Data)
	}
}

func TestHealthHandler_Ready_Success(t *testing.T) {
	db := &mockDB{connected: true, pingErr: nil}
	h := NewHandler(db, config.EnvProduction)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp response.SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", resp.Data)
	}
}

func TestHealthHandler_Ready_DBDown(t *testing.T) {
	db := &mockDB{connected: true, pingErr: errors.New("timeout connecting to neon")}
	h := NewHandler(db, config.EnvProduction)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var resp response.SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "not_ready" {
		t.Errorf("expected status 'not_ready', got %v", resp.Data)
	}
}
