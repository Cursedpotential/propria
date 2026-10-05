// Byline: Codex · GPT-6.1 · 2026-10-05. Only isolated synthetic HTTPS fixtures are called.
package librarysync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPBackendFrozenRoutesAuthAndPayloadHash(t *testing.T) {
	_, _, memory := fixtureService(t)
	claim := memory.claim
	id := claim.Operation.BindingID
	paths := []string{}
	token := strings.Repeat("synthetic-token-", 3)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("dedicated auth absent")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case APIBase + "/outbox/claim":
			var in OperationInput
			requireNoError(t, json.NewDecoder(r.Body).Decode(&in))
			if in.OperationID != fixtureOperation || in.AttemptID != "attempt" {
				t.Error("wrong claim body")
			}
			json.NewEncoder(w).Encode(claim)
		case APIBase + "/outbox/" + fixtureOperation + "/payload":
			if r.Header.Get("X-Toolkit-Sync-Lease") != claim.LeaseID {
				t.Error("lease header absent")
			}
			w.Header().Set("Content-Type", "text/markdown")
			w.Write(memory.payload)
		case APIBase + "/outbox/" + fixtureOperation + "/write-intent":
			var in map[string]any
			requireNoError(t, json.NewDecoder(r.Body).Decode(&in))
			if in["lease_id"] != claim.LeaseID || in["attempt_id"] != "attempt" || in["fence"] != float64(1) {
				t.Error("wrong intent body")
			}
			json.NewEncoder(w).Encode(Intent{MayWrite: true, IntentID: "i"})
		case APIBase + "/observations/" + digest([]byte("observation")) + "/status":
			w.Write([]byte(`{"seen":true}`))
		case APIBase + "/observations":
			var in Observation
			requireNoError(t, json.NewDecoder(r.Body).Decode(&in))
			if in.Status != Blocked {
				t.Error("wrong observation body")
			}
			json.NewEncoder(w).Encode(Outcome{Status: Blocked})
		case APIBase + "/outbox/" + fixtureOperation + "/complete":
			var in Completion
			requireNoError(t, json.NewDecoder(r.Body).Decode(&in))
			if in.IntentID != "i" || in.Coverage != "partial" {
				t.Error("wrong completion body")
			}
			json.NewEncoder(w).Encode(Outcome{Status: WriteUnknown})
		case APIBase + "/outbox/" + fixtureOperation + "/failure":
			json.NewEncoder(w).Encode(Outcome{Status: Blocked, Code: "SAFE_CODE"})
		case APIBase + "/bindings/" + id + "/original":
			if r.URL.Query().Get("version_id") != "exact/version+id" {
				t.Error("unpinned original")
			}
			json.NewEncoder(w).Encode(Binding{ID: id, Provider: "b2", Scope: fixtureScope})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	backend, err := NewHTTPBackend(server.URL, token, server.Client().Transport)
	requireNoError(t, err)
	c, err := backend.Claim(context.Background(), OperationInput{fixtureOperation, "attempt"})
	requireNoError(t, err)
	raw, err := backend.Payload(context.Background(), c)
	requireNoError(t, err)
	if string(raw) != string(memory.payload) {
		t.Fatal("payload changed")
	}
	_, err = backend.BeginWrite(context.Background(), c, "attempt")
	requireNoError(t, err)
	seen, err := backend.Seen(context.Background(), digest([]byte("observation")))
	requireNoError(t, err)
	if !seen {
		t.Fatal("seen response lost")
	}
	_, err = backend.Observe(context.Background(), Observation{Status: Blocked})
	requireNoError(t, err)
	_, err = backend.Complete(context.Background(), c, Completion{IntentID: "i", Coverage: "partial"})
	requireNoError(t, err)
	_, err = backend.Failure(context.Background(), c, Blocked, "SAFE_CODE")
	requireNoError(t, err)
	_, err = backend.Original(context.Background(), id, "exact/version+id")
	requireNoError(t, err)
	if len(paths) != 8 {
		t.Fatalf("route coverage %v", paths)
	}
}

func TestHTTPBackendRejectsRedirectExtraFieldsBudgetsAndPayloadMismatch(t *testing.T) {
	for _, kind := range []string{"redirect", "unknown-field", "trailing-json", "body-budget", "status-error", "payload-mismatch"} {
		t.Run(kind, func(t *testing.T) {
			targetCalls := 0
			target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targetCalls++
				t.Error("redirect exposed service request")
			}))
			defer target.Close()
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch kind {
				case "redirect":
					http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
				case "unknown-field":
					w.Write([]byte(`{"seen":true,"extra":1}`))
				case "trailing-json":
					w.Write([]byte(`{"seen":true} {}`))
				case "body-budget":
					io.WriteString(w, strings.Repeat("x", int(metadataBudget)+1))
				case "status-error":
					http.Error(w, "synthetic-private-server-error", http.StatusConflict)
				case "payload-mismatch":
					w.Write([]byte("different bytes"))
				}
			}))
			defer server.Close()
			backend, err := NewHTTPBackend(server.URL, strings.Repeat("s", 32), server.Client().Transport)
			requireNoError(t, err)
			if kind == "payload-mismatch" {
				_, _, memory := fixtureService(t)
				_, err = backend.Payload(context.Background(), memory.claim)
			} else {
				_, err = backend.Seen(context.Background(), digest([]byte("obs")))
			}
			if err == nil {
				t.Fatalf("%s accepted", kind)
			}
			if strings.Contains(err.Error(), "synthetic-private") || targetCalls != 0 {
				t.Fatal("private error/credential leaked")
			}
		})
	}
}

func TestEnvReusesExistingApprovedB2MountWithoutProvisioning(t *testing.T) {
	// Preserve fixture files; deliberately avoid TempDir/Cleanup deletion under the owner's retention rule.
	root := filepath.Join(os.TempDir(), "toolkit-librarysync-retained-fixtures")
	requireNoError(t, os.MkdirAll(root, 0700))
	dir, err := os.MkdirTemp(root, "config-")
	requireNoError(t, err)
	tokenFile := filepath.Join(dir, "service-token")
	cfgFile := filepath.Join(dir, "existing-b2.json")
	requireNoError(t, os.WriteFile(tokenFile, []byte("TOOLKIT_LIBRARY_SYNC_TOKEN='"+strings.Repeat("synthetic", 5)+"'\n"), 0600))
	cfg := B2Config{Endpoint: "https://s3.us-west-004.backblazeb2.com", Region: "us-west-004", AccessKeyID: "synthetic-existing-key", SecretAccessKey: "synthetic-existing-secret"}
	raw, _ := json.Marshal(cfg)
	requireNoError(t, os.WriteFile(cfgFile, raw, 0600))
	t.Setenv(EnvTokenFile, tokenFile)
	t.Setenv(EnvB2ConfigFile, cfgFile)
	t.Setenv(EnvBackendURL, "https://synthetic-familycourt.example")
	t.Setenv(EnvB2Bucket, fixtureScope.Bucket)
	t.Setenv(EnvAccountScope, fixtureScope.AccountScope)
	t.Setenv(EnvLegalRoot, fixtureScope.LegalRoot)
	reader, err := NewOriginalReaderFromEnv()
	requireNoError(t, err)
	if reader.Scope != fixtureScope || reader.Storage == nil || reader.Backend == nil {
		t.Fatal("read-only starter dependencies absent")
	}
	s, _, _ := fixtureService(t)
	ready, err := NewServiceFromEnv(s.Artifacts, s.Extractor, s.SigningKey)
	requireNoError(t, err)
	if ready.Scope != fixtureScope {
		t.Fatal("scope changed")
	}
	cfg.Endpoint = "https://retired.r2.cloudflarestorage.com"
	raw, _ = json.Marshal(cfg)
	requireNoError(t, os.WriteFile(cfgFile, raw, 0600))
	if _, err = NewServiceFromEnv(s.Artifacts, s.Extractor, s.SigningKey); err == nil {
		t.Fatal("retired provider accepted")
	}
	for _, origin := range []string{"http://host", "https://user:secret@host", "https://host/path", "https://host?token=x"} {
		if _, err = NewHTTPBackend(origin, strings.Repeat("s", 32), nil); err == nil {
			t.Fatalf("unsafe origin %s", origin)
		}
	}
}
