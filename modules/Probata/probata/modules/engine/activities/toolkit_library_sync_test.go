// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"context"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type syncFixtureRegistrar struct{ names map[string]any }

func (r *syncFixtureRegistrar) RegisterActivityWithOptions(fn any, options activity.RegisterOptions) {
	if _, ok := r.names[options.Name]; ok {
		panic("duplicate sync activity")
	}
	r.names[options.Name] = fn
}

func TestToolkitLibrarySyncRegistrationAndUnconfiguredActivityFailVisible(t *testing.T) {
	r := &syncFixtureRegistrar{names: map[string]any{}}
	RegisterToolkitLibrarySyncActivities(r, ToolkitLibrarySyncActivities{})
	if len(r.names) != 16 {
		t.Fatalf("registered %d units", len(r.names))
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	for name, fn := range r.names {
		env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
	if _, err := env.ExecuteActivity(librarysync.ListActivity, librarysync.ListInput{Child: "case-law"}); err == nil {
		t.Fatal("unconfigured listing silently succeeded")
	}
	a := ToolkitLibrarySyncActivities{}
	if _, err := a.Hydrate(context.Background(), librarysync.Handle{}); err == nil {
		t.Fatal("unconfigured hydration silently succeeded")
	}
	if _, err := a.Write(context.Background(), librarysync.WriteRequest{}); err == nil {
		t.Fatal("unconfigured writer silently succeeded")
	}
	if _, err := a.Acknowledge(context.Background(), librarysync.AckInput{}); err == nil {
		t.Fatal("unconfigured ack silently succeeded")
	}
}
