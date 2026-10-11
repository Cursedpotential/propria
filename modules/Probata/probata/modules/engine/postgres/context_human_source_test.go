package postgres

import (
 "encoding/json"
 "testing"
 "time"
 "go.temporal.io/sdk/activity"
 "go.temporal.io/sdk/workflow"
 "github.com/Cursedpotential/probata/engine/humancontext"
)

// TestHumanRecoveredStageBindsNativePins rejects replay with changed attribution, source pins or predecessor bundle.
// Inputs: synthetic metadata only. Outputs: exact receipt or mismatch. Effects: no SQL or file access.
func TestHumanRecoveredStageBindsNativePins(t *testing.T){
 req:=humancontext.Request{ContractVersion:humancontext.Contract,WorkflowID:"synthetic-workflow",ManifestRef:"file:///synthetic/framed.json",Source:humancontext.Source{SourceRef:"synthetic://sms.xml",SourceVersionRef:"synthetic-version",ETag:"synthetic-etag",ByteLength:123,OwnerPersonRef:"synthetic-owner"}}
 result:=humancontext.Result{ContractVersion:humancontext.Contract,Stage:"extracted",ManifestRef:"file:///synthetic/extracted.json",ReceiptRef:"synthetic-receipt"}
 raw,_:=json.Marshal(humanStageMetadata{Contract:humancontext.Contract,WorkflowID:req.WorkflowID,Source:req.Source,InputManifestRef:req.ManifestRef,Result:result})
 if got,err:=decodeHumanStage(raw,req,"extracted");err!=nil||got!=result{t.Fatalf("receipt %+v %v",got,err)}
 for _,change:=range []func(*humancontext.Request){func(r *humancontext.Request){r.Source.ETag="changed"},func(r *humancontext.Request){r.Source.OwnerPersonRef="different"},func(r *humancontext.Request){r.ManifestRef="different"},func(r *humancontext.Request){r.Source.ByteLength++}}{altered:=req;change(&altered);if _,err:=decodeHumanStage(raw,altered,"extracted");err==nil{t.Fatal("changed request silently recovered")}}
}

// TestNativeStageTimingPreservesActualRetryClock proves the fourth attempt retains its real start time.
// Inputs: synthetic SDK Activity info with distinct timestamps. Output: exact native coordinates; no DB effects.
func TestNativeStageTimingPreservesActualRetryClock(t *testing.T){started:=time.Date(2026,10,11,1,2,3,0,time.UTC);completed:=started.Add(23*time.Minute);info:=activity.Info{StartedTime:started,Attempt:4,ActivityID:"synthetic-chunk",WorkflowExecution:workflow.Execution{ID:"synthetic-workflow",RunID:"synthetic-run"}};got,err:=nativeStageTiming(info,completed);if err!=nil||got.Attempt!=4||!got.StartedAt.Equal(started)||!got.OperationCompletedAt.Equal(completed)||got.StartedAt.Equal(got.OperationCompletedAt)||got.WorkflowRunID!="synthetic-run"{t.Fatalf("actual metadata lost %+v %v",got,err)};info.StartedTime=time.Time{};if _,err:=nativeStageTiming(info,completed);err==nil{t.Fatal("unknown start clock fabricated")}}
