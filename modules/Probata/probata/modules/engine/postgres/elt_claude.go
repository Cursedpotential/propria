// Byline: Codex · GPT-6 · 2026-10-03.
package postgres

import (
	_ "embed"
	"errors"
	"strings"
)

//go:embed elt_templates/claude_ai_export_json_v1.sql
var claudeAIExportTemplate string

// claudeAIExportQuery renders the native Claude conversation extraction query.
// Input is a DuckDB-readable source locator; output is SQL with the existing
// stored_bytes/native_fields/native_metadata row shape. It performs no I/O or
// database writes. Pick it for native claude.ai exports with chat_messages;
// Markdown transcripts and other JSON formats use their own handlers.
// Text precedence follows server/tools/parsers/ai_chat/claude_ai_export.py's
// transcripts.claude-ai-export tool; native JSON, nontext messages and roles
// remain preserved without its synthesized owner/Claude participant identities.
// The existing Consignatio comm_timeline_mvp/elt/elt_ai_claude_v1.sql is the SQL
// sibling; this adaptation keeps source timestamps and roles while using the
// engine's source-version-linked bundle instead of inserting catalog ai_turns.
func claudeAIExportQuery(sourceURL string) (string, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return "", errors.New("structured elt requires a non-empty DuckDB source url")
	}
	return strings.ReplaceAll(claudeAIExportTemplate, "{{SOURCE}}", sqlLiteral(sourceURL)), nil
}
