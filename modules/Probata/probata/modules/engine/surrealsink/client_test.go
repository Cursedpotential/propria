// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package surrealsink

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSurreal speaks just enough of SurrealDB's HTTP /rpc to prove the wire contract the Client depends on:
// basic auth, the namespace and database headers, bound variables, and per-statement results. It keeps the
// record ids it was asked to UPSERT per table, so a resend can be shown to leave one copy.
type fakeSurreal struct {
	mu      sync.Mutex
	headers http.Header
	user    string
	queries []string
	tables  map[string]map[string]bool
	fail    string
}

func (f *fakeSurreal) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path != "/rpc" {
		http.NotFound(w, r)
		return
	}
	f.headers = r.Header.Clone()
	f.user, _, _ = r.BasicAuth()
	var request struct {
		Method string `json:"method"`
		Params []json.RawMessage
	}
	_ = json.NewDecoder(r.Body).Decode(&request)
	var sql string
	var vars map[string]json.RawMessage
	_ = json.Unmarshal(request.Params[0], &sql)
	_ = json.Unmarshal(request.Params[1], &vars)
	f.queries = append(f.queries, sql)
	if f.tables == nil {
		f.tables = map[string]map[string]bool{}
	}
	results := []map[string]any{}
	switch {
	case f.fail != "":
		results = append(results, map[string]any{"status": "ERR", "result": f.fail})
	case strings.Contains(sql, "count()"):
		for _, table := range []string{TableMessage, TableEntity, TableEvent} {
			results = append(results, map[string]any{"status": "OK", "result": []map[string]int{{"n": len(f.tables[table])}}})
		}
	default:
		var rows []struct {
			ID string `json:"id"`
		}
		table := TableThread
		for _, candidate := range []string{TableMessage, TableRun, TableEntity, TableEvent} {
			if strings.Contains(sql, "'"+candidate+"'") && strings.Contains(sql, "FOR $") && strings.Contains(sql, "UPSERT type::record('"+candidate+"'") {
				table = candidate
			}
		}
		if raw, ok := vars["rows"]; ok {
			_ = json.Unmarshal(raw, &rows)
		} else if raw, ok := vars["t"]; ok {
			var thread struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(raw, &thread)
			rows = append(rows, struct {
				ID string `json:"id"`
			}{thread.ID})
		}
		if f.tables[table] == nil {
			f.tables[table] = map[string]bool{}
		}
		for _, row := range rows {
			f.tables[table][row.ID] = true
		}
		results = append(results, map[string]any{"status": "OK", "result": nil})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "result": results})
}

func newTestClient(t *testing.T) (*Client, *fakeSurreal) {
	t.Helper()
	fake := &fakeSurreal{}
	server := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(server.Close)
	client, err := New(Config{URL: server.URL, Namespace: "fct", Database: "case", User: "proffer_conversations", Password: "p", AuthLevel: "database", CaseRef: "current"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return client, fake
}

// TestClientSendsCredentialsNamespaceAndVariables proves the wire contract against the shape of SurrealDB's /rpc.
func TestClientSendsCredentialsNamespaceAndVariables(t *testing.T) {
	client, fake := newTestClient(t)
	now := time.Now().UTC()
	if err := client.UpsertThread(context.Background(), Thread{ID: "t1", MatterID: "m", ExportKey: "e", Conv: "c", MessageCount: 2, SentAt: now, SentBy: "owner", SendRequestID: "r"}); err != nil {
		t.Fatal(err)
	}
	if fake.user != "proffer_conversations" || fake.headers.Get("surreal-ns") != "fct" || fake.headers.Get("surreal-db") != "case" ||
		fake.headers.Get("surreal-auth-ns") != "fct" || fake.headers.Get("surreal-auth-db") != "case" {
		t.Fatalf("user = %q, headers = %v", fake.user, fake.headers)
	}
	if !strings.Contains(fake.queries[0], "UPSERT type::record('"+TableThread+"', $t.id)") {
		t.Fatalf("query = %s", fake.queries[0])
	}
}

// TestResendLeavesOneCopyOfEverything proves a second send of the same thread and messages adds no records.
func TestResendLeavesOneCopyOfEverything(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()
	messages := []Message{{ID: "a", ThreadID: "t1", Recipients: []string{}, Participants: json.RawMessage("[]")}, {ID: "b", ThreadID: "t1", Recipients: []string{}, Participants: json.RawMessage("[]")}}
	for i := 0; i < 2; i++ {
		if err := client.UpsertThread(ctx, Thread{ID: "t1", SentAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		if err := client.UpsertMessages(ctx, messages); err != nil {
			t.Fatal(err)
		}
		if err := client.UpsertEntities(ctx, []Entity{{ID: "e1", ThreadID: "t1", RunID: "r1", Mentions: json.RawMessage("[]")}}); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := client.CountThread(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Messages != 2 || counts.Entities != 1 || counts.Events != 0 {
		t.Fatalf("counts after a resend = %+v, want 2 messages, 1 entity, 0 events", counts)
	}
}

// TestFailedStatementIsAnError proves SurrealDB's per-statement failures (reported inside HTTP 200) are not swallowed.
func TestFailedStatementIsAnError(t *testing.T) {
	client, fake := newTestClient(t)
	fake.fail = "permission denied"
	err := client.UpsertMessages(context.Background(), []Message{{ID: "a", Participants: json.RawMessage("[]")}})
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v", err)
	}
}

// TestConfigRefusesIncompleteOrUnsafeSettings proves the boot checks.
func TestConfigRefusesIncompleteOrUnsafeSettings(t *testing.T) {
	good := Config{URL: "http://x", Namespace: "fct", Database: "case", User: "u", Password: "p", AuthLevel: "database"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"non-http url": func(c *Config) { c.URL = "ws://x" },
		"no password":  func(c *Config) { c.Password = "" },
		"bad level":    func(c *Config) { c.AuthLevel = "namespace" },
	} {
		bad := good
		mutate(&bad)
		if bad.Validate() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	t.Setenv(EnvURL, "")
	if _, err := ConfigFromEnv(); err != ErrNotConfigured {
		t.Fatalf("an unset URL gave %v, want ErrNotConfigured", err)
	}
}

// TestOversizedRequestIsRefusedBeforeItIsSent proves the body limit.
func TestOversizedRequestIsRefusedBeforeItIsSent(t *testing.T) {
	client, fake := newTestClient(t)
	big := strings.Repeat("x", maxBody+1)
	err := client.UpsertMessages(context.Background(), []Message{{ID: "a", Body: big, Participants: json.RawMessage("[]")}})
	if err == nil || len(fake.queries) != 0 {
		t.Fatalf("error = %v, requests = %d", err, len(fake.queries))
	}
}

// TestMissingCredentialFilesDisableTheSendInsteadOfStoppingTheWorker proves an unmounted credential file is
// ErrNotConfigured (the worker boots, the send reports why), not a boot failure.
func TestMissingCredentialFilesDisableTheSendInsteadOfStoppingTheWorker(t *testing.T) {
	t.Setenv(EnvURL, "http://100.91.190.107:8471")
	t.Setenv(EnvUserFile, t.TempDir()+"/absent-user")
	t.Setenv(EnvPasswordFile, t.TempDir())
	_, err := ConfigFromEnv()
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}
