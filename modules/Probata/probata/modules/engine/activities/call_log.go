// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// commit_call_log_activity: the owner's committed call records into
// working.call_log (owner 2026-10-02, "calls follow the same path as
// messages"). It runs after the same preview decision as the message commit
// (the owner's approval, or the clean_checks automatic approval), reads the
// same recorded participant resolution, and takes the device's owner from the
// run's perspective person. It commits only; it does not parse, hash or decide.
package activities

import (
	"context"
	"fmt"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// CallLogCommitSpec is one call-log commit.
type CallLogCommitSpec struct {
	RequestID               string
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	// PreviewHandle names the preview whose approved decision is the gate.
	PreviewHandle proffer.Ref
	Resolution    disclosure.Resolution
	// PerspectivePersonID is whose phone the call log came from.
	PerspectivePersonID string
	Attempt             int32
}

// CallLogStore is the PostgreSQL boundary of commit_call_log_activity.
type CallLogStore interface {
	// CommitCallLog proves the gate (an approved decision on this run's
	// preview of this generation), then writes every call record of the
	// generation, idempotently, and returns its receipt. calls is 0 when the
	// generation holds no call record.
	CommitCallLog(ctx context.Context, spec CallLogCommitSpec) (resultRef, receiptRef proffer.Ref, calls int, err error)
	LoadParticipantResolution(ctx context.Context, ref proffer.Ref) (disclosure.Resolution, error)
}

// CallLogActivities implements commit_call_log_activity.
type CallLogActivities struct {
	Store   CallLogStore
	Attempt Attempt
}

// NewCallLogActivities binds the Activity to Temporal attempts.
func NewCallLogActivities(store CallLogStore) CallLogActivities {
	return CallLogActivities{Store: store, Attempt: func(ctx context.Context) int32 {
		return activity.GetInfo(ctx).Attempt
	}}
}

// RegisterCallLogActivities installs commit_call_log_activity.
func RegisterCallLogActivities(registrar ActivityRegistrar, acts CallLogActivities) {
	registrar.RegisterActivityWithOptions(acts.CommitCallLog, activity.RegisterOptions{Name: string(stagegraph.CommitCallLog)})
}

// CommitCallLog is commit_call_log_activity.
func (a CallLogActivities) CommitCallLog(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.commit(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a CallLogActivities) commit(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	stage := stagegraph.CommitCallLog
	if a.Store == nil {
		return proffer.StageResult{}, fmt.Errorf("%s: store is required", stage)
	}
	if strings.TrimSpace(req.RequestID) == "" || req.SourceVersionRef == "" {
		return proffer.StageResult{}, permanent(fmt.Errorf("%s requires request and source version references", stage))
	}
	generationRef, err := requiredRef(req, "normalized_generation")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	handle, err := requiredRef(req, "preview_handle")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	spec := CallLogCommitSpec{
		RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
		PreviewHandle: handle, PerspectivePersonID: strings.TrimSpace(string(req.Refs["perspective_person"])),
		Attempt: 1,
	}
	if a.Attempt != nil {
		if attempt := a.Attempt(ctx); attempt >= 1 {
			spec.Attempt = attempt
		}
	}
	if ref := req.Refs["participant_resolution"]; ref != "" {
		resolution, err := a.Store.LoadParticipantResolution(ctx, ref)
		if err != nil {
			return proffer.StageResult{}, err
		}
		spec.Resolution = resolution
		if spec.PerspectivePersonID == "" {
			spec.PerspectivePersonID = strings.TrimSpace(resolution.PerspectivePersonID)
		}
	}
	resultRef, receiptRef, calls, err := a.Store.CommitCallLog(ctx, spec)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if calls == 0 {
		return proffer.StageResult{Stage: stage, Status: proffer.StatusNotApplicable, ReceiptRef: receiptRef,
			Reason: "the normalized generation holds no call records"}, nil
	}
	return success(stage, resultRef, receiptRef), nil
}

// CallType maps one normalized call record's content to working.call_log's
// call_type, direction and is_blocked (SMS Backup & Restore call types). An
// unknown combination is refused rather than guessed.
func CallType(direction, disposition string, missed bool) (callType, callDirection string, blocked bool, err error) {
	direction, disposition = strings.ToLower(strings.TrimSpace(direction)), strings.ToLower(strings.TrimSpace(disposition))
	switch {
	case direction == "outgoing" && disposition == "completed" && !missed:
		return "outgoing", "outbound", false, nil
	case direction == "incoming" && (disposition == "missed" || missed):
		return "missed", "inbound", false, nil
	case direction == "incoming" && disposition == "completed":
		return "incoming", "inbound", false, nil
	case direction == "incoming" && disposition == "rejected":
		return "rejected", "inbound", false, nil
	case direction == "incoming" && disposition == "refused":
		// SMS Backup & Restore type 6: the number is on the phone's refused list.
		return "blocked_incoming", "inbound", true, nil
	case direction == "incoming" && disposition == "voicemail":
		return "voicemail", "inbound", false, nil
	}
	return "", "", false, permanent(fmt.Errorf("call record has direction %q and disposition %q, which no call_log type covers", direction, disposition))
}
