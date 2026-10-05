// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryArtifacts struct {
	mu     sync.Mutex
	bodies map[string][]byte
}

func (m *memoryArtifacts) Put(_ context.Context, _ string, body []byte, _ string) (ArtifactRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hash := Hash(body)
	version := "fixture-" + strings.TrimPrefix(hash, "sha256:")
	uri := "b2://fixture/validation/" + strings.TrimPrefix(hash, "sha256:") + "?versionId=" + version
	m.bodies[uri] = append([]byte(nil), body...)
	return ArtifactRef{URI: uri, VersionID: version, SHA256: hash, Bytes: int64(len(body))}, nil
}
func (m *memoryArtifacts) Read(_ context.Context, ref ArtifactRef, max int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.bodies[ref.URI]
	if !ok || int64(len(body)) != ref.Bytes || ref.Bytes > max || Hash(body) != ref.SHA256 || ref.VersionID == "" {
		return nil, errors.New("fixture artifact integrity mismatch")
	}
	return append([]byte(nil), body...), nil
}

type memoryRepository struct {
	proposal Proposal
	records  map[string]Record
	commits  int
	receipt  Receipt
}

func (m *memoryRepository) Proposal(_ context.Context, _ string) (Proposal, error) {
	return m.proposal, nil
}
func (m *memoryRepository) Record(_ context.Context, id string) (Record, error) {
	return m.records[id], nil
}
func (m *memoryRepository) Commit(_ context.Context, r Receipt) (string, error) {
	m.commits++
	m.receipt = r
	id := "library_validation:" + strings.TrimPrefix(r.ProposalID, "library_proposal:")
	raw, _ := json.Marshal(r)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	m.records[id] = Record{ID: id, Fields: fields}
	return id, nil
}

type fixtureExtractor struct {
	store Artifacts
	text  string
}

func (e fixtureExtractor) Extract(ctx context.Context, s Snapshot) (Extracted, error) {
	raw, err := e.store.Read(ctx, s.Raw, MaxSourceBytes)
	if err != nil {
		return Extracted{}, err
	}
	if !strings.Contains(string(raw), fixtureQuote) {
		return Extracted{}, errors.New("fixture source text differs")
	}
	return Extracted{Pages: []string{e.text}, InputSHA256: strings.TrimPrefix(Hash(raw), "sha256:"), VersionID: s.Raw.VersionID, Extractor: "fixture-existing-html-contract", ExtractorVersion: "1"}, nil
}

func serviceFixture(t *testing.T) (*Service, *memoryRepository, *memoryArtifacts) {
	t.Helper()
	c := Citation{SourceID: "source:mcr3215", SourceVersion: "sha256:" + strings.Repeat("a", 64), Pinpoint: "MCR 3.215(E)(4)", Claim: fixtureQuote}
	body := []byte("<html><body><pre>" + fixtureText + "</pre></body></html>")
	proposal := Proposal{ID: "library_proposal:11111111-2222-3333-4444-555555555555", Version: "sha256:" + strings.Repeat("d", 64), Target: "reference:personal-guide", ExpectedVersion: "sha256:" + strings.Repeat("e", 64), ProposedHash: "sha256:" + strings.Repeat("f", 64), Status: "pending_validation", Citations: []Citation{c}, ProposedRecord: map[string]json.RawMessage{"personal_note": json.RawMessage(`"Preserve the complete case-specific person, child and address fields."`)}}
	repo := &memoryRepository{proposal: proposal, records: map[string]Record{proposal.Target: {ID: proposal.Target, Version: proposal.ExpectedVersion, Fields: map[string]json.RawMessage{"personal_note": json.RawMessage(`"Full existing personal text"`)}}, c.SourceID: {ID: c.SourceID, Version: c.SourceVersion, Fields: map[string]json.RawMessage{"primary_url": json.RawMessage(`"` + fixtureURL + `"`)}}}}
	currency := signedCurrency(t, c, strings.TrimPrefix(Hash(body), "sha256:"))
	raw, _ := json.Marshal(currency)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	repo.records["library_currency:mcr3215"] = Record{ID: "library_currency:mcr3215", Fields: fields}
	artifacts := &memoryArtifacts{bodies: map[string][]byte{}}
	fetcher := fixtureFetcher(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(body)
	})
	service := &Service{Repository: repo, Artifacts: artifacts, Fetcher: fetcher, Extractor: fixtureExtractor{store: artifacts, text: fixtureText}, SigningKey: fixtureKey, Now: func() time.Time { return fixtureNow }, Verifier: ClaimVerifier{Model: fixtureModel(t, verdict{Status: Verified, SourceID: c.SourceID, SourceVersion: c.SourceVersion, Pinpoint: c.Pinpoint, Quote: fixtureQuote}), CurrencyKey: fixtureCurrencyKey}}
	return service, repo, artifacts
}

func runClaim(t *testing.T, s *Service, plan Plan, index int) StepResult {
	t.Helper()
	snapshot, err := s.SourceSnapshot(t.Context(), ClaimInput{Plan: plan, Index: index})
	require.NoError(t, err)
	check, err := s.VerifyClaim(t.Context(), ClaimInput{Plan: plan, Index: index, SnapshotRef: snapshot.Ref})
	require.NoError(t, err)
	return check
}

func TestFullValidationOperationBindsEveryClaimAndPreservesPersonalFields(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	before, _ := json.Marshal(repo.proposal.ProposedRecord)
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	history, _ := json.Marshal(plan)
	require.NotContains(t, string(history), "case-specific")
	require.NotContains(t, string(history), fixtureQuote)
	check := runClaim(t, s, plan, 0)
	result, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
	require.NoError(t, err)
	require.Equal(t, Verified, result.Status)
	require.True(t, result.DatabaseCommitted)
	require.Equal(t, Cleared, result.CurrencyStatus)
	require.Equal(t, repo.proposal.Citations, repo.receipt.Claims)
	require.Len(t, repo.receipt.ClaimChecks, 1)
	evidence := repo.receipt.ClaimChecks[0]
	require.Equal(t, repo.proposal.Citations[0], evidence.Citation)
	require.True(t, rawDigest(evidence.SnapshotSHA256))
	require.True(t, rawDigest(evidence.QuoteSHA256))
	require.NotEmpty(t, evidence.SnapshotVersionID)
	require.NotEmpty(t, evidence.Currency)
	unsigned := repo.receipt
	unsigned.Signature = ""
	require.True(t, matchesSignature(unsigned, repo.receipt.Signature, fixtureKey))
	after, _ := json.Marshal(repo.proposal.ProposedRecord)
	require.Equal(t, before, after)
	again, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
	require.NoError(t, err)
	require.Equal(t, result.ReceiptID, again.ReceiptID)
	require.Equal(t, 1, repo.commits, "retry must reuse the signed receipt")
}

func TestSignedClaimFromAnotherProposalCannotBeReplayed(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	first, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	check := runClaim(t, s, first, 0)
	repo.proposal.ID = "library_proposal:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	second, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	_, err = s.Finish(t.Context(), FinishInput{Plan: second, Checks: []StepResult{check}})
	require.ErrorContains(t, err, "exact ordered citation")
	require.Zero(t, repo.commits)
}

func TestProvisionalCurrencyAndFetch403NeverPassProposal(t *testing.T) {
	for _, name := range []string{"currency missing", "currency tampered", "HTTP 403"} {
		t.Run(name, func(t *testing.T) {
			s, repo, _ := serviceFixture(t)
			switch name {
			case "currency missing":
				delete(repo.records, "library_currency:mcr3215")
			case "currency tampered":
				r := repo.records["library_currency:mcr3215"]
				r.Fields["signature"] = json.RawMessage(`"forged"`)
				repo.records[r.ID] = r
			case "HTTP 403":
				s.Fetcher = fixtureFetcher(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) })
			}
			plan, err := s.Prepare(t.Context(), repo.proposal.ID)
			require.NoError(t, err)
			check := runClaim(t, s, plan, 0)
			result, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
			require.NoError(t, err)
			require.Equal(t, Blocked, result.Status)
			require.NotEqual(t, Cleared, result.CurrencyStatus)
			if name == "HTTP 403" {
				require.Equal(t, "HTTP_403", repo.receipt.ClaimChecks[0].FailureCode)
			}
		})
	}
}

func TestFailedUnprocessedDuplicatedAndMisalignedClaims(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	repo.proposal.Citations = append(repo.proposal.Citations, repo.proposal.Citations[0])
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	first := runClaim(t, s, plan, 0)
	result, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{first}})
	require.NoError(t, err)
	require.Equal(t, Partial, result.Status)
	require.Equal(t, "CLAIM_UNPROCESSED", repo.receipt.ClaimChecks[1].FailureCode)
	_, err = s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{first, first}})
	require.ErrorContains(t, err, "duplicate")
	s, repo, _ = serviceFixture(t)
	repo.proposal.Citations = append(repo.proposal.Citations, repo.proposal.Citations[0])
	plan, err = s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	first = runClaim(t, s, plan, 0)
	s.Verifier.Model = nil
	second := runClaim(t, s, plan, 1)
	result, err = s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{second, first}})
	require.NoError(t, err)
	require.Equal(t, Partial, result.Status)
	require.Equal(t, "MODEL_NOT_CONFIGURED", repo.receipt.ClaimChecks[1].FailureCode)
	wrong := first
	wrong.Index = 1
	_, err = s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{first, wrong}})
	require.ErrorContains(t, err, "ordered citation")
}

func TestStaleProposalSourceAndSnapshotTampering(t *testing.T) {
	for _, change := range []string{"proposal", "source", "snapshot"} {
		t.Run(change, func(t *testing.T) {
			s, repo, artifacts := serviceFixture(t)
			plan, err := s.Prepare(t.Context(), repo.proposal.ID)
			require.NoError(t, err)
			snapshot, err := s.SourceSnapshot(t.Context(), ClaimInput{Plan: plan, Index: 0})
			require.NoError(t, err)
			if change == "snapshot" {
				artifacts.bodies[snapshot.Ref.URI][0] ^= 1
				_, err = s.VerifyClaim(t.Context(), ClaimInput{Plan: plan, Index: 0, SnapshotRef: snapshot.Ref})
				require.ErrorContains(t, err, "integrity")
				return
			}
			if change == "proposal" {
				repo.proposal.Version = "sha256:" + strings.Repeat("1", 64)
			} else {
				record := repo.records["source:mcr3215"]
				record.Version = "sha256:" + strings.Repeat("2", 64)
				repo.records[record.ID] = record
			}
			check, err := s.VerifyClaim(t.Context(), ClaimInput{Plan: plan, Index: 0, SnapshotRef: snapshot.Ref})
			require.NoError(t, err)
			require.Equal(t, Stale, check.Status)
			result, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
			require.NoError(t, err)
			require.Equal(t, Stale, result.Status)
			require.False(t, result.DatabaseCommitted)
			require.Zero(t, repo.commits)
		})
	}
}

func TestAbsentSourceRequiresNewSelfCapture(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	p := repo.proposal
	p.Target = "source:new-primary"
	p.ExpectedVersion = "absent"
	p.Citations[0].SourceID = p.Target
	p.Citations[0].SourceVersion = "absent"
	p.ProposedRecord["primary_url"] = json.RawMessage(`"` + fixtureURL + `"`)
	repo.proposal = p
	require.NoError(t, ValidateProposal(p))
	plan, err := s.Prepare(t.Context(), p.ID)
	require.NoError(t, err)
	snapshot, err := s.SourceSnapshot(t.Context(), ClaimInput{Plan: plan, Index: 0})
	require.NoError(t, err)
	require.Equal(t, Fetched, snapshot.Status)
	p.Citations[0].SourceID = "source:other-missing"
	require.Error(t, ValidateProposal(p))
	p.Citations[0].SourceID = p.Target
	p.ExpectedVersion = "sha256:" + strings.Repeat("a", 64)
	require.Error(t, ValidateProposal(p))
}

func TestAuthenticatedCheckCannotBeAlteredWithoutSigningKey(t *testing.T) {
	s, repo, artifacts := serviceFixture(t)
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	check := runClaim(t, s, plan, 0)
	var envelope authenticatedArtifact
	require.NoError(t, json.Unmarshal(artifacts.bodies[check.Ref.URI], &envelope))
	var forged ClaimCheck
	require.NoError(t, json.Unmarshal(envelope.Payload, &forged))
	forged.Claim = "Automatic waiver by signature"
	envelope.Payload, _ = json.Marshal(forged)
	raw, _ := json.Marshal(envelope)
	ref, err := artifacts.Put(t.Context(), "forged", raw, "application/json")
	require.NoError(t, err)
	check.Ref = ref
	_, err = s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
	require.ErrorContains(t, err, "signature")
	require.Zero(t, repo.commits)
	require.False(t, reflect.DeepEqual(forged.Citation, repo.proposal.Citations[0]))
}

func TestNewWorkflowAttemptCanReplaceBlockedReceiptAfterTrustedCurrencyReview(t *testing.T) {
	s, repo, store := serviceFixture(t)
	currency := repo.records["library_currency:mcr3215"]
	delete(repo.records, currency.ID)
	first, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	first.AttemptID = "first-run"
	check := runClaim(t, s, first, 0)
	blocked, err := s.Finish(t.Context(), FinishInput{Plan: first, Checks: []StepResult{check}})
	require.NoError(t, err)
	require.Equal(t, Blocked, blocked.Status)
	prior := repo.receipt
	again, err := s.Finish(t.Context(), FinishInput{Plan: first, Checks: []StepResult{check}})
	require.NoError(t, err)
	require.Equal(t, Blocked, again.Status)
	require.Equal(t, 1, repo.commits)
	repo.records[currency.ID] = currency
	second, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	second.AttemptID = "second-run"
	check = runClaim(t, s, second, 0)
	verified, err := s.Finish(t.Context(), FinishInput{Plan: second, Checks: []StepResult{check}})
	require.NoError(t, err)
	require.Equal(t, Verified, verified.Status)
	require.Equal(t, prior.Signature, repo.receipt.PreviousSignature)
	require.Equal(t, 2, repo.commits)
	var retained Receipt
	require.NoError(t, readSigned(t.Context(), store, blocked.ReceiptRef, &retained, s.SigningKey))
	require.Equal(t, prior, retained, "earlier failed receipt must remain immutable")
}
