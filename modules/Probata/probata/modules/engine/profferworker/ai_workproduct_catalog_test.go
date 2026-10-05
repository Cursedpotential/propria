// Byline: Codex · GPT-6 · 2026-10-05.
package profferworker

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

// aiCatalogAdmissionFake isolates admission and verifies reuse of the existing writer without database effects.
// Inputs: admission outcome. Outputs: call count and outcome. Effects: in-memory counter only; choose for worker configuration tests.
type aiCatalogAdmissionFake struct {
	activities.ToolkitRecoveryCatalogRepository
	activities.AIWorkproductOccurrenceCatalog
	calls int
	err   error
}

// AdmitAIWorkproductCatalog records one bounded fake admission attempt.
// Inputs: context. Outputs: configured error. Effects: counter increment only; choose over a live database in registry tests.
func (f *aiCatalogAdmissionFake) AdmitAIWorkproductCatalog(context.Context) error {
	f.calls++
	return f.err
}

// TestAIWorkproductCatalogConfiguration proves optional disablement, adapter rejection and explicit narrow admission.
// Inputs: synthetic worker seams. Outputs: configuration assertions. Effects: no connections or source writes.
// Choose to verify worker lifecycle reuse; repository SQL behavior belongs to isolated catalog tests.
func TestAIWorkproductCatalogConfiguration(t *testing.T) {
	group, err := configureAIWorkproductCatalog(context.Background(), nil, nil)
	if err != nil || group != nil {
		t.Fatalf("disabled: %v, %v", group, err)
	}
	repo := &aiCatalogAdmissionFake{}
	toolkit := &activities.ToolkitCatalogRegistrationActivities{Catalog: repo}
	if _, err = configureAIWorkproductCatalog(context.Background(), toolkit, nil); err == nil || repo.calls != 0 {
		t.Fatal("missing adapters must reject before admission")
	}
	root, err := filepath.Abs("synthetic-ai-catalog-derive-scratch")
	if err != nil {
		t.Fatal(err)
	}
	placement := activities.NewAIWorkproductPlacementActivities(root, func(string) (smsthreads.ObjectStore, error) { return nil, errors.New("must not open storage") })
	repo.err = errors.New("missing SQL grant")
	if _, err = configureAIWorkproductCatalog(context.Background(), toolkit, placement); err == nil || repo.calls != 1 {
		t.Fatal("failed admission accepted")
	}
	repo.err = nil
	group, err = configureAIWorkproductCatalog(context.Background(), toolkit, placement)
	if err != nil || group == nil || group.Catalog != repo || group.AllowedRoot != placement.AllowedRoot || repo.calls != 2 {
		t.Fatalf("writer/root reuse failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = configureAIWorkproductCatalog(ctx, toolkit, placement); err == nil || repo.calls != 2 {
		t.Fatal("cancelled configuration attempted admission")
	}
}

// TestAIWorkproductCatalogRegistry proves the optional metadata group adds only its own workflow and three unique Activities.
// Inputs: disabled/enabled registries. Outputs: exact registration assertions. Effects: in-memory registration only.
// Choose to detect accidental duplicate registration and disabled-group leakage.
func TestAIWorkproductCatalogRegistry(t *testing.T) {
	base := &registrationRecorder{}
	RegisterAll(base, Registrations{})
	enabled := &registrationRecorder{}
	RegisterAll(enabled, Registrations{AIWorkproductCatalog: activities.NewAIWorkproductCatalogActivities("/data/proffer/derive-scratch", nil, nil)})
	if enabled.workflowCount != base.workflowCount+1 || len(enabled.names) != len(base.names)+3 {
		t.Fatal("unexpected catalog registry delta")
	}
	for _, want := range []string{activities.AIWorkproductCatalogAuthenticateActivityName, activities.AIWorkproductCatalogRegisterActivityName, activities.AIWorkproductCatalogReadbackActivityName} {
		count := 0
		for _, name := range enabled.names {
			if name == want {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%s registered %d times", want, count)
		}
		for _, name := range base.names {
			if name == want {
				t.Fatalf("disabled catalog registered %s", want)
			}
		}
	}
	count := 0
	for _, name := range enabled.workflowNames {
		if name == activities.AIWorkproductCatalogWorkflowName {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("catalog workflows %d", count)
	}
}
