// Byline: Codex | GPT-6.1-sol | 2026-10-07
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contextgraphflow"
	"github.com/Cursedpotential/probata/engine/surrealsink"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// Context graph Activity names are stable registration keys on the existing worker.
const (
	ProjectContextGraphActivityName  = contextgraphflow.ProjectActivityName
	VerifyContextGraphActivityName   = contextgraphflow.VerifyActivityName
	TraverseContextGraphActivityName = contextgraphflow.TraverseActivityName
)

// ContextGraphActivityConfig binds an existing managed case and private RW graph mount.
// Inputs are Root, admitted Scope and explicit Surreal DATABASE config; output is
// configuration only. No I/O occurs here; pick for the production worker constructor.
type ContextGraphActivityConfig struct {
	Root    string
	Scope   surrealsink.ContextGraphScope
	Surreal surrealsink.Config
}

// ContextGraphActivities projects or traverses retained bundles in fct/analysis only.
// Inputs are sealed references and hashes; outputs are reference-only receipts.
// Effects are scoped graph writes or private readback artifacts; pick downstream
// of ingestion/extraction, never as an ingestion, model or schema Activity.
type ContextGraphActivities struct {
	root   string
	scope  surrealsink.ContextGraphScope
	client *surrealsink.Client
}

// NewContextGraphActivities binds canonical case identity and existing private root.
// Input is server-managed configuration; output is a production Activity group.
// Effects inspect root metadata and construct the existing analytical Query client;
// no network or database writes occur. Pick before registering the Activities.
func NewContextGraphActivities(cfg ContextGraphActivityConfig) (ContextGraphActivities, error) {
	if !caseidentity.AdmittedIdentity(cfg.Scope.MatterID, cfg.Scope.CaseID) || !contextGraphID(cfg.Scope.AccessPolicyID) || !contextGraphID(cfg.Scope.CreatedByService) {
		return ContextGraphActivities{}, errors.New("managed canonical graph scope required")
	}
	root, e := canonicalDirectory(cfg.Root)
	if e != nil {
		return ContextGraphActivities{}, errors.New("private graph root must be an existing absolute directory")
	}
	client, e := surrealsink.NewAnalysis(cfg.Surreal)
	if e != nil {
		return ContextGraphActivities{}, e
	}
	return ContextGraphActivities{root: root, scope: cfg.Scope, client: client}, nil
}

// ContextGraphActivitiesFromEnv loads exact mounted production configuration without defaults.
// Inputs: ANALYSIS_GRAPH_ROOT points to an existing private RW mount; managed scope
// uses ANALYSIS_GRAPH_MATTER_ID, ANALYSIS_GRAPH_CASE_ID, ANALYSIS_GRAPH_ACCESS_POLICY_ID,
// ANALYSIS_GRAPH_CREATED_BY_SERVICE. Surreal uses ANALYSIS_SURREAL_URL, explicit
// ANALYSIS_SURREAL_NAMESPACE=fct, ANALYSIS_SURREAL_DATABASE=analysis,
// ANALYSIS_SURREAL_AUTH_LEVEL=database, ANALYSIS_SURREAL_USER_FILE and
// ANALYSIS_SURREAL_PASSWORD_FILE (existing mounted credential files, never values).
// Output is the Activity group; effects read metadata/secrets only. Pick at worker
// boot; mount the same private bundle/artifact directory RW at ANALYSIS_GRAPH_ROOT.
func ContextGraphActivitiesFromEnv() (ContextGraphActivities, error) {
	surreal, e := surrealsink.AnalysisConfigFromEnv()
	if e != nil {
		return ContextGraphActivities{}, e
	}
	return NewContextGraphActivities(ContextGraphActivityConfig{Root: os.Getenv("ANALYSIS_GRAPH_ROOT"), Scope: surrealsink.ContextGraphScope{MatterID: os.Getenv("ANALYSIS_GRAPH_MATTER_ID"), CaseID: os.Getenv("ANALYSIS_GRAPH_CASE_ID"), AccessPolicyID: os.Getenv("ANALYSIS_GRAPH_ACCESS_POLICY_ID"), CreatedByService: os.Getenv("ANALYSIS_GRAPH_CREATED_BY_SERVICE")}, Surreal: surreal})
}

// RegisterContextGraphActivities installs three atomic units on the existing Temporal registrar.
// Inputs are ActivityRegistrar and the configured production group; output is none.
// Effects register exact named functions only; pick in the existing Go worker's
// registration path without adding a workflow or widening conversation extraction.
func RegisterContextGraphActivities(registrar ActivityRegistrar, acts ContextGraphActivities) {
	registrar.RegisterActivityWithOptions(acts.ProjectContextGraph, activity.RegisterOptions{Name: ProjectContextGraphActivityName})
	registrar.RegisterActivityWithOptions(acts.VerifyContextGraph, activity.RegisterOptions{Name: VerifyContextGraphActivityName})
	registrar.RegisterActivityWithOptions(acts.TraverseContextGraph, activity.RegisterOptions{Name: TraverseContextGraphActivityName})
}

// ContextGraphActivityRequest pins a sealed local bundle to managed scope and extraction run.
// Inputs are references/identifiers/hashes only; output is a typed request with no
// effects or granted authority. Pick for projection; never put bundle bodies in history.
type ContextGraphActivityRequest = contextgraphflow.BatchRequest

// ContextGraphTraversalRequest adds bounded traversal coordinates and a private result reference.
// Inputs are a sealed bundle pin, node/hops and output file URI; output is a typed
// request without effects. Pick for source-cited traversal; result bodies remain external.
type ContextGraphTraversalRequest struct {
	ContextGraphActivityRequest
	NodeID    string `json:"node_id"`
	Hops      int    `json:"hops"`
	OutputRef string `json:"output_ref"`
}

// ContextGraphActivityResult carries verified checkpoint/artifact pins and actual Temporal identity.
// Inputs are verified store/readback results; output contains only references, hashes,
// counts and runtime IDs. No effects occur; pick as the only Temporal result payload.
type ContextGraphActivityResult = contextgraphflow.BatchResult

func contextGraphID(s string) bool {
	return len(s) > 0 && len(s) <= 240 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\r\n\x00")
}
func contextGraphRef(s string) bool {
	return len(s) > 0 && len(s) <= 2000 && !strings.ContainsAny(s, "\r\n\x00")
}
func contextGraphBadInput() error {
	return temporal.NewNonRetryableApplicationError("sealed graph bundle, run or managed scope is invalid", "ContextGraphInput", nil)
}

func (a ContextGraphActivities) filePath(ref string) (string, error) {
	if !contextGraphRef(ref) {
		return "", errors.New("bounded sealed graph file reference required")
	}
	u, e := url.Parse(ref)
	if e != nil || u.Scheme != "file" || u.Host != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("sealed graph reference must be a local absolute file URI")
	}
	p := filepath.FromSlash(u.Path)
	if len(p) > 1 && p[0] == os.PathSeparator && filepath.VolumeName(p[1:]) != "" {
		p = p[1:]
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("absolute graph file URI required")
	}
	clean := filepath.Clean(p)
	rel, e := filepath.Rel(a.root, clean)
	if e != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("graph file reference escaped managed root")
	}
	return clean, nil
}

func (a ContextGraphActivities) load(request ContextGraphActivityRequest) (surrealsink.ContextGraphBundle, error) {
	empty := surrealsink.ContextGraphBundle{}
	if a.client == nil || a.root == "" {
		return empty, temporal.NewNonRetryableApplicationError("context graph Activities are not configured", "ContextGraphConfiguration", nil)
	}
	scope := surrealsink.ContextGraphScope{MatterID: request.MatterID, CaseID: request.CaseID, AccessPolicyID: request.AccessPolicyID, CreatedByService: request.CreatedByService}
	if scope != a.scope || !contextGraphID(request.GenerationID) || !contextGraphRef(request.ExtractionRunRef) || !validSHA256(request.BundleSHA256) {
		return empty, contextGraphBadInput()
	}
	for _, pin := range []string{request.RequestID, request.SourceVersionID, request.NormalizedGenerationID, request.VerificationID, request.OperatingMode} {
		if !contextGraphID(pin) {
			return empty, contextGraphBadInput()
		}
	}
	if request.CourtCaseID != a.scope.CaseID || request.ExtractionRunRef != request.PreparedRef || !contextGraphRef(request.WorkProductsRef) || len(request.SourcePins) < 1 || len(request.SourcePins) > 64 {
		return empty, contextGraphBadInput()
	}
	path, e := a.filePath(request.BundleRef)
	if e != nil {
		return empty, contextGraphBadInput()
	}
	relative, e := filepath.Rel(a.root, path)
	if e != nil {
		return empty, contextGraphBadInput()
	}
	file, e := toolkitPlacementOpenLocal(a.root, filepath.ToSlash(relative))
	if e != nil {
		return empty, contextGraphBadInput()
	}
	defer file.Close()
	before, e := file.Stat()
	if e != nil || before.Size() > surrealsink.MaxContextGraphBundleBytes {
		return empty, contextGraphBadInput()
	}
	raw, e := io.ReadAll(io.LimitReader(file, surrealsink.MaxContextGraphBundleBytes+1))
	if e != nil || len(raw) > surrealsink.MaxContextGraphBundleBytes {
		return empty, contextGraphBadInput()
	}
	after, e := file.Stat()
	if e != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return empty, contextGraphBadInput()
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != request.BundleSHA256 {
		return empty, contextGraphBadInput()
	}
	bundle, e := surrealsink.DecodeContextGraphBundle(bytes.NewReader(raw))
	if e != nil || bundle.Scope != a.scope || bundle.GenerationID != request.GenerationID || bundle.ExtractionRunRef != request.ExtractionRunRef {
		return empty, contextGraphBadInput()
	}
	{
		expected := map[[4]string]bool{}
		for _, pin := range request.SourcePins {
			if !contextGraphID(pin.SourceID) || !contextGraphID(pin.SourceVersionID) || !validSHA256(pin.SourceHash) || len(pin.Locator) == 0 || len(pin.Locator) > 2000 || (pin.ValidationRef != "" && !contextGraphID(pin.ValidationRef)) {
				return empty, contextGraphBadInput()
			}
			expected[contextGraphRootPin(pin)] = true
		}
		for _, node := range bundle.Nodes {
			for _, pin := range node.SourcePins {
				if !expected[contextGraphRootPin(pin)] {
					return empty, contextGraphBadInput()
				}
			}
		}
		for _, edge := range bundle.Edges {
			for _, pin := range edge.SourcePins {
				if !expected[contextGraphRootPin(pin)] {
					return empty, contextGraphBadInput()
				}
			}
		}
	}
	return bundle, nil
}

func contextGraphRootPin(pin surrealsink.ContextSourcePin) [4]string {
	return [4]string{pin.SourceID, pin.SourceVersionID, pin.SourceHash, pin.ValidationRef}
}

// ContextGraphInputAdmission reports sealed-input validation without implying graph persistence.
// Inputs are locally verified bundle pins/counts; output excludes source bodies.
// No effects occur here; pick for offline mount/root-pin/mode validation receipts.
type ContextGraphInputAdmission struct {
	BundleRef        string `json:"bundle_ref"`
	BundleSHA256     string `json:"bundle_sha256"`
	GenerationID     string `json:"generation_id"`
	ExtractionRunRef string `json:"extraction_run_ref"`
	SourceTurns      int    `json:"source_turns"`
	CreatedWorks     int    `json:"created_works"`
	Nodes            int    `json:"nodes"`
	Edges            int    `json:"edges"`
}

// ValidateSealedInput checks production bundle admission and LIVE mode without network access.
// Input is the same reference/hash/source/run/scope request used by project; output
// is a reference/count admission receipt. Effects read the bounded sealed file only.
// Pick for offline deployment preflight; it never calls Surreal or claims persistence.
func (a ContextGraphActivities) ValidateSealedInput(request ContextGraphActivityRequest) (ContextGraphInputAdmission, error) {
	bundle, e := a.load(request)
	if e != nil {
		return ContextGraphInputAdmission{}, e
	}
	if caseidentity.RequireCanonicalWrite(caseidentity.Mode(request.OperatingMode)) != nil {
		return ContextGraphInputAdmission{}, contextGraphBadInput()
	}
	out := ContextGraphInputAdmission{BundleRef: request.BundleRef, BundleSHA256: request.BundleSHA256, GenerationID: request.GenerationID, ExtractionRunRef: request.ExtractionRunRef, Nodes: len(bundle.Nodes), Edges: len(bundle.Edges)}
	for _, node := range bundle.Nodes {
		if node.DerivativeKind == "source_turn" || node.DerivativeKind == "ai_source_turn" {
			out.SourceTurns++
		}
		if node.DerivativeKind == "created_work" {
			out.CreatedWorks++
		}
	}
	return out, nil
}

func contextGraphHeartbeat(ctx context.Context, request ContextGraphActivityRequest, operation string) func() {
	stop := make(chan struct{})
	progress := map[string]string{"operation": operation, "generation_id": request.GenerationID, "bundle_sha256": request.BundleSHA256}
	activity.RecordHeartbeat(ctx, progress)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx, progress)
			}
		}
	}()
	return func() { close(stop) }
}

func contextGraphRuntime(ctx context.Context, hash string, receipt surrealsink.ContextGraphReceipt, bundle surrealsink.ContextGraphBundle) ContextGraphActivityResult {
	info := activity.GetInfo(ctx)
	result := ContextGraphActivityResult{Receipt: receipt, BundleSHA256: hash, TemporalWorkflowID: info.WorkflowExecution.ID, TemporalRunID: info.WorkflowExecution.RunID, TemporalActivityID: info.ActivityID, TemporalAttempt: info.Attempt}
	for _, node := range bundle.Nodes {
		if node.DerivativeKind == "source_turn" || node.DerivativeKind == "ai_source_turn" {
			result.SourceTurns++
		}
		if node.DerivativeKind == "created_work" {
			result.CreatedWorks++
		}
	}
	return result
}

// ProjectContextGraph projects one sealed retained bundle through the analytical client.
// Input is a reference/hash and managed scope/run pins; output is a verified reference-only
// checkpoint with actual Temporal identity. Effects atomically write admitted graph records;
// heartbeat/cancellation use the Activity context and retries reuse the immutable generation.
// Pick downstream of extraction after root-reviewed schema admission, never to create schema.
func (a ContextGraphActivities) ProjectContextGraph(ctx context.Context, request ContextGraphActivityRequest) (ContextGraphActivityResult, error) {
	bundle, e := a.load(request)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	if caseidentity.RequireCanonicalWrite(caseidentity.Mode(request.OperatingMode)) != nil {
		return ContextGraphActivityResult{}, contextGraphBadInput()
	}
	stop := contextGraphHeartbeat(ctx, request, "project")
	defer stop()
	receipt, e := a.client.ProjectContextGraph(ctx, bundle)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	return contextGraphRuntime(ctx, request.BundleSHA256, receipt, bundle), nil
}

// VerifyContextGraph independently verifies the complete stored generation against sealed input.
// Input is the exact reference/hash and managed source/run/scope pins; output is a
// reference-only checkpoint/count receipt with actual Temporal identity. Effects
// read Surreal only; heartbeat/cancellation use Activity context. Pick after project
// as its own Activity, never substitute the projector's same-call success receipt.
func (a ContextGraphActivities) VerifyContextGraph(ctx context.Context, request ContextGraphActivityRequest) (ContextGraphActivityResult, error) {
	bundle, e := a.load(request)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	stop := contextGraphHeartbeat(ctx, request, "verify")
	defer stop()
	receipt, e := a.client.VerifyContextGraph(ctx, bundle)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	return contextGraphRuntime(ctx, request.BundleSHA256, receipt, bundle), nil
}

func (a ContextGraphActivities) saveTraversal(ref string, view surrealsink.ContextGraphReadback) (string, error) {
	path, e := a.filePath(ref)
	if e != nil {
		return "", contextGraphBadInput()
	}
	parent, e := canonicalDirectory(filepath.Dir(path))
	if e != nil || parent != filepath.Dir(path) {
		return "", contextGraphBadInput()
	}
	relative, e := filepath.Rel(a.root, path)
	if e != nil {
		return "", contextGraphBadInput()
	}
	raw, e := json.Marshal(view)
	if e != nil || len(raw)+1 > 2*surrealsink.MaxContextGraphBundleBytes+(64<<10) {
		return "", errors.New("private graph traversal exceeds artifact bound")
	}
	raw = append(raw, '\n')
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e == nil {
		n, writeErr := file.Write(raw)
		syncErr := file.Sync()
		closeErr := file.Close()
		if writeErr != nil || n != len(raw) || syncErr != nil || closeErr != nil {
			return "", errors.New("private graph traversal write incomplete; retained bytes preserved")
		}
	} else if !errors.Is(e, os.ErrExist) {
		return "", errors.New("private graph traversal exclusive create failed")
	}
	savedFile, e := toolkitPlacementOpenLocal(a.root, filepath.ToSlash(relative))
	if e != nil {
		return "", errors.New("private graph traversal file guard rejected output")
	}
	defer savedFile.Close()
	info, e := savedFile.Stat()
	if e != nil || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return "", errors.New("private graph traversal output permissions invalid")
	}
	saved, e := io.ReadAll(io.LimitReader(savedFile, int64(len(raw))+1))
	if e != nil || !bytes.Equal(saved, raw) {
		return "", errors.New("private graph traversal output conflicts; existing bytes preserved")
	}
	sum := sha256.Sum256(saved)
	return hex.EncodeToString(sum[:]), nil
}

// TraverseContextGraph verifies source-cited stored links and saves private readback externally.
// Input is a sealed bundle pin, bounded node/hops and output URI under the managed root;
// output contains only artifact/checkpoint references, hashes, counts and actual runtime IDs.
// Effects read the graph and exclusively create/replay a mode-0600 private JSON file;
// heartbeat/cancellation use the Activity context. Pick for inspection or downstream analysis.
func (a ContextGraphActivities) TraverseContextGraph(ctx context.Context, request ContextGraphTraversalRequest) (ContextGraphActivityResult, error) {
	bundle, e := a.load(request.ContextGraphActivityRequest)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	if !contextGraphID(request.NodeID) || request.Hops < 1 || request.Hops > 2 {
		return ContextGraphActivityResult{}, contextGraphBadInput()
	}
	if _, e = a.filePath(request.OutputRef); e != nil {
		return ContextGraphActivityResult{}, contextGraphBadInput()
	}
	stop := contextGraphHeartbeat(ctx, request.ContextGraphActivityRequest, "traverse")
	defer stop()
	view, e := a.client.TraverseContextGraph(ctx, bundle.Scope, bundle.GenerationID, request.NodeID, request.Hops)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	hash, e := a.saveTraversal(request.OutputRef, view)
	if e != nil {
		return ContextGraphActivityResult{}, e
	}
	result := contextGraphRuntime(ctx, request.BundleSHA256, view.Receipt, bundle)
	result.OutputRef = request.OutputRef
	result.OutputSHA256 = hash
	result.TraversalNodeCount = len(view.Bundle.Nodes)
	result.TraversalEdgeCount = len(view.Bundle.Edges)
	return result, nil
}
