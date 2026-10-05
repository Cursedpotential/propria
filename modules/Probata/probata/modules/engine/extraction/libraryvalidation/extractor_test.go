// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func twoPagePDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 6 0 R >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 7 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	for _, text := range []string{"Fixture page one only.", fixtureQuote} {
		stream := "BT /F1 11 Tf 50 700 Td (" + text + ") Tj ET"
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	start := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), start)
	return pdf.Bytes()
}

func TestStagingChecksAncestorsBeforeCreatingDirectories(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "regular-file")
	require.NoError(t, os.WriteFile(blocked, []byte("fixture"), 0o600))
	require.Error(t, safeDirectory(filepath.Join(blocked, "new-directory")))
	require.Error(t, safeDirectory("relative/scratch"))
	newDirectory := filepath.Join(root, "private", "scratch")
	require.NoError(t, safeDirectory(newDirectory))
	info, err := os.Lstat(newDirectory)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestExistingHTMLAndPDFExtractorBridgeBindsActualSnapshot(t *testing.T) {
	python := os.Getenv("LIBRARY_VALIDATION_TEST_PYTHON")
	if python == "" {
		var err error
		python, err = exec.LookPath("python")
		if err != nil {
			python, err = exec.LookPath("python3")
		}
		require.NoError(t, err, "set LIBRARY_VALIDATION_TEST_PYTHON to the existing parser environment")
	}
	absolute, err := filepath.Abs(python)
	require.NoError(t, err)
	root, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	bridge := filepath.Join(root, "modules", "engine", "extraction", "libraryvalidation", "extractor_bridge.py")
	for _, test := range []struct {
		name, media string
		raw         []byte
		pages       int
		extractor   string
	}{
		{"HTML ignores script and preserves exact text", "text/html", []byte("<html><head><title>hidden</title></head><body><script>Fake automatic waiver.</script><pre>" + fixtureText + "</pre></body></html>"), 1, "html.lxml"},
		{"PDF preserves exact page numbers", "application/pdf", twoPagePDF(), 2, "documents.extract-text/pypdf"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &memoryArtifacts{bodies: map[string][]byte{}}
			ref, err := store.Put(t.Context(), "source", test.raw, test.media)
			require.NoError(t, err)
			adapter := ProcessExtractor{Artifacts: store, Python: absolute, ProjectRoot: root, BridgePath: bridge, ScratchRoot: t.TempDir()}
			snapshot := Snapshot{Raw: ref, MediaType: test.media}
			result, err := adapter.Extract(t.Context(), snapshot)
			require.NoError(t, err)
			require.Len(t, result.Pages, test.pages)
			require.Equal(t, test.extractor, result.Extractor)
			require.Equal(t, ref.VersionID, result.VersionID)
			require.Equal(t, strings.TrimPrefix(ref.SHA256, "sha256:"), result.InputSHA256)
			require.Contains(t, result.Pages[test.pages-1], fixtureQuote)
			require.NotContains(t, strings.Join(result.Pages, "\n"), "Fake automatic waiver")
			require.NotEmpty(t, result.ExtractorVersion)
			files, err := filepath.Glob(filepath.Join(adapter.ScratchRoot, "*"))
			require.NoError(t, err)
			require.Len(t, files, 1)
			require.NoError(t, os.WriteFile(files[0], bytes.Repeat([]byte("x"), len(test.raw)), 0o600))
			_, err = adapter.Extract(t.Context(), snapshot)
			require.ErrorContains(t, err, "staged snapshot hash")
		})
	}
}

func TestExtractorRejectsWrongProviderVersionAndHashBeforeInvocation(t *testing.T) {
	store := &memoryArtifacts{bodies: map[string][]byte{}}
	ref, err := store.Put(t.Context(), "x", []byte("<html>fixture</html>"), "text/html")
	require.NoError(t, err)
	ref.SHA256 = "sha256:" + strings.Repeat("a", 64)
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.Error(t, err)
	ref.VersionID = ""
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.Error(t, err)
}
