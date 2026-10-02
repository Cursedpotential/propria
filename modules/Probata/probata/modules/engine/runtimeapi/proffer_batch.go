// Byline: Claude Code · Opus 5 · 2026-09-21
//
// The batch-by-folder HTTP surface (owner 2026-09-20 23:52: "it gets batched
// by folder ... process them one at a time"; build order step 4 in
// docs/decisions/2026-09-20-bulk-intake-owner-requirements.md).
//
// Fan-out belongs here rather than in a child workflow because the preview
// binding — the thing that makes a run visible in Review — is created on the
// start path (proffer_preview.go start). This endpoint starts ONE durable
// batch workflow, which creates each item's binding through the same store;
// the sequencing lives in that workflow, not in a handler goroutine that a
// redeploy would kill.
//
// The folder locator is validated exactly as a single start validates a
// source: it must resolve inside a configured source root, so no caller can
// name a bucket outside the configured stores.

package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// batchIDPattern keeps a batch id addressable in a path segment.
var batchIDPattern = previewHandlePattern

// BatchWorkflowClient is the Temporal boundary for a folder batch.
type BatchWorkflowClient interface {
	StartBatch(ctx context.Context, in proffer.BatchInput) (workflowID, runID string, err error)
	BatchStatus(ctx context.Context, batchID string) (proffer.BatchStatus, error)
}

// UseBatchWorkflow enables POST /reference-import/start-batch and
// GET /reference-import/batches/{id}. Without it both routes answer 503
// rather than 404, so a misconfigured deployment is visible instead of
// looking like an unbuilt feature.
func (h *PreviewHTTPHandler) UseBatchWorkflow(client BatchWorkflowClient) error {
	if client == nil {
		return errors.New("proffer batch: workflow client is required")
	}
	h.batch = client
	return nil
}

type batchStartRequest struct {
	BatchID          string `json:"batch_id"`
	MatterID         string `json:"matter_id"`
	CourtCaseID      string `json:"court_case_id"`
	FolderRef        string `json:"folder_ref"`
	DeclaredFormat   string `json:"declared_format"`
	ParserOptionsRef string `json:"parser_options_ref"`
	SourceContextRef string `json:"source_context_ref"`
	MaxInFlight      int    `json:"max_in_flight"`
	// Explicit D04 identity, passed to every item's run. Byline: Claude Code · Opus 5.5 · 2026-10-01
	OwnerPersonID       string `json:"owner_person_id"`
	PerspectivePersonID string `json:"perspective_person_id"`
	// "clean_checks" switches the owner's "auto-approve clean runs" policy on
	// for this batch only. Absent means every item waits for the owner.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	AutoApproval string `json:"auto_approval"`
}

// startBatch validates the folder and starts one batch workflow.
func (h *PreviewHTTPHandler) startBatch(w http.ResponseWriter, r *http.Request) {
	if h.batch == nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("batch import is not configured on this service"))
		return
	}
	var req batchStartRequest
	if err := decodePreviewJSON(w, r, &req); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.BatchID) == "" || strings.TrimSpace(req.FolderRef) == "" ||
		strings.TrimSpace(req.DeclaredFormat) == "" || strings.TrimSpace(req.ParserOptionsRef) == "" {
		previewError(w, http.StatusBadRequest, errors.New("start-batch request is incomplete"))
		return
	}
	if !batchIDPattern.MatchString(req.BatchID) {
		previewError(w, http.StatusBadRequest, errors.New("batch_id must be 32-128 URL-safe characters"))
		return
	}
	if _, err := uuid.Parse(req.MatterID); err != nil {
		previewError(w, http.StatusBadRequest, errors.New("matter_id must be a UUID"))
		return
	}
	if _, err := uuid.Parse(req.CourtCaseID); err != nil {
		previewError(w, http.StatusBadRequest, errors.New("court_case_id must be a UUID"))
		return
	}
	if req.MaxInFlight < 0 || req.MaxInFlight > 16 {
		previewError(w, http.StatusUnprocessableEntity, errors.New("max_in_flight must be between 0 and 16"))
		return
	}
	scheme, bucket, prefix, err := validateAuthorizedFolderRef(req.FolderRef)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if req.SourceContextRef != "" {
		if _, err := uuid.Parse(req.SourceContextRef); err != nil {
			previewError(w, http.StatusUnprocessableEntity, errors.New("source_context_ref must be a UUID"))
			return
		}
		if h.sourceContext == nil {
			previewError(w, http.StatusServiceUnavailable, errors.New("source context validation is unavailable"))
			return
		}
	}
	if err := validateOptionalPersonIDs(req.OwnerPersonID, req.PerspectivePersonID); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	if !proffer.ValidAutoApproval(strings.TrimSpace(req.AutoApproval)) {
		previewError(w, http.StatusUnprocessableEntity, errors.New("auto_approval must be empty or \"clean_checks\""))
		return
	}
	maxInFlight := req.MaxInFlight
	if maxInFlight == 0 {
		maxInFlight = batchMaxInFlightFromEnv(os.Getenv("PROFFER_BATCH_MAX_IN_FLIGHT"))
	}
	in := proffer.BatchInput{
		BatchID: req.BatchID, MatterID: req.MatterID, CourtCaseID: req.CourtCaseID,
		Scheme: scheme, Bucket: bucket, Prefix: prefix,
		DeclaredFormat: req.DeclaredFormat, ParserOptionsRef: proffer.Ref(req.ParserOptionsRef),
		SourceContextRef: proffer.Ref(req.SourceContextRef), MaxInFlight: maxInFlight,
		OwnerPersonID: strings.TrimSpace(req.OwnerPersonID), PerspectivePersonID: strings.TrimSpace(req.PerspectivePersonID),
		AutoApproval: strings.TrimSpace(req.AutoApproval),
	}
	if _, _, err := h.batch.StartBatch(r.Context(), in); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	previewJSON(w, http.StatusCreated, map[string]string{"batch_id": req.BatchID})
}

// batchStatus reports per-item status and counts for one batch.
func (h *PreviewHTTPHandler) batchStatus(w http.ResponseWriter, r *http.Request) {
	if h.batch == nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("batch import is not configured on this service"))
		return
	}
	id := r.PathValue("batch_id")
	if !batchIDPattern.MatchString(id) {
		previewError(w, http.StatusBadRequest, errors.New("batch_id must be 32-128 URL-safe characters"))
		return
	}
	status, err := h.batch.BatchStatus(r.Context(), id)
	if err != nil {
		previewError(w, http.StatusNotFound, errors.New("batch not found"))
		return
	}
	// A nil slice marshals to null and the BFF rejects it.
	if status.Items == nil {
		status.Items = []proffer.BatchItem{}
	}
	previewJSON(w, http.StatusOK, status)
}

// validateAuthorizedFolderRef admits only a prefix that lies inside a
// configured source root, the same authority a single start uses. It returns
// the scheme, bucket and a prefix that always ends in "/".
func validateAuthorizedFolderRef(value string) (scheme, bucket, prefix string, err error) {
	outside := errors.New("folder_ref is outside the authorized Case Bible intake roots")
	parsed, parseErr := url.Parse(strings.TrimSpace(value))
	if parseErr != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return "", "", "", outside
	}
	key, unescapeErr := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
	if unescapeErr != nil || !safeObjectKey(key) {
		return "", "", "", outside
	}
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	roots, rootsErr := objectstores.RootsFromEnv()
	if rootsErr != nil {
		return "", "", "", outside
	}
	// A folder is authorized exactly when an object beneath it would be, so
	// the check is Match on a sentinel child key. Roots.Match refuses a key
	// equal to a bare root prefix (it is not an object), and a batch may
	// legitimately name a configured root itself as its folder.
	//
	// This is the same authority a single start uses, so a caller-supplied
	// bucket outside the configured stores never reaches the lister.
	if _, ok := roots.Match(parsed.Scheme, parsed.Host, key+"\x00"); !ok {
		return "", "", "", outside
	}
	return strings.ToLower(parsed.Scheme), parsed.Host, key, nil
}

// batchMaxInFlightFromEnv reads PROFFER_BATCH_MAX_IN_FLIGHT. It is the default
// applied when a request does not ask for one; one at a time is the owner's
// instruction and the fallback.
func batchMaxInFlightFromEnv(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 || value > 16 {
		return 1
	}
	return value
}
