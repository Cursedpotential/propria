// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strings"
)

var privateKey = regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----\s*([A-Za-z0-9+/=\r\n]{32,})`)
var assignment = regexp.MustCompile(`(?im)(?:^|[\s{,])(?:export\s+)?["']?([A-Za-z_][A-Za-z0-9_]*(?:api_key|apikey|token|secret|password)|api_key|apikey|token|secret|password)["']?\s*[:=]\s*["']?([^\s"'#,;}]+)`)

// CredentialSignature reports exact secret-bearing signatures without returning their values.
// Inputs: bounded source bytes. Output: private_key_pem_v1, oauth_credentials_json_v1,
// service_account_credentials_json_v1, secret_assignment_v1, or empty string.
// Side effects: none. Pick this admission check before indexing any detected content format.
func CredentialSignature(head []byte) string {
	if browserPasswordCSV(head) {
		return "browser_password_export_csv_v1"
	}
	if privateKey.Match(head) {
		return "private_key_pem_v1"
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimPrefix(head, []byte{0xef, 0xbb, 0xbf}), &fields) == nil {
		var kind string
		_ = json.Unmarshal(fields["type"], &kind)
		if kind == "service_account" && jsonSecret(fields, "private_key") && jsonSecret(fields, "client_email") {
			return "service_account_credentials_json_v1"
		}
		if kind == "authorized_user" && jsonSecret(fields, "client_id") && jsonSecret(fields, "client_secret") && jsonSecret(fields, "refresh_token") {
			return "oauth_credentials_json_v1"
		}
		for _, key := range []string{"installed", "web"} {
			var oauth map[string]json.RawMessage
			if json.Unmarshal(fields[key], &oauth) == nil && jsonSecret(oauth, "client_id") && jsonSecret(oauth, "client_secret") {
				return "oauth_credentials_json_v1"
			}
		}
	}
	for _, match := range assignment.FindAllSubmatch(head, -1) {
		if actualSecret(string(match[2])) {
			return "secret_assignment_v1"
		}
	}
	// Decode string tokens so escaped newlines and quotes in native AI exports do
	// not hide secret assignments; incomplete prefixes still expose completed tokens.
	trimmed := bytes.TrimSpace(head)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			value, ok := token.(string)
			if !ok {
				continue
			}
			decoded := []byte(value)
			if privateKey.Match(decoded) {
				return "private_key_pem_v1"
			}
			for _, match := range assignment.FindAllSubmatch(decoded, -1) {
				if actualSecret(string(match[2])) {
					return "secret_assignment_v1"
				}
			}
		}
	}
	return ""
}

// browserPasswordCSV recognizes exported browser passwords by exact header fields and a populated password cell.
// Input: bounded bytes. Output: match only for name,url,username,password[,note] CSV with an actual password row.
// Side effects: none. Pick inside credential admission; password values are never returned or logged.
func browserPasswordCSV(head []byte) bool {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(head, []byte{0xef, 0xbb, 0xbf})))
	header, err := reader.Read()
	if err != nil || (len(header) != 4 && len(header) != 5) {
		return false
	}
	for index, key := range []string{"name", "url", "username", "password"} {
		if strings.ToLower(strings.TrimSpace(header[index])) != key {
			return false
		}
	}
	if len(header) == 5 && strings.ToLower(strings.TrimSpace(header[4])) != "note" {
		return false
	}
	for {
		row, err := reader.Read()
		if err != nil {
			return false
		}
		password := strings.TrimSpace(row[3])
		if password != "" && !placeholderValue(password) {
			return true
		}
	}
}

// placeholderValue rejects recognizable template or redacted credential values.
// Input: candidate value. Output: placeholder flag. Side effects: none.
// Pick before declaring actual secret-bearing assignments or browser password rows.
func placeholderValue(value string) bool {
	lower := strings.ToLower(value)
	for _, placeholder := range []string{"example", "placeholder", "your_", "your-", "replace", "redacted", "changeme", "insert_", "<", "${", "{{", "os.getenv", "process.env", "getenv("} {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}
	return false
}

// jsonSecret tests whether a named JSON field has a concrete non-placeholder value.
// Inputs: JSON map and field key. Output: eligibility for a credential signature.
// Side effects: none. Pick internally when checking exact OAuth or service-account shapes.
func jsonSecret(fields map[string]json.RawMessage, key string) bool {
	var value string
	return json.Unmarshal(fields[key], &value) == nil && actualSecret(value)
}

// actualSecret rejects common template values before recognizing a concrete credential assignment.
// Input: candidate assignment value. Output: whether its length and characters indicate an actual value.
// Side effects: none. Pick this only with a recognized secret key or credential shape, never prose.
func actualSecret(value string) bool {
	if len(value) < 12 {
		return false
	}
	if placeholderValue(value) {
		return false
	}
	distinct := map[rune]bool{}
	for _, character := range value {
		distinct[character] = true
	}
	return len(distinct) >= 5
}
