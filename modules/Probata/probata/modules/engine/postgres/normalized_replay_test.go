// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/normalize"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"strings"
	"testing"
)

type normalizeReplayDB struct {
	DB
	row   pgx.Row
	query string
	args  []any
}

// QueryRow captures a read for ownership assertions and returns its fixture row.
// Inputs are SQL/args; output is pgx.Row. It mutates memory only; pick for replay
// seam tests without granting database mutation authority.
func (d *normalizeReplayDB) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.query = q
	d.args = args
	return d.row
}

type historicalNormalizeReader struct {
	header  normalize.BundleHeader
	nextErr error
	closed  bool
	reads   int
}

// Header returns the original fixture version without changing it.
// No input or effects; output is the retained-header fixture for replay tests.
func (r *historicalNormalizeReader) Header() normalize.BundleHeader { return r.header }

// Next supplies EOF or a simulated integrity failure for the existing reader seam.
// Input is a context; output is the stream outcome. It records reads in memory;
// pick for proving replay consumes validation through the accounting trailer.
func (r *historicalNormalizeReader) Next(context.Context) (normalize.RecordEnvelope, error) {
	r.reads++
	if r.nextErr != nil {
		return normalize.RecordEnvelope{}, r.nextErr
	}
	return normalize.RecordEnvelope{}, io.EOF
}
// Close records fixture cleanup without external effects.
// No input; output is nil. Pick for replay reader lifetime assertions.
func (r *historicalNormalizeReader) Close() error { r.closed = true; return nil }

// TestNormalizeReplayValidatesHistoricalSourceAndBundle proves prior receipt recovery is fail-closed.
// Inputs are synthetic SQL receipts and historical headers/integrity outcomes.
// Outputs are original refs or rejection with no writer calls. No database or
// filesystem mutation occurs. Pick for the repository replay boundary.
func TestNormalizeReplayValidatesHistoricalSourceAndBundle(t *testing.T) {
	source, raw, receipt, bundle := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, tc := range []struct {
		name                                     string
		bound                                    bool
		kind                                     string
		changeHeader, streamFailure, openFailure bool
		want                                     bool
	}{
		{"historical 1.1", true, "normalized_bundle", false, false, false, true},
		{"unbound retained object", false, "normalized_bundle", false, false, false, false},
		{"wrong receipt kind", true, "raw_bundle", false, false, false, false},
		{"wrong raw header", true, "normalized_bundle", true, false, false, false},
		{"stream integrity failure", true, "normalized_bundle", false, true, false, false},
		{"preflight digest failure", true, "normalized_bundle", false, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &normalizeReplayDB{row: coverageProofRow{values: []any{receipt, normalizedRefJSON(tc.kind, bundle.String()), tc.bound}}}
			reader := &historicalNormalizeReader{header: normalize.BundleHeader{ContractVersion: normalize.ContractVersion, SourceVersionRef: source.String(), RawGenerationRef: raw.String(), NormalizerID: "generic_message_normalizer", NormalizerVersion: "1.1.0"}}
			if tc.changeHeader {
				reader.header.RawGenerationRef = uuid.NewString()
			}
			if tc.streamFailure {
				reader.nextErr = errors.New("changed during stream")
			}
			opened := 0
			repo := &NormalizedPipelineRepository{db: db, readerFactory: func(context.Context, proffer.Ref) (NormalizedBundleReader, error) {
				opened++
				if tc.openFailure {
					return nil, errors.New("digest mismatch")
				}
				return reader, nil
			}}
			ref, gotReceipt, found, err := repo.LoadPersistedNormalizeExecution(context.Background(), proffer.StageRequest{RequestID: "request", SourceVersionRef: proffer.Ref(source.String()), Refs: map[string]proffer.Ref{"raw_generation": proffer.Ref(raw.String())}})
			if (err == nil && found) != tc.want {
				t.Fatalf("replay result found=%v err=%v", found, err)
			}
			if tc.want && (ref != proffer.Ref(bundle.String()) || gotReceipt != proffer.Ref(receipt.String()) || reader.header.NormalizerVersion != "1.1.0" || reader.reads != 1 || !reader.closed) {
				t.Fatal("historical version/refs/integrity not reused")
			}
			if (!tc.bound || tc.kind != "normalized_bundle") && opened != 0 {
				t.Fatal("unbound reader opened")
			}
			for _, required := range []string{"source.workflow_id=$3", "raw.source_version_id=source.id", "execution.workflow_id=$3", "execution.idempotency_key=$5", "receipt.status='success'", "link.parent_object_id=raw.extraction_bundle_object_id"} {
				if !strings.Contains(db.query, required) {
					t.Fatalf("missing scope %s", required)
				}
			}
			if db.args[4] != "normalize-generation:request:"+source.String()+":"+raw.String() {
				t.Fatal("historical coordinate changed")
			}
		})
	}
}
