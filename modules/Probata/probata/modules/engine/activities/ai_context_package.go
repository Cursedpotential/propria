// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
)

const AIContextPackageActivityName = "retain_ai_context_package_activity"
const aiContextPackagePrefix = "consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/"
const aiContextPackageMaxFiles = 64
const aiContextPackageMaxBytes = 128 << 20

// ContextPackageMetadataStore records only exact package pins in existing context metadata.
// Inputs: registered source and verified package summary. Outputs: error or nil. Effects: SQL metadata writes.
// Choose rather than placing AI records in PostgreSQL or fabricating a Case Bible catalog occurrence.
type ContextPackageMetadataStore interface {
	RecordContextPackage(context.Context, proffer.ContextPackageRequest, proffer.ContextPackageResult) error
	RecordContextPackageCatalog(context.Context, proffer.ContextPackageRequest, proffer.ContextPackageResult) error
}

// AIContextPackageActivities adapts existing placement to a complete native chat source unit.
// Inputs: existing mounted placement group and metadata store. Outputs: one atomic package Activity.
// Effects: none until called; choose beside the Markdown-only work-product import without widening its contract.
type AIContextPackageActivities struct {
	Placement *AIWorkproductPlacementActivities
	Metadata  ContextPackageMetadataStore
	Catalog   func(context.Context) (ContextPackageCatalog, func(), error)
}

type aiContextPackageSource struct {
	SourceRef         string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id"`
	PackageRef        string `json:"package_ref"`
	SourceFormat      string `json:"source_format"`
}
type aiContextPackageBundle struct {
	ContractVersion string                 `json:"contract_version"`
	Stage           string                 `json:"stage"`
	Source          aiContextPackageSource `json:"source"`
	OriginalRef     string                 `json:"original_ref"`
	OriginalSHA256  string                 `json:"original_sha256"`
	PreparedRef     string                 `json:"prepared_ref"`
	WorkProducts    []struct {
		FileRef string `json:"file_ref"`
		Content string `json:"content"`
	} `json:"work_products"`
}
type aiContextPackageMember struct {
	Ref    proffer.Ref `json:"source_ref"`
	Path   string      `json:"path"`
	SHA256 string      `json:"sha256"`
	Bytes  int64       `json:"bytes"`
}
type aiContextPackageManifest struct {
	Contract      string                        `json:"contract"`
	Request       proffer.ContextPackageRequest `json:"request"`
	Members       []aiContextPackageMember      `json:"members"`
	CatalogStatus string                        `json:"catalog_status"`
	CatalogReason string                        `json:"catalog_reason"`
}
type aiContextPackageReceipt struct {
	Manifest aiContextPackageManifest        `json:"manifest"`
	Objects  []ToolkitContentPlacementObject `json:"objects"`
	Result   proffer.ContextPackageResult    `json:"result"`
}

// aiContextMemberPath resolves an existing regular member through the checked placement reader.
// Inputs: mounted root and exact file URI. Outputs: checked file path. Effects: opens/closes one local file.
// Choose over the inventory helper, whose contracts accept directories or JSON receipt paths only.
func aiContextMemberPath(root string, ref proffer.Ref) (string, error) {
	f, err := aiWorkproductLocalRef(root, ref)
	if err != nil {
		return "", err
	}
	name := f.Name()
	return name, f.Close()
}

// readAIContextMember reads one complete mounted file under the existing scratch boundary.
// Inputs: root, URI and hard ceiling. Outputs: unchanged bytes and size/hash. Effects: bounded local read only.
// Choose instead of following arbitrary file refs or buffering an entire package at once.
func readAIContextMember(ctx context.Context, root string, ref proffer.Ref, name string) (aiContextPackageMember, []byte, error) {
	f, err := aiWorkproductLocalRef(root, ref)
	if err != nil {
		return aiContextPackageMember{}, nil, err
	}
	defer f.Close()
	b, err := aiWorkproductRead(ctx, f, 32<<20, nil)
	if err != nil || len(b) == 0 {
		return aiContextPackageMember{}, nil, errors.Join(err, errors.New("package member empty or unreadable"))
	}
	return aiContextPackageMember{Ref: ref, Path: name, SHA256: digestBytes(b), Bytes: int64(len(b))}, b, nil
}

// inspectAIContextPackage derives a closed complete-member manifest from existing sealed bundles.
// Inputs: checked scratch root and request. Outputs: manifest and source directory. Effects: local reads only.
// Choose before any remote writes; missing/escaping members fail rather than producing a truncated package.
func inspectAIContextPackage(ctx context.Context, root string, in proffer.ContextPackageRequest) (aiContextPackageManifest, string, error) {
	m := aiContextPackageManifest{Contract: "ai-context-package-v1", Request: in, CatalogStatus: "pending", CatalogReason: "native_context_catalog_adapter_unavailable"}
	if in.RequestID == "" || in.WorkflowID == "" || in.RunID == "" || in.SourceVersionRef == "" || in.RegistrationReceiptRef == "" || in.SourceRef == "" || in.PreparedRef == "" || in.WorkProductsRef == "" {
		return m, "", errors.New("package requires registered source and complete preparation/work-product refs")
	}
	preparedPath, err := aiContextMemberPath(root, proffer.Ref(in.PreparedRef))
	if err != nil {
		return m, "", err
	}
	sourceRoot := filepath.Dir(preparedPath)
	seen := map[string]bool{}
	var total int64
	add := func(ref, name string) ([]byte, error) {
		p, e := aiContextMemberPath(root, proffer.Ref(ref))
		if e != nil {
			return nil, e
		}
		rel, e := filepath.Rel(sourceRoot, p)
		if e != nil || !toolkitPlacementPath(filepath.ToSlash(rel)) || filepath.ToSlash(rel) != name || seen[name] || len(m.Members) >= aiContextPackageMaxFiles {
			return nil, errors.New("package member outside sealed unit, duplicated or over 64-file ceiling")
		}
		member, b, e := readAIContextMember(ctx, root, proffer.Ref(ref), name)
		if e != nil {
			return nil, e
		}
		total += member.Bytes
		if total > aiContextPackageMaxBytes {
			return nil, errors.New("complete package exceeds 128 MiB ceiling")
		}
		seen[name] = true
		m.Members = append(m.Members, member)
		return b, nil
	}
	stages := []struct{ ref, name, stage string }{{in.PreparedRef, "prepared.json", "prepared"}, {in.WorkProductsRef, "work_products.json", "work_products"}, {in.CandidatesRef, "candidates.json", "candidates"}, {in.EmbeddingsRef, "embedded.json", "embedded"}, {in.PublicationRef, "published.json", "published"}, {in.VerificationRef, "verified.json", "verified"}}
	var prepared, works aiContextPackageBundle
	for _, s := range stages {
		if s.ref == "" {
			continue
		}
		b, e := add(s.ref, s.name)
		if e != nil {
			return m, "", e
		}
		var bundle aiContextPackageBundle
		if e = json.Unmarshal(b, &bundle); e != nil || bundle.ContractVersion != "ai-context-v1" || bundle.Stage != s.stage || bundle.Source.SourceRef != in.SourceRef || bundle.Source.ProviderVersionID != in.ProviderVersionID || bundle.Source.PackageRef != in.PackageRef || bundle.Source.SourceFormat != in.SourceFormat {
			return m, "", errors.New("package bundle source/stage differs from registered request")
		}
		if s.stage == "prepared" {
			prepared = bundle
		}
		if s.stage == "work_products" {
			works = bundle
		}
	}
	if works.PreparedRef != in.PreparedRef {
		return m, "", errors.New("work-products predecessor differs")
	}
	original := "original.json"
	if in.SourceFormat == "gemini_markdown" {
		original = "original.md"
	}
	if _, err = add(prepared.OriginalRef, original); err != nil {
		return m, "", err
	}
	if !validSHA256(prepared.OriginalSHA256) || m.Members[len(m.Members)-1].SHA256 != prepared.OriginalSHA256 {
		return m, "", errors.New("exact original bytes differ from prepared actual original SHA-256")
	}
	for _, w := range works.WorkProducts {
		p, e := aiContextMemberPath(root, proffer.Ref(w.FileRef))
		if e != nil {
			return m, "", e
		}
		rel, e := filepath.Rel(sourceRoot, p)
		name := filepath.ToSlash(rel)
		if e != nil || !strings.HasPrefix(name, "created_works/") {
			return m, "", errors.New("created work is outside sealed created_works directory")
		}
		b, e := add(w.FileRef, name)
		if e != nil {
			return m, "", e
		}
		if string(b) != w.Content {
			return m, "", errors.New("complete created work differs from its native work-product JSON")
		}
	}
	return m, sourceRoot, nil
}

// verifyAIContextObject independently reads one PUT-returned version and checks concurrent versions.
// Inputs: existing placement store and pinned mapping. Outputs: error or verified mapping. Effects: remote reads only.
// Choose for native JSON/text packages because the Markdown sibling fixes a different destination prefix.
func verifyAIContextObject(ctx context.Context, store toolkitPlacementStore, obj *ToolkitContentPlacementObject) error {
	if !strings.HasPrefix(obj.ObjectKey, aiContextPackagePrefix) || !validToolkitVersionID(obj.VersionID) || !validSHA256(obj.SHA256) || obj.Bytes <= 0 || obj.Bytes > 32<<20 {
		return errors.New("invalid or over-budget native package object pin")
	}
	r, err := store.OpenVersion(ctx, "salem-data", obj.ObjectKey, obj.VersionID)
	if err != nil {
		return err
	}
	sha, n, err := streamToolkitDigest(io.LimitReader(r, obj.Bytes+1), obj.Bytes, func() error { return ctx.Err() })
	err = errors.Join(err, r.Close())
	if err != nil {
		return err
	}
	if sha != obj.SHA256 || n != obj.Bytes {
		return errors.New("native package exact-version readback differs")
	}
	obj.AfterVersions, err = store.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	head, err := store.HeadVersion(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	if err = toolkitPlacementConflictChecks(obj.BeforeVersions, obj.AfterVersions, obj.VersionID, head.VersionID); err != nil {
		return err
	}
	obj.LatestVersionID = head.VersionID
	obj.Status = "verified"
	return nil
}

// RetainAIContextPackage retains the original, full native JSON and complete created works together.
// Inputs: existing prepared/work-product refs and any completed later bundles. Outputs: exact-version manifest/readback pins.
// Effects: bounded versioned B2 writes, exclusive local receipts and compact existing PG metadata; never source rewrites.
// Choose for context-first native AI imports; catalog remains explicitly pending until its native adapter is admitted.
func (a AIContextPackageActivities) RetainAIContextPackage(ctx context.Context, in proffer.ContextPackageRequest) (proffer.ContextPackageResult, error) {
	if a.Placement == nil || a.Metadata == nil {
		return proffer.ContextPackageResult{}, errors.New("native package placement/metadata adapters unavailable")
	}
	root, err := canonicalDirectory(a.Placement.AllowedRoot)
	if err != nil {
		return proffer.ContextPackageResult{}, err
	}
	m, sourceRoot, err := inspectAIContextPackage(ctx, root, in)
	if err != nil {
		return proffer.ContextPackageResult{}, err
	}
	manifestBytes, err := json.Marshal(m)
	if err != nil {
		return proffer.ContextPackageResult{}, err
	}
	manifestSHA := digestBytes(manifestBytes)
	unit := "context-" + manifestSHA
	// Source/run/content identity is in the manifest; unchanged retries choose the same collision-safe unit.
	base := filepath.Join(sourceRoot, "package-"+manifestSHA)
	ref := toolkitFileRef(base + ".receipt.json")
	store, err := a.Placement.aiWorkproductStore()
	if err != nil {
		return proffer.ContextPackageResult{}, err
	}
	var saved aiContextPackageReceipt
	if _, e := os.Lstat(base + ".receipt.json"); e == nil {
		sha, e := aiWorkproductReadJSON(ctx, root, ref, "", &saved)
		if e != nil {
			return saved.Result, e
		}
		if !saved.Result.Complete || !reflect.DeepEqual(saved.Manifest, m) || len(saved.Objects) != len(m.Members)+1 {
			return saved.Result, errors.New("package receipt incomplete or mapping differs")
		}
		for i := range saved.Objects {
			if e = verifyAIContextObject(ctx, store, &saved.Objects[i]); e != nil {
				return saved.Result, e
			}
		}
		saved.Result.ReceiptRef = string(ref)
		saved.Result.ReceiptSHA256 = sha
		return saved.Result, a.Metadata.RecordContextPackage(ctx, in, saved.Result)
	} else if !errors.Is(e, os.ErrNotExist) {
		return saved.Result, e
	}
	if _, err = aiWorkproductWriteExclusive(base+".attempt.json", m); err != nil {
		return saved.Result, errors.New("package attempt already exists or cannot be retained; uncertain writes are not replayed")
	}
	result := proffer.ContextPackageResult{ReceiptRef: string(ref), CatalogStatus: "pending"}
	receipt := aiContextPackageReceipt{Manifest: m}
	pulse := func() error {
		if a.Placement.Heartbeat != nil {
			a.Placement.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "ai-context-package"})
		}
		return ctx.Err()
	}
	members := append([]aiContextPackageMember{}, m.Members...)
	members = append(members, aiContextPackageMember{Path: "package-manifest.json", SHA256: manifestSHA, Bytes: int64(len(manifestBytes))})
	for i, member := range members {
		var b []byte
		if member.Path == "package-manifest.json" {
			b = manifestBytes
		} else {
			var current aiContextPackageMember
			current, b, err = readAIContextMember(ctx, root, member.Ref, member.Path)
			if err == nil && current != member {
				err = errors.New("sealed package member changed before upload")
			}
		}
		if err != nil {
			return result, err
		}
		obj := ToolkitContentPlacementObject{UnitID: unit, SourceRef: member.Ref, Path: member.Path, ObjectKey: aiContextPackagePrefix + unit + "/" + member.Path, SHA256: member.SHA256, Bytes: member.Bytes}
		media := "text/plain; charset=utf-8"
		if strings.HasSuffix(member.Path, ".json") {
			media = "application/json"
		}
		if strings.HasSuffix(member.Path, ".md") {
			media = "text/markdown; charset=utf-8"
		}
		err = aiWorkproductCopyMedia(ctx, store, b, &obj, pulse, func() error {
			_, e := aiWorkproductWriteExclusive(fmt.Sprintf("%s.object-%02d.json", base, i), obj)
			return e
		}, media)
		if err != nil {
			return result, err
		}
		if err = verifyAIContextObject(ctx, store, &obj); err != nil {
			return result, err
		}
		receipt.Objects = append(receipt.Objects, obj)
		result.Files++
		result.Bytes += obj.Bytes
		if member.Path == "package-manifest.json" {
			result.ManifestRef = string(obj.ObjectRef)
			result.ManifestSHA256 = member.SHA256
		}
	}
	result.Complete = true
	receipt.Result = result
	// This exclusive receipt is also a durable pending-catalog queue item with exact object/version pins.
	result.ReceiptSHA256, err = aiWorkproductWriteExclusive(base+".receipt.json", receipt)
	if err != nil {
		return result, err
	}
	if err = a.Metadata.RecordContextPackage(ctx, in, result); err != nil {
		return result, err
	}
	return result, nil
}
