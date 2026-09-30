// MCP surface for the tool gateway: ONE tool, `atomic_tools`.
//
// Owner design, 2026-09-28 05:21: expose the atomic tools efficiently — a single tool whose
// description summarises what exists, with a directory below it, so an agent loads detail
// progressively instead of paying for one schema per tool:
//
//	path=""                          the families, with counts and capabilities
//	path="messages"                  that family's tools, one line each
//	path="messages.sms-xml"          the full contract for one tool
//	path=<tool id>, run={source_ref, args}   executes it
//
// It lives HERE rather than in tool-runtime because this component is the agent-facing door
// that enforces the locator contract (D-132): a run names a tool and a LOCATOR, never a host
// path. Runs with a source_ref go through Gateway.Run, which resolves and materializes the
// locator. Tools that take no input (capability probes) run with args only, and "path" is
// refused in args either way.
//
// The directory is generated from the live tool-runtime manifest (the same index GET /tools
// proxies), cached briefly, so what an agent sees cannot drift from what the runtime has.
//
// Access follows the owner's tailnet rule (2026-09-23): on the tailnet, Tailscale is the only
// barrier, so /mcp checks the tailnet peer and asks for no bearer token.
//
// Byline: Claude Code · Opus 5.5 · 2026-09-28.
package toolgateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Cursedpotential/probata/engine/proffer"
)

const (
	// manifestTTL bounds how stale the directory can be. Short, because the manifest is small
	// and a new tool should appear without restarting the gateway.
	manifestTTL = 60 * time.Second
	// previewPerFamily is how many tool names each family shows in the master description.
	previewPerFamily = 6
)

type manifestEntry = map[string]any

// AtomicToolsInput is the single tool's argument shape.
type AtomicToolsInput struct {
	Path string     `json:"path,omitempty" jsonschema:"'' lists families; '<family>' lists its tools; '<family>.<tool>' shows the full contract"`
	Run  *AtomicRun `json:"run,omitempty" jsonschema:"set to execute the tool named by path"`
}

// AtomicRun names what to run it on: a locator, never a host path.
type AtomicRun struct {
	SourceRef string         `json:"source_ref,omitempty" jsonschema:"locator of the input (upload://, r2://, b2://); omit for tools that take no input"`
	Args      map[string]any `json:"args,omitempty" jsonschema:"tool options such as format or sample_limit; never a host path"`
}

// atomicDirectory caches the tool-runtime manifest for manifestTTL.
type atomicDirectory struct {
	index   func() (json.RawMessage, error)
	mu      sync.Mutex
	cached  []manifestEntry
	fetched time.Time
}

func (d *atomicDirectory) manifest() ([]manifestEntry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached != nil && time.Since(d.fetched) < manifestTTL {
		return d.cached, nil
	}
	if d.index == nil {
		return nil, errors.New("tool index is unavailable")
	}
	raw, err := d.index()
	if err != nil {
		if d.cached != nil {
			return d.cached, nil // serve the last good manifest rather than go dark
		}
		return nil, err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("tool index is not a JSON list: %w", err)
	}
	d.cached, d.fetched = entries, time.Now()
	return entries, nil
}

func entryID(e manifestEntry) string {
	s, _ := e["id"].(string)
	return s
}

func familyOf(id string) string {
	fam, _, _ := strings.Cut(id, ".")
	return fam
}

func shortName(id string) string {
	if _, rest, ok := strings.Cut(id, "."); ok {
		return rest
	}
	return id
}

// families groups entries by the family prefix of their id, sorted by family then id.
func families(m []manifestEntry) ([]string, map[string][]manifestEntry) {
	by := map[string][]manifestEntry{}
	for _, e := range m {
		if id := entryID(e); id != "" {
			by[familyOf(id)] = append(by[familyOf(id)], e)
		}
	}
	names := make([]string, 0, len(by))
	for name, tools := range by {
		names = append(names, name)
		sort.Slice(tools, func(i, j int) bool { return entryID(tools[i]) < entryID(tools[j]) })
	}
	sort.Strings(names)
	return names, by
}

func capabilities(tools []manifestEntry) []string {
	seen := map[string]bool{}
	for _, t := range tools {
		c, _ := t["capability"].(string)
		if c == "" {
			c = "unspecified"
		}
		seen[c] = true
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// masterDescription is what an agent reads before calling anything: a brief map of what exists.
func masterDescription(m []manifestEntry) string {
	names, by := families(m)
	var b strings.Builder
	fmt.Fprintf(&b, "Probata's atomic evidence tools: %d tools in %d families (parsers, extractors, "+
		"repair and engine probes). Browse first, then run one.\n", len(m), len(names))
	for _, name := range names {
		tools := by[name]
		preview := make([]string, 0, previewPerFamily)
		for i, t := range tools {
			if i == previewPerFamily {
				break
			}
			preview = append(preview, shortName(entryID(t)))
		}
		more := ""
		if len(tools) > previewPerFamily {
			more = fmt.Sprintf(", +%d more", len(tools)-previewPerFamily)
		}
		fmt.Fprintf(&b, "- %s (%d; %s): %s%s\n", name, len(tools),
			strings.Join(capabilities(tools), ", "), strings.Join(preview, ", "), more)
	}
	b.WriteString("Use: path='' lists families; path='<family>' lists its tools; " +
		"path='<family>.<tool>' shows the full contract; add run={source_ref, args} to execute it. " +
		"source_ref is a locator (upload://, r2://, b2://), never a host path; omit it for tools " +
		"that take no input.")
	return b.String()
}

// browse resolves one level of the directory: root, a family, or a single tool.
func browse(m []manifestEntry, path string) map[string]any {
	path = strings.Trim(strings.TrimSpace(path), "/.")
	names, by := families(m)

	if path == "" {
		fams := make([]map[string]any, 0, len(names))
		for _, name := range names {
			fams = append(fams, map[string]any{
				"family":       name,
				"tools":        len(by[name]),
				"capabilities": capabilities(by[name]),
				"open":         fmt.Sprintf("atomic_tools(path='%s')", name),
			})
		}
		return map[string]any{"level": "root", "tool_count": len(m), "families": fams}
	}

	if tools, ok := by[path]; ok {
		rows := make([]map[string]any, 0, len(tools))
		for _, t := range tools {
			rows = append(rows, map[string]any{
				"id":          entryID(t),
				"description": t["description"],
				"capability":  t["capability"],
				"side_effect": t["side_effect"],
				"formats":     t["formats"],
			})
		}
		return map[string]any{
			"level":  "family",
			"family": path,
			"tools":  rows,
			"open":   "atomic_tools(path='<id>') for the full contract",
		}
	}

	for _, t := range m {
		if entryID(t) == path {
			return map[string]any{
				"level": "tool",
				"tool":  t,
				"run":   fmt.Sprintf("atomic_tools(path='%s', run={source_ref: '<locator>', args: {...}})", path),
			}
		}
	}

	return map[string]any{
		"level":       "not_found",
		"path":        path,
		"suggestions": suggest(m, path),
		"families":    names,
	}
}

func suggest(m []manifestEntry, path string) []string {
	needle := strings.ToLower(path)
	out := []string{}
	for _, t := range m {
		if id := entryID(t); strings.Contains(strings.ToLower(id), needle) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

// runAtomic executes one tool under the locator contract and returns a result map.
func (h *HTTPHandler) runAtomic(ctx context.Context, m []manifestEntry, toolID string, run AtomicRun) map[string]any {
	fail := func(kind string, err error) map[string]any {
		return map[string]any{"ok": false, "tool": toolID, "error": kind, "detail": err.Error()}
	}
	if err := ValidateToolID(toolID); err != nil {
		return fail("invalid_tool_id", err)
	}
	known := false
	for _, t := range m {
		if entryID(t) == toolID {
			known = true
			break
		}
	}
	if !known {
		out := fail("unknown_tool", fmt.Errorf("unknown tool %q", toolID))
		out["suggestions"] = suggest(m, toolID)
		return out
	}
	if _, named := run.Args["path"]; named {
		return fail("contract_rejected", errors.New("callers never name a host path; pass a locator in source_ref"))
	}
	if h.Gateway == nil {
		return fail("gateway_unavailable", errors.New("tool gateway is not configured"))
	}

	var (
		raw json.RawMessage
		err error
	)
	if strings.TrimSpace(run.SourceRef) != "" {
		raw, err = h.Gateway.Run(ctx, toolID, proffer.Ref(run.SourceRef), run.Args)
	} else {
		if h.Gateway.Runner == nil {
			return fail("gateway_unavailable", errors.New("tool runner is not configured"))
		}
		args := run.Args
		if args == nil {
			args = map[string]any{}
		}
		raw, err = h.Gateway.Runner.Run(ctx, toolID, args)
	}
	if err != nil {
		return fail("run_failed", err)
	}
	var result any
	if err := json.Unmarshal(raw, &result); err != nil {
		result = string(raw)
	}
	return map[string]any{"ok": true, "tool": toolID, "result": result}
}

// newAtomicServer builds the one-tool MCP server over the current manifest.
func (h *HTTPHandler) newAtomicServer(dir *atomicDirectory) *mcp.Server {
	m, err := dir.manifest()
	description := ""
	if err != nil {
		description = "Probata's atomic evidence tools are UNAVAILABLE right now: " + err.Error() +
			". Calls return the same error until tool-runtime answers."
	} else {
		description = masterDescription(m)
	}

	srv := mcp.NewServer(
		&mcp.Implementation{Name: "probata-atomic-tools", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: "One tool, atomic_tools. Browse with path, then run with run={source_ref, args}."},
	)
	mcp.AddTool(srv, &mcp.Tool{Name: "atomic_tools", Description: description},
		func(ctx context.Context, _ *mcp.CallToolRequest, in AtomicToolsInput) (*mcp.CallToolResult, any, error) {
			if err != nil {
				return nil, map[string]any{"ok": false, "error": "index_unavailable", "detail": err.Error()}, nil
			}
			if in.Run != nil {
				return nil, h.runAtomic(ctx, m, strings.TrimSpace(in.Path), *in.Run), nil
			}
			return nil, browse(m, in.Path), nil
		})
	return srv
}

// mcpHandler serves /mcp. Stateless: the directory holds no per-client state, so each request
// gets a fresh server over the (briefly cached) manifest and no session id is involved.
// Localhost protection is off because the tsnet listener hands every request over loopback;
// the tailnet peer check in tailnetOnly is the barrier.
func (h *HTTPHandler) mcpHandler() http.Handler {
	dir := &atomicDirectory{index: h.Index}
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return h.newAtomicServer(dir) },
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			DisableLocalhostProtection: true,
			MaxRequestBodyBytes:        maxRequestBytes,
		},
	)
}

// tailnetOnly admits tailnet peers without a bearer token (owner rule, 2026-09-23).
func (h *HTTPHandler) tailnetOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.authorizedTailnetPeer(r) {
			writeError(w, http.StatusUnauthorized, errors.New("tool gateway: tailnet authorization required"))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
