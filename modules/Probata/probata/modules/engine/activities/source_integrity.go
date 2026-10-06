// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package activities

// Byline: Codex · 2026-10-04. Receipt sibling: postgres.DeriveStore; stream sibling: HashActivities.FingerprintSource.

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceintegrity"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// SourceIntegrityReceiptSpec binds bounded coverage to a source, operation, and Temporal attempt.
// Input: source references and scanner evidence. Output: persistence coordinates; no source bodies or hashes.
type SourceIntegrityReceiptSpec struct {
	Request        proffer.StageRequest
	Assessment     sourceintegrity.Assessment
	Attempt        int32
	OperationRunID string
	StartedAt      time.Time
	ActivityID     string
	ExecutionHost  string
}

// SourceIntegrityStore resolves retained bytes and records this operation independently of metadata and hashes.
// Inputs: validated source/operation references. Outputs: stream/expected length or exact durable terminal refs.
// Side effects: read-only source access and integrity receipts; Load must recover terminal outcomes before reopening bytes.
// Implementations enforce the exact retained source-version/original relation on load/open/persist,
// open from offset zero, and keep errors free of source bodies; choose over current-selection stores.
type SourceIntegrityStore interface {
	LoadSourceIntegrity(context.Context, proffer.StageRequest) (proffer.StageResult, bool, error)
	OpenSourceIntegrity(context.Context, proffer.StageRequest) (io.ReadCloser, int64, error)
	PersistSourceIntegrity(context.Context, SourceIntegrityReceiptSpec) (proffer.StageResult, error)
}

// SourceIntegrityActivities owns only whole-stream byte assessment and its own execution receipt.
// Inputs: store and optional Temporal callbacks. Outputs: one reference-only Activity result.
// Pick independently of fingerprinting, format detection, parsing, repair, and canonical selection.
type SourceIntegrityActivities struct {
	Store           SourceIntegrityStore
	Attempt         Attempt
	Heartbeat       Heartbeat
	Execution       func(context.Context) (workflowID, runID string)
	HeartbeatEvery  time.Duration
	RuntimeMetadata func(context.Context) (startedAt time.Time, activityID, executionHost string)
}

// NewSourceIntegrityActivities binds the existing Temporal attempt and heartbeat conventions to integrity.
// Input: production receipt/source adapter. Output: one Activity implementation. Side effects: none until called.
// Pick for worker wiring; missing storage fails closed rather than producing a synthetic receipt.
func NewSourceIntegrityActivities(store SourceIntegrityStore) SourceIntegrityActivities {
	return SourceIntegrityActivities{
		Store:     store,
		Attempt:   func(ctx context.Context) int32 { return activity.GetInfo(ctx).Attempt },
		Heartbeat: func(ctx context.Context, progress Progress) { activity.RecordHeartbeat(ctx, progress) },
		Execution: func(ctx context.Context) (string, string) {
			info := activity.GetInfo(ctx)
			return info.WorkflowExecution.ID, info.WorkflowExecution.RunID
		},
		RuntimeMetadata: func(ctx context.Context) (time.Time, string, string) {
			info := activity.GetInfo(ctx)
			host, _ := os.Hostname()
			return info.StartedTime, info.ActivityID, host
		},
	}
}

// RegisterSourceIntegrityActivity registers exactly one independent integrity Activity.
// Inputs: registrar and body. Output: named worker registration. Side effects: worker registry only.
// Pick for standalone integrity jobs; it adds no processing to existing Activities or workflows.
func RegisterSourceIntegrityActivity(registrar ActivityRegistrar, body SourceIntegrityActivities) {
	registrar.RegisterActivityWithOptions(body.AssessSourceIntegrity, activity.RegisterOptions{Name: string(stagegraph.AssessSourceIntegrity)})
}

// AssessSourceIntegrity records a bounded byte assessment of one retained original, independently of all other operations.
// Input: source version/original references, originating RequestID, and integrity_operation workflow ID in Refs.
// Output: durable result/receipt refs; incomplete reads produce failed receipts with unknown evidence.
// Side effects: streams retained bytes and writes only its own receipt; no hashing, metadata extraction, parsing, or promotion.
// Pick before relying on whole-file zero-content claims; successful execution still leaves nonzero usability unknown.
func (a SourceIntegrityActivities) AssessSourceIntegrity(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	startedAt := time.Now().UTC()
	activityID, executionHost := "", ""
	if a.RuntimeMetadata != nil {
		observed, id, host := a.RuntimeMetadata(ctx)
		if !observed.IsZero() {
			startedAt = observed.UTC()
		}
		activityID, executionHost = id, host
	}
	if a.Store == nil {
		return proffer.StageResult{}, errors.New("source integrity receipt store is required")
	}
	if strings.TrimSpace(req.RequestID) == "" || len(req.RequestID) > 256 || req.SourceVersionRef == "" || len(req.SourceVersionRef) > 64 || req.Refs["original"] == "" || len(req.Refs["original"]) > 64 || strings.TrimSpace(string(req.Refs["integrity_operation"])) == "" || len(req.Refs["integrity_operation"]) > 256 || len(req.DeclaredFormat) > 128 || len(req.Refs) != 2 || req.MatterID != "" || req.CourtCaseID != "" {
		return proffer.StageResult{}, errors.New("source integrity requires source, original, request and operation references")
	}
	runID := ""
	if a.Execution != nil {
		workflowID, actualRun := a.Execution(ctx)
		if workflowID != string(req.Refs["integrity_operation"]) {
			return proffer.StageResult{}, errors.New("source integrity operation does not match Temporal workflow")
		}
		runID = actualRun
	}
	assessment := sourceintegrity.Unchecked(-1)
	if ctx.Err() == nil {
		prior, found, err := a.Store.LoadSourceIntegrity(ctx, req)
		if err != nil && ctx.Err() == nil {
			return proffer.StageResult{}, err
		}
		if found {
			if ctx.Err() != nil {
				return prior, temporal.NewCanceledError(prior)
			}
			return prior, nil
		}
		reader, expected, err := a.Store.OpenSourceIntegrity(ctx, req)
		assessment = sourceintegrity.Unchecked(expected)
		if err != nil {
			if reader != nil {
				_ = reader.Close()
			}
			assessment.Reason = "source_access_failed"
		} else if reader == nil {
			assessment.Reason = "source_reader_unavailable"
		} else {
			guard := &integrityReadCloser{ReadCloser: reader}
			stopClose := context.AfterFunc(ctx, func() { _ = guard.Close() })
			counter := &integrityProgressReader{reader: guard}
			stopHeartbeat := a.integrityHeartbeat(ctx, counter)
			assessment, _ = sourceintegrity.InspectStream(ctx, counter, expected)
			stopHeartbeat()
			stopClose()
			if closeErr := guard.Close(); closeErr != nil && ctx.Err() == nil {
				assessment.Status, assessment.Reason, assessment.Complete = "unknown", "integrity_stream_close_failed", false
			}
			if a.Heartbeat != nil && ctx.Err() == nil {
				a.Heartbeat(ctx, Progress{Stage: stagegraph.AssessSourceIntegrity, BytesComplete: assessment.CheckedBytes})
			}
		}
	}
	if ctx.Err() != nil {
		assessment.Status, assessment.Reason, assessment.Complete = "unknown", "context_canceled", false
	}
	attempt := int32(1)
	if a.Attempt != nil {
		if actual := a.Attempt(ctx); actual > 0 {
			attempt = actual
		}
	}
	// A canceled source read must not cancel its bounded failure receipt write.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	result, err := a.Store.PersistSourceIntegrity(persistCtx, SourceIntegrityReceiptSpec{Request: req, Assessment: assessment, Attempt: attempt, OperationRunID: runID, StartedAt: startedAt, ActivityID: activityID, ExecutionHost: executionHost})
	if err != nil {
		return proffer.StageResult{}, err
	}
	if ctx.Err() != nil {
		return result, temporal.NewCanceledError(result)
	}
	return result, nil
}

// integrityReadCloser closes a retained stream once even when cancellation races finalization.
// Input: retained reader. Output: stable close result. Side effects: closes that stream only.
type integrityReadCloser struct {
	io.ReadCloser
	once sync.Once
	err  error
}

// Close serializes source closure across cancellation and normal completion.
// Input: none. Output: first close error. Side effects: one underlying Close; pick for cancellable integrity I/O.
func (r *integrityReadCloser) Close() error {
	r.once.Do(func() { r.err = r.ReadCloser.Close() })
	return r.err
}

// integrityProgressReader counts stream reads without keeping any bytes.
// Input: read-only reader. Output: atomic progress count; side effects: reads only.
type integrityProgressReader struct {
	reader io.Reader
	count  atomic.Int64
}

// Read counts bytes after each source read for compact liveness heartbeats.
// Input: scanner buffer. Output: reader result. Side effects: reads/count updates; pick only for integrity progress.
func (r *integrityProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count.Add(int64(n))
	return n, err
}

// integrityHeartbeat reports initial and timed liveness without resuming a partial zero check.
// Inputs: Activity context and atomic counter. Output: stop function. Side effects: compact heartbeats only.
// Pick for blocked/long source streams; it performs no additional source reads and stops before receipt publication.
func (a SourceIntegrityActivities) integrityHeartbeat(ctx context.Context, reader *integrityProgressReader) func() {
	if a.Heartbeat == nil {
		return func() {}
	}
	a.Heartbeat(ctx, Progress{Stage: stagegraph.AssessSourceIntegrity})
	every := a.HeartbeatEvery
	if every <= 0 {
		every = 20 * time.Second
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				a.Heartbeat(ctx, Progress{Stage: stagegraph.AssessSourceIntegrity, BytesComplete: reader.count.Load()})
			}
		}
	}()
	return func() { close(stop); <-done }
}
