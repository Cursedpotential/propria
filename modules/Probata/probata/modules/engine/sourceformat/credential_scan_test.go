// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type credentialFailureReader struct{}

// Read simulates source access failure during a whole-stream credential scan.
// Input: buffer. Output: unexpected EOF. Side effects: none. Pick for fail-closed regression.
func (credentialFailureReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// TestCredentialScanCoversWholeStreamAndChunkBoundaries proves late and escaped assignments cannot evade admission.
// Inputs: synthetic streams. Outputs: signature/completion results. Side effects: none.
// Pick before using scan completion as a prerequisite for source copying.
func TestCredentialScanCoversWholeStreamAndChunkBoundaries(t *testing.T) {
	for _, content := range []string{
		strings.Repeat("x", int(ReadLimit)+500) + "\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90",
		strings.Repeat("x", (64<<10)-7) + "\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90",
		`{"text":"Configuration\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90"}`,
		"name,url,username,password\nSite,https://site.test,user,Ab12\n",
	} {
		scan, err := ScanCredentials(strings.NewReader(content))
		if err != nil || scan.Signature == "" || scan.Complete {
			t.Fatalf("expected early credential exclusion: %+v %v", scan, err)
		}
	}
	for _, content := range []string{"A document about client_secret and API_KEY terms.", "API_KEY=YOUR_API_KEY_HERE", "name,url,username,password\nSite,https://site.test,user,\n", string([]byte{0xff, 0xfe, 0x00}), ""} {
		scan, err := ScanCredentials(strings.NewReader(content))
		if err != nil || scan.Signature != "" || !scan.Complete || scan.BytesRead != int64(len(content)) {
			t.Fatalf("unexpected exclusion/incomplete scan: %+v %v", scan, err)
		}
	}
	scan, err := ScanCredentials(credentialFailureReader{})
	if !errors.Is(err, io.ErrUnexpectedEOF) || scan.Complete {
		t.Fatalf("access failure did not fail closed: %+v %v", scan, err)
	}
}
