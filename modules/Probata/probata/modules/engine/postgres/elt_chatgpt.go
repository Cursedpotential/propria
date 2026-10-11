// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	_ "embed"
	"errors"
	"strings"
)

//go:embed elt_templates/chatgpt_json_array_v2.sql
var chatGPTNativeTemplate string

// legacyChatGPTQueryFormat is internal query dispatch, never a source format.
const legacyChatGPTQueryFormat = "chatgpt_json_array_legacy"

// chatGPTNativeQuery renders lossless native ChatGPT mapping extraction.
// Input is a DuckDB source locator; output is a five-column SQL stream containing
// canonical JSON and explicit raw status. No I/O or writes occur here. Pick it
// only for a persisted v2 selection; legacy selections retain their v1 query.
func chatGPTNativeQuery(sourceURL string) (string, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return "", errors.New("native ChatGPT query requires a source locator")
	}
	return strings.ReplaceAll(chatGPTNativeTemplate, "{{SOURCE}}", sqlLiteral(sourceURL)), nil
}
