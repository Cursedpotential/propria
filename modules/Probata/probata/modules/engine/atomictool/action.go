// Package atomictool runs a source-pinned, read-only platform tool as one
// Temporal Activity. It returns a ContentStore reference rather than content.
// Byline: Codex · GPT-6.1 · 2026-10-07.
package atomictool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/toolgateway"
)

const (
	WorkflowName = "source_pinned_atomic_tool_workflow"
	ActivityName = "run_source_pinned_atomic_tool_activity"
	StatusQuery = "status"
	WorkflowIDPrefix = "source-pinned-tool:"
)

// Request is one reviewed source/tool/options decision. It binds an actor,
// canonical matter, idempotency identity and exact source digest; its only
// side effect is scheduling a read-only tool through the gateway.
type Request struct {
	OperatingMode string         `json:"operating_mode"`
	CourtCaseID   string         `json:"court_case_id"`
	MatterID      string         `json:"matter_id"`
	Actor         entities.Actor `json:"actor"`
	RequestID     string         `json:"request_id"`
	ToolID        string         `json:"tool_id"`
	SourceRef     string         `json:"source_ref"`
	SourceSHA256  string         `json:"source_sha256"`
	Args          map[string]any `json:"args"`
}

// Validate rejects missing case identity, source pin or reserved tool options.
// It changes no state and should be called at both starter and workflow entry.
func (r Request) Validate() error {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(r.OperatingMode)); err != nil { return err }
	if !caseidentity.AdmittedIdentity(r.MatterID, r.CourtCaseID) { return errors.New("atomic tool action requires the approved matter and court case") }
	if r.Actor.SubjectUID == "" || r.RequestID == "" || len(r.RequestID) > 128 { return errors.New("atomic tool action requires actor and bounded request ID") }
	if !toolgateway.IsInitialSourceTool(r.ToolID) { return errors.New("atomic tool action is not in the initial source-backed catalog") }
	if !strings.HasPrefix(r.SourceRef, "r2://") && !strings.HasPrefix(r.SourceRef, "b2://") && !strings.HasPrefix(r.SourceRef, "upload://") { return errors.New("atomic tool action requires a supported immutable source locator") }
	if len(r.SourceSHA256) != 64 { return errors.New("atomic tool action requires an exact source SHA-256") }
	for _, c := range r.SourceSHA256 { if !strings.ContainsRune("0123456789abcdef", c) { return errors.New("atomic tool action requires lowercase hex SHA-256") } }
	for key := range r.Args { if key == "path" || strings.HasPrefix(key, "_") { return errors.New("atomic tool action cannot supply host paths or reserved options") } }
	return nil
}

// Result is the bounded durable metadata from one tool invocation. Its ref
// points to the content store; there is no original content in Temporal.
type Result struct {
	ToolID         string `json:"tool_id"`
	OperationID    string `json:"operation_id"`
	AuditChainHead string `json:"audit_chain_head"`
	ResultRef      string `json:"result_ref"`
	ResultSize     int64  `json:"result_size"`
	SourceSHA256   string `json:"source_sha256"`
}

// Client is the pinned ToolGatewayClient boundary; it has no raw-runtime or
// MCP execution method, so Activities cannot bypass source verification.
type Client interface {
	RunPinned(context.Context, string, proffer.Ref, string, string, map[string]any) (json.RawMessage, error)
}

// Activities supplies the one source-backed read-only tool Activity.
type Activities struct { Client Client }

// Run invokes the pinned gateway and returns only a content-store pointer and
// audit metadata. It does not return inline tool output or mutate the source.
func (a Activities) Run(ctx context.Context, in Request) (Result, error) {
	if a.Client == nil { return Result{}, errors.New("atomic tool action requires the pinned gateway client") }
	if err := in.Validate(); err != nil { return Result{}, err }
	raw, err := a.Client.RunPinned(ctx, in.ToolID, proffer.Ref(in.SourceRef), in.SourceSHA256, in.RequestID, in.Args)
	if err != nil { return Result{}, fmt.Errorf("pinned tool action: %w", err) }
	var envelope struct {
		OperationID    string `json:"operation_id"`
		AuditChainHead string `json:"audit_chain_head"`
		Inline         bool   `json:"inline"`
		Ref            string `json:"ref"`
		Size           int64  `json:"size"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil { return Result{}, errors.New("pinned tool action returned an invalid envelope") }
	if envelope.Inline || !strings.HasPrefix(envelope.Ref, "sha256:") || len(envelope.Ref) != 71 || envelope.OperationID != in.RequestID || envelope.AuditChainHead == "" {
		return Result{}, errors.New("pinned tool action did not return a durable audited content reference")
	}
	return Result{ToolID: in.ToolID, OperationID: envelope.OperationID, AuditChainHead: envelope.AuditChainHead, ResultRef: envelope.Ref, ResultSize: envelope.Size, SourceSHA256: in.SourceSHA256}, nil
}
