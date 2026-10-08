// Byline: Codex · GPT-6 · 2026-10-08.
package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/jackc/pgx/v5"
)

type approvedPinDB struct {
	query     string
	available *time.Time
}

func (d *approvedPinDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func (d *approvedPinDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected multirow query")
}

func (d *approvedPinDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	d.query = query
	return approvedPinRow{available: d.available}
}

type approvedPinRow struct{ available *time.Time }

func (r approvedPinRow) Scan(dest ...any) error {
	if len(dest) != 5 {
		return errors.New("unexpected pin shape")
	}
	*dest[0].(*string) = "record"
	*dest[1].(*string) = "version"
	*dest[2].(*string) = strings.Repeat("a", 64)
	*dest[3].(**time.Time) = nil
	*dest[4].(**time.Time) = r.available
	return nil
}

func TestApprovedRecordPinUsesVerifiedSpineClockNotPayloadFallback(t *testing.T) {
	canonical := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	db := &approvedPinDB{available: &canonical}
	reader, err := NewApprovedGraphReader(db)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := reader.ReadRecordPin(context.Background(), approvedgraph.Scope{GenerationID: "generation", SourceVersionID: "version"}, "record")
	if err != nil || pin.SourceAvailableFrom == nil || !pin.SourceAvailableFrom.Equal(canonical) {
		t.Fatalf("canonical source clock missing: pin=%+v err=%v", pin, err)
	}
	for _, binding := range []string{
		"w.id=c.id", "w.source_version_id=c.source_version_id",
		"w.derived_from_raw_table='context.normalized_record_identity'", "w.derived_from_raw_id=c.id",
		"route.normalized_record_id=w.id", "route.decision_state='approved'",
		"working.source_available_from(w.id)",
	} {
		if !strings.Contains(db.query, binding) {
			t.Fatalf("canonical source clock lost binding %q", binding)
		}
	}
	if strings.Contains(db.query, "normalized_payload") || strings.Contains(db.query, "acquired_at") || strings.Contains(db.query, "ingested_at") {
		t.Fatal("unverified payload or ingestion clock can override the canonical source clock")
	}
	db.available = nil
	pin, err = reader.ReadRecordPin(context.Background(), approvedgraph.Scope{GenerationID: "generation", SourceVersionID: "version"}, "record")
	if err != nil || pin.SourceAvailableFrom != nil {
		t.Fatalf("missing approved spine route was assigned a source clock: pin=%+v err=%v", pin, err)
	}
}
