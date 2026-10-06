// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/sourceformat"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// TestClaudeCompleteConversationReconstruction proves every envelope and message survives native extraction.
// Inputs are synthetic mixed/wholly empty exports; output is full semantic
// equality and explicit envelope accounting. Effects are an in-memory DuckDB
// process only. Pick for provider completeness, rather than flattened turn tests.
func TestClaudeCompleteConversationReconstruction(t *testing.T) {
	binary, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("DuckDB absent; server proof required")
	}
	for _, filename := range []string{"array.json", "single.json", "empty.json"} {
		t.Run(filename, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("testdata", "claude-elt", filename))
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			format, _, err := sourceformat.Detect(original)
			if err != nil || format != sourceformat.ClaudeAIExportJSON {
				t.Fatalf("empty/mixed native detection: %s %v", format, err)
			}
			var source any
			if err := json.Unmarshal(original, &source); err != nil {
				t.Fatal(err)
			}
			conversations, array := source.([]any)
			if !array {
				conversations = []any{source}
			}
			query, err := claudeAIExportQuery(filepath.ToSlash(path))
			if err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(binary, "-json", "-c", query).CombinedOutput()
			if err != nil {
				t.Fatalf("native SQL: %v %s", err, output)
			}
			var rows []struct {
				Stored   string `json:"stored_bytes"`
				Metadata string `json:"native_metadata"`
				Status   string `json:"record_status"`
				Reason   string `json:"status_reason"`
			}
			if err := json.Unmarshal(output, &rows); err != nil {
				t.Fatal(err)
			}
			rebuilt := make([]any, len(conversations))
			envelopes, parsed, expectedMessages := 0, 0, 0
			for _, conversation := range conversations {
				expectedMessages += len(conversation.(map[string]any)["chat_messages"].([]any))
			}
			for _, row := range rows {
				var metadata map[string]any
				var stored any
				if err := json.Unmarshal([]byte(row.Metadata), &metadata); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(row.Stored), &stored); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stored, metadata["source_row"]) || metadata["source_byte_offsets_available"] != false {
					t.Fatal("source content/locator changed")
				}
				index := int(metadata["conversation_index"].(float64))
				if row.Status == "envelope" {
					if row.Reason == "" || rebuilt[index] != nil {
						t.Fatal("missing envelope reason or duplicate envelope")
					}
					conv := stored.(map[string]any)
					conv["chat_messages"] = []any{}
					rebuilt[index] = conv
					envelopes++
				} else if row.Status == "parsed" {
					conv := rebuilt[index].(map[string]any)
					conv["chat_messages"] = append(conv["chat_messages"].([]any), stored)
					parsed++
				} else {
					t.Fatalf("unexpected status %s", row.Status)
				}
			}
			if envelopes != len(conversations) || parsed != expectedMessages || !reflect.DeepEqual(rebuilt, conversations) {
				t.Fatalf("conversation loss: envelope=%d parsed=%d expected=%d", envelopes, parsed, expectedMessages)
			}
		})
	}
}
