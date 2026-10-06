// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package normalize

import (
	"encoding/json"
	"math/big"
	"strings"
	"time"
)

// nativeObject preserves an entire source-native object in normalized Content.
// Input is raw JSON already validated by the raw contract; output is that JSON
// or an empty object for absent metadata. It performs no I/O. Pick only for the
// persisted AI format branch; human content retains its existing representation.
func nativeObject(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

// nativeAITimestamp interprets an explicit message date without a conversation fallback.
// Inputs are native raw fields and the existing string-date convenience value.
// Outputs are the optional date, unchanged native date evidence and supported
// precision/certainty. No I/O occurs. Pick for persisted AI formats; unknown,
// zero, invalid or unrepresentable dates remain unknown and are preserved.
func nativeAITimestamp(raw RawRecordView, fallback string) (*time.Time, string, TimestampGranularity, TimestampCertainty) {
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw.NativeFields, &fields)
	evidence := fallback
	value, present := fields["source_created_at"]
	if present && string(value) != "null" {
		if err := json.Unmarshal(value, &evidence); err != nil {
			evidence = string(value)
		}
	} else if present {
		evidence = ""
	}
	if evidence == "" {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	if parsed, err := time.Parse(time.RFC3339Nano, evidence); err == nil {
		if parsed.IsZero() {
			return nil, evidence, GranularityUnknown, CertaintyUnknown
		}
		if dot := strings.IndexByte(evidence, '.'); dot >= 0 {
			end := dot + 1
			for end < len(evidence) && evidence[end] >= '0' && evidence[end] <= '9' {
				end++
			}
			if end-dot-1 > 9 && strings.Trim(evidence[dot+10:end], "0") != "" {
				return nil, evidence, GranularityUnknown, CertaintyUnknown
			}
		}
		granularity := GranularitySecond
		if strings.Contains(evidence, ".") {
			granularity = GranularitySubsecond
		}
		return &parsed, evidence, granularity, CertaintyExact
	}
	// Epoch seconds are recognized only at ChatGPT's explicit numeric field.
	if !present || len(value) == 0 || value[0] == '"' {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	var metadata struct {
		Template string `json:"duckdb_template"`
	}
	_ = json.Unmarshal(raw.NativeMetadata, &metadata)
	if metadata.Template != "chatgpt_json_array_v2" {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	rational, ok := new(big.Rat).SetString(evidence)
	if !ok || rational.Sign() <= 0 {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	seconds, remainder := new(big.Int), new(big.Int)
	seconds.QuoRem(rational.Num(), rational.Denom(), remainder)
	if !seconds.IsInt64() {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	nanos, loss := new(big.Int), new(big.Int)
	nanos.QuoRem(new(big.Int).Mul(remainder, big.NewInt(1_000_000_000)), rational.Denom(), loss)
	if loss.Sign() != 0 {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	parsed := time.Unix(seconds.Int64(), nanos.Int64()).UTC()
	if parsed.Year() < 1 || parsed.Year() > 9999 {
		return nil, evidence, GranularityUnknown, CertaintyUnknown
	}
	granularity := GranularitySecond
	if remainder.Sign() != 0 {
		granularity = GranularitySubsecond
	}
	return &parsed, evidence, granularity, CertaintyExact
}
