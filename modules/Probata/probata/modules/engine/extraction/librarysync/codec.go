// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"path"
	"strings"
	"unicode/utf8"
)

const (
	MarkdownCodec = "markdown-utf8/1"
	RecordCodec   = "surreal-record-json/1"
)

// RecordExport preserves full JSON values plus native Surreal scalar type annotations.
// Inputs: same-transaction immutable typed snapshot encoded by the parent. Outputs: an opaque companion export.
// Effects: none. Types uses JSON pointers into this envelope, for example /record/id and /record/created_at.
// Parent retains the original typed snapshot, seals bytes once before claim, and rejects unsupported native-type encoding.
type RecordExport struct {
	Codec         string            `json:"codec_version"`
	RecordID      string            `json:"record_id"`
	RecordVersion string            `json:"record_version"`
	Record        json.RawMessage   `json:"record"`
	Types         map[string]string `json:"types"`
}

// ValidatePayload refuses format-changing exports and verifies the opaque immutable payload contract.
// Inputs: operation and exact full bytes. Outputs: nil only for UTF-8 Markdown or a bound complete-record JSON companion.
// Effects: none; no parser/model or field removal. Choose before retaining or writing an app export.
func ValidatePayload(op Operation, raw []byte) error {
	if int64(len(raw)) != op.PayloadSize || digest(raw) != op.PayloadSHA256 || !utf8.Valid(raw) {
		return errors.New("immutable export payload integrity/UTF-8 mismatch")
	}
	media, params, err := mime.ParseMediaType(op.ContentType)
	if err != nil {
		return errors.New("invalid export content type")
	}
	if len(params) > 1 || (len(params) == 1 && !strings.EqualFold(params["charset"], "utf-8")) {
		return errors.New("unsupported export media parameters")
	}
	switch op.CodecVersion {
	case MarkdownCodec:
		if path.Ext(op.Key) != ".md" || media != "text/markdown" {
			return errors.New("Markdown exports require the actual .md body destination")
		}
	case RecordCodec:
		if path.Ext(op.Key) != ".json" || media != "application/json" {
			return errors.New("structured record exports require an explicit .json companion")
		}
		var value RecordExport
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&value) != nil {
			return errors.New("record companion envelope invalid")
		}
		var tail any
		if d.Decode(&tail) != io.EOF {
			return errors.New("record companion trailing data")
		}
		if value.Codec != RecordCodec || value.RecordID != op.RecordID || value.RecordVersion != op.RecordVersion || value.Types == nil {
			return errors.New("record companion identity or type annotations missing")
		}
		var record map[string]json.RawMessage
		if json.Unmarshal(value.Record, &record) != nil || record == nil {
			return errors.New("record companion must preserve a complete record object")
		}
		for pointer, native := range value.Types {
			if !strings.HasPrefix(pointer, "/record/") || native == "" || len(native) > 80 || strings.ContainsAny(native, "\r\n\x00") {
				return errors.New("record native-type annotation invalid")
			}
			if !annotationExists(value.Record, strings.TrimPrefix(pointer, "/record")) {
				return errors.New("record type annotation points to absent value")
			}
		}
	default:
		return errors.New("unsupported immutable export codec")
	}
	return nil
}

func annotationExists(raw json.RawMessage, pointer string) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	for _, part := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		// RFC 6901 permits only ~0 and ~1 escapes; reject malformed annotations.
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 == len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return false
				}
				i++
			}
		}
		key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch node := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = node[key]
			if !ok {
				return false
			}
		case []any:
			if key == "" || (len(key) > 1 && key[0] == '0') {
				return false
			}
			index := 0
			for _, c := range key {
				if c < '0' || c > '9' {
					return false
				}
				index = index*10 + int(c-'0')
				if index >= len(node) {
					return false
				}
			}
			if index >= len(node) {
				return false
			}
			value = node[index]
		default:
			return false
		}
	}
	return true
}
