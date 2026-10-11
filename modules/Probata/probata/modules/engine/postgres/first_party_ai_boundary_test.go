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

// humanCommitProofTx supplies persisted formats for the direct-commit source check and rejects any other SQL.
// Inputs: immutable synthetic source rows. Outputs: provenance fixture. Effects: none.
// Choose to exercise a spoofed SMS Plan without a database or receipt/write transaction.
type humanCommitProofTx struct {
	pgx.Tx
	source           uuid.UUID
	declared, format string
	foreign          bool
}

func (tx humanCommitProofTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if !strings.Contains(sql, "JOIN context.raw_generation raw") {
		return reviewFakeRow{err: pgx.ErrNoRows}
	}
	source := tx.source
	if tx.foreign {
		source = uuid.New()
	}
	return reviewFakeRow{values: []any{source, tx.declared, tx.format, "retained", "synthetic-run"}}
}

// TestAINeutralDirectCommitSourceProofCannotTrustAnSMSPlan checks persisted provenance wins over a supplied human Plan.
// Inputs: synthetic declared/raw AI and human formats with an SMS Plan. Outputs: assertions. Effects: no SQL writes.
// Choose for the proof shared by both direct SQL commit methods, independently of Activity rebuild validation.
func TestAINeutralDirectCommitSourceProofCannotTrustAnSMSPlan(t *testing.T) {
	source, generation := uuid.New(), uuid.New()
	spec := activities.FirstPartyCommitSpec{RequestID: "synthetic-run", Plan: firstparty.Plan{Source: firstparty.Source{SourceVersionID: source.String(), NormalizedGenerationID: generation.String(), DeclaredFormat: "smsbackuprestore_xml"}}}
	for _, tc := range []struct {
		declared, raw string
		ai            bool
	}{{"json", "chatgpt_official_json", true}, {"claude_conversations_json", "generic_message", true}, {"smsbackuprestore_xml", "smsbackuprestore_xml", false}} {
		tx := humanCommitProofTx{source: source, declared: tc.declared, format: tc.raw}
		err := verifyHumanCommitSource(context.Background(), tx, source, spec)
		if (err != nil) != tc.ai {
			t.Fatalf("AI=%v error=%v", tc.ai, err)
		}
		tx.foreign = true
		if err := verifyHumanCommitSource(context.Background(), tx, source, spec); err == nil {
			t.Fatal("foreign generation accepted")
		}
	}
}

// aiDirectCommitDB models a valid existing human gate backed by an actually AI raw generation.
// Inputs: source coordinates and expected gate kind. Outputs: synthetic gate/provenance rows. Effects: read counters only.
// Choose to exercise the public SQL commit entry points, not merely their shared verification helper.
type aiDirectCommitDB struct {
	DB
	tx       *aiDirectCommitTx
	source   uuid.UUID
	gateKind string
	begins   int
}

func (db *aiDirectCommitDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return reviewFakeRow{values: []any{db.source, []byte(`{"ref_kind":"` + db.gateKind + `","plan_digest":"synthetic-plan"}`)}}
}
func (db *aiDirectCommitDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	db.begins++
	return db.tx, nil
}

// aiDirectCommitTx records provenance reads and rollback; every unimplemented write path would fail the test.
// Inputs: retained AI source. Outputs: fixture row. Effects: counters only. Choose to prove refusal precedes execution writes.
type aiDirectCommitTx struct {
	pgx.Tx
	source           uuid.UUID
	reads, rollbacks int
}

func (tx *aiDirectCommitTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.reads++
	if !strings.Contains(sql, "JOIN context.raw_generation raw") {
		return reviewFakeRow{err: pgx.ErrNoRows}
	}
	return reviewFakeRow{values: []any{tx.source, "json", "chatgpt_official_json", "retained", "synthetic-run"}}
}
func (tx *aiDirectCommitTx) Rollback(context.Context) error { tx.rollbacks++; return nil }

// TestAINeutralPublicSQLCommitsRejectPersistedRawAIBehindFabricatedSMSPlan covers both direct SQL methods end to end.
// Inputs: a valid matching gate/digest, caller-supplied SMS Plan and independently stored AI raw format. Outputs: assertions.
// Effects: synthetic transaction counters only; no database access. Choose to defend callers bypassing Activity rebuilds.
func TestAINeutralPublicSQLCommitsRejectPersistedRawAIBehindFabricatedSMSPlan(t *testing.T) {
	for _, threads := range []bool{false, true} {
		source, generation := uuid.New(), uuid.New()
		tx := &aiDirectCommitTx{source: source}
		db := &aiDirectCommitDB{source: source, tx: tx, gateKind: activities.FirstPartyConfirmationKind}
		if threads {
			db.gateKind = activities.FirstPartyMessagesKind
		}
		store, _ := NewFirstPartyContextStore(db)
		spec := activities.FirstPartyCommitSpec{RequestID: "synthetic-run", SourceVersionRef: proffer.Ref(source.String()), GateRef: proffer.Ref(uuid.NewString()), Plan: firstparty.Plan{Digest: "synthetic-plan", Source: firstparty.Source{SourceVersionID: source.String(), NormalizedGenerationID: generation.String(), DeclaredFormat: "smsbackuprestore_xml"}}}
		write := store.CommitFirstPartyMessages
		if threads {
			write = store.CommitFirstPartyContextThreads
		}
		if _, _, err := write(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "AI chat") || db.begins != 1 || tx.reads != 1 || tx.rollbacks != 1 {
			t.Fatalf("threads=%v error=%v begins=%d reads=%d rollbacks=%d", threads, err, db.begins, tx.reads, tx.rollbacks)
		}
	}
}
