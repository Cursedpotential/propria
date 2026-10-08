package atomictool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/proffer"
)

type fakePinnedClient struct {
	called   bool
	envelope json.RawMessage
}

func (f *fakePinnedClient) RunPinned(_ context.Context, _ string, _ proffer.Ref, _, _ string, _ map[string]any) (json.RawMessage, error) {
	f.called = true
	return f.envelope, nil
}

func sourceRequest() Request {
	return Request{
		OperatingMode: "LIVE", MatterID: caseidentity.AuthoritativeMatterID,
		CourtCaseID: caseidentity.AuthoritativeCourtCaseID,
		Actor:       entities.Actor{SubjectUID: "operator-1"}, RequestID: "11111111-1111-1111-1111-111111111111",
		ToolID: "repair.detect", SourceRef: "upload://sealed-source", SourceSHA256: strings.Repeat("a", 64),
	}
}

// TestActionRejectsUnapprovedToolAndHostPath proves malformed requests never
// reach the gateway, even if a caller bypasses the HTTP API.
func TestActionRejectsUnapprovedToolAndHostPath(t *testing.T) {
	f := &fakePinnedClient{}
	request := sourceRequest()
	request.ToolID = "repair.write-derived"
	if _, err := (Activities{Client: f}).Run(context.Background(), request); err == nil || f.called {
		t.Fatal("unapproved tool reached pinned gateway")
	}
	request = sourceRequest()
	request.Args = map[string]any{"path": "/etc/passwd"}
	if _, err := (Activities{Client: f}).Run(context.Background(), request); err == nil || f.called {
		t.Fatal("caller host path reached pinned gateway")
	}
}

// TestActionRequiresStoredEnvelope proves inline source-derived bytes cannot
// enter Temporal activity results and only audited refs are returned.
func TestActionRequiresStoredEnvelope(t *testing.T) {
	request := sourceRequest()
	f := &fakePinnedClient{envelope: json.RawMessage(`{"operation_id":"11111111-1111-1111-1111-111111111111","audit_chain_head":"audit-1","inline":true,"result":{"private":"content"}}`)}
	if _, err := (Activities{Client: f}).Run(context.Background(), request); err == nil {
		t.Fatal("inline content was accepted")
	}
	f.envelope = json.RawMessage(`{"operation_id":"11111111-1111-1111-1111-111111111111","audit_chain_head":"audit-1","inline":false,"ref":"sha256:` + strings.Repeat("b", 64) + `","size":21,"preview":"private content"}`)
	result, err := (Activities{Client: f}).Run(context.Background(), request)
	if err != nil || result.ResultRef != "sha256:"+strings.Repeat("b", 64) || result.AuditChainHead != "audit-1" {
		t.Fatalf("stored envelope rejected: %+v %v", result, err)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "private content") {
		t.Fatal("source-derived preview entered Temporal result")
	}
}
