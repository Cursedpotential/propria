// Byline: Codex · GPT-5 · 2026-10-05.
package profferworker

import (
	"context"
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
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
