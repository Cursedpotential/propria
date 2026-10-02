// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// match_message_occurrences_activity (owner 2026-10-02: "both Facebook
// exports, deduped"; decision C, messages present in more than one source match
// up). Before the Weaviate-first stage, it plans the generation exactly as the
// first-party proposal does and looks up which of its messages another source
// version already committed (same platform, parties, sender, sent time to the
// second and body hash). It records that list and writes nothing else: the
// search stage skips those messages, and the commit, which repeats the look-up
// under a lock, records them as further occurrences instead of second rows.
package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// MessageMatch is one message an earlier source already committed.
type MessageMatch struct {
	RecordID        string `json:"record_id"`
	PrimaryRecordID string `json:"primary_record_id"`
	MatchKey        string `json:"match_key"`
	CrossDevice     bool   `json:"cross_device"`
}

// MessageMatchSpec is one look-up's receipt.
type MessageMatchSpec struct {
	RequestID               string
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	Messages                int
	Matches                 []MessageMatch
	Attempt                 int32
}

// MessageMatchStore is the PostgreSQL boundary of the look-up.
type MessageMatchStore interface {
	FindMessageMatches(ctx context.Context, plan firstparty.Plan) ([]MessageMatch, error)
	PersistMessageMatches(ctx context.Context, spec MessageMatchSpec) (resultRef, receiptRef proffer.Ref, err error)
	LoadMatchedRecords(ctx context.Context, ref proffer.Ref) (map[string]bool, error)
}

// MessageMatchActivities implements match_message_occurrences_activity. Context
// is the first-party context store, whose loaders plan the generation.
type MessageMatchActivities struct {
	Context FirstPartyContextStore
	Matches MessageMatchStore
	Attempt Attempt
}

// NewMessageMatchActivities binds the Activity to Temporal attempts.
func NewMessageMatchActivities(contextStore FirstPartyContextStore, matches MessageMatchStore) MessageMatchActivities {
	return MessageMatchActivities{Context: contextStore, Matches: matches, Attempt: func(ctx context.Context) int32 {
		return activity.GetInfo(ctx).Attempt
	}}
}

// RegisterMessageMatchActivities installs match_message_occurrences_activity.
func RegisterMessageMatchActivities(registrar ActivityRegistrar, acts MessageMatchActivities) {
	registrar.RegisterActivityWithOptions(acts.MatchMessageOccurrences, activity.RegisterOptions{Name: string(stagegraph.MatchMessageOccurrences)})
}

// MatchMessageOccurrences is match_message_occurrences_activity.
func (a MessageMatchActivities) MatchMessageOccurrences(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.match(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a MessageMatchActivities) match(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	stage := stagegraph.MatchMessageOccurrences
	if a.Context == nil || a.Matches == nil {
		return proffer.StageResult{}, fmt.Errorf("%s: stores are required", stage)
	}
	firstParty := FirstPartyContextActivities{Store: a.Context, Attempt: a.Attempt}
	if err := firstParty.ready(req, stage); err != nil {
		return proffer.StageResult{}, err
	}
	generationRef, err := requiredRef(req, "normalized_generation")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	verificationRef, err := requiredRef(req, "normalized_verification")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	input, err := a.Context.LoadFirstPartyContext(ctx, req, generationRef, verificationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	notApplicable := func(reason string) (proffer.StageResult, error) {
		_, receiptRef, err := a.Matches.PersistMessageMatches(ctx, MessageMatchSpec{
			RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
			Attempt: firstParty.attempt(ctx),
		})
		if err != nil {
			return proffer.StageResult{}, err
		}
		return proffer.StageResult{Stage: stage, Status: proffer.StatusNotApplicable, ReceiptRef: receiptRef, Reason: reason}, nil
	}
	if len(input.Messages) == 0 {
		return notApplicable("the normalized generation holds no message records")
	}
	if !input.PlatformResolved {
		// The proposal refuses such a source with the reason; nothing to match.
		return notApplicable(input.Reason)
	}
	resolutionRef := req.Refs["participant_resolution"]
	if resolutionRef == "" {
		return notApplicable("the run recorded no participant resolution")
	}
	identity, err := firstParty.identity(ctx, req)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resolution, err := a.Context.LoadParticipantResolution(ctx, resolutionRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	plan, err := firstparty.Build(identity, input.Source, input.Messages, resolution)
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	matches, err := a.Matches.FindMessageMatches(ctx, plan)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resultRef, receiptRef, err := a.Matches.PersistMessageMatches(ctx, MessageMatchSpec{
		RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
		Messages: plan.MessageCount, Matches: matches, Attempt: firstParty.attempt(ctx),
	})
	if err != nil {
		return proffer.StageResult{}, err
	}
	return success(stage, resultRef, receiptRef), nil
}
