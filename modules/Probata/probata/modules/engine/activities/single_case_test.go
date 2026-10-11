// Byline: Codex · GPT-5 · 2026-10-05.
package activities

import (
	"context"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"strings"
	"testing"
)

func TestExtractionUnknownAndDevPerformNoStoreCalls(t *testing.T) {
	// A nil store would otherwise be consulted at entry. Policy must reject first.
	for _, mode := range []string{"", "DEV", "REAL", "unknown"} {
		activities := EntityExtractionActivities{}
		run := flow.RunRef{MatterMode: mode}
		_, err := activities.ProposeEntitiesRules(context.Background(), flow.ExtractionRequest{Run: run})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.ExtractEntitiesEventsModel(context.Background(), flow.ExtractionRequest{Run: run})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.ReconcileEntityProposals(context.Background(), flow.ReconcileRequest{Extraction: flow.ExtractionRequest{Run: run}})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.FinalizeExtractionCommit(context.Background(), flow.FinalizeRequest{Commit: flow.CommitRequest{Run: run}})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
	}
}

func TestConversationWriteBodiesDenyUnknownDevWithoutDependencies(t *testing.T) {
	for _, mode := range []string{"", "DEV", "REAL", "invalid"} {
		activities := ConversationActivities{}
		input := flow.ExternalRunInput{Run: flow.RunRef{MatterMode: mode}}
		_, err := activities.BeginExternalExtractionRun(context.Background(), flow.ExternalRunStart{Input: input})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.StageExternalExtractionPage(context.Background(), flow.StageExternalPage{Input: input})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		err = activities.FinishExternalExtractionRun(context.Background(), flow.FinishExternalRun{Run: input.Run})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.UpsertConversationToSurreal(context.Background(), flow.SendTarget{OperatingMode: mode})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
		_, err = activities.UpsertExtractionsToSurreal(context.Background(), flow.SendTarget{OperatingMode: mode})
		if err == nil || !strings.Contains(err.Error(), "canonical writes") {
			t.Fatal(mode, err)
		}
	}
}
