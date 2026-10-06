// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"bytes"
	"regexp"
)

var goSource = regexp.MustCompile(`(?m)\Apackage\s+[A-Za-z_][A-Za-z0-9_]*\s*$`)
var pythonSource = regexp.MustCompile(`\A(?:from [A-Za-z_.]+ import |import [A-Za-z_.]+|def [A-Za-z_][A-Za-z0-9_]*\([^\n]*\):|class [A-Za-z_][A-Za-z0-9_]*[:(])`)

// codeSignature recognizes conservative native code content signatures without a filename.
// Input: UTF-8 prefix. Output: code_go_v1, code_script_shebang_v1, code_python_v1 or empty.
// Side effects: none. Pick for code exclusion of extensionless source; prose remains a text document.
func codeSignature(head []byte) string {
	if bytes.HasPrefix(head, []byte("#!")) {
		return "code_script_shebang_v1"
	}
	if goSource.Match(head) && (bytes.Contains(head, []byte("import ")) || bytes.Contains(head, []byte("func "))) {
		return "code_go_v1"
	}
	if pythonSource.Match(head) && (bytes.Contains(head, []byte("def ")) || bytes.Contains(head, []byte("class "))) {
		return "code_python_v1"
	}
	return ""
}
