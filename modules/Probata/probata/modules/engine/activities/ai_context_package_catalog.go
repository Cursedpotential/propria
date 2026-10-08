// Byline: Codex · GPT-6.1-sol · 2026-10-08.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
)

const AIContextCatalogActivityName = "catalog_ai_context_package_activity"

// ContextPackageCatalog is the existing writer's narrow native registration/readback surface.
// Inputs: complete canonical metadata and digest. Outputs: independently verified stored payload.
// Effects: catalog metadata only; choose instead of granting broad writes to existing catalog tables.
type ContextPackageCatalog interface {
	RegisterAIContextPackage(context.Context, []byte, string) ([]byte, error)
}

type aiContextCatalogMember struct {
	Path      string `json:"path"`
	SourceRef string `json:"source_ref"`
	ObjectRef string `json:"object_ref"`
	VersionID string `json:"version_id"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
}
type aiContextCatalogPayload struct {
	Contract          string                   `json:"contract"`
	Operation         string                   `json:"operation"`
	SourceVersionID   string                   `json:"source_version_id"`
	SourceRef         string                   `json:"source_ref"`
	ProviderVersionID string                   `json:"provider_version_id"`
	SourceFormat      string                   `json:"source_format"`
	OriginalSHA256    string                   `json:"original_sha256"`
	PackageRef        string                   `json:"package_ref"`
	ManifestRef       string                   `json:"manifest_ref"`
	ManifestSHA256    string                   `json:"manifest_sha256"`
	ReceiptRef        string                   `json:"receipt_ref"`
	ReceiptSHA256     string                   `json:"receipt_sha256"`
	Members           []aiContextCatalogMember `json:"members"`
}
type aiContextCatalogReceipt struct {
	Payload       aiContextCatalogPayload `json:"payload"`
	PayloadSHA256 string                  `json:"payload_sha256"`
	Status        string                  `json:"status"`
}

// CatalogAIContextPackage registers and independently reads one complete native package catalog unit.
// Inputs: exact physical receipt and registered source. Outputs: separate catalog receipt with complete status.
// Effects: bounded exact-version reads, narrow catalog SQL, exclusive receipt and compact PG status; no raw AI rows.
// Choose after RetainAIContextPackage; unavailable catalog returns an error while its durable pending receipt remains.
func (a AIContextPackageActivities) CatalogAIContextPackage(ctx context.Context, in proffer.ContextCatalogRequest) (proffer.ContextPackageResult, error) {
	result := in.Package
	if a.Placement == nil || a.Metadata == nil || a.Catalog == nil || !result.Complete || !validSHA256(result.ReceiptSHA256) || !validSHA256(result.ManifestSHA256) {
		return result, errors.New("native catalog requires exact complete package and configured narrow writer")
	}
	catalog, closeCatalog, err := a.Catalog(ctx)
	if err != nil {
		return result, err
	}
	if closeCatalog != nil {
		defer closeCatalog()
	}
	if catalog == nil {
		return result, errors.New("native catalog writer unavailable")
	}
	root, err := canonicalDirectory(a.Placement.AllowedRoot)
	if err != nil {
		return result, err
	}
	var physical aiContextPackageReceipt
	if _, err = aiWorkproductReadJSON(ctx, root, proffer.Ref(result.ReceiptRef), result.ReceiptSHA256, &physical); err != nil {
		return result, err
	}
	if !physical.Result.Complete || physical.Manifest.Request != in.Request || physical.Result.ManifestRef != result.ManifestRef || physical.Result.ManifestSHA256 != result.ManifestSHA256 || len(physical.Objects) != result.Files || len(physical.Objects) < 4 || len(physical.Objects) > 65 {
		return result, errors.New("catalog physical receipt/source mapping differs")
	}
	var total int64
	for _, obj := range physical.Objects {
		if obj.Bytes <= 0 || obj.Bytes > 32<<20 {
			return result, errors.New("catalog member readback exceeds file bound")
		}
		total += obj.Bytes
	}
	if total > 129<<20 || total != physical.Result.Bytes || total != result.Bytes {
		return result, errors.New("catalog aggregate readback budget differs")
	}
	store, err := a.Placement.aiWorkproductStore()
	if err != nil {
		return result, err
	}
	payload := aiContextCatalogPayload{Contract: "ai-context-catalog/v1", Operation: "context-" + result.ManifestSHA256, SourceVersionID: in.Request.SourceVersionRef, SourceRef: in.Request.SourceRef, ProviderVersionID: in.Request.ProviderVersionID, SourceFormat: in.Request.SourceFormat, PackageRef: in.Request.PackageRef, ManifestRef: result.ManifestRef, ManifestSHA256: result.ManifestSHA256, ReceiptRef: result.ReceiptRef, ReceiptSHA256: result.ReceiptSHA256}
	original := "original.json"
	if in.Request.SourceFormat == "gemini_markdown" {
		original = "original.md"
	}
	for i := range physical.Objects {
		obj := &physical.Objects[i]
		if err = verifyAIContextObject(ctx, store, obj); err != nil {
			return result, err
		}
		payload.Members = append(payload.Members, aiContextCatalogMember{Path: obj.Path, SourceRef: string(obj.SourceRef), ObjectRef: string(obj.ObjectRef), VersionID: obj.VersionID, SHA256: obj.SHA256, Bytes: obj.Bytes})
		if obj.Path == original {
			payload.OriginalSHA256 = obj.SHA256
		}
	}
	if !validSHA256(payload.OriginalSHA256) {
		return result, errors.New("catalog lacks exact original member")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return result, err
	}
	if len(raw) > 256<<10 {
		return result, errors.New("native catalog metadata exceeds bounded writer payload")
	}
	sha := digestBytes(raw)
	readback, err := catalog.RegisterAIContextPackage(ctx, raw, sha)
	if err != nil {
		return result, err
	}
	var observed aiContextCatalogPayload
	if err = toolkitPlacementDecode(readback, &observed); err != nil || !reflect.DeepEqual(observed, payload) {
		return result, errors.New("native catalog independent stored rows differ from complete package")
	}
	// JSONB may reorder object keys; compare complete typed values, then canonicalize for exact byte digest.
	canonical, err := json.Marshal(observed)
	if err != nil || !bytes.Equal(canonical, raw) {
		return result, errors.New("native catalog canonical readback differs")
	}
	physicalPath, err := aiContextMemberPath(root, proffer.Ref(result.ReceiptRef))
	if err != nil {
		return result, err
	}
	file := strings.TrimSuffix(physicalPath, ".json") + ".catalog.json"
	ref := toolkitFileRef(file)
	receipt := aiContextCatalogReceipt{Payload: payload, PayloadSHA256: sha, Status: "complete"}
	if _, e := os.Lstat(file); e == nil {
		var existing aiContextCatalogReceipt
		result.CatalogReceiptSHA256, err = aiWorkproductReadJSON(ctx, root, ref, "", &existing)
		if err == nil && !reflect.DeepEqual(existing, receipt) {
			err = errors.New("native catalog receipt differs from immutable package")
		}
	} else if errors.Is(e, os.ErrNotExist) {
		result.CatalogReceiptSHA256, err = aiWorkproductWriteExclusive(file, receipt)
	} else {
		err = e
	}
	if err != nil {
		return result, err
	}
	result.CatalogStatus = "complete"
	result.CatalogReceiptRef = string(ref)
	if err = a.Metadata.RecordContextPackageCatalog(ctx, in.Request, result); err != nil {
		return result, err
	}
	return result, nil
}
