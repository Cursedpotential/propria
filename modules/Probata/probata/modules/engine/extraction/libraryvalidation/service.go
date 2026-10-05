// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Plan contains references and counts only; inputs: prepared proposal; outputs: safe Temporal coordinates; effects: none.
type Plan struct {
	ProposalID      string      `json:"proposal_id"`
	ProposedHash    string      `json:"proposed_hash"`
	ProposalVersion string      `json:"proposal_version"`
	AttemptID       string      `json:"attempt_id,omitempty"`
	ClaimCount      int         `json:"claim_count"`
	Ref             ArtifactRef `json:"plan_ref"`
}

// ClaimInput identifies one claim without carrying its personal text; inputs: plan/index; outputs: Activity coordinates.
type ClaimInput struct {
	Plan        Plan        `json:"plan"`
	Index       int         `json:"index"`
	SnapshotRef ArtifactRef `json:"snapshot_ref"`
}

// StepResult returns durable evidence coordinates and safe normalized status only; inputs: completed step; effects: none.
type StepResult struct {
	Ref    ArtifactRef `json:"ref"`
	Index  int         `json:"index"`
	Status string      `json:"status"`
}

// FinishInput names every ordered check reference for independent receipt construction; no claim bodies cross history.
type FinishInput struct {
	Plan   Plan         `json:"plan"`
	Checks []StepResult `json:"checks"`
}

// Result reports a tracked outcome without personal content; inputs: receipt commit; outputs: safe workflow result.
type Result struct {
	ProposalID        string      `json:"proposal_id"`
	ReceiptID         string      `json:"receipt_id,omitempty"`
	ReceiptRef        ArtifactRef `json:"receipt_ref"`
	Status            string      `json:"status"`
	CurrencyStatus    string      `json:"currency_status"`
	DatabaseCommitted bool        `json:"database_committed"`
	FailureCode       string      `json:"failure_code,omitempty"`
}

// Service exposes atomic preparation, source fetch, claim check and receipt commit units.
// Inputs: explicit shared/B2/model/parser dependencies; outputs: reference-only step results. Effects: named per operation.
type Service struct {
	Repository Repository
	Artifacts  Artifacts
	Fetcher    *PrimaryFetcher
	Extractor  Extractor
	Verifier   ClaimVerifier
	SigningKey []byte
	Now        func() time.Time
}

// now uses a millisecond UTC clock so SDK datetime normalization cannot invalidate signed receipt serialization.
// Inputs: injected clock; outputs: stable timestamp; effects: none.
func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC().Truncate(time.Millisecond)
	}
	return time.Now().UTC().Truncate(time.Millisecond)
}

// ValidateProposal enforces citation and self-capture boundaries without removing personal fields.
// Inputs: complete shared proposal; outputs: contract error only. Effects: none; choose at preparation and readback.
func ValidateProposal(p Proposal) error {
	if !proposalPattern.MatchString(p.ID) || !digestPattern.MatchString(p.Version) || !digestPattern.MatchString(p.ProposedHash) || len(p.Citations) < 1 || len(p.Citations) > MaxClaims || p.ProposedRecord == nil || (!strings.HasPrefix(p.Target, "reference:") && !sourcePattern.MatchString(p.Target)) || (p.ExpectedVersion != "absent" && !digestPattern.MatchString(p.ExpectedVersion)) {
		return errors.New("invalid library proposal contract")
	}
	for _, c := range p.Citations {
		self := c.SourceID == p.Target && c.SourceVersion == "absent" && p.ExpectedVersion == "absent" && sourcePattern.MatchString(p.Target)
		if !sourcePattern.MatchString(c.SourceID) || (!self && !digestPattern.MatchString(c.SourceVersion)) || strings.TrimSpace(c.Claim) == "" || len(c.Claim) > 8000 || strings.TrimSpace(c.Pinpoint) == "" || len(c.Pinpoint) > 512 {
			return errors.New("invalid library citation or absent source exception")
		}
	}
	raw, _ := json.Marshal(p)
	if int64(len(raw)) > MaxArtifactBytes {
		return errors.New("library proposal exceeds validation budget")
	}
	return nil
}

// Prepare retains an authenticated proposal plan and returns compact reference/count coordinates.
// Inputs: existing shared proposal ID. Outputs: immutable plan. Effects: case_record read and derivative plan write only.
func (s Service) Prepare(ctx context.Context, id string) (Plan, error) {
	if s.Repository == nil || s.Artifacts == nil || len(s.SigningKey) < 32 {
		return Plan{}, errors.New("library validation service is not configured")
	}
	p, err := s.Repository.Proposal(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if err = ValidateProposal(p); err != nil {
		return Plan{}, err
	}
	if p.Status != "pending_validation" {
		return Plan{}, errors.New("library proposal is not pending validation")
	}
	ref, err := putSigned(ctx, s.Artifacts, artifactKey(p, "plan.json"), p, s.SigningKey)
	if err != nil {
		return Plan{}, err
	}
	return Plan{ProposalID: p.ID, ProposedHash: p.ProposedHash, ProposalVersion: p.Version, ClaimCount: len(p.Citations), Ref: ref}, nil
}

// loadPlan authenticates history coordinates against the retained proposal, without fetching personal text from history.
// Inputs: compact plan; outputs: retained proposal; effects: pinned descriptor read only.
func (s Service) loadPlan(ctx context.Context, plan Plan) (Proposal, error) {
	if s.Artifacts == nil {
		return Proposal{}, errors.New("validation artifact store missing")
	}
	var p Proposal
	if err := readSigned(ctx, s.Artifacts, plan.Ref, &p, s.SigningKey); err != nil {
		return Proposal{}, err
	}
	if err := ValidateProposal(p); err != nil {
		return Proposal{}, err
	}
	if p.ID != plan.ProposalID || p.ProposedHash != plan.ProposedHash || p.Version != plan.ProposalVersion || len(p.Citations) != plan.ClaimCount {
		return Proposal{}, errors.New("validation plan reference mismatch")
	}
	return p, nil
}

// current checks the proposal, target and all cited shared versions before trust can be committed.
// Inputs: retained proposal; outputs: stale/error. Effects: shared case_record reads only.
func (s Service) current(ctx context.Context, p Proposal) error {
	fresh, err := s.Repository.Proposal(ctx, p.ID)
	if err != nil {
		return err
	}
	if fresh.Version != p.Version || fresh.ProposedHash != p.ProposedHash || fresh.Status != "pending_validation" || !reflect.DeepEqual(fresh.Citations, p.Citations) {
		return ErrStale
	}
	target, err := s.Repository.Record(ctx, p.Target)
	if err != nil {
		return err
	}
	targetVersion := "absent"
	if target.ID != "" {
		targetVersion = target.Version
	}
	if targetVersion != p.ExpectedVersion {
		return ErrStale
	}
	for _, c := range p.Citations {
		record, err := s.Repository.Record(ctx, c.SourceID)
		if err != nil {
			return err
		}
		self := c.SourceID == p.Target && c.SourceVersion == "absent" && p.ExpectedVersion == "absent"
		if (self && record.ID != "") || (!self && (record.ID == "" || record.Version != c.SourceVersion)) {
			return ErrStale
		}
	}
	return nil
}

// SourceSnapshot fetches one cited official source and retains raw bytes plus its signed fetch descriptor.
// Inputs: authenticated plan/index. Outputs: reference/status only. Effects: shared reads, bounded official fetch and derivative writes.
// Choose independently per citation; active R2 storage, a 403, unsupported media or stale version produces visible blocked/stale evidence.
func (s Service) SourceSnapshot(ctx context.Context, input ClaimInput) (StepResult, error) {
	p, err := s.loadPlan(ctx, input.Plan)
	if err != nil {
		return StepResult{}, err
	}
	if input.Index < 0 || input.Index >= len(p.Citations) {
		return StepResult{}, errors.New("claim index outside plan")
	}
	citation := p.Citations[input.Index]
	snapshot := Snapshot{ProposalID: p.ID, ProposedHash: p.ProposedHash, ProposalVersion: p.Version, Index: input.Index, Citation: citation, FetchedAt: s.now(), Status: Blocked}
	stateErr := s.current(ctx, p)
	if errors.Is(stateErr, ErrStale) {
		snapshot.Status = Stale
		snapshot.FailureCode = "SHARED_VERSION_CHANGED"
	} else if stateErr != nil {
		return StepResult{}, stateErr
	} else {
		var fields map[string]json.RawMessage
		if citation.SourceVersion == "absent" {
			fields = p.ProposedRecord
		} else {
			record, err := s.Repository.Record(ctx, citation.SourceID)
			if err != nil {
				return StepResult{}, err
			}
			if record.Version != citation.SourceVersion {
				return StepResult{}, ErrStale
			}
			fields = record.Fields
		}
		snapshot.PrimaryURL = stringField(fields, "primary_url")
		if snapshot.PrimaryURL == "" {
			snapshot.PrimaryURL = stringField(fields, "url")
		}
		if activeR2Source(fields) || (sourcePattern.MatchString(p.Target) && activeR2Source(p.ProposedRecord)) {
			snapshot.FailureCode = "R2_SOURCE_STORAGE_RETIRED"
		} else if s.Fetcher == nil {
			snapshot.FailureCode = "FETCHER_NOT_CONFIGURED"
		} else {
			body, fetchErr := s.Fetcher.Fetch(ctx, snapshot.PrimaryURL)
			if fetchErr != nil {
				var blocked *FetchError
				if errors.As(fetchErr, &blocked) {
					snapshot.FailureCode = blocked.Code
				} else {
					snapshot.FailureCode = "FETCH_FAILED"
				}
			} else {
				snapshot.FinalURL = body.FinalURL
				snapshot.MediaType = body.MediaType
				snapshot.ETag = body.ETag
				snapshot.LastModified = body.LastModified
				suffix := ".html"
				if body.MediaType == "application/pdf" {
					suffix = ".pdf"
				} else if body.MediaType == "text/plain" {
					suffix = ".txt"
				}
				snapshot.Raw, err = s.Artifacts.Put(ctx, artifactKey(p, fmt.Sprintf("source-%d%s", input.Index, suffix)), body.Bytes, body.MediaType)
				if err != nil {
					return StepResult{}, err
				}
				snapshot.Status = Fetched
				// A stored primary-byte version is distinct from the hash of the shared source record.
				if pinned := stringField(fields, "primary_sha256"); pinned != "" && pinned != snapshot.Raw.SHA256 && pinned != strings.TrimPrefix(snapshot.Raw.SHA256, "sha256:") {
					snapshot.Status = Stale
					snapshot.FailureCode = "PRIMARY_BYTES_CHANGED"
				}
				currency, err := s.Repository.Record(ctx, "library_currency:"+strings.TrimPrefix(citation.SourceID, "source:"))
				if err != nil {
					return StepResult{}, err
				}
				if currency.ID != "" {
					raw, _ := json.Marshal(currency.Fields)
					var evidence CurrencyEvidence
					if json.Unmarshal(raw, &evidence) == nil {
						snapshot.Currency = &evidence
					}
				}
			}
		}
	}
	ref, err := putSigned(ctx, s.Artifacts, artifactKey(p, fmt.Sprintf("snapshot-%d.json", input.Index)), snapshot, s.SigningKey)
	if err != nil {
		return StepResult{}, err
	}
	return StepResult{Ref: ref, Index: input.Index, Status: snapshot.Status}, nil
}

// VerifyClaim checks one exact signed snapshot and stores an authenticated result outside workflow history.
// Inputs: proposal plan/index and snapshot reference. Outputs: check reference/status. Effects: extraction/model calls and derivative write.
func (s Service) VerifyClaim(ctx context.Context, input ClaimInput) (StepResult, error) {
	p, err := s.loadPlan(ctx, input.Plan)
	if err != nil {
		return StepResult{}, err
	}
	var snapshot Snapshot
	if err = readSigned(ctx, s.Artifacts, input.SnapshotRef, &snapshot, s.SigningKey); err != nil {
		return StepResult{}, err
	}
	if input.Index < 0 || input.Index >= len(p.Citations) || snapshot.ProposalID != p.ID || snapshot.ProposedHash != p.ProposedHash || snapshot.ProposalVersion != p.Version || snapshot.Index != input.Index || snapshot.Citation != p.Citations[input.Index] {
		return StepResult{}, errors.New("snapshot citation/plan mismatch")
	}
	var extracted Extracted
	if snapshot.Status == Fetched {
		if s.Extractor == nil {
			snapshot.Status = Blocked
			snapshot.FailureCode = "EXTRACTOR_NOT_CONFIGURED"
		} else {
			extracted, err = s.Extractor.Extract(ctx, snapshot)
			if err != nil {
				snapshot.Status = Blocked
				snapshot.FailureCode = "PINNED_EXTRACTION_FAILED"
			}
		}
	}
	verifier := s.Verifier
	verifier.Now = s.now
	check := verifier.Verify(ctx, snapshot, extracted)
	if err = s.current(ctx, p); errors.Is(err, ErrStale) {
		check.Status = Stale
		check.FailureCode = "SHARED_VERSION_CHANGED"
	} else if err != nil {
		return StepResult{}, err
	}
	ref, err := putSigned(ctx, s.Artifacts, artifactKey(p, fmt.Sprintf("check-%d.json", input.Index)), check, s.SigningKey)
	if err != nil {
		return StepResult{}, err
	}
	return StepResult{Ref: ref, Index: input.Index, Status: check.Status}, nil
}

// Finish authenticates all ordered checks, preserves partial failures, signs and atomically commits a trusted receipt.
// Inputs: plan and signed check references. Outputs: tracked receipt result. Effects: derivative receipt and privileged validation row only.
// Choose only after every claim Activity; any absent, failed, stale or currency-uncleared check prevents whole-proposal verification.
func (s Service) Finish(ctx context.Context, input FinishInput) (Result, error) {
	p, err := s.loadPlan(ctx, input.Plan)
	if err != nil {
		return Result{}, err
	}
	now := s.now()
	receipt := Receipt{ProposalID: p.ID, ProposalVersion: p.Version, ProposedHash: p.ProposedHash, Status: Verified, CurrencyStatus: Cleared, ValidatorVersion: ValidatorVersion, CompletedAt: now, ExpiresAt: now.Add(ReceiptLifetime), Claims: p.Citations, ClaimChecks: make([]ClaimCheck, len(p.Citations))}
	receipt.AttemptID = input.Plan.AttemptID
	indexed := map[int]StepResult{}
	for _, step := range input.Checks {
		if step.Index < 0 || step.Index >= len(p.Citations) {
			return Result{}, errors.New("check index outside proposal")
		}
		if _, duplicate := indexed[step.Index]; duplicate {
			return Result{}, errors.New("duplicate claim result")
		}
		indexed[step.Index] = step
	}
	for index, citation := range p.Citations {
		check := ClaimCheck{Citation: citation, Index: index, Status: Blocked, CurrencyStatus: Provisional, CheckVersion: CheckVersion, EvidenceTime: now, FailureCode: "CLAIM_UNPROCESSED"}
		if step, ok := indexed[index]; ok {
			if err = readSigned(ctx, s.Artifacts, step.Ref, &check, s.SigningKey); err != nil {
				return Result{}, err
			}
			if check.Index != index || check.Citation != citation || check.Status != step.Status || check.ProposalID != p.ID || check.ProposalVersion != p.Version || check.ProposedHash != p.ProposedHash {
				return Result{}, errors.New("claim check does not match exact ordered citation")
			}
		}
		if check.Status == Verified && (!rawDigest(check.SnapshotSHA256) || !rawDigest(check.QuoteSHA256) || !b2Reference(check.SnapshotRef) || check.VersionID == "" || check.VersionID != check.SnapshotVersionID || check.EvidenceTime.IsZero() || check.EvidenceTime.After(now) || now.Sub(check.EvidenceTime) > ReceiptLifetime || check.QuoteSHA256 != strings.TrimPrefix(Hash([]byte(check.Quote)), "sha256:")) {
			check.Status = Blocked
			check.FailureCode = "CHECK_EVIDENCE_INCOMPLETE"
		}
		if check.CurrencyStatus == Cleared {
			snapshot := Snapshot{Citation: check.Citation, PrimaryURL: check.PrimaryURL, Raw: ArtifactRef{SHA256: "sha256:" + check.SnapshotSHA256}, Currency: check.Currency}
			if !currencyCleared(snapshot, now, s.Verifier.CurrencyKey) {
				check.CurrencyStatus = Provisional
				check.FailureCode = "CURRENCY_REVIEW_INVALID_OR_EXPIRED"
			}
		}
		if check.CurrencyStatus != Cleared {
			receipt.CurrencyStatus = Provisional
		}
		if check.Currency != nil && check.Currency.ExpiresAt.Before(receipt.ExpiresAt) {
			receipt.ExpiresAt = check.Currency.ExpiresAt
		}
		receipt.ClaimChecks[index] = check
	}
	receipt.Status = aggregateStatus(receipt.ClaimChecks, receipt.CurrencyStatus, len(indexed) == len(p.Citations))
	if receipt.ExpiresAt.Before(now.Add(time.Minute)) {
		receipt.ExpiresAt = now.Add(time.Minute)
		if receipt.Status == Verified {
			receipt.Status = Blocked
			receipt.CurrencyStatus = Provisional
		}
	}
	// Lost commit replies/retries reuse the identical stored signed receipt rather than changing timestamps/signature.
	existing, err := s.Repository.Record(ctx, "library_validation:"+proposalPattern.FindStringSubmatch(p.ID)[1])
	if err != nil {
		return Result{}, err
	}
	if existing.ID != "" {
		raw, _ := json.Marshal(existing.Fields)
		var old Receipt
		if json.Unmarshal(raw, &old) != nil {
			return Result{}, errors.New("existing receipt is malformed")
		}
		sig := old.Signature
		old.Signature = ""
		if !matchesSignature(old, sig, s.SigningKey) || old.ProposedHash != p.ProposedHash || old.ProposalVersion != p.Version || !reflect.DeepEqual(old.Claims, p.Citations) {
			return Result{}, errors.New("existing trusted receipt conflicts with plan")
		}
		old.Signature = sig
		if stateErr := s.current(ctx, p); errors.Is(stateErr, ErrStale) {
			return Result{ProposalID: p.ID, ReceiptID: existing.ID, Status: Stale, CurrencyStatus: Provisional, DatabaseCommitted: true, FailureCode: "SHARED_VERSION_OR_RECEIPT_CHANGED"}, nil
		} else if stateErr != nil {
			return Result{}, stateErr
		}
		if old.AttemptID == input.Plan.AttemptID || old.Status == Verified {
			status, currency := old.Status, old.CurrencyStatus
			if !old.ExpiresAt.After(now) {
				status, currency = Stale, Provisional
			}
			return Result{ProposalID: p.ID, ReceiptID: existing.ID, Status: status, CurrencyStatus: currency, DatabaseCommitted: true}, nil
		}
		// A new failed-workflow retry may replace a blocked receipt by compare-and-swap.
		// The earlier signed receipt remains version-pinned in B2 and in its original run's history.
		receipt.PreviousSignature = sig
	}
	stateErr := s.current(ctx, p)
	if errors.Is(stateErr, ErrStale) {
		receipt.Status = Stale
	} else if stateErr != nil {
		return Result{}, stateErr
	}
	receipt.Signature, err = signJSON(receipt, s.SigningKey)
	if err != nil {
		return Result{}, err
	}
	ref, err := putSigned(ctx, s.Artifacts, artifactKey(p, "receipt.json"), receipt, s.SigningKey)
	if err != nil {
		return Result{}, err
	}
	result := Result{ProposalID: p.ID, ReceiptRef: ref, Status: receipt.Status, CurrencyStatus: receipt.CurrencyStatus}
	if errors.Is(stateErr, ErrStale) {
		result.FailureCode = "SHARED_VERSION_CHANGED"
		return result, nil
	}
	result.ReceiptID, err = s.Repository.Commit(ctx, receipt)
	if err != nil {
		return Result{}, err
	}
	result.DatabaseCommitted = true
	return result, nil
}

// aggregateStatus fails closed while distinguishing complete failures from missing/partial work.
// Inputs: all checks/currency/coverage; outputs: normalized overall status. Effects: none.
func aggregateStatus(checks []ClaimCheck, currency string, complete bool) string {
	if !complete {
		return Partial
	}
	passed := 0
	status := Blocked
	for _, check := range checks {
		if check.Status == Stale {
			return Stale
		}
		if check.Status == Conflicted {
			status = Conflicted
		}
		if check.Status == Verified {
			passed++
		}
	}
	if passed == len(checks) && len(checks) > 0 && currency == Cleared {
		return Verified
	}
	if status == Conflicted {
		return Conflicted
	}
	if passed > 0 && passed < len(checks) {
		return Partial
	}
	return Blocked
}

// artifactKey isolates one proposal/version's derivatives; inputs: proposal/role; outputs: safe relative key; effects: none.
func artifactKey(p Proposal, role string) string {
	return strings.TrimPrefix(p.ID, "library_proposal:") + "/" + strings.TrimPrefix(p.ProposedHash, "sha256:") + "/" + role
}

// stringField reads an exact configured source coordinate without converting or redacting body fields.
// Inputs: shared JSON fields/key; outputs: string or empty; effects: none.
func stringField(fields map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(fields[key], &value)
	return value
}
