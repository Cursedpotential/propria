// Byline: Claude Code · Opus 5.5 · 2026-09-26
package contextreview

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hindsightOnlyNames are the relations that hold (or can surface) the
// hindsight-only foreshadowing flag.
var hindsightOnlyNames = []string{
	"record_foreshadowing_flag",
	"vw_record_foreshadowing_current",
	"vw_record_context_review_horizon",
}

// hindsightReaders is every file allowed to name those relations. A new
// reader — a Weaviate or SurrealDB projection, an agent tool, a report — must
// be added here deliberately, after its own horizon pre-filter is in place.
// AGENTS.md "WHY THIS EXISTS": a leak is silent; this tripwire makes it loud.
var hindsightReaders = map[string]bool{
	"scripts/2026-09-25-context-review-overlay.sql": true,
	"sql/bootstrap/schema_snapshot_20260907.sql":    true,
	// DDL only: drops the flag table's append-only triggers at go-live; reads no rows (2026-10-02).
	"sql/bootstrap/seed_live_case_registry_20261001.sql":        true,
	"sql/validation/2026-09-25-context-review-horizon-test.sql": true,
	"modules/engine/postgres/context_review_store.go":           true,
	"modules/engine/postgres/context_review_store_test.go":      true,
	"modules/engine/contextreview/horizon_tripwire_test.go":     true,
}

var scannedRoots = []string{"server", "modules/engine", "modules/workbench/api", "modules/workbench/web/src", "modules/workbench/web/smoke", "scripts", "deploy", "sql"}

var scannedExtensions = map[string]bool{
	".go": true, ".py": true, ".ts": true, ".tsx": true, ".mjs": true, ".sql": true, ".sh": true,
	".yaml": true, ".yml": true, ".json": true, ".surql": true,
}

var skippedDirs = map[string]bool{"vendor": true, "node_modules": true, "_stale": true, "__pycache__": true, ".git": true}

func TestOnlyDeclaredReadersNameTheHindsightFlag(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "sql", "bootstrap")); err != nil {
		t.Skip("repository root is not available beside this module; the tripwire runs from a full checkout")
	}
	for _, scanned := range scannedRoots {
		base := filepath.Join(root, filepath.FromSlash(scanned))
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if skippedDirs[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !scannedExtensions[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if hindsightReaders[relative] {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			for _, name := range hindsightOnlyNames {
				if strings.Contains(text, name) {
					t.Errorf("%s names %s: add it to hindsightReaders only once its horizon pre-filter is proven", relative, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
