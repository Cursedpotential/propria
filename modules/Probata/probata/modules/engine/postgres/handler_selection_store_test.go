package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/stretchr/testify/require"
)

func TestDetectHandlerContentUsesRetainedBytesAndSeparatesCallsXML(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantFormat string
		wantError  string
	}{
		{name: "sms backup and restore", content: `<?xml version="1.0"?><smses count="1"><sms address="+1"/></smses>`, wantFormat: "smsbackuprestore_xml"},
		{name: "calls backup is not sms", content: `<?xml version="1.0"?><calls count="1"><call number="+1"/></calls>`, wantFormat: "callsbackuprestore_xml"},
		{name: "chatgpt official", content: `[{"title":"Chat","conversation_id":"c-1","mapping":{"node":{"message":{"author":{"role":"user"},"content":{"content_type":"text","parts":["hello"]}}}}}]`, wantFormat: "chatgpt_official_json"},
		{name: "facebook messenger thread", content: `{"participants":[{"name":"A"},{"name":"B"}],"messages":[{"sender_name":"A","timestamp_ms":1,"content":"hi"}]}`, wantFormat: "facebook_messenger_json"},
		{name: "other json object is not messenger", content: `{"messages":[{"sender_name":"A","timestamp_ms":1}],"participants":[]}`, wantFormat: "json"},
		{name: "one-message derived sms thread chunk", content: "{\"thread\":\"t1\",\"source_pos\":\"sms:1\",\"kind\":\"sms\",\"status\":\"parsed\",\"content\":\"hi\"}\n", wantFormat: "ndjson"},
		{name: "one-line json document is not ndjson", content: "{\"a\":1}\n", wantFormat: "json"},
		{name: "message transcript", content: "[2026-09-12 8:04 PM] Matthew Salem:\nhello\n", wantFormat: "messages_transcript"},
		{name: "filename-like text is not an sms signature", content: "sms-backup.xml\nnot actually an export", wantFormat: "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, signature, err := detectHandlerContent([]byte(tt.content))
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				require.Empty(t, format)
				require.Empty(t, signature)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantFormat, format)
			require.NotEmpty(t, signature)
		})
	}
}

func TestDetectHandlerContentKeepsEstablishedFormatsOnDecoderPath(t *testing.T) {
	tests := []struct {
		name       string
		content    []byte
		wantFormat string
	}{
		{name: "pdf", content: []byte("%PDF-1.7\n1 0 obj\n"), wantFormat: "pdf"},
		{name: "docx", content: append([]byte{'P', 'K', 0x03, 0x04}, []byte("[Content_Types].xml\x00word/document.xml")...), wantFormat: "docx"},
		{name: "zip archive", content: append([]byte{'P', 'K', 0x03, 0x04}, []byte("ordinary/member.txt")...), wantFormat: "archive"},
		{name: "ordinary json", content: []byte(`{"kind":"ordinary","items":[1,2]}`), wantFormat: "json"},
		{name: "csv", content: []byte("sender,body\nMatthew,hello\nOther,goodbye\n"), wantFormat: "csv"},
		{name: "ndjson", content: []byte("{\"body\":\"one\"}\n{\"body\":\"two\"}\n"), wantFormat: "ndjson"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, signature, err := detectHandlerContent(tt.content)
			require.NoError(t, err)
			require.Equal(t, tt.wantFormat, format)
			require.NotEmpty(t, signature)
		})
	}
}

// Byline: Claude Code · Opus 5.5 · 2026-10-02 (live: a 34 MB thread chunk was detected as "json")
func TestSignatureHeadDetectsALargeThreadChunkOnWholeLines(t *testing.T) {
	line := `{"thread":"8102959302_8102959303","source_pos":"1","kind":"sms","body":"` + strings.Repeat("x", 90) + `"}` + "\n"
	content := []byte(strings.Repeat(line, 200))
	limit := int64(len(line)*150 + 37) // stops inside line 151
	read := content[:limit+1]          // what LimitReader(limit+1) returns
	format, _, err := detectHandlerContent(read[:limit])
	require.NoError(t, err)
	require.Equal(t, "json", format, "the old cut keeps a broken last line")
	head := signatureHead(read, limit)
	require.Equal(t, byte('\n'), head[len(head)-1])
	format, kind, err := detectHandlerContent(head)
	require.NoError(t, err)
	require.Equal(t, "ndjson", format)
	require.Equal(t, "newline_delimited_json_v1", kind)
	small := []byte(line)
	require.Equal(t, small, signatureHead(small, limit), "a source read whole is not cut")
}

func TestHandlerCandidatesSelectOneSignatureHandlerWithoutDecoderAlternative(t *testing.T) {
	decoder := parser.Capability{ParserID: "sbv_csv", ParserVersion: "1.4.0"}
	structured := handlerCandidatesForDetectedFormat("csv", decoder)
	require.Len(t, structured, 1)
	require.Equal(t, "duckdb", string(structured[0].ExecutionPath))

	for _, detected := range []string{"pdf", "docx", "archive", "json", "text", "binary"} {
		candidates := handlerCandidatesForDetectedFormat(detected, decoder)
		require.Len(t, candidates, 1, detected)
		require.Equal(t, decoder.ParserID, candidates[0].HandlerID, detected)
		require.Equal(t, decoder.ParserVersion, candidates[0].HandlerVersion, detected)
		require.Equal(t, "decoder", string(candidates[0].ExecutionPath), detected)
		require.NotEmpty(t, candidates[0].CompatibilityRef, detected)
	}
}

func TestHandlerDetectedFormatConstraintCoversEveryDetectorOutput(t *testing.T) {
	snapshot, err := os.ReadFile(filepath.Join("..", "..", "..", "sql", "bootstrap", "schema_snapshot_20260907.sql"))
	require.NoError(t, err)
	const constraintName = "CONSTRAINT handler_detected_format_format_id_check"
	start := strings.Index(string(snapshot), constraintName)
	require.NotEqual(t, -1, start)
	constraint := string(snapshot[start:])
	if end := strings.IndexByte(constraint, '\n'); end >= 0 {
		constraint = constraint[:end]
	}
	expected := []string{
		"smsbackuprestore_xml", "chatgpt_official_json", "messages_transcript",
		"pdf", "docx", "archive", "callsbackuprestore_xml", "xml", "ndjson", "json", "csv", "text", "binary",
		"facebook_messenger_json",                          // d1cb113a's detector output; missing from the CHECK until 2026-10-02 (Claude Code · Opus 5.5)
		"facebook_messenger_html", "generic_html_document", // HTML detectors (Claude Code · Sonnet · 2026-10-02)
	}
	for _, format := range expected {
		require.Contains(t, constraint, "'"+format+"'::text", format)
	}
	require.Equal(t, len(expected)*2, strings.Count(constraint, "'"), "detected-format CHECK contains an unexpected or missing value")
}

func TestDetectHandlerContentRecognizesRealAppleMessagesStructure(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "probata_mixed_demo", "imessage-thread.txt"))
	require.NoError(t, err)
	format, signature, err := detectHandlerContent(content)
	require.NoError(t, err)
	require.Equal(t, "messages_transcript", format)
	require.Equal(t, "apple_messages_timestamp_sender_body_v1", signature)
}

func TestDetectHandlerContentStreamsFirstChatGPTConversationWithoutClosingArray(t *testing.T) {
	first := `{"title":"Chat","conversation_id":"c-1","mapping":{"node":{"message":{"author":{"role":"user"},"content":{"content_type":"text","parts":["hello"]}}}}}`
	truncatedLargeArray := "[" + first + `,{"title":"unfinished","mapping":{"node":{"message":{"content":"` + strings.Repeat("x", 1<<20)
	format, signature, err := detectHandlerContent([]byte(truncatedLargeArray))
	require.NoError(t, err)
	require.Equal(t, "chatgpt_official_json", format)
	require.Equal(t, "chatgpt_official_conversations_array_v1", signature)
}

// HTML signatures. Byline: Claude Code · Sonnet · 2026-10-02
func TestDetectHandlerContentRecognizesHTMLFamilies(t *testing.T) {
	card2024 := `<html><head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /><style>._a6-g{x:y}</style></head><body><div class="_a6-g"><div class="_2ph_ _a6-h _a6-i">A</div><div class="_2ph_ _a6-p"><div>hi</div></div><div class="_3-94 _a6-o"><div class="_a72d">Jul 12, 2024 4:55:32pm</div></div></div></body></html>`
	card2025 := `<html><body><main><section class="_3-95 _a6-g"><h2 class="_2ph_ _a6-h _a6-i">A</h2><div class="_2ph_ _a6-p"><div>hi</div></div><footer class="_3-94 _a6-o"><div class="_a72d">Jun 01, 2025 3:31:29 pm</div></footer></section></main></body></html>`
	section := `<html><body><header><h1>Logins and Logouts</h1><p class="_a70f">A history of your logins</p></header><main><section class="_a6-g"><h2 class="_2ph_ _a6-h _a6-i">Login</h2><div class="_2ph_ _a6-p">x</div><footer class="_3-94 _a6-o">t</footer></section></main></body></html>`
	xhtml := `<?xml version="1.0" ?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><body><div>Aug 19, 2025: Me: hi</div></body></html>`
	tests := []struct{ name, content, wantFormat, wantSignature string }{
		{"facebook thread 2024 layout", card2024, "facebook_messenger_html", "facebook_messenger_thread_html_v1"},
		{"facebook thread 2025 layout", card2025, "facebook_messenger_html", "facebook_messenger_thread_html_v1"},
		{"facebook section page shares the card classes but is not a thread", section, "generic_html_document", "html_document_root_v1"},
		{"google voice xhtml behind an xml prolog", xhtml, "generic_html_document", "html_document_root_v1"},
		{"plain doctype page", "<!DOCTYPE html>\n<html lang=\"en\"><head><title>t</title></head><body><p>x</p></body></html>", "generic_html_document", "html_document_root_v1"},
		{"upper-case legacy doctype", "<!DOCTYPE HTML PUBLIC \"-//W3C//DTD HTML 4.01//EN\">\n<HTML><BODY>x</BODY></HTML>", "generic_html_document", "html_document_root_v1"},
		{"sms backup xml is still sms", `<?xml version="1.0"?><smses count="1"><sms address="+1"/></smses>`, "smsbackuprestore_xml", "sms_backup_restore_smses_root_v1"},
		{"non-html xml is still xml", `<?xml version="1.0"?><root><a/></root>`, "xml", "xml_root_v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, signature, err := detectHandlerContent([]byte(tt.content))
			require.NoError(t, err)
			require.Equal(t, tt.wantFormat, format)
			require.Equal(t, tt.wantSignature, signature)
		})
	}
}
