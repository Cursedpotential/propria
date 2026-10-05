// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeAdmissionUsesExistingParserImportsAndRegularInterpreter(t *testing.T) {
	python := os.Getenv("LIBRARY_VALIDATION_TEST_PYTHON")
	if python == "" {
		var err error
		python, err = exec.LookPath("python")
		require.NoError(t, err)
	}
	python, err := filepath.Abs(python)
	require.NoError(t, err)
	root, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	p := ProcessExtractor{Python: python, ProjectRoot: root, BridgePath: filepath.Join(root, "modules", "engine", "extraction", "libraryvalidation", "extractor_bridge.py"), ScratchRoot: t.TempDir()}
	info, err := p.ValidateRuntime(t.Context())
	require.NoError(t, err)
	require.Equal(t, RuntimeContract, info.Contract)
	require.Equal(t, "html.lxml", info.HTMLExtractor)
	require.Equal(t, "documents.extract-text/pypdf", info.PDFExtractor)
	bad := p
	bad.Python = filepath.Join(root, "missing-python")
	_, err = bad.ValidateRuntime(t.Context())
	require.ErrorContains(t, err, "Go-only worker image")
	bad = p
	bad.ProjectRoot = p.BridgePath
	_, err = bad.ValidateRuntime(t.Context())
	require.Error(t, err)
	t.Setenv("TOOLKIT_VALIDATION_SIGNING_KEY", "fixture-secret-never-forward")
	t.Setenv("ENTITY_MODEL_API_KEY", "fixture-nim-secret-never-forward")
	t.Setenv("PYTHONPATH", "untrusted-ambient-parser-path")
	env := strings.Join(parserEnvironment(root), "\n")
	require.NotContains(t, env, "secret-never-forward")
	require.NotContains(t, env, "untrusted-ambient-parser-path")
	require.Contains(t, env, "PYTHONPATH="+root)
}
