// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"path/filepath"
	"strings"
)

// PolicyID identifies the deterministic source-admission policy, independently of parsing and custody.
const PolicyID = "source_admission_v1"

// Detection carries format and safe signature IDs without any source payload or credential values.
type Detection struct {
	Format     string `json:"detected_format,omitempty"`
	Signature  string `json:"content_signature,omitempty"`
	Credential string `json:"credential_signature,omitempty"`
}

// Decision records admission eligibility; no status authorizes a write, ingest, deletion, or deduplication.
type Decision struct {
	PolicyID string `json:"policy_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

// Inspect produces shared byte-backed format and credential signatures from a bounded prefix.
// Input: retained prefix. Output: safe IDs and detector error. Side effects: none.
// Pick this for source plans; durable recommendations can call Detect when admission is already governed.
func Inspect(head []byte) (Detection, error) {
	format, signature, err := Detect(head)
	return Detection{Format: format, Signature: signature, Credential: CredentialSignature(head)}, err
}

// Admit determines a source occurrence's eligible destination from byte signatures and path class.
// Inputs: occurrence path, byte-derived detection, stat size. Output: stable policy/status/reason.
// Side effects: none. Pick after Inspect; paths affect eligibility only and never establish content format.
func Admit(path string, detection Detection, size int64) Decision {
	decision := Decision{PolicyID: PolicyID}
	switch {
	case size == 0:
		decision.Status = "empty"
		decision.Reason = "source_zero_bytes"
	case detection.Credential != "":
		decision.Status = "excluded_credentials"
		decision.Reason = detection.Credential
	case generatedPath(path):
		decision.Status = "excluded_generated"
		decision.Reason = "generated_path_class_v1"
	case codePath(path) || strings.HasPrefix(detection.Signature, "code_"):
		decision.Status = "excluded_code"
		decision.Reason = "code_source_class_v1"
	case detection.Format == "archive":
		decision.Status = "archive_container"
		decision.Reason = "archive_members_require_separate_inspection"
	case detection.Format == ChatGPTMarkdown || detection.Format == ClaudeMarkdown:
		decision.Status = "index_document"
		decision.Reason = "ai_markdown_parser_route_not_validated_index_only"
	case messageFormat(detection.Format):
		decision.Status = "proffer_message"
		decision.Reason = "recognized_message_export_signature"
	case detection.Format == "text" || detection.Format == "json" || detection.Format == "xml" || detection.Format == "csv" || detection.Format == "ndjson" || detection.Format == "generic_html_document" || detection.Format == "pdf" || detection.Format == "docx":
		decision.Status = "index_document"
		decision.Reason = "document_content_case_bible_index_only"
	default:
		decision.Status = "unsupported"
		decision.Reason = "no_supported_document_or_message_signature"
	}
	return decision
}

// messageFormat tests only formats with explicit message-export signatures.
// Input: detected format ID. Output: message eligibility. Side effects: none.
// Pick inside admission; generic documents never become fabricated messages.
func messageFormat(format string) bool {
	switch format {
	case "chatgpt_official_json", ClaudeAIExportJSON, "messages_transcript", "smsbackuprestore_xml", "callsbackuprestore_xml", "facebook_messenger_json", "facebook_messenger_html":
		return true
	}
	return false
}

// generatedPath tests exact generated/runtime path segments and known compiled artifact extensions.
// Input: occurrence path. Output: exclusion match. Side effects: none.
// Pick for admission eligibility, never format detection or source removal.
func generatedPath(path string) bool {
	for _, segment := range strings.Split(strings.ToLower(strings.ReplaceAll(path, "\\", "/")), "/") {
		switch segment {
		case ".git", "node_modules", "vendor", "dist", "build", ".venv", "venv", "__pycache__", ".cocoindex_code", "to_be_deleted":
			return true
		}
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".lock", ".map", ".pyc", ".pyo", ".exe", ".dll", ".o", ".obj", ".class", ".sqlite", ".sqlite3", ".db":
		return true
	}
	return false
}

// codePath tests language extensions for source code admission eligibility.
// Input: occurrence path. Output: exclusion match. Side effects: none.
// Pick for eligibility after content detection; an export named .go still retains its byte-derived format.
func codePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rs", ".java", ".c", ".cpp", ".h", ".cs", ".sh", ".ps1", ".sql", ".rb", ".php":
		return true
	}
	return false
}
