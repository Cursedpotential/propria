// Byline: Codex | GPT-6.1-sol | 2026-10-07
package surrealsink

import (
	"encoding/json"
	"fmt"
	"strings"
)

func contextGraphProjectionPlan(b ContextGraphBundle) (string, map[string]any, error) {
	if e := b.Validate(); e != nil {
		return "", nil, e
	}
	b = graphCanonical(b)
	base := graphBase(b)
	ns, es := graphTables(b)
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
	return sql.String(), vars, nil
}

func contextGraphRPCBytes(sql string, vars map[string]any) (int, error) {
	raw, e := json.Marshal(rpcRequest{ID: 1, Method: "query", Params: []any{sql, vars}})
	return len(raw), e
}

// ContextGraphPreflight reports the exact full serialized projection RPC size.
// Inputs are derived plan coordinates; output contains counts/hashes/size only.
// No effects occur; pick to inspect batch admission without exposing source bodies.
type ContextGraphPreflight struct {
	RequestBytes int    `json:"request_bytes"`
	LimitBytes   int    `json:"limit_bytes"`
	Fits         bool   `json:"fits"`
	BundleHash   string `json:"bundle_hash"`
	Nodes        int    `json:"nodes"`
	Edges        int    `json:"edges"`
}

// PreflightContextGraph measures the exact SQL, envelope and bound-row request before writing.
// Input is the validated retained bundle; output is exact serialized request bytes,
// limit/admission, bundle hash and counts. Effects are none: no credentials, HTTP,
// schema or database access. Pick before batching/projection; false Fits requires
// rebatching all source units, never truncating them or raising the RPC limit.
func PreflightContextGraph(b ContextGraphBundle) (ContextGraphPreflight, error) {
	sql, vars, e := contextGraphProjectionPlan(b)
	if e != nil {
		return ContextGraphPreflight{}, e
	}
	n, e := contextGraphRPCBytes(sql, vars)
	if e != nil {
		return ContextGraphPreflight{}, e
	}
	return ContextGraphPreflight{RequestBytes: n, LimitBytes: maxBody, Fits: n <= maxBody, BundleHash: graphBase(b).BundleHash, Nodes: len(b.Nodes), Edges: len(b.Edges)}, nil
}
