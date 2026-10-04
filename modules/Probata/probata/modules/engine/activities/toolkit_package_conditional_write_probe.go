// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/aws/smithy-go"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitPackageConditionalWriteProbeActivityName = "toolkit_package_conditional_write_probe_activity"
	ToolkitPackageConditionalWriteProbeWorkflowName = "ToolkitPackageConditionalWriteProbeWorkflow"
	toolkitPackageProbeRoot                         = "b2://salem-data/consignatio/casevault/recovery/library-sources/_provider-probes/"
	toolkitPackageProbePayloadBytes                 = 16
	toolkitPackageProbeMaxNamespaceBytes            = 64
	toolkitPackageProbeMinNamespaceBytes            = 16
	toolkitPackageProbeActivityTimeout              = 2 * time.Minute
)

// ToolkitPackageConditionalWriteProbeInput identifies one fresh disposable probe key.
// Inputs: a safe, globally unique probe namespace that has not been used before.
// Outputs: none; the Activity returns a bounded result with the fixed probe reference.
// Side effects: writes only two synthetic 16-byte values beneath the dedicated provider-probes prefix; any uploaded probe remains there.
// Choose this probe before preservation runs when provider-enforced create-only behavior lacks documented or live proof.
type ToolkitPackageConditionalWriteProbeInput struct {
	ProbeNamespace string `json:"probe_namespace"`
}

// ToolkitPackageConditionalWriteProbeResult reports conditional-write behavior without archive or credential data.
// Inputs: one probe attempt; outputs: support verdict, phase, fixed object ref, and bounded diagnostic fields.
// Side effects: none while marshaling; a live probe may leave its synthetic key and version present indefinitely.
// Choose this result instead of treating an SDK serialization test as proof of provider enforcement.
type ToolkitPackageConditionalWriteProbeResult struct {
	Supported         bool        `json:"supported"`
	Outcome           string      `json:"outcome"`
	ProbeRef          proffer.Ref `json:"probe_ref"`
	ProviderErrorCode string      `json:"provider_error_code,omitempty"`
	ProviderError     string      `json:"provider_error,omitempty"`
	FirstPutAccepted  bool        `json:"first_put_accepted"`
	SecondPutRejected bool        `json:"second_put_rejected"`
	FinalReadMatchesA bool        `json:"final_read_matches_a"`
	ProbeMayExist     bool        `json:"probe_may_exist"`
}

// ToolkitPackageConditionalWriteProbeActivities reuses the preservation group's configured resolver and heartbeat.
// Inputs: the already-constructed preservation Activity group; outputs: an independently registerable probe Activity.
// Side effects: none until the probe Activity executes; it creates no client or credential configuration.
// Choose this constructor to probe the exact same resolver instance used by archive preservation.
type ToolkitPackageConditionalWriteProbeActivities struct {
	Preservation ToolkitPackagePreservationActivities
}

// NewToolkitPackageConditionalWriteProbeActivities binds the exact configured preservation resolver to the provider probe.
// Inputs: the existing preservation Activity group; outputs: a probe-only Activity handler.
// Side effects: none. Choose to avoid a second client, credential source, or provider configuration path.
// Byline: Codex · GPT-6 · 2026-10-04.
func NewToolkitPackageConditionalWriteProbeActivities(preservation ToolkitPackagePreservationActivities) ToolkitPackageConditionalWriteProbeActivities {
	return ToolkitPackageConditionalWriteProbeActivities{Preservation: preservation}
}

// ToolkitPackageConditionalWriteProbeWorkflow schedules one bounded live provider-semantics probe.
// Inputs: a fresh unique synthetic-key namespace; outputs: a supported verdict and bounded diagnostics.
// Side effects: schedules one Activity that can leave only a small synthetic probe object in Casevault.
// Choose before original-package writes; this workflow never accepts or opens an original archive reference.
// Byline: Codex · GPT-6 · 2026-10-04.
func ToolkitPackageConditionalWriteProbeWorkflow(ctx workflow.Context, input ToolkitPackageConditionalWriteProbeInput) (ToolkitPackageConditionalWriteProbeResult, error) {
	if !safeToolkitProbeNamespace(input.ProbeNamespace) {
		return ToolkitPackageConditionalWriteProbeResult{}, temporal.NewNonRetryableApplicationError("probe_namespace must be a fresh unique 16-64 character alphanumeric, dash, or underscore segment", "ToolkitProbeInvalid", nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    toolkitPackageProbeActivityTimeout,
		ScheduleToCloseTimeout: toolkitPackageProbeActivityTimeout,
		HeartbeatTimeout:       30 * time.Second,
		WaitForCancellation:    true,
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	var result ToolkitPackageConditionalWriteProbeResult
	err := workflow.ExecuteActivity(ctx, ToolkitPackageConditionalWriteProbeActivityName, input).Get(ctx, &result)
	return result, err
}

// RunToolkitPackageConditionalWriteProbe checks that a second different conditional write cannot replace the first payload.
// Inputs: context and a fresh probe namespace; outputs: supported only after conflict refusal plus independent final readback of payload A.
// Side effects: may create or overwrite at most the 16-byte synthetic object at the unique probe key; it never deletes or touches originals.
// Choose as provider behavior proof before any preservation archive upload; no unconditional Put fallback is available.
// Byline: Codex · GPT-6 · 2026-10-04.
func (a ToolkitPackageConditionalWriteProbeActivities) RunToolkitPackageConditionalWriteProbe(ctx context.Context, input ToolkitPackageConditionalWriteProbeInput) (ToolkitPackageConditionalWriteProbeResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolkitPackageConditionalWriteProbeResult{}, err
	}
	if !safeToolkitProbeNamespace(input.ProbeNamespace) {
		return ToolkitPackageConditionalWriteProbeResult{}, temporal.NewNonRetryableApplicationError("probe_namespace must be a fresh unique 16-64 character alphanumeric, dash, or underscore segment", "ToolkitProbeInvalid", nil)
	}
	result := ToolkitPackageConditionalWriteProbeResult{
		Outcome:  "resolver_unavailable",
		ProbeRef: toolkitPackageProbeRef(input.ProbeNamespace),
	}
	if a.Preservation.Stores == nil {
		result.ProviderError = "preservation object-store resolver is not configured"
		return result, nil
	}
	store, err := a.Preservation.Stores("b2")
	if err != nil {
		result.Outcome = "resolver_error"
		result.ProviderErrorCode, result.ProviderError = boundedProbeProviderError(err)
		return result, nil
	}
	if store == nil {
		result.Outcome = "resolver_error"
		result.ProviderError = "preservation resolver returned no object store"
		return result, nil
	}
	writer, ok := store.(toolkitConditionalObjectWriter)
	if !ok {
		result.Outcome = "conditional_writer_unavailable"
		result.ProviderError = "resolved adapter does not expose the conditional-write operation"
		return result, nil
	}
	bucket, key, err := toolkitObjectCoordinates(result.ProbeRef)
	if err != nil {
		return ToolkitPackageConditionalWriteProbeResult{}, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if a.Preservation.Heartbeat != nil {
		a.Preservation.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "conditional-probe-preflight"})
	}
	exists, err := store.Exists(ctx, bucket, key)
	if err != nil {
		result.Outcome = "preflight_error"
		result.ProviderErrorCode, result.ProviderError = boundedProbeProviderError(err)
		return result, nil
	}
	if exists {
		result.Outcome = "probe_key_already_exists"
		result.ProviderError = "probe key is occupied; no write was attempted; use a fresh namespace"
		result.ProbeMayExist = true
		return result, nil
	}

	payloadA := []byte("probe-payload-A1")
	payloadB := []byte("probe-payload-B1")
	if len(payloadA) != toolkitPackageProbePayloadBytes || len(payloadB) != toolkitPackageProbePayloadBytes {
		return ToolkitPackageConditionalWriteProbeResult{}, errors.New("synthetic probe payload size invariant failed")
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	result.ProbeMayExist = true // A timed-out PUT may have committed despite its response being lost.
	if a.Preservation.Heartbeat != nil {
		a.Preservation.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "conditional-probe-first-put", Bytes: toolkitPackageProbePayloadBytes})
	}
	err = writer.PutIfAbsent(ctx, bucket, key, strings.NewReader(string(payloadA)), int64(len(payloadA)), "application/octet-stream")
	if err != nil {
		result.Outcome = "first_put_rejected"
		result.ProviderErrorCode, result.ProviderError = boundedProbeProviderError(err)
		result.FinalReadMatchesA = readProbeMatches(ctx, store, bucket, key, payloadA)
		if result.FinalReadMatchesA {
			result.FirstPutAccepted = true
		}
		return result, nil
	}
	result.FirstPutAccepted = true
	if err = ctx.Err(); err != nil {
		result.Outcome = "cancelled_after_first_put"
		return result, err
	}
	if a.Preservation.Heartbeat != nil {
		a.Preservation.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "conditional-probe-second-put", Bytes: toolkitPackageProbePayloadBytes})
	}
	secondErr := writer.PutIfAbsent(ctx, bucket, key, strings.NewReader(string(payloadB)), int64(len(payloadB)), "application/octet-stream")
	result.SecondPutRejected = errors.Is(secondErr, smsthreads.ErrConditionalObjectMismatch)
	result.FinalReadMatchesA = readProbeMatches(ctx, store, bucket, key, payloadA)
	if result.SecondPutRejected && result.FinalReadMatchesA {
		result.Supported = true
		result.Outcome = "conditional_create_verified"
		result.ProviderErrorCode = "conditional_object_mismatch"
		result.ProviderError = "second different payload was refused; fresh read still matches the first synthetic payload"
		return result, nil
	}
	result.Outcome = "conditional_create_not_proven"
	switch {
	case secondErr != nil:
		result.ProviderErrorCode, result.ProviderError = boundedProbeProviderError(secondErr)
	case !result.FinalReadMatchesA:
		result.ProviderErrorCode = "final_readback_mismatch"
		result.ProviderError = "second write was not refused with the expected mismatch and final read does not match payload A"
	default:
		result.ProviderErrorCode = "conditional_conflict_not_observed"
		result.ProviderError = "second write did not return the expected existing-object mismatch"
	}
	return result, nil
}

// safeToolkitProbeNamespace validates a fresh single path segment without accepting separators or URI syntax.
// Inputs: caller-supplied probe namespace; outputs: true only for a 16-64 character safe ASCII segment.
// Side effects: none. Choose before deriving the fixed synthetic-only B2 key.
// Byline: Codex · GPT-6 · 2026-10-04.
func safeToolkitProbeNamespace(value string) bool {
	if len(value) < toolkitPackageProbeMinNamespaceBytes || len(value) > toolkitPackageProbeMaxNamespaceBytes {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// toolkitPackageProbeRef locates one disposable key under the fixed Casevault provider-probes subnamespace.
// Inputs: a namespace already validated by safeToolkitProbeNamespace; outputs: a B2 object reference for synthetic bytes only.
// Side effects: none. Choose this namespace rather than the preservation archive root.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitPackageProbeRef(namespace string) proffer.Ref {
	return proffer.Ref(toolkitPackageProbeRoot + namespace)
}

// boundedProbeProviderError returns a short provider code and safe stage-neutral summary without exposing raw SDK error strings.
// Inputs: one error from the configured provider adapter; outputs: a bounded code and diagnostic phrase.
// Side effects: none. Choose over serializing raw errors, which can contain endpoint or request details.
// Byline: Codex · GPT-6 · 2026-10-04.
func boundedProbeProviderError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		code := boundedProbeToken(apiError.ErrorCode(), 64)
		if code == "" {
			code = "provider_error"
		}
		return code, "provider rejected or failed the conditional-write request"
	}
	return "adapter_error", "configured object-store adapter returned an error"
}

// boundedProbeToken keeps provider codes short, printable and safe for Temporal result history.
// Inputs: arbitrary provider error code and a maximum length; outputs: ASCII alphanumeric/underscore/dash/dot prefix only.
// Side effects: none. Choose before exposing remote error metadata in a workflow result.
// Byline: Codex · GPT-6 · 2026-10-04.
func boundedProbeToken(value string, limit int) string {
	var out strings.Builder
	for _, char := range value {
		if out.Len() >= limit {
			break
		}
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char == '-' || char == '.' {
			out.WriteRune(char)
		}
	}
	return out.String()
}

// readProbeMatches opens the probe key afresh and accepts only exactly sixteen bytes equal to payload A.
// Inputs: context, resolved object store and coordinates, and the synthetic expected payload; outputs: true only for an exact bounded readback.
// Side effects: one remote read limited to payload length plus one byte; it never reads archives or follows a listing.
// Choose after both write attempts to verify the final object state independently.
// Byline: Codex · GPT-6 · 2026-10-04.
func readProbeMatches(ctx context.Context, store smsthreads.ObjectStore, bucket, key string, expected []byte) bool {
	if err := ctx.Err(); err != nil {
		return false
	}
	stream, err := store.Open(ctx, bucket, key)
	if err != nil {
		return false
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, toolkitPackageProbePayloadBytes+1))
	return err == nil && len(data) == toolkitPackageProbePayloadBytes && len(expected) == toolkitPackageProbePayloadBytes && string(data) == string(expected)
}

// compile-time assertion keeps the probe bound to the exact conditional-write seam used by preservation.
var _ toolkitConditionalObjectWriter = (smsthreads.S3Store{})

// compile-time assertion documents that the probe itself does not widen the shared ObjectStore interface.
var _ smsthreads.ObjectStore = (smsthreads.S3Store{})
