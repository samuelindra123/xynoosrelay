package messaging

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xynoss/relay/services/api/internal/authctx"
)

func TestChannelMessageAccessRequiresSessionAndValidChannelID(t *testing.T) {
	h := NewHandler(nil).Routes()
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method+"/unauthenticated", func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, "/not-a-uuid/messages", nil))
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("got %d, want 401", w.Code)
			}
		})
		t.Run(method+"/invalid-id", func(t *testing.T) {
			r := httptest.NewRequest(method, "/not-a-uuid/messages", nil)
			r = r.WithContext(authctx.WithActor(r.Context(), &authctx.Actor{UserID: "test-user"}))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", w.Code)
			}
		})
		t.Run(method+"/database-unavailable", func(t *testing.T) {
			r := httptest.NewRequest(method, "/00000000-0000-0000-0000-000000000001/messages", nil)
			r = r.WithContext(authctx.WithActor(r.Context(), &authctx.Actor{UserID: "test-user"}))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("got %d, want 503", w.Code)
			}
		})
	}
}
