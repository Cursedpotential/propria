// Byline: Codex · GPT-6.1-sol · 2026-10-07
package contextgraphflow

import "github.com/Cursedpotential/probata/engine/surrealsink"

// Names bind the standalone graph workflow to the existing Go and Python workers.
const (
	WorkflowName         = "context_graph_workflow"
	PrepareActivityName  = "ai_prepare_context_graph_activity"
	ProjectActivityName  = "project_context_graph_activity"
	VerifyActivityName   = "verify_context_graph_activity"
	TraverseActivityName = "traverse_context_graph_activity"
	PythonTaskQueue      = "evidence-pipeline"
)

// Request carries exact existing AI admission pins and retained predecessor references.
// Inputs are server-managed scope, seven verified AI binding fields, external refs
// and explicit completeness counts; output is a reference-only request, without
// I/O or granted authority. Pick for the standalone downstream graph workflow.
type Request struct {
	RequestID              string                         `json:"request_id"`
	SourceVersionID        string                         `json:"source_version_id"`
	NormalizedGenerationID string                         `json:"normalized_generation_id"`
	VerificationID         string                         `json:"verification_id"`
	OperatingMode          string                         `json:"operating_mode"`
	MatterID               string                         `json:"matter_id"`
	CourtCaseID            string                         `json:"court_case_id"`
	PreparedRef            string                         `json:"prepared_ref"`
	WorkProductsRef        string                         `json:"work_products_ref"`
	AccessPolicyID         string                         `json:"access_policy_id"`
	CreatedByService       string                         `json:"created_by_service"`
	ExpectedSourceTurns    int                            `json:"expected_source_turns"`
	ExpectedCreatedWorks   int                            `json:"expected_created_works"`
	ExpectedConversations  int                            `json:"expected_conversations"`
	SourcePins             []surrealsink.ContextSourcePin `json:"source_pins,omitempty"`
}

// BatchRef identifies one complete bounded projection batch without source bodies.
// Inputs are retained preparation manifest coordinates; output is a typed value.
// No effects occur; pick to schedule independent project and verification units.
type BatchRef struct {
	BundleRef        string `json:"bundle_ref"`
	BundleSHA256     string `json:"bundle_sha256"`
	GenerationID     string `json:"generation_id"`
	ExtractionRunRef string `json:"extraction_run_ref"`
	SourceTurns      int    `json:"source_turns"`
	CreatedWorks     int    `json:"created_works"`
	Nodes            int    `json:"nodes"`
	Edges            int    `json:"edges"`
}

// PreparationResult reports actual retained batch refs and echoed verified AI bindings.
// Inputs are the Python Activity's actual admission and runtime receipts; output
// is bounded manifest/count metadata. No effects occur; pick at its workflow boundary.
type PreparationResult struct {
	RequestID              string     `json:"request_id"`
	SourceVersionID        string     `json:"source_version_id"`
	NormalizedGenerationID string     `json:"normalized_generation_id"`
	VerificationID         string     `json:"verification_id"`
	OperatingMode          string     `json:"operating_mode"`
	MatterID               string     `json:"matter_id"`
	CourtCaseID            string     `json:"court_case_id"`
	ManifestRef            string     `json:"manifest_ref"`
	ManifestHash           string     `json:"manifest_hash"`
	SourceTurns            int        `json:"source_turns"`
	CreatedWorks           int        `json:"created_works"`
	Conversations          int        `json:"conversations"`
	Batches                int        `json:"batches"`
	BatchRefs              []BatchRef `json:"batch_refs"`
	TemporalWorkflowID     string     `json:"temporal_workflow_id"`
	TemporalRunID          string     `json:"temporal_run_id"`
}

// BatchRequest keeps one sealed bundle tied to the workflow's admitted source pins.
// Inputs are existing request pins plus exact batch locator/hash/generation/run.
// Output is a reference-only Activity request with no effects or granted authority.
type BatchRequest struct {
	Request
	BundleRef        string `json:"bundle_ref"`
	BundleSHA256     string `json:"bundle_sha256"`
	CaseID           string `json:"case_id"`
	GenerationID     string `json:"generation_id"`
	ExtractionRunRef string `json:"extraction_run_ref"`
}

// BatchResult carries verified graph counts/checkpoints and actual Activity identity.
// Inputs are independent store readback; output excludes source bodies. No effects
// occur; pick as the Temporal result for project, verify and private traversal.
type BatchResult struct {
	Receipt            surrealsink.ContextGraphReceipt `json:"receipt"`
	BundleSHA256       string                          `json:"bundle_sha256"`
	SourceTurns        int                             `json:"source_turns"`
	CreatedWorks       int                             `json:"created_works"`
	OutputRef          string                          `json:"output_ref,omitempty"`
	OutputSHA256       string                          `json:"output_sha256,omitempty"`
	TraversalNodeCount int                             `json:"traversal_node_count,omitempty"`
	TraversalEdgeCount int                             `json:"traversal_edge_count,omitempty"`
	TemporalWorkflowID string                          `json:"temporal_workflow_id"`
	TemporalRunID      string                          `json:"temporal_run_id"`
	TemporalActivityID string                          `json:"temporal_activity_id"`
	TemporalAttempt    int32                           `json:"temporal_attempt"`
}

// Result locates the complete independently verified graph without copying case content.
// Inputs are settled manifest and batch receipts; output is a reference/count
// workflow result. No effects occur; pick only after every batch passes readback.
type Result struct {
	ManifestRef        string                            `json:"manifest_ref"`
	ManifestHash       string                            `json:"manifest_hash"`
	SourceTurns        int                               `json:"source_turns"`
	CreatedWorks       int                               `json:"created_works"`
	Conversations      int                               `json:"conversations"`
	Batches            int                               `json:"batches"`
	Verified           []surrealsink.ContextGraphReceipt `json:"verified"`
	TemporalWorkflowID string                            `json:"temporal_workflow_id"`
	TemporalRunID      string                            `json:"temporal_run_id"`
}
