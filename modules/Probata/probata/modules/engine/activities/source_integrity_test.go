// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package activities

// Byline: Codex · 2026-10-04. Synthetic readers only; no production source access.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// integrityTestStore retains evidence and counts reads for bounded Activity contract tests.
// Inputs: synthetic reader. Outputs: exact receipt refs; side effects: test counters only.
type integrityTestStore struct {
	reader        io.ReadCloser
	size          int64
	prior         proffer.StageResult
	found         bool
	opens         int
	spec          SourceIntegrityReceiptSpec
	persisted     bool
	persistLive   bool
	persistErr    error
	openErr       error
	loadedRequest proffer.StageRequest
	openedRequest proffer.StageRequest
}

// TestSourceIntegrityActivityReferenceBounds rejects oversized coordinates and unsupported payload slots before I/O.
// Inputs: synthetic reference-only requests with one invalid field. Outputs: no load/open/persist assertions.
// Effects: memory only; choose to keep this Activity's history inputs bounded and source-body-free.
func TestSourceIntegrityActivityReferenceBounds(t *testing.T) {
	for _, kind := range []string{"request", "version", "original", "operation", "format", "extra-ref", "matter", "court-case"} {
		t.Run(kind, func(t *testing.T) {
			req := integrityTestRequest()
			switch kind {
			case "request":
				req.RequestID = strings.Repeat("x", 257)
			case "version":
				req.SourceVersionRef = proffer.Ref(strings.Repeat("x", 65))
			case "original":
				req.Refs["original"] = proffer.Ref(strings.Repeat("x", 65))
			case "operation":
				req.Refs["integrity_operation"] = proffer.Ref(strings.Repeat("x", 257))
			case "format":
				req.DeclaredFormat = strings.Repeat("x", 129)
			case "extra-ref":
				req.Refs["source-body"] = "synthetic forbidden payload"
			case "matter":
				req.MatterID = "unsupported legacy slot"
			case "court-case":
				req.CourtCaseID = "unsupported legacy slot"
			}
			store := &integrityTestStore{}
			_, err := (SourceIntegrityActivities{Store: store}).AssessSourceIntegrity(context.Background(), req)
			if err == nil || store.opens != 0 || store.persisted || store.loadedRequest.RequestID != "" {
				t.Fatal("invalid coordinates reached source/receipt storage")
			}
		})
	}
}

// integrityClosingFixture counts reads/closes and optionally fails final closure.
// Inputs: synthetic reader/close error. Outputs: bounded I/O and counters. Effects: memory only;
// choose to prove failed opening and failed closure cannot produce complete coverage.
type integrityClosingFixture struct {
	reader   io.Reader
	closeErr error
	reads    int
	closes   int
}

// Read forwards one synthetic read and counts it.
// Inputs: scanner buffer. Outputs: reader result. Effects: memory counters only; choose for open-error tripwires.
func (r *integrityClosingFixture) Read(p []byte) (int, error) { r.reads++; return r.reader.Read(p) }

// Close records synthetic closure without altering any source object.
// Inputs: none. Outputs: configured error. Effects: memory counter only; choose for close-once evidence.
func (r *integrityClosingFixture) Close() error { r.closes++; return r.closeErr }

// TestSourceIntegrityActivityOpenCloseAndReferenceBinding preserves exact coordinates and unknown I/O failures.
// Inputs: synthetic opener/closer faults and one reference-only request. Outputs: bounded evidence assertions.
// Effects: memory only; choose to catch resource leaks and accidental current-source substitution.
func TestSourceIntegrityActivityOpenCloseAndReferenceBinding(t *testing.T) {
	for _, kind := range []string{"nil-reader", "open-error", "reader-and-open-error", "close-error", "exact-references"} {
		t.Run(kind, func(t *testing.T) {
			reader := &integrityClosingFixture{reader: bytes.NewReader([]byte{0, 1})}
			store := &integrityTestStore{reader: reader, size: 2}
			switch kind {
			case "nil-reader":
				store.reader = nil
			case "open-error":
				store.reader = nil
				store.openErr = errors.New("private source error must not enter evidence")
			case "reader-and-open-error":
				store.openErr = errors.New("private source error must not enter evidence")
			case "close-error":
				reader.closeErr = errors.New("private close error must not enter evidence")
			}
			req := integrityTestRequest()
			req.DeclaredFormat = "synthetic"
			want, _ := json.Marshal(req)
			body := SourceIntegrityActivities{Store: store, Execution: func(context.Context) (string, string) { return "check-job", "synthetic-run" }}
			result, err := body.AssessSourceIntegrity(context.Background(), req)
			if err != nil || !store.persisted || result.ReceiptRef == "" {
				t.Fatal("missing terminal evidence", err)
			}
			for _, got := range []proffer.StageRequest{store.loadedRequest, store.openedRequest, store.spec.Request} {
				encoded, _ := json.Marshal(got)
				if !bytes.Equal(encoded, want) {
					t.Fatal("source-version/original/request/operation binding changed")
				}
			}
			if store.spec.Assessment.Status != "unknown" || store.spec.Assessment.FormatValidation != "unassessed" {
				t.Fatal("I/O or positive bytes promoted")
			}
			if kind == "exact-references" {
				if !store.spec.Assessment.Complete || result.Status != proffer.StatusSuccess || reader.closes != 1 {
					t.Fatal("complete byte-only assessment changed")
				}
			} else {
				if store.spec.Assessment.Complete || result.Status != proffer.StatusFailed || result.Ref != "" {
					t.Fatal("failed source access promoted")
				}
				if kind == "reader-and-open-error" && (reader.closes != 1 || reader.reads != 0) {
					t.Fatal("failed opener reader leaked or scanned")
				}
				if kind == "close-error" && (reader.closes != 1 || store.spec.Assessment.Reason != "integrity_stream_close_failed") {
					t.Fatal("close failure lost")
				}
			}
		})
	}
}

// LoadSourceIntegrity returns a configured terminal receipt without reopening bytes.
// Inputs: test request. Outputs: configured result; side effects: none.
func (s *integrityTestStore) LoadSourceIntegrity(_ context.Context, req proffer.StageRequest) (proffer.StageResult, bool, error) {
	s.loadedRequest = req
	return s.prior, s.found, nil
}

// OpenSourceIntegrity counts test source access and exposes its synthetic reader.
// Inputs: request. Outputs: reader/size; side effects: test counter only.
func (s *integrityTestStore) OpenSourceIntegrity(_ context.Context, req proffer.StageRequest) (io.ReadCloser, int64, error) {
	s.opens++
	s.openedRequest = req
	return s.reader, s.size, s.openErr
}

// PersistSourceIntegrity captures coverage with the same success/failure shape as the SQL adapter.
// Inputs: spec. Outputs: compact receipt refs; side effects: test state only.
func (s *integrityTestStore) PersistSourceIntegrity(ctx context.Context, spec SourceIntegrityReceiptSpec) (proffer.StageResult, error) {
	s.spec = spec
	s.persisted = true
	s.persistLive = ctx.Err() == nil
	if s.persistErr != nil {
		return proffer.StageResult{}, s.persistErr
	}
	r := proffer.StageResult{Stage: stagegraph.AssessSourceIntegrity, ReceiptRef: "receipt"}
	if spec.Assessment.Complete {
		r.Status = proffer.StatusSuccess
		r.Ref = "receipt"
	} else {
		r.Status = proffer.StatusFailed
		r.Reason = spec.Assessment.Reason
	}
	return r, nil
}

// integrityTestRequest builds only synthetic compact references for wrapper tests.
// Input: none. Output: request; side effects: none.
func integrityTestRequest() proffer.StageRequest {
	return proffer.StageRequest{RequestID: "source-job", SourceVersionRef: "version", Refs: map[string]proffer.Ref{"original": "object", "integrity_operation": "check-job"}}
}

// TestSourceIntegrityActivityCoverageAndTerminalRecovery preserves complete byte outcomes and terminal replay.
// Inputs: synthetic streams. Outputs: coverage and no-reopen assertions. Effects: memory only;
// choose to distinguish successful execution from usable or canonical evidence.
func TestSourceIntegrityActivityCoverageAndTerminalRecovery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   []byte
		status string
	}{{"zero", nil, "zero_byte"}, {"allzero", make([]byte, 131075), "all_zero"}, {"late-nonzero", append(make([]byte, 131075), 1), "unknown"}} {
		t.Run(tc.name, func(t *testing.T) {
			s := &integrityTestStore{reader: io.NopCloser(bytes.NewReader(tc.body)), size: int64(len(tc.body))}
			a := SourceIntegrityActivities{Store: s, Attempt: func(context.Context) int32 { return 2 }, Execution: func(context.Context) (string, string) { return "check-job", "run" }}
			r, err := a.AssessSourceIntegrity(context.Background(), integrityTestRequest())
			if err != nil || r.Status != proffer.StatusSuccess || s.spec.Assessment.Status != tc.status || !s.spec.Assessment.Complete || s.spec.Assessment.CheckedBytes != s.size || s.spec.Attempt != 2 || s.spec.OperationRunID != "run" {
				t.Fatalf("result=%+v evidence=%+v err=%v", r, s.spec, err)
			}
			s.prior = r
			s.found = true
			_, err = a.AssessSourceIntegrity(context.Background(), integrityTestRequest())
			if err != nil || s.opens != 1 {
				t.Fatalf("recovery reopened bytes: %d %v", s.opens, err)
			}
		})
	}
	failed := &integrityTestStore{found: true, prior: proffer.StageResult{Stage: stagegraph.AssessSourceIntegrity, Status: proffer.StatusFailed, ReceiptRef: "failed", Reason: "stream_read_failed"}}
	r, err := (SourceIntegrityActivities{Store: failed}).AssessSourceIntegrity(context.Background(), integrityTestRequest())
	if err != nil || r.ReceiptRef != "failed" || failed.opens != 0 {
		t.Fatal("terminal failure recovery must not retry source")
	}
}

// TestSourceIntegrityActivitySDKMetadataAndHeartbeat proves Temporal attempt/run correlation.
// Inputs: SDK test environment and synthetic reader. Outputs: recorded metadata assertions.
// Effects: memory only; choose before worker wiring without running a live workflow.
func TestSourceIntegrityActivitySDKMetadataAndHeartbeat(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	store := &integrityTestStore{reader: io.NopCloser(bytes.NewReader([]byte{0, 0})), size: 2}
	body := NewSourceIntegrityActivities(store)
	RegisterSourceIntegrityActivity(env, body)
	env.ExecuteWorkflow(proffer.SourceIntegrityWorkflow, proffer.SourceIntegrityInput{RequestID: "source-job", SourceVersionRef: "version", OriginalRef: "object"})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if store.spec.StartedAt.IsZero() || store.spec.ActivityID == "" || store.spec.ExecutionHost == "" || store.spec.OperationRunID == "" || store.spec.Attempt != 1 {
		t.Fatalf("SDK metadata missing: %+v", store.spec)
	}
}

// integrityBrokenReader supplies a prefix and an I/O failure without leaking error content into evidence.
// Inputs: scanner buffer. Outputs: three zeros and error; side effects: none.
type integrityBrokenReader struct{}

// Read returns a synthetic prefix and fixed I/O failure.
// Inputs: scanner buffer. Outputs: zeros/error. Effects: memory only; choose to reject partial coverage.
func (integrityBrokenReader) Read(p []byte) (int, error) {
	copy(p, []byte{0, 0, 0})
	return 3, errors.New("private body must never persist")
}

// Close completes the synthetic failure reader without source side effects.
// Input: none. Output: nil; side effects: none.
func (integrityBrokenReader) Close() error { return nil }

// integrityBlockingReader blocks until cancellation closes it, proving liveness and interruption.
// Inputs: scanner reads/Close. Outputs: I/O failure; side effects: test signals only.
type integrityBlockingReader struct {
	closed chan struct{}
	closes atomic.Int32
}

// Read waits for Close in the cancellation fixture.
// Input: buffer. Output: fixed error; side effects: waits on test signal only.
func (r *integrityBlockingReader) Read([]byte) (int, error) { <-r.closed; return 0, io.ErrClosedPipe }

// Close releases the synthetic blocked read once.
// Input: none. Output: nil; side effects: increments test close count.
func (r *integrityBlockingReader) Close() error {
	if r.closes.Add(1) == 1 {
		close(r.closed)
	}
	return nil
}

// TestSourceIntegrityActivityFailureCancellationAndFailClosed retains unknown coverage and cancellation receipts.
// Inputs: failed/blocked synthetic readers and persistence faults. Outputs: bounded failure assertions.
// Effects: memory only; choose to prove source errors cannot promote byte prefixes.
func TestSourceIntegrityActivityFailureCancellationAndFailClosed(t *testing.T) {
	s := &integrityTestStore{reader: integrityBrokenReader{}, size: 9}
	r, err := (SourceIntegrityActivities{Store: s}).AssessSourceIntegrity(context.Background(), integrityTestRequest())
	if err != nil || r.Status != proffer.StatusFailed || s.spec.Assessment.CheckedBytes != 3 || s.spec.Assessment.Status != "unknown" || s.spec.Assessment.Complete || s.spec.Assessment.Reason != "stream_read_failed" {
		t.Fatalf("failure %+v %+v %v", r, s.spec, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	br := &integrityBlockingReader{closed: make(chan struct{})}
	s = &integrityTestStore{reader: br, size: 9}
	var beats atomic.Int32
	a := SourceIntegrityActivities{Store: s, HeartbeatEvery: time.Millisecond, Heartbeat: func(context.Context, Progress) {
		if beats.Add(1) >= 2 {
			cancel()
		}
	}}
	r, err = a.AssessSourceIntegrity(ctx, integrityTestRequest())
	if !temporal.IsCanceledError(err) || r.Status != proffer.StatusFailed || !s.persistLive || s.spec.Assessment.Reason != "context_canceled" || br.closes.Load() != 1 || beats.Load() < 2 {
		t.Fatalf("cancellation %+v %+v %v closes=%d beats=%d", r, s.spec, err, br.closes.Load(), beats.Load())
	}
	if _, err = (SourceIntegrityActivities{}).AssessSourceIntegrity(context.Background(), integrityTestRequest()); err == nil {
		t.Fatal("missing store must fail closed")
	}
	s = &integrityTestStore{persistErr: errors.New("database unavailable"), reader: io.NopCloser(bytes.NewReader(nil)), size: 0}
	r, err = (SourceIntegrityActivities{Store: s}).AssessSourceIntegrity(context.Background(), integrityTestRequest())
	if err == nil || r.ReceiptRef != "" {
		t.Fatal("unavailable persistence fabricated receipt")
	}
	a = SourceIntegrityActivities{Store: s, Execution: func(context.Context) (string, string) { return "wrong-job", "run" }}
	if _, err = a.AssessSourceIntegrity(context.Background(), integrityTestRequest()); err == nil {
		t.Fatal("wrong operation binding accepted")
	}
}
