// Byline: Codex · GPT-5 · 2026-10-05.
package profferworker

import (
	"context"
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// Each reviewed non-StageRequest family is exercised with its exact typed signature.
// Empty legacy mode, DEV and foreign scope must produce zero body calls.
func TestImportActivityGuardPreservesSignaturesAndDeniesLegacyBodies(t *testing.T) {
	live := proffer.StageRequest{OperatingMode: "LIVE", MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID}
	for _, payload := range []interface{}{
		live,
		proffer.HandlerRecoveryRequest{Request: live},
		proffer.PreviewPublicationRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		proffer.AutoApprovalRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		activities.ListBatchFolderRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		activities.BindImportOperationRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		activities.AICandidateStageInput{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		repairplan.StepRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
		repairplan.ReceiptRequest{OperatingMode: "LIVE", MatterID: live.MatterID, CourtCaseID: live.CourtCaseID},
	} {
		t.Run(reflect.TypeOf(payload).Name(), func(t *testing.T) {
			calls := 0
			sig := reflect.FuncOf([]reflect.Type{reflect.TypeOf((*context.Context)(nil)).Elem(), reflect.TypeOf(payload)}, []reflect.Type{reflect.TypeOf(proffer.StageResult{}), reflect.TypeOf((*error)(nil)).Elem()}, false)
			body := reflect.MakeFunc(sig, func([]reflect.Value) []reflect.Value {
				calls++
				return []reflect.Value{reflect.ValueOf(proffer.StageResult{ReceiptRef: "receipt"}), reflect.Zero(sig.Out(1))}
			})
			guarded := reflect.ValueOf(guardImportActivity(body.Interface(), "reviewed-test"))
			require.Equal(t, sig, guarded.Type())
			for _, mode := range []string{"", "DEV", "REAL", "unknown"} {
				denied := reflect.New(reflect.TypeOf(payload)).Elem()
				denied.Set(reflect.ValueOf(payload))
				target := denied
				if target.FieldByName("Request").IsValid() {
					target = target.FieldByName("Request")
				}
				target.FieldByName("OperatingMode").SetString(mode)
				out := guarded.Call([]reflect.Value{reflect.ValueOf(context.Background()), denied})
				require.False(t, out[1].IsNil())
				require.Zero(t, calls)
			}
			out := guarded.Call([]reflect.Value{reflect.ValueOf(context.Background()), reflect.ValueOf(payload)})
			require.True(t, out[1].IsNil())
			require.Equal(t, 1, calls)
			foreign := reflect.New(reflect.TypeOf(payload)).Elem()
			foreign.Set(reflect.ValueOf(payload))
			target := foreign
			if target.FieldByName("Request").IsValid() {
				target = target.FieldByName("Request")
			}
			target.FieldByName("CourtCaseID").SetString("unrelated")
			require.False(t, guarded.Call([]reflect.Value{reflect.ValueOf(context.Background()), foreign})[1].IsNil())
			require.Equal(t, 1, calls)
		})
	}
	require.Error(t, importActivityAdmission(struct{}{}))
}

type guardedRegistryProof struct{ functions map[string]interface{} }

func (*guardedRegistryProof) RegisterWorkflow(interface{})                                      {}
func (*guardedRegistryProof) RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions) {}
func (r *guardedRegistryProof) RegisterActivityWithOptions(fn interface{}, options activity.RegisterOptions) {
	r.functions[options.Name] = fn
}

// Exercise every reviewed actual registration, not merely a convenient mock signature.
// Unwired Activity bodies must never be reached with historical zero-value arguments.
func TestActualImportRegistryDeniesEveryMissingOperatingContext(t *testing.T) {
	registry := &guardedRegistryProof{functions: map[string]interface{}{}}
	RegisterAll(registry, Registrations{HandlerSelection: HandlerSelectionActivities{
		Recommend: func(context.Context, proffer.StageRequest) (proffer.HandlerRecommendationResult, error) {
			panic("body reached")
		},
		Validate: func(context.Context, proffer.StageRequest) (proffer.HandlerSelectionValidationResult, error) {
			panic("body reached")
		},
		Recover: func(context.Context, proffer.HandlerRecoveryRequest) (proffer.HandlerRecommendationResult, error) {
			panic("body reached")
		},
	}})
	reviewed := 0
	for name, fn := range registry.functions {
		if !guardedImportActivity(name) {
			continue
		}
		reviewed++
		t.Run(name, func(t *testing.T) {
			value := reflect.ValueOf(fn)
			results := value.Call([]reflect.Value{reflect.ValueOf(context.Background()), reflect.Zero(value.Type().In(1))})
			require.False(t, results[1].IsNil())
			require.Contains(t, results[1].Interface().(error).Error(), "operating_mode")
		})
	}
	require.Greater(t, reviewed, 35)
}
