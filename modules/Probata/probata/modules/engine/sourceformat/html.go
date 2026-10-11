// Byline: Claude Code · Sonnet · 2026-10-02
package sourceformat

import (
	"bytes"
	"regexp"
)

// htmlRoot matches an HTML document root after an optional XML prolog and
// comments: <!DOCTYPE html ...> (any case, public or not) or <html ...>.
// Google Voice exports are XHTML behind an <?xml ...?> prolog and used to be
// detected as bare xml; the DOCTYPE is the cue.
var htmlRoot = regexp.MustCompile(`(?is)\A(?:<\?xml[^>]*\?>\s*|<!--.*?-->\s*)*(?:<!doctype\s+html\b|<html\b)`)

// classToken reports whether a quoted HTML class attribute contains an exact token.
// Inputs: bounded markup and class token. Output: match. Side effects: none.
// Pick inside the shared HTML signature check for either quote style.
func classToken(head []byte, token string) bool {
	quoted := regexp.QuoteMeta(token)
	return regexp.MustCompile(`(?i)class\s*=\s*(?:"[^"]*\b` + quoted + `\b[^"]*"|'[^']*\b` + quoted + `\b[^']*')`).Match(head)
}

// detectHTMLContent recognizes iMessage and Facebook Messenger HTML exports.
// iMessage requires message direction and metadata classes. Facebook
// Messenger threads use the "Download your information" card classes
// (_a6-g card, _a6-h sender, _a6-p body, _a6-o timestamp) WITHOUT the section
// description paragraph (_a70f) that every other Facebook export page
// (logins, search history, friends...) carries under its title. Section pages
// share the card classes, so the cards alone are not a thread signature.
// The signature reads content only; neither the name nor the path is an input.
// Inputs: bounded retained markup. Outputs: format/signature IDs and recognized flag.
// Side effects: none. Pick through Detect for HTML and Messenger-family inspection.
func detectHTMLContent(head []byte) (format, signatureKind string, ok bool) {
	probe := head
	if len(probe) > 4096 {
		probe = probe[:4096]
	}
	if !htmlRoot.Match(probe) {
		return "", "", false
	}
	ownerIMessage := classToken(head, "bubble") &&
		(classToken(head, "from-me") || classToken(head, "from-them")) &&
		classToken(head, "meta")
	stockIMessage := classToken(head, "message") &&
		(classToken(head, "sent") || classToken(head, "received")) &&
		(classToken(head, "timestamp") || classToken(head, "sender"))
	if ownerIMessage || stockIMessage {
		return "imessage_html", "imessage_message_html_v1", true
	}
	if classToken(head, "_a6-g") && classToken(head, "_a6-h") && classToken(head, "_a6-p") &&
		classToken(head, "_a6-o") && !bytes.Contains(head, []byte(`"_a70f"`)) {
		return "facebook_messenger_html", "facebook_messenger_thread_html_v1", true
	}
	return "generic_html_document", "html_document_root_v1", true
}
