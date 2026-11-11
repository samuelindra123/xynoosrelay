package workspaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xynoss/relay/services/api/internal/authctx"
)

// Requests traverse the real Chi routing, JSON decoder and service boundary.
// A nil store deliberately ensures invalid requests cannot reach persistence.
func TestInviteRequestValidation(t *testing.T) {
	h := NewHandler(NewService(nil, "http://localhost:3000"))
	for _, body := range []string{`{`, `{"duration_hours":-1}`, `{"duration_hours":721}`, `{"duration_hours":999999999999}`, `{"max_uses":-1}`, `{"max_uses":10001}`, `{"unknown":true}`, `{} {}`} {
		t.Run(body, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/workspace/invites", strings.NewReader(body))
			req = req.WithContext(authctx.WithActor(context.Background(), &authctx.Actor{UserID: "owner"}))
			result := httptest.NewRecorder()
			h.WorkspaceRoutes().ServeHTTP(result, req)
			if result.Code != 400 && result.Code != 422 {
				t.Fatalf("status=%d body=%s", result.Code, result.Body.String())
			}
		})
	}
}

func TestWorkspaceCreationValidation(t *testing.T) {
	h := NewHandler(NewService(nil, "http://localhost:3000"))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"   "}`))
	req = req.WithContext(authctx.WithActor(context.Background(), &authctx.Actor{UserID: "owner"}))
	result := httptest.NewRecorder()
	h.WorkspaceRoutes().ServeHTTP(result, req)
	if result.Code != 422 {
		t.Fatalf("expected validation error, got %d", result.Code)
	}
}

func TestUnauthenticatedInviteCreation(t *testing.T) {
	h := NewHandler(NewService(nil, "http://localhost:3000"))
	result := httptest.NewRecorder()
	h.WorkspaceRoutes().ServeHTTP(result, httptest.NewRequest(http.MethodPost, "/workspace/invites", strings.NewReader(`{}`)))
	if result.Code != 401 {
		t.Fatalf("expected 401, got %d", result.Code)
	}
}
