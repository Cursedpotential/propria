// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

func originalFixture(t *testing.T) (*Service, *memoryStorage, *memoryBackend, string) {
	t.Helper()
	s, store, back := fixtureService(t)
	key := fixtureScope.LegalRoot + "benchbooks/Original fixture.pdf"
	id, _ := fixtureScope.BindingID(key)
	raw := []byte("%PDF-1.7\nSynthetic private original bytes.\n")
	obj := Object{Bucket: fixtureScope.Bucket, Key: key, VersionID: "retained-old-version", ContentType: "application/pdf", UploadedAt: fixtureNow}
	store.add(raw, obj)
	back.binding = Binding{ID: id, Provider: "b2", Scope: fixtureScope, Key: key, RecordID: "source:fixture", OriginalPointer: &Pointer{VersionID: obj.VersionID, SHA256: digest(raw), Size: int64(len(raw)), ContentType: "application/pdf", ObservedAt: fixtureNow}}
	return s, store, back, id
}

func TestOriginalHandlerAuthenticatesAndStreamsExactRetainedPrivatePDF(t *testing.T) {
	s, store, _, id := originalFixture(t)
	reader, err := NewOriginalReader(s.Scope, s.Storage, s.Backend)
	requireNoError(t, err)
	// The starter reader has no extractor, artifact store or signing key.
	store.head = "other-latest-version"
	handler, err := NewOriginalHandler(reader, func(r *http.Request) bool { return r.Header.Get("Authorization") == "Bearer synthetic-service-token" })
	requireNoError(t, err)
	route := "/toolkit/library/files/" + id + "/original?version_id=retained-old-version"
	r := httptest.NewRequest(http.MethodGet, route, nil)
	r.Header.Set("Authorization", "Bearer synthetic-service-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), store.versions[0].Raw) {
		t.Fatalf("original response %d %q", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(w.Header().Get("Content-Disposition"), "Original fixture.pdf") {
		t.Fatal("private response headers absent")
	}
	if strings.Contains(w.Body.String(), "Bearer") || strings.Contains(w.Body.String(), "access_key") {
		t.Fatal("credentials exposed")
	}
}

func TestOriginalHandlerRejectsAuthReferenceIntegrityAndPDFMismatch(t *testing.T) {
	for _, kind := range []string{"auth", "version", "binding", "provider", "scope", "hash", "size", "overbudget", "signature", "not-pdf", "method", "duplicate-query", "extra-query"} {
		t.Run(kind, func(t *testing.T) {
			s, store, back, id := originalFixture(t)
			authorized := true
			method := http.MethodGet
			query := "version_id=retained-old-version"
			expected := http.StatusBadGateway
			switch kind {
			case "auth":
				authorized = false
				expected = http.StatusUnauthorized
			case "version":
				query = "version_id=unapproved-version"
			case "binding":
				back.binding.ID = "library_file:" + digest([]byte("other"))
			case "provider":
				back.binding.Provider = "r2"
			case "scope":
				back.binding.Scope.Bucket = "different"
			case "hash":
				back.binding.OriginalPointer.SHA256 = digest([]byte("different"))
			case "size":
				back.binding.OriginalPointer.Size++
			case "overbudget":
				back.binding.OriginalPointer.Size = MaxOriginalBytes + 1
			case "signature":
				store.versions[0].Raw = []byte("NOT A PDF")
				store.versions[0].Object.Size = int64(len(store.versions[0].Raw))
				back.binding.OriginalPointer.Size = store.versions[0].Object.Size
				back.binding.OriginalPointer.SHA256 = digest(store.versions[0].Raw)
			case "not-pdf":
				back.binding.OriginalPointer.ContentType = "text/html"
			case "method":
				method = http.MethodPost
				expected = http.StatusMethodNotAllowed
			case "duplicate-query":
				query += "&version_id=latest"
				expected = http.StatusBadRequest
			case "extra-query":
				query += "&bucket=unauthorized"
				expected = http.StatusBadRequest
			}
			h, err := NewOriginalHandler(s, func(*http.Request) bool { return authorized })
			requireNoError(t, err)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, "/toolkit/library/files/"+id+"/original?"+query, nil))
			if w.Code != expected {
				t.Fatalf("%s got %d expected %d", kind, w.Code, expected)
			}
			if bytes.Contains(w.Body.Bytes(), store.versions[0].Raw) {
				t.Fatal("failed original bytes leaked")
			}
		})
	}
}

func TestOriginalPDFBoundaryAndDownloadConcurrency(t *testing.T) {
	s, store, back, id := originalFixture(t)
	raw := make([]byte, MaxOriginalBytes)
	copy(raw, []byte("%PDF-1.7\n"))
	store.versions[0].Raw = raw
	store.versions[0].Object.Size = int64(len(raw))
	back.binding.OriginalPointer.Size = int64(len(raw))
	back.binding.OriginalPointer.SHA256 = digest(raw)
	out, err := s.ReadOriginal(context.Background(), id, "retained-old-version")
	requireNoError(t, err)
	if len(out.Bytes) != len(raw) {
		t.Fatal("boundary PDF truncated")
	}
	h, err := NewOriginalHandler(s, func(*http.Request) bool { return true })
	requireNoError(t, err)
	h.slots <- struct{}{}
	h.slots <- struct{}{}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/toolkit/library/files/"+url.PathEscape(id)+"/original?version_id=retained-old-version", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("unbounded download concurrency")
	}
	if _, err = NewOriginalHandler(s, nil); err == nil {
		t.Fatal("unauthenticated original handler constructed")
	}
}

func TestScopeAndBindingIdentityCannotEscapeLegalPrefix(t *testing.T) {
	for _, key := range []string{"KnowledgeBase/legal/case-law-extra/x.pdf", "KnowledgeBase/legal/case-law/../private.pdf", "/KnowledgeBase/legal/benchbooks/x.pdf", "KnowledgeBase/legal/reference-data//x.md", "KnowledgeBase/legal/reference-data/", "KnowledgeBase/legal/other/x.pdf", "KnowledgeBase/legal/case-law/a\\b.pdf"} {
		if fixtureScope.Admit(fixtureScope.Bucket, key) == nil {
			t.Fatalf("invalid scope admitted %q", key)
		}
	}
	if fixtureScope.Admit("r2-other", fixtureScope.LegalRoot+"case-law/a.pdf") == nil {
		t.Fatal("different store bucket admitted")
	}
	id, err := fixtureScope.BindingID(fixtureScope.LegalRoot + "case-law/α<&>.pdf")
	requireNoError(t, err)
	again, err := fixtureScope.BindingID(fixtureScope.LegalRoot + "case-law/α<&>.pdf")
	requireNoError(t, err)
	if id != again || !rawHash.MatchString(strings.TrimPrefix(id, "library_file:")) {
		t.Fatal("unstable exact-key identity")
	}
	if err := libraryvalidation.ValidateB2StorageEndpoint("https://x.r2.cloudflarestorage.com"); err == nil {
		t.Fatal("active R2 endpoint accepted")
	}
}
