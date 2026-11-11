package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xynoss/relay/services/api/internal/logger"
)

func TestJSON_SuccessEnvelope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(logger.WithRequestID(req.Context(), "req-test-123"))
	rec := httptest.NewRecorder()

	data := map[string]string{"status": "ok"}
	JSON(rec, req, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("unexpected content type: %s", contentType)
	}

	var resp SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RequestID != "req-test-123" {
		t.Errorf("expected request_id req-test-123, got %s", resp.RequestID)
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "ok" {
		t.Errorf("unexpected data content: %v", resp.Data)
	}
}

func TestRenderError_StandardErrorEnvelope(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req = req.WithContext(logger.WithRequestID(req.Context(), "req-err-456"))
	rec := httptest.NewRecorder()

	validationErr := NewValidationError("validation failed", []ErrorDetail{
		{Field: "email", Code: "INVALID_FORMAT"},
	})

	RenderError(rec, req, validationErr)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}

	var resp ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}

	if resp.Error.Code != CodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %s", resp.Error.Code)
	}
	if resp.Error.Message != "validation failed" {
		t.Errorf("unexpected message: %s", resp.Error.Message)
	}
	if resp.Error.RequestID != "req-err-456" {
		t.Errorf("expected request_id req-err-456, got %s", resp.Error.RequestID)
	}
	if len(resp.Error.Details) != 1 || resp.Error.Details[0].Field != "email" {
		t.Errorf("unexpected details: %v", resp.Error.Details)
	}
}

func TestRenderError_GenericInternalError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	rawErr := errors.New("database connection failed")
	RenderError(rec, req, rawErr)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	var resp ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}

	// Important security rule: don't leak raw driver/DB errors to client
	if resp.Error.Message == "database connection failed" {
		t.Errorf("raw database error was leaked to client: %s", resp.Error.Message)
	}
	if resp.Error.Code != CodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %s", resp.Error.Code)
	}
}
