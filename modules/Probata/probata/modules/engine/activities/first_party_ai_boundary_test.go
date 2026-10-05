// Byline: Codex · GPT-6-Sol · 2026-10-05.
package activities

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// aiBoundaryStore models verified inputs and captures receipts, rejecting unplanned store calls.
// Inputs: synthetic source and gate. Outputs: the same verified source and captured receipt.
// Effects: test memory only. Use for the AI exclusion boundary rather than persistence integration.
type aiBoundaryStore struct {
	FirstPartyContextStore
	input         FirstPartyContextInput
	identity      contextthread.Identity
	receipt       FirstPartyReceipt
	specs         []FirstPartyReceiptSpec
	identityCalls int
}

func (s *aiBoundaryStore) LoadFirstPartyContext(context.Context, proffer.StageRequest, proffer.Ref, proffer.Ref) (FirstPartyContextInput, error) {
	return s.input, nil
}
func (s *aiBoundaryStore) ResolveFirstPartyIdentity(context.Context, proffer.StageRequest, string, string) (contextthread.Identity, error) {
	s.identityCalls++
	return s.identity, nil
}
func (s *aiBoundaryStore) LoadParticipantResolution(context.Context, proffer.Ref) (disclosure.Resolution, error) {
	return disclosure.Resolution{Basis: disclosure.Basis, OwnerPersonID: s.identity.OwnerPersonID, PerspectivePersonID: s.identity.PerspectivePersonID, Identifiers: []disclosure.ResolvedIdentifier{{Raw: "+18105550100", Normalized: "8105550100", EntityID: s.identity.OwnerPersonID, IsOwner: true}}}, nil
}
func (s *aiBoundaryStore) PersistFirstPartyReceipt(_ context.Context, spec FirstPartyReceiptSpec) (proffer.Ref, proffer.Ref, error) {
	s.specs = append(s.specs, spec)
	return "receipt", "receipt", nil
}
func (s *aiBoundaryStore) LoadFirstPartyReceipt(context.Context, string, proffer.Ref) (FirstPartyReceipt, error) {
	return s.receipt, nil
}

// aiBoundaryFixture supplies a valid SMS baseline that an AI source label cannot project.
// Inputs: none. Outputs: synthetic store and reference-only request. Effects: none.
// Use to compare exclusion, ordinary messaging and stale-success gate rebuilds.
func aiBoundaryFixture() (*aiBoundaryStore, proffer.StageRequest) {
	id := contextthread.Identity{OwnerPersonID: "01a0f751-e07b-76b6-afcb-63acfbba373e", PerspectivePersonID: "01a0f751-e07b-76c7-8c0f-65692ad656b8", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c"}
	at := time.Date(2024, 11, 1, 10, 0, 0, 0, time.UTC)
	s := &aiBoundaryStore{identity: id, input: FirstPartyContextInput{PlatformResolved: true, Source: firstparty.Source{SourceVersionID: "01a0c181-02a6-7217-93ee-9512956dfc56", NormalizedGenerationID: "01a0c181-02a6-7217-93ee-9512956dfc57", DeclaredFormat: "ndjson", SourceKey: "b2://synthetic/sms.derived/threads/example.ndjson", Platform: "sms", CaptureKind: "sms_backup_restore", RepresentationKind: "json"}, Messages: []firstparty.SourceMessage{{RecordID: "01a0c181-0000-7000-8000-000000000001", Ordinal: 0, OccurredAt: &at, Sender: "self", Recipients: []string{"+18105550100"}, Parties: []string{"self", "+18105550100"}, Body: "synthetic message"}}}}
	r := proffer.StageRequest{RequestID: "synthetic-ai-boundary", SourceVersionRef: proffer.Ref(s.input.Source.SourceVersionID), MatterID: id.MatterID, CourtCaseID: id.CourtCaseID, DeclaredFormat: "ndjson", Refs: map[string]proffer.Ref{"normalized_generation": proffer.Ref(s.input.Source.NormalizedGenerationID), "normalized_verification": "verification", "participant_resolution": "resolution", "owner_person": proffer.Ref(id.OwnerPersonID), "perspective_person": proffer.Ref(id.PerspectivePersonID), "context_proposal": "proposal", "context_confirmation": "confirmation", "context_messages": "messages"}}
	s.receipt = FirstPartyReceipt{SourceVersionRef: r.SourceVersionRef, NormalizedGenerationRef: r.Refs["normalized_generation"], ResolutionRef: "resolution", Identity: id}
	return s, r
}

// TestAIContextProposalExclusion proves AI messages record an exclusion without identity resolution or projection.
// Inputs: synthetic persisted AI format with a conflicting request label. Outputs: assertions.
// Effects: in-memory receipts only. Use to protect the pre-preview boundary.
func TestAIContextProposalExclusion(t *testing.T) {
	for _, format := range []string{"chatgpt_official_json", "claude_conversations_json", "gemini_activity_json", "ai_markdown_transcript", "ai_chat_file"} {
		t.Run(format, func(t *testing.T) {
			s, r := aiBoundaryFixture()
			s.input.Source.DeclaredFormat = format
			delete(r.Refs, "owner_person")
			delete(r.Refs, "perspective_person")
			delete(r.Refs, "participant_resolution")
			acts := FirstPartyContextActivities{Store: s}
			for attempt := 0; attempt < 2; attempt++ {
				got, err := acts.ProposeFirstPartyContext(context.Background(), r)
				if err != nil || got.Status != proffer.StatusNotApplicable || got.ReceiptRef == "" || !strings.Contains(got.Reason, "AI chat") {
					t.Fatalf("result=%+v error=%v", got, err)
				}
			}
			if s.identityCalls != 0 || len(s.specs) != 2 {
				t.Fatalf("unexpected identity calls/receipts: %d/%d", s.identityCalls, len(s.specs))
			}
			for _, spec := range s.specs {
				if spec.NotApplicable == "" || spec.PlanDigest != "" || spec.Messages != 0 {
					t.Fatalf("projection proposed: %+v", spec)
				}
			}
		})
	}
}

// TestAIContextVerifiedRawFormatExclusion protects AI formats identified by the raw generation rather than its label.
// Inputs: synthetic store exclusion. Outputs: assertions. Effects: in-memory receipt only.
// Use where the persisted declared format is generic but the verified raw format is AI.
func TestAIContextVerifiedRawFormatExclusion(t *testing.T) {
	s, r := aiBoundaryFixture()
	s.input.NotApplicable = "AI chat sources remain AI context and are not first-party messaging"
	got, err := (FirstPartyContextActivities{Store: s}).ProposeFirstPartyContext(context.Background(), r)
	if err != nil || got.Status != proffer.StatusNotApplicable {
		t.Fatalf("result=%+v error=%v", got, err)
	}
}

// TestAIContextStaleSuccessGatesRefuseWrites checks every rebuild boundary rejects an AI source behind a stale success gate.
// Inputs: synthetic success receipt rebound to AI provenance. Outputs: assertions.
// Effects: none; write methods intentionally have no implementation. Use for confirm and both commits.
func TestAIContextStaleSuccessGatesRefuseWrites(t *testing.T) {
	for _, stage := range []string{"confirm", "messages", "threads"} {
		t.Run(stage, func(t *testing.T) {
			s, r := aiBoundaryFixture()
			s.input.Source.DeclaredFormat = "chatgpt_official_json"
			a := FirstPartyContextActivities{Store: s}
			var err error
			switch stage {
			case "confirm":
				_, err = a.ConfirmFirstPartyContext(context.Background(), r)
			case "messages":
				_, err = a.CommitFirstPartyMessages(context.Background(), r)
			case "threads":
				_, err = a.CommitFirstPartyContextThreads(context.Background(), r)
			}
			if err == nil || !strings.Contains(err.Error(), "AI chat") || len(s.specs) != 0 {
				t.Fatalf("error=%v receipts=%+v", err, s.specs)
			}
		})
	}
}

// TestAIContextSMSProposalStillApplicable proves ordinary verified SMS keeps its proposal path.
// Inputs: synthetic SMS baseline. Outputs: assertions. Effects: in-memory proposal only.
// Use as the regression counterpart of AI exclusion.
func TestAIContextSMSProposalStillApplicable(t *testing.T) {
	s, r := aiBoundaryFixture()
	got, err := (FirstPartyContextActivities{Store: s}).ProposeFirstPartyContext(context.Background(), r)
	if err != nil || got.Status != proffer.StatusSuccess || len(s.specs) != 1 || s.specs[0].PlanDigest == "" || s.specs[0].Messages != 1 {
		t.Fatalf("result=%+v error=%v receipts=%+v", got, err, s.specs)
	}
}
