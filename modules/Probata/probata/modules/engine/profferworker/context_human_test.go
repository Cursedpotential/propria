package profferworker

import (
	"context"
	"testing"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"go.temporal.io/sdk/activity"
)

// humanRegistryRecorder captures actual typed native registrations beside the existing complete registry.
// Inputs: registration calls. Outputs: names and executable functions. Effects: in-memory collection only.
// Choose to verify native source processing is reachable without replacing historical registrations.
type humanRegistryRecorder struct {
	registrationRecorder
	bodies map[string]interface{}
}

// RegisterActivityWithOptions records each executable body and preserves the existing name recorder.
// Inputs: typed function and SDK options. Outputs: captured entry. Effects: in-memory map only.
// Choose for tests that must distinguish registration from live Activity execution.
func (r *humanRegistryRecorder) RegisterActivityWithOptions(fn interface{}, options activity.RegisterOptions) {
	r.registrationRecorder.RegisterActivityWithOptions(fn, options)
	if r.bodies == nil {
		r.bodies = make(map[string]interface{})
	}
	r.bodies[options.Name] = fn
}

// TestNativeHumanRegistryPreservesLegacyAndExposesAtomicUnits verifies the complete worker's additive native seam.
// Inputs: disabled/enabled in-memory registries. Outputs: exact four-Activity delta and callable typed bodies.
// Effects: no source, provider, database or Temporal calls; choose before enabling native SMS processing.
func TestNativeHumanRegistryPreservesLegacyAndExposesAtomicUnits(t *testing.T) {
	baseline := &registrationRecorder{}
	RegisterAll(baseline, Registrations{})
	native := &humanRegistryRecorder{}
	RegisterAll(native, Registrations{HumanContext: &humancontext.Activities{}})
	if native.workflowCount != baseline.workflowCount || len(native.names) != len(baseline.names)+4 {
		t.Fatal("native registry changed legacy workflow ownership or omitted atomic units")
	}
	counts := make(map[string]int)
	for _, name := range native.names {
		counts[name]++
	}
	for _, name := range baseline.names {
		if counts[name] != 1 {
			t.Fatalf("legacy registration %s changed", name)
		}
	}
	for _, name := range []string{humancontext.FrameActivity, humancontext.ExtractActivity, humancontext.IndexActivity, humancontext.VerifyActivity} {
		if counts[name] != 1 {
			t.Fatalf("native activity %s registered %d times", name, counts[name])
		}
		fn, ok := native.bodies[name].(func(context.Context, humancontext.Request) (humancontext.Result, error))
		if !ok {
			t.Fatalf("native activity %s lost its typed request", name)
		}
		_, err := fn(context.Background(), humancontext.Request{})
		if err == nil || (err.Error() != "human bundles unavailable" && err.Error() != "human indexer unavailable") {
			t.Fatalf("native %s entered unexpected admission/body: %v", name, err)
		}
	}
}
