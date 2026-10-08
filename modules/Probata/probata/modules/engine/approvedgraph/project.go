// Byline: Codex · GPT-6 · 2026-10-07
package approvedgraph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// Claim is one exact owner-committed candidate assertion with a root source citation.
type Claim struct {
	ID                  string     `json:"id"`
	Kind                string     `json:"kind"`
	Text                string     `json:"text"`
	MatterID            string     `json:"matter_id"`
	CourtCaseID         string     `json:"court_case_id"`
	ReceiptID           string     `json:"approved_revision_id"`
	ApprovalDigest      string     `json:"approval_digest"`
	ControlGenerationID string     `json:"control_generation_id"`
	CandidateID         string     `json:"candidate_id"`
	CandidateSHA256     string     `json:"candidate_sha256"`
	SourceID            string     `json:"source_id"`
	SourceVersionID     string     `json:"source_version_id"`
	SourceObjectID      string     `json:"source_object_id"`
	SourceObjectURI     string     `json:"source_object_uri"`
	SourceSHA256        string     `json:"source_sha256"`
	RecordID            string     `json:"record_id"`
	RecordSHA256        string     `json:"record_sha256"`
	OccurredAt          *time.Time `json:"occurred_at,omitempty"`
	SourceAvailableFrom time.Time  `json:"source_available_from"`
	ApprovedAt          time.Time  `json:"approved_at"`
	ApprovedBy          string     `json:"approved_by"`
}

// Sink applies idempotent approved assertions to the governed analysis graph.
// Inputs: exact claims with stable IDs. Outputs: write error only.
// Effects: graph projection. Pick after Resolve and BuildClaims, never from raw turns.
type Sink interface {
	ApplyApprovedClaims(context.Context, []Claim) error
}

// BuildClaims converts a verified revision into cited entity and event assertions.
// Inputs: a revision from Resolve. Outputs: bounded graph claims.
// Effects: none. Pick before Sink.ApplyApprovedClaims to keep projection retryable.
func BuildClaims(revision Revision) ([]Claim, error) {
	if revision.Source.ObjectURI == "" || !validDigest(revision.Source.SHA256) || revision.Receipt.ID == "" {
		return nil, errors.New("approved graph: revision has no recorded source object citation")
	}
	makeClaim := func(kind, text, candidateID, candidateHash, recordID string, occurred *time.Time) (Claim, error) {
		pin, exists := revision.Records[recordID]
		if text == "" || candidateID == "" || recordID == "" || !exists || pin.SourceAvailableFrom == nil || !validDigest(pin.SHA256) {
			return Claim{}, fmt.Errorf("approved graph: %s assertion lacks text, record locator, or availability", kind)
		}
		return Claim{
			ID:   flow.DeterministicID("approved_graph_claim", revision.Receipt.ID, kind, candidateID, recordID),
			Kind: kind, Text: text, MatterID: revision.Scope.MatterID, CourtCaseID: revision.Scope.CourtCaseID,
			ReceiptID: revision.Receipt.ID, ApprovalDigest: revision.Receipt.Digest,
			ControlGenerationID: flow.DeterministicID("approved_graph_generation", revision.Receipt.ID, revision.Receipt.Digest),
			CandidateID:         candidateID, CandidateSHA256: candidateHash,
			SourceID: revision.Source.SourceID, SourceVersionID: revision.Scope.SourceVersionID, SourceObjectID: revision.Source.ObjectID,
			SourceObjectURI: revision.Source.ObjectURI, SourceSHA256: revision.Source.SHA256,
			RecordID: recordID, RecordSHA256: pin.SHA256, OccurredAt: occurred,
			SourceAvailableFrom: *pin.SourceAvailableFrom, ApprovedAt: revision.Receipt.FinishedAt,
			ApprovedBy: revision.Receipt.Actor.SubjectUID,
		}, nil
	}
	var claims []Claim
	for _, entity := range revision.Entities {
		mentions := entity.Proposal.ModelMentions
		if len(mentions) == 0 {
			return nil, fmt.Errorf("approved graph: entity %s has no record locator", entity.Proposal.CandidateID)
		}
		for _, mention := range mentions {
			pin := revision.Records[mention.RecordID]
			claim, err := makeClaim("entity_mention", entity.Proposal.Name, entity.Proposal.CandidateID, entity.ContentHex, mention.RecordID, pin.OccurredAt)
			if err != nil {
				return nil, err
			}
			claims = append(claims, claim)
		}
	}
	for _, event := range revision.Events {
		if len(event.Proposal.SourceRecords) == 0 {
			return nil, fmt.Errorf("approved graph: event %s has no record locator", event.Proposal.CandidateID)
		}
		for _, record := range event.Proposal.SourceRecords {
			claim, err := makeClaim("event_account", event.Proposal.Title, event.Proposal.CandidateID, event.ContentHex, record.RecordID, event.Proposal.OccurredAt)
			if err != nil {
				return nil, err
			}
			claims = append(claims, claim)
		}
	}
	return claims, nil
}

// ClaimSetDigest hashes the complete, ID-sorted approved projection without source bodies.
// Inputs: one complete claim set. Outputs: control-plane generation hash. Effects: none.
// Pick for completion ledgers and query snapshot selection, not for source custody hashing.
func ClaimSetDigest(claims []Claim) string {
	ordered := append([]Claim(nil), claims...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	raw, _ := json.Marshal(ordered)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
