// Byline: Codex · GPT-6.1-sol · 2026-10-07
package surrealsink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// AnalysisConfigFromEnv loads explicit fct/analysis DATABASE credentials from mounted files.
// Inputs are ANALYSIS_SURREAL_URL/NAMESPACE/DATABASE/AUTH_LEVEL/USER_FILE/PASSWORD_FILE.
// Output is existing Client configuration; effects only read mounted secret files.
// Pick for analytical projection; no case defaults, root login or ambient identity is accepted.
func AnalysisConfigFromEnv() (Config, error) {
	c := Config{URL: strings.TrimRight(strings.TrimSpace(os.Getenv("ANALYSIS_SURREAL_URL")), "/"), Namespace: os.Getenv("ANALYSIS_SURREAL_NAMESPACE"), Database: os.Getenv("ANALYSIS_SURREAL_DATABASE"), AuthLevel: os.Getenv("ANALYSIS_SURREAL_AUTH_LEVEL")}
	var e error
	c.User, e = readSecret(os.Getenv("ANALYSIS_SURREAL_USER_FILE"))
	if e != nil {
		return Config{}, errors.New("analysis mounted user credential unavailable")
	}
	c.Password, e = readSecret(os.Getenv("ANALYSIS_SURREAL_PASSWORD_FILE"))
	if e != nil {
		return Config{}, errors.New("analysis mounted password credential unavailable")
	}
	return c, validateAnalysisConfig(c)
}

func validateAnalysisConfig(c Config) error {
	if c.Validate() != nil || c.Namespace != "fct" || c.Database != "analysis" || c.AuthLevel != "database" {
		return errors.New("explicit fct/analysis DATABASE identity required")
	}
	u, e := url.Parse(c.URL)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("invalid analysis endpoint")
	}
	return nil
}

// NewAnalysis constructs the existing Query client with bounded nonredirecting HTTP transport.
// Inputs are explicit analytical config; output is a Client. No network effects occur.
// Pick for projector/CLI use; mounted credentials are loaded by AnalysisConfigFromEnv.
func NewAnalysis(c Config) (*Client, error) {
	if e := validateAnalysisConfig(c); e != nil {
		return nil, e
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return New(c, &http.Client{Timeout: 60 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
}

type graphRow struct {
	MatterID            string             `json:"matter_id"`
	CaseID              string             `json:"case_id"`
	AccessPolicyID      string             `json:"access_policy_id"`
	CreatedByService    string             `json:"created_by_service"`
	GenerationID        string             `json:"generation_id"`
	GenerationKey       string             `json:"generation_key"`
	BundleHash          string             `json:"bundle_hash"`
	ExtractionRunRef    string             `json:"extraction_run_ref"`
	Active              bool               `json:"active"`
	NodeID              string             `json:"node_id,omitempty"`
	EdgeID              string             `json:"edge_id,omitempty"`
	Kind                string             `json:"kind,omitempty"`
	DerivativeKind      string             `json:"derivative_kind"`
	SourceOrigin        string             `json:"source_origin"`
	SourceAvailableFrom *time.Time         `json:"source_available_from,omitempty"`
	OccurredAt          *time.Time         `json:"occurred_at,omitempty"`
	TimePrecision       string             `json:"time_precision"`
	RelativeAnchorRef   string             `json:"relative_anchor_ref"`
	FromNodeID          string             `json:"from_node_id,omitempty"`
	ToNodeID            string             `json:"to_node_id,omitempty"`
	SourcePins          []ContextSourcePin `json:"source_pins,omitempty"`
	PayloadJSON         string             `json:"payload_json,omitempty"`
	NodeCount           int                `json:"node_count,omitempty"`
	EdgeCount           int                `json:"edge_count,omitempty"`
	CheckpointRef       string             `json:"checkpoint_ref,omitempty"`
	NodeTables          []string           `json:"node_tables,omitempty"`
	EdgeTables          []string           `json:"edge_tables,omitempty"`
	FromRecordRef       string             `json:"from_record_ref,omitempty"`
	ToRecordRef         string             `json:"to_record_ref,omitempty"`
}

// ContextGraphReceipt is a reference-only verified generation checkpoint.
// Inputs are read-back counts/hashes; output contains no case body. No I/O occurs.
// Pick for CLI/Activity results after complete record and relation readback.
type ContextGraphReceipt struct {
	GenerationRef string `json:"generation_ref"`
	GenerationID  string `json:"generation_id"`
	BundleHash    string `json:"bundle_hash"`
	CheckpointRef string `json:"checkpoint_ref"`
	NodeCount     int    `json:"node_count"`
	EdgeCount     int    `json:"edge_count"`
	Active        bool   `json:"active"`
}

func graphBase(b ContextGraphBundle) graphRow {
	return graphRow{MatterID: b.Scope.MatterID, CaseID: b.Scope.CaseID, AccessPolicyID: b.Scope.AccessPolicyID, CreatedByService: b.Scope.CreatedByService, GenerationID: b.GenerationID, GenerationKey: "cg_" + graphHash([]any{b.Scope, b.GenerationID}), BundleHash: graphHash(graphCanonical(b)), ExtractionRunRef: b.ExtractionRunRef, Active: true}
}
func graphRowMap(r graphRow, generation bool, edge bool) map[string]any {
	raw, _ := json.Marshal(r)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	if generation {
		for _, k := range []string{"derivative_kind", "source_origin", "time_precision", "relative_anchor_ref"} {
			delete(m, k)
		}
	} else if edge {
		for _, k := range []string{"derivative_kind", "source_origin", "time_precision", "relative_anchor_ref"} {
			delete(m, k)
		}
	}
	return m
}
func graphNodeRow(base graphRow, n ContextGraphNode) graphRow {
	raw, _ := json.Marshal(n)
	base.NodeID = n.NodeID
	base.Kind = n.Kind
	base.DerivativeKind = n.DerivativeKind
	base.SourceOrigin = n.SourceOrigin
	base.SourceAvailableFrom = n.SourceAvailableFrom
	base.OccurredAt = n.OccurredAt
	base.TimePrecision = n.TimePrecision
	base.RelativeAnchorRef = n.RelativeAnchorRef
	base.SourcePins = n.SourcePins
	base.PayloadJSON = string(raw)
	return base
}
func graphEdgeRow(base graphRow, e ContextGraphEdge) graphRow {
	raw, _ := json.Marshal(e)
	base.EdgeID = e.EdgeID
	base.Kind = e.Kind
	base.FromNodeID = e.FromNodeID
	base.ToNodeID = e.ToNodeID
	base.SourcePins = e.SourcePins
	base.PayloadJSON = string(raw)
	return base
}
func graphRecordKey(base graphRow, id string) string {
	return "cg_" + graphHash([]string{base.GenerationKey, id})
}
func graphVariables(s ContextGraphScope, generation string) map[string]any {
	return map[string]any{"matter": s.MatterID, "case": s.CaseID, "policy": s.AccessPolicyID, "service": s.CreatedByService, "generation_key": "cg_" + graphHash([]any{s, generation})}
}

const graphWhere = `generation_key = $generation_key AND matter_id = $matter AND case_id = $case AND access_policy_id = $policy AND created_by_service = $service`

func (c *Client) graphQuery(ctx context.Context, sql string, vars map[string]any) ([]json.RawMessage, error) {
	if e := validateAnalysisConfig(c.cfg); e != nil {
		return nil, e
	}
	rows, e := c.Query(ctx, sql, vars)
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("analysis graph query failed")
	}
	return rows, nil
}

func (c *Client) graphMetadata(ctx context.Context, tables []string) error {
	results, e := c.graphQuery(ctx, "INFO FOR DB;", nil)
	if e != nil {
		return e
	}
	if len(results) != 1 {
		return errors.New("analysis metadata result missing")
	}
	var info struct {
		Tables map[string]string `json:"tables"`
	}
	if json.Unmarshal(results[0], &info) != nil {
		return errors.New("analysis metadata invalid")
	}
	var sql strings.Builder
	for _, t := range tables {
		definition, ok := info.Tables[t]
		if !ok || !strings.Contains(definition, "SCHEMAFULL") {
			return errors.New("analysis graph schema not admitted")
		}
		if graphAllowed(t, contextEdgeKinds) && !strings.Contains(definition, "TYPE RELATION") {
			return errors.New("analysis relation table not admitted")
		}
		fmt.Fprintf(&sql, "INFO FOR TABLE %s;\n", t)
	}
	results, e = c.graphQuery(ctx, sql.String(), nil)
	if e != nil {
		return e
	}
	if len(results) != len(tables) {
		return errors.New("analysis table metadata incomplete")
	}
	for i, t := range tables {
		var table struct {
			Fields map[string]string `json:"fields"`
		}
		if json.Unmarshal(results[i], &table) != nil {
			return errors.New("analysis field metadata invalid")
		}
		for name, typ := range graphFields(t) {
			definition, ok := table.Fields[name]
			if !ok || !strings.Contains(definition, "TYPE "+typ) {
				return errors.New("analysis graph field schema differs")
			}
		}
	}
	return nil
}

// ProjectContextGraph atomically projects one admitted real extraction bundle using Client.Query.
// Input is a bounded complete bundle; output is a reference-only verified checkpoint.
// Effects create only existing admitted tables under deterministic scoped IDs; identical
// retries read back, changed or inactive generations fail. Pick downstream of ingestion/model
// extraction; this unit performs neither and never creates schema or promotes evidence.
func (c *Client) ProjectContextGraph(ctx context.Context, b ContextGraphBundle) (ContextGraphReceipt, error) {
	if e := b.Validate(); e != nil {
		return ContextGraphReceipt{}, e
	}
	b = graphCanonical(b)
	base := graphBase(b)
	ns, es := graphTables(b)
	tables := append([]string{contextGenerationTable}, append(ns, es...)...)
	if e := c.graphMetadata(ctx, tables); e != nil {
		return ContextGraphReceipt{}, e
	}
	vars := graphVariables(b.Scope, b.GenerationID)
	base.NodeCount = len(b.Nodes)
	base.EdgeCount = len(b.Edges)
	base.NodeTables = ns
	base.EdgeTables = es
	base.CheckpointRef = "context-graph:" + base.BundleHash
	vars["generation"] = graphRowMap(base, true, false)
	var sql strings.Builder
	sql.WriteString("BEGIN TRANSACTION;\nLET $old = (SELECT * FROM type::record('ana_projection_generation', $generation_key))[0];\nIF $old != NONE AND ($old.bundle_hash != $generation.bundle_hash OR $old.active != true) { THROW 'immutable graph generation conflict'; };\nIF $old = NONE {\n")
	nodes := map[string]ContextGraphNode{}
	for i, n := range b.Nodes {
		nodes[n.NodeID] = n
		row := graphNodeRow(graphBase(b), n)
		key := fmt.Sprintf("n%d", i)
		vars[key] = graphRowMap(row, false, false)
		vars[key+"_key"] = graphRecordKey(base, n.NodeID)
		fmt.Fprintf(&sql, "CREATE type::record('%s', $%s_key) CONTENT object::extend($%s, {source_available_from: <option<datetime>> $%s.source_available_from, occurred_at: <option<datetime>> $%s.occurred_at});\n", n.Kind, key, key, key, key)
	}
	for i, e := range b.Edges {
		key := fmt.Sprintf("e%d", i)
		vars[key] = graphRowMap(graphEdgeRow(graphBase(b), e), false, true)
		vars[key+"_key"] = graphRecordKey(base, e.EdgeID)
		vars[key+"_from"] = graphRecordKey(base, e.FromNodeID)
		vars[key+"_to"] = graphRecordKey(base, e.ToNodeID)
		fmt.Fprintf(&sql, "RELATE type::record('%s', $%s_from)->type::record('%s', $%s_key)->type::record('%s', $%s_to) CONTENT $%s;\n", nodes[e.FromNodeID].Kind, key, e.Kind, key, nodes[e.ToNodeID].Kind, key, key)
	}
	sql.WriteString("CREATE type::record('ana_projection_generation', $generation_key) CONTENT $generation;\n};\nCOMMIT TRANSACTION;")
	if _, e := c.graphQuery(ctx, sql.String(), vars); e != nil {
		return ContextGraphReceipt{}, e
	}
	saved, e := c.readGraph(ctx, b.Scope, b.GenerationID, true)
	if e != nil {
		return ContextGraphReceipt{}, e
	}
	if saved.Receipt.BundleHash != base.BundleHash || graphHash(graphCanonical(saved.Bundle)) != base.BundleHash {
		return ContextGraphReceipt{}, errors.New("complete graph readback differs")
	}
	return saved.Receipt, nil
}

// ContextGraphReadback returns the source-cited nodes and actual relation endpoints.
// Inputs are persisted records; output is a bounded typed private graph view, with
// no effects. Pick for downstream traversal; emit only Receipt into Temporal history.
type ContextGraphReadback struct {
	Receipt ContextGraphReceipt `json:"receipt"`
	Bundle  ContextGraphBundle  `json:"bundle"`
}

func (c *Client) readGraph(ctx context.Context, scope ContextGraphScope, generation string, active bool) (ContextGraphReadback, error) {
	out := ContextGraphReadback{}
	for _, id := range []string{scope.MatterID, scope.CaseID, scope.AccessPolicyID, scope.CreatedByService, generation} {
		if !graphIdentifier(id) {
			return out, errors.New("explicit graph read scope required")
		}
	}
	vars := graphVariables(scope, generation)
	results, e := c.graphQuery(ctx, "SELECT * FROM ana_projection_generation WHERE "+graphWhere+" LIMIT 2;", vars)
	if e != nil {
		return out, e
	}
	if len(results) != 1 {
		return out, errors.New("generation readback missing")
	}
	var generations []graphRow
	if json.Unmarshal(results[0], &generations) != nil || len(generations) != 1 {
		return out, errors.New("scoped graph generation not found")
	}
	g := generations[0]
	if active && !g.Active {
		return out, errors.New("graph generation inactive")
	}
	if g.GenerationKey != vars["generation_key"] || g.GenerationID != generation || g.MatterID != scope.MatterID || g.CaseID != scope.CaseID || g.AccessPolicyID != scope.AccessPolicyID || g.CreatedByService != scope.CreatedByService {
		return out, errors.New("generation scope mismatch")
	}
	if len(g.NodeTables) == 0 || len(g.NodeTables) > len(contextNodeKinds) || len(g.EdgeTables) == 0 || len(g.EdgeTables) > len(contextEdgeKinds) {
		return out, errors.New("generation table manifest invalid")
	}
	seen := map[string]bool{}
	tables := append(append([]string{}, g.NodeTables...), g.EdgeTables...)
	var sql strings.Builder
	for _, t := range tables {
		if seen[t] || (!graphAllowed(t, contextNodeKinds) && !graphAllowed(t, contextEdgeKinds)) {
			return out, errors.New("generation table allowlist mismatch")
		}
		seen[t] = true
		if graphAllowed(t, contextEdgeKinds) {
			fmt.Fprintf(&sql, "SELECT *, type::string(in) AS from_record_ref, type::string(out) AS to_record_ref FROM %s WHERE %s LIMIT 257;\n", t, graphWhere)
		} else {
			fmt.Fprintf(&sql, "SELECT * FROM %s WHERE %s LIMIT 257;\n", t, graphWhere)
		}
	}
	results, e = c.graphQuery(ctx, sql.String(), vars)
	if e != nil {
		return out, e
	}
	if len(results) != len(tables) {
		return out, errors.New("graph readback result incomplete")
	}
	out.Bundle = ContextGraphBundle{Scope: scope, GenerationID: generation, ExtractionRunRef: g.ExtractionRunRef}
	actualLinks := map[string][2]string{}
	for i, raw := range results {
		var rows []graphRow
		if json.Unmarshal(raw, &rows) != nil || len(rows) > 256 {
			return out, errors.New("graph readback bound exceeded")
		}
		for _, r := range rows {
			if r.GenerationKey != g.GenerationKey || r.BundleHash != g.BundleHash || r.Active != g.Active || r.MatterID != scope.MatterID || r.CaseID != scope.CaseID || r.AccessPolicyID != scope.AccessPolicyID || r.CreatedByService != scope.CreatedByService {
				return out, errors.New("graph row scope/checkpoint mismatch")
			}
			if graphAllowed(tables[i], contextNodeKinds) {
				var n ContextGraphNode
				if json.Unmarshal([]byte(r.PayloadJSON), &n) != nil || n.NodeID != r.NodeID || n.Kind != tables[i] || graphHash(n.SourcePins) != graphHash(r.SourcePins) {
					return out, errors.New("source-cited node readback differs")
				}
				exact, _ := json.Marshal(n)
				if string(exact) != r.PayloadJSON {
					return out, errors.New("persisted node payload is not exact typed JSON")
				}
				if n.DerivativeKind != r.DerivativeKind || n.SourceOrigin != r.SourceOrigin || !graphTimesEqual(n.SourceAvailableFrom, r.SourceAvailableFrom) || !graphTimesEqual(n.OccurredAt, r.OccurredAt) || n.TimePrecision != r.TimePrecision || n.RelativeAnchorRef != r.RelativeAnchorRef {
					return out, errors.New("indexed source node fields differ")
				}
				out.Bundle.Nodes = append(out.Bundle.Nodes, n)
			} else {
				var edge ContextGraphEdge
				if json.Unmarshal([]byte(r.PayloadJSON), &edge) != nil || edge.EdgeID != r.EdgeID || edge.Kind != tables[i] || edge.FromNodeID != r.FromNodeID || edge.ToNodeID != r.ToNodeID || graphHash(edge.SourcePins) != graphHash(r.SourcePins) {
					return out, errors.New("source-cited relation readback differs")
				}
				exact, _ := json.Marshal(edge)
				if string(exact) != r.PayloadJSON {
					return out, errors.New("persisted relation payload is not exact typed JSON")
				}
				out.Bundle.Edges = append(out.Bundle.Edges, edge)
				actualLinks[edge.EdgeID] = [2]string{r.FromRecordRef, r.ToRecordRef}
			}
		}
	}
	if e := out.Bundle.Validate(); e != nil {
		return out, errors.New("persisted graph contract invalid")
	}
	nodeKinds := map[string]string{}
	for _, node := range out.Bundle.Nodes {
		nodeKinds[node.NodeID] = node.Kind
	}
	for _, edge := range out.Bundle.Edges {
		expected := [2]string{nodeKinds[edge.FromNodeID] + ":" + graphRecordKey(g, edge.FromNodeID), nodeKinds[edge.ToNodeID] + ":" + graphRecordKey(g, edge.ToNodeID)}
		if actualLinks[edge.EdgeID] != expected {
			return out, errors.New("actual relation links differ from cited endpoints")
		}
	}
	ns, es := graphTables(out.Bundle)
	if len(out.Bundle.Nodes) != g.NodeCount || len(out.Bundle.Edges) != g.EdgeCount || graphHash(graphCanonical(out.Bundle)) != g.BundleHash || g.CheckpointRef != "context-graph:"+g.BundleHash || graphHash(ns) != graphHash(g.NodeTables) || graphHash(es) != graphHash(g.EdgeTables) {
		return out, errors.New("graph count/hash readback differs")
	}
	out.Receipt = ContextGraphReceipt{GenerationRef: contextGenerationTable + ":" + g.GenerationKey, GenerationID: generation, BundleHash: g.BundleHash, CheckpointRef: g.CheckpointRef, NodeCount: g.NodeCount, EdgeCount: g.EdgeCount, Active: g.Active}
	return out, nil
}

func graphTimesEqual(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}

// TraverseContextGraph follows actual source-cited relations from one persisted node.
// Inputs are admitted scope/generation, node ID and one/two-hop bound; output is
// a private source-cited subgraph. Effects are read-only complete generation checks.
// Pick for bounded downstream investigation; inactive generations cannot traverse.
func (c *Client) TraverseContextGraph(ctx context.Context, scope ContextGraphScope, generation, nodeID string, hops int) (ContextGraphReadback, error) {
	if !graphIdentifier(nodeID) || hops < 1 || hops > 2 {
		return ContextGraphReadback{}, errors.New("bounded traversal node and hops required")
	}
	view, e := c.readGraph(ctx, scope, generation, true)
	if e != nil {
		return view, e
	}
	found := false
	for _, n := range view.Bundle.Nodes {
		if n.NodeID == nodeID {
			found = true
		}
	}
	if !found {
		return ContextGraphReadback{}, errors.New("traversal node not found")
	}
	reached := map[string]bool{nodeID: true}
	selected := map[string]bool{}
	for step := 0; step < hops; step++ {
		next := map[string]bool{}
		for _, edge := range view.Bundle.Edges {
			if reached[edge.FromNodeID] || reached[edge.ToNodeID] {
				next[edge.FromNodeID] = true
				next[edge.ToNodeID] = true
				selected[edge.EdgeID] = true
			}
		}
		for id := range next {
			reached[id] = true
		}
	}
	nodes := []ContextGraphNode{}
	edges := []ContextGraphEdge{}
	for _, n := range view.Bundle.Nodes {
		if reached[n.NodeID] {
			nodes = append(nodes, n)
		}
	}
	for _, edge := range view.Bundle.Edges {
		if selected[edge.EdgeID] {
			edges = append(edges, edge)
		}
	}
	view.Bundle.Nodes = nodes
	view.Bundle.Edges = edges
	return view, nil
}

// DeactivateContextGraph reversibly excludes a complete generation without deleting history.
// Inputs are admitted scope and generation; output is verified inactive checkpoint.
// Effects atomically flip only that generation's node/relation/ledger active flags.
// Pick for withdrawal/correction; projection replay cannot reactivate the generation.
func (c *Client) DeactivateContextGraph(ctx context.Context, scope ContextGraphScope, generation string) (ContextGraphReceipt, error) {
	view, e := c.readGraph(ctx, scope, generation, false)
	if e != nil {
		return ContextGraphReceipt{}, e
	}
	tables := append([]string{contextGenerationTable}, view.bundleTables()...)
	sort.Strings(tables)
	vars := graphVariables(scope, generation)
	var sql strings.Builder
	sql.WriteString("BEGIN TRANSACTION;\n")
	for _, t := range tables {
		fmt.Fprintf(&sql, "UPDATE %s SET active = false WHERE %s;\n", t, graphWhere)
	}
	sql.WriteString("COMMIT TRANSACTION;")
	if _, e := c.graphQuery(ctx, sql.String(), vars); e != nil {
		return ContextGraphReceipt{}, e
	}
	saved, e := c.readGraph(ctx, scope, generation, false)
	if e != nil {
		return ContextGraphReceipt{}, e
	}
	if saved.Receipt.Active || saved.Receipt.BundleHash != view.Receipt.BundleHash {
		return ContextGraphReceipt{}, errors.New("inactive graph readback differs")
	}
	return saved.Receipt, nil
}

func (v ContextGraphReadback) bundleTables() []string {
	ns, es := graphTables(v.Bundle)
	return append(ns, es...)
}
