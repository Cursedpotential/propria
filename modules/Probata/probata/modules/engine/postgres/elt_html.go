// Byline: Claude Code · Sonnet · 2026-10-02
//
// The two HTML structured-ELT templates. Both read the retained HTML with the
// DuckDB webbed extension and emit the same three-column wire shape as every
// other structured-ELT template (stored_bytes, native_fields, native_metadata).
//
//	facebook_messenger_html_v1  one thread file (message_N.html) of a Facebook
//	                            "Download your information" export: one MESSAGE row
//	                            per message block, in the SMS native_fields shape.
//	generic_html_document_v1    any other HTML file: one OBJECT row per structural
//	                            block (heading, paragraph, list item, table cell...)
//	                            in document order, so it lands on the document path
//	                            as record_type other with the text preserved.
//
// The SQL lives in elt_templates/ as tracked, directly runnable files (a
// DuckDB shell can execute them after substituting the {{...}} markers); this
// file only fills in the markers.
package postgres

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
)

//go:embed elt_templates/facebook_messenger_html_v1.sql
var facebookMessengerHTMLTemplate string

//go:embed elt_templates/generic_html_document_v1.sql
var genericHTMLDocumentTemplate string

func isHTMLFormat(format activities.StructuredELTFormat) bool {
	return format == activities.StructuredELTFormatFacebookMessengerHTML ||
		format == activities.StructuredELTFormatGenericHTML
}

// sqlLiteral escapes a value for use inside a single-quoted DuckDB string.
func sqlLiteral(value string) string { return strings.ReplaceAll(value, "'", "''") }

// facebookExportRoot is the locator prefix the export's relative links resolve
// against. Facebook writes every media href relative to the export root
// (a <base href> in the file), which is the folder above your_facebook_activity/
// (or the older messages/ folder). When no marker is present the thread
// file's own folder is used.
func facebookExportRoot(sourceLocator string) (string, error) {
	slash := strings.LastIndex(sourceLocator, "/")
	if slash < 0 || !strings.Contains(sourceLocator, "://") {
		return "", fmt.Errorf("facebook messenger html source locator %q has no folder", sourceLocator)
	}
	for _, marker := range []string{"/your_facebook_activity/", "/your_activity_across_facebook/", "/messages/"} {
		if index := strings.Index(sourceLocator, marker); index >= 0 {
			return sourceLocator[:index+1], nil
		}
	}
	return sourceLocator[:slash+1], nil
}

func facebookThreadDir(sourceLocator string) string {
	trimmed := sourceLocator[:strings.LastIndex(sourceLocator, "/")]
	return trimmed[strings.LastIndex(trimmed, "/")+1:]
}

func facebookMessengerHTMLQuery(sourceURL, sourceLocator string) (string, error) {
	root, err := facebookExportRoot(sourceLocator)
	if err != nil {
		return "", err
	}
	return strings.NewReplacer(
		"{{SOURCE}}", sqlLiteral(sourceURL),
		"{{EXPORT_ROOT}}", sqlLiteral(root),
		"{{THREAD_DIR}}", sqlLiteral(facebookThreadDir(sourceLocator)),
	).Replace(facebookMessengerHTMLTemplate), nil
}

func genericHTMLDocumentQuery(sourceURL string) (string, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return "", errors.New("structured elt requires a non-empty DuckDB source url")
	}
	return strings.NewReplacer("{{SOURCE}}", sqlLiteral(sourceURL)).Replace(genericHTMLDocumentTemplate), nil
}
