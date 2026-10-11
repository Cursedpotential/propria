// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package activities

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/normalize"
	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/Cursedpotential/probata/engine/proffer"
	"io"
	"testing"
)

// TestNormalizeHistoricalReceiptBeforeRecomputation protects successful 1.1 bundles from adapter upgrades.
// Inputs are historical refs or failed replay validation. Output is original
// refs or a closed failure; no raw stream, writer or normalizer runs. Pick for
// Activity retry ordering rather than post-computation persistence conflicts.
func TestNormalizeHistoricalReceiptBeforeRecomputation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		found        bool
		ref, receipt proffer.Ref
		err          error
		want         bool
	}{
		{"historical success", true, "historical-1.1-bundle", "original-receipt", nil, true},
		{"integrity failure", false, "", "", errors.New("digest mismatch"), false},
		{"incomplete receipt", true, "historical-1.1-bundle", "", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeNormalizedPipelineStore{priorFound: tc.found, priorRef: tc.ref, priorReceipt: tc.receipt, priorErr: tc.err}
			result, err := (NormalizedPipelineActivities{Store: store, Normalizer: fakeNormalizer{failErr: errors.New("normalizer must never run")}}).NormalizeGeneration(context.Background(), proffer.StageRequest{RequestID: "workflow:1", SourceVersionRef: "source-version:1", Refs: map[string]proffer.Ref{"raw_generation": "raw-generation:1", "raw_source_verification": "verification:1"}})
			if (err == nil) != tc.want || store.resolveCalls != 0 || store.writerCalls != 0 || store.persistExecSpec.BundleRef != "" {
				t.Fatalf("replay recomputed: %+v %v", store, err)
			}
			if tc.want && (result.Ref != tc.ref || result.ReceiptRef != tc.receipt) {
				t.Fatal("historical refs changed")
			}
		})
	}
}

type nativeNormalizeSource struct{ read bool }

// Next supplies one synthetic native message and then EOF.
// Input is a context; output is fixture data. It advances memory only and is
// picked for fresh-version normalization tests instead of a retained source.
func (s *nativeNormalizeSource) Next(context.Context) (normalize.RawRecordView, error) {
	if s.read {
		return normalize.RawRecordView{}, io.EOF
	}
	s.read = true
	return normalize.RawRecordView{RecordStatus: parser.StatusParsed, FormatID: "claude_ai_export_json", NativeFields: []byte(`{"record_kind":"message","body":"","source_role":"assistant"}`), NativeMetadata: []byte(`{"duckdb_template":"claude_ai_export_json_v1","source_row":{"content":[{"type":"thinking","text":"keep"}]}}`)}, nil
}
// Close satisfies the fixture stream contract without external effects.
// It takes no input and returns nil; pick only for this in-memory source.
func (*nativeNormalizeSource) Close() error { return nil }

// TestFreshNativeNormalizeUsesActualVersion verifies that replay reuse does not conceal new version truth.
// Input is a fresh synthetic native row. Output is a 1.2.0 header/receipt and
// preserved native content without human IDs. Effects are in-memory fixtures.
// Pick for new runs; historical headers remain unchanged in the prior test.
func TestFreshNativeNormalizeUsesActualVersion(t *testing.T) {
	input := baseNormalizerInput()
	input.DeclaredFormat = "claude_ai_export_json"
	input.Records = &nativeNormalizeSource{}
	writer := &fakeBundleWriter{bundleRef: "new-native-bundle"}
	store := &fakeNormalizedPipelineStore{resolveInput: input, openWriter: writer, persistExecRef: "new-native-bundle", persistExecReceipt: "new-receipt"}
	_, err := (NormalizedPipelineActivities{Store: store, Normalizer: normalize.GenericMessageNormalizer{}}).NormalizeGeneration(context.Background(), proffer.StageRequest{RequestID: "workflow:1", SourceVersionRef: "source-version:1", Refs: map[string]proffer.Ref{"raw_generation": "raw-generation:1", "raw_source_verification": "verification:1"}})
	if err != nil || writer.beganHeader.NormalizerVersion != "1.2.0" || store.persistExecSpec.NormalizerVersion != "1.2.0" || len(writer.emitted) != 1 || len(writer.emitted[0].Participants) != 0 {
		t.Fatalf("new version/content truth: %+v %v", writer, err)
	}
}
