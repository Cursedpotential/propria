// Package tsnetlisten is the one place this repository gives a Go service its
// own Tailscale identity (D-134: "every service gets its own Tailscale
// identity"; D-132 built the first one).
//
// It is an EXTRACTION, not a new design: every rule here was already proven
// live by cmd/tool-gateway, which now calls this package instead of carrying
// its own copy. What changed is only that the rules are stated once.
//
// The contract, in the order it matters:
//
//  1. A service listens on its tsnet listener and nothing else. GuardExclusiveBind
//     refuses to start a process that asks for BOTH a tsnet listener and a plain
//     host bind, unless the caller passes the rollout flag explicitly.
//  2. A Tailscale SERVICE (svc:<name>) is preferred over a plain node, because a
//     service FQDN belongs to the service rather than to whichever host it
//     landed on — the decoupling whose absence made the D-132 cross-host path
//     defect possible in the first place.
//  3. Services need a TAG-BASED identity, so Tags must be non-empty in service
//     mode and the auth key must be minted with the same tag.
//  4. The auth key is read from a mounted FILE. Its value is never logged; only
//     its presence and length are.
//  5. StateDir must be persistent or the node re-registers on every deploy.
//
// Byline: Claude Code subagent · Opus 5 · 2026-09-07.
package tsnetlisten

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"tailscale.com/tsnet"
)

// ServiceName is a canonical component name from docs/NAMING.md section 2 — the
// same string used for the tsnet hostname, the Tailscale Service, and the state
// directory. Naming it as a type keeps a caller from passing a Coolify app
// name, a container name, or a host name by accident.
type ServiceName string

// The canonical component names that currently have (or are getting) a tsnet
// identity. docs/NAMING.md section 2 is the authority; this list is the subset
// with a network surface.
const (
	ToolGateway    ServiceName = "tool-gateway"
	ParserRuntime  ServiceName = "parser-runtime"
	ProfferStarter ServiceName = "proffer-starter"
	Workbench      ServiceName = "workbench"
)

// EnabledEnv is the rollout flag (D-127). Its removal condition: delete the
// flag and the legacy bind branch once every service in the constant block
// above is serving from tsnet and verified live.
const EnabledEnv = "TSNET_LISTENER_ENABLED"

// HostnameEnv overrides the canonical hostname for one process.
const HostnameEnv = "TSNET_HOSTNAME"

// AuthKeyEnv names the mounted Tailscale auth key file. Same secret-mount
// pattern as the tool gateway: a read-only file, never an environment value.
const AuthKeyEnv = "TSNET_AUTHKEY_FILE"

// StateDirEnv names the persistent tsnet state directory inside the container.
const StateDirEnv = "TSNET_STATE_DIR"

// ServiceEnv overrides the Tailscale Service name ("-" forces a plain node).
const ServiceEnv = "TSNET_SERVICE"

// TagsEnv is the comma-separated tag list the auth key was minted with.
const TagsEnv = "TSNET_TAGS"

// StateRoot is the host root every service's tsnet state lives under
// (/data/probata per the 2026-09-07 host-root cut, register section 13).
const StateRoot = "/data/probata/tsnet"

// Config fully describes one service's tsnet identity.
type Config struct {
	// Service is the canonical component name. Required.
	Service ServiceName

	// Hostname defaults to Service. It is the tailnet node name.
	Hostname string

	// TailscaleService, when non-empty, advertises a Tailscale Service and
	// serves HTTPS on 443 for it. Defaults to "svc:<Service>". Set it to "-"
	// to force a plain node listener instead.
	TailscaleService string

	// Tags must be non-empty in service mode. The tailnet's existing nodes use
	// tag:docker; the auth key must carry the same tag.
	Tags []string

	// AuthKeyPath is a mounted file holding the auth key. Required.
	AuthKeyPath string

	// StateDir must be persistent. Defaults to StateRoot/<Service>.
	StateDir string

	// Port is the node-listener port. Ignored in service mode (443/HTTPS).
	Port int

	// Logf, when nil, discards tsnet's own logging.
	Logf func(string, ...any)
}

// Listen is the short form: a canonical service name, a mounted auth key, a
// state dir. Everything else takes its canonical default — a Tailscale Service
// named svc:<service>, tag:docker, HTTPS on 443.
//
// Pass an empty stateDir to accept StateRoot/<service>.
func Listen(ctx context.Context, service ServiceName, authKeyPath, stateDir string) (net.Listener, *tsnet.Server, error) {
	return ListenWith(ctx, Config{
		Service:     service,
		AuthKeyPath: authKeyPath,
		StateDir:    stateDir,
		Tags:        []string{"tag:docker"},
	})
}

// ListenWith is the full form. On success the caller owns the returned server
// and must Close it; the listener is closed by the server.
func ListenWith(ctx context.Context, cfg Config) (net.Listener, *tsnet.Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(string(cfg.Service))
	if name == "" {
		return nil, nil, errors.New("tsnetlisten: Service (canonical component name) is required")
	}
	if strings.TrimSpace(cfg.AuthKeyPath) == "" {
		return nil, nil, fmt.Errorf("tsnetlisten: %s: AuthKeyPath is required (a mounted Tailscale auth key file)", name)
	}

	hostname := strings.TrimSpace(cfg.Hostname)
	if hostname == "" {
		hostname = name
	}
	stateDir := strings.TrimSpace(cfg.StateDir)
	if stateDir == "" {
		stateDir = path.Join(StateRoot, name)
	}
	service := strings.TrimSpace(cfg.TailscaleService)
	if service == "" {
		service = "svc:" + name
	}
	if service == "-" {
		service = ""
	}
	if service != "" && !strings.HasPrefix(service, "svc:") {
		return nil, nil, fmt.Errorf("tsnetlisten: %s: TailscaleService %q must start with \"svc:\"", name, service)
	}

	authKey, err := ReadSecretFile(cfg.AuthKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("tsnetlisten: %s: %w", name, err)
	}
	if authKey == "" {
		return nil, nil, fmt.Errorf("tsnetlisten: %s: auth key file %s is empty", name, cfg.AuthKeyPath)
	}

	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("tsnetlisten: %s: create tsnet state dir: %w", name, err)
	}

	logf := cfg.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	srv := &tsnet.Server{
		Hostname: hostname,
		AuthKey:  authKey,
		Dir:      stateDir,
		Logf:     logf,
	}
	for _, tag := range cfg.Tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			srv.AdvertiseTags = append(srv.AdvertiseTags, trimmed)
		}
	}

	// Start explicitly so a registration failure surfaces as a startup failure
	// rather than as a confusing listen failure. Without an auth key tsnet
	// would print an authentication URL here instead of failing.
	if err := srv.Start(); err != nil {
		_ = srv.Close()
		return nil, nil, fmt.Errorf("tsnetlisten: %s: tsnet start: %w", name, err)
	}

	if service != "" {
		if len(srv.AdvertiseTags) == 0 {
			_ = srv.Close()
			return nil, nil, fmt.Errorf("tsnetlisten: %s: Tailscale Service %q requires at least one tag: services need a tag-based identity (D-134)", name, service)
		}
		listener, err := srv.ListenService(service, tsnet.ServiceModeHTTP{HTTPS: true, Port: 443})
		if err != nil {
			_ = srv.Close()
			return nil, nil, fmt.Errorf("tsnetlisten: %s: tsnet listen service %q: %w", name, service, err)
		}
		return listener, srv, nil
	}

	port := cfg.Port
	if port <= 0 {
		port = 443
	}
	listener, err := srv.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		_ = srv.Close()
		return nil, nil, fmt.Errorf("tsnetlisten: %s: tsnet listen: %w", name, err)
	}
	return listener, srv, nil
}

// Describe renders a log-safe address for a listener produced by this package.
func Describe(listener net.Listener) string {
	if listener == nil {
		return "<nil>"
	}
	if svc, ok := listener.(*tsnet.ServiceListener); ok {
		return "https://" + svc.FQDN
	}
	return "tsnet:" + listener.Addr().String()
}

// Enabled reports the rollout flag (D-127; named removal condition above).
//
// The default is FALSE — today's proven bind — so that pushing this code can
// never silently move a live service onto a tailnet identity whose auth key
// and state directory have not been prepared on the host. Each deployment
// turns it on explicitly, one service at a time.
func Enabled() bool { return EnvBool(EnabledEnv, false) }

// EnvBool parses a boolean environment variable, falling back to def when the
// variable is unset, empty, or unparseable.
func EnvBool(name string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	}
	return def
}

// Hostname resolves the tsnet hostname for a service: TSNET_HOSTNAME when set,
// otherwise the canonical component name.
func Hostname(service ServiceName) string {
	if v := strings.TrimSpace(os.Getenv(HostnameEnv)); v != "" {
		return v
	}
	return string(service)
}

// FromEnv builds a Config for one service from the shared TSNET_* variables,
// applying the canonical default for everything the deployment does not set.
// It does NOT read the rollout flag; callers check Enabled first so that the
// legacy branch stays reachable and testable (D-127).
func FromEnv(service ServiceName) Config {
	cfg := Config{
		Service:          service,
		Hostname:         Hostname(service),
		TailscaleService: strings.TrimSpace(os.Getenv(ServiceEnv)),
		AuthKeyPath:      strings.TrimSpace(os.Getenv(AuthKeyEnv)),
		StateDir:         strings.TrimSpace(os.Getenv(StateDirEnv)),
		Tags:             []string{"tag:docker"},
	}
	if tags := strings.TrimSpace(os.Getenv(TagsEnv)); tags != "" {
		cfg.Tags = strings.Split(tags, ",")
	}
	return cfg
}

// ErrConflictingBind is returned by GuardExclusiveBind.
var ErrConflictingBind = errors.New("tsnetlisten: a tsnet listener and a host bind were both requested")

// GuardExclusiveBind is the fail-closed check.
//
// "Bind to TS only" is not satisfied by a service that ALSO holds a host
// socket, so a process that has a tsnet listener and a configured host bind
// address is a configuration defect and must not start. allowBoth exists only
// for a deliberate, logged migration window; nothing sets it today.
func GuardExclusiveBind(tsnetEnabled bool, hostBindAddr string, allowBoth bool) error {
	addr := strings.TrimSpace(hostBindAddr)
	if !tsnetEnabled || addr == "" || allowBoth {
		return nil
	}
	return fmt.Errorf("%w: host bind %q is configured while %s is on — unset the host bind, or the service is not tailnet-only", ErrConflictingBind, addr, EnabledEnv)
}

// ReadSecretFile reads a mounted secret. The VALUE is never returned to a log:
// callers report presence and length only, per the platform's secret rule.
func ReadSecretFile(filePath string) (string, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	return strings.TrimRight(string(raw), "\r\n"), nil
}
