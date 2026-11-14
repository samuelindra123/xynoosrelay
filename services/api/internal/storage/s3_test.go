package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestS3PrivateObjectRequestsAreSigned(t *testing.T) {
	methods := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" || r.Header.Get("x-amz-date") == "" || r.Header.Get("x-amz-content-sha256") == "" {
			t.Errorf("%s request is missing SigV4 headers", r.Method)
		}
		if r.URL.Path != "/private-media/messages/file.txt" {
			t.Errorf("unexpected object path: %s", r.URL.Path)
		}
		methods <- r.Method
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "hello")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider, err := NewS3Provider(S3Config{
		Endpoint: server.URL, Bucket: "private-media", Region: "us-east-2",
		AccessKeyID: "test-access", SecretAccessKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	body, contentType, err := provider.GetObject(context.Background(), "messages/file.txt")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer body.Close()
	readBody, _ := io.ReadAll(body)
	if string(readBody) != "hello" || contentType != "text/plain" {
		t.Fatalf("unexpected object response: body=%q type=%q", readBody, contentType)
	}
	if err := provider.DeleteObject(context.Background(), "messages/file.txt"); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if got := <-methods; got != http.MethodGet {
		t.Fatalf("first request method = %q", got)
	}
	if got := <-methods; got != http.MethodDelete {
		t.Fatalf("second request method = %q", got)
	}
}

func TestNewS3ProviderRejectsInvalidEndpoint(t *testing.T) {
	_, err := NewS3Provider(S3Config{Endpoint: "file:///tmp", Bucket: "media", Region: "us-east-2"})
	if err == nil || !strings.Contains(err.Error(), "absolute HTTP or HTTPS URL") {
		t.Fatalf("expected invalid endpoint error, got %v", err)
	}
}
