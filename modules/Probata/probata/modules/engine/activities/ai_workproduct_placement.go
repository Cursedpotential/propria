// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AIWorkproductPlacementWorkflowName = "AIWorkproductPlacementWorkflow"
	AIWorkproductInspectActivityName   = "ai_workproduct_inspect_activity"
	AIWorkproductCopyActivityName      = "ai_workproduct_copy_activity"
	AIWorkproductReadbackActivityName  = "ai_workproduct_readback_activity"
	aiWorkproductPrefix                = "consignatio/casevault/KnowledgeBase/ai-chats/_derived/misc/"
)

// AIWorkproductPlacementInput pins a reviewed Markdown manifest and exclusive receipt base.
// Inputs: mounted manifest reference/hash and receipt base; outputs: three separately pinned stage receipts. Effects: none as data; choose for bounded AI work products rather than the legal-library or ZIP-recovery siblings.
type AIWorkproductPlacementInput struct {
	ManifestRef    proffer.Ref `json:"manifest_ref"`
	ManifestSHA256 string      `json:"manifest_sha256"`
	ReceiptRef     proffer.Ref `json:"receipt_ref"`
}

// AIWorkproductManifest preserves one complete reviewed source unit without inferring dates or provider identity.
// Inputs: source directory, portable unit name, explicit provider and all reviewed files; outputs: exact placement plan. Effects: none; choose explicit mapping over recursive discovery or inferred catalog identities.
type AIWorkproductManifest struct {
	SourceUnit       string              `json:"source_unit"`
	SourceRef        proffer.Ref         `json:"source_ref"`
	Provider         string              `json:"provider"`
	ProvenanceRef    proffer.Ref         `json:"provenance_ref"`
	ProvenanceSHA256 string              `json:"provenance_sha256"`
	Files            []AIWorkproductFile `json:"files"`
}

// AIWorkproductFile maps one transported source path to its complete supplied Markdown name.
// Inputs: relative source path, unchanged destination name, SHA-256 and positive bytes; outputs: source identity. Effects: none; choose exact names instead of slugging, truncating or renaming originals.
type AIWorkproductFile struct {
	SourcePath string `json:"source_path"`
	Name       string `json:"name"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes"`
}

// AIWorkproductStageInput binds an independent stage to the prior durable receipt.
// Inputs: original request and optional predecessor reference/hash; outputs: stage request. Effects: none; choose references instead of passing source bytes or manifests through Temporal history.
type AIWorkproductStageInput struct {
	Input          AIWorkproductPlacementInput `json:"input"`
	PreviousRef    proffer.Ref                 `json:"previous_ref,omitempty"`
	PreviousSHA256 string                      `json:"previous_sha256,omitempty"`
}

// AIWorkproductSummary carries only bounded receipt metadata through workflow history.
// Inputs: persisted stage; outputs: receipt pin, object count and completion state. Effects: none; choose over claiming ingestion or catalog publication.
type AIWorkproductSummary struct {
	ReceiptRef    proffer.Ref `json:"receipt_ref"`
	ReceiptSHA256 string      `json:"receipt_sha256"`
	Objects       int         `json:"objects"`
	Complete      bool        `json:"complete"`
}

// AIWorkproductReceipt retains the full reviewed mapping and every returned provider version.
// Inputs: stage request, original manifest and object observations; outputs: durable physical-placement evidence. Effects: none as data; choose for later ingestion/catalog work, whose status remains explicitly pending.
type AIWorkproductReceipt struct {
	Stage           string                          `json:"stage"`
	Request         AIWorkproductStageInput         `json:"request"`
	Manifest        AIWorkproductManifest           `json:"manifest"`
	Objects         []ToolkitContentPlacementObject `json:"objects"`
	Complete        bool                            `json:"complete"`
	Error           string                          `json:"error,omitempty"`
	IngestionStatus string                          `json:"ingestion_status"`
	CatalogStatus   string                          `json:"catalog_status"`
}

// AIWorkproductPlacementActivities reuses the worker's mounted scratch root and configured B2 adapter.
// Inputs: allowed root, existing resolver and optional heartbeat; outputs: independent inspection/copy/readback Activities. Effects: retained remote versions and exclusive scratch receipts only; choose beside toolkit placement without broadening its legal destinations.
type AIWorkproductPlacementActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewAIWorkproductPlacementActivities constructs the group without reading sources or opening stores.
// Inputs: existing DeriveScratchDir and object-store resolver; outputs: separately registered Activity group. Effects: none until invoked; choose this seam over a second storage runtime.
func NewAIWorkproductPlacementActivities(root string, stores func(string) (smsthreads.ObjectStore, error)) *AIWorkproductPlacementActivities {
	return &AIWorkproductPlacementActivities{AllowedRoot: root, Stores: stores, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// AIWorkproductPlacementWorkflow sequences inspection, copy and independent exact-version readback.
// Inputs: reviewed manifest pin and receipt base; outputs: verified receipt pin. Effects: three named Activities with reference-only history; choose for physical placement, leaving ingestion and catalog admission pending.
func AIWorkproductPlacementWorkflow(ctx workflow.Context, in AIWorkproductPlacementInput) (AIWorkproductSummary, error) {
	if !validSHA256(in.ManifestSHA256) || in.ManifestRef == "" || in.ReceiptRef == "" || in.ManifestRef == in.ReceiptRef {
		return AIWorkproductSummary{}, temporal.NewNonRetryableApplicationError("invalid manifest/receipt pin", "AIWorkproductInvalid", nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Minute, ScheduleToCloseTimeout: 30 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2}})
	request := AIWorkproductStageInput{Input: in}
	var out AIWorkproductSummary
	for _, name := range []string{AIWorkproductInspectActivityName, AIWorkproductCopyActivityName, AIWorkproductReadbackActivityName} {
		if err := workflow.ExecuteActivity(ctx, name, request).Get(ctx, &out); err != nil {
			return out, err
		}
		if !out.Complete || !validSHA256(out.ReceiptSHA256) {
			return out, temporal.NewNonRetryableApplicationError("stage has no complete pinned receipt", "AIWorkproductIncomplete", nil)
		}
		request.PreviousRef, request.PreviousSHA256 = out.ReceiptRef, out.ReceiptSHA256
	}
	return out, nil
}

// InspectAIWorkproduct verifies every complete source stream as nonempty UTF-8 with exact SHA-256 and size.
// Inputs: pinned manifest only; outputs: exclusive inspection receipt pin. Effects: mounted read-only source reads and a scratch receipt; choose before copy without parsing, extracting dates or writing source metadata.
func (a *AIWorkproductPlacementActivities) InspectAIWorkproduct(ctx context.Context, in AIWorkproductStageInput) (AIWorkproductSummary, error) {
	return a.runAIWorkproductStage(ctx, "inspect", in)
}

// CopyAIWorkproduct retains authenticated Markdown under the fixed Case Bible AI-derived namespace.
// Inputs: pinned inspection receipt; outputs: exclusive copy receipt retaining PUT response versions and race observations. Effects: one fresh retained PUT per absent target, no delete/overwrite/recovery guess; choose before the independent readback Activity.
func (a *AIWorkproductPlacementActivities) CopyAIWorkproduct(ctx context.Context, in AIWorkproductStageInput) (AIWorkproductSummary, error) {
	return a.runAIWorkproductStage(ctx, "copy", in)
}

// ReadbackAIWorkproduct independently hashes exact retained versions and checks intervening object versions.
// Inputs: pinned copy receipt; outputs: exclusive verified physical-placement receipt. Effects: remote reads only plus scratch receipt; choose over HEAD/ETag proof or unversioned current-key reads.
func (a *AIWorkproductPlacementActivities) ReadbackAIWorkproduct(ctx context.Context, in AIWorkproductStageInput) (AIWorkproductSummary, error) {
	return a.runAIWorkproductStage(ctx, "readback", in)
}

// aiWorkproductManifestValid applies fixed limits and rejects ambiguous names before any source or B2 reads.
// Inputs: reviewed manifest; outputs: error unless at most 16 files, 1 MiB each and 16 MiB total. Effects: none; choose strict validation instead of normalizing paths or guessing completeness.
func aiWorkproductManifestValid(m AIWorkproductManifest) error {
	if !toolkitPlacementPath(m.SourceUnit) || strings.Contains(m.SourceUnit, "/") || len(m.SourceUnit) > 128 || len(m.Provider) < 1 || len(m.Provider) > 64 || strings.TrimSpace(m.Provider) != m.Provider || strings.ContainsAny(m.Provider, "\r\n\x00") || !validSHA256(m.ProvenanceSHA256) || len(m.ProvenanceRef) < 1 || len(m.ProvenanceRef) > 4096 || len(m.SourceRef) < 1 || len(m.SourceRef) > 4096 || len(m.Files) < 1 || len(m.Files) > 16 {
		return errors.New("invalid source unit/provider/provenance pin or 16-file ceiling")
	}
	sources, names := map[string]bool{}, map[string]bool{}
	var total int64
	for _, f := range m.Files {
		if !toolkitPlacementPath(f.SourcePath) || len(f.SourcePath) > 512 || !toolkitPlacementPath(f.Name) || strings.Contains(f.Name, "/") || len(f.Name) > 255 || !strings.EqualFold(filepath.Ext(f.Name), ".md") || sources[f.SourcePath] || names[f.Name] || !validSHA256(f.SHA256) || f.Bytes < 1 || f.Bytes > 1<<20 {
			return errors.New("invalid/duplicate reviewed Markdown name, path or file pin")
		}
		sources[f.SourcePath], names[f.Name] = true, true
		total += f.Bytes
	}
	if total > 16<<20 {
		return errors.New("16 MiB batch ceiling exceeded")
	}
	return nil
}

// aiWorkproductLocalRef opens a checked file reference beneath the existing mounted root.
// Inputs: canonical allowed root and absolute file URI; outputs: nonlinked read-only regular file handle. Effects: metadata/source reads; choose the placement sibling's checked open over unchecked JSON/file helpers.
func aiWorkproductLocalRef(root string, ref proffer.Ref) (*os.File, error) {
	u, err := url.Parse(string(ref))
	if err != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("expected mounted file reference")
	}
	p := filepath.FromSlash(u.Path)
	if len(p) > 1 && p[0] == os.PathSeparator && filepath.VolumeName(p[1:]) != "" {
		p = p[1:]
	}
	if !filepath.IsAbs(p) {
		return nil, errors.New("mounted reference must be absolute")
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || !toolkitPlacementPath(filepath.ToSlash(rel)) {
		return nil, errors.New("file reference escaped mounted root")
	}
	return toolkitPlacementOpenLocal(root, filepath.ToSlash(rel))
}

// aiWorkproductRead streams bounded metadata or Markdown while observing cancellation.
// Inputs: context, complete reader and hard ceiling; outputs: all bounded bytes or a visible read error. Effects: reads only; choose for small metadata/Markdown rather than unbounded buffering.
func aiWorkproductRead(ctx context.Context, r io.Reader, limit int64, pulse func() error) ([]byte, error) {
	var b bytes.Buffer
	buf := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if pulse != nil {
			if err := pulse(); err != nil {
				return nil, err
			}
		}
		n, err := r.Read(buf)
		if n > 0 {
			if int64(b.Len()+n) > limit {
				return nil, errors.New("bounded stream exceeded ceiling")
			}
			b.Write(buf[:n])
		}
		if err == io.EOF {
			return b.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
}

// aiWorkproductReadJSON authenticates a complete mounted metadata reference before strict decoding.
// Inputs: root, reference, optional hash and context; outputs: decoded bounded metadata and actual hash. Effects: local read only; choose for manifest/predecessor/replay pins.
func aiWorkproductReadJSON(ctx context.Context, root string, ref proffer.Ref, pin string, target any) (string, error) {
	f, err := aiWorkproductLocalRef(root, ref)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := aiWorkproductRead(ctx, f, 256<<10, nil)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", errors.New("metadata manifest/receipt must contain complete UTF-8")
	}
	sha := digestBytes(b)
	if pin != "" && sha != pin {
		return "", errors.New("metadata receipt/manifest SHA-256 mismatch")
	}
	return sha, toolkitPlacementDecode(b, target)
}

// aiWorkproductSource authenticates one whole immutable source before a copy or inspection.
// Inputs: source root, reviewed file and liveness callback; outputs: bounded exact UTF-8 bytes. Effects: read-only source read; choose over reopening unverified original bytes at upload time.
func aiWorkproductSource(ctx context.Context, root string, f AIWorkproductFile, pulse func() error) ([]byte, error) {
	r, err := toolkitPlacementOpenLocal(root, f.SourcePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := aiWorkproductRead(ctx, r, f.Bytes, pulse)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != f.Bytes || digestBytes(b) != f.SHA256 || !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return nil, errors.New("source is not complete pinned nonempty UTF-8 Markdown")
	}
	return b, nil
}

// aiWorkproductStore selects retained-version capabilities from the worker's existing B2 resolver.
// Inputs: existing configured resolver; outputs: bounded placement adapter. Effects: resolver construction only; choose the concrete S3 sibling over unsafe ordinary Put fallback.
func (a *AIWorkproductPlacementActivities) aiWorkproductStore() (toolkitPlacementStore, error) {
	if a.Stores == nil {
		return nil, errors.New("existing B2 resolver required")
	}
	base, err := a.Stores("b2")
	if err != nil {
		return nil, err
	}
	if s, ok := base.(smsthreads.S3Store); ok {
		base = toolkitPlacementS3{s}
	}
	if s, ok := base.(*smsthreads.S3Store); ok && s != nil {
		base = toolkitPlacementS3{*s}
	}
	s, ok := base.(toolkitPlacementStore)
	if !ok {
		return nil, errors.New("B2 resolver lacks retained-version placement capabilities")
	}
	return s, nil
}

// aiWorkproductReceiptPath resolves an exclusive stage receipt outside the source unit.
// Inputs: mounted root, source root and derived file reference; outputs: checked existing parent and absent-or-regular target. Effects: metadata reads only; choose before creating an attempt marker or final receipt.
func aiWorkproductReceiptPath(root, source string, ref proffer.Ref) (string, error) {
	p, err := resolveFileRef(ref, root, false)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(source, p)
	if err != nil {
		return "", err
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("receipt must remain outside unchanged source unit")
	}
	parentRel, err := filepath.Rel(root, filepath.Dir(p))
	if err != nil {
		return "", err
	}
	current := root
	if parentRel != "." {
		for _, part := range strings.Split(filepath.ToSlash(parentRel), "/") {
			current = filepath.Join(current, part)
			info, e := os.Lstat(current)
			if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return "", errors.New("receipt parent must be existing nonlinked mounted directory")
			}
		}
	}
	return p, nil
}

// aiWorkproductWriteExclusive persists and syncs a stage/attempt record without replacing any path.
// Inputs: checked path and bounded metadata; outputs: actual receipt SHA-256. Effects: one exclusive local create; choose instead of overwrite or delete-based retry cleanup.
func aiWorkproductWriteExclusive(path string, v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if len(b) > 256<<10 {
		return "", errors.New("receipt ceiling exceeded")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return "", err
	}
	return digestBytes(b), nil
}

// runAIWorkproductStage applies one independent phase and persists incomplete outcomes before returning.
// Inputs: phase, request and prior pin; outputs: exclusive phase receipt summary. Effects: only the named phase's I/O; choose this shared envelope to make uncertain copy attempts block replay without conflating source inspection with readback.
func (a *AIWorkproductPlacementActivities) runAIWorkproductStage(ctx context.Context, stage string, in AIWorkproductStageInput) (out AIWorkproductSummary, retErr error) {
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return out, err
	}
	if !validSHA256(in.Input.ManifestSHA256) || in.Input.ManifestRef == in.Input.ReceiptRef {
		return out, errors.New("invalid reviewed manifest pin")
	}
	var m AIWorkproductManifest
	if _, err = aiWorkproductReadJSON(ctx, root, in.Input.ManifestRef, in.Input.ManifestSHA256, &m); err != nil {
		return out, err
	}
	if err = aiWorkproductManifestValid(m); err != nil {
		return out, err
	}
	if stage == "inspect" {
		var provenance json.RawMessage
		if _, err = aiWorkproductReadJSON(ctx, root, m.ProvenanceRef, m.ProvenanceSHA256, &provenance); err != nil {
			return out, fmt.Errorf("root transport provenance: %w", err)
		}
	}
	source, err := resolveFileRef(m.SourceRef, root, true)
	if err != nil {
		return out, err
	}
	// Reject symlinks in the source root itself, including links that point inside the allowed mount.
	u, _ := url.Parse(string(m.SourceRef))
	requested := filepath.FromSlash(u.Path)
	if len(requested) > 1 && requested[0] == os.PathSeparator && filepath.VolumeName(requested[1:]) != "" {
		requested = requested[1:]
	}
	if filepath.Clean(requested) != source {
		return out, errors.New("linked or aliased source root refused")
	}
	if stage != "inspect" && stage != "copy" && stage != "readback" {
		return out, errors.New("unknown independent phase")
	}
	out.ReceiptRef = proffer.Ref(string(in.Input.ReceiptRef) + "." + stage + ".json")
	p, err := aiWorkproductReceiptPath(root, source, out.ReceiptRef)
	if err != nil {
		return out, err
	}
	var prior AIWorkproductReceipt
	previousStage := "inspect"
	if stage == "readback" {
		previousStage = "copy"
	}
	if stage == "inspect" {
		if in.PreviousRef != "" || in.PreviousSHA256 != "" {
			return out, errors.New("inspection must have no predecessor")
		}
	} else {
		if in.PreviousRef != proffer.Ref(string(in.Input.ReceiptRef)+"."+previousStage+".json") || !validSHA256(in.PreviousSHA256) {
			return out, errors.New("incorrect predecessor reference/pin")
		}
		if _, err = aiWorkproductReadJSON(ctx, root, in.PreviousRef, in.PreviousSHA256, &prior); err != nil {
			return out, err
		}
		if prior.Stage != previousStage || !prior.Complete || prior.Request.Input != in.Input || !reflect.DeepEqual(prior.Manifest, m) || len(prior.Objects) != len(m.Files) {
			return out, errors.New("predecessor receipt is incomplete or mapping differs")
		}
	}
	if _, err = os.Lstat(p); err == nil {
		var saved AIWorkproductReceipt
		out.ReceiptSHA256, err = aiWorkproductReadJSON(ctx, root, out.ReceiptRef, "", &saved)
		if err != nil {
			return out, err
		}
		if saved.Stage != stage || saved.Request != in || !saved.Complete || !reflect.DeepEqual(saved.Manifest, m) || len(saved.Objects) != len(m.Files) {
			return out, errors.New("exclusive receipt retains incomplete or different attempt; review before a new receipt")
		}
		if stage == "readback" {
			store, e := a.aiWorkproductStore()
			if e != nil {
				return out, e
			}
			for i := range saved.Objects {
				if e = aiWorkproductVerify(ctx, store, m.Files[i], m.SourceUnit, source, &saved.Objects[i], nil); e != nil {
					return out, e
				}
			}
		}
		out.Objects, out.Complete = len(saved.Objects), true
		return out, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return out, err
	}
	r := AIWorkproductReceipt{Stage: stage, Request: in, Manifest: m, IngestionStatus: "pending", CatalogStatus: "pending"}
	if _, err = aiWorkproductWriteExclusive(p+".attempt", r); err != nil {
		return out, fmt.Errorf("exclusive attempt exists or cannot be retained; uncertain attempts require review: %w", err)
	}
	defer func() {
		if retErr != nil {
			r.Error = retErr.Error()
		}
		sha, e := aiWorkproductWriteExclusive(p, r)
		out.ReceiptSHA256 = sha
		out.Objects = len(r.Objects)
		out.Complete = r.Complete && e == nil
		retErr = errors.Join(retErr, e)
	}()
	pulse := func() error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if a.Heartbeat != nil {
			a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "ai-workproduct-" + stage})
		}
		return nil
	}
	var store toolkitPlacementStore
	if stage != "inspect" {
		store, err = a.aiWorkproductStore()
		if err != nil {
			return out, err
		}
	}
	for i, f := range m.Files {
		if err = pulse(); err != nil {
			return out, err
		}
		obj := ToolkitContentPlacementObject{UnitID: m.SourceUnit, SourceRef: toolkitFileRef(filepath.Join(source, filepath.FromSlash(f.SourcePath))), Path: f.Name, ObjectKey: aiWorkproductPrefix + m.SourceUnit + "/" + f.Name, SHA256: f.SHA256, Bytes: f.Bytes, Status: "inspected"}
		if stage == "readback" {
			obj = prior.Objects[i]
		}
		r.Objects = append(r.Objects, obj)
		target := &r.Objects[len(r.Objects)-1]
		if stage == "inspect" {
			_, err = aiWorkproductSource(ctx, source, f, pulse)
		} else if stage == "copy" {
			var b []byte
			b, err = aiWorkproductSource(ctx, source, f, pulse)
			if err == nil {
				err = aiWorkproductCopy(ctx, store, b, target, pulse, func() error {
					checkpoint := AIWorkproductReceipt{Stage: "copy-put-response", Request: in, Manifest: m, Objects: []ToolkitContentPlacementObject{*target}, IngestionStatus: "pending", CatalogStatus: "pending"}
					_, e := aiWorkproductWriteExclusive(fmt.Sprintf("%s.object-%02d.json", p, i+1), checkpoint)
					return e
				})
			}
		} else {
			err = aiWorkproductVerify(ctx, store, f, m.SourceUnit, source, target, pulse)
		}
		if err != nil {
			target.Status = "incomplete"
			target.Error = err.Error()
			return out, err
		}
	}
	r.Complete = true
	return out, nil
}

// aiWorkproductCopy records a fresh retained PUT response and bounded before/after versions.
// Inputs: authenticated source bytes, fixed mapping and exclusive response checkpoint; outputs: returned provider version in the mapping. Effects: one retained B2 PUT only after absence observations plus durable returned-version checkpoint; choose no lost-response recovery and leave byte readback to its sibling Activity.
func aiWorkproductCopy(ctx context.Context, s toolkitPlacementStore, b []byte, obj *ToolkitContentPlacementObject, pulse func() error, checkpoint func() error) error {
	return aiWorkproductCopyMedia(ctx, s, b, obj, pulse, checkpoint, "text/markdown; charset=utf-8")
}

// aiWorkproductCopyMedia reuses exact retained placement for a caller's complete native media type.
// Inputs: existing store, pinned bytes/mapping, checkpoint and media type. Outputs: PUT-returned version.
// Effects: the same bounded absence/race checks and fresh retained PUT; choose for native JSON/text packages.
func aiWorkproductCopyMedia(ctx context.Context, s toolkitPlacementStore, b []byte, obj *ToolkitContentPlacementObject, pulse func() error, checkpoint func() error, media string) error {
	var err error
	obj.BeforeVersions, err = s.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	if err = toolkitPlacementObservations(obj.BeforeVersions); err != nil {
		return err
	}
	if len(obj.BeforeVersions) != 0 {
		return errors.New("prior target versions exist; refusing overwrite")
	}
	_, err = s.HeadVersion(ctx, "salem-data", obj.ObjectKey)
	if !errors.Is(err, smsthreads.ErrRecoveredVersionNotFound) {
		if err == nil {
			return errors.New("target appeared during precheck")
		}
		return err
	}
	if err = pulse(); err != nil {
		return err
	}
	obj.VersionID, err = s.PutRecoveredVersion(ctx, "salem-data", obj.ObjectKey, bytes.NewReader(b), obj.Bytes, media, obj.SHA256, nil)
	if validToolkitVersionID(obj.VersionID) {
		u := url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + obj.ObjectKey, RawQuery: url.Values{"versionId": []string{obj.VersionID}}.Encode()}
		obj.ObjectRef = proffer.Ref(u.String())
		obj.Status = "copied-awaiting-version-observations"
		if checkpoint != nil {
			err = errors.Join(err, checkpoint())
		}
	}
	// Retain available conflict evidence even after an uncertain response; never choose an observed latest version as our own.
	var observeErr error
	obj.AfterVersions, observeErr = s.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
	if head, e := s.HeadVersion(ctx, "salem-data", obj.ObjectKey); e == nil {
		obj.LatestVersionID = head.VersionID
	} else {
		observeErr = errors.Join(observeErr, e)
	}
	if err != nil || observeErr != nil {
		return errors.Join(err, observeErr)
	}
	if !validToolkitVersionID(obj.VersionID) {
		return errors.New("PUT response lacks exact retained version")
	}
	if err = toolkitPlacementConflictChecks(obj.BeforeVersions, obj.AfterVersions, obj.VersionID, obj.LatestVersionID); err != nil {
		return err
	}
	obj.Status = "copied-awaiting-independent-readback"
	return nil
}

// aiWorkproductVerify checks the manifest mapping and independently reads only the PUT-returned version.
// Inputs: reviewed file, source unit and copy mapping; outputs: verified mapping or specific conflict. Effects: exact-version remote GET plus version observations; choose instead of current-key readback or accepting matching unrelated versions.
func aiWorkproductVerify(ctx context.Context, s toolkitPlacementStore, f AIWorkproductFile, unit, source string, obj *ToolkitContentPlacementObject, pulse func() error) error {
	key := aiWorkproductPrefix + unit + "/" + f.Name
	u := url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + key, RawQuery: url.Values{"versionId": []string{obj.VersionID}}.Encode()}
	if obj.UnitID != unit || obj.SourceRef != toolkitFileRef(filepath.Join(source, filepath.FromSlash(f.SourcePath))) || obj.Path != f.Name || obj.ObjectKey != key || obj.SHA256 != f.SHA256 || obj.Bytes != f.Bytes || !validToolkitVersionID(obj.VersionID) || obj.ObjectRef != proffer.Ref(u.String()) || len(obj.BeforeVersions) != 0 {
		return errors.New("copy receipt mapping/version differs from reviewed manifest")
	}
	r, err := s.OpenVersion(ctx, "salem-data", key, obj.VersionID)
	if err != nil {
		return err
	}
	b, err := aiWorkproductRead(ctx, r, f.Bytes, pulse)
	err = errors.Join(err, r.Close())
	if err != nil {
		return err
	}
	if int64(len(b)) != f.Bytes || digestBytes(b) != f.SHA256 || !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return errors.New("independent pinned readback SHA-256/size/UTF-8 mismatch")
	}
	obj.AfterVersions, err = s.PlacementVersions(ctx, "salem-data", key)
	if err != nil {
		return err
	}
	head, err := s.HeadVersion(ctx, "salem-data", key)
	if err != nil {
		return err
	}
	obj.LatestVersionID = head.VersionID
	if err = toolkitPlacementConflictChecks(obj.BeforeVersions, obj.AfterVersions, obj.VersionID, head.VersionID); err != nil {
		return err
	}
	obj.Status = "verified"
	return nil
}
