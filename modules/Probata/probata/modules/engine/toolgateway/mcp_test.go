// Tests for the single `atomic_tools` MCP tool, driven through the real HTTP handler.
//
// Byline: Claude Code · Opus 5.5 · 2026-09-28.
package toolgateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testManifest = `[
 {"id":"messages.sms-xml","capability":"parse.sms-xml","description":"SMS Backup XML","side_effect":"read_only","formats":["xml"]},
 {"id":"messages.whatsapp-txt","capability":"parse.whatsapp","description":"WhatsApp text export","side_effect":"read_only","formats":["txt"]},
 {"id":"repair.capabilities","capability":"repair.inspect","description":"Engine readiness probe","side_effect":"read_only","formats":[]}
]`

func mcpHandler(t *testing.T, runner *fakeRunner, token string) http.Handler {
	t.Helper()
	h := &HTTPHandler{
		Gateway:      &Gateway{Runner: runner},
		Index:        func() (json.RawMessage, error) { return json.RawMessage(testManifest), nil },
		ServiceToken: token,
	}
	return h.Routes()
}

// rpc posts one JSON-RPC call to /mcp from a tailnet peer and returns the decoded envelope.
func rpc(t *testing.T, handler http.Handler, method string, params any) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(string(body)))
	req.RemoteAddr = "100.91.190.107:5555"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var env map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env
}

// call invokes atomic_tools and returns its structured result.
func call(t *testing.T, handler http.Handler, args map[string]any) map[string]any {
	t.Helper()
	code, env := rpc(t, handler, "tools/call", map[string]any{"name": "atomic_tools", "arguments": args})
	if code != http.StatusOK {
		t.Fatalf("tools/call status %d, body %v", code, env)
	}
	result, _ := env["result"].(map[string]any)
	out, _ := result["structuredContent"].(map[string]any)
	if out == nil {
		t.Fatalf("no structuredContent in %v", env)
	}
	return out
}

func TestMCPListsExactlyOneToolWithGeneratedDescription(t *testing.T) {
	handler := mcpHandler(t, &fakeRunner{}, "")
	code, env := rpc(t, handler, "tools/list", map[string]any{})
	if code != http.StatusOK {
		t.Fatalf("tools/list status %d, body %v", code, env)
	}
	tools := env["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("want exactly one tool, got %d", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "atomic_tools" {
		t.Fatalf("tool name %v", tool["name"])
	}
	desc, _ := tool["description"].(string)
	for _, want := range []string{"3 tools in 2 families", "- messages (2;", "- repair (1;", "never a host path"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description missing %q:\n%s", want, desc)
		}
	}
}

func TestMCPBrowsesEveryLevel(t *testing.T) {
	handler := mcpHandler(t, &fakeRunner{}, "")

	root := call(t, handler, map[string]any{"path": ""})
	if root["level"] != "root" || root["tool_count"].(float64) != 3 {
		t.Fatalf("root: %v", root)
	}
	fam := call(t, handler, map[string]any{"path": "messages"})
	if fam["level"] != "family" || len(fam["tools"].([]any)) != 2 {
		t.Fatalf("family: %v", fam)
	}
	one := call(t, handler, map[string]any{"path": "messages.sms-xml"})
	if one["level"] != "tool" || one["tool"].(map[string]any)["id"] != "messages.sms-xml" {
		t.Fatalf("tool: %v", one)
	}
	miss := call(t, handler, map[string]any{"path": "sms"})
	if miss["level"] != "not_found" || len(miss["suggestions"].([]any)) != 1 {
		t.Fatalf("not_found should suggest messages.sms-xml: %v", miss)
	}
}

func TestMCPRunWithoutSourceRefCallsRunnerDirectly(t *testing.T) {
	runner := &fakeRunner{}
	handler := mcpHandler(t, runner, "")
	out := call(t, handler, map[string]any{"path": "repair.capabilities", "run": map[string]any{"args": map[string]any{"verbose": true}}})
	if out["ok"] != true || runner.calls != 1 || runner.lastID != "repair.capabilities" {
		t.Fatalf("run: %v, runner calls=%d id=%q", out, runner.calls, runner.lastID)
	}
	if runner.lastArgs["verbose"] != true {
		t.Fatalf("args not passed through: %v", runner.lastArgs)
	}
}

func TestMCPRunRefusesAHostPath(t *testing.T) {
	runner := &fakeRunner{}
	handler := mcpHandler(t, runner, "")
	out := call(t, handler, map[string]any{"path": "messages.sms-xml", "run": map[string]any{"args": map[string]any{"path": "/r2/evidence/x.xml"}}})
	if out["ok"] != false || out["error"] != "contract_rejected" {
		t.Fatalf("want contract_rejected, got %v", out)
	}
	if runner.calls != 0 {
		t.Fatalf("runner must not be called when a host path is named (calls=%d)", runner.calls)
	}
}

func TestMCPRunRejectsUnknownTool(t *testing.T) {
	runner := &fakeRunner{}
	out := call(t, mcpHandler(t, runner, ""), map[string]any{"path": "messages.nope", "run": map[string]any{}})
	if out["error"] != "unknown_tool" || runner.calls != 0 {
		t.Fatalf("want unknown_tool without a run, got %v (calls=%d)", out, runner.calls)
	}
}

func TestMCPRejectsNonTailnetPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.RemoteAddr = "203.0.113.9:5555"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mcpHandler(t, &fakeRunner{}, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("non-tailnet peer: want 401, got %d", rec.Code)
	}
}

// The tailnet is the only barrier (owner rule 2026-09-23): /mcp needs no bearer token even when
// the gateway has one configured for its REST routes.
func TestMCPNeedsNoBearerTokenWhileRESTStillDoes(t *testing.T) {
	handler := mcpHandler(t, &fakeRunner{}, strings.Repeat("s", 40))
	if code, env := rpc(t, handler, "tools/list", map[string]any{}); code != http.StatusOK {
		t.Fatalf("/mcp without a token: want 200, got %d %v", code, env)
	}
	req := httptest.NewRequest(http.MethodGet, "/tools", nil)
	req.RemoteAddr = "100.91.190.107:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("/tools without a token: want 401, got %d", rec.Code)
	}
}

// In production tsnet terminates TLS in-process and hands the request over loopback with the
// real peer in X-Forwarded-For and the service hostname in Host. That must be accepted.
func TestMCPAcceptsTsnetLoopbackHandoff(t *testing.T) {
	h := &HTTPHandler{
		Gateway:                    &Gateway{Runner: &fakeRunner{}},
		Index:                      func() (json.RawMessage, error) { return json.RawMessage(testManifest), nil },
		TrustForwardedFromLoopback: true,
	}
	req := httptest.NewRequest(http.MethodPost, "https://tool-gateway.tilapia-skilift.ts.net/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.RemoteAddr = "127.0.0.1:41234"
	req.Host = "tool-gateway.tilapia-skilift.ts.net"
	// The connection itself lands on loopback, as it does behind tsnet. Without this the SDK's
	// DNS-rebinding guard never engages and the test could not catch it being switched back on.
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey,
		&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8443}))
	req.Header.Set("X-Forwarded-For", "100.91.190.107")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tsnet loopback handoff: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
