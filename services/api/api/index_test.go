package handler

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRestoreRewrittenPath(t *testing.T) {
	request := httptest.NewRequest("GET", "/api?__relay_path=%2Fv1%2Fauth%2Fgoogle%2Fstart&next=hello", nil)

	if !restoreRewrittenPath(request) {
		t.Fatal("expected the Vercel rewrite path to be restored")
	}
	if request.URL.Path != "/v1/auth/google/start" {
		t.Fatalf("restored path = %q", request.URL.Path)
	}
	if got := request.URL.Query().Get("__relay_path"); got != "" {
		t.Fatalf("rewrite parameter leaked to API query: %q", got)
	}
	if got := request.URL.Query().Get("next"); got != "hello" {
		t.Fatalf("original query was not preserved: %q", got)
	}
}

func TestRestoreRewrittenPathRejectsNonAPIPath(t *testing.T) {
	request := httptest.NewRequest("GET", "/api?__relay_path=%2Fadmin%2Fsecret", nil)
	originalQuery := request.URL.RawQuery

	if restoreRewrittenPath(request) {
		t.Fatal("unexpectedly restored a path outside /v1")
	}
	if request.URL.Path != "/api" || request.URL.RawQuery != originalQuery {
		t.Fatalf("request changed after rejecting path: %s?%s", request.URL.Path, request.URL.RawQuery)
	}
}

func TestRestoreRewrittenPathKeepsRepeatedQueryParameters(t *testing.T) {
	request := httptest.NewRequest("GET", "/api?__relay_path=%2Fv1%2Fchannels&cursor=older&cursor=oldest", nil)
	restoreRewrittenPath(request)

	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	if got := query["cursor"]; len(got) != 2 || got[0] != "older" || got[1] != "oldest" {
		t.Fatalf("repeated query parameters not preserved: %#v", got)
	}
}
