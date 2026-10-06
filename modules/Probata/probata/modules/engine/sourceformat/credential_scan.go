// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"strings"
)

// CredentialScan records whole-stream signature coverage without exposing inspected values.
// Complete means EOF was reached under this signature policy, not that all possible secrets were ruled out.
type CredentialScan struct {
	Signature string `json:"credential_signature,omitempty"`
	Complete  bool   `json:"credential_scan_complete"`
	BytesRead int64  `json:"credential_scan_bytes"`
}

// countedCredentialReader counts actual reads for the independent credential scan.
type countedCredentialReader struct {
	reader io.Reader
	count  int64
}

// Read delegates credential-stream reads and tracks their byte count.
// Input: destination buffer. Output: bytes and reader error. Side effects: source reads and count updates.
// Pick only inside ScanCredentials; it performs no hashing or parsing for ingestion.
func (r *countedCredentialReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.count += int64(n)
	return n, err
}

// ScanCredentials inspects a source stream for secret-bearing signatures with fixed memory bounds.
// Input: read-only reader. Output: safe signature, EOF completion, bytes read, and access/limit error.
// Side effects: source reads; no values are emitted. Pick separately from Detect before copying source bytes.
// General scanning uses 64 KiB chunks and 16 KiB overlap; browser CSV rows are limited to 1 MiB.
// A positive match returns early with Complete=false because exclusion already prevents copying.
func ScanCredentials(reader io.Reader) (CredentialScan, error) {
	counted := &countedCredentialReader{reader: reader}
	buffered := bufio.NewReaderSize(counted, 64<<10)
	peek, _ := buffered.Peek(512)
	if browserCSVHeader(peek) {
		scanner := bufio.NewScanner(buffered)
		scanner.Buffer(make([]byte, 4096), 1<<20)
		scanner.Scan() // exact header, already checked by the bounded peek
		for scanner.Scan() {
			row, err := csv.NewReader(strings.NewReader(scanner.Text())).Read()
			if err != nil || (len(row) != 4 && len(row) != 5) {
				return CredentialScan{BytesRead: counted.count}, io.ErrUnexpectedEOF
			}
			password := strings.TrimSpace(row[3])
			if password != "" && !placeholderValue(password) {
				return CredentialScan{Signature: "browser_password_export_csv_v1", BytesRead: counted.count}, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return CredentialScan{BytesRead: counted.count}, err
		}
		return CredentialScan{Complete: true, BytesRead: counted.count}, nil
	}
	const overlap = 16 << 10
	window := make([]byte, overlap+(64<<10))
	retained := 0
	for {
		n, err := buffered.Read(window[retained:])
		used := retained + n
		if n > 0 {
			if signature := fragmentCredentialSignature(window[:used]); signature != "" {
				return CredentialScan{Signature: signature, BytesRead: counted.count}, nil
			}
			retained = used
			if retained > overlap {
				retained = overlap
			}
			copy(window[:retained], window[used-retained:used])
		}
		if err == io.EOF {
			return CredentialScan{Complete: true, BytesRead: counted.count}, nil
		}
		if err != nil {
			return CredentialScan{BytesRead: counted.count}, err
		}
		if n == 0 {
			return CredentialScan{BytesRead: counted.count}, io.ErrNoProgress
		}
	}
}

// browserCSVHeader recognizes the exact supported browser export header in a bounded prefix.
// Input: prefix. Output: header match. Side effects: none. Pick before streaming browser CSV rows.
func browserCSVHeader(prefix []byte) bool {
	if end := bytes.IndexByte(prefix, '\n'); end >= 0 {
		prefix = prefix[:end]
	}
	header, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(prefix, []byte{0xef, 0xbb, 0xbf}))).Read()
	if err != nil || (len(header) != 4 && len(header) != 5) {
		return false
	}
	for index, key := range []string{"name", "url", "username", "password"} {
		if strings.ToLower(strings.TrimSpace(header[index])) != key {
			return false
		}
	}
	return len(header) == 4 || strings.ToLower(strings.TrimSpace(header[4])) == "note"
}

// fragmentCredentialSignature recognizes concrete credentials across overlapping stream chunks.
// Input: chunk with overlap. Output: fixed signature ID. Side effects: none.
// Pick for whole-source coverage, including assignments escaped within JSON conversation text.
func fragmentCredentialSignature(fragment []byte) string {
	decoded := fragment
	if bytes.ContainsRune(fragment, '\\') {
		decoded = bytes.ReplaceAll(decoded, []byte(`\n`), []byte{'\n'})
		decoded = bytes.ReplaceAll(decoded, []byte(`\r`), []byte{'\r'})
		decoded = bytes.ReplaceAll(decoded, []byte(`\t`), []byte{'\t'})
		decoded = bytes.ReplaceAll(decoded, []byte(`\"`), []byte{'"'})
	}
	if privateKey.Match(decoded) {
		return "private_key_pem_v1"
	}
	for _, match := range assignment.FindAllSubmatch(decoded, -1) {
		if actualSecret(string(match[2])) {
			return "secret_assignment_v1"
		}
	}
	return ""
}
