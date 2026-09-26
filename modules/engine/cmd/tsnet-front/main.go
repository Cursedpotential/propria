// Command tsnet-front is the tailnet identity for a service that cannot embed
// one itself.
//
// WHY IT EXISTS: the owner's 2026-09-07 directive is tsnet per service, bound
// to the tailnet only, with no host networking endpoint. A Go service does that
// in-process (engine/tsnetlisten). Python cannot: there is no tsnet for
// CPython, so the Workbench's FastAPI has been reachable only because the HOST
// runs `tailscale serve` for svc:workbench into a published loopback port
// (127.0.0.1:18080). That published port is a host networking endpoint, and it
// ties the service's identity to whichever host it happens to run on — exactly
// what D-134 says to stop doing.
//
// This binary closes that gap without touching the Python. It takes the tsnet
// identity, resolves the caller through the tailnet's own WhoIs, stamps
// Tailscale-User-Login (the header the Workbench's auth middleware ALREADY
// trusts from its configured proxy peer — see
// modules/workbench/api/app/runtime/auth.py), and reverse-proxies to the
// upstream. Nothing about the application changes; only who terminates the
// connection.
//
// DEPLOYMENT SHAPE: run it in the upstream container's own network namespace
// (compose `network_mode: "service:<name>"`) and point it at 127.0.0.1. That
// makes the proxy peer the loopback address the Workbench already allows in
// TRUSTED_TAILSCALE_SERVE_PROXY_CIDRS, so no CIDR has to be discovered, no
// compose-DNS name has to be resolved at startup, and no Python changes. It is
// the simplest arrangement that actually works.
//
// Byline: Claude Code subagent · Opus 5 · 2026-09-07.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Cursedpotential/probata/engine/tsnetlisten"
)

const (
	// UpstreamEnv is the service this front stands in front of.
	UpstreamEnv = "TSNET_FRONT_UPSTREAM"
	// ServiceNameEnv names which canonical component this front belongs to.
	ServiceNameEnv = "TSNET_FRONT_SERVICE"

	defaultUpstream          = "http://127.0.0.1:8020"
	shutdownGrace            = 20 * time.Second
	defaultReadHeaderTimeout = 15 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("tsnet front failed to start", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	service := tsnetlisten.ServiceName(strings.TrimSpace(os.Getenv(ServiceNameEnv)))
	if service == "" {
		service = tsnetlisten.Workbench
	}

	upstreamRaw := strings.TrimSpace(os.Getenv(UpstreamEnv))
	if upstreamRaw == "" {
		upstreamRaw = defaultUpstream
	}
	upstream, err := url.Parse(upstreamRaw)
	if err != nil || upstream.Scheme == "" || upstream.Host == "" {
		return fmt.Errorf("%s must be an absolute http(s) URL, got %q", UpstreamEnv, upstreamRaw)
	}

	// This binary exists to REPLACE a host endpoint, so refusing to run
	// alongside one is the whole point (owner: "bind to TS only").
	if !tsnetlisten.Enabled() {
		return fmt.Errorf("tsnet front: %s is off — this service has no listener other than its tsnet identity", tsnetlisten.EnabledEnv)
	}

	listener, server, err := tsnetlisten.ListenWith(context.Background(), tsnetlisten.FromEnv(service))
	if err != nil {
		return err
	}
	defer func() { _ = server.Close() }()

	localClient, err := server.LocalClient()
	if err != nil {
		return fmt.Errorf("tsnet front: local client: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(upstream)
			r.Out.Host = upstream.Host
			// The tailnet terminated TLS in this process; tell the upstream so
			// it builds absolute URLs correctly.
			r.Out.Header.Set("X-Forwarded-Proto", "https")
		},
		// Immediate flush keeps server-sent events and streamed responses
		// usable through the proxy.
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			slog.Warn("tsnet front: upstream error", "upstream", upstream.String(), "error", err.Error())
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		},
	}

	handler := tsnetlisten.NewIdentity(tsnetlisten.NewWhoIser(localClient), proxy)

	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	slog.Info("tsnet front listening",
		"service", string(service),
		"address", tsnetlisten.Describe(listener),
		"upstream", upstream.String())

	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
