// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// retainedToolkitFixtureRoot creates a unique synthetic fixture beneath an explicit owner-controlled quarantine.
// Inputs: test and directory pattern; outputs: retained absolute fixture path. Effects: creates directories only; choose for toolkit tests that must leave evidence for owner-controlled removal.
func retainedToolkitFixtureRoot(t *testing.T, pattern string) string {
	t.Helper()
	base := os.Getenv("TOOLKIT_TEST_ROOT")
	if base == "" {
		t.Fatal("TOOLKIT_TEST_ROOT must identify an absolute retained fixture directory ending in to_be_deleted")
	}
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || filepath.Base(base) != "to_be_deleted" {
		t.Fatalf("TOOLKIT_TEST_ROOT must be a clean absolute path ending in to_be_deleted: %q", base)
	}
	if err := checkRetainedToolkitPath(base); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	if err := checkRetainedToolkitPath(base); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, pattern)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained synthetic toolkit fixture: %s", root)
	return root
}

// checkRetainedToolkitPath rejects symlinks and non-directories in every existing path component.
// Inputs: clean absolute fixture base; outputs: nil or path error. Effects: read-only metadata checks; choose before and after creating a retained fixture base.
func checkRetainedToolkitPath(base string) error {
	for path := base; ; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("retained toolkit fixture path contains a link or non-directory: %s", path)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect retained toolkit fixture path %s: %w", path, err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
	}
}
