package humancontext

import (
 "context"
 "errors"
 "time"
 "go.temporal.io/sdk/activity"
)

// Activities exposes framing, semantic extraction, publication and readback as independently retryable units.
// Inputs: configured native bundles/indexer. Outputs: compact external-reference receipts.
// Effects: each method performs its named unit only, with cancellable reference-only heartbeats.
type Activities struct { Bundles *Bundles; Indexer *Indexer }

// Frame measures native original syntax after registration, without a legacy handler or custody gate.
// Inputs: registered source references. Outputs: framed bundle result. Effects: original reads/derivative metadata.
func(a *Activities)Frame(ctx context.Context,req Request)(Result,error){if a.Bundles==nil{return Result{},errors.New("human bundles unavailable")};defer heartbeat(ctx,req,FrameActivity)();return a.Bundles.Frame(ctx,req)}

// Extract maps one native lexical record at a time through DuckDB, independently of indexing.
// Inputs: framed manifest reference. Outputs: extracted native bundle. Effects: derivative files/metadata only.
func(a *Activities)Extract(ctx context.Context,req Request)(Result,error){if a.Bundles==nil{return Result{},errors.New("human bundles unavailable")};defer heartbeat(ctx,req,ExtractActivity)();return a.Bundles.Extract(ctx,req)}

// Index publishes externally planned native conversation chunks through the configured existing embedder and search store.
// Inputs: chunk-plan manifest reference. Outputs: publication receipt. Effects: deterministic search upserts.
func(a *Activities)Index(ctx context.Context,req Request)(Result,error){if a.Indexer==nil{return Result{},errors.New("human indexer unavailable")};defer heartbeat(ctx,req,IndexActivity)();return a.Indexer.Publish(ctx,req)}

// Verify independently reads every published object and its vector before recording verification.
// Inputs: publication manifest reference. Outputs: readback receipt. Effects: search reads/derivative metadata.
func(a *Activities)Verify(ctx context.Context,req Request)(Result,error){if a.Indexer==nil{return Result{},errors.New("human indexer unavailable")};defer heartbeat(ctx,req,VerifyActivity)();return a.Indexer.Verify(ctx,req)}

// heartbeat records compact stage/source references while a long-running atomic unit is active.
// Inputs: Activity context and references. Outputs: a joined stop function. Effects: Temporal heartbeats only.
// Direct callers require no Temporal runtime and get a no-op stop function.
func heartbeat(ctx context.Context,req Request,stage string)func(){
 if !activity.IsActivity(ctx){return func(){}}
 beat:=func(){activity.RecordHeartbeat(ctx,map[string]string{"stage":stage,"source_version_ref":req.Source.SourceVersionRef,"manifest_ref":req.ManifestRef})}
 beat();done,joined:=make(chan struct{}),make(chan struct{})
 go func(){defer close(joined);timer:=time.NewTicker(15*time.Second);defer timer.Stop();for{select{case <-done:return;case <-ctx.Done():return;case <-timer.C:beat()}}}()
 return func(){close(done);<-joined}
}
