// Byline: Codex · GPT-5 · 2026-10-05
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// The embedded nil Store makes any unexpected read or write panic.
type ownerAdmissionStore struct {
	Store
	durable                 flow.RunRef
	reads, begins, finishes int
}

func (s *ownerAdmissionStore) ResolveRun(context.Context, string) (flow.RunRef, error) {
	s.reads++
	return s.durable, nil
}
func (s *ownerAdmissionStore) BeginRun(context.Context, RunRow) error { s.begins++; return nil }
func (s *ownerAdmissionStore) FinishRun(context.Context, string, string, map[string]any, string) error {
	s.finishes++
	return nil
}

func ownerTestRun() flow.RunRef {
	return flow.RunRef{PreviewHandle: "handle", GenerationID: "generation", SourceVersionID: "source", MatterMode: "LIVE"}
}

func TestOwnerMutatorsRejectDevOrUnknownBeforeAnyStoreCall(t *testing.T) {
	for _, mode := range []string{"DEV", "", "REAL", "TEST", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			run := ownerTestRun()
			run.MatterMode = mode
			store := &ownerAdmissionStore{durable: ownerTestRun()}
			if err := Correct(context.Background(), store, run, CorrectionEnvelope{}, entities.Actor{}, time.Time{}, "digest"); err == nil {
				t.Fatal("correction admitted")
			}
			if _, err := MarkEvent(context.Background(), store, run, events.MarkRequest{}, entities.Actor{}, time.Time{}, "digest"); err == nil {
				t.Fatal("event mark admitted")
			}
			if store.reads+store.begins+store.finishes != 0 {
				t.Fatal("store reached", store)
			}
		})
	}
}

func TestOwnerMutatorsRequireDurableLiveReceiptAndMatchingCoordinates(t *testing.T) {
	for _, field := range []string{"unknown", "dev", "handle", "generation", "source"} {
		t.Run(field, func(t *testing.T) {
			run, durable := ownerTestRun(), ownerTestRun()
			switch field {
			case "unknown":
				durable.MatterMode = ""
			case "dev":
				durable.MatterMode = "DEV"
			case "handle":
				durable.PreviewHandle = "other"
			case "generation":
				durable.GenerationID = "other"
			case "source":
				durable.SourceVersionID = "other"
			}
			store := &ownerAdmissionStore{durable: durable}
			if err := Correct(context.Background(), store, run, CorrectionEnvelope{}, entities.Actor{}, time.Time{}, "digest"); err == nil {
				t.Fatal("correction admitted")
			}
			if _, err := MarkEvent(context.Background(), store, run, events.MarkRequest{}, entities.Actor{}, time.Time{}, "digest"); err == nil {
				t.Fatal("event mark admitted")
			}
			if store.reads != 2 || store.begins+store.finishes != 0 {
				t.Fatal("write reached", store)
			}
		})
	}
}

func TestOwnerCorrectionLiveReceiptPreservesOwnerRunWrites(t *testing.T) {
	store := &ownerAdmissionStore{durable: ownerTestRun()}
	err := Correct(context.Background(), store, ownerTestRun(), CorrectionEnvelope{}, entities.Actor{}, time.Time{}, "digest")
	var invalid ErrInvalid
	if !errors.As(err, &invalid) || errors.Is(err, caseidentity.ErrOperatingModeUnknown) || store.reads != 1 || store.begins != 1 || store.finishes != 1 {
		t.Fatalf("LIVE write path: err=%v store=%+v", err, store)
	}
}
