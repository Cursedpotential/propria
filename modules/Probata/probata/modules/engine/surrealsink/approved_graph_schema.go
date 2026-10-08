// Byline: Codex · GPT-6 · 2026-10-07
package surrealsink

import (
	"fmt"
	"sort"
	"strings"
)

var approvedClaimFields = map[string]string{
	"approved_revision_id": "string", "approval_digest": "string", "control_generation_id": "string", "approved_at": "datetime", "approved_by": "string",
	"candidate_id": "string", "candidate_sha256": "string", "record_id": "string", "record_sha256": "string",
	"source_id": "string", "source_version_id": "string", "source_object_id": "string",
	"source_object_sha256": "string", "source_object_uri": "string", "claim_text": "string",
}

// ApprovedGraphSchema generates the smallest additive typed Surreal schema for owner-approved assertions.
// Inputs: none. Outputs: reviewable fct/analysis DDL text. Effects: none.
// Pick only after checking existing table metadata; application requires a separate owner-controlled release.
func ApprovedGraphSchema() string {
	var out strings.Builder
	out.WriteString("-- Byline: Codex · GPT-6 · 2026-10-07\n-- Additive fct/analysis approved assertion schema; review before applying.\n")
	for _, table := range []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"} {
		fmt.Fprintf(&out, "DEFINE TABLE IF NOT EXISTS %s TYPE NORMAL SCHEMAFULL PERMISSIONS NONE;\n", table)
		fields := graphFields("ctx_statement")
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE %s TYPE %s;\n", key, table, fields[key])
		}
		keys = make([]string, 0, len(approvedClaimFields))
		for key := range approvedClaimFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE %s TYPE %s;\n", key, table, approvedClaimFields[key])
		}
		fmt.Fprintf(&out, "DEFINE INDEX IF NOT EXISTS approved_scope ON TABLE %s FIELDS matter_id, case_id, approved_revision_id, source_available_from;\n", table)
	}
	out.WriteString("DEFINE TABLE IF NOT EXISTS ctx_approved_source_version TYPE NORMAL SCHEMAFULL PERMISSIONS NONE;\n")
	for _, field := range []string{"matter_id", "case_id", "source_id", "source_version_id", "source_object_id", "source_object_sha256", "source_object_uri"} {
		fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE ctx_approved_source_version TYPE string;\n", field)
	}
	out.WriteString("DEFINE INDEX IF NOT EXISTS approved_source_scope ON TABLE ctx_approved_source_version FIELDS matter_id, case_id, source_version_id;\n")
	out.WriteString("DEFINE TABLE IF NOT EXISTS ctx_approved_record TYPE NORMAL SCHEMAFULL PERMISSIONS NONE;\n")
	for _, field := range []string{"matter_id", "case_id", "source_version_id", "record_id", "record_sha256", "source_available_from", "occurred_at"} {
		typ := "string"
		if field == "source_available_from" {
			typ = "option<datetime>"
		}
		if field == "occurred_at" {
			typ = "option<datetime>"
		}
		fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE ctx_approved_record TYPE %s;\n", field, typ)
	}
	out.WriteString("DEFINE INDEX IF NOT EXISTS approved_record_scope ON TABLE ctx_approved_record FIELDS matter_id, case_id, source_version_id, record_id;\n")
	out.WriteString("DEFINE TABLE IF NOT EXISTS ctx_approved_provenance TYPE RELATION SCHEMAFULL PERMISSIONS NONE;\n")
	out.WriteString("DEFINE FIELD IF NOT EXISTS in ON TABLE ctx_approved_provenance TYPE record;\nDEFINE FIELD IF NOT EXISTS out ON TABLE ctx_approved_provenance TYPE record;\n")
	for _, field := range []string{"matter_id", "case_id", "approved_revision_id", "approval_digest", "candidate_id", "record_id", "record_sha256", "source_version_id"} {
		fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE ctx_approved_provenance TYPE string;\n", field)
	}
	out.WriteString("DEFINE INDEX IF NOT EXISTS approved_provenance_scope ON TABLE ctx_approved_provenance FIELDS matter_id, case_id, approved_revision_id;\n")
	out.WriteString("DEFINE TABLE IF NOT EXISTS ctx_approved_record_source TYPE RELATION SCHEMAFULL PERMISSIONS NONE;\n")
	out.WriteString("DEFINE FIELD IF NOT EXISTS in ON TABLE ctx_approved_record_source TYPE record;\nDEFINE FIELD IF NOT EXISTS out ON TABLE ctx_approved_record_source TYPE record;\n")
	for _, field := range []string{"matter_id", "case_id", "source_version_id", "record_id", "record_sha256"} {
		fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE ctx_approved_record_source TYPE string;\n", field)
	}
	out.WriteString("DEFINE TABLE IF NOT EXISTS ana_approved_projection TYPE NORMAL SCHEMAFULL PERMISSIONS NONE;\n")
	for _, field := range []string{"matter_id", "case_id", "approved_revision_id", "approval_digest", "control_generation_id", "claims_hash", "node_set_hash", "access_policy_id"} {
		fmt.Fprintf(&out, "DEFINE FIELD IF NOT EXISTS %s ON TABLE ana_approved_projection TYPE string;\n", field)
	}
	out.WriteString("DEFINE FIELD IF NOT EXISTS claim_count ON TABLE ana_approved_projection TYPE int;\nDEFINE FIELD IF NOT EXISTS completed_at ON TABLE ana_approved_projection TYPE datetime;\n")
	return out.String()
}
