// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
package postgres

import (
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
)

// A generation of messages and calls only publishes nothing per record (they go out as conversation chunks), and that
// is still a recorded outcome; a pass that published nothing and skipped nothing is not.
func TestContextSearchOutcomeIsRecordableWhenEverythingWasLeftToChunks(t *testing.T) {
	cases := []struct {
		name    string
		outcome activities.ContextSearchPublicationOutcome
		want    bool
	}{
		{"published", activities.ContextSearchPublicationOutcome{Published: 2, Collections: map[string]int{"Docs": 2}}, true},
		{"only matched", activities.ContextSearchPublicationOutcome{SkippedMatched: 3}, true},
		{"only left to chunks", activities.ContextSearchPublicationOutcome{SkippedToChunks: 3}, true},
		{"published without a collection", activities.ContextSearchPublicationOutcome{Published: 1}, false},
		{"nothing at all", activities.ContextSearchPublicationOutcome{}, false},
	}
	for _, c := range cases {
		if got := contextSearchOutcomeRecordable(c.outcome); got != c.want {
			t.Errorf("%s: recordable = %v, want %v", c.name, got, c.want)
		}
	}
}
