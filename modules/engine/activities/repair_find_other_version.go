// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// repair.find_other_version — owner 2026-09-20: "find another version based on
// filename, or wait for more detail after parsing, offer to look in other
// sources." One job: ask the catalog for other copies with the source's file
// name, confirm each still exists in its store, and name the best one. It
// writes nothing. The catalog proposes; the store confirms — a catalog name is
// never turned into a source reference without a live HEAD, and a copy whose
// first 64 KiB are all zero bytes (a zero-filled husk, 2026-09-13 quarantine)
// is refused after one ranged GET, never a whole-object read.

package activities

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/repairplan"
)

const (
	defaultFindCandidates = 5
	maxFindCandidates     = 20
	// maxCatalogRowsByName bounds one catalog lookup.
	maxCatalogRowsByName = 50
)

// CatalogObject is one catalog row: an object key in the catalog's store.
type CatalogObject struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
	// SHA1 is empty when the catalog has not hashed the object.
	SHA1 string `json:"sha1,omitempty"`
	// Snapshot names the catalog table the row came from.
	Snapshot string `json:"snapshot"`
}

// CatalogVersionFinder is the read-only Case Bible catalog seam.
type CatalogVersionFinder interface {
	FindByBasename(ctx context.Context, basename string, limit int) ([]CatalogObject, error)
	LookupKey(ctx context.Context, key string) (CatalogObject, bool, error)
}

// CatalogStore is the object store (scheme and bucket) the catalog's keys are
// relative to. It is configuration, never code.
type CatalogStore struct {
	Scheme string
	Bucket string
}

// RepairFindOtherVersionActivity implements repair.find_other_version.
type RepairFindOtherVersionActivity struct {
	Catalog CatalogVersionFinder
	Store   CatalogStore
	// Stores resolves a configured scheme to its object store; each must be
	// able to HEAD (smsthreads.ObjectStatter).
	Stores func(scheme string) (smsthreads.ObjectStore, error)
	// Roots are the admissible source roots: a copy outside them could not
	// start a Proffer run, so it is never chosen.
	Roots objectstores.Roots
}

type findParams struct {
	MaxCandidates     *int `json:"max_candidates"`
	RequireLarger     bool `json:"require_larger"`
	IncludeQuarantine bool `json:"include_quarantine"`
}

// inQuarantine reports a key under a folder the owner set aside (a path
// segment beginning "_quarantine", e.g. consignatio/intake/_quarantine/...).
// Such copies are eligible only when the step asks for them.
func inQuarantine(key string) bool {
	for _, segment := range strings.Split(key, "/") {
		if strings.HasPrefix(strings.ToLower(segment), "_quarantine") {
			return true
		}
	}
	return false
}

// zeroCheckBytes is the head sample every candidate must pass: a copy whose
// first 64 KiB are all zero bytes is a zero-filled husk (the 2026-09-13
// zero-fill quarantine covered R2, B2, Drive and D:), however large it is.
// It is read with one ranged GET, never the whole object.
// Byline: Claude Code · Opus 5.5 · 2026-09-25
const zeroCheckBytes = 64 << 10

// maxReportedZeroFilled bounds the zero-filled locators listed in a summary.
const maxReportedZeroFilled = 20

const rejectZeroFilled = "zero-filled (first 64 KiB all zero bytes)"

type findCandidate struct {
	SourceRef   string `json:"source_ref"`
	Size        int64  `json:"size"`
	CatalogSize int64  `json:"catalog_size"`
	SHA1        string `json:"sha1,omitempty"`
	Snapshot    string `json:"snapshot"`
	// ZeroCheckedBytes is how many head bytes a ranged GET read and found
	// not all zero.
	ZeroCheckedBytes int `json:"zero_checked_bytes"`
}

type findSummary struct {
	Basename       string          `json:"basename"`
	OriginalSize   int64           `json:"original_size"`
	OriginalSHA1   string          `json:"original_sha1,omitempty"`
	CatalogRows    int             `json:"catalog_rows"`
	Considered     int             `json:"considered"`
	Candidates     []findCandidate `json:"candidates"`
	Chosen         string          `json:"chosen"`
	Rejected       map[string]int  `json:"rejected,omitempty"`
	CatalogStore   string          `json:"catalog_store"`
	RequireLarger  bool            `json:"require_larger"`
	Quarantine     bool            `json:"include_quarantine"`
	MaxCandidates  int             `json:"max_candidates"`
	CatalogSources []string        `json:"catalog_snapshots"`
	// ZeroCheckBytes is the head sample size every candidate must pass, and
	// ZeroFilled the candidates rejected because theirs was all zero bytes.
	ZeroCheckBytes int      `json:"zero_check_bytes"`
	ZeroFilled     []string `json:"zero_filled"`
}

// FindOtherVersion names the largest verified other copy of the source.
func (a RepairFindOtherVersionActivity) FindOtherVersion(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	result, err := a.find(ctx, request)
	return result, stopRetryingPermanent(err)
}

func (a RepairFindOtherVersionActivity) find(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	var params findParams
	if err := decodeStepParams(request.Params, &params); err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	limit := defaultFindCandidates
	if params.MaxCandidates != nil {
		limit = *params.MaxCandidates
	}
	if limit < 1 || limit > maxFindCandidates {
		return repairplan.StepResult{}, permanent(fmt.Errorf("max_candidates must be between 1 and %d", maxFindCandidates))
	}
	source, err := parseRepairLocator(request.SourceRef)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	if a.Catalog == nil || a.Stores == nil || a.Store.Scheme == "" || a.Store.Bucket == "" {
		return repairplan.StepResult{}, permanent(errors.New(
			"find other version: the Case Bible catalog is not configured on this worker (INTAKE_DISCOVERY_PG_* and INTAKE_DISCOVERY_OBJECT_STORE)"))
	}
	catalogStore, err := a.candidateStore(a.Store.Scheme)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	basename := path.Base(source.Key)
	summary := findSummary{
		Basename: basename, OriginalSize: -1, Candidates: []findCandidate{},
		Rejected: map[string]int{}, CatalogStore: a.Store.Scheme + "://" + a.Store.Bucket + "/",
		RequireLarger: params.RequireLarger, Quarantine: params.IncludeQuarantine, MaxCandidates: limit,
		CatalogSources: []string{}, ZeroCheckBytes: zeroCheckBytes, ZeroFilled: []string{},
	}

	// What the original is now: its live size, and its catalog hash when the
	// catalog describes the original's own store.
	sameStore := source.Scheme == a.Store.Scheme && source.Bucket == a.Store.Bucket
	if sourceStore, storeErr := a.statter(source.Scheme); storeErr == nil {
		if info, exists, statErr := sourceStore.Stat(ctx, source.Bucket, source.Key); statErr != nil {
			return repairplan.StepResult{}, fmt.Errorf("stat the source %s: %w", source.URI(), statErr)
		} else if exists {
			summary.OriginalSize = info.Size
		}
	}
	if sameStore {
		if row, found, lookupErr := a.Catalog.LookupKey(ctx, source.Key); lookupErr != nil {
			return repairplan.StepResult{}, fmt.Errorf("look up the source in the catalog: %w", lookupErr)
		} else if found {
			summary.OriginalSHA1 = row.SHA1
			if summary.OriginalSize < 0 {
				summary.OriginalSize = row.Size
			}
		}
	}

	rows, err := a.Catalog.FindByBasename(ctx, basename, maxCatalogRowsByName)
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("find copies of %s in the catalog: %w", basename, err)
	}
	summary.CatalogRows = len(rows)
	snapshots := map[string]bool{}
	var eligible []CatalogObject
	for _, row := range rows {
		if row.Snapshot != "" && !snapshots[row.Snapshot] {
			snapshots[row.Snapshot] = true
			summary.CatalogSources = append(summary.CatalogSources, row.Snapshot)
		}
		switch {
		case sameStore && row.Key == source.Key:
			summary.Rejected["the source itself"]++
		case path.Base(row.Key) != basename:
			summary.Rejected["a different file name"]++
		case !params.IncludeQuarantine && inQuarantine(row.Key):
			summary.Rejected["set aside under _quarantine/ (include_quarantine is off)"]++
		case !otherVersion(row.Size, row.SHA1, summary.OriginalSize, summary.OriginalSHA1, params.RequireLarger):
			summary.Rejected["the same size and hash, or not larger"]++
		default:
			eligible = append(eligible, row)
		}
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Size != eligible[j].Size {
			return eligible[i].Size > eligible[j].Size
		}
		return eligible[i].Key < eligible[j].Key
	})
	seen := map[string]bool{}
	for _, row := range eligible {
		if len(summary.Candidates) >= limit {
			break
		}
		if seen[row.Key] {
			continue
		}
		seen[row.Key] = true
		summary.Considered++
		candidate := repairLocator{Scheme: a.Store.Scheme, Bucket: a.Store.Bucket, Key: row.Key}
		if _, admitted := a.Roots.Match(candidate.Scheme, candidate.Bucket, candidate.Key); !admitted {
			summary.Rejected["outside the configured source roots"]++
			continue
		}
		info, exists, statErr := catalogStore.Stat(ctx, candidate.Bucket, candidate.Key)
		if statErr != nil {
			return repairplan.StepResult{}, fmt.Errorf("confirm candidate %s: %w", candidate.URI(), statErr)
		}
		if !exists {
			summary.Rejected["no longer in the store"]++
			continue
		}
		if !otherVersion(info.Size, row.SHA1, summary.OriginalSize, summary.OriginalSHA1, params.RequireLarger) {
			summary.Rejected["changed since the catalog snapshot and no longer qualifies"]++
			continue
		}
		head, readErr := catalogStore.ReadRange(ctx, candidate.Bucket, candidate.Key, 0, zeroCheckBytes)
		if readErr != nil {
			return repairplan.StepResult{}, fmt.Errorf("read the first %d bytes of candidate %s: %w", zeroCheckBytes, candidate.URI(), readErr)
		}
		switch {
		case len(head) == 0:
			summary.Rejected["empty (no bytes to read)"]++
			continue
		case allZero(head):
			summary.Rejected[rejectZeroFilled]++
			if len(summary.ZeroFilled) < maxReportedZeroFilled {
				summary.ZeroFilled = append(summary.ZeroFilled, candidate.URI())
			}
			continue
		}
		summary.Candidates = append(summary.Candidates, findCandidate{
			SourceRef: candidate.URI(), Size: info.Size, CatalogSize: row.Size, SHA1: row.SHA1, Snapshot: row.Snapshot,
			ZeroCheckedBytes: len(head),
		})
	}
	if len(summary.Candidates) == 0 {
		return repairplan.StepResult{}, permanent(fmt.Errorf(
			"no other version of %s was found: %d catalog rows by that name, none a verified different copy (%s)",
			basename, summary.CatalogRows, describeRejections(summary.Rejected)))
	}
	summary.Chosen = summary.Candidates[0].SourceRef
	return repairplan.StepResult{
		OutputRef: summary.Chosen, OutputType: request.SourceType, OutputKind: repairplan.OutputExistingObject,
		Summary: repairSummary(summary),
	}, nil
}

// otherVersion applies the ratified rule: the same file name with a different
// hash or a larger size. An unknown original size admits any copy with a
// different known hash, never an unproven one.
func otherVersion(size int64, sha1 string, originalSize int64, originalSHA1 string, requireLarger bool) bool {
	larger := originalSize >= 0 && size > originalSize
	if requireLarger {
		return larger
	}
	differentHash := sha1 != "" && originalSHA1 != "" && !strings.EqualFold(sha1, originalSHA1)
	return larger || differentHash
}

func describeRejections(rejected map[string]int) string {
	if len(rejected) == 0 {
		return "no rows"
	}
	reasons := make([]string, 0, len(rejected))
	for reason, count := range rejected {
		reasons = append(reasons, fmt.Sprintf("%d %s", count, reason))
	}
	sort.Strings(reasons)
	return strings.Join(reasons, ", ")
}

// allZero reports a sample made only of zero bytes.
func allZero(sample []byte) bool {
	for _, b := range sample {
		if b != 0 {
			return false
		}
	}
	return true
}

// candidateStore is what confirming a candidate needs: a HEAD and a ranged
// head read.
type candidateStore interface {
	smsthreads.ObjectStatter
	smsthreads.ObjectRangeReader
}

func (a RepairFindOtherVersionActivity) candidateStore(scheme string) (candidateStore, error) {
	store, err := a.Stores(scheme)
	if err != nil {
		return nil, fmt.Errorf("object store %q: %w", scheme, err)
	}
	capable, ok := store.(candidateStore)
	if !ok {
		return nil, fmt.Errorf("object store %q cannot report object size and read a byte range", scheme)
	}
	return capable, nil
}

func (a RepairFindOtherVersionActivity) statter(scheme string) (smsthreads.ObjectStatter, error) {
	store, err := a.Stores(scheme)
	if err != nil {
		return nil, fmt.Errorf("object store %q: %w", scheme, err)
	}
	statter, ok := store.(smsthreads.ObjectStatter)
	if !ok {
		return nil, fmt.Errorf("object store %q cannot report object size", scheme)
	}
	return statter, nil
}
