package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/jackc/pgx/v5"
)

type currentTestDB struct {
	scope          surrealsink.ApprovedQueryScope
	target, source string
	generic        bool
	failure        error
}

// BeginTx rejects writes from the read-only guard fixture.
// Inputs: context/options. Outputs: error. Effects: none; choose in freshness tests.
func (d *currentTestDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("unexpected write")
}

// Query rejects unbounded multirow reads from the ledger-classification fixture.
// Inputs: SQL. Outputs: error. Effects: none; choose in guard tests.
func (d *currentTestDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected multirow read")
}

// QueryRow supplies only the existing native ledger or retained-message commit shape.
// Inputs: selected existing SQL. Outputs: row fixture. Effects: none; choose for route compatibility checks.
func (d *currentTestDB) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return currentTestRow{d: d, sql: sql}
}

type currentTestRow struct {
	d   *currentTestDB
	sql string
}

// Scan fills fixed ledger/receipt fields and rejects any unexpected read shape.
// Inputs: destinations. Outputs: scan or error. Effects: fixture writes only; choose for source-only tests.
func (r currentTestRow) Scan(dest ...any) error {
	d := r.d
	if d.failure != nil {
		return d.failure
	}
	if strings.Contains(r.sql, "analysis.review_decision") {
		if d.generic {
			return pgx.ErrNoRows
		}
		*dest[0].(*string) = d.target
		*dest[1].(*string) = "working.candidate_fact"
		*dest[2].(*[]byte), _ = json.Marshal(map[string]string{"source_version_id": d.source, "request_digest": d.scope.ApprovalDigest})
		return nil
	}
	if d.generic && len(dest) == 8 {
		for i, value := range []string{d.scope.ApprovedRevisionID, "completed", "committed", "preview:existing generation:existing source:existing", d.scope.ApprovalDigest, "LIVE"} {
			*dest[i].(*string) = value
		}
		*dest[6].(*[]byte) = []byte(`{"subject_uid":"owner","username":"owner"}`)
		*dest[7].(*time.Time) = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
		return nil
	}
	return errors.New("unexpected row shape")
}

type currentTestNative struct{ row approvedgraphai.Snapshot }

// ReadApprovedAI returns the currently selected fixture receipt.
// Inputs: scope. Outputs: snapshot. Effects: none; choose for supersession races.
func (r *currentTestNative) ReadApprovedAI(context.Context, approvedgraphai.Scope) (approvedgraphai.Snapshot, error) {
	return r.row, nil
}

type currentTestGraph struct {
	calls  int
	claim  approvedgraph.Claim
	during func()
}

// QueryApprovedClaims records whether current-decision rejection happened before graph retrieval.
// Inputs: selected scope. Outputs: one fixture claim. Effects: optional fixture mutation; choose for concurrent revision tests.
func (g *currentTestGraph) QueryApprovedClaims(context.Context, surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error) {
	g.calls++
	if g.during != nil {
		g.during()
	}
	return surrealsink.ApprovedQueryResult{Claims: []approvedgraph.Claim{g.claim}}, nil
}

// ListApprovedProjections returns an empty fixture page without source bodies.
// Inputs: list scope. Outputs: empty page. Effects: none; choose for wrapper interface completeness.
func (g *currentTestGraph) ListApprovedProjections(context.Context, surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error) {
	return surrealsink.ApprovedProjectionPage{}, nil
}

// TestCurrentReaderRejectsSupersessionBeforeAndDuringGraphRead verifies stale native decisions never escape the guarded query.
// Inputs: exact approved native receipt and concurrent owner revision. Outputs: rejection before/after graph read.
// Effects: fixture state only; choose for query/walk reuse and original source-pin integrity.
func TestCurrentReaderRejectsSupersessionBeforeAndDuringGraphRead(t *testing.T) {
	q, row, target, source := currentFixture(t)
	db := &currentTestDB{scope: q, target: target, source: source}
	native := &currentTestNative{row: row}
	claim, err := approvedgraphai.Resolve(context.Background(), native, approvedgraphai.Scope{MatterID: q.MatterID, CourtCaseID: q.CourtCaseID, SourceVersionID: source, CandidateID: target, DecisionID: q.ApprovedRevisionID, RequestDigest: q.ApprovalDigest})
	if err != nil {
		t.Fatal(err)
	}
	g := &currentTestGraph{claim: claim}
	r := &ApprovedCurrentReader{db: db, native: native, Graph: g}
	if _, err := r.QueryApprovedClaims(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	native.row.ReviewState = "rejected"
	g.calls = 0
	if _, err := r.QueryApprovedClaims(context.Background(), q); !errors.Is(err, ErrSupersededApprovedDecision) || g.calls != 0 {
		t.Fatalf("stale prefilter: %v calls=%d", err, g.calls)
	}
	native.row = row
	g.during = func() { native.row.ReviewState = "rejected" }
	if _, err := r.QueryApprovedClaims(context.Background(), q); !errors.Is(err, ErrSupersededApprovedDecision) {
		t.Fatalf("concurrent revision escaped: %v", err)
	}
	native.row = row
	g.during = nil
	g.claim.SourceSHA256 = strings.Repeat("e", 64)
	if _, err := r.QueryApprovedClaims(context.Background(), q); err == nil {
		t.Fatal("graph original pin drift admitted")
	}
	db.failure = errors.New("database unavailable")
	if err := r.CheckCurrentApprovedProjection(context.Background(), q); err == nil || errors.Is(err, ErrSupersededApprovedDecision) {
		t.Fatal("backend failure mislabeled supersession")
	}
}

// TestCurrentReaderPreservesRetainedMessageApprovals checks the existing extraction-commit route without requiring a native ledger.
// Inputs: actual legacy receipt shape and exact generation/digest. Outputs: eligible query or rejected changed digest.
// Effects: fixture state only; choose for backward compatibility with retained-message approvals.
func TestCurrentReaderPreservesRetainedMessageApprovals(t *testing.T) {
	q, _, _, _ := currentFixture(t)
	q.ProjectionGenerationID = flow.DeterministicID("approved_graph_generation", q.ApprovedRevisionID, q.ApprovalDigest)
	db := &currentTestDB{scope: q, generic: true}
	g := &currentTestGraph{}
	r := &ApprovedCurrentReader{db: db, Graph: g}
	if _, err := r.QueryApprovedClaims(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	q.ApprovalDigest = strings.Repeat("f", 64)
	if _, err := r.QueryApprovedClaims(context.Background(), q); err == nil {
		t.Fatal("legacy changed receipt digest admitted")
	}
}

// currentFixture constructs a digest-bound native candidate and its existing receipt vocabulary.
// Inputs: test handle. Outputs: selected scope and exact persisted snapshot. Effects: none; choose for guard tests only.
func currentFixture(t *testing.T) (surrealsink.ApprovedQueryScope, approvedgraphai.Snapshot, string, string) {
	t.Helper()
	target := "00000000-0000-0000-0000-000000000004"
	source := "00000000-0000-0000-0000-000000000003"
	object := "00000000-0000-0000-0000-000000000005"
	digest := strings.Repeat("a", 64)
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	c := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: source, SourceObjectID: object, SourceSHA256: strings.Repeat("b", 64)}, Kind: "fact", ReportedKind: "fact", ReviewDomain: "ai_chat_content", Predicate: "account", Statement: "Fixture account", EvidenceQuote: "text", SpanUnit: "unicode_codepoint", SourceSpan: service.AISourceSpan{Start: 0, End: 4, SHA256: strings.Repeat("c", 64)}}
	h := service.AICandidateDigest(c)
	hash := hex.EncodeToString(h[:])
	id := flow.DeterministicID("ai_review_decision", target, digest)
	attrs, _ := json.Marshal(map[string]any{"candidate": c, "reported_kind": c.ReportedKind, "review_domain": c.ReviewDomain, "graph_promotion": "approved_context", "projection_scope": "context", "review_decision": map[string]any{"decision_id": id, "actor": map[string]string{"subject_uid": "owner", "username": "owner"}, "request_digest": digest, "approved_content_sha256": hash, "decision": "approved", "at": at}})
	rationale, _ := json.Marshal(aiReviewRationale(source, hash, "owner", digest, "working.candidate_fact", c.ReportedKind, c.ReviewDomain, "approved_context"))
	row := approvedgraphai.Snapshot{Table: "working.candidate_fact", ReviewState: "approved", SourceRawTable: "context.source_version", SourceRawID: source, Attrs: attrs, CandidateSHA256: hash, LedgerDecision: "approved", LedgerTargetKind: "working.candidate_fact", LedgerTargetID: target, LedgerReviewer: "owner", LedgerRationale: rationale, DecidedAt: at, SourceID: "00000000-0000-0000-0000-000000000002", SourceObjectID: object, SourceObjectURI: "b2://fixture/source", SourceSHA256: c.SourceSHA256}
	return surrealsink.ApprovedQueryScope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, ApprovedRevisionID: id, ApprovalDigest: digest, ProjectionGenerationID: flow.DeterministicID("approved_graph_generation", id, hash)}, row, target, source
}
