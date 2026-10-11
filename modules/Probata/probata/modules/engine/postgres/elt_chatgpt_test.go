// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// TestChatGPTNativeAllSlotsAndEnvelope exercises actual DuckDB over synthetic native data.
// Input is the retained fixture; output is complete semantic reconstruction and
// status assertions. Side effects are an in-memory DuckDB process only. Pick for
// the v2 template; no temporary directories or production data are used.
func TestChatGPTNativeAllSlotsAndEnvelope(t *testing.T) {
	binary, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("DuckDB CLI absent; VPS native proof required")
	}
	path, err := filepath.Abs(filepath.Join("testdata", "chatgpt-native", "all-shapes.json"))
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var source []map[string]any
	if err := json.Unmarshal(bytes, &source); err != nil {
		t.Fatal(err)
	}
	query, err := chatGPTNativeQuery(filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(binary, "-json", "-c", query).CombinedOutput()
	if err != nil {
		t.Fatalf("native SQL: %v: %s", err, output)
	}
	var rows []struct {
		StoredBytes    string `json:"stored_bytes"`
		NativeFields   string `json:"native_fields"`
		NativeMetadata string `json:"native_metadata"`
		Status         string `json:"record_status"`
		Reason         string `json:"status_reason"`
	}
	if err := json.Unmarshal(output, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 11 {
		t.Fatalf("captured %d rows, want 3 envelopes + 8 mapping slots", len(rows))
	}
	rebuilt := make([]map[string]any, len(source))
	statuses := map[string]int{}
	for _, row := range rows {
		var metadata, fields map[string]any
		var stored any
		if err := json.Unmarshal([]byte(row.NativeMetadata), &metadata); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(row.NativeFields), &fields); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(row.StoredBytes), &stored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored, metadata["source_row"]) {
			t.Fatal("canonical source row semantics changed")
		}
		if metadata["source_byte_offsets_available"] != false || metadata["duckdb_template"] != "chatgpt_json_array_v2" {
			t.Fatal("invalid native provenance")
		}
		index := int(metadata["conversation_index"].(float64))
		if key, ok := metadata["mapping_key"].(string); ok {
			rebuilt[index]["mapping"].(map[string]any)[key] = stored
			if key == "a" && fields["body"] != " first \nsecond\nthird\n\n" {
				t.Fatalf("not all textual parts retained: %#v", fields["body"])
			}
			if key == "orphan" && (fields["body"] != "" || fields["source_role"] != "future_role" || fields["source_created_at"] != nil) {
				t.Fatalf("native tool/unknown role/date changed: %#v", fields)
			}
		} else {
			rebuilt[index] = stored.(map[string]any)
			if _, ok := source[index]["mapping"].(map[string]any); ok {
				rebuilt[index]["mapping"] = map[string]any{}
			}
		}
		statuses[row.Status]++
		if row.Status != "parsed" && row.Reason == "" {
			t.Fatal("nonsemantic row lacks reason")
		}
	}
	if !reflect.DeepEqual(source, rebuilt) {
		t.Fatalf("full native semantics lost: source=%#v reconstructed=%#v", source, rebuilt)
	}
	if statuses["parsed"] != 4 || statuses["envelope"] != 5 || statuses["unknown"] != 1 || statuses["malformed"] != 1 {
		t.Fatalf("raw status accounting=%v", statuses)
	}
}
