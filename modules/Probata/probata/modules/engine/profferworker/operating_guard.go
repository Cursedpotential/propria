// Byline: Codex · GPT-5 · 2026-10-05.
package profferworker

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.temporal.io/sdk/activity"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

type completeRegistrar interface {
	activities.ActivityRegistrar
	RegisterWorkflow(interface{})
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}

type operatingRegistrar struct{ completeRegistrar }

// RegisterActivityWithOptions fences the reviewed Proffer import registration families.
// Inputs: the original typed function and registered name. Outputs: the same function signature.
// Effects: registration only; rejected unknown/DEV/foreign-scope invocations never call the body.
// Independent governed maintenance Activities remain unchanged; extraction and
// Python publishers use separate typed admission at their own write boundaries.
func (r operatingRegistrar) RegisterActivityWithOptions(fn interface{}, options activity.RegisterOptions) {
	if guardedImportActivity(options.Name) {
		fn = guardImportActivity(fn, options.Name)
	}
	r.completeRegistrar.RegisterActivityWithOptions(fn, options)
}

func guardedImportActivity(name string) bool {
	for _, group := range [][]stagegraph.Descriptor{stagegraph.Stages, stagegraph.OptionalStages} {
		for _, stage := range group {
			if string(stage.ID) == name {
				return true
			}
		}
	}
	switch name {
	case "hash_source_activity", "hash_raw_records_activity", "hash_raw_generation_activity",
		string(stagegraph.RepairFindOtherVersion), string(stagegraph.RepairSalvageTruncatedXML),
		string(stagegraph.RepairLenientDecode), string(stagegraph.RepairRecordStepReceipt),
		proffer.SelectStructuredELTActivityName, proffer.ExecuteStructuredELTActivityName,
		proffer.RecommendHandlerActivityName, proffer.ValidateHandlerSelectionActivityName, proffer.RecoverHandlerActivityName,
		activities.ListBatchFolderActivityName, activities.BindImportOperationActivityName:
		return true
	}
	return false
}

// guardImportActivity preserves SDK argument/return types and rejects unexpected registered
// signatures at startup. Only the reviewed concrete payload types below can authorize a body.
func guardImportActivity(fn interface{}, name string) interface{} {
	value := reflect.ValueOf(fn)
	signature := value.Type()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if signature.Kind() != reflect.Func || signature.NumIn() != 2 || signature.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		signature.NumOut() != 2 || signature.Out(1) != errorType {
		panic("operating guard: unreviewed import Activity signature: " + name)
	}
	return reflect.MakeFunc(signature, func(args []reflect.Value) []reflect.Value {
		if err := importActivityAdmission(args[1].Interface()); err != nil {
			denied := sdktemporal.NewNonRetryableApplicationError(err.Error(), "operating_context_denied", err)
			return []reflect.Value{reflect.Zero(signature.Out(0)), reflect.ValueOf(denied)}
		}
		return value.Call(args)
	}).Interface()
}

func importActivityAdmission(payload interface{}) error {
	var mode, matter, court string
	switch req := payload.(type) {
	case proffer.StageRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case repairplan.StepRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case repairplan.ReceiptRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case proffer.HandlerRecoveryRequest:
		return importActivityAdmission(req.Request)
	case proffer.PreviewPublicationRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case proffer.AutoApprovalRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case activities.ListBatchFolderRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case activities.BindImportOperationRequest:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	case activities.AICandidateStageInput:
		mode, matter, court = req.OperatingMode, req.MatterID, req.CourtCaseID
	default:
		return fmt.Errorf("unreviewed import Activity payload %T", payload)
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(mode)); err != nil {
		return err
	}
	if !caseidentity.AdmittedIdentity(matter, court) {
		return errors.New("import Activity requires the approved case identity")
	}
	return nil
}
