// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const CurrencyReviewScope = "current_release_and_intervening_changes"
const CurrencyApprovalLifetime = 30 * time.Minute

// CurrencyReviewApproval binds an authenticated review decision to retained primary and official release evidence.
// Inputs: protected review authority's decision, actor, exact snapshots, release passage and review horizon.
// Outputs: signed approval; effects: none. Choose only after a reviewer checks the release AND intervening amendments.
// A passage or a model verdict cannot establish that no later amendment exists; the protected review authority owns that judgment.
type CurrencyReviewApproval struct {
	ReviewID           string      `json:"review_id"`
	ReviewerID         string      `json:"reviewer_id"`
	DecisionID         string      `json:"decision_id"`
	ReviewScope        string      `json:"review_scope"`
	SourceSnapshotRef  ArtifactRef `json:"source_snapshot_ref"`
	ReleaseSnapshotRef ArtifactRef `json:"release_snapshot_ref"`
	ReleaseVersion     string      `json:"release_version"`
	ReleasePinpoint    string      `json:"release_pinpoint"`
	ReleaseQuote       string      `json:"release_quote"`
	EffectiveAt        time.Time   `json:"effective_at"`
	ReviewedThrough    time.Time   `json:"reviewed_through"`
	ApprovedAt         time.Time   `json:"approved_at"`
	ExpiresAt          time.Time   `json:"expires_at"`
	Signature          string      `json:"signature"`
}

// SignCurrencyReviewApproval authenticates a completed review using a key unavailable to the validator and ordinary clients.
// Inputs: approval after authenticated reviewer authorization and independent authority key; outputs: signed approval.
// Effects: none. Call inside the protected review service, never expose as an MCP tool or accept a client assertion of actor authority.
func SignCurrencyReviewApproval(approval CurrencyReviewApproval, authorityKey []byte) (CurrencyReviewApproval, error) {
	approval.Signature = ""
	approval.EffectiveAt = approval.EffectiveAt.UTC().Truncate(time.Millisecond)
	approval.ReviewedThrough = approval.ReviewedThrough.UTC().Truncate(time.Millisecond)
	approval.ApprovedAt = approval.ApprovedAt.UTC().Truncate(time.Millisecond)
	approval.ExpiresAt = approval.ExpiresAt.UTC().Truncate(time.Millisecond)
	signature, err := signJSON(approval, authorityKey)
	approval.Signature = signature
	return approval, err
}

// CurrencyReviewService creates trusted rows from independently approved and byte-verified current-release evidence.
// Inputs: existing repository, pinned store/parser, separate approval/currency/validation keys, dedicated currency DB writer.
// Outputs: signed library_currency rows; effects: explicit official reads, derivative audit writes and currency-only transactions.
// Choose in the protected reviewer composition; the ordinary validation worker holds no ApprovalKey and registers no currency-write tool.
type CurrencyReviewService struct {
	Repository  Repository
	Artifacts   Artifacts
	Extractor   Extractor
	Fetcher     *PrimaryFetcher
	DB          SQLClient
	SigningKey  []byte
	ApprovalKey []byte
	CurrencyKey []byte
	Now         func() time.Time
}

// now returns the same bounded datetime representation used by validation receipts; inputs: clock; outputs: UTC milliseconds.
func (s CurrencyReviewService) now() time.Time { return (Service{Now: s.Now}).now() }

// configured ensures independent trust authorities cannot accidentally share signing material; effects: none.
func (s CurrencyReviewService) configured() error {
	if s.Repository == nil || s.Artifacts == nil || s.Extractor == nil || len(s.SigningKey) < 32 || len(s.ApprovalKey) < 32 || len(s.CurrencyKey) < 32 || bytes.Equal(s.SigningKey, s.ApprovalKey) || bytes.Equal(s.SigningKey, s.CurrencyKey) || bytes.Equal(s.CurrencyKey, s.ApprovalKey) {
		return errors.New("currency review needs independent signing, approval and currency keys plus pinned dependencies")
	}
	return nil
}

// CaptureRelease retains a directly fetched official release as an authenticated descriptor for subsequent review.
// Inputs: allowed official HTTPS URL. Outputs: pinned signed snapshot reference; effects: bounded HTTP and derivative B2 writes.
// Choose before authenticated legal review; this operation never approves currency and never writes a database row.
func (s CurrencyReviewService) CaptureRelease(ctx context.Context, url string) (ArtifactRef, error) {
	if err := s.configured(); err != nil {
		return ArtifactRef{}, err
	}
	if s.Fetcher == nil {
		return ArtifactRef{}, errors.New("currency release fetcher is missing")
	}
	body, err := s.Fetcher.Fetch(ctx, url)
	if err != nil {
		return ArtifactRef{}, err
	}
	key := "currency-releases/" + strings.TrimPrefix(Hash(body.Bytes), "sha256:")
	raw, err := s.Artifacts.Put(ctx, key+"/primary", body.Bytes, body.MediaType)
	if err != nil {
		return ArtifactRef{}, err
	}
	snapshot := Snapshot{PrimaryURL: url, FinalURL: body.FinalURL, MediaType: body.MediaType, Raw: raw, FetchedAt: s.now(), Status: Fetched, ETag: body.ETag, LastModified: body.LastModified}
	return putSigned(ctx, s.Artifacts, key+"/snapshot.json", snapshot, s.SigningKey)
}

// CommitApprovedReview verifies authenticated review, exact primary/release bytes and passage before signing a currency row.
// Inputs: independent authority-signed approval of retained snapshots and current release/intervening-change review.
// Outputs: committed CurrencyEvidence. Effects: shared version reads, pinned extraction, retained approval audit, currency-only DB commit.
// Choose after protected reviewer authorization; provisional MCR evidence and unsigned client flags cannot enter this seam.
func (s CurrencyReviewService) CommitApprovedReview(ctx context.Context, approval CurrencyReviewApproval) (CurrencyEvidence, error) {
	if err := s.configured(); err != nil {
		return CurrencyEvidence{}, err
	}
	now := s.now()
	unsigned := approval
	unsigned.Signature = ""
	if !matchesSignature(unsigned, approval.Signature, s.ApprovalKey) || approval.ReviewScope != CurrencyReviewScope || approval.ReviewID == "" || len(approval.ReviewID) > 200 || approval.ReviewerID == "" || approval.DecisionID == "" || strings.TrimSpace(approval.ReleaseVersion) == "" || strings.TrimSpace(approval.ReleaseQuote) == "" || len(approval.ReleaseQuote) > MaxPassageBytes || approval.ApprovedAt.IsZero() || approval.ApprovedAt.After(now) || now.Sub(approval.ApprovedAt) > CurrencyApprovalLifetime || approval.ReviewedThrough.IsZero() || approval.ReviewedThrough.After(approval.ApprovedAt) || approval.ApprovedAt.Sub(approval.ReviewedThrough) > 5*time.Minute || approval.EffectiveAt.IsZero() || approval.EffectiveAt.After(approval.ReviewedThrough) || !approval.ExpiresAt.After(now) || approval.ExpiresAt.After(approval.ApprovedAt.Add(ReceiptLifetime)) {
		return CurrencyEvidence{}, errors.New("currency approval missing, unauthenticated, provisional or outside review horizon")
	}
	var primary, release Snapshot
	if !validPinnedRef(approval.SourceSnapshotRef) || !validPinnedRef(approval.ReleaseSnapshotRef) {
		return CurrencyEvidence{}, errors.New("currency review requires pinned B2 snapshot references")
	}
	if err := readSigned(ctx, s.Artifacts, approval.SourceSnapshotRef, &primary, s.SigningKey); err != nil {
		return CurrencyEvidence{}, err
	}
	if err := readSigned(ctx, s.Artifacts, approval.ReleaseSnapshotRef, &release, s.SigningKey); err != nil {
		return CurrencyEvidence{}, err
	}
	for _, snapshot := range []Snapshot{primary, release} {
		if !validPinnedRef(snapshot.Raw) {
			return CurrencyEvidence{}, errors.New("currency primary bytes require pinned B2 storage")
		}
		if snapshot.Status != Fetched || snapshot.FetchedAt.IsZero() || snapshot.FetchedAt.After(approval.ApprovedAt) || approval.ReviewedThrough.Sub(snapshot.FetchedAt) > CurrencyApprovalLifetime || snapshot.Raw.VersionID == "" || !digestPattern.MatchString(snapshot.Raw.SHA256) {
			return CurrencyEvidence{}, errors.New("currency snapshot is not a recent authenticated primary fetch")
		}
		if _, err := PrimaryURL(snapshot.PrimaryURL); err != nil {
			return CurrencyEvidence{}, err
		}
		if _, err := PrimaryURL(snapshot.FinalURL); err != nil {
			return CurrencyEvidence{}, err
		}
		raw, err := s.Artifacts.Read(ctx, snapshot.Raw, MaxSourceBytes)
		if err != nil {
			return CurrencyEvidence{}, err
		}
		if Hash(raw) != snapshot.Raw.SHA256 || int64(len(raw)) != snapshot.Raw.Bytes {
			return CurrencyEvidence{}, errors.New("currency pinned snapshot bytes changed")
		}
	}
	if !sourcePattern.MatchString(primary.Citation.SourceID) || (primary.Citation.SourceVersion != "absent" && !digestPattern.MatchString(primary.Citation.SourceVersion)) {
		return CurrencyEvidence{}, errors.New("currency primary source version is invalid")
	}
	// The same extractor contract that proves claim input bytes also proves the official release passage.
	extracted, err := s.Extractor.Extract(ctx, release)
	if err != nil {
		return CurrencyEvidence{}, err
	}
	if extracted.InputSHA256 != strings.TrimPrefix(release.Raw.SHA256, "sha256:") || extracted.VersionID != release.Raw.VersionID || extracted.Extractor == "" || extracted.ExtractorVersion == "" {
		return CurrencyEvidence{}, errors.New("currency release extractor binding mismatch")
	}
	passage, err := LocatePassage(extracted.Pages, approval.ReleasePinpoint)
	if err != nil {
		return CurrencyEvidence{}, err
	}
	if len(passage) > MaxPassageBytes || !strings.Contains(passage, approval.ReleaseQuote) {
		return CurrencyEvidence{}, errors.New("currency release quote not at exact pinpoint")
	}
	// A new source can only be reviewed in the same retained self-capture proposal before publication.
	if primary.Citation.SourceVersion == "absent" {
		p, err := s.Repository.Proposal(ctx, primary.ProposalID)
		if err != nil {
			return CurrencyEvidence{}, err
		}
		if p.Target != primary.Citation.SourceID || p.ExpectedVersion != "absent" || p.Version != primary.ProposalVersion || p.ProposedHash != primary.ProposedHash || p.Status != "pending_validation" {
			return CurrencyEvidence{}, ErrStale
		}
		if activeR2Source(p.ProposedRecord) {
			return CurrencyEvidence{}, errors.New("R2_SOURCE_STORAGE_RETIRED")
		}
	}
	source, err := s.Repository.Record(ctx, primary.Citation.SourceID)
	if err != nil {
		return CurrencyEvidence{}, err
	}
	if (primary.Citation.SourceVersion == "absent" && source.ID != "") || (primary.Citation.SourceVersion != "absent" && (source.ID == "" || source.Version != primary.Citation.SourceVersion)) {
		return CurrencyEvidence{}, ErrStale
	}
	if activeR2Source(source.Fields) {
		return CurrencyEvidence{}, errors.New("R2_SOURCE_STORAGE_RETIRED")
	}
	// A lost DB reply must reuse the original provider-pinned approval audit, not mint a new signature on retry.
	existing, err := s.Repository.Record(ctx, "library_currency:"+strings.TrimPrefix(primary.Citation.SourceID, "source:"))
	if err != nil {
		return CurrencyEvidence{}, err
	}
	if existing.ID != "" {
		raw, _ := json.Marshal(existing.Fields)
		var old CurrencyEvidence
		if json.Unmarshal(raw, &old) != nil {
			return CurrencyEvidence{}, errors.New("existing currency row is malformed")
		}
		if old.ReviewID == approval.ReviewID {
			sig := old.Signature
			old.Signature = ""
			if !matchesSignature(old, sig, s.CurrencyKey) || old.SourceID != primary.Citation.SourceID || old.SourceVersion != primary.Citation.SourceVersion {
				return CurrencyEvidence{}, errors.New("existing currency review conflicts with approval")
			}
			var retained CurrencyReviewApproval
			if err = readSigned(ctx, s.Artifacts, old.ApprovalRef, &retained, s.SigningKey); err != nil {
				return CurrencyEvidence{}, err
			}
			if retained.Signature != approval.Signature {
				return CurrencyEvidence{}, errors.New("existing currency approval was changed")
			}
			old.Signature = sig
			return old, nil
		}
	}
	audit, err := putSigned(ctx, s.Artifacts, "currency-reviews/"+strings.TrimPrefix(Hash([]byte(approval.ReviewID)), "sha256:")+"/approval.json", approval, s.SigningKey)
	if err != nil {
		return CurrencyEvidence{}, err
	}
	evidence := CurrencyEvidence{SourceID: primary.Citation.SourceID, SourceVersion: primary.Citation.SourceVersion, SnapshotSHA256: strings.TrimPrefix(primary.Raw.SHA256, "sha256:"), PrimaryURL: primary.PrimaryURL, ReleaseURL: release.PrimaryURL, ReleaseVersion: approval.ReleaseVersion, ReleaseSHA256: strings.TrimPrefix(release.Raw.SHA256, "sha256:"), ReviewID: approval.ReviewID, ReviewerID: approval.ReviewerID, DecisionID: approval.DecisionID, ReviewedThrough: approval.ReviewedThrough, ApprovalRef: audit, SourceSnapshotRef: approval.SourceSnapshotRef, ReleaseSnapshotRef: approval.ReleaseSnapshotRef, ReleasePinpoint: approval.ReleasePinpoint, ReleaseQuoteSHA256: strings.TrimPrefix(Hash([]byte(approval.ReleaseQuote)), "sha256:"), EffectiveAt: approval.EffectiveAt, CheckedAt: approval.ApprovedAt, ExpiresAt: approval.ExpiresAt, Status: Cleared}
	evidence.Signature, err = signJSON(evidence, s.CurrencyKey)
	if err != nil {
		return CurrencyEvidence{}, err
	}
	raw, _ := json.Marshal(evidence)
	var data map[string]any
	if err = json.Unmarshal(raw, &data); err != nil {
		return CurrencyEvidence{}, err
	}
	results, err := s.DB.Query(ctx, currencyCommitSQL, map[string]any{"evidence": data, "proposal_id": primary.ProposalID, "proposal_version": primary.ProposalVersion, "proposed_hash": primary.ProposedHash})
	if err != nil {
		return CurrencyEvidence{}, err
	}
	for _, raw := range results {
		var saved struct {
			Signature string `json:"signature"`
		}
		if json.Unmarshal(raw, &saved) == nil && saved.Signature == evidence.Signature {
			return evidence, nil
		}
	}
	return CurrencyEvidence{}, errors.New("trusted currency commit readback mismatch")
}

// currencyCommitSQL atomically gates the actual source version and preserves approved datetime/audit bindings.
// Inputs: server-authenticated evidence and optional self-capture proposal coordinates. Outputs: saved signature.
// Effects: library_currency only. A dedicated writer is required; no proposal/source/receipt is modified.
const currencyCommitSQL = `
BEGIN TRANSACTION;
LET $key = string::split($evidence.source_id, ':')[1];
LET $source = (SELECT * OMIT embedding FROM ONLY type::record('source', $key));
IF $evidence.source_version = 'absent' {
 LET $p = (SELECT * OMIT embedding FROM ONLY type::record('library_proposal', string::split($proposal_id, ':')[1]));
 IF $source != NONE OR $p = NONE OR record::tb($p.target) != 'source'
  OR record::tb($p.target) + ':' + <string> record::id($p.target) != $evidence.source_id
  OR $p.expected_version != 'absent' OR $p.status != 'pending_validation'
  OR 'sha256:' + crypto::sha256(<string> object::remove($p, ['dispatch', 'dispatch_at'])) != $proposal_version
  OR $p.proposed_hash != $proposed_hash { THROW 'Library source changed'; };
} ELSE {
 IF $source = NONE OR 'sha256:' + crypto::sha256(<string> $source) != $evidence.source_version { THROW 'Library source changed'; };
};
LET $typed = object::extend($evidence, { effective_at: <datetime> $evidence.effective_at,
 reviewed_through: <datetime> $evidence.reviewed_through, checked_at: <datetime> $evidence.checked_at,
 expires_at: <datetime> $evidence.expires_at });
IF $typed.status != 'cleared' OR $typed.checked_at > time::now() OR $typed.expires_at <= time::now()
 OR $typed.reviewed_through > $typed.checked_at OR $typed.effective_at > $typed.reviewed_through
 OR $typed.expires_at > $typed.checked_at + 1d { THROW 'Library currency review changed'; };
LET $old = (SELECT * FROM ONLY type::record('library_currency', $key));
IF $old != NONE AND $old.signature != $typed.signature AND $old.checked_at >= $typed.checked_at { THROW 'Library currency review changed'; };
UPSERT type::record('library_currency', $key) CONTENT $typed;
LET $saved = (SELECT * FROM ONLY type::record('library_currency', $key));
RETURN { signature: $saved.signature };
COMMIT TRANSACTION;
`
