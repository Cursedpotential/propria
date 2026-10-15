package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceformat"
)

const batchUnitManifestLimit = 32 << 20
const batchUnitMemberLimit = 10000

// BatchUnitHead resolves provider metadata only through an authorized object-store adapter.
type BatchUnitHead func(context.Context, string) (proffer.BatchUnitMember, error)

// BatchUnitRead reads at most the detector prefix from one already-pinned original.
type BatchUnitRead func(context.Context, proffer.BatchUnitMember) ([]byte, error)

// FreezeBatchUnit snapshots all original membership and selects one byte-verified HTML entrypoint.
// Inputs: explicit authorized unit root, scope and format override. Outputs: compact external manifest identity.
// Effects: bounded provider listing/HEAD/range reads and one exclusive immutable local manifest write.
// Choose instead of list-per-object batching when attachments belong to one owner-selected package.
func (a BatchImportActivities) FreezeBatchUnit(ctx context.Context, req proffer.BatchUnitRequest) (proffer.BatchUnitResult, error) {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(req.OperatingMode)); err != nil {
		return proffer.BatchUnitResult{}, err
	}
	if !caseidentity.AdmittedIdentity(req.MatterID, req.CourtCaseID) || req.BatchID == "" || req.DeclaredFormat == "" {
		return proffer.BatchUnitResult{}, errors.New("package inventory requires approved case scope, batch identity and declared format")
	}
	u, err := url.Parse(req.UnitRoot)
	if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	prefix := strings.TrimPrefix(u.Path, "/")
	if _, err = proffer.CanonicalBatchUnitRoots(u.Scheme, u.Host, prefix, []string{req.UnitRoot}); err != nil {
		return proffer.BatchUnitResult{}, err
	}
	root, err := filepath.Abs(a.UnitManifestRoot)
	if err != nil || a.UnitManifestRoot == "" {
		return proffer.BatchUnitResult{}, errors.New("package inventory manifest root is unavailable")
	}
	requestBytes, _ := json.Marshal(req)
	digest := sha256.Sum256(requestBytes)
	filename := filepath.Join(root, "batch-units", hex.EncodeToString(digest[:])+".json")
	if a.UnitAuthorize == nil {
		return proffer.BatchUnitResult{}, errors.New("package root authority is unavailable")
	}
	if err := a.UnitAuthorize(req.UnitRoot); err != nil {
		return proffer.BatchUnitResult{}, err
	}
	if _, readErr := os.Stat(filename); readErr == nil {
		manifest, sha, err := ReadBatchUnitManifest(root, proffer.Ref((&url.URL{Scheme: "file", Path: filepath.ToSlash(filename)}).String()), "")
		if err != nil {
			return proffer.BatchUnitResult{}, err
		}
		if !reflect.DeepEqual(manifest.Request, req) {
			return proffer.BatchUnitResult{}, errors.New("frozen unit inventory is owned by different request coordinates")
		}
		return batchUnitResult(filename, sha, manifest), nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return proffer.BatchUnitResult{}, readErr
	}
	if a.Lister == nil || a.UnitHead == nil || a.UnitRead == nil || a.UnitAuthorize == nil {
		return proffer.BatchUnitResult{}, errors.New("package inventory provider is unavailable")
	}
	keys, err := a.batchUnitKeys(ctx, u.Scheme, u.Host, prefix)
	if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	if len(keys) == 0 {
		return proffer.BatchUnitResult{}, errors.New("selected package contains no original objects")
	}
	manifest := proffer.BatchUnitManifest{Contract: proffer.BatchUnitContract, Request: req, Members: make([]proffer.BatchUnitMember, 0, len(keys))}
	for _, key := range keys {
		sourceRef := (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/" + key}).String()
		member, err := a.UnitHead(ctx, sourceRef)
		if err != nil {
			return proffer.BatchUnitResult{}, fmt.Errorf("inspect package member %s: %w", key, err)
		}
		if member.SourceRef != sourceRef || member.ProviderVersionID == "" || member.ProviderVersionID == "null" || len(member.ProviderVersionID) > 512 || member.ETag == "" || member.ByteLength < 0 {
			return proffer.BatchUnitResult{}, errors.New("package member has no verified immutable provider version")
		}
		member.RelativePath = strings.TrimPrefix(key, prefix)
		manifest.Members = append(manifest.Members, member)
	}
	// A changing inventory must be retried rather than silently freezing a partial selection.
	after, err := a.batchUnitKeys(ctx, u.Scheme, u.Host, prefix)
	if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	if !reflect.DeepEqual(keys, after) {
		return proffer.BatchUnitResult{}, errors.New("package membership changed while inventory was captured")
	}
	var candidates []int
	for i, member := range manifest.Members {
		if strings.Contains(member.RelativePath, "/") {
			continue
		}
		ext := strings.ToLower(path.Ext(member.RelativePath))
		if ext == ".html" || ext == ".htm" {
			candidates = append(candidates, i)
		}
	}
	// Prefer only conventional root entrypoint names, then verify their actual bytes.
	var named []int
	for _, i := range candidates {
		name := strings.ToLower(manifest.Members[i].RelativePath)
		if name == "index.html" || name == "index.htm" {
			named = append(named, i)
		}
	}
	if len(named) > 0 {
		candidates = named
	}
	var primary *proffer.BatchUnitMember
	for _, i := range candidates {
		member := manifest.Members[i]
		head, err := a.UnitRead(ctx, member)
		if err != nil {
			return proffer.BatchUnitResult{}, fmt.Errorf("inspect package entrypoint: %w", err)
		}
		if len(head) > int(sourceformat.ReadLimit) {
			return proffer.BatchUnitResult{}, errors.New("entrypoint detector read exceeded its bound")
		}
		detected, signature, err := sourceformat.Detect(head)
		if err != nil || !strings.Contains(detected, "html") {
			continue
		}
		if primary != nil {
			return proffer.BatchUnitResult{}, errors.New("selected package has multiple verified HTML entrypoints")
		}
		primary = &member
		manifest.DetectedFormat = detected
		manifest.Signature = signature
	}
	if primary == nil {
		return proffer.BatchUnitResult{}, errors.New("selected package has no uniquely verified root HTML entrypoint")
	}
	primaryURL, _ := url.Parse(primary.SourceRef)
	q := url.Values{}
	q.Set("versionId", primary.ProviderVersionID)
	primaryURL.RawQuery = q.Encode()
	manifest.PrimaryRef = proffer.Ref(primaryURL.String())
	manifest.SelectedFormat = req.DeclaredFormat
	if manifest.SelectedFormat == "unknown" || manifest.SelectedFormat == "unknown_binary" {
		manifest.SelectedFormat = manifest.DetectedFormat
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	if len(raw) > batchUnitManifestLimit {
		return proffer.BatchUnitResult{}, errors.New("package manifest exceeds its explicit metadata bound")
	}
	if err = os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return proffer.BatchUnitResult{}, err
	}
	resolvedDir, err := filepath.EvalSymlinks(filepath.Dir(filename))
	if err != nil || resolvedDir != filepath.Dir(filename) {
		return proffer.BatchUnitResult{}, errors.New("package manifest directory may not traverse symlinks")
	}
	// Keep completed pending files for owner cleanup and publish the final name only after fsync.
	f, err := os.CreateTemp(filepath.Dir(filename), "inventory-*.pending")
	if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	_, writeErr := f.Write(raw)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return proffer.BatchUnitResult{}, writeErr
	}
	if closeErr != nil {
		return proffer.BatchUnitResult{}, closeErr
	}
	if err = os.Link(f.Name(), filename); errors.Is(err, os.ErrExist) {
		existing, sha, readErr := ReadBatchUnitManifest(root, proffer.Ref((&url.URL{Scheme: "file", Path: filepath.ToSlash(filename)}).String()), "")
		if readErr != nil {
			return proffer.BatchUnitResult{}, readErr
		}
		if !reflect.DeepEqual(existing, manifest) {
			return proffer.BatchUnitResult{}, errors.New("frozen package inventory conflicts with current original membership")
		}
		return batchUnitResult(filename, sha, existing), nil
	} else if err != nil {
		return proffer.BatchUnitResult{}, err
	}
	sum := sha256.Sum256(raw)
	return batchUnitResult(filename, hex.EncodeToString(sum[:]), manifest), nil
}

// batchUnitKeys lists all selected original paths without applying standalone-import suffix or derived-file filters.
// Inputs: authorized folder and bounded provider pages. Outputs: sorted unique keys or an explicit bound failure.
// Effects: metadata listing only; use solely inside the atomic unit snapshot Activity.
func (a BatchImportActivities) batchUnitKeys(ctx context.Context, scheme, bucket, prefix string) ([]string, error) {
	keys := []string{}
	seen := map[string]bool{}
	cursors := map[string]bool{}
	cursor := ""
	for {
		page, next, err := a.Lister(ctx, scheme, bucket, prefix, cursor, maxBatchListingPage)
		if err != nil {
			return nil, err
		}
		for _, key := range page {
			if !strings.HasPrefix(key, prefix) || strings.ContainsAny(key, "\\\x00") || len(key) > 2048 {
				return nil, errors.New("provider returned a member outside the selected unit or path bound")
			}
			for _, part := range strings.Split(key, "/") {
				if part == "." || part == ".." {
					return nil, errors.New("provider returned unsafe package path")
				}
			}
			if strings.HasSuffix(key, "/") {
				continue
			}
			if seen[key] {
				return nil, errors.New("provider returned duplicate package member")
			}
			seen[key] = true
			keys = append(keys, key)
			if len(keys) > batchUnitMemberLimit {
				return nil, errors.New("package exceeds the 10000-member inventory bound; no members were dispatched")
			}
		}
		if next == "" {
			break
		}
		if cursors[next] || next == cursor {
			return nil, errors.New("package listing cursor repeated")
		}
		cursors[next] = true
		cursor = next
	}
	sort.Strings(keys)
	return keys, nil
}

// ReadBatchUnitManifest verifies a bounded manifest beneath the configured scratch root and its exact digest.
// Inputs: configured root, file URI and optional expected SHA. Outputs: immutable package metadata and digest.
// Effects: local metadata read only; choose for registration provenance validation without loading source bodies.
func ReadBatchUnitManifest(root string, ref proffer.Ref, expectedSHA string) (proffer.BatchUnitManifest, string, error) {
	var out proffer.BatchUnitManifest
	if !filepath.IsAbs(root) {
		return out, "", errors.New("package manifest requires an absolute configured root")
	}
	u, err := url.Parse(string(ref))
	if err != nil || u.Scheme != "file" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
		return out, "", errors.New("invalid package manifest reference")
	}
	base, err := filepath.Abs(filepath.Join(root, "batch-units"))
	if err != nil {
		return out, "", err
	}
	filename := filepath.FromSlash(u.Path)
	relative, err := filepath.Rel(base, filename)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return out, "", errors.New("package manifest is outside the configured root")
	}
	resolved, err := filepath.EvalSymlinks(filename)
	if err != nil {
		return out, "", err
	}
	if resolved != filename {
		return out, "", errors.New("package manifest may not traverse symlinks")
	}
	f, err := os.Open(filename)
	if err != nil {
		return out, "", err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, batchUnitManifestLimit+1))
	if err != nil || len(raw) > batchUnitManifestLimit {
		return out, "", errors.New("package manifest read exceeds its bound")
	}
	sum := sha256.Sum256(raw)
	sha := hex.EncodeToString(sum[:])
	if expectedSHA != "" && sha != expectedSHA {
		return out, "", errors.New("package manifest SHA does not match frozen identity")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&out); err != nil {
		return out, "", err
	}
	if out.Contract != proffer.BatchUnitContract || out.PrimaryRef == "" || out.SelectedFormat == "" || len(out.Members) < 1 || len(out.Members) > batchUnitMemberLimit {
		return out, "", errors.New("package manifest lacks its immutable inventory contract")
	}
	if err := validateFrozenBatchUnit(out); err != nil {
		return out, "", err
	}
	return out, sha, nil
}

// validateFrozenBatchUnit proves the primary is exactly one pinned member of a safe complete inventory.
// Inputs: decoded external manifest. Outputs: validation or refusal. Effects: none.
// Choose on every manifest read so registration never trusts detached primary or member references.
func validateFrozenBatchUnit(m proffer.BatchUnitManifest) error {
	u, err := url.Parse(m.Request.UnitRoot)
	if err != nil {
		return err
	}
	prefix := strings.TrimPrefix(u.Path, "/")
	if _, err := proffer.CanonicalBatchUnitRoots(u.Scheme, u.Host, prefix, []string{m.Request.UnitRoot}); err != nil {
		return err
	}
	primary, err := url.Parse(string(m.PrimaryRef))
	if err != nil {
		return err
	}
	query := primary.Query()
	version := query.Get("versionId")
	if version == "" || len(query) != 1 || len(query["versionId"]) != 1 {
		return errors.New("frozen primary requires one exact provider version")
	}
	primary.RawQuery = ""
	seen := map[string]bool{}
	matches := 0
	for _, member := range m.Members {
		if member.RelativePath == "" || strings.HasPrefix(member.RelativePath, "/") || seen[member.RelativePath] || strings.ContainsAny(member.RelativePath, "\\\x00") || member.ProviderVersionID == "" || member.ProviderVersionID == "null" || member.ETag == "" || member.ByteLength < 0 {
			return errors.New("frozen inventory contains invalid or duplicate member identity")
		}
		for _, part := range strings.Split(member.RelativePath, "/") {
			if part == "" || part == "." || part == ".." {
				return errors.New("frozen inventory member path is unsafe")
			}
		}
		expected := (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/" + prefix + member.RelativePath}).String()
		if member.SourceRef != expected {
			return errors.New("frozen inventory member does not belong to the selected root")
		}
		seen[member.RelativePath] = true
		if member.SourceRef == primary.String() && member.ProviderVersionID == version {
			matches++
		}
	}
	if matches != 1 {
		return errors.New("frozen primary is not an exact version-pinned inventory member")
	}
	return nil
}

// batchUnitResult projects external inventory metadata into the bounded workflow result contract.
// Inputs: immutable manifest filename, digest and metadata. Outputs: compact reference projection.
// Effects: none; choose instead of passing membership arrays through Temporal history.
func batchUnitResult(filename, sha string, m proffer.BatchUnitManifest) proffer.BatchUnitResult {
	return proffer.BatchUnitResult{UnitRoot: m.Request.UnitRoot, UnitManifestRef: proffer.Ref((&url.URL{Scheme: "file", Path: filepath.ToSlash(filename)}).String()), ManifestSHA256: sha, MemberCount: len(m.Members), PrimaryRef: m.PrimaryRef, DetectedFormat: m.DetectedFormat, SelectedFormat: m.SelectedFormat, Signature: m.Signature}
}
