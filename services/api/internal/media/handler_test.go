package media

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/xynoss/relay/services/api/internal/authctx"
	"github.com/xynoss/relay/services/api/internal/storage"
)

type fixedProfileMediaAccess bool

func (a fixedProfileMediaAccess) IsProfileMedia(context.Context, string) (bool, error) {
	return bool(a), nil
}

func TestMediaUploadHTTPFlow(t *testing.T) {
	store := newMemoryMediaStore()
	provider := storage.NewMockProvider()
	routes := NewHandler(NewService(store, provider)).Routes(nil)
	imageBytes := validPNG(t)

	request := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req = req.WithContext(authctx.WithActor(req.Context(), &authctx.Actor{UserID: "user-1"}))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		return req
	}

	createBody, _ := json.Marshal(MediaIntentRequest{
		Purpose: "avatar", ContentType: "image/png", SizeBytes: int64(len(imageBytes)),
	})
	createRec := httptest.NewRecorder()
	routes.ServeHTTP(createRec, request(http.MethodPost, "/uploads", string(createBody)))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("upload intent status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	var uploadResponse struct {
		Data storage.UploadIntent `json:"data"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&uploadResponse); err != nil {
		t.Fatalf("decode upload intent: %v", err)
	}
	intent := uploadResponse.Data
	if _, err := provider.PutObject(t.Context(), intent.ObjectKey, bytes.NewReader(imageBytes), int64(len(imageBytes)), "image/png"); err != nil {
		t.Fatalf("store test upload: %v", err)
	}

	finalizeRec := httptest.NewRecorder()
	routes.ServeHTTP(finalizeRec, request(http.MethodPost, "/"+intent.MediaID+"/finalize", `{}`))
	if finalizeRec.Code != http.StatusOK {
		t.Fatalf("finalize status=%d body=%s", finalizeRec.Code, finalizeRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	routes.ServeHTTP(getRec, request(http.MethodGet, "/"+intent.MediaID, ""))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getRec.Code, getRec.Body.String())
	}

	deleteRec := httptest.NewRecorder()
	routes.ServeHTTP(deleteRec, request(http.MethodDelete, "/"+intent.MediaID, ""))
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestLocalContentUploadAndServeHTTPFlow(t *testing.T) {
	imageBytes := validPNG(t)
	localProvider, err := storage.NewLocalProvider(t.TempDir(), "http://api.test/v1/media/content", "test-session-signing-key")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(newMemoryMediaStore(), localProvider)
	intent, err := service.CreateUploadIntent(t.Context(), "user-1", storage.PurposeAvatar, "image/png", int64(len(imageBytes)), "")
	if err != nil {
		t.Fatalf("CreateUploadIntent: %v", err)
	}
	parsedUploadURL, err := url.Parse(intent.UploadURL)
	if err != nil {
		t.Fatal(err)
	}
	uploadReq := httptest.NewRequest(http.MethodPut, "/content/upload?"+parsedUploadURL.RawQuery, bytes.NewReader(imageBytes))
	uploadReq.Header.Set("Content-Type", "image/png")
	uploadRec := httptest.NewRecorder()
	NewHandler(service).Routes(nil).ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("local upload status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}

	contentRec := httptest.NewRecorder()
	contentReq := httptest.NewRequest(http.MethodGet, "/content/"+intent.ObjectKey, nil)
	NewHandler(service).Routes(nil).ServeHTTP(contentRec, contentReq)
	if contentRec.Code != http.StatusOK {
		t.Fatalf("content status=%d body=%s", contentRec.Code, contentRec.Body.String())
	}
	if contentRec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected content type %q", contentRec.Header().Get("Content-Type"))
	}
	readBack, _ := io.ReadAll(contentRec.Body)
	if !bytes.Equal(readBack, imageBytes) {
		t.Fatal("served local image does not match uploaded bytes")
	}
}

func TestPublicProfileMediaRequiresAttachedReadyAsset(t *testing.T) {
	imageBytes := validPNG(t)
	store := newMemoryMediaStore()
	provider := storage.NewMockProvider()
	service := NewService(store, provider)
	intent, err := service.CreateUploadIntent(t.Context(), "user-1", storage.PurposeAvatar, "image/png", int64(len(imageBytes)), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.PutObject(t.Context(), intent.ObjectKey, bytes.NewReader(imageBytes), int64(len(imageBytes)), "image/png"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FinalizeUpload(t.Context(), "user-1", intent.MediaID, ""); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/profile/"+intent.MediaID, nil)
	privateRecorder := httptest.NewRecorder()
	NewHandler(service, fixedProfileMediaAccess(false)).Routes(nil).ServeHTTP(privateRecorder, request)
	if privateRecorder.Code != http.StatusNotFound {
		t.Fatalf("unattached profile media status=%d, want 404", privateRecorder.Code)
	}

	publicRecorder := httptest.NewRecorder()
	NewHandler(service, fixedProfileMediaAccess(true)).Routes(nil).ServeHTTP(publicRecorder, request)
	if publicRecorder.Code != http.StatusOK {
		t.Fatalf("attached profile media status=%d body=%s", publicRecorder.Code, publicRecorder.Body.String())
	}
	if publicRecorder.Header().Get("Content-Type") != "image/png" || publicRecorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unsafe or incorrect headers: %#v", publicRecorder.Header())
	}
	if !bytes.Equal(publicRecorder.Body.Bytes(), imageBytes) {
		t.Fatal("public profile media did not match the validated image")
	}
}
