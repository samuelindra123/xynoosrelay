package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/session"
)

type failingLogoutStore struct{ *mockSessionStore }

func (s *failingLogoutStore) RevokeSession(context.Context, string) error {
	return errors.New("database unavailable")
}

func TestLogoutReportsNativeRevocationFailure(t *testing.T) {
	store := &failingLogoutStore{newMockSessionStore()}
	store.sessions["session"] = &session.Session{ID: "session", UserID: "user"}
	h := NewHandler(nil, session.NewService(store, "test-key"), nil, false, "")
	r := httptest.NewRequest(http.MethodPost, "/logout", nil)
	r = r.WithContext(authctx.WithActor(r.Context(), &authctx.Actor{UserID: "user", SessionID: "session"}))
	w := httptest.NewRecorder()
	h.Logout(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", w.Code)
	}
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("cookie must not be cleared when native revocation fails")
	}
}

func TestLogoutDoesNotTreatManagedSessionAsNative(t *testing.T) {
	store := &failingLogoutStore{newMockSessionStore()}
	h := NewHandler(nil, session.NewService(store, "test-key"), nil, false, "")
	r := httptest.NewRequest(http.MethodPost, "/logout", nil)
	r = r.WithContext(authctx.WithActor(r.Context(), &authctx.Actor{UserID: "user", SessionID: "managed-session", ManagedSession: true}))
	w := httptest.NewRecorder()
	h.Logout(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("got %d, want 204", w.Code)
	}
}
