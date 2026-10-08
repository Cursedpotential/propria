// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package activities

import (
	"context"
	"testing"
)

type contextSourceStoreProbe struct {
	called bool
	got    ContextSourceRegistrationRequest
}

func (p *contextSourceStoreProbe) RegisterContextSource(_ context.Context, req ContextSourceRegistrationRequest) (ContextSourceRegistrationResult, error) {
	p.called, p.got = true, req
	return ContextSourceRegistrationResult{SourceVersionRef: "version", ReceiptRef: "receipt", Status: "registered"}, nil
}

func TestContextSourceRegistrationPassesActualCoordinatesWithoutCustody(t *testing.T) {
	store := &contextSourceStoreProbe{}
	activity := ContextSourceRegistrationActivities{Store: store}
	in := ContextSourceRegistrationRequest{RequestID: "request-1", WorkflowID: "temporal-workflow-1",
		SourceRef: "native://first/message/1", ProviderVersionID: "provider-v2", PackageRef: "export-1",
		SourceKind: "first_party_ai_turn"}
	got, err := activity.RegisterContextSourceActivity(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !store.called || store.got != in || got.SourceVersionRef != "version" || got.ReceiptRef != "receipt" || got.Status != "registered" {
		t.Fatalf("context registration lost coordinates or references: %+v %+v", store.got, got)
	}
}

func TestContextSourceRegistrationRequiresRealWorkflowAndSourcePointer(t *testing.T) {
	store := &contextSourceStoreProbe{}
	activity := ContextSourceRegistrationActivities{Store: store}
	base := ContextSourceRegistrationRequest{RequestID: "request-1", SourceRef: "native://message/1", SourceKind: "document"}
	if _, err := activity.RegisterContextSourceActivity(context.Background(), base); err == nil || store.called {
		t.Fatal("accepted a missing actual Temporal workflow ID")
	}
	base.WorkflowID = "temporal-workflow-1"
	base.SourceRef = ""
	if _, err := activity.RegisterContextSourceActivity(context.Background(), base); err == nil || store.called {
		t.Fatal("accepted a missing source pointer")
	}
}

func TestContextSourceRegistrationKeepsUnknownKindDiscoverable(t *testing.T) {
	store := &contextSourceStoreProbe{}
	activity := ContextSourceRegistrationActivities{Store: store}
	_, err := activity.RegisterContextSourceActivity(context.Background(), ContextSourceRegistrationRequest{
		RequestID: "request-1", WorkflowID: "temporal-workflow-1", SourceRef: "native://unknown/1",
	})
	if err != nil || !store.called || store.got.SourceKind != "unknown" {
		t.Fatalf("unknown native source kind was lost: %+v, %v", store.got, err)
	}
}
