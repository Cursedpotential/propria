// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDetectAIExportSignatures validates native exports and rejects unrelated JSON and prose.
// Input: synthetic fixtures. Output: test results. Side effects: none. Pick for signature regression.
func TestDetectAIExportSignatures(t *testing.T) {
	native := `{"uuid":"conversation-1","name":"Chat","chat_messages":[{"uuid":"message-1","sender":"human","text":"hello"},{"uuid":"message-2","sender":"assistant","content":[{"type":"text","text":"answer"}]}]}`
	tests := []struct{ name, content, format, signature string }{
		{"native claude array", "[" + native + "]", ClaudeAIExportJSON, "claude_ai_conversations_array_v1"},
		{"native claude object", native, ClaudeAIExportJSON, "claude_ai_conversation_object_v1"},
		{"native claude large prefix", "[" + native + `,{"uuid":"later","name":"Chat","chat_messages":[{"text":"` + strings.Repeat("x", 1<<20), ClaudeAIExportJSON, "claude_ai_conversations_array_v1"},
		{"incomplete first conversation", `[{"uuid":"conversation-1","name":"Chat","chat_messages":[`, "json", "json_container_prefix_v1"},
		{"unrelated chat fields", `{"uuid":"x","name":"y","chat_messages":[{"sender":"human"}]}`, "json", "json_container_prefix_v1"},
		{"unknown role only", `{"uuid":"x","name":"y","chat_messages":[{"uuid":"m1","sender":"system","text":"note"}]}`, "json", "json_container_prefix_v1"},
		{"malformed native blocks", `{"uuid":"x","name":"y","chat_messages":[{"uuid":"m1","sender":"human","content":{"text":"note"}}]}`, "json", "json_container_prefix_v1"},
		{"attachment only", `{"uuid":"x","name":"y","chat_messages":[{"uuid":"m1","sender":"human","attachments":[{"file_name":"synthetic.png"}]}]}`, ClaudeAIExportJSON, "claude_ai_conversation_object_v1"},
		{"chatgpt markdown", "# Conversation\n\n## Prompt:\nquestion\n\n## Response:\nanswer\n", ChatGPTMarkdown, "chatgpt_prompt_response_markdown_v1"},
		{"claude markdown", "# Conversation\n\n### Human:\nquestion\n\n### Assistant:\nanswer\n", ClaudeMarkdown, "claude_human_assistant_markdown_v1"},
		{"ordinary prose mentions", "Human and Assistant discuss Prompt and Response formats.", "text", "utf8_text_v1"},
		{"role prose", "Human: our policy\nAssistant: our job", "text", "utf8_text_v1"},
		{"empty markdown turns", "## Prompt:\n\n## Response:\n", "text", "utf8_text_v1"},
		{"single role heading", "## Prompt:\nquestion", "text", "utf8_text_v1"},
		{"code fence example", "# Formats\n```markdown\n## Prompt:\nquestion\n## Response:\nanswer\n```", "text", "utf8_text_v1"},
		{"reversed roles", "## Response:\nanswer\n## Prompt:\nquestion", "text", "utf8_text_v1"},
		{"mixed provider roles", "## Prompt:\nquestion\n## Assistant:\nanswer", "text", "utf8_text_v1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			format, signature, err := Detect([]byte(test.content))
			if err != nil || format != test.format || signature != test.signature {
				t.Fatalf("Detect = %q %q %v; expected %q %q", format, signature, err, test.format, test.signature)
			}
		})
	}
}

// TestClaudeDetectorAcceptsStructuredELTProofFixtures checks the exact synthetic array/single native export fixtures.
// Inputs: retained synthetic JSON fixtures used for the parser's DuckDB proof. Output: matching byte signatures.
// Side effects: read-only fixture I/O. Pick for detector/parser compatibility including unknown roles and nontext blocks.
func TestClaudeDetectorAcceptsStructuredELTProofFixtures(t *testing.T) {
	for _, name := range []string{"claude-array.json", "claude-single.json"} {
		content, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		format, _, err := Detect(content)
		if err != nil || format != ClaudeAIExportJSON {
			t.Fatalf("fixture %s: %s %v", name, format, err)
		}
	}
}

// TestBoundedDetectorPreservesPriorFormats checks bounded reads, binary fallback and oversized lines.
// Input: synthetic retained prefixes. Output: test results. Side effects: none. Pick after detector extraction.
func TestBoundedDetectorPreservesPriorFormats(t *testing.T) {
	tests := []struct {
		content []byte
		format  string
	}{
		{[]byte{0xff, 0x00, 0xfe}, "binary"},
		{[]byte(strings.Repeat("x", 2<<20)), "text"},
		{[]byte("%PDF-1.7\n"), "pdf"},
		{[]byte("PK\x03\x04[Content_Types].xml\x00word/document.xml"), "docx"},
		{[]byte("PK\x03\x04ordinary.txt"), "archive"},
		{[]byte(`<?xml version="1.0"?><smses><sms/></smses>`), "smsbackuprestore_xml"},
		{[]byte(`<?xml version="1.0"?><calls><call/></calls>`), "callsbackuprestore_xml"},
		{[]byte("a,b\n1,2\n"), "csv"},
		{[]byte("{\"a\":1}\n{\"a\":2}\n"), "ndjson"},
	}
	for _, test := range tests {
		format, _, err := Detect(test.content)
		if err != nil || format != test.format {
			t.Fatalf("expected %s, got %s (%v)", test.format, format, err)
		}
	}
	if _, _, err := Detect([]byte(" \n")); err == nil {
		t.Fatal("empty source must be rejected")
	}
	line := `{"thread":"t1","source_pos":"1","kind":"sms","body":"hi"}` + "\n"
	head := SignatureHead([]byte(strings.Repeat(line, 8)), int64(len(line)*4+12))
	format, _, err := Detect(head)
	if err != nil || format != "ndjson" || head[len(head)-1] != '\n' {
		t.Fatalf("bounded NDJSON failed: %s %v", format, err)
	}
}

// TestCredentialAndAdmissionPolicy checks credential precision and document/message separation.
// Input: synthetic values and paths. Output: admission results. Side effects: none. Pick before source planning.
func TestCredentialAndAdmissionPolicy(t *testing.T) {
	tests := []struct{ name, path, content, status string }{
		{"oauth concrete", "notes.json", `{"installed":{"client_id":"543210987654.apps.provider.org","client_secret":"Zxy1234567890Abcd"}}`, "excluded_credentials"},
		{"service account", "notes.json", `{"type":"service_account","private_key":"-----BEGIN PRIVATE KEY-----\nQUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=\n-----END PRIVATE KEY-----","client_email":"robot@project.provider.org"}`, "excluded_credentials"},
		{"private key", "notes.txt", "-----BEGIN PRIVATE KEY-----\nQUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=\n-----END PRIVATE KEY-----", "excluded_credentials"},
		{"assignment actual", "notes.txt", "NVIDIA_NIM_API_KEY=Ab12Cd34Ef56Gh78Ij90", "excluded_credentials"},
		{"json assignment actual", "notes.json", `{"api_key":"Ab12Cd34Ef56Gh78Ij90"}`, "excluded_credentials"},
		{"escaped assignment actual", "notes.json", `{"text":"Configuration\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90"}`, "excluded_credentials"},
		{"browser passwords", "notes.csv", "name,url,username,password\nSite,https://site.test,user,Ab12\n", "excluded_credentials"},
		{"browser empty password", "notes.csv", "name,url,username,password\nSite,https://site.test,user,\n", "index_document"},
		{"ordinary password column", "notes.csv", "topic,password\npolicy,rotation\n", "index_document"},
		{"credential terms in docs", "credentials.md", "An OAuth client_secret and a service_account private_key are sensitive. Never share an API_KEY.", "index_document"},
		{"placeholder", "notes.md", "API_KEY=YOUR_API_KEY_HERE\nPASSWORD=${MY_PASSWORD}", "index_document"},
		{"native code", "extensionless", "package main\nimport \"fmt\"\nfunc main() {}", "excluded_code"},
		{"code example document", "notes.md", "# Example\n```go\npackage main\nimport \"fmt\"\nfunc main() {}\n```", "index_document"},
		{"code suffix", "conversation.go", "## Prompt:\nquestion\n## Response:\nanswer", "excluded_code"},
		{"generated path", "node_modules/conversation.md", "## Prompt:\nquestion\n## Response:\nanswer", "excluded_generated"},
		{"text document", "claude-conversation.txt", "A report about a conversation.", "index_document"},
		{"markdown awaits parser", "notes.md", "## Prompt:\nquestion\n## Response:\nanswer", "index_document"},
		{"message export", "notes.json", `[{"title":"Chat","conversation_id":"c-1","mapping":{"node":{"message":{"author":{"role":"user"},"content":{"content_type":"text","parts":["hello"]}}}}}]`, "proffer_message"},
		{"binary", "unknown.bin", string([]byte{0xff, 0x00}), "unsupported"},
		{"empty", "notes.md", "", "empty"},
		{"archive", "notes.md", "PK\x03\x04member.txt", "archive_container"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			detection, err := Inspect([]byte(test.content))
			if err != nil && test.content != "" {
				t.Fatal(err)
			}
			result := Admit(test.path, detection, int64(len(test.content)))
			if result.Status != test.status || result.PolicyID != PolicyID {
				t.Fatalf("expected %s, got %+v", test.status, result)
			}
			if test.name == "code suffix" && detection.Format != ChatGPTMarkdown {
				t.Fatal("path changed format")
			}
		})
	}
}
