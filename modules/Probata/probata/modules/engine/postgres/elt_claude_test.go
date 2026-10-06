// Byline: Codex · GPT-6 · 2026-10-03.
package postgres

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceformat"
)

// TestClaudeAIExportQueryUsesNativeMessageShape verifies that exact-format
// dispatch selects the embedded template without synthesizing source identity.
// Input is a quoted storage locator; output is test assertions, with no I/O.
func TestClaudeAIExportQueryUsesNativeMessageShape(t *testing.T) {
	query, err := structuredELTQueryFor(activities.StructuredELTFormatClaudeJSON,
		"s3://fixture/o'brien/claude.json", "r2://fixture/o'brien/claude.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"read_text('s3://fixture/o''brien/claude.json')", "message::VARCHAR AS stored_bytes",
		"AS native_fields", "AS native_metadata", "'duckdb_template', 'claude_ai_export_json_v1'",
		"'$.chat_messages'", "'$.created_at'", "'$.uuid'", "'source_row', message",
		"'participants', json_array()", "'nontext_block_count'", "'attachment_count'",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("Claude query lacks %q", fragment)
		}
	}
	for _, forbidden := range []string{"'owner'", "'claude'", "WHERE length(trim(body)) > 0", "read_csv", "{{SOURCE}}"} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("Claude query contains unexpected %q", forbidden)
		}
	}
	if structuredELTRequiresWebbed(activities.StructuredELTFormatClaudeJSON) {
		t.Fatal("native JSON must not require Webbed")
	}
}

// TestClaudeAIExportQueryRejectsEmptyLocator verifies fail-closed rendering.
// Inputs are empty locators; output is assertions, with no side effects.
func TestClaudeAIExportQueryRejectsEmptyLocator(t *testing.T) {
	for _, locator := range []string{"", "  "} {
		if _, err := claudeAIExportQuery(locator); err == nil {
			t.Fatal("empty Claude source locator was admitted")
		}
	}
}

// TestClaudeDetectionSelectsPinnedStructuredELT verifies the integrated detector
// and template route. Inputs are native single/array exports with empty tool
// text; output is one DuckDB handler and exact template. No database I/O occurs.
func TestClaudeDetectionSelectsPinnedStructuredELT(t *testing.T) {
	conversation := `{"uuid":"conversation-1","name":"Synthetic native export","chat_messages":[{"uuid":"message-1","sender":"human","text":"question"},{"uuid":"message-2","sender":"assistant","text":"","content":[{"type":"tool_use","name":"calculator"}]}]}`
	for _, content := range []string{conversation, "[" + conversation + "]"} {
		detection, err := sourceformat.Inspect([]byte(content))
		if err != nil || detection.Format != sourceformat.ClaudeAIExportJSON || detection.Credential != "" {
			t.Fatalf("native detection failed: %+v, %v", detection, err)
		}
		candidates := handlerCandidatesForDetectedFormat(detection.Format, parser.Capability{})
		if len(candidates) != 1 || candidates[0].ExecutionPath != proffer.HandlerPathDuckDB ||
			candidates[0].HandlerID != activities.StructuredELTParserID || candidates[0].HandlerVersion != activities.StructuredELTParserVersion {
			t.Fatalf("Claude did not select exact structured handler: %+v", candidates)
		}
		format, err := activities.StructuredELTFormatForDeclaredFormat(detection.Format)
		if err != nil {
			t.Fatal(err)
		}
		template, err := activities.StructuredELTTemplateForFormat(format)
		if err != nil || template != "claude_ai_export_json_v1" {
			t.Fatalf("wrong Claude template %q: %v", template, err)
		}
	}
}

// TestClaudeMarkdownHasNoNativeJSONTemplate protects document/message routing.
// Input is a role-heading transcript; output is an index-only policy with no
// native JSON template. It has no I/O or side effects.
func TestClaudeMarkdownHasNoNativeJSONTemplate(t *testing.T) {
	content := []byte("## Human:\nquestion\n\n## Assistant:\nanswer\n")
	detection, err := sourceformat.Inspect(content)
	if err != nil {
		t.Fatal(err)
	}
	decision := sourceformat.Admit("transcript.md", detection, int64(len(content)))
	if detection.Format != sourceformat.ClaudeMarkdown || decision.Status != "index_document" {
		t.Fatalf("Markdown lost index-only routing: %+v %+v", detection, decision)
	}
	if _, err := activities.StructuredELTFormatForDeclaredFormat(detection.Format); err == nil {
		t.Fatal("Markdown was admitted to native Claude JSON extraction")
	}
}

// TestClaudeDetectedTemplateStreamsEveryNativeMessage exercises native fixtures
// through the detector and actual DuckDB SQL. Input is synthetic single/array
// JSON; output is semantic equality and accounting assertions. It only reads
// fixture files and runs in-memory DuckDB; skip when the optional CLI is absent.
// The same fixtures are additionally verified on the VPS, with retained receipts.
func TestClaudeDetectedTemplateStreamsEveryNativeMessage(t *testing.T) {
	binary, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("DuckDB CLI unavailable; native SQL requires the retained VPS proof")
	}
	for _, filename := range []string{"array.json", "single.json"} {
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
				t.Fatalf("native fixture did not reach Claude template: %s, %v", format, err)
			}
			eltFormat, err := activities.StructuredELTFormatForDeclaredFormat(format)
			if err != nil {
				t.Fatal(err)
			}
			query, err := structuredELTQuery(eltFormat, filepath.ToSlash(path))
			if err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command(binary, "-json", "-c", query).CombinedOutput()
			if err != nil {
				t.Fatalf("synthetic Claude SQL failed: %v: %s", err, output)
			}
			var rows []struct {
				StoredBytes    string `json:"stored_bytes"`
				NativeFields   string `json:"native_fields"`
				NativeMetadata string `json:"native_metadata"`
			}
			if err := json.Unmarshal(output, &rows); err != nil {
				t.Fatal(err)
			}
			var conversations []struct {
				Messages []map[string]any `json:"chat_messages"`
			}
			if filename == "single.json" {
				original = append(append([]byte{'['}, original...), ']')
			}
			if err := json.Unmarshal(original, &conversations); err != nil {
				t.Fatal(err)
			}
			var messages []map[string]any
			for _, conversation := range conversations {
				messages = append(messages, conversation.Messages...)
			}
			if len(rows) != len(messages) {
				t.Fatalf("native messages dropped: rows=%d source_messages=%d", len(rows), len(messages))
			}
			empty := 0
			for i, row := range rows {
				var stored, fields, metadata map[string]any
				for value, target := range map[string]*map[string]any{row.StoredBytes: &stored, row.NativeFields: &fields, row.NativeMetadata: &metadata} {
					if err := json.Unmarshal([]byte(value), target); err != nil {
						t.Fatal(err)
					}
				}
				if !reflect.DeepEqual(stored, messages[i]) || !reflect.DeepEqual(metadata["source_row"], messages[i]) {
					t.Fatalf("native JSON semantics changed at message %d", i)
				}
				if fields["message_id"] != messages[i]["uuid"] || fields["sender"] != messages[i]["sender"] ||
					metadata["created_at"] != messages[i]["created_at"] || metadata["duckdb_template"] != "claude_ai_export_json_v1" ||
					metadata["stored_bytes_representation"] != "duckdb_canonical_native_message_json" || metadata["source_byte_offsets_available"] != false {
					t.Fatalf("native provenance changed at message %d: %+v", i, metadata)
				}
				if fields["body"] == "" {
					empty++
				}
				if len(fields["participants"].([]any)) != 0 {
					t.Fatal("participant identity invented")
				}
			}
			if empty != 3 {
				t.Fatalf("bodyless message count=%d, want 3", empty)
			}
		})
	}
}

// TestClaudeDetectorAndTemplateRejectMalformedFixtures checks both boundaries.
// Inputs are synthetic malformed/native-unrelated JSON; output is rejection.
// Side effects are limited to in-memory DuckDB execution and fixture reads.
func TestClaudeDetectorAndTemplateRejectMalformedFixtures(t *testing.T) {
	for _, filename := range []string{"malformed-message.json", "generic-role.json"} {
		path, err := filepath.Abs(filepath.Join("testdata", "claude-elt", filename))
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		format, _, err := sourceformat.Detect(content)
		if err != nil || format == sourceformat.ClaudeAIExportJSON {
			t.Fatalf("unrelated/malformed source admitted as Claude: %s, %v", format, err)
		}
		binary, err := exec.LookPath("duckdb")
		if err != nil {
			continue
		}
		query, err := claudeAIExportQuery(filepath.ToSlash(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := exec.Command(binary, "-json", "-c", query).CombinedOutput(); err == nil {
			t.Fatalf("native Claude SQL accepted malformed source %s", filename)
		}
	}
}
