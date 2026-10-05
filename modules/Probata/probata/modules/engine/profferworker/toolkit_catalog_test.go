// Byline: Codex · GPT-6 · 2026-10-04. Configuration/registration fixtures only; no source or database writes.
package profferworker

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

// workerCatalogFixture implements only repository lifecycle and rejects unexpected metadata operations.
// Inputs: configuration tests. Outputs: explicit operation errors. Effects: close counter only; choose without a database.
type workerCatalogFixture struct{ closes int }

// RegisterToolkitRecovery rejects accidental writes during worker setup or registration tests.
// Inputs: context/batch. Outputs: error. Effects: none; choose as a constructor write tripwire.
func (*workerCatalogFixture) RegisterToolkitRecovery(context.Context, activities.ToolkitRecoveryCatalogBatch) error {
	return errors.New("unexpected catalog write")
}

// ReadToolkitRecovery rejects unrequested record reads during worker setup.
// Inputs: context/operation. Outputs: empty/error. Effects: none; choose to limit setup to admission only.
func (*workerCatalogFixture) ReadToolkitRecovery(context.Context, string) (activities.ToolkitRecoveryCatalogBatch, error) {
	return activities.ToolkitRecoveryCatalogBatch{}, errors.New("unexpected catalog read")
}

// Close counts separate repository shutdown without closing other worker adapters.
// Inputs/outputs: none. Effects: count only; choose to verify exactly-once cleanup.
func (r *workerCatalogFixture) Close() { r.closes++ }

// workerCatalogPreservation reuses the test package directory as the explicit root and a counting resolver.
// Inputs: test handle and call counter. Outputs: existing-shaped preservation group. Effects: directory metadata read only.
// Choose instead of creating or removing test directories for configuration tests.
func workerCatalogPreservation(t *testing.T, calls *int) activities.ToolkitPackagePreservationActivities {
	t.Helper()
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	return activities.ToolkitPackagePreservationActivities{AllowedRoot: root, Stores: func(string) (smsthreads.ObjectStore, error) {
		*calls++
		return nil, errors.New("synthetic store resolver")
	}}
}

// TestToolkitCatalogWorkerOptionalAndReuse proves opt-in admission, adapter reuse and shutdown ownership.
// Inputs: injected opener/existing adapters. Outputs: configuration assertions. Effects: process-memory counters only.
// Choose to prove unset configuration leaves the current worker untouched.
func TestToolkitCatalogWorkerOptionalAndReuse(t *testing.T) {
	calls, opens := 0, 0
	preservation := workerCatalogPreservation(t, &calls)
	repo := &workerCatalogFixture{}
	path := filepath.Join(preservation.AllowedRoot, "synthetic-writer.url")
	opener := func(_ context.Context, file string) (toolkitCatalogRepository, error) {
		opens++
		if file != path {
			t.Fatal("wrong explicit config path")
		}
		return repo, nil
	}
	group, close, err := configureToolkitCatalogWithOpener(context.Background(), "  ", activities.ToolkitPackagePreservationActivities{}, opener)
	if err != nil || group != nil || close != nil || opens != 0 {
		t.Fatal("disabled config opened a repository")
	}
	group, close, err = configureToolkitCatalogWithOpener(context.Background(), " "+path+" ", preservation, opener)
	if err != nil || group == nil || close == nil || opens != 1 || calls != 0 {
		t.Fatalf("enabled admission: %v", err)
	}
	if group.AllowedRoot != preservation.AllowedRoot || group.Catalog != repo || group.Heartbeat == nil {
		t.Fatal("existing root/repository not reused")
	}
	_, _ = group.Stores("b2")
	if calls != 1 {
		t.Fatal("existing resolver not reused")
	}
	close()
	close()
	if repo.closes != 1 {
		t.Fatal("shutdown close was not once-only")
	}
}

// TestToolkitCatalogWorkerRejectsConfiguredFailures exercises visible failures without fallback or remote access.
// Inputs: synthetic config failures. Outputs: rejection/admission-count assertions. Effects: memory only.
// Choose for config/root/cancellation guard coverage before production startup.
func TestToolkitCatalogWorkerRejectsConfiguredFailures(t *testing.T) {
	for _, kind := range []string{"relative-file", "missing-root", "relative-root", "missing-store", "open-error", "nil-repository", "canceled", "canceled-after-open"} {
		t.Run(kind, func(t *testing.T) {
			calls, opens := 0, 0
			preservation := workerCatalogPreservation(t, &calls)
			path := filepath.Join(preservation.AllowedRoot, "synthetic-writer.url")
			repo := &workerCatalogFixture{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opener := func(context.Context, string) (toolkitCatalogRepository, error) {
				opens++
				if kind == "open-error" {
					return nil, errors.New("DO_NOT_ECHO_CREDENTIAL")
				}
				if kind == "nil-repository" {
					return nil, nil
				}
				if kind == "canceled-after-open" {
					cancel()
				}
				return repo, nil
			}
			switch kind {
			case "relative-file":
				path = "relative.url"
			case "missing-root":
				preservation.AllowedRoot = filepath.Join(preservation.AllowedRoot, "nonexistent-synthetic-root")
			case "relative-root":
				preservation.AllowedRoot = "."
			case "missing-store":
				preservation.Stores = nil
			case "canceled":
				cancel()
			}
			group, close, err := configureToolkitCatalogWithOpener(ctx, path, preservation, opener)
			if err == nil || group != nil || close != nil || calls != 0 || strings.Contains(err.Error(), "DO_NOT_ECHO_CREDENTIAL") {
				t.Fatal("configured failure did not fail visibly/safely")
			}
			if strings.HasPrefix(kind, "canceled") && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost")
			}
			if kind == "canceled-after-open" && repo.closes != 1 {
				t.Fatal("admitted connection leaked on cancellation")
			}
			if kind != "open-error" && kind != "nil-repository" && kind != "canceled-after-open" && opens != 0 {
				t.Fatal("invalid config reached opener")
			}
		})
	}
}

// TestToolkitCatalogWorkerRegistration adds exactly one workflow and three Activities only when enabled.
// Inputs: baseline and configured registrations. Outputs: registry-delta assertions. Effects: in-memory registrars only.
// Choose over a full worker run so tests require no Temporal server, DB, credentials or deployments.
func TestToolkitCatalogWorkerRegistration(t *testing.T) {
	base := &registrationRecorder{}
	RegisterAll(base, Registrations{})
	calls := 0
	preservation := workerCatalogPreservation(t, &calls)
	repo := &workerCatalogFixture{}
	group, close, err := configureToolkitCatalogWithOpener(context.Background(), filepath.Join(preservation.AllowedRoot, "synthetic-writer.url"), preservation, func(context.Context, string) (toolkitCatalogRepository, error) { return repo, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	enabled := &registrationRecorder{}
	RegisterAll(enabled, Registrations{ToolkitCatalog: group})
	if enabled.workflowCount != base.workflowCount+1 || len(enabled.names) != len(base.names)+3 {
		t.Fatal("unexpected worker registry delta")
	}
	extra := []string{activities.ToolkitCatalogMetadataActivityName, activities.ToolkitCatalogRegisterActivityName, activities.ToolkitCatalogReadbackActivityName}
	for _, name := range extra {
		count := 0
		for _, got := range enabled.names {
			if got == name {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("catalog Activity %s registered %d times", name, count)
		}
	}
	var remaining []string
	for _, name := range enabled.names {
		isExtra := false
		for _, added := range extra {
			if name == added {
				isExtra = true
			}
		}
		if !isExtra {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	baselineNames := append([]string(nil), base.names...)
	sort.Strings(baselineNames)
	if !reflect.DeepEqual(remaining, baselineNames) {
		t.Fatal("existing Activity registration changed")
	}
	var workflows []string
	found := 0
	for _, name := range enabled.workflowNames {
		if name == activities.ToolkitCatalogRegistrationWorkflowName {
			found++
		} else {
			workflows = append(workflows, name)
		}
	}
	if found != 1 || !reflect.DeepEqual(workflows, base.workflowNames) {
		t.Fatal("existing workflow registration changed")
	}
	if calls != 0 {
		t.Fatal("registration resolved B2 or read sources")
	}
}
