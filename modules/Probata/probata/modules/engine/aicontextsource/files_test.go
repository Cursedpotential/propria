package aicontextsource

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreparedGeminiOriginalAndUnicodeSpan checks the copied original and an absolute codepoint citation.
// Inputs: one native Markdown bundle in a temporary mounted root. Outputs: exact quote/hash checks.
// Effects: temporary files only; pick to guard source identity without a retained object or preview.
func TestPreparedGeminiOriginalAndUnicodeSpan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AI_CONTEXT_ROOT", root)
	dir := filepath.Join(root, "scope")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte("**You:**\n😀 ask\n**Gemini:**\nanswer\n")
	originalPath := filepath.Join(dir, "original.md")
	if err := os.WriteFile(originalPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	prepared := Prepared{ContractVersion: "ai-context-v1", Stage: "prepared", Source: Identity{SourceRef: "b2://original", SourceFormat: "gemini_markdown"}, OriginalRef: testFileURI(originalPath), OriginalSHA256: Hash(original)}
	raw, err := json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	preparedPath := filepath.Join(dir, "prepared.json")
	if err := os.WriteFile(preparedPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, bytes, err := ReadPrepared(testFileURI(preparedPath))
	if err != nil {
		t.Fatal(err)
	}
	if got.OriginalSHA256 != Hash(original) || string(bytes) != string(original) {
		t.Fatal("copied original differs")
	}
	start := len([]rune("**You:**\n"))
	end := start + len([]rune("😀 ask"))
	spanSHA, err := VerifyQuote("gemini_markdown", bytes, "", start, end, "😀 ask")
	if err != nil || spanSHA != Hash([]byte("😀 ask")) {
		t.Fatalf("absolute Unicode citation failed: %v", err)
	}
	if _, err := VerifyQuote("gemini_markdown", bytes, "", -1, end, "😀 ask"); err == nil {
		t.Fatal("negative span admitted")
	}
	if _, err := VerifyQuote("gemini_markdown", bytes, "", start, end, "changed"); err == nil {
		t.Fatal("changed quote admitted")
	}
	if _, err := VerifyQuote("gemini_markdown", bytes, "/wrong", start, end, "😀 ask"); err == nil {
		t.Fatal("JSON pointer admitted for Markdown")
	}
	if err := os.WriteFile(originalPath, append(original, 'x'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadPrepared(testFileURI(preparedPath)); err == nil {
		t.Fatal("changed original admitted")
	}
}

// TestNativeJSONQuoteRejectsMalformedSource checks that a JSON locator cannot silently fall back to whole text.
// Inputs: malformed JSON and a pointer. Outputs: validation failure. Effects: none.
// Pick for legacy JSON source formats where the native pointer is required.
func TestNativeJSONQuoteRejectsMalformedSource(t *testing.T) {
	if _, err := VerifyQuote("chatgpt_official_json", []byte("{bad"), "/messages/0/body", 0, 1, "x"); err == nil {
		t.Fatal("malformed native JSON admitted")
	}
	if _, err := VerifyQuote("chatgpt_official_json", []byte(`{"body":"x"}`), "", 0, 1, "x"); err == nil {
		t.Fatal("missing native JSON pointer admitted")
	}
}

// testFileURI turns an absolute platform path into the native producer's file URI shape.
// Inputs: temporary absolute path. Outputs: file URI. Effects: none.
// Pick for tests of the real URI reader rather than bypassing its containment checks.
func testFileURI(path string) string {
	slash := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return (&url.URL{Scheme: "file", Path: slash}).String()
}
