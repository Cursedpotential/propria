package runtimeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/sourcepreview"
)

func rawPreviewFixture(t *testing.T) (*SourceReadPreviewHandler, string) {
	t.Helper()
	id := strings.Repeat("a", 43)
	guard := &PreviewHTTPHandler{serviceTokenPath: serviceTokenPath(t)}
	h := &SourceReadPreviewHandler{auth: guard.auth, entries: map[string]*rawPreviewEntry{}, cursors: map[string]string{}}
	h.entries[id] = &rawPreviewEntry{task: sourcepreview.Task{Request: sourcepreview.Request{SourceRef: "b2://synthetic/messages.xml", ExpectedETag: "etag", VersionID: "v1", Limit: 25}}, actor: "authentik-user-1", created: time.Now(), done: make(chan struct{})}
	return h, id
}
func TestRawPreviewCallbackPinsReplayAndActor(t *testing.T) {
	h, id := rawPreviewFixture(t)
	result := sourcepreview.Completion{Page: sourcepreview.Page{PageID: id, Status: "ready", PreviewOnly: true, Source: sourcepreview.Source{SourceRef: "b2://synthetic/messages.xml", ETag: "etag", VersionID: "v1", ByteLength: 100}, Messages: []sourcepreview.Message{}, ScannedBytes: 50, TruncationReason: "scan_byte_limit"}, State: sourcepreview.State{Offset: 50, SourceLength: 100}}
	result.Page.Source.ETag = "changed"
	body, _ := json.Marshal(result)
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview/"+id+"/result", body); got.Code != 422 {
		t.Fatalf("pin mismatch status %d", got.Code)
	}
	result.Page.Source.ETag = "etag"
	body, _ = json.Marshal(result)
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview/"+id+"/result", body); got.Code != 200 {
		t.Fatalf("callback status %d: %s", got.Code, got.Body)
	}
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview/"+id+"/result", body); got.Code != 409 {
		t.Fatal("replay accepted")
	}
	if got := servePreview(h.Routes(), http.MethodGet, "/reference-import/source-preview/"+id, nil); got.Code != 202 {
		t.Fatal("page exposed before workflow receipt")
	}
	h.entries[id].finished = true
	if got := servePreview(h.Routes(), http.MethodGet, "/reference-import/source-preview/"+id, nil); got.Code != 200 {
		t.Fatal("ready page missing")
	}
	h.entries[id].actor = "another-actor"
	if got := servePreview(h.Routes(), http.MethodGet, "/reference-import/source-preview/"+id, nil); got.Code != 410 {
		t.Fatal("cross actor page exposed")
	}
}
func TestRawPreviewAuthenticationExpiryAndAggregateCapacity(t *testing.T) {
	h, id := rawPreviewFixture(t)
	request := httptest.NewRequest(http.MethodGet, "/reference-import/source-preview/"+id, nil)
	request.RemoteAddr = "100.64.1.9:1234"
	recorder := httptest.NewRecorder()
	h.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatal("unauthenticated access")
	}
	h.entries[id].created = time.Now().Add(-previewCursorTTL - time.Second)
	if got := servePreview(h.Routes(), http.MethodGet, "/reference-import/source-preview/"+id, nil); got.Code != 410 {
		t.Fatal("expired cache exposed")
	}
	for i := 0; i < previewCacheBudget/previewCacheReservation; i++ {
		h.entries[fmt.Sprint(i)] = &rawPreviewEntry{created: time.Now()}
	}
	if h.reserveLocked() {
		t.Fatal("pending aggregate capacity exceeded")
	}
	h.entries["0"].finished = true
	if !h.reserveLocked() {
		t.Fatal("ready entry was not evicted")
	}
}
func TestRawPreviewCallbackRejectsOversizeBodyAndState(t *testing.T) {
	h, id := rawPreviewFixture(t)
	oversize := []byte(`{"page":{"body":"` + strings.Repeat("x", sourcepreview.MaxPayloadBytes) + `"}}`)
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview/"+id+"/result", oversize); got.Code != 400 {
		t.Fatal("oversize callback accepted")
	}
}

func TestRawPreviewCursorPinMismatchAndExpiry(t *testing.T) {
	t.Setenv("SOURCE_ROOTS_JSON", `[{"id":"synthetic","label":"Synthetic","url":"b2://synthetic/"}]`)
	h, id := rawPreviewFixture(t)
	h.entries[id].finished = true
	h.entries[id].cursor = "opaque-cursor"
	h.cursors["opaque-cursor"] = id
	request := sourcepreview.Request{SourceRef: "b2://synthetic/messages.xml", ExpectedETag: "changed", VersionID: "v1", Cursor: "opaque-cursor", Limit: 25}
	body, _ := json.Marshal(request)
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview", body); got.Code != 409 {
		t.Fatalf("cursor pin mismatch status %d", got.Code)
	}
	request.ExpectedETag = "etag"
	request.Cursor = "expired-cursor"
	body, _ = json.Marshal(request)
	if got := servePreview(h.Routes(), http.MethodPost, "/reference-import/source-preview", body); got.Code != 410 {
		t.Fatal("unknown/expired cursor accepted")
	}
}
