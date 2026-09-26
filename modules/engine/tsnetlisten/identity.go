// WhoIs-based identity for tsnet listeners.
//
// Host `tailscale serve` stamps Tailscale-User-Login on every request it
// proxies, and the Workbench already trusts that header from its configured
// proxy peer (modules/workbench/api/app/runtime/auth.py). A Go tsnet front
// must produce the SAME header from the SAME source of truth — the tailnet's
// own WhoIs — so replacing host Serve with an in-process listener is invisible
// to the application behind it.
//
// Two things this must get right, both learned live:
//
//   - In tsnet HTTPS service mode the connection is terminated by tailscale's
//     serve proxy inside the process and handed to the handler over loopback,
//     so RemoteAddr is 127.0.0.1 and the real peer travels in X-Forwarded-For
//     (vendor/tailscale.com/ipn/ipnlocal/serve.go; the same finding is recorded
//     in engine/toolgateway/http.go). WhoIs must be asked about the real peer.
//   - Both Tailscale address families reach a listener: IPv4 CGNAT
//     100.64.0.0/10 and the IPv6 ULA fd7a:115c:a1e0::/48. An IPv4-only check
//     rejected every VIP-service call on 2026-09-05.
//
// Byline: Claude Code subagent · Opus 5 · 2026-09-07.
package tsnetlisten

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"tailscale.com/client/local"
)

// Headers this package stamps. They are STRIPPED from every inbound request
// before being set, so a caller can never assert its own identity.
const (
	UserLoginHeader = "Tailscale-User-Login"
	UserNameHeader  = "Tailscale-User-Name"
)

// whoIsTimeout bounds the local API call so a stalled tailscaled cannot hold a
// request open.
const whoIsTimeout = 5 * time.Second

// WhoIser is the slice of the tailscale local client this package needs. It
// exists so the middleware is testable without a tailnet.
type WhoIser interface {
	LoginName(ctx context.Context, remoteAddr string) (login, display string, err error)
}

// localWhoIser adapts *local.Client.
type localWhoIser struct{ client *local.Client }

func (l localWhoIser) LoginName(ctx context.Context, remoteAddr string) (string, string, error) {
	if l.client == nil {
		return "", "", errors.New("tsnetlisten: no tailscale local client")
	}
	who, err := l.client.WhoIs(ctx, remoteAddr)
	if err != nil {
		return "", "", err
	}
	if who == nil || who.UserProfile == nil {
		return "", "", errors.New("tsnetlisten: WhoIs returned no user profile")
	}
	return who.UserProfile.LoginName, who.UserProfile.DisplayName, nil
}

// NewWhoIser wraps a tailscale local client (tsnet.Server.LocalClient()).
func NewWhoIser(client *local.Client) WhoIser { return localWhoIser{client: client} }

// Identity stamps Tailscale-User-Login (and -User-Name) from WhoIs.
//
// A request whose peer cannot be identified is REJECTED when Required is true,
// which is the only setting anything in this repository uses: an unidentified
// caller reaching a tailnet-only service is a defect, not an anonymous user.
type Identity struct {
	WhoIs    WhoIser
	Next     http.Handler
	Required bool
}

// NewIdentity is the constructor callers should use: identity required.
func NewIdentity(who WhoIser, next http.Handler) *Identity {
	return &Identity{WhoIs: who, Next: next, Required: true}
}

func (i *Identity) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Never let a caller supply its own identity.
	r.Header.Del(UserLoginHeader)
	r.Header.Del(UserNameHeader)

	peer, ok := TailnetPeer(r)
	if !ok {
		slog.Warn("tsnet identity: rejected non-tailnet peer", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
		http.Error(w, "tailnet peer required", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), whoIsTimeout)
	defer cancel()
	login, display, err := i.WhoIs.LoginName(ctx, peer.String())
	if err != nil || strings.TrimSpace(login) == "" {
		if i.Required {
			reason := "no login name"
			if err != nil {
				reason = err.Error()
			}
			slog.Warn("tsnet identity: WhoIs failed", "peer", peer.String(), "path", r.URL.Path, "reason", reason)
			http.Error(w, "tailnet identity required", http.StatusUnauthorized)
			return
		}
		i.Next.ServeHTTP(w, r)
		return
	}

	r.Header.Set(UserLoginHeader, login)
	if trimmed := strings.TrimSpace(display); trimmed != "" {
		r.Header.Set(UserNameHeader, trimmed)
	}
	i.Next.ServeHTTP(w, r)
}

// TailnetPeer resolves the real tailnet peer of a request delivered by a tsnet
// listener. It returns false for anything that is not a tailnet address.
func TailnetPeer(r *http.Request) (netip.Addr, bool) {
	direct, err := parseHostAddr(r.RemoteAddr)
	if err == nil && IsTailnetAddr(direct) {
		return direct, true
	}
	// tsnet HTTPS service mode: the in-process serve proxy hands the request
	// over loopback with the tailnet peer in X-Forwarded-For. This is trusted
	// ONLY because the listener is a tsnet listener and the direct peer is
	// loopback inside this very process — never on a host-bound listener.
	if err == nil && direct.IsLoopback() {
		forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		if addr, perr := netip.ParseAddr(forwarded); perr == nil && IsTailnetAddr(addr) {
			return addr, true
		}
	}
	return netip.Addr{}, false
}

func parseHostAddr(remoteAddr string) (netip.Addr, error) {
	trimmed := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}
	return netip.ParseAddr(trimmed)
}

var tailscaleULA = netip.MustParsePrefix("fd7a:115c:a1e0::/48")
var tailscaleCGNAT = netip.MustParsePrefix("100.64.0.0/10")

// IsTailnetAddr reports whether an address is inside Tailscale's IPv4 CGNAT
// range or its IPv6 ULA range.
func IsTailnetAddr(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return tailscaleCGNAT.Contains(addr)
	}
	return tailscaleULA.Contains(addr)
}
