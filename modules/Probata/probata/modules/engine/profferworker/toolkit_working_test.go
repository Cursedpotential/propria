// Byline: Codex · GPT-6 · 2026-10-05. Optional setup/lifecycle/registration tests; no external services or filesystem mutations.
package profferworker

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// workerWorkingFixture rejects catalog execution during configuration and registration tests.
// Inputs: setup operations. Outputs: read/write tripwire errors and close counter.
// Effects: memory only; choose instead of a live database.
type workerWorkingFixture struct{ closes int }

// RegisterToolkitWorking refuses accidental writes during worker configuration.
// Inputs: context/batch. Outputs: error. Effects: none; choose as a constructor write tripwire.
func (*workerWorkingFixture) RegisterToolkitWorking(context.Context, activities.ToolkitWorkingCatalogBatch) error {
	return errors.New("unexpected working catalog write")
}

// ReadToolkitWorking refuses unrequested row reads during worker configuration.
// Inputs: context/operation. Outputs: empty batch/error. Effects: none; choose as a setup read tripwire.
func (*workerWorkingFixture) ReadToolkitWorking(context.Context, string) (activities.ToolkitWorkingCatalogBatch, error) {
	return activities.ToolkitWorkingCatalogBatch{}, errors.New("unexpected working catalog read")
}

// Close counts shutdown of only this dedicated fixture repository.
// Inputs/outputs: none. Effects: close counter; choose to verify once-only lifecycle ownership.
func (r *workerWorkingFixture) Close() { r.closes++ }

// workingWorkerRoot supplies the existing test source directory without creating or deleting fixtures.
// Inputs: test handle. Outputs: absolute directory. Effects: path lookup only; choose for mounted-root validation.
func workingWorkerRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestToolkitWorkingWorkerOptionalAndLifecycle proves optional setup, dedicated opener and concurrent once-only cleanup.
// Inputs: existing directory and injected repository. Outputs: setup/lifecycle assertions. Effects: memory only.
// Choose before parent integration without accessing any source content, credentials or database.
func TestToolkitWorkingWorkerOptionalAndLifecycle(t *testing.T) {
	opens := 0
	repo := &workerWorkingFixture{}
	root := workingWorkerRoot(t)
	path := filepath.Join(root, "synthetic-working.url")
	opener := func(_ context.Context, file string) (toolkitWorkingRepository, error) {
		opens++
		if file != path {
			t.Fatal("wrong dedicated config path")
		}
		return repo, nil
	}
	group, close, err := configureToolkitWorkingWithOpener(context.Background(), " ", "", opener)
	if err != nil || group != nil || close != nil || opens != 0 {
		t.Fatal("disabled group opened connection")
	}
	group, close, err = configureToolkitWorkingWithOpener(context.Background(), " "+path+" ", " "+root+" ", opener)
	if err != nil || group == nil || close == nil || opens != 1 || group.AllowedRoot != root || group.Catalog != repo || group.Heartbeat == nil {
		t.Fatal("enabled setup", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); close() }()
	}
	wg.Wait()
	if repo.closes != 1 {
		t.Fatal("dedicated cleanup not once-only")
	}
}

// TestToolkitWorkingWorkerRejectsConfiguredFailures verifies guards, sanitized errors and connection cleanup after failed admission.
// Inputs: independent failure fixtures. Outputs: no group/registry or leaked repository. Effects: memory only.
// Choose to prove a configured failure never falls back to recovery/platform credentials.
func TestToolkitWorkingWorkerRejectsConfiguredFailures(t *testing.T) {
	for _, kind := range []string{"relative-file", "relative-root", "missing-root", "root-is-file", "nil-opener", "nil-repository", "open-error", "open-error-with-repository", "canceled", "canceled-after-open"} {
		t.Run(kind, func(t *testing.T) {
			root := workingWorkerRoot(t)
			path := filepath.Join(root, "synthetic-working.url")
			repo := &workerWorkingFixture{}
			opens := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opener := toolkitWorkingOpener(func(context.Context, string) (toolkitWorkingRepository, error) {
				opens++
				switch kind {
				case "nil-repository":
					return nil, nil
				case "open-error":
					return nil, errors.New("DO_NOT_ECHO_SECRET")
				case "open-error-with-repository":
					return repo, errors.New("DO_NOT_ECHO_SECRET")
				case "canceled-after-open":
					cancel()
				}
				return repo, nil
			})
			switch kind {
			case "relative-file":
				path = "relative.url"
			case "relative-root":
				root = "."
			case "missing-root":
				root = filepath.Join(root, "missing-working-root")
			case "root-is-file":
				root = filepath.Join(root, "toolkit_working.go")
			case "nil-opener":
				opener = nil
			case "canceled":
				cancel()
			}
			group, close, err := configureToolkitWorkingWithOpener(ctx, path, root, opener)
			if err == nil || group != nil || close != nil || strings.Contains(err.Error(), "DO_NOT_ECHO_SECRET") {
				t.Fatal("unsafe configured failure")
			}
			if strings.HasPrefix(kind, "canceled") && !errors.Is(err, context.Canceled) {
				t.Fatal("lost cancellation")
			}
			if (kind == "open-error-with-repository" || kind == "canceled-after-open") && repo.closes != 1 {
				t.Fatal("connection leaked")
			}
			if kind != "nil-repository" && kind != "open-error" && kind != "open-error-with-repository" && kind != "canceled-after-open" && opens != 0 {
				t.Fatal("invalid config opened connection")
			}
		})
	}
}

// workingRegistryFixture captures exact names and executable function types without running any workflow or Activity.
// Inputs: registry calls. Outputs: bounded registration observations. Effects: memory only; choose without a Temporal server.
type workingRegistryFixture struct {
	names     []string
	functions []any
}

// RegisterWorkflowWithOptions records a named workflow function.
// Inputs: function/options. Outputs: none. Effects: captured registry; choose to prove one existing-queue workflow registration.
func (r *workingRegistryFixture) RegisterWorkflowWithOptions(fn interface{}, opts workflow.RegisterOptions) {
	r.names = append(r.names, opts.Name)
	r.functions = append(r.functions, fn)
}

// RegisterActivityWithOptions records a separately named Activity function.
// Inputs: function/options. Outputs: none. Effects: captured registry; choose to prove registration/readback remain separate.
func (r *workingRegistryFixture) RegisterActivityWithOptions(fn interface{}, opts activity.RegisterOptions) {
	r.names = append(r.names, opts.Name)
	r.functions = append(r.functions, fn)
}

// TestToolkitWorkingWorkerRegistration proves disabled no-op, three exact executable registrations and pre-mutation rejection.
// Inputs: optional or invalid groups and recording registrar. Outputs: count/name/type assertions. Effects: memory only.
// Choose for parent-call integration contract; does not claim worker.go wiring or deployment.
func TestToolkitWorkingWorkerRegistration(t *testing.T) {
	registrar := &workingRegistryFixture{}
	if err := RegisterToolkitWorkingCatalog(registrar, nil); err != nil || len(registrar.names) != 0 {
		t.Fatal("disabled registry mutation")
	}
	group := activities.NewToolkitWorkingCatalogActivities(workingWorkerRoot(t), &workerWorkingFixture{})
	if err := RegisterToolkitWorkingCatalog(registrar, &group); err != nil {
		t.Fatal(err)
	}
	expected := []string{activities.ToolkitWorkingCatalogWorkflowName, activities.ToolkitWorkingCatalogRegisterActivityName, activities.ToolkitWorkingCatalogReadbackActivityName}
	if !reflect.DeepEqual(registrar.names, expected) {
		t.Fatal("unexpected registry names", registrar.names)
	}
	wantFns := []any{activities.ToolkitWorkingCatalogWorkflow, group.RegisterToolkitWorkingCatalog, group.ReadbackToolkitWorkingCatalog}
	for i, want := range wantFns {
		if reflect.TypeOf(registrar.functions[i]) != reflect.TypeOf(want) || reflect.ValueOf(registrar.functions[i]).Pointer() != reflect.ValueOf(want).Pointer() {
			t.Fatal("wrong registry function", i)
		}
	}
	for _, kind := range []string{"missing-repository", "relative-root", "missing-heartbeat", "nil-registrar"} {
		t.Run(kind, func(t *testing.T) {
			g := group
			r := &workingRegistryFixture{}
			var sink ToolkitWorkingRegistrar = r
			switch kind {
			case "missing-repository":
				g.Catalog = nil
			case "relative-root":
				g.AllowedRoot = "."
			case "missing-heartbeat":
				g.Heartbeat = nil
			case "nil-registrar":
				sink = nil
			}
			if err := RegisterToolkitWorkingCatalog(sink, &g); err == nil || len(r.names) > 0 {
				t.Fatal("invalid group changed registry")
			}
		})
	}
}
