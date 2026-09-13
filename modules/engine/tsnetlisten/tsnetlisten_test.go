// Unit tests for the shared tsnet listener. Nothing here touches a tailnet:
// the parts that need one (registration, ListenService) are proven live, and
// the parts that are pure policy — the exclusive-bind guard, the rollout flag,
// peer resolution across both address families and the serve-proxy loopback
// hop, and header stripping — are proven here.
//
// Byline: Claude Code subagent · Opus 5 · 2026-09-07.
package tsnetlisten

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGuardExclusiveBindRefusesBothListeners(t *testing.T) {
	if err := GuardExclusiveBind(true, "100.91.190.107:8090", false); err == nil {
		t.Fatal("a tsnet listener plus a host bind must fail closed")
	} else if !errors.Is(err, ErrConflictingBind) {
		t.Fatalf("want ErrConflictingBind, got %v", err)
	}
	if err := GuardExclusiveBind(true, "  ", false); err != nil {
		t.Fatalf("no host bind configured must pass: %v", err)
	}
	if err := GuardExclusiveBind(false, "100.91.190.107:8090", false); err != nil {
		t.Fatalf("flag off keeps today's bind: %v", err)
	}
	if err := GuardExclusiveBind(true, "100.91.190.107:8090", true); err != nil {
		t.Fatalf("an explicit migration window must be allowed: %v", err)
	}
}

func TestEnabledDefaultsOffSoAPushCannotFlipALiveService(t *testing.T) {
	t.Setenv(EnabledEnv, "")
	if Enabled() {
		t.Fatal("the rollout flag must default OFF (D-127: the gated path stays proven; the flip is a deploy act)")
	}
	for _, on := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv(EnabledEnv, on)
		if !Enabled() {
			t.Fatalf("%q must enable the tsnet listener", on)
		}
	}
	for _, off := range []string{"0", "false", "no", "off"} {
		t.Setenv(EnabledEnv, off)
		if Enabled() {
			t.Fatalf("%q must keep the legacy bind", off)
		}
	}
	t.Setenv(EnabledEnv, "banana")
	if Enabled() {
		t.Fatal("an unparseable value must fall back to the default, not to on")
	}
}

func TestHostnameFallsBackToTheCanonicalComponentName(t *testing.T) {
	t.Setenv(HostnameEnv, "")
	if got := Hostname(ParserRuntime); got != "parser-runtime" {
		t.Fatalf("Hostname = %q, want the canonical name from docs/NAMING.md", got)
	}
	t.Setenv(HostnameEnv, "parser-runtime-canary")
	if got := Hostname(ParserRuntime); got != "parser-runtime-canary" {
		t.Fatalf("Hostname = %q, want the override", got)
	}
}

func TestListenWithRejectsMisconfigurationBeforeTouchingTheTailnet(t *testing.T) {
	ctx := context.Background()
	if _, _, err := ListenWith(ctx, Config{AuthKeyPath: "/x"}); err == nil {
		t.Fatal("a missing service name must fail")
	}
	if _, _, err := ListenWith(ctx, Config{Service: ParserRuntime}); err == nil {
		t.Fatal("a missing auth key path must fail")
	}
	if _, _, err := ListenWith(ctx, Config{Service: ParserRuntime, AuthKeyPath: "/x", TailscaleService: "parser-runtime"}); err == nil {
		t.Fatal("a Tailscale Service name without the svc: prefix must fail")
	}
	if _, _, err := ListenWith(ctx, Config{Service: ParserRuntime, AuthKeyPath: filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Fatal("an unreadable auth key file must fail")
	}
	empty := filepath.Join(t.TempDir(), "authkey")
	if err := os.WriteFile(empty, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ListenWith(ctx, Config{Service: ParserRuntime, AuthKeyPath: empty}); err == nil {
		t.Fatal("an empty auth key file must fail")
	}
}

func TestReadSecretFileTrimsOnlyTheTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("  value-with-padding  \r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSecretFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "  value-with-padding  " {
		t.Fatalf("ReadSecretFile trimmed more than the line ending: %q", got)
	}
}

// --- identity -------------------------------------------------------------

type stubWhoIs struct {
	login   string
	display string
	err     error
	asked   string
}

func (s *stubWhoIs) LoginName(_ context.Context, remoteAddr string) (string, string, error) {
	s.asked = remoteAddr
	return s.login, s.display, s.err
}

func requestWith(remoteAddr, forwarded string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/anything", nil)
	r.RemoteAddr = remoteAddr
	if forwarded != "" {
		r.Header.Set("X-Forwarded-For", forwarded)
	}
	return r
}

func TestTailnetPeerAcceptsBothFamiliesAndTheServeProxyLoopbackHop(t *testing.T) {
	cases := []struct {
		name      string
		remote    string
		forwarded string
		want      string
		ok        bool
	}{
		{"ipv4 cgnat direct", "100.91.190.107:41234", "", "100.91.190.107", true},
		{"ipv6 ula direct", "[fd7a:115c:a1e0::1b29:fb86]:41234", "", "fd7a:115c:a1e0::1b29:fb86", true},
		{"serve proxy loopback hop", "127.0.0.1:59786", "100.91.190.107", "100.91.190.107", true},
		{"serve proxy multi hop", "127.0.0.1:1", "fd7a:115c:a1e0::1b29:fb86, 10.0.0.9", "fd7a:115c:a1e0::1b29:fb86", true},
		{"loopback with public forwarded", "127.0.0.1:1", "203.0.113.5", "", false},
		{"loopback with no forwarded", "127.0.0.1:1", "", "", false},
		{"public direct peer", "203.0.113.5:443", "100.91.190.107", "", false},
		{"private direct peer", "192.168.112.2:443", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, ok := TailnetPeer(requestWith(tc.remote, tc.forwarded))
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && addr.String() != tc.want {
				t.Fatalf("peer = %q, want %q", addr.String(), tc.want)
			}
		})
	}
}

func TestIdentityStampsTheLoginAndStripsAnySuppliedIdentity(t *testing.T) {
	var seenLogin, seenName string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenLogin = r.Header.Get(UserLoginHeader)
		seenName = r.Header.Get(UserNameHeader)
	})
	who := &stubWhoIs{login: "owner@example.com", display: "Owner"}
	handler := NewIdentity(who, next)

	r := requestWith("100.91.190.107:41234", "")
	r.Header.Set(UserLoginHeader, "attacker@example.com")
	r.Header.Set(UserNameHeader, "Attacker")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if seenLogin != "owner@example.com" {
		t.Fatalf("login header = %q, want the WhoIs answer (a supplied header must be discarded)", seenLogin)
	}
	if seenName != "Owner" {
		t.Fatalf("name header = %q, want %q", seenName, "Owner")
	}
	if who.asked != "100.91.190.107" {
		t.Fatalf("WhoIs asked about %q, want the real tailnet peer", who.asked)
	}
}

func TestIdentityRejectsNonTailnetPeersAndUnknownIdentities(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })

	w := httptest.NewRecorder()
	NewIdentity(&stubWhoIs{login: "owner@example.com"}, next).ServeHTTP(w, requestWith("203.0.113.5:443", ""))
	if w.Code != http.StatusUnauthorized || reached {
		t.Fatalf("a non-tailnet peer must be rejected: status=%d reached=%v", w.Code, reached)
	}

	w = httptest.NewRecorder()
	NewIdentity(&stubWhoIs{err: errors.New("whois down")}, next).ServeHTTP(w, requestWith("100.91.190.107:1", ""))
	if w.Code != http.StatusUnauthorized || reached {
		t.Fatalf("a failed WhoIs must fail closed: status=%d reached=%v", w.Code, reached)
	}

	w = httptest.NewRecorder()
	NewIdentity(&stubWhoIs{login: "  "}, next).ServeHTTP(w, requestWith("100.91.190.107:1", ""))
	if w.Code != http.StatusUnauthorized || reached {
		t.Fatalf("a blank login must fail closed: status=%d reached=%v", w.Code, reached)
	}
}

func TestIdentityOptionalModePassesThroughWithoutAHeader(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.Header.Get(UserLoginHeader) })
	handler := &Identity{WhoIs: &stubWhoIs{err: errors.New("whois down")}, Next: next, Required: false}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, requestWith("100.91.190.107:1", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if seen != "" {
		t.Fatalf("no identity must mean no header, got %q", seen)
	}
}
