package runtimeapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// readContextReviewBundle reads only workflow-returned immutable files under the existing AI roots.
// Inputs: actual stage reference and configured roots. Output: full JSON object; effects: bounded file read.
// Choose for review content rather than exposing arbitrary browser-selected file paths or normalized IDs.
func readContextReviewBundle(ref string) (map[string]any, error) {
	u, e := url.Parse(ref)
	if e != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("review bundle must be an actual local stage reference")
	}
	p, e := filepath.EvalSymlinks(filepath.FromSlash(u.Path))
	if e != nil {
		return nil, e
	}
	allowed := false
	for _, pair := range [][2]string{{"AI_CONTEXT_ROOT", "/data/proffer/derive-scratch/ai-context"}, {"AI_CONTENT_ROOT", "/data/proffer/derive-scratch/ai-content"}} {
		root := os.Getenv(pair[0])
		if root == "" {
			root = pair[1]
		}
		root, e = filepath.EvalSymlinks(root)
		if e != nil {
			continue
		}
		rel, e := filepath.Rel(root, p)
		if e == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			allowed = true
		}
	}
	if !allowed {
		return nil, errors.New("review bundle outside configured AI derivative roots")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > 64<<20 {
		return nil, errors.New("review bundle exceeds its explicit 64 MiB bound")
	}
	var value map[string]any
	e = json.Unmarshal(raw, &value)
	if e != nil || value == nil {
		return nil, errors.New("invalid retained review bundle")
	}
	return value, nil
}

// contextReviewArrays selects actual retained artifact arrays without generating graph relations.
// Inputs: stage bundle and section. Outputs: original rows for that section; effects: none.
// Choose to show native fields losslessly; interpretive kinds remain proposals until review commit.
func contextReviewArrays(bundle map[string]any, section string) []any {
	array := func(key string) []any { v, _ := bundle[key].([]any); return v }
	if strings.HasPrefix(section, "native:") {
		return array(strings.TrimPrefix(section, "native:"))
	}
	switch section {
	case "chunks":
		if bundle["stage"] == "prepared" {
			return array("chunks")
		}
	case "records", "timeline":
		if bundle["stage"] == "prepared" {
			return array("records")
		}
	case "documents":
		return array("work_products")
	case "edges":
		return array("edges")
	case "embeddings":
		return array("vectors")
	case "entities", "events", "locations", "strategy", "proposals":
		rows := []any{}
		for _, raw := range array("candidates") {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			kind, _ := row["kind"].(string)
			if section == "proposals" || (section == "entities" && kind == "entity") || (section == "events" && kind == "event") || (section == "locations" && (kind == "location" || row["entity_type"] == "location")) || (section == "strategy" && (kind == "strategy" || kind == "history")) {
				rows = append(rows, raw)
			}
		}
		return rows
	}
	return nil
}

// packageContent reads all retained AI stage artifacts for one actor-owned context workflow.
// Inputs: workflow ID, section and integer offset. Output: bounded rows, counts and immutable stage metadata.
// Effects: Temporal status plus read-only derivative files; no source/graph/model/canonical writes.
func (h *ContextSourceHTTPHandler) packageContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("workflow_id")
	if !strings.HasPrefix(id, proffer.ContextWorkflowIDPrefix) || len(id) > 160 {
		previewError(w, 404, errors.New("unknown context workflow"))
		return
	}
	actor, _, e := authenticatedActor(r)
	if e != nil {
		previewError(w, 401, e)
		return
	}
	status, e := h.workflows.Status(r.Context(), id)
	if e != nil {
		previewError(w, 503, e)
		return
	}
	if actor != status.ActorSubjectUID {
		previewError(w, 404, errors.New("unknown context workflow"))
		return
	}
	section := r.URL.Query().Get("section")
	if section == "" {
		section = "allmetadata"
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, e = strconv.Atoi(raw)
		if e != nil || offset < 0 {
			previewError(w, 422, errors.New("invalid package offset"))
			return
		}
	}
	keys := []string{"allmetadata", "records", "chunks", "events", "timeline", "entities", "locations", "strategy", "documents", "proposals", "edges", "embeddings"}
	supported := false
	for _, key := range keys {
		if key == section {
			supported = true
		}
	}
	if !supported && !strings.HasPrefix(section, "native:") {
		previewError(w, 422, errors.New("unknown package section"))
		return
	}
	refs := []string{status.PreparedRef, status.WorkProductsRef, status.CandidatesRef, status.EmbeddingsRef, status.PublicationRef, status.VerificationRef}
	counts := map[string]int{}
	available := map[string]bool{}
	rows := []any{}
	metadata := []any{}
	var source any
	seen := map[string]bool{}
	for _, ref := range refs {
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		bundle, err := readContextReviewBundle(ref)
		if err != nil {
			previewError(w, 503, errors.New("retained AI package bundle unavailable: "+err.Error()))
			return
		}
		if source == nil {
			source = bundle["source"]
		} else if !reflect.DeepEqual(source, bundle["source"]) {
			previewError(w, 409, errors.New("package stage source pins disagree"))
			return
		}
		meta := map[string]any{"bundle_ref": ref}
		for key, value := range bundle {
			if array, ok := value.([]any); ok {
				nativeKey := "native:" + key
				if _, known := available[nativeKey]; !known {
					keys = append(keys, nativeKey)
				}
				available[nativeKey] = true
				meta[key] = map[string]any{"section": nativeKey, "count": len(array), "bundle_ref": ref}
				continue
			}
			meta[key] = value
		}
		metadata = append(metadata, meta)
		for _, key := range keys {
			if key == "allmetadata" {
				continue
			}
			items := contextReviewArrays(bundle, key)
			if items != nil {
				available[key] = true
				counts[key] += len(items)
				if key == section {
					for ordinal, artifact := range items {
						fields, ok := artifact.(map[string]any)
						if !ok {
							fields = map[string]any{"value": artifact}
						}
						sourceRef := ""
						if pins, ok := source.(map[string]any); ok {
							sourceRef, _ = pins["source_ref"].(string)
						}
						rows = append(rows, map[string]any{"id": fmt.Sprintf("%s#%s/%d", ref, key, ordinal), "kind": key, "fields": fields,
							"source_refs": []any{map[string]any{"source_ref": sourceRef, "source_version_ref": status.SourceVersionRef,
								"locator": map[string]any{"bundle_ref": ref, "section": key, "ordinal": ordinal}}}})
					}
				}
			}
		}
	}
	if !supported && !available[section] {
		previewError(w, 422, errors.New("unknown native package array"))
		return
	}
	if section == "allmetadata" {
		rows = metadata
	}
	counts["allmetadata"] = len(metadata)
	available["allmetadata"] = true
	page := []any{}
	next := any(nil)
	size := 0
	for i := offset; i < len(rows) && len(page) < 50; i++ {
		raw, _ := json.Marshal(rows[i])
		if len(raw) > 3<<20 {
			previewError(w, 413, errors.New("one artifact exceeds review page budget; full body needs a source reference reader"))
			return
		}
		if size+len(raw) > 3<<20 {
			next = i
			break
		}
		page = append(page, rows[i])
		size += len(raw)
	}
	if offset+len(page) < len(rows) {
		next = offset + len(page)
	}
	revisionBytes, _ := json.Marshal(refs)
	revision := fmt.Sprintf("%x", sha256.Sum256(revisionBytes))
	previewJSON(w, 200, map[string]any{"workflow_id": id, "revision": revision, "source_version_ref": status.SourceVersionRef, "source": source, "counts": counts, "available": available, "items": page, "next_offset": next, "section": section})
}
