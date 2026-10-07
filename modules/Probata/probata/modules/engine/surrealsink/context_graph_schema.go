// Byline: Codex · GPT-6.1-sol · 2026-10-07
package surrealsink

import (
	"fmt"
	"sort"
	"strings"
)

const contextGenerationTable = "ana_projection_generation"

var graphCommonFields = map[string]string{
	"matter_id": "string", "case_id": "string", "access_policy_id": "string", "created_by_service": "string",
	"generation_id": "string", "generation_key": "string", "bundle_hash": "string", "extraction_run_ref": "string", "active": "bool",
}
var graphCitationFields = map[string]string{
	"source_pins": "array<object>", "source_pins.*": "object", "source_pins.*.source_id": "string",
	"source_pins.*.source_version_id": "string", "source_pins.*.source_hash": "string", "source_pins.*.locator": "string",
	"source_pins.*.validation_ref": "option<string>", "payload_json": "string", "kind": "string",
}

func graphTables(b ContextGraphBundle) ([]string, []string) {
	nodes, edges := map[string]bool{}, map[string]bool{}
	for _, n := range b.Nodes {
		nodes[n.Kind] = true
	}
	for _, e := range b.Edges {
		edges[e.Kind] = true
	}
	ns, es := []string{}, []string{}
	for k := range nodes {
		ns = append(ns, k)
	}
	for k := range edges {
		es = append(es, k)
	}
	sort.Strings(ns)
	sort.Strings(es)
	return ns, es
}

func graphFields(kind string) map[string]string {
	f := map[string]string{}
	for k, v := range graphCommonFields {
		f[k] = v
	}
	if kind == contextGenerationTable {
		for k, v := range map[string]string{"node_count": "int", "edge_count": "int", "checkpoint_ref": "string", "node_tables": "array<string>", "edge_tables": "array<string>"} {
			f[k] = v
		}
		return f
	}
	for k, v := range graphCitationFields {
		f[k] = v
	}
	if graphAllowed(kind, contextNodeKinds) {
		for k, v := range map[string]string{"node_id": "string", "derivative_kind": "string", "source_origin": "string", "source_available_from": "option<datetime>", "occurred_at": "option<datetime>", "time_precision": "string", "relative_anchor_ref": "string"} {
			f[k] = v
		}
	} else {
		for k, v := range map[string]string{"edge_id": "string", "from_node_id": "string", "to_node_id": "string", "in": "record", "out": "record"} {
			f[k] = v
		}
	}
	return f
}

// MinimalContextGraphSchema generates additive DDL for exactly the bundle's supported families.
// Input is a validated real-output bundle; output is schema text with no body or I/O.
// Effects are none: the root operator reviews/applies it after target metadata checks.
// Pick for absent fct/analysis tables; it never overwrites existing definitions or creates users/databases.
func MinimalContextGraphSchema(b ContextGraphBundle) (string, error) {
	if e := b.Validate(); e != nil {
		return "", e
	}
	ns, es := graphTables(b)
	tables := append([]string{contextGenerationTable}, append(ns, es...)...)
	var out strings.Builder
	out.WriteString("-- Byline: Codex · GPT-6.1-sol · 2026-10-07\n-- Generated from typed surrealsink context graph contracts; review before applying.\n")
	for _, table := range tables {
		typ := "NORMAL"
		if graphAllowed(table, contextEdgeKinds) {
			typ = "RELATION"
		}
		fmt.Fprintf(&out, "DEFINE TABLE IF NOT EXISTS %s TYPE %s SCHEMAFULL PERMISSIONS NONE;\n", table, typ)
		fields := graphFields(table)
		names := []string{}
		for k := range fields {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			assertion := ""
			switch name {
			case "payload_json":
				assertion = " ASSERT string::len($value) <= 65536"
			case "source_pins":
				assertion = " ASSERT array::len($value) > 0 AND array::len($value) <= 64"
			case "node_count":
				assertion = " ASSERT $value > 0 AND $value <= 128"
			case "edge_count":
				assertion = " ASSERT $value > 0 AND $value <= 256"
			}
			fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE %s TYPE %s%s;\n", name, table, fields[name], assertion)
		}
		fmt.Fprintf(&out, "DEFINE INDEX IF NOT EXISTS projection_generation_scope ON TABLE %s FIELDS generation_key, active;\n", table)
		if table != contextGenerationTable {
			fmt.Fprintf(&out, "DEFINE INDEX IF NOT EXISTS projection_source_versions ON TABLE %s FIELDS source_pins.*.source_version_id;\n", table)
		}
	}
	return out.String(), nil
}
