package proffer

// ContextPackageRequest pins one prepared native source and all completed derivative bundles.
// Inputs: actual workflow/source identities and mounted bundle references. Outputs: atomic adapter request.
// Effects: none; choose after preparation and complete work-product extraction, including partial later stages.
type ContextPackageRequest struct {
	RequestID              string `json:"request_id"`
	WorkflowID             string `json:"workflow_id"`
	RunID                  string `json:"run_id"`
	SourceRef              string `json:"source_ref"`
	ProviderVersionID      string `json:"provider_version_id,omitempty"`
	PackageRef             string `json:"package_ref,omitempty"`
	SourceFormat           string `json:"source_format"`
	SourceVersionRef       string `json:"source_version_ref"`
	RegistrationReceiptRef string `json:"registration_receipt_ref"`
	PreparedRef            string `json:"prepared_ref"`
	WorkProductsRef        string `json:"work_products_ref"`
	CandidatesRef          string `json:"candidates_ref,omitempty"`
	EmbeddingsRef          string `json:"embeddings_ref,omitempty"`
	PublicationRef         string `json:"publication_ref,omitempty"`
	VerificationRef        string `json:"verification_ref,omitempty"`
}

// ContextPackageResult reports exact physical placement separately from catalog admission.
// Inputs: retained package/readback and metadata registration. Outputs: compact manifest and receipt pins.
// Effects: none; choose for Temporal/API results without returning native turns or created-work bodies.
type ContextPackageResult struct {
	ManifestRef          string `json:"manifest_ref"`
	ManifestSHA256       string `json:"manifest_sha256"`
	ReceiptRef           string `json:"receipt_ref"`
	ReceiptSHA256        string `json:"receipt_sha256"`
	Files                int    `json:"files"`
	Bytes                int64  `json:"bytes"`
	Complete             bool   `json:"complete"`
	CatalogStatus        string `json:"catalog_status"`
	CatalogReceiptRef    string `json:"catalog_receipt_ref,omitempty"`
	CatalogReceiptSHA256 string `json:"catalog_receipt_sha256,omitempty"`
}

// ContextReviewSource carries registered native coordinates without inventing custody object pins.
// Inputs: registration version, actual source URI and optional provider version. Outputs: staging identity.
// Effects: none; choose for the existing candidate Activity's context-v1 adapter.
type ContextReviewSource struct {
	SourceVersionID string  `json:"source_version_id"`
	SourceRef       string  `json:"source_ref"`
	VersionID       *string `json:"version_id"`
}

// ContextReviewRequest asks the existing staging Activity to independently derive and verify file pins.
// Inputs: existing LIVE scope, request identity and complete prepared/candidate refs. Outputs: compact Activity input.
// Effects: none; choose after candidate extraction without reading files inside a workflow.
type ContextReviewRequest struct {
	ContractVersion string              `json:"contract_version"`
	OperatingMode   string              `json:"operating_mode"`
	MatterID        string              `json:"matter_id"`
	CourtCaseID     string              `json:"court_case_id"`
	RequestID       string              `json:"request_id"`
	Source          ContextReviewSource `json:"source"`
	PreparedRef     string              `json:"prepared_ref"`
	BundleRef       string              `json:"bundle_ref"`
}

// ContextReviewResult preserves genuine staged candidate IDs for existing review surfaces.
// Inputs: the existing candidate staging Activity result. Outputs: run/digest/candidate identity only.
// Effects: none; choose rather than copying native AI turns into workflow or PostgreSQL records.
type ContextReviewResult struct {
	RunID         string                       `json:"run_id"`
	RequestDigest string                       `json:"request_digest"`
	CandidateIDs  []string                     `json:"candidate_ids"`
	Source        *ContextVerifiedReviewSource `json:"source,omitempty"`
	Candidates    int                          `json:"candidates"`
	Staged        int                          `json:"staged_occurrences"`
}

// ContextVerifiedReviewSource preserves exact pins returned by the independent native staging check.
// Inputs: the existing Stage Activity result. Outputs: registered identity, original digest and prepared reference.
// Effects: none; choose for review UI requests rather than deriving hashes from registration or HEAD metadata.
type ContextVerifiedReviewSource struct {
	ContextReviewSource
	SourceObjectID string `json:"source_object_id"`
	SourceSHA256   string `json:"source_sha256"`
	PreparedRef    string `json:"prepared_ref"`
}

// ContextCandidateStage reports staging completion separately from candidate review decisions.
// Inputs: successful Activity counts or a specific execution failure. Outputs: compact status/count.
// Effects: none; choose for polling without claiming approved graph promotion.
type ContextCandidateStage struct {
	Status     string `json:"status"`
	Candidates int    `json:"candidates"`
	Staged     int    `json:"staged_occurrences"`
}

// ContextCatalogRequest pins a verified physical package for the existing catalog functions.
// Inputs: registered source request and immutable physical readback. Outputs: compact Activity coordinates.
// Effects: none; choose after package retention, separately from Weaviate publication or graph approval.
type ContextCatalogRequest struct {
	Request ContextPackageRequest `json:"request"`
	Package ContextPackageResult  `json:"package"`
}
