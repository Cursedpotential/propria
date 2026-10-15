// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package activities

import (
	"context"
	"errors"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
)

// ContextSourceRegistrationRequest is the workflow-owned source identity contract.
// Inputs/outputs: bounded native coordinates. Effects: none; choose for the registration Activity.
type ContextSourceRegistrationRequest = proffer.ContextSourceRegistrationRequest

// ContextSourceRegistrationResult is the workflow-owned compact registration receipt.
// Inputs/outputs: version and receipt refs. Effects: none; choose for Activity return values.
type ContextSourceRegistrationResult = proffer.ContextSourceRegistrationResult

// ContextSourceRegistrationStore persists the identity, receipt and native locator metadata atomically.
// Inputs: one bounded request. Outputs: durable compact references. Effects: database writes only.
// Choose this store before parsing; the lifecycle RetainOriginal store is a separate custody boundary.
type ContextSourceRegistrationStore interface {
	RegisterContextSource(context.Context, ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error)
}

// ContextSourceRegistrationActivities exposes the atomic source-registration Activity.
// Inputs: a registration store. Outputs: one Temporal-callable method. Effects: delegates one database transaction.
// Choose for context sources, including native AI exports, before any parser runs.
type ContextSourceRegistrationActivities struct {
	Store ContextSourceRegistrationStore
}

// RegisterContextSourceActivity records one source identity without asserting custody or owner knowledge time.
// Inputs: actual request/workflow and native source coordinates. Outputs: source-version and receipt references.
// Effects: delegates one idempotent registration write. Choose before parsing, not for retention or approval.
func (a ContextSourceRegistrationActivities) RegisterContextSourceActivity(ctx context.Context, req ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
	if a.Store == nil {
		return ContextSourceRegistrationResult{}, errors.New("context source registration store is required")
	}
	if err := ctx.Err(); err != nil {
		return ContextSourceRegistrationResult{}, err
	}
	if req.WorkflowID == "" && activity.IsActivity(ctx) {
		req.WorkflowID = activity.GetInfo(ctx).WorkflowExecution.ID
	}
	if strings.TrimSpace(req.SourceKind) == "" {
		req.SourceKind = "unknown"
	}
	if strings.TrimSpace(req.RequestID) == "" || strings.TrimSpace(req.WorkflowID) == "" ||
		strings.TrimSpace(req.SourceRef) == "" {
		return ContextSourceRegistrationResult{}, errors.New("context source registration requires request, actual workflow and source pointer")
	}
	return a.Store.RegisterContextSource(ctx, req)
}
