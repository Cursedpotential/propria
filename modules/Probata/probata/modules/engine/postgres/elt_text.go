// Byline: Claude Code · Opus 5.5 · 2026-10-15
//
// plain_text_document_v1: the structured-ELT template for text that no more
// specific signature claims (detected format "text"). One OBJECT row per
// paragraph in file order, text kept as written; see the tracked SQL in
// elt_templates/. Owner decision 2026-10-15: unclaimed text imports as a plain
// document rather than failing handler selection.
package postgres

import (
	_ "embed"
	"errors"
	"strings"
)

//go:embed elt_templates/plain_text_document_v1.sql
var plainTextDocumentTemplate string

func plainTextDocumentQuery(sourceURL string) (string, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return "", errors.New("structured elt requires a non-empty DuckDB source url")
	}
	return strings.NewReplacer("{{SOURCE}}", sqlLiteral(sourceURL)).Replace(plainTextDocumentTemplate), nil
}
