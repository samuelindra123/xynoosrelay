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

func TestMessageActionRoutesRequireAuthorizationAndDatabase(t *testing.T) {
	routes := NewHandler(nil)
	channel := routes.Routes()
	direct := routes.DirectRoutes()
	for _, tc := range []struct {
		router http.Handler
		method string
		path   string
	}{{channel, http.MethodPatch, "/bad/messages/1"}, {channel, http.MethodDelete, "/bad/messages/1"}, {channel, http.MethodPut, "/bad/messages/1/reactions/heart"}, {direct, http.MethodPatch, "/bad/messages/1"}, {direct, http.MethodDelete, "/bad/messages/1"}, {direct, http.MethodPut, "/bad/messages/1/reactions/heart"}} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		tc.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s unauthenticated: got %d, want 401", tc.method, tc.path, w.Code)
		}
		req = httptest.NewRequest(tc.method, tc.path, nil)
		req = req.WithContext(authctx.WithActor(req.Context(), &authctx.Actor{UserID: "test-user"}))
		w = httptest.NewRecorder()
		tc.router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s %s invalid id: got %d, want 400", tc.method, tc.path, w.Code)
		}
	}
}

func TestPrivateChannelAccessRoutesRequireOwnerSessionAndDatabase(t *testing.T) {
	routes := NewHandler(nil).Routes()
	channelID := "00000000-0000-0000-0000-000000000001"
	for _, tc := range []struct {
		method string
		path   string
	}{{http.MethodGet, "/" + channelID + "/access"}, {http.MethodPut, "/" + channelID + "/access"}, {http.MethodDelete, "/" + channelID + "/access/00000000-0000-0000-0000-000000000002"}} {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			routes.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated status = %d, want 401", w.Code)
			}

			req = httptest.NewRequest(tc.method, tc.path, nil)
			req = req.WithContext(authctx.WithActor(req.Context(), &authctx.Actor{UserID: "00000000-0000-0000-0000-000000000003"}))
			w = httptest.NewRecorder()
			routes.ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("database unavailable status = %d, want 503", w.Code)
			}
		})
	}
}
