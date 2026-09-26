// tool-gateway is the locator-addressed front end for the Python tool-runtime
// registry (D-132).
//
// It gets its OWN Tailscale identity via tsnet, so callers address the gateway
// by a stable tailnet name and which physical host it — or tool-runtime —
// happens to run on stops mattering. That is the durable fix for the defect
// found live on 2026-09-02: the Proffer worker (ovh-files) handed tool-runtime
// (then named platform-tools, on ovh-app) a worker-local filesystem path, and
// the runtime 404'd with the
// path as the response body because the file was not there.
//
// Source bytes cross hosts through the object store, never a shared disk
// (owner, 2026-09-02: "you can object store but use b2 / mount an object if you
// need to"). Only the short-lived materialized copy is local.
//
// Byline: Claude Code · Opus 5 · 2026-09-02.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/objectstores"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/Cursedpotential/probata/engine/toolgateway"
	"github.com/Cursedpotential/probata/engine/tsnetlisten"
)

func main() {
	if err := run(); err != nil {
		slog.Error("tool gateway failed to start", "error", err.Error())
		os.Exit(1)
	}
}

func env(name string) string { return strings.TrimSpace(os.Getenv(name)) }

func requireEnv(name string) (string, error) {
	if v := env(name); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%s is required", name)
}

func requireToolRuntimeURL() (string, error) {
	if value := env("TOOL_RUNTIME_BASE_URL"); value != "" {
		return value, nil
	}
	if value := env("PLATFORM_TOOLS_BASE_URL"); value != "" {
		slog.Warn("PLATFORM_TOOLS_BASE_URL is deprecated; use TOOL_RUNTIME_BASE_URL")
		return value, nil
	}
	return "", errors.New("TOOL_RUNTIME_BASE_URL is required")
}

// readSecretFile reads a mounted secret. The VALUE is never logged — only
// whether it was present and its length, per the platform's secret-handling
// rule.
func readSecretFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	return strings.TrimRight(string(raw), "\r\n"), nil
}

func run() error {
	runtimeBaseURL, err := requireToolRuntimeURL()
	if err != nil {
		return err
	}
	materializeDir, err := requireEnv("TOOL_GATEWAY_MATERIALIZE_DIR")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(materializeDir) {
		return errors.New("TOOL_GATEWAY_MATERIALIZE_DIR must be an absolute path shared with tool-runtime")
	}

	runner, err := runtimeapi.NewToolRuntimeClient(runtimeBaseURL)
	if err != nil {
		return err
	}

	resolver, schemes, err := buildResolver()
	if err != nil {
		return err
	}

	handler := &toolgateway.HTTPHandler{
		Gateway: &toolgateway.Gateway{
			Runner:         runner,
			Resolve:        resolver,
			MaterializeDir: materializeDir,
		},
		Index: toolIndexFunc(runtimeBaseURL),
	}
	if path := env("TOOL_GATEWAY_SERVICE_TOKEN_FILE"); path != "" {
		token, err := readSecretFile(path)
		if err != nil {
			return err
		}
		if len(token) < 32 {
			return errors.New("tool gateway service token is too short to be credible")
		}
		handler.ServiceToken = token
		slog.Info("tool gateway service token loaded", "token_length", len(token))
	}

	listener, describe, cleanup, err := buildListener()
	if err != nil {
		return err
	}
	defer cleanup()
	// tsnet listeners (service or node) hand requests over loopback with the
	// tailnet peer in X-Forwarded-For; only then is that header trusted.
	handler.TrustForwardedFromLoopback = env("TOOL_GATEWAY_TS_AUTHKEY_FILE") != ""

	server := &http.Server{
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 15 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	slog.Info("tool gateway listening",
		"address", describe,
		"tool_runtime", runtimeBaseURL,
		"materialize_dir", materializeDir,
		"resolver_schemes", strings.Join(schemes, ","))

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// buildResolver assembles the scheme router from whatever credentials are
// mounted. An unregistered scheme fails closed rather than silently falling
// back to some default provider.
func buildResolver() (platformpostgres.ImmutableAcquisitionResolver, []string, error) {
	sealRoot, err := requireEnv("TOOL_GATEWAY_SEAL_DIR")
	if err != nil {
		return nil, nil, fmt.Errorf("%w (the directory this gateway seals fetched objects into)", err)
	}
	resolvers := map[string]platformpostgres.ImmutableAcquisitionResolver{}
	var schemes []string

	// Local sealed objects, when the gateway shares a host with an object store.
	if dir := env("SOURCE_OBJECT_DIR"); dir != "" {
		fsResolver, err := runtimeapi.NewFilesystemImmutableAcquisitionResolver(dir)
		if err != nil {
			return nil, nil, err
		}
		uploadResolver, err := acquisition.NewUploadIngressResolver(dir)
		if err != nil {
			return nil, nil, err
		}
		resolvers["file"] = fsResolver
		resolvers["upload"] = uploadResolver
		schemes = append(schemes, "file", "upload")
	}

	// D-132: upload:// objects live on the host that accepted them (the Proffer
	// starter). Off that host, fetch by digest and re-hash before trusting.
	if origin := env("UPLOAD_ORIGIN_URL"); origin != "" && resolvers["upload"] == nil {
		maxBytes, err := strconv.ParseInt(env("PROFFER_UPLOAD_MAX_BYTES"), 10, 64)
		if err != nil || maxBytes <= 0 {
			return nil, nil, errors.New("tool gateway: UPLOAD_ORIGIN_URL requires PROFFER_UPLOAD_MAX_BYTES (positive integer, same bound as the starter)")
		}
		remote, err := acquisition.NewRemoteUploadResolver(sealRoot, origin, maxBytes, &http.Client{Timeout: 10 * time.Minute})
		if err != nil {
			return nil, nil, err
		}
		resolvers["upload"] = remote
		schemes = append(schemes, "upload")
	}

	// Cross-host source bytes travel via object storage. Stores are configuration
	// (OBJECT_STORES_JSON): any S3-compatible provider, under the scheme it is given.
	stores, err := objectstores.StoresFromEnv()
	if err != nil {
		return nil, nil, err
	}
	storeResolvers, err := acquisition.ObjectStoreResolvers(sealRoot, stores, map[string]string{
		"r2": env("CASEBIBLE_R2_CONFIG_PATH"),
		"b2": env("B2_CONFIG_PATH"),
	})
	if err != nil {
		return nil, nil, err
	}
	for scheme, resolver := range storeResolvers {
		resolvers[scheme] = resolver
		schemes = append(schemes, scheme)
	}

	if len(resolvers) == 0 {
		return nil, nil, errors.New("tool gateway: no acquisition resolvers configured — set SOURCE_OBJECT_DIR, UPLOAD_ORIGIN_URL and/or OBJECT_STORES_JSON")
	}
	router, err := acquisition.NewSchemeRouter(resolvers)
	if err != nil {
		return nil, nil, err
	}
	return router, schemes, nil
}

// buildListener gives the gateway its own Tailscale identity.
//
// The tsnet mechanics moved to engine/tsnetlisten on 2026-09-07 so that
// parser-runtime, proffer-starter and the Workbench front use the SAME proven
// listener instead of three copies of it (owner directive: tsnet per service,
// tailnet-only bind). This function keeps the TOOL_GATEWAY_TS_* environment
// names, which are already set in the live Coolify app — the behaviour is
// unchanged, only its implementation is now shared.
//
// PREFERRED: a Tailscale SERVICE (TOOL_GATEWAY_TS_SERVICE, e.g. "svc:tool-gateway").
// Falling back: a plain tsnet node listener, then TOOL_GATEWAY_BIND_IP for hosts
// not yet joined via tsnet. The HTTP layer enforces tailnet-only peers in every
// mode, so no fallback widens exposure.
func buildListener() (net.Listener, string, func(), error) {
	port := env("TOOL_GATEWAY_PORT")
	if port == "" {
		port = "8099"
	}

	keyPath := env("TOOL_GATEWAY_TS_AUTHKEY_FILE")
	if keyPath == "" {
		bindIP, err := requireEnv("TOOL_GATEWAY_BIND_IP")
		if err != nil {
			return nil, "", func() {}, fmt.Errorf("%w (or set TOOL_GATEWAY_TS_AUTHKEY_FILE for a tsnet identity)", err)
		}
		addr := net.JoinHostPort(bindIP, port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, "", func() {}, err
		}
		return listener, addr, func() {}, nil
	}

	stateDir, err := requireEnv("TOOL_GATEWAY_TS_STATE_DIR")
	if err != nil {
		return nil, "", func() {}, err
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("TOOL_GATEWAY_PORT %q is not a port number: %w", port, err)
	}

	cfg := tsnetlisten.Config{
		Service:          tsnetlisten.ToolGateway,
		Hostname:         env("TOOL_GATEWAY_TS_HOSTNAME"),
		TailscaleService: "-", // a plain node listener unless the service is named below
		AuthKeyPath:      keyPath,
		StateDir:         stateDir,
		Port:             portNumber,
	}
	if service := env("TOOL_GATEWAY_TS_SERVICE"); service != "" {
		cfg.TailscaleService = service
	}
	if tags := env("TOOL_GATEWAY_TS_TAGS"); tags != "" {
		cfg.Tags = strings.Split(tags, ",")
	}

	listener, srv, err := tsnetlisten.ListenWith(context.Background(), cfg)
	if err != nil {
		return nil, "", func() {}, err
	}
	return listener, tsnetlisten.Describe(listener), func() { _ = srv.Close() }, nil
}

// toolIndexFunc proxies the tool-runtime registry so callers discover tools
// on the same surface they invoke them on.
func toolIndexFunc(baseURL string) func() (json.RawMessage, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	endpoint := strings.TrimRight(baseURL, "/") + "/tools"
	return func() (json.RawMessage, error) {
		resp, err := client.Get(endpoint)
		if err != nil {
			return nil, fmt.Errorf("tool gateway: list tools: %w", err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil {
			return nil, fmt.Errorf("tool gateway: read tool index: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("tool gateway: tool-runtime index returned %d", resp.StatusCode)
		}
		return json.RawMessage(body), nil
	}
}
