// Package approvedgraphai verifies an existing native AI review before analytical graph projection.
package approvedgraphai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/google/uuid"
)

// Scope identifies one actual review decision and its native source candidate.
// Inputs: canonical case, source/candidate/decision IDs and review request digest. Outputs: a projection selector.
// Effects: none. Pick after the owner decision commits, never for candidate staging.
type Scope struct {
	MatterID        string `json:"matter_id"`
	CourtCaseID     string `json:"court_case_id"`
	SourceVersionID string `json:"source_version_id"`
	CandidateID     string `json:"candidate_id"`
	DecisionID      string `json:"decision_id"`
	RequestDigest   string `json:"request_digest"`
}

// Snapshot is the bounded persisted decision, proposal and retained original pin.
// Inputs: reader result. Outputs: verification material. Effects: none.
// Pick for the native AI resolver; normalized SMS records use approvedgraph.Revision.
type Snapshot struct {
	Table            string
	ReviewState      string
	SourceRawTable   string
	SourceRawID      string
	Attrs            []byte
	CandidateSHA256  string
	LedgerDecision   string
	LedgerTargetKind string
	LedgerTargetID   string
	LedgerReviewer   string
	LedgerRationale  []byte
	DecidedAt        time.Time
	SourceID         string
	SourceObjectID   string
	SourceObjectURI  string
	SourceSHA256     string
}

// Reader fetches an actual approved native AI decision with its retained source.
// Inputs: exact scope. Outputs: one snapshot. Effects: read-only.
// Pick for native AI decisions, not the normalized SMS approval reader.
type Reader interface {
	ReadApprovedAI(context.Context, Scope) (Snapshot, error)
}

// ValidateScope checks complete canonical case and bounded stable review identifiers.
// Inputs: request scope. Outputs: error or nil. Effects: none.
// Pick before scheduling and repeat inside the Activity before database access.
func ValidateScope(s Scope) error {
	if !caseidentity.AdmittedIdentity(s.MatterID, s.CourtCaseID) {
		return errors.New("approved AI graph: canonical case required")
	}
	for _, id := range []string{s.SourceVersionID, s.CandidateID, s.DecisionID} {
		if _, err := uuid.Parse(id); err != nil {
			return errors.New("approved AI graph: source, candidate and decision UUIDs required")
		}
	}
	if !digest(s.RequestDigest) {
		return errors.New("approved AI graph: request digest required")
	}
	return nil
}

// WorkflowID derives one retry-stable Temporal ID from the committed review identity.
// Inputs: validated scope. Outputs: deterministic workflow ID or error. Effects: none.
// Pick when the decision API starts projection after committing approval.
func WorkflowID(s Scope) (string, error) {
	if err := ValidateScope(s); err != nil {
		return "", err
	}
	return flow.DeterministicID("approved_ai_graph_workflow", s.DecisionID, s.CandidateID, s.RequestDigest), nil
}

// Resolve reads and verifies the owner ledger, transformed candidate and original source before building a claim.
// Inputs: read-only native AI reader and exact decision scope. Outputs: one cited claim.
// Effects: database reads through Reader only. Pick before the existing ApprovedClaimsSink.
func Resolve(ctx context.Context, reader Reader, scope Scope) (approvedgraph.Claim, error) {
	if err := ValidateScope(scope); err != nil {
		return approvedgraph.Claim{}, err
	}
	if reader == nil {
		return approvedgraph.Claim{}, errors.New("approved AI graph: reader required")
	}
	row, err := reader.ReadApprovedAI(ctx, scope)
	if err != nil {
		return approvedgraph.Claim{}, err
	}
	if row.ReviewState != "approved" || row.LedgerDecision != "approved" || row.LedgerTargetID != scope.CandidateID || row.LedgerTargetKind != row.Table || row.SourceRawTable != "context.source_version" || row.SourceRawID != scope.SourceVersionID || row.DecidedAt.IsZero() || row.SourceID == "" || row.SourceObjectURI == "" || !digest(row.SourceSHA256) || !digest(row.CandidateSHA256) {
		return approvedgraph.Claim{}, errors.New("approved AI graph: review or source pin incomplete")
	}
	var attrs struct {
		Candidate       service.AICandidate `json:"candidate"`
		ReportedKind    string              `json:"reported_kind"`
		ReviewDomain    string              `json:"review_domain"`
		GraphPromotion  string              `json:"graph_promotion"`
		ProjectionScope string              `json:"projection_scope"`
		ReviewDecision  struct {
			DecisionID string `json:"decision_id"`
			Actor      struct {
				SubjectUID string `json:"subject_uid"`
				Username   string `json:"username"`
			} `json:"actor"`
			RequestDigest         string    `json:"request_digest"`
			ApprovedContentSHA256 string    `json:"approved_content_sha256"`
			Decision              string    `json:"decision"`
			At                    time.Time `json:"at"`
		} `json:"review_decision"`
	}
	if err := json.Unmarshal(row.Attrs, &attrs); err != nil {
		return approvedgraph.Claim{}, fmt.Errorf("approved AI graph: candidate attrs: %w", err)
	}
	c := attrs.Candidate
	computed := service.AICandidateDigest(c)
	nativeSource := row.SourceObjectID == "" && c.SourceObjectID == "" && c.SourceRef == row.SourceObjectURI && c.PreparedRef != ""
	retainedSource := row.SourceObjectID != "" && c.SourceObjectID == row.SourceObjectID && c.SourceRef == "" && c.VersionID == nil
	if (!nativeSource && !retainedSource) || hex.EncodeToString(computed[:]) != row.CandidateSHA256 || c.SourceVersionID != scope.SourceVersionID || c.SourceSHA256 != row.SourceSHA256 || c.ReportedKind != attrs.ReportedKind || c.ReviewDomain != attrs.ReviewDomain || (c.ReviewDomain != "ai_chat_content" && c.ReviewDomain != "ai_chat_account") || attrs.GraphPromotion != "approved_context" || attrs.ProjectionScope != "context" || attrs.ReviewDecision.DecisionID != scope.DecisionID || attrs.ReviewDecision.RequestDigest != scope.RequestDigest || attrs.ReviewDecision.ApprovedContentSHA256 != row.CandidateSHA256 || attrs.ReviewDecision.Decision != "approved" || attrs.ReviewDecision.At.IsZero() || attrs.ReviewDecision.At.UTC().UnixMicro() != row.DecidedAt.UTC().UnixMicro() || attrs.ReviewDecision.Actor.SubjectUID == "" || attrs.ReviewDecision.Actor.Username != row.LedgerReviewer {
		return approvedgraph.Claim{}, errors.New("approved AI graph: candidate hash or review receipt differs")
	}
	var rationale struct {
		SourceVersionID string `json:"source_version_id"`
		CandidateSHA256 string `json:"candidate_content_sha256"`
		ActorUID        string `json:"actor_subject_uid"`
		RequestDigest   string `json:"request_digest"`
		CandidateKind   string `json:"candidate_kind"`
		ReportedKind    string `json:"reported_kind"`
		ReviewDomain    string `json:"review_domain"`
		GraphPromotion  string `json:"graph_promotion"`
		ProjectionScope string `json:"projection_scope"`
	}
	if err := json.Unmarshal(row.LedgerRationale, &rationale); err != nil {
		return approvedgraph.Claim{}, fmt.Errorf("approved AI graph: ledger rationale: %w", err)
	}
	if rationale.SourceVersionID != scope.SourceVersionID || rationale.CandidateSHA256 != row.CandidateSHA256 || rationale.ActorUID != attrs.ReviewDecision.Actor.SubjectUID || rationale.RequestDigest != scope.RequestDigest || rationale.CandidateKind != strings.TrimPrefix(row.Table, "working.candidate_") || rationale.ReportedKind != c.ReportedKind || rationale.ReviewDomain != c.ReviewDomain || rationale.GraphPromotion != "approved_context" || rationale.ProjectionScope != "context" || scope.DecisionID != flow.DeterministicID("ai_review_decision", scope.CandidateID, scope.RequestDigest) {
		return approvedgraph.Claim{}, errors.New("approved AI graph: ledger does not bind exact candidate and actor")
	}
	if c.NativeJSONPointer != "" && !strings.HasPrefix(c.NativeJSONPointer, "/") {
		return approvedgraph.Claim{}, errors.New("approved AI graph: malformed native pointer")
	}
	if c.SourceSpan.Start < 0 || c.SourceSpan.End <= c.SourceSpan.Start || !digest(c.SourceSpan.SHA256) || c.SpanUnit != "unicode_codepoint" || c.EvidenceQuote == "" {
		return approvedgraph.Claim{}, errors.New("approved AI graph: native span incomplete")
	}
	kind, text, predicate := "", "", ""
	switch row.Table {
	case "working.candidate_entity":
		if c.Kind != "entity" {
			return approvedgraph.Claim{}, errors.New("approved AI graph: entity kind mismatch")
		}
		kind, text = "entity_mention", c.Name
	case "working.candidate_event":
		if c.Kind != "event" {
			return approvedgraph.Claim{}, errors.New("approved AI graph: event kind mismatch")
		}
		kind, text = "event_account", c.Statement
	case "working.candidate_fact":
		if c.Kind != "fact" || strings.TrimSpace(c.Predicate) == "" {
			return approvedgraph.Claim{}, errors.New("approved AI graph: fact predicate missing")
		}
		kind, text, predicate = "statement", c.Statement, c.Predicate
	default:
		return approvedgraph.Claim{}, errors.New("approved AI graph: unsupported candidate table")
	}
	if strings.TrimSpace(text) == "" {
		return approvedgraph.Claim{}, errors.New("approved AI graph: exact typed text missing")
	}
	locator, _ := json.Marshal(struct {
		Version   string `json:"source_version_id"`
		Object    string `json:"source_object_id"`
		SourceRef string `json:"source_ref,omitempty"`
		SHA       string `json:"source_sha256"`
		Pointer   string `json:"native_json_pointer"`
		Start     int    `json:"start"`
		End       int    `json:"end"`
		SpanSHA   string `json:"span_sha256"`
	}{scope.SourceVersionID, row.SourceObjectID, c.SourceRef, row.SourceSHA256, c.NativeJSONPointer, c.SourceSpan.Start, c.SourceSpan.End, c.SourceSpan.SHA256})
	locatorHash := sha256.Sum256(locator)
	locatorID := flow.DeterministicID("approved_ai_native_locator", scope.SourceVersionID, c.NativeJSONPointer, fmt.Sprint(c.SourceSpan.Start), fmt.Sprint(c.SourceSpan.End), c.SourceSpan.SHA256)
	start, end := c.SourceSpan.Start, c.SourceSpan.End
	return approvedgraph.Claim{ID: flow.DeterministicID("approved_ai_graph_claim", scope.DecisionID, scope.CandidateID, locatorID), Kind: kind, Text: text, MatterID: scope.MatterID, CourtCaseID: scope.CourtCaseID, ReceiptID: scope.DecisionID, ApprovalDigest: scope.RequestDigest, ControlGenerationID: flow.DeterministicID("approved_graph_generation", scope.DecisionID, row.CandidateSHA256), CandidateID: scope.CandidateID, CandidateSHA256: row.CandidateSHA256, SourceID: row.SourceID, SourceVersionID: scope.SourceVersionID, SourceObjectID: row.SourceObjectID, SourceObjectURI: row.SourceObjectURI, SourceSHA256: row.SourceSHA256, RecordID: locatorID, RecordSHA256: hex.EncodeToString(locatorHash[:]), Predicate: predicate, NativeJSONPointer: c.NativeJSONPointer, NativeSpanStart: &start, NativeSpanEnd: &end, NativeSpanUnit: c.SpanUnit, NativeSpanSHA256: c.SourceSpan.SHA256, OccurredAt: utcClock(c.OccurredAt), SourceAvailableFrom: utcClock(c.SourceAvailableFrom), ApprovedAt: row.DecidedAt.UTC(), ApprovedBy: attrs.ReviewDecision.Actor.SubjectUID}, nil
}

// utcClock canonicalizes a verified claim clock without changing its instant or candidate digest.
// Inputs: optional native clock. Outputs: UTC copy or nil. Effects: none.
// Pick before analytical hashing because SurrealDB datetime readback uses UTC.
func utcClock(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

// digest checks a full lowercase hexadecimal SHA256 without exposing source bytes.
// Inputs: candidate digest. Outputs: validity. Effects: none. Pick for source and review pins.
func digest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
