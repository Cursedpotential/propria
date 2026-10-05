// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodecsRetainMarkdownAndTypedPrivateRecordWithoutFormatOverwrite(t *testing.T) {
	_, _, back := fixtureService(t)
	op := back.claim.Operation
	raw := back.payload
	requireNoError(t, ValidatePayload(op, raw))
	record := json.RawMessage(`{"id":"source:fixture","created_at":"2026-10-05T12:00:00Z","body":"All synthetic private body","unknown_context":{"minor":"Synthetic full identity","notes":["complete private note"]}}`)
	value := RecordExport{Codec: RecordCodec, RecordID: op.RecordID, RecordVersion: op.RecordVersion, Record: record, Types: map[string]string{"/record/id": "record", "/record/created_at": "datetime"}}
	encoded, err := json.Marshal(value)
	requireNoError(t, err)
	op.CodecVersion = RecordCodec
	op.Key = strings.TrimSuffix(op.Key, ".md") + ".json"
	op.ContentType = "application/json"
	op.PayloadSize = int64(len(encoded))
	op.PayloadSHA256 = digest(encoded)
	requireNoError(t, ValidatePayload(op, encoded))
	if !strings.Contains(string(encoded), "Synthetic full identity") || !strings.Contains(string(encoded), "complete private note") {
		t.Fatal("private unknown fields lost")
	}
	for _, ext := range []string{".md", ".pdf"} {
		op.Key = strings.TrimSuffix(op.Key, ".json") + ext
		if ValidatePayload(op, encoded) == nil {
			t.Fatal("JSON format-changing overwrite admitted")
		}
		op.Key = strings.TrimSuffix(op.Key, ext) + ".json"
	}
}

func TestCodecRejectsUnboundAndMalformedPayloads(t *testing.T) {
	for _, kind := range []string{"hash", "size", "utf8", "wrong-media", "wrong-extension", "unknown-codec", "trailing-json", "wrong-record", "missing-types", "absent-type-path", "bad-pointer"} {
		t.Run(kind, func(t *testing.T) {
			_, _, back := fixtureService(t)
			op := back.claim.Operation
			raw := append([]byte(nil), back.payload...)
			if kind == "trailing-json" || kind == "wrong-record" || kind == "missing-types" || kind == "absent-type-path" || kind == "bad-pointer" {
				value := RecordExport{Codec: RecordCodec, RecordID: op.RecordID, RecordVersion: op.RecordVersion, Record: json.RawMessage(`{"id":"source:fixture"}`), Types: map[string]string{"/record/id": "record"}}
				switch kind {
				case "wrong-record":
					value.RecordID = "source:other"
				case "missing-types":
					value.Types = nil
				case "absent-type-path":
					value.Types["/record/missing"] = "datetime"
				case "bad-pointer":
					value.Types["/record/bad~3path"] = "datetime"
				}
				raw, _ = json.Marshal(value)
				if kind == "trailing-json" {
					raw = append(raw, []byte(` {}`)...)
				}
				op.Key = strings.TrimSuffix(op.Key, ".md") + ".json"
				op.CodecVersion = RecordCodec
				op.ContentType = "application/json"
			}
			switch kind {
			case "utf8":
				raw = []byte{0xff}
			case "wrong-media":
				op.ContentType = "application/pdf"
			case "wrong-extension":
				op.Key = strings.TrimSuffix(op.Key, ".md") + ".pdf"
			case "unknown-codec":
				op.CodecVersion = "invented"
			}
			op.PayloadSize = int64(len(raw))
			op.PayloadSHA256 = digest(raw)
			if kind == "hash" {
				op.PayloadSHA256 = digest([]byte("different"))
			}
			if kind == "size" {
				op.PayloadSize++
			}
			if ValidatePayload(op, raw) == nil {
				t.Fatalf("%s payload accepted", kind)
			}
		})
	}
}
