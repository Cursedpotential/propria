// Byline: Codex · GPT-6-Sol · 2026-10-05.
package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// aiSourceBoundaryDB supplies persisted verified provenance and counts platform lookups.
// Inputs: synthetic source/generation/request and two persisted formats. Outputs: fixture rows.
// Effects: test counters only. Use to prove request labels cannot override persisted AI provenance.
type aiSourceBoundaryDB struct {
	DB
	request          proffer.StageRequest
	generation       uuid.UUID
	declared, format string
	wrongOwner       bool
	platformQueries  int
}

// TestAIContextStoreDirectCommitRefused blocks directly supplied AI plans before any SQL access.
// Inputs: synthetic AI plan with no connected database. Outputs: assertions.
// Effects: none. Use to defend both public commit methods from callers that bypass Build.
func TestAIContextStoreDirectCommitRefused(t *testing.T) {
	s := &FirstPartyContextStore{}
	spec := activities.FirstPartyCommitSpec{Plan: firstparty.Plan{Source: firstparty.Source{DeclaredFormat: "chatgpt_official_json"}}}
	for _, write := range []func(context.Context, activities.FirstPartyCommitSpec) (proffer.Ref, proffer.Ref, error){s.CommitFirstPartyMessages, s.CommitFirstPartyContextThreads} {
		if _, _, err := write(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "AI chat") {
			t.Fatalf("direct AI commit error=%v", err)
		}
	}
}

func (d *aiSourceBoundaryDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if strings.Contains(query, "context.reconciliation_receipt") {
		return reviewFakeRow{values: []any{d.generation, "success"}}
	}
	if strings.Contains(query, "FROM context.normalized_generation generation") {
		workflow := d.request.RequestID
		if d.wrongOwner {
			workflow = "different-run"
		}
		return reviewFakeRow{values: []any{uuid.MustParse(string(d.request.SourceVersionRef)), workflow, "retained", d.declared, "b2://synthetic/export.json", uuid.NullUUID{UUID: uuid.MustParse(d.request.MatterID), Valid: true}, uuid.NullUUID{UUID: uuid.MustParse(d.request.CourtCaseID), Valid: true}, d.format}}
	}
	d.platformQueries++
	return reviewFakeRow{err: pgx.ErrNoRows}
}

func (d *aiSourceBoundaryDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &reviewFakeRows{rows: [][]any{{"01a0c181-0000-7000-8000-000000000001", "message", int64(0), pgtype.Timestamptz{}, []byte(`{"content":{"body":"synthetic AI message"},"participants":[{"role":"sender","identifier":"assistant"}]}`)}}}, nil
}

// TestAIContextStoreUsesPersistedFormats checks exclusions after ownership proof while retaining shared search inputs.
// Inputs: synthetic verified source rows. Outputs: assertions. Effects: no database or object writes.
// Use for persisted declared/raw classification and a spoofed incoming Activity label.
func TestAIContextStoreUsesPersistedFormats(t *testing.T) {
	for _, tc := range []struct {
		name, declared, format, request string
		excluded                        bool
	}{
		{"declared", "chatgpt_official_json", "generic_message", "ndjson", true},
		{"raw", "json", "claude_conversations_json", "ndjson", true},
		{"request-label-cannot-exclude", "json", "generic_message", "chatgpt_official_json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := proffer.StageRequest{RequestID: "synthetic-ai-source", SourceVersionRef: "01a0c181-02a6-7217-93ee-9512956dfc56", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", DeclaredFormat: tc.request}
			db := &aiSourceBoundaryDB{request: req, generation: uuid.MustParse("01a0c181-02a6-7217-93ee-9512956dfc57"), declared: tc.declared, format: tc.format}
			store, err := NewFirstPartyContextStore(db)
			if err != nil {
				t.Fatal(err)
			}
			input, err := store.LoadFirstPartyContext(context.Background(), req, proffer.Ref(db.generation.String()), "01a0c181-02a6-7217-93ee-9512956dfc58")
			if err != nil || (input.NotApplicable != "") != tc.excluded || len(input.Messages) != 1 || len(input.StatedIdentifiers) != 1 {
				t.Fatalf("input=%+v error=%v", input, err)
			}
			if tc.excluded && db.platformQueries != 0 {
				t.Fatalf("AI source entered messaging inference: %d queries", db.platformQueries)
			}
			db.wrongOwner = true
			if _, err := store.LoadFirstPartyContext(context.Background(), req, proffer.Ref(db.generation.String()), "01a0c181-02a6-7217-93ee-9512956dfc58"); err == nil {
				t.Fatal("foreign generation bypassed ownership proof")
			}
		})
	}
}
