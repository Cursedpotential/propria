// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// The step Activities of dedupe.MessageDedupeWorkflow (owner 2026-10-02 20:02:
// "Make sure all these things get run as Temporal activities and are
// traceable"). Each one runs its step through the PostgreSQL store, heartbeats
// while it works through the plan in batches, and returns the step's receipt.
// A refusal (a plan that differs from the expected count, a copy with rows the
// removal does not know, an assertion that fails) is not retried.
package activities

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/dedupe"
)

// MessageDedupeStore is the PostgreSQL boundary of the removal.
type MessageDedupeStore interface {
	PlanDedupe(ctx context.Context, request dedupe.StepRequest, attempt int32) (dedupe.Receipt, error)
	RunDedupeStep(ctx context.Context, request dedupe.StepRequest, attempt int32, progress func(done int64)) (dedupe.Receipt, error)
}

// MessageDedupeActivities implements every step Activity.
type MessageDedupeActivities struct {
	Store     MessageDedupeStore
	Attempt   Attempt
	Heartbeat func(ctx context.Context, details ...interface{})
}

// NewMessageDedupeActivities binds the Activities to Temporal attempts and heartbeats.
func NewMessageDedupeActivities(store MessageDedupeStore) MessageDedupeActivities {
	return MessageDedupeActivities{
		Store:     store,
		Attempt:   func(ctx context.Context) int32 { return activity.GetInfo(ctx).Attempt },
		Heartbeat: activity.RecordHeartbeat,
	}
}

// RegisterMessageDedupeActivities installs one Activity per step.
func RegisterMessageDedupeActivities(registrar ActivityRegistrar, acts MessageDedupeActivities) {
	registrar.RegisterActivityWithOptions(acts.Plan, activity.RegisterOptions{Name: dedupe.ActivityName(dedupe.StepPlan)})
	for _, step := range dedupe.Steps {
		registrar.RegisterActivityWithOptions(acts.forStep(step), activity.RegisterOptions{Name: dedupe.ActivityName(step)})
	}
}

// Plan is message_dedupe_plan_activity.
func (a MessageDedupeActivities) Plan(ctx context.Context, request dedupe.StepRequest) (dedupe.Receipt, error) {
	if a.Store == nil {
		return dedupe.Receipt{}, errors.New("message dedupe: store is required")
	}
	if request.Step != dedupe.StepPlan {
		return dedupe.Receipt{}, refused(fmt.Errorf("the plan Activity was asked to run step %q", request.Step))
	}
	receipt, err := a.Store.PlanDedupe(ctx, request, a.attempt(ctx))
	return receipt, refused(err)
}

func (a MessageDedupeActivities) forStep(step dedupe.Step) func(context.Context, dedupe.StepRequest) (dedupe.Receipt, error) {
	return func(ctx context.Context, request dedupe.StepRequest) (dedupe.Receipt, error) {
		if a.Store == nil {
			return dedupe.Receipt{}, errors.New("message dedupe: store is required")
		}
		if request.Step != step {
			return dedupe.Receipt{}, refused(fmt.Errorf("the %s Activity was asked to run step %q", step, request.Step))
		}
		receipt, err := a.Store.RunDedupeStep(ctx, request, a.attempt(ctx), func(done int64) {
			if a.Heartbeat != nil {
				a.Heartbeat(ctx, map[string]any{"step": string(step), "done": done})
			}
		})
		return receipt, refused(err)
	}
}

func (a MessageDedupeActivities) attempt(ctx context.Context) int32 {
	if a.Attempt == nil {
		return 1
	}
	return a.Attempt(ctx)
}

// refused turns a dedupe.Refusal into Temporal's non-retryable failure; every
// other error keeps the step's bounded retry.
func refused(err error) error {
	var refusal dedupe.Refusal
	if err != nil && errors.As(err, &refusal) {
		return temporal.NewNonRetryableApplicationError(err.Error(), dedupe.RefusalType, err)
	}
	return err
}
