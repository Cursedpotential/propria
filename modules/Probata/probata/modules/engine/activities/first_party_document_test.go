package activities

import (
	"context"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// Owner UX test 2026-10-16: a plain-text document passed 24 stages and then failed
// participant resolution for want of messaging owner/perspective references. A
// generation with no message records has no participants to resolve.
func TestDocumentWithoutMessagesSkipsParticipantResolution(t *testing.T) {
	s, req := aiBoundaryFixture()
	s.input.Messages = nil
	s.input.StatedIdentifiers = []string{"self"}
	s.input.Source.DeclaredFormat = "delimited_text"
	delete(req.Refs, "owner_person")
	delete(req.Refs, "perspective_person")
	result, err := (FirstPartyContextActivities{Store: s}).ResolveContextParticipants(context.Background(), req)
	if err != nil || result.Status != proffer.StatusNotApplicable || result.ReceiptRef == "" || result.AIChatSource {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result.Reason != "the normalized generation holds no message records" {
		t.Fatalf("reason=%q", result.Reason)
	}
	if s.identityCalls != 0 || len(s.participantSpecs) != 1 || s.participantSpecs[0].NotApplicable == "" {
		t.Fatalf("a document reached identity resolution: calls=%d specs=%+v", s.identityCalls, s.participantSpecs)
	}
}

func TestMessagingStillRequiresExplicitOwnerAndPerspective(t *testing.T) {
	s, req := aiBoundaryFixture()
	s.input.StatedIdentifiers = []string{"self", "+18105550100"}
	delete(req.Refs, "owner_person")
	delete(req.Refs, "perspective_person")
	if _, err := (FirstPartyContextActivities{Store: s}).ResolveContextParticipants(context.Background(), req); err == nil {
		t.Fatal("a messaging import without owner/perspective was accepted")
	}
}
