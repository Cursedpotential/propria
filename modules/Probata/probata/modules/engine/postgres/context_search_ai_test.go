// Byline: Codex · GPT-6-Sol · 2026-10-05.
package postgres

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// aiSearchDB models independently persisted source/raw provenance and counts human resolution lookups.
// Inputs: source/generation coordinates and declared/raw formats. Outputs: fixture rows. Effects: counters only.
// Choose to prove a request cannot waive the SQL source and normalized-verification gates.
type aiSearchDB struct {
	DB
	source, generation uuid.UUID
	declared, format   string
	foreign            bool
	resolutionReads    int
}

func (d *aiSearchDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "context.reconciliation_receipt") {
		return reviewFakeRow{values: []any{d.generation, "success"}}
	}
	if strings.Contains(sql, "FROM context.normalized_generation generation") {
		source := d.source
		if d.foreign {
			source = uuid.New()
		}
		return reviewFakeRow{values: []any{source, "normalizer", "1", "parser", "1", d.declared, d.format, uuid.NullUUID{UUID: uuid.New(), Valid: true}, uuid.NullUUID{}, bytes.Repeat([]byte{3}, 32)}}
	}
	d.resolutionReads++
	return reviewFakeRow{err: pgx.ErrNoRows}
}

// TestAINeutralSearchStoreUsesPersistedFormats checks AI bypasses human resolution only after the source gates.
// Inputs: synthetic stored declared/raw formats, source ownership and verified generation. Outputs: assertions.
// Effects: no database or target writes. Choose as the SQL counterpart to the Activity and wire-payload tests.
func TestAINeutralSearchStoreUsesPersistedFormats(t *testing.T) {
	for _, tc := range []struct {
		declared, raw string
		ai            bool
	}{{"chatgpt_official_json", "generic_message", true}, {"json", "claude_conversations_json", true}, {"smsbackuprestore_xml", "smsbackuprestore_xml", false}} {
		db := &aiSearchDB{source: uuid.New(), generation: uuid.New(), declared: tc.declared, format: tc.raw}
		store, _ := NewContextSearchStore(db)
		spec := activities.PublishContextSearchSpec{SourceVersionRef: proffer.Ref(db.source.String()), NormalizedGenerationRef: proffer.Ref(db.generation.String()), NormalizedVerificationRef: proffer.Ref(uuid.NewString())}
		plan, err := store.OpenContextSearchRecords(context.Background(), spec)
		if (err == nil) != tc.ai || db.resolutionReads != 0 {
			t.Fatalf("AI=%v plan=%+v error=%v resolutionReads=%d", tc.ai, plan, err, db.resolutionReads)
		}
		db.foreign = true
		if _, err := store.OpenContextSearchRecords(context.Background(), spec); err == nil {
			t.Fatal("foreign source bypassed normalized/source proof")
		}
	}
}
