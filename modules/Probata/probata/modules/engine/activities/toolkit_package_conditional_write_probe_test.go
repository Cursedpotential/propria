// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/aws/smithy-go"
)

// toolkitProviderProbeFixture models provider conditional-write behavior using memory only.
// Inputs: overwrite/reject mode and optional seeded objects; outputs: ObjectStore plus conditional-operation counters.
// Side effects: process-memory mutations only. Choose for behavior tests without network, secrets, or retained real objects.
// Byline: Codex · GPT-6 · 2026-10-04.
type toolkitProviderProbeFixture struct {
	objects           map[string][]byte
	ignoreCondition   bool
	rejectFirst       bool
	conditionalWrites int
	plainWrites       int
}

// Open returns an independent bounded-test reader for one exact fake object.
// Inputs: context, bucket and key; outputs: reader or missing-key error.
// Side effects: none. Choose to exercise the probe's independent fresh-read path.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitProviderProbeFixture) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	data, ok := s.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("fake object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Put is a tripwire that proves the probe never falls back to overwrite-capable storage.
// Inputs: ordinary object-store write arguments; outputs: explicit failure.
// Side effects: increments a fixture counter only. Choose to detect an unsafe unconditional fallback.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitProviderProbeFixture) Put(_ context.Context, _, _ string, _ io.ReadSeeker, _ int64, _ string) error {
	s.plainWrites++
	return errors.New("unconditional Put must not be called by the provider probe")
}

// Exists reports one exact fake key without performing writes.
// Inputs: context, bucket and key; outputs: key presence.
// Side effects: none. Choose to test the required preflight no-put-on-existing behavior.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitProviderProbeFixture) Exists(_ context.Context, bucket, key string) (bool, error) {
	_, exists := s.objects[bucket+"/"+key]
	return exists, nil
}

// PutIfAbsent models strict, ignored, or rejected conditional-write provider behavior.
// Inputs: exact key/body/size; outputs: create, overwrite, or provider API error according to fixture mode.
// Side effects: mutates only this process-local fixture map. Choose to model the three expected provider outcomes.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitProviderProbeFixture) PutIfAbsent(_ context.Context, bucket, key string, body io.ReadSeeker, size int64, _ string) error {
	s.conditionalWrites++
	if s.rejectFirst && s.conditionalWrites == 1 {
		return &smithy.GenericAPIError{Code: "NotImplemented", Message: "conditional writes unsupported", Fault: smithy.FaultClient}
	}
	objectKey := bucket + "/" + key
	if _, exists := s.objects[objectKey]; exists && !s.ignoreCondition {
		return smsthreads.ErrConditionalObjectMismatch
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if int64(len(data)) != size {
		return errors.New("fake body size mismatch")
	}
	s.objects[objectKey] = data
	return nil
}

// TestToolkitPackageConditionalWriteProbeProviderOutcomes distinguishes enforcement, ignored headers, and rejection.
// Inputs: three in-memory provider behaviors; outputs: support verdict, safe diagnostics and final synthetic bytes.
// Side effects: mutates only in-memory fixture maps. Choose to establish deterministic probe classification before a live run.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackageConditionalWriteProbeProviderOutcomes(t *testing.T) {
	payloadA := []byte("probe-payload-A1")
	payloadB := []byte("probe-payload-B1")
	if len(payloadA) != toolkitPackageProbePayloadBytes || len(payloadB) != toolkitPackageProbePayloadBytes {
		t.Fatal("fixture payloads must remain exactly sixteen bytes")
	}
	for _, test := range []struct {
		name             string
		fixture          *toolkitProviderProbeFixture
		wantSupported    bool
		wantOutcome      string
		wantFinal        []byte
		wantProviderCode string
	}{
		{name: "enforced", fixture: &toolkitProviderProbeFixture{objects: map[string][]byte{}}, wantSupported: true, wantOutcome: "conditional_create_verified", wantFinal: payloadA, wantProviderCode: "conditional_object_mismatch"},
		{name: "ignored", fixture: &toolkitProviderProbeFixture{objects: map[string][]byte{}, ignoreCondition: true}, wantOutcome: "conditional_create_not_proven", wantFinal: payloadB, wantProviderCode: "final_readback_mismatch"},
		{name: "rejected", fixture: &toolkitProviderProbeFixture{objects: map[string][]byte{}, rejectFirst: true}, wantOutcome: "first_put_rejected", wantProviderCode: "NotImplemented"},
	} {
		t.Run(test.name, func(t *testing.T) {
			group := ToolkitPackagePreservationActivities{Stores: func(scheme string) (smsthreads.ObjectStore, error) {
				if scheme != "b2" {
					t.Fatalf("resolver scheme = %q, want b2", scheme)
				}
				return test.fixture, nil
			}}
			probe := NewToolkitPackageConditionalWriteProbeActivities(group)
			result, err := probe.RunToolkitPackageConditionalWriteProbe(context.Background(), ToolkitPackageConditionalWriteProbeInput{ProbeNamespace: "probe-" + test.name + "-20261004"})
			if err != nil {
				t.Fatal(err)
			}
			if result.Supported != test.wantSupported || result.Outcome != test.wantOutcome || result.ProviderErrorCode != test.wantProviderCode {
				t.Fatalf("result = %+v, want supported=%t outcome=%q provider_code=%q", result, test.wantSupported, test.wantOutcome, test.wantProviderCode)
			}
			if result.ProbeRef == "" || !strings.HasPrefix(string(result.ProbeRef), toolkitPackageProbeRoot) || !result.ProbeMayExist {
				t.Fatalf("result did not expose the synthetic retained-key boundary: %+v", result)
			}
			if test.wantFinal != nil {
				bucket, key, parseErr := toolkitObjectCoordinates(result.ProbeRef)
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				if !bytes.Equal(test.fixture.objects[bucket+"/"+key], test.wantFinal) {
					t.Fatalf("retained probe bytes = %q, want %q", test.fixture.objects[bucket+"/"+key], test.wantFinal)
				}
			}
			if test.fixture.plainWrites != 0 {
				t.Fatalf("unconditional Put fallback called %d times", test.fixture.plainWrites)
			}
		})
	}
}

// TestToolkitPackageConditionalWriteProbeRefusesOccupiedNamespace proves a pre-existing probe key is never written.
// Inputs: one occupied synthetic key; outputs: visible refusal and unchanged fixture bytes.
// Side effects: none beyond an in-memory existing-object fixture. Choose before conducting a probe to guard namespace reuse.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackageConditionalWriteProbeRefusesOccupiedNamespace(t *testing.T) {
	const namespace = "probe-existing-20261004"
	ref := toolkitPackageProbeRef(namespace)
	bucket, key, err := toolkitObjectCoordinates(ref)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte("do-not-touch-this")
	fixture := &toolkitProviderProbeFixture{objects: map[string][]byte{bucket + "/" + key: append([]byte(nil), original...)}}
	group := ToolkitPackagePreservationActivities{Stores: func(string) (smsthreads.ObjectStore, error) { return fixture, nil }}
	result, err := NewToolkitPackageConditionalWriteProbeActivities(group).RunToolkitPackageConditionalWriteProbe(context.Background(), ToolkitPackageConditionalWriteProbeInput{ProbeNamespace: namespace})
	if err != nil {
		t.Fatal(err)
	}
	if result.Supported || result.Outcome != "probe_key_already_exists" || fixture.conditionalWrites != 0 || fixture.plainWrites != 0 {
		t.Fatalf("occupied key was not refused safely: result=%+v conditional=%d plain=%d", result, fixture.conditionalWrites, fixture.plainWrites)
	}
	if !bytes.Equal(fixture.objects[bucket+"/"+key], original) {
		t.Fatal("pre-existing probe key changed")
	}
}

// TestSafeToolkitProbeNamespaceRejectsPathSyntax enforces the fixed probe-prefix boundary.
// Inputs: valid and hostile candidate namespaces; outputs: validation assertions.
// Side effects: none. Choose before creating the probe reference.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestSafeToolkitProbeNamespaceRejectsPathSyntax(t *testing.T) {
	for _, candidate := range []string{"probe-safe-20261004", strings.Repeat("x", toolkitPackageProbeMaxNamespaceBytes)} {
		if !safeToolkitProbeNamespace(candidate) {
			t.Errorf("valid probe namespace rejected: %q", candidate)
		}
	}
	for _, candidate := range []string{"short", "../escape-probe-20261004", "probe/path/20261004", "probe space 20261004", strings.Repeat("x", toolkitPackageProbeMaxNamespaceBytes+1)} {
		if safeToolkitProbeNamespace(candidate) {
			t.Errorf("unsafe probe namespace accepted: %q", candidate)
		}
	}
}
