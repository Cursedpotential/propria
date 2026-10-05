// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Extracted contains page-separated text bound to the actual staged bytes and provider version.
// Inputs: existing extractor result; outputs: text plus source identity. Effects: none; use inside verification only.
type Extracted struct {
	Pages            []string `json:"pages"`
	InputSHA256      string   `json:"input_sha256"`
	VersionID        string   `json:"version_id"`
	Extractor        string   `json:"extractor"`
	ExtractorVersion string   `json:"extractor_version"`
	LowConfidence    bool     `json:"low_confidence"`
}

// Extractor reads an exact snapshot through the supplied versioned artifact store.
// Inputs: immutable snapshot; outputs: input-bound pages. Effects: staged extraction only, never source fetching or model inference.
type Extractor interface {
	Extract(context.Context, Snapshot) (Extracted, error)
}

// ProcessExtractor stages independently verified bytes and invokes the existing Python PDF/HTML units on the same host.
// Inputs: explicit Python executable, project root, bridge file and scratch root; outputs: input-bound extraction.
// Effects: retains private scratch files and runs a bounded child process; no files are deleted and no source record is modified.
// Choose this explicitly authorized pinning seam when the generic tool gateway cannot attest its input identity.
type ProcessExtractor struct {
	Artifacts                                    Artifacts
	Python, ProjectRoot, BridgePath, ScratchRoot string
}

// Extract opens the exact B2 version, stages bytes with a format suffix, and verifies both file and result descriptor after invocation.
// Inputs: signed source snapshot; outputs: bounded pages with actual input hash/version. Effects: local staging and existing parser call.
func (p ProcessExtractor) Extract(ctx context.Context, s Snapshot) (Extracted, error) {
	if p.Artifacts == nil {
		return Extracted{}, errors.New("snapshot extractor artifacts are not configured")
	}
	for _, dir := range []string{p.ProjectRoot, p.ScratchRoot} {
		if !filepath.IsAbs(dir) {
			return Extracted{}, errors.New("snapshot extractor paths must be absolute")
		}
		if err := safeDirectory(dir); err != nil {
			return Extracted{}, err
		}
	}
	for _, file := range []string{p.Python, p.BridgePath} {
		if !filepath.IsAbs(file) {
			return Extracted{}, errors.New("snapshot extractor executable/bridge must be absolute")
		}
		info, err := os.Lstat(file)
		if err != nil || !info.Mode().IsRegular() {
			return Extracted{}, errors.New("snapshot extractor executable/bridge is not a regular file")
		}
	}
	raw, err := p.Artifacts.Read(ctx, s.Raw, MaxSourceBytes)
	if err != nil {
		return Extracted{}, err
	}
	suffix := ""
	switch s.MediaType {
	case "application/pdf":
		suffix = ".pdf"
	case "text/html", "application/xhtml+xml":
		suffix = ".html"
	default:
		return Extracted{}, errors.New("snapshot format has no supported extractor")
	}
	if s.MediaType == "application/pdf" && !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return Extracted{}, errors.New("snapshot PDF signature mismatch")
	}
	if suffix == ".html" && !utf8.Valid(raw) {
		return Extracted{}, errors.New("snapshot HTML is not valid UTF-8")
	}
	name := strings.TrimPrefix(s.Raw.SHA256, "sha256:") + "-" + strings.TrimPrefix(Hash([]byte(s.Raw.VersionID)), "sha256:") + suffix
	staged := filepath.Join(p.ScratchRoot, name)
	file, err := os.OpenFile(staged, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err == nil {
		_, writeErr := file.Write(raw)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			return Extracted{}, errors.New("snapshot staging failed; partial file retained")
		}
	} else if !errors.Is(err, os.ErrExist) {
		return Extracted{}, errors.New("snapshot staging unavailable")
	}
	if err = verifyStaged(staged, s.Raw); err != nil {
		return Extracted{}, err
	}
	descriptor, _ := json.Marshal(map[string]any{"path": staged, "input_sha256": strings.TrimPrefix(s.Raw.SHA256, "sha256:"), "version_id": s.Raw.VersionID, "media_type": s.MediaType})
	ctx, cancel := context.WithTimeout(ctx, ExtractTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Python, "-B", p.BridgePath)
	cmd.Dir = p.ProjectRoot
	// Only dependency/runtime paths reach the parser process; service/database/model credentials never do.
	cmd.Env = parserEnvironment(p.ProjectRoot)
	cmd.Stdin = bytes.NewReader(descriptor)
	output := &budgetWriter{Limit: MaxArtifactBytes}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	if err = cmd.Run(); err != nil {
		return Extracted{}, errors.New("pinned snapshot extraction failed or exceeded time/output budget")
	}
	if err = verifyStaged(staged, s.Raw); err != nil {
		return Extracted{}, err
	}
	var result Extracted
	if err = json.Unmarshal(output.Buffer.Bytes(), &result); err != nil || result.InputSHA256 != strings.TrimPrefix(s.Raw.SHA256, "sha256:") || result.VersionID != s.Raw.VersionID || result.Extractor == "" || result.ExtractorVersion == "" || result.LowConfidence || len(result.Pages) == 0 || len(result.Pages) > MaxPages {
		return Extracted{}, errors.New("snapshot extraction descriptor is incomplete or does not match input")
	}
	total := 0
	for _, page := range result.Pages {
		if !utf8.ValidString(page) {
			return Extracted{}, errors.New("extracted text is not UTF-8")
		}
		total += len(page)
	}
	if total == 0 || total > MaxTextBytes {
		return Extracted{}, errors.New("extracted text exceeds verification budget or is empty")
	}
	return result, nil
}

// safeDirectory rejects symlinked ancestors before staging; inputs: absolute directory; outputs: error; effects: creates scratch only.
func safeDirectory(dir string) error {
	if !filepath.IsAbs(dir) {
		return errors.New("snapshot staging directory must be absolute")
	}
	var missing []string
	for current := filepath.Clean(dir); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			missing = append(missing, current)
		} else if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("snapshot staging path is not a real directory")
		}
		if current == filepath.Dir(current) {
			break
		}
	}
	// Check existing ancestors before any creation, then create one checked level at a time.
	for i := len(missing) - 1; i >= 0; i-- {
		if err := os.Mkdir(missing[i], 0o700); err != nil && !os.IsExist(err) {
			return errors.New("snapshot staging directory unavailable")
		}
		info, err := os.Lstat(missing[i])
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("snapshot staging path changed during creation")
		}
	}
	return nil
}

// verifyStaged hashes the complete regular staged file; inputs: path/pin; outputs: integrity error; effects: bounded file read.
func verifyStaged(file string, ref ArtifactRef) error {
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Size() != ref.Bytes {
		return errors.New("staged snapshot size/type mismatch")
	}
	f, err := os.Open(file)
	if err != nil {
		return errors.New("staged snapshot unreadable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, ref.Bytes+1))
	if err != nil || int64(len(raw)) != ref.Bytes || Hash(raw) != ref.SHA256 {
		return errors.New("staged snapshot hash mismatch")
	}
	return nil
}

type budgetWriter struct {
	Buffer bytes.Buffer
	Limit  int64
}

// Write bounds child output in memory; inputs: chunk; outputs: written count/error; effects: buffer append only.
func (b *budgetWriter) Write(raw []byte) (int, error) {
	if int64(b.Buffer.Len()+len(raw)) > b.Limit {
		return 0, errors.New("extractor output budget exceeded")
	}
	return b.Buffer.Write(raw)
}
