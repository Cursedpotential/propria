// Package sourceformat identifies source bytes and separately decides their admission eligibility.
package sourceformat

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ReadLimit is the maximum retained prefix inspected by the detector (8 MiB).
const ReadLimit int64 = 8 << 20

var transcriptSignatureLine = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{1,2}:\d{2}(?::\d{2})?\s*(?:AM|PM)?\]\s*[^:]+:\s*$`)
var imessageSignatureLine = regexp.MustCompile(`(?i)^(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{1,2}, \d{4}\s+\d{1,2}:\d{2}(?::\d{2})?\s*[AP]M(?:\s*\([^\r\n]*\))?\s*$`)

// SignatureHead bounds a retained read to complete lines when a source exceeds its read limit.
// Inputs: at most limit+1 bytes and a positive byte limit. Output: an unchanged or shortened view.
// Side effects: none. Pick this before Detect when the caller used a bounded prefix read.
func SignatureHead(head []byte, limit int64) []byte {
	if limit < 1 {
		return nil
	}
	if int64(len(head)) <= limit {
		return head
	}
	head = head[:limit]
	if cut := bytes.LastIndexByte(head, '\n'); cut > 0 {
		return head[:cut+1]
	}
	return head
}

// Detect identifies a bounded source prefix from its bytes, never its filename or declared format.
// Inputs: retained bytes; callers limit file reads with ReadLimit. Outputs: format ID, signature ID, error.
// Side effects: none. Pick this for both durable handler recommendation and pre-admission source plans.
func Detect(head []byte) (format, signatureKind string, err error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(head, []byte{0xef, 0xbb, 0xbf}))
	if len(trimmed) == 0 {
		return "", "", errors.New("retained source is empty")
	}
	if bytes.HasPrefix(trimmed, []byte("%PDF-")) {
		return "pdf", "pdf_header_v1", nil
	}
	if isZIPContent(trimmed) {
		if bytes.Contains(trimmed, []byte("[Content_Types].xml")) && bytes.Contains(trimmed, []byte("word/")) {
			return "docx", "office_open_xml_word_package_v1", nil
		}
		return "archive", "zip_container_v1", nil
	}
	if isArchiveContent(trimmed) {
		return "archive", "archive_magic_v1", nil
	}
	if trimmed[0] == '<' {
		if format, kind, ok := detectHTMLContent(trimmed); ok {
			return format, kind, nil
		}
		decoder := xml.NewDecoder(bytes.NewReader(trimmed))
		for {
			token, tokenErr := decoder.Token()
			if tokenErr != nil {
				return "xml", "xml_prefix_v1", nil
			}
			if start, ok := token.(xml.StartElement); ok {
				switch strings.ToLower(start.Name.Local) {
				case "smses":
					return "smsbackuprestore_xml", "sms_backup_restore_smses_root_v1", nil
				case "calls":
					return "callsbackuprestore_xml", "sms_backup_restore_calls_root_v1", nil
				default:
					return "xml", "xml_root_v1", nil
				}
			}
		}
	}
	if trimmed[0] == '[' {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		opening, openingErr := decoder.Token()
		if openingErr == nil && opening == json.Delim('[') && decoder.More() {
			var conversation json.RawMessage
			var first map[string]json.RawMessage
			if decoder.Decode(&conversation) == nil && json.Unmarshal(conversation, &first) == nil && chatGPTConversationSignature(first) {
				return "chatgpt_official_json", "chatgpt_official_conversations_array_v1", nil
			}
			if claudeConversationSignature(first) {
				return "claude_ai_export_json", "claude_ai_conversations_array_v1", nil
			}
		}
	}
	if trimmed[0] == '{' {
		var first map[string]json.RawMessage
		if json.Unmarshal(trimmed, &first) == nil && claudeConversationSignature(first) {
			return "claude_ai_export_json", "claude_ai_conversation_object_v1", nil
		}
	}
	if trimmed[0] == '{' && facebookMessengerThreadSignature(trimmed) {
		return "facebook_messenger_json", "facebook_messenger_thread_json_v1", nil
	}
	if detectedJSONLines(trimmed) {
		return "ndjson", "newline_delimited_json_v1", nil
	}
	if !utf8.Valid(trimmed) || bytes.ContainsRune(trimmed, '\x00') {
		return "binary", "opaque_binary_v1", nil
	}
	if signature := codeSignature(trimmed); signature != "" {
		return "text", signature, nil
	}
	if format, signature := detectAIMarkdown(trimmed); format != "" {
		return format, signature, nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	scanner.Buffer(make([]byte, 4096), int(ReadLimit)+1)
	lines := make([]string, 0, 64)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
		if transcriptSignatureLine.MatchString(line) {
			return "messages_transcript", "bracketed_message_transcript_v1", nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("inspect retained transcript signature: %w", err)
	}
	for index, line := range lines {
		if !imessageSignatureLine.MatchString(line) {
			continue
		}
		nonblank := 0
		for next := index + 1; next < len(lines) && next <= index+4; next++ {
			if lines[next] != "" {
				nonblank++
			}
		}
		if nonblank >= 2 {
			return "messages_transcript", "apple_messages_timestamp_sender_body_v1", nil
		}
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return "json", "json_container_prefix_v1", nil
	}
	if detectedCSV(trimmed) {
		return "csv", "delimited_rows_v1", nil
	}
	if utf8.Valid(trimmed) && !bytes.ContainsRune(trimmed, '\x00') {
		return "text", "utf8_text_v1", nil
	}
	return "binary", "opaque_binary_v1", nil
}

// isZIPContent checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func isZIPContent(content []byte) bool {
	return bytes.HasPrefix(content, []byte{'P', 'K', 0x03, 0x04}) ||
		bytes.HasPrefix(content, []byte{'P', 'K', 0x05, 0x06}) ||
		bytes.HasPrefix(content, []byte{'P', 'K', 0x07, 0x08})
}

// isArchiveContent checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func isArchiveContent(content []byte) bool {
	return bytes.HasPrefix(content, []byte{0x1f, 0x8b}) ||
		bytes.HasPrefix(content, []byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}) ||
		bytes.HasPrefix(content, []byte("Rar!\x1a\x07")) ||
		(len(content) > 262 && string(content[257:262]) == "ustar")
}

// detectedJSONLines checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func detectedJSONLines(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	values := 0
	var first []byte
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if !json.Valid(line) {
			return false
		}
		if values == 0 {
			first = append([]byte(nil), line...)
		}
		values++
	}
	if scanner.Err() != nil {
		return false
	}
	// A derived SMS thread chunk with exactly one message is one line; it is
	// still a newline-delimited thread file, not a JSON document. Only that
	// derive/smsthreads line shape is accepted on its own, so a minified
	// one-line JSON document keeps its own signature.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02 (live: 105-chunk backup, one-message threads failed as "json")
	return values >= 2 || (values == 1 && smsThreadsLine(first))
}

// smsThreadsLine checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func smsThreadsLine(line []byte) bool {
	var fields struct {
		Thread    *string `json:"thread"`
		SourcePos *string `json:"source_pos"`
		Kind      *string `json:"kind"`
	}
	return json.Unmarshal(line, &fields) == nil && fields.Thread != nil && fields.SourcePos != nil && fields.Kind != nil
}

// detectedCSV checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func detectedCSV(content []byte) bool {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = 0
	records := 0
	columns := 0
	for records < 8 {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return false
		}
		if records == 0 {
			columns = len(record)
			if columns < 2 {
				return false
			}
		} else if len(record) != columns {
			return false
		}
		records++
	}
	return records >= 2
}

// facebookMessengerThreadSignature recognizes one thread file of a Facebook
// "Download your information" export: a JSON object whose first member is
// "participants" and whose head carries the messages[] entry keys
// sender_name and timestamp_ms. Only the head is read, so the document is not
// decoded whole. Byline: Claude Code · Opus 5.5 · 2026-10-02
// facebookMessengerThreadSignature checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func facebookMessengerThreadSignature(head []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(head))
	if opening, err := decoder.Token(); err != nil || opening != json.Delim('{') {
		return false
	}
	if key, err := decoder.Token(); err != nil || key != "participants" {
		return false
	}
	for _, marker := range []string{`"messages"`, `"sender_name"`, `"timestamp_ms"`} {
		if !bytes.Contains(head, []byte(marker)) {
			return false
		}
	}
	return true
}

// chatGPTConversationSignature checks the retained content signature for its named format.
// Inputs: bounded bytes or decoded JSON fields. Output: true for a matching structure.
// Side effects: none. Pick this internal check through Detect.
func chatGPTConversationSignature(first map[string]json.RawMessage) bool {
	if first["mapping"] == nil || (first["title"] == nil && first["conversation_id"] == nil && first["id"] == nil) {
		return false
	}
	var mapping map[string]struct {
		Message *struct {
			Author  map[string]json.RawMessage `json:"author"`
			Content map[string]json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(first["mapping"], &mapping) != nil {
		return false
	}
	for _, node := range mapping {
		if node.Message != nil && node.Message.Author != nil && node.Message.Content != nil {
			return true
		}
	}
	return false
}
