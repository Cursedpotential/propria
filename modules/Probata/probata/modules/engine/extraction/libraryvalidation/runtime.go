// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const RuntimeContract = "library-validation-runtime/v1"

// RuntimeInfo identifies the installed existing parser environment admitted for pinned extraction.
// Inputs: bounded bridge preflight; outputs: contract/parser versions. Effects: none; never claims source or legal currency validation.
type RuntimeInfo struct {
	Contract      string `json:"contract"`
	PythonVersion string `json:"python_version"`
	HTMLExtractor string `json:"html_extractor"`
	HTMLVersion   string `json:"html_version"`
	PDFExtractor  string `json:"pdf_extractor"`
	PDFVersion    string `json:"pdf_version"`
}

// ValidateRuntime admits only an actual same-host Python/bridge and existing HTML/PDF imports before registration.
// Inputs: configured adapter; outputs: versioned RuntimeInfo or configuration error. Effects: bounded child import check only.
// Choose at worker boot; the deployed Go-only image fails explicitly until the bounded runtime extension is built.
func (p ProcessExtractor) ValidateRuntime(ctx context.Context) (RuntimeInfo, error) {
	for _, entry := range []struct {
		path      string
		directory bool
	}{{p.ProjectRoot, true}, {p.Python, false}, {p.BridgePath, false}} {
		info, err := os.Lstat(entry.path)
		if !filepath.IsAbs(entry.path) || err != nil || info.Mode()&os.ModeSymlink != 0 || (entry.directory && !info.IsDir()) || (!entry.directory && !info.Mode().IsRegular()) {
			return RuntimeInfo{}, errors.New("library validation needs existing absolute Python, project and bridge paths; Go-only worker image is insufficient")
		}
	}
	if !filepath.IsAbs(p.ScratchRoot) {
		return RuntimeInfo{}, errors.New("library validation needs an absolute private scratch mount")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Python, "-B", p.BridgePath, "--check-runtime")
	cmd.Dir = p.ProjectRoot
	cmd.Env = parserEnvironment(p.ProjectRoot)
	output := &budgetWriter{Limit: 16 << 10}
	cmd.Stdout, cmd.Stderr = output, io.Discard
	if cmd.Run() != nil {
		return RuntimeInfo{}, errors.New("library validation existing Python lxml/pypdf runtime is unavailable; build the bounded extractor runtime profile")
	}
	var result RuntimeInfo
	if json.Unmarshal(output.Buffer.Bytes(), &result) != nil || result.Contract != RuntimeContract || result.PythonVersion == "" || result.HTMLExtractor != "html.lxml" || result.PDFExtractor != "documents.extract-text/pypdf" || result.HTMLVersion == "" || result.PDFVersion == "" {
		return RuntimeInfo{}, errors.New("library validation extractor runtime descriptor mismatch")
	}
	return result, nil
}

// parserEnvironment passes only interpreter dependencies and the immutable project path to child extractors.
// Inputs: same-host project root; outputs: bounded environment. Effects: env reads only; no service/model/DB credentials.
func parserEnvironment(root string) []string {
	var env []string
	for _, name := range []string{"PATH", "SYSTEMROOT", "WINDIR", "LD_LIBRARY_PATH", "VIRTUAL_ENV", "LANG", "LC_ALL"} {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return append(env, "PYTHONPATH="+root, "PYTHONDONTWRITEBYTECODE=1")
}
