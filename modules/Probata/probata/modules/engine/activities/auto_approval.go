// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// record_auto_approval_activity: the owner's "auto-approve clean runs" policy
// (2026-10-02). The workflow schedules it only for a run started with the
// policy switched on, only after the preview is published, and only when every
// check in proffer.AutoApprovalChecks settled success. It writes one row: the
// same context.proffer_preview_decision record (and its decision_recorded
// event) a human approval in Review writes, with actor "auto:clean-checks" and
// the passed checks, by receipt, as the reason. It decides nothing itself; it
// refuses a request whose checks are incomplete or not all success.
package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// PreviewDecisionRecorder is the durable preview decision record, the one
// the Review decision endpoint writes (postgres.ProfferPreviewStore).
type PreviewDecisionRecorder interface {
	RecordDecision(ctx context.Context, handle string, approved bool, reason, actor string, selection, options proffer.Ref) error
}

// AutoApprovalActivity records an automatic approval.
type AutoApprovalActivity struct{ Store PreviewDecisionRecorder }

// RegisterAutoApprovalActivity installs record_auto_approval_activity.
func RegisterAutoApprovalActivity(registrar ActivityRegistrar, auto AutoApprovalActivity) {
	registrar.RegisterActivityWithOptions(auto.Record, activity.RegisterOptions{Name: string(stagegraph.RecordAutoApproval)})
}

// AutoApprovalReason is the decision reason: every passed check, in the
// policy's order, by receipt. It is deterministic, so a retried Activity
// produces the same decision key and the insert stays idempotent.
func AutoApprovalReason(checks []proffer.AutoApprovalCheck) string {
	parts := make([]string, 0, len(checks))
	for _, check := range checks {
		parts = append(parts, fmt.Sprintf("%s=%s(%s)", check.Stage, check.Status, check.ReceiptRef))
	}
	return "approved automatically, every check passed: " + strings.Join(parts, "; ")
}

// Record writes the automatic approval.
func (a AutoApprovalActivity) Record(ctx context.Context, request proffer.AutoApprovalRequest) (proffer.StageResult, error) {
	if a.Store == nil {
		return proffer.StageResult{}, errors.New("automatic approval requires the durable preview decision store")
	}
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(string(request.PreviewHandle)) == "" ||
		strings.TrimSpace(string(request.SelectionRef)) == "" || strings.TrimSpace(string(request.ParserOptionsRef)) == "" {
		return proffer.StageResult{}, stopRetryingPermanent(permanent(errors.New("automatic approval requires request, preview handle, selection and parser options references")))
	}
	if len(request.Checks) != len(proffer.AutoApprovalChecks) {
		return proffer.StageResult{}, stopRetryingPermanent(permanent(fmt.Errorf("automatic approval requires all %d checks, got %d", len(proffer.AutoApprovalChecks), len(request.Checks))))
	}
	for index, want := range proffer.AutoApprovalChecks {
		check := request.Checks[index]
		if check.Stage != want || check.Status != proffer.StatusSuccess || strings.TrimSpace(string(check.ReceiptRef)) == "" {
			return proffer.StageResult{}, stopRetryingPermanent(permanent(fmt.Errorf("automatic approval refused: check %s is %q, not a receipted success", want, check.Status)))
		}
	}
	reason := AutoApprovalReason(request.Checks)
	if err := a.Store.RecordDecision(ctx, string(request.PreviewHandle), true, reason, stagegraph.AutoApprovalActor,
		request.SelectionRef, request.ParserOptionsRef); err != nil {
		return proffer.StageResult{}, fmt.Errorf("record automatic approval: %w", err)
	}
	receipt := proffer.Ref("proffer_preview_decision:" + string(request.PreviewHandle) + ":" + stagegraph.AutoApprovalActor)
	return proffer.StageResult{Stage: stagegraph.RecordAutoApproval, Status: proffer.StatusSuccess, Ref: request.PreviewHandle, ReceiptRef: receipt}, nil
}
