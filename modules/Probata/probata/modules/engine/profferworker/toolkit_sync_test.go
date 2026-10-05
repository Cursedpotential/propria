// Byline: Codex · GPT-6.1 · 2026-10-05. Worker admission/registry fixtures only; no live runtime or source writes.
package profferworker

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

type syncArtifactsFixture struct{}

func (*syncArtifactsFixture) Put(context.Context, string, []byte, string) (libraryvalidation.ArtifactRef, error) {
	panic("unexpected artifact write")
}
func (*syncArtifactsFixture) Read(context.Context, libraryvalidation.ArtifactRef, int64) ([]byte, error) {
	panic("unexpected artifact read")
}

type syncExtractorFixture struct{}

func (*syncExtractorFixture) Extract(context.Context, libraryvalidation.Snapshot) (libraryvalidation.Extracted, error) {
	panic("unexpected Python/extraction call")
}

func syncValidatorFixture() *activities.ToolkitLibraryValidationActivities {
	return &activities.ToolkitLibraryValidationActivities{Service: &libraryvalidation.Service{Artifacts: &syncArtifactsFixture{}, Extractor: &syncExtractorFixture{}, SigningKey: []byte(strings.Repeat("synthetic-key", 4))}}
}

// TestToolkitSyncOptionalAdmissionReusesValidator verifies URL opt-in and exact existing dependency identity without I/O.
// Inputs: disabled/configured origins and injected factory. Outputs: call and identity assertions. Effects: counters only.
// Choose instead of running the worker or constructing another validator runtime.
func TestToolkitSyncOptionalAdmissionReusesValidator(t *testing.T) {
	v := syncValidatorFixture()
	calls := 0
	service := &librarysync.Service{}
	factory := func(a libraryvalidation.Artifacts, e libraryvalidation.Extractor, key []byte) (*librarysync.Service, error) {
		calls++
		if a != v.Service.Artifacts || e != v.Service.Extractor || len(key) == 0 || &key[0] != &v.Service.SigningKey[0] {
			t.Fatal("validator dependencies replaced or second runtime constructed")
		}
		return service, nil
	}
	group, err := configureToolkitSyncWithFactory(context.Background(), "  ", nil, factory)
	if err != nil || group != nil || calls != 0 {
		t.Fatal("disabled sync invoked constructor")
	}
	group, err = configureToolkitSyncWithFactory(context.Background(), " https://synthetic-familycourt.example/ ", v, factory)
	if err != nil || group == nil || group.Service != service || calls != 1 {
		t.Fatalf("configured sync admission failed: %v", err)
	}
}

// TestToolkitSyncAdmissionFailsClosed covers missing validator, malformed origins, constructor failures and cancellation.
// Inputs: independent synthetic fixtures. Outputs: failure/no-call/secret-redaction assertions. Effects: memory only.
// Choose to establish startup guards without credentials, Python, B2, Temporal or databases.
func TestToolkitSyncAdmissionFailsClosed(t *testing.T) {
	for _, kind := range []string{"no-validator", "no-service", "no-artifacts", "no-extractor", "short-key", "http", "userinfo", "path", "query", "fragment", "relative", "factory-error", "nil-service", "nil-factory", "canceled", "canceled-in-factory"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			v := syncValidatorFixture()
			origin := "https://synthetic-familycourt.example"
			calls := 0
			factory := func(libraryvalidation.Artifacts, libraryvalidation.Extractor, []byte) (*librarysync.Service, error) {
				calls++
				if kind == "factory-error" {
					return nil, errors.New("DO_NOT_ECHO_CREDENTIAL")
				}
				if kind == "nil-service" {
					return nil, nil
				}
				if kind == "canceled-in-factory" {
					cancel()
				}
				return &librarysync.Service{}, nil
			}
			switch kind {
			case "no-validator":
				v = nil
			case "no-service":
				v.Service = nil
			case "no-artifacts":
				v.Service.Artifacts = nil
			case "no-extractor":
				v.Service.Extractor = nil
			case "short-key":
				v.Service.SigningKey = []byte("short")
			case "http":
				origin = "http://synthetic.example"
			case "userinfo":
				origin = "https://user:DO_NOT_ECHO_CREDENTIAL@synthetic.example"
			case "path":
				origin += "/api"
			case "query":
				origin += "?token=DO_NOT_ECHO_CREDENTIAL"
			case "fragment":
				origin += "#token"
			case "relative":
				origin = "/api"
			case "nil-factory":
				factory = nil
			case "canceled":
				cancel()
			}
			group, err := configureToolkitSyncWithFactory(ctx, origin, v, factory)
			if err == nil || group != nil || strings.Contains(err.Error(), "DO_NOT_ECHO_CREDENTIAL") {
				t.Fatalf("configured failure admitted or exposed details: %v", err)
			}
			if strings.HasPrefix(kind, "canceled") && !errors.Is(err, context.Canceled) {
				t.Fatal("context cancellation lost")
			}
			if kind != "factory-error" && kind != "nil-service" && kind != "canceled-in-factory" && calls != 0 {
				t.Fatal("invalid config reached constructor")
			}
		})
	}
}

// TestToolkitSyncProductionOptInRejectsMissingMount checks the actual wrapper disables cleanly or returns safe configuration failure.
// Inputs: environment and configured synthetic validator. Outputs: admission assertions. Effects: bounded missing-file check only.
func TestToolkitSyncProductionOptInRejectsMissingMount(t *testing.T) {
	t.Setenv(librarysync.EnvBackendURL, "")
	if group, err := configureToolkitSync(context.Background(), nil); err != nil || group != nil {
		t.Fatal("unset backend URL was not disabled")
	}
	t.Setenv(librarysync.EnvBackendURL, "https://synthetic-familycourt.example")
	t.Setenv(librarysync.EnvTokenFile, "relative-DO_NOT_ECHO_CREDENTIAL")
	if group, err := configureToolkitSync(context.Background(), syncValidatorFixture()); err == nil || group != nil || strings.Contains(err.Error(), "DO_NOT_ECHO_CREDENTIAL") {
		t.Fatal("missing token mount did not fail safely")
	}
}

// TestToolkitSyncRegistryAddsExactlyTwoWorkflowsAndSixteenActivities preserves all validator/catalog registration siblings.
// Inputs: baseline and configured Activity groups. Outputs: exact registry delta/uniqueness assertions. Effects: in-memory registrars only.
// Choose instead of a live worker/Temporal deployment to prove optional registration.
func TestToolkitSyncRegistryAddsExactlyTwoWorkflowsAndSixteenActivities(t *testing.T) {
	v := syncValidatorFixture()
	base := &registrationRecorder{}
	RegisterAll(base, Registrations{ToolkitValidation: v})
	enabled := &registrationRecorder{}
	RegisterAll(enabled, Registrations{ToolkitValidation: v, ToolkitSync: &activities.ToolkitLibrarySyncActivities{Service: &librarysync.Service{}}})
	if enabled.workflowCount != base.workflowCount+2 || len(enabled.names) != len(base.names)+16 {
		t.Fatal("unexpected sync registry delta")
	}
	wanted := []string{librarysync.ListActivity, librarysync.SeenActivity, librarysync.HashSourceActivity, librarysync.RetainActivity, librarysync.ExtractActivity, librarysync.HydrateActivity, librarysync.ObserveActivity, librarysync.ClaimActivity, librarysync.PrepareActivity, librarysync.WriteActivity, librarysync.RefreshActivity, librarysync.HistoryActivity, librarysync.HashVersionActivity, librarysync.CurrentActivity, librarysync.AckActivity, librarysync.FailureActivity}
	counts := map[string]int{}
	for _, name := range enabled.names {
		counts[name]++
	}
	for _, name := range wanted {
		if counts[name] != 1 {
			t.Fatalf("sync activity %s count %d", name, counts[name])
		}
	}
	var rest []string
	for _, name := range enabled.names {
		if !strings.HasPrefix(name, "toolkit_library_sync_") {
			rest = append(rest, name)
		}
	}
	prior := append([]string(nil), base.names...)
	sort.Strings(rest)
	sort.Strings(prior)
	if !reflect.DeepEqual(rest, prior) {
		t.Fatal("existing Activities changed")
	}
	var workflows []string
	workflowCounts := map[string]int{}
	for _, name := range enabled.workflowNames {
		if name == librarysync.CycleWorkflowName || name == librarysync.WriteWorkflowName {
			workflowCounts[name]++
		} else {
			workflows = append(workflows, name)
		}
	}
	if workflowCounts[librarysync.CycleWorkflowName] != 1 || workflowCounts[librarysync.WriteWorkflowName] != 1 || !reflect.DeepEqual(workflows, base.workflowNames) {
		t.Fatal("sync workflows duplicated or prior workflows changed")
	}
}

// TestToolkitSyncRegistryRejectsUnwiredValidator proves direct registration cannot bypass optional admission dependency gates.
// Inputs: invalid Activity groups. Outputs: safe panic assertions. Effects: in-memory registration only.
func TestToolkitSyncRegistryRejectsUnwiredValidator(t *testing.T) {
	for _, in := range []Registrations{{ToolkitSync: &activities.ToolkitLibrarySyncActivities{Service: &librarysync.Service{}}}, {ToolkitValidation: &activities.ToolkitLibraryValidationActivities{}, ToolkitSync: &activities.ToolkitLibrarySyncActivities{Service: &librarysync.Service{}}}, {ToolkitValidation: syncValidatorFixture(), ToolkitSync: &activities.ToolkitLibrarySyncActivities{}}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("unwired sync registered")
				}
			}()
			RegisterAll(&registrationRecorder{}, in)
		}()
	}
}
