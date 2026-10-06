// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package normalize

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/parser"
	"reflect"
	"testing"
)

// TestNativeAIContentAndDates verifies lossless Content and source-only date interpretation.
// Inputs are synthetic persisted-format raw rows; outputs are semantic/date
// assertions. No I/O occurs. Pick for AI normalization and human isolation.
func TestNativeAIContentAndDates(t *testing.T) {
	for _, gate := range []string{"declared", "raw"} {
		for _, date := range []string{"1700000000.123456", "null", "0", `"invalid"`, `"0001-01-01T00:00:00Z"`, `"2026-01-01T01:02:03.1234567891Z"`, `"2026-01-01T01:02:03.123Z"`} {
			raw := RawRecordView{RecordOrdinal: 7, RecordStatus: parser.StatusParsed, NativeFields: []byte(`{"record_kind":"message","body":"","source_role":"future_role","source_created_at":` + date + `,"parts":[null,{"thinking":"keep","tool":{"id":"actual-native-id"}}]}`), NativeMetadata: []byte(`{"source_row":{"attachments":[{"id":"native-asset"}],"unknown":null}}`)}
			raw.NativeMetadata = []byte(`{"duckdb_template":"chatgpt_json_array_v2","source_row":{"attachments":[{"id":"native-asset"}],"unknown":null}}`)
			input := baseInput(nil)
			input.SourceProvenanceClass = ProvenanceAcquiredThirdParty
			if gate == "declared" {
				input.DeclaredFormat = "claude_ai_export_json"
			} else {
				raw.FormatID = "chatgpt_official_json"
			}
			record, err := normalizeOne(input, raw, 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := record.Validate(); err != nil {
				t.Fatal(err)
			}
			if len(record.Participants) != 0 || record.Lineage[0].RawRecordOrdinal != 7 || record.SourceAvailableFrom != input.AcquiredAt {
				t.Fatal("invented identity or changed lineage/knowledge time")
			}
			var content, fields, metadata map[string]any
			_ = json.Unmarshal(record.Content, &content)
			_ = json.Unmarshal(raw.NativeFields, &fields)
			_ = json.Unmarshal(raw.NativeMetadata, &metadata)
			if !reflect.DeepEqual(content["native_fields"], fields) || !reflect.DeepEqual(content["native_metadata"], metadata) {
				t.Fatal("native content stripped")
			}
			valid := date == "1700000000.123456" || date == `"2026-01-01T01:02:03.123Z"`
			if (record.OccurredAt != nil) != valid {
				t.Fatalf("date %s inferred or lost: %+v", date, record)
			}
			if valid && (record.TimestampGranularity != GranularitySubsecond || record.TimestampCertainty != CertaintyExact) {
				t.Fatal("fractional source date precision lost")
			}
			if date == "0" && record.OccurredAtRaw != "0" {
				t.Fatal("zero evidence lost")
			}
		}
	}
	for _, format := range []string{"claude_ai_export_json", "ai_generic_json"} {
		input := baseInput(nil)
		input.DeclaredFormat = parser.FormatID(format)
		raw := RawRecordView{RecordStatus: parser.StatusParsed, NativeFields: []byte(`{"record_kind":"message","body":"","source_created_at":1700000000.25}`), NativeMetadata: []byte(`{"duckdb_template":"claude_ai_export_json_v1"}`)}
		record, err := normalizeOne(input, raw, 0)
		if err != nil || record.OccurredAt != nil || record.OccurredAtRaw != "1700000000.25" {
			t.Fatalf("guessed non-ChatGPT epoch: %s %+v %v", format, record, err)
		}
	}
	input := baseInput(nil)
	raw := RawRecordView{RecordStatus: parser.StatusParsed, NativeFields: []byte(`{"body":"human","sender":"actual-person"}`), NativeMetadata: []byte(`{"unknown":"kept raw"}`)}
	record, err := normalizeOne(input, raw, 0)
	if err != nil || string(record.Content) != `{"body":"human"}` || len(record.Participants) != 1 {
		t.Fatalf("human path changed: %+v %v", record, err)
	}
}
