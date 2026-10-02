// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (participant resolution stage; owner-participation split)
//
// The first-party context import (D04) as four Activities, on the
// extract -> confirm -> commit pattern of entity_extraction.go and the store
// split of normalized_pipeline.go (owner rulings 2026-09-26: the Go engine
// orchestrates everything, everything goes through Temporal; D-161):
//
//	propose_first_party_context_activity   EXTRACT, before the preview: plan the
//	    generation's messages into conversations, check the identity, record the
//	    plan digest. Writes no working.* row.
//	confirm_first_party_context_activity   CONFIRM, after the owner's decision:
//	    rebuild the plan and prove it is the one proposed.
//	commit_first_party_messages_activity   COMMIT the spine: working.normalized_record,
//	    message_projection_route, message (id = normalized record id) and
//	    message_participant.
//	commit_first_party_context_threads_activity  COMMIT the threads:
//	    working.first_party_context_thread and its version / membership / source rows.
//
// Every Activity does compute and validation; the Store owns every SQL
// transaction and idempotency coordinate. Exactly one Activity owns each
// write. Only references cross the workflow boundary: the plan is rebuilt
// from PostgreSQL by each Activity that needs it and bound to the recorded
// digest, never carried in Temporal history.
//
// Identity is never invented. The owner and perspective person arrive as
// explicit run inputs (refs "owner_person" / "perspective_person"); a
// generation with messages and no identity fails loudly and permanently.
package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/firstparty"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// Receipt kinds recorded in context.activity_receipt.result_ref.
const (
	FirstPartyProposalKind     = "first_party_context_proposal"
	FirstPartyConfirmationKind = "first_party_context_confirmation"
	FirstPartyMessagesKind     = "first_party_messages"
	FirstPartyThreadsKind      = "first_party_context_threads"
	// ParticipantResolutionKind is the resolve stage's receipt: result_ref is
	// {"ref_kind":"participant_resolution","ref_id":<uuid>,"resolution":<Resolution>},
	// the contract publish_context_search_activity reads too.
	ParticipantResolutionKind = "participant_resolution"
)

// ParticipantResolutionSpec is one resolve-stage receipt.
type ParticipantResolutionSpec struct {
	RequestID               string
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	Resolution              disclosure.Resolution
	NotApplicable           string
	Attempt                 int32
}

// FirstPartyContextInput is what the Store resolves for one generation.
// Messages is empty when the generation holds no message record; Source is
// then partially filled and the stage is not applicable.
type FirstPartyContextInput struct {
	Source   firstparty.Source
	Messages []firstparty.SourceMessage
	// StatedIdentifiers is every distinct identifier any record of the
	// generation states (messages and calls), except the device marker "self";
	// the participant resolution covers exactly these.
	StatedIdentifiers []string
	// PlatformResolved is false when the generation holds messages but traces
	// to no registered derivation; Reason then says why.
	PlatformResolved bool
	Reason           string
}

// FirstPartyReceiptSpec is one proposal or confirmation receipt.
type FirstPartyReceiptSpec struct {
	Stage                   stagegraph.StageID
	Kind                    string
	RequestID               string
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	// ParentRef is the receipt this one follows (the proposal, for a
	// confirmation). Empty for a proposal.
	ParentRef proffer.Ref
	// ResolutionRef is the participant resolution the plan was built on.
	ResolutionRef proffer.Ref
	Identity      contextthread.Identity
	PlanDigest    string
	Messages      int
	Threads       int
	NotApplicable string
	Attempt       int32
}

// FirstPartyReceipt is a recorded proposal or confirmation, read back.
type FirstPartyReceipt struct {
	Kind                    string
	ReceiptRef              proffer.Ref
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	ParentRef               proffer.Ref
	ResolutionRef           proffer.Ref
	Identity                contextthread.Identity
	PlanDigest              string
}

// FirstPartyCommitSpec is one commit Activity's write.
type FirstPartyCommitSpec struct {
	RequestID        string
	SourceVersionRef proffer.Ref
	// GateRef is the receipt that authorizes this write: the confirmation for
	// the spine, the spine commit for the threads.
	GateRef proffer.Ref
	Plan    firstparty.Plan
	Attempt int32
}

// FirstPartyContextStore is the PostgreSQL boundary of the four Activities.
type FirstPartyContextStore interface {
	// LoadFirstPartyContext proves the verification receipt certifies this
	// generation of this request's source version, then reads its message
	// records in ordinal order and resolves its platform from the registered
	// derivation that produced the source.
	LoadFirstPartyContext(ctx context.Context, req proffer.StageRequest, generationRef, verificationRef proffer.Ref) (FirstPartyContextInput, error)
	// ResolveFirstPartyIdentity checks the explicit owner and perspective
	// against registry.person and the run's admitted matter and court case.
	ResolveFirstPartyIdentity(ctx context.Context, req proffer.StageRequest, ownerPersonID, perspectivePersonID string) (contextthread.Identity, error)
	// ResolveParticipants resolves every stated identifier against the
	// registry's confirmed identifiers, marking the owner's. It fails closed
	// when the owner has no confirmed identifier.
	ResolveParticipants(ctx context.Context, identity contextthread.Identity, raws []string) (disclosure.Resolution, error)
	PersistParticipantResolution(ctx context.Context, spec ParticipantResolutionSpec) (resultRef, receiptRef proffer.Ref, err error)
	LoadParticipantResolution(ctx context.Context, ref proffer.Ref) (disclosure.Resolution, error)
	PersistFirstPartyReceipt(ctx context.Context, spec FirstPartyReceiptSpec) (resultRef, receiptRef proffer.Ref, err error)
	LoadFirstPartyReceipt(ctx context.Context, kind string, ref proffer.Ref) (FirstPartyReceipt, error)
	CommitFirstPartyMessages(ctx context.Context, spec FirstPartyCommitSpec) (resultRef, receiptRef proffer.Ref, err error)
	CommitFirstPartyContextThreads(ctx context.Context, spec FirstPartyCommitSpec) (resultRef, receiptRef proffer.Ref, err error)
}

// FirstPartyContextActivities implements the four Activities.
type FirstPartyContextActivities struct {
	Store   FirstPartyContextStore
	Attempt Attempt
}

// NewFirstPartyContextActivities binds the Activities to Temporal attempts.
func NewFirstPartyContextActivities(store FirstPartyContextStore) FirstPartyContextActivities {
	return FirstPartyContextActivities{Store: store, Attempt: func(ctx context.Context) int32 {
		return activity.GetInfo(ctx).Attempt
	}}
}

// RegisterFirstPartyContextActivities installs the four Activities under their
// exact stage-graph identities.
func RegisterFirstPartyContextActivities(registrar ActivityRegistrar, acts FirstPartyContextActivities) {
	registrar.RegisterActivityWithOptions(acts.ResolveContextParticipants, activity.RegisterOptions{Name: string(stagegraph.ResolveContextParticipants)})
	registrar.RegisterActivityWithOptions(acts.ProposeFirstPartyContext, activity.RegisterOptions{Name: string(stagegraph.ProposeFirstPartyContext)})
	registrar.RegisterActivityWithOptions(acts.ConfirmFirstPartyContext, activity.RegisterOptions{Name: string(stagegraph.ConfirmFirstPartyContext)})
	registrar.RegisterActivityWithOptions(acts.CommitFirstPartyMessages, activity.RegisterOptions{Name: string(stagegraph.CommitFirstPartyMessages)})
	registrar.RegisterActivityWithOptions(acts.CommitFirstPartyContextThreads, activity.RegisterOptions{Name: string(stagegraph.CommitFirstPartyContextThreads)})
}

func (a FirstPartyContextActivities) attempt(ctx context.Context) int32 {
	if a.Attempt == nil {
		return 1
	}
	if attempt := a.Attempt(ctx); attempt >= 1 {
		return attempt
	}
	return 1
}

func (a FirstPartyContextActivities) ready(req proffer.StageRequest, stage stagegraph.StageID) error {
	if a.Store == nil {
		return fmt.Errorf("%s: store is required", stage)
	}
	if strings.TrimSpace(req.RequestID) == "" || req.SourceVersionRef == "" {
		return permanent(fmt.Errorf("%s requires request and source version references", stage))
	}
	return nil
}

// ResolveContextParticipants resolves every identifier the generation states,
// once, and records the resolution both the Weaviate-first stage and the
// first-party context stages read.
func (a FirstPartyContextActivities) ResolveContextParticipants(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.resolve(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a FirstPartyContextActivities) resolve(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	stage := stagegraph.ResolveContextParticipants
	if err := a.ready(req, stage); err != nil {
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
	input, err := a.Store.LoadFirstPartyContext(ctx, req, generationRef, verificationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if len(input.StatedIdentifiers) == 0 {
		const reason = "the normalized generation states no participant identifier"
		_, receiptRef, err := a.Store.PersistParticipantResolution(ctx, ParticipantResolutionSpec{
			RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
			NotApplicable: reason, Attempt: a.attempt(ctx),
		})
		if err != nil {
			return proffer.StageResult{}, err
		}
		return proffer.StageResult{Stage: stage, Status: proffer.StatusNotApplicable, ReceiptRef: receiptRef, Reason: reason}, nil
	}
	identity, err := a.identity(ctx, req)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resolution, err := a.Store.ResolveParticipants(ctx, identity, input.StatedIdentifiers)
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	resultRef, receiptRef, err := a.Store.PersistParticipantResolution(ctx, ParticipantResolutionSpec{
		RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
		Resolution: resolution, Attempt: a.attempt(ctx),
	})
	if err != nil {
		return proffer.StageResult{}, err
	}
	return success(stage, resultRef, receiptRef), nil
}

// ProposeFirstPartyContext is the EXTRACT step.
func (a FirstPartyContextActivities) ProposeFirstPartyContext(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.propose(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a FirstPartyContextActivities) propose(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	stage := stagegraph.ProposeFirstPartyContext
	if err := a.ready(req, stage); err != nil {
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
	input, err := a.Store.LoadFirstPartyContext(ctx, req, generationRef, verificationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if proffer.IsAIChatFormat(input.Source.DeclaredFormat) {
		return proffer.StageResult{}, permanent(errors.New(proffer.AIChatRefusalMessage(input.Source.DeclaredFormat)))
	}
	if len(input.Messages) == 0 {
		// Nothing to import is a recorded outcome, not a silent skip.
		_, receiptRef, err := a.Store.PersistFirstPartyReceipt(ctx, FirstPartyReceiptSpec{
			Stage: stage, Kind: FirstPartyProposalKind, RequestID: req.RequestID,
			SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
			NotApplicable: "the normalized generation holds no message records", Attempt: a.attempt(ctx),
		})
		if err != nil {
			return proffer.StageResult{}, err
		}
		return proffer.StageResult{
			Stage: stage, Status: proffer.StatusNotApplicable, ReceiptRef: receiptRef,
			Reason: "the normalized generation holds no message records",
		}, nil
	}
	if !input.PlatformResolved {
		return proffer.StageResult{}, permanent(fmt.Errorf("first-party context import refused: %s", input.Reason))
	}
	identity, err := a.identity(ctx, req)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resolutionRef, err := requiredRef(req, "participant_resolution")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	resolution, err := a.Store.LoadParticipantResolution(ctx, resolutionRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	plan, err := firstparty.Build(identity, input.Source, input.Messages, resolution)
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	if err := validatePlanThreads(plan); err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	resultRef, receiptRef, err := a.Store.PersistFirstPartyReceipt(ctx, FirstPartyReceiptSpec{
		Stage: stage, Kind: FirstPartyProposalKind, RequestID: req.RequestID,
		SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: generationRef,
		ResolutionRef: resolutionRef,
		Identity:      identity, PlanDigest: plan.Digest, Messages: plan.MessageCount,
		Threads: len(plan.Conversations), Attempt: a.attempt(ctx),
	})
	if err != nil {
		return proffer.StageResult{}, err
	}
	return success(stage, resultRef, receiptRef), nil
}

// identity reads the explicit person references and has the Store check them.
func (a FirstPartyContextActivities) identity(ctx context.Context, req proffer.StageRequest) (contextthread.Identity, error) {
	owner := strings.TrimSpace(string(req.Refs["owner_person"]))
	perspective := strings.TrimSpace(string(req.Refs["perspective_person"]))
	if owner == "" || perspective == "" {
		return contextthread.Identity{}, permanent(errors.New(
			"first-party context import requires explicit owner_person_id and perspective_person_id on the run; they are never derived"))
	}
	identity, err := a.Store.ResolveFirstPartyIdentity(ctx, req, owner, perspective)
	if err != nil {
		return contextthread.Identity{}, permanent(err)
	}
	return identity, nil
}

// validatePlanThreads proves every conversation's version-1 thread commit is
// acceptable to the thread contract before anything is proposed, so a plan
// that could never commit is refused at proposal, not after approval.
func validatePlanThreads(plan firstparty.Plan) error {
	for _, conversation := range plan.FirstParty() {
		if err := plan.NewThreadVersion("", conversation).Validate(); err != nil {
			return fmt.Errorf("conversation %s cannot form a thread version: %w", conversation.Key, err)
		}
	}
	return nil
}

// rebuild re-derives the plan for a recorded receipt's generation and binds
// it to the recorded digest.
func (a FirstPartyContextActivities) rebuild(ctx context.Context, req proffer.StageRequest, receipt FirstPartyReceipt, verificationRef proffer.Ref) (firstparty.Plan, error) {
	if receipt.SourceVersionRef != req.SourceVersionRef {
		return firstparty.Plan{}, permanent(errors.New("first-party context receipt belongs to a different source version"))
	}
	input, err := a.Store.LoadFirstPartyContext(ctx, req, receipt.NormalizedGenerationRef, verificationRef)
	if err != nil {
		return firstparty.Plan{}, err
	}
	if !input.PlatformResolved {
		return firstparty.Plan{}, permanent(fmt.Errorf("first-party context import refused: %s", input.Reason))
	}
	// Backstop: AI chats are search-only and never write working.message.
	if proffer.IsAIChatFormat(input.Source.DeclaredFormat) {
		return firstparty.Plan{}, permanent(errors.New(proffer.AIChatRefusalMessage(input.Source.DeclaredFormat)))
	}
	resolution, err := a.Store.LoadParticipantResolution(ctx, receipt.ResolutionRef)
	if err != nil {
		return firstparty.Plan{}, err
	}
	plan, err := firstparty.Build(receipt.Identity, input.Source, input.Messages, resolution)
	if err != nil {
		return firstparty.Plan{}, permanent(err)
	}
	if err := plan.Rebuilt(receipt.PlanDigest); err != nil {
		return firstparty.Plan{}, permanent(err)
	}
	return plan, nil
}

// ConfirmFirstPartyContext is the CONFIRM step, run after the owner's
// preview decision: the plan rebuilt now must be the plan proposed.
func (a FirstPartyContextActivities) ConfirmFirstPartyContext(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.confirm(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a FirstPartyContextActivities) confirm(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	stage := stagegraph.ConfirmFirstPartyContext
	if err := a.ready(req, stage); err != nil {
		return proffer.StageResult{}, err
	}
	proposalRef, err := requiredRef(req, "context_proposal")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	verificationRef, err := requiredRef(req, "normalized_verification")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	proposal, err := a.Store.LoadFirstPartyReceipt(ctx, FirstPartyProposalKind, proposalRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	// The run's identity must still be the identity proposed.
	identity, err := a.identity(ctx, req)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if identity != proposal.Identity {
		return proffer.StageResult{}, permanent(errors.New("the run's owner/perspective identity differs from the proposed one"))
	}
	plan, err := a.rebuild(ctx, req, proposal, verificationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resultRef, receiptRef, err := a.Store.PersistFirstPartyReceipt(ctx, FirstPartyReceiptSpec{
		Stage: stage, Kind: FirstPartyConfirmationKind, RequestID: req.RequestID,
		SourceVersionRef: req.SourceVersionRef, NormalizedGenerationRef: proposal.NormalizedGenerationRef,
		ParentRef: proposalRef, ResolutionRef: proposal.ResolutionRef, Identity: identity, PlanDigest: plan.Digest,
		Messages: plan.MessageCount, Threads: len(plan.Conversations), Attempt: a.attempt(ctx),
	})
	if err != nil {
		return proffer.StageResult{}, err
	}
	return success(stage, resultRef, receiptRef), nil
}

// CommitFirstPartyMessages is the spine COMMIT.
func (a FirstPartyContextActivities) CommitFirstPartyMessages(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.commit(ctx, req, stagegraph.CommitFirstPartyMessages, "context_confirmation", a.Store.CommitFirstPartyMessages)
	return result, stopRetryingPermanent(err)
}

// CommitFirstPartyContextThreads is the thread COMMIT. It requires the spine
// commit's receipt: thread membership references working.message rows.
func (a FirstPartyContextActivities) CommitFirstPartyContextThreads(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	result, err := a.commit(ctx, req, stagegraph.CommitFirstPartyContextThreads, "context_messages", a.Store.CommitFirstPartyContextThreads)
	return result, stopRetryingPermanent(err)
}

func (a FirstPartyContextActivities) commit(
	ctx context.Context, req proffer.StageRequest, stage stagegraph.StageID, gateName string,
	write func(context.Context, FirstPartyCommitSpec) (proffer.Ref, proffer.Ref, error),
) (proffer.StageResult, error) {
	if err := a.ready(req, stage); err != nil {
		return proffer.StageResult{}, err
	}
	gateRef, err := requiredRef(req, gateName)
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	confirmationRef, err := requiredRef(req, "context_confirmation")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	verificationRef, err := requiredRef(req, "normalized_verification")
	if err != nil {
		return proffer.StageResult{}, permanent(err)
	}
	confirmation, err := a.Store.LoadFirstPartyReceipt(ctx, FirstPartyConfirmationKind, confirmationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	plan, err := a.rebuild(ctx, req, confirmation, verificationRef)
	if err != nil {
		return proffer.StageResult{}, err
	}
	resultRef, receiptRef, err := write(ctx, FirstPartyCommitSpec{
		RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, GateRef: gateRef,
		Plan: plan, Attempt: a.attempt(ctx),
	})
	if err != nil {
		return proffer.StageResult{}, err
	}
	return success(stage, resultRef, receiptRef), nil
}
