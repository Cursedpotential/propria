package postgres

import (
	"context"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"strings"
	"testing"
)

// TestStandaloneWorkerPreviewRequiresLatestExactPublishedPins verifies current worker admission uses exact latest pins.
// Inputs: available/unavailable test rows. Outputs: matching admission. Effects: fake reads only; choose for resolver regressions.
func TestStandaloneWorkerPreviewRequiresLatestExactPublishedPins(t *testing.T) {
	for _, available := range []bool{false, true} {
		db := &operationResultTestDB{replies: [][]any{{available}}}
		run := flow.RunRef{PreviewHandle: "exact-handle", SourceVersionID: "exact-source", GenerationID: "exact-generation"}
		actual, err := (&EntityExtractionStore{db: db}).StandaloneWorkerPreview(context.Background(), run)
		if err != nil || actual != available {
			t.Fatalf("admission=%v %v", actual, err)
		}
		if db.args[0][0] != run.PreviewHandle || db.args[0][1] != run.SourceVersionID || db.args[0][2] != run.GenerationID {
			t.Fatal("worker prerequisite omitted exact source/generation pins")
		}
		if !strings.Contains(db.queries[0], "ORDER BY snapshot_seq DESC LIMIT 1") {
			t.Fatal("worker admission can accept an obsolete preview")
		}
	}
}
