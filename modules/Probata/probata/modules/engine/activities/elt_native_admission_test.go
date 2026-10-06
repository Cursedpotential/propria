// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package activities

import (
	"context"
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/Cursedpotential/probata/engine/proffer"
	"testing"
)

// nativeStructuredELTStageFixture creates source-consistent synthetic native references.
// Input is the actual declared format; outputs are in-memory Activity fixtures.
// It mutates fixtures only. Pick for native admission tests, never real sources.
func nativeStructuredELTStageFixture(declared string) (proffer.StageRequest, *runtimeStore, *runtimeBundleWriter) {
	req, store, writer := structuredELTStageFixture()
	req.DeclaredFormat = declared
	store.selection.DeclaredFormat = parser.FormatID(declared)
	store.input.DeclaredFormat = parser.FormatID(declared)
	return req, store, writer
}

// TestNativeExecutionDeclarationFence rejects native detection under persisted human labels.
// Inputs are synthetic declared formats and durable pins; outputs are rejection
// or successful native emission assertions. No I/O occurs. Pick for direct
// execution bypass defense; legacy v1 execution remains separately tested.
func TestNativeExecutionDeclarationFence(t *testing.T) {
	for _, native := range []struct{ declared, pin string }{{"chatgpt_official_json", "chatgpt_json_array_v2"}, {"claude_ai_export_json", "claude_ai_export_json_v1"}} {
		for _, declared := range []string{"smsbackuprestore_xml", "json", native.declared} {
			req, store, writer := nativeStructuredELTStageFixture(declared)
			store.selection.TemplateID = native.pin
			authorization := structuredELTAuthorization()
			authorization.authorization.DetectedFormat = native.declared
			repo := &fakeStructuredELTRowRepository{reader: &fakeStructuredELTRowReader{rows: []StructuredELTRow{{StoredBytes: []byte(`{}`), NativeFields: json.RawMessage(`{"record_kind":"message","body":""}`), NativeMetadata: json.RawMessage(`{"duckdb_template":"` + native.pin + `"}`)}}}}
			_, err := (StructuredELTActivities{Rows: repo, Store: store, Authorization: authorization}).ExecuteStructuredELT(context.Background(), req)
			if declared == native.declared {
				if err != nil || writer.records != 1 {
					t.Fatalf("valid native rejected: %v", err)
				}
			} else if err == nil || writer.records != 0 || writer.finalizes != 0 || writer.aborts != 0 || store.persistExecCalls != 0 || repo.req.RequestID != "" {
				t.Fatalf("mismatched native reached extraction/write: %v %+v", err, writer)
			}
		}
	}
}
