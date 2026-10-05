// Byline: Codex · GPT-6.1 · 2026-10-04.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/stretchr/testify/require"
)

type fakeReviewer struct {
	captures, submits int
	approval          libraryvalidation.CurrencyReviewApproval
	fail              bool
}

func (f *fakeReviewer) CaptureRelease(_ context.Context, _ string) (libraryvalidation.ArtifactRef, error) {
	f.captures++
	if f.fail {
		return libraryvalidation.ArtifactRef{}, errors.New("private quote and secret must not print")
	}
	return fixtureRef(), nil
}
func (f *fakeReviewer) CommitApprovedReview(_ context.Context, a libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyEvidence, error) {
	f.submits++
	f.approval = a
	if f.fail {
		return libraryvalidation.CurrencyEvidence{}, errors.New("private quote and secret must not print")
	}
	return libraryvalidation.CurrencyEvidence{SourceID: "source:fixture", Status: libraryvalidation.Cleared, ApprovalRef: fixtureRef(), SourceSnapshotRef: a.SourceSnapshotRef, ReleaseSnapshotRef: a.ReleaseSnapshotRef, ExpiresAt: a.ExpiresAt}, nil
}
func fixtureRef() libraryvalidation.ArtifactRef {
	return libraryvalidation.ArtifactRef{URI: "b2://fixture/validation?versionId=v1", VersionID: "v1", SHA256: "sha256:" + strings.Repeat("a", 64), Bytes: 100}
}
func fixtureApproval() libraryvalidation.CurrencyReviewApproval {
	now := time.Now().UTC()
	return libraryvalidation.CurrencyReviewApproval{ReviewID: "fixture-review", ReviewerID: "real-authorized-reviewer-fixture", DecisionID: "explicit-decision-fixture", ReviewScope: libraryvalidation.CurrencyReviewScope, SourceSnapshotRef: fixtureRef(), ReleaseSnapshotRef: fixtureRef(), ReleaseVersion: "fixture-release", ReleasePinpoint: "page:1", ReleaseQuote: "Private exact release quote fixture.", EffectiveAt: now.Add(-time.Hour), ReviewedThrough: now, ApprovedAt: now, ExpiresAt: now.Add(time.Hour)}
}

func TestOperatorCommandsCallOnlySelectedOperationAndKeepPrivateFieldsOutOfOutput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "review.json")
	a := fixtureApproval()
	raw, _ := json.Marshal(a)
	require.NoError(t, os.WriteFile(file, raw, 0o600))
	for _, command := range [][]string{{"capture-release", "--url", "https://www.courts.michigan.gov/fixture"}, {"submit-review", "--file", file}} {
		f := &fakeReviewer{}
		signed := 0
		makeDeps := func() (dependencies, error) {
			return dependencies{Review: f, Sign: func(input libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyReviewApproval, error) {
				signed++
				input.Signature = "protected-fixture-signature"
				return input, nil
			}}, nil
		}
		var out, diagnostic bytes.Buffer
		require.Zero(t, run(t.Context(), command, &out, &diagnostic, makeDeps))
		require.Empty(t, diagnostic.String())
		require.NotContains(t, out.String(), a.ReleaseQuote)
		require.NotContains(t, out.String(), a.ReviewerID)
		require.NotContains(t, out.String(), "protected-fixture-signature")
		if command[0] == "capture-release" {
			require.Equal(t, 1, f.captures)
			require.Zero(t, f.submits)
			require.Zero(t, signed)
		} else {
			require.Zero(t, f.captures)
			require.Equal(t, 1, f.submits)
			require.Equal(t, 1, signed)
			require.Equal(t, a.ReviewedThrough, f.approval.ReviewedThrough)
		}
	}
}

func TestOperatorFileRejectsAmbiguityFlagsUnsignedIdentityAndBudgetBeforeConfig(t *testing.T) {
	a := fixtureApproval()
	raw, _ := json.Marshal(a)
	for _, mode := range []string{"unknown clearance flag", "duplicate reviewer", "trailing JSON", "externally signed", "missing reviewer", "body budget"} {
		t.Run(mode, func(t *testing.T) {
			body := string(raw)
			switch mode {
			case "unknown clearance flag":
				body = strings.Replace(body, "{", `{"currency_status":"cleared",`, 1)
			case "duplicate reviewer":
				body = strings.Replace(body, "{", `{"reviewer_id":"different",`, 1)
			case "trailing JSON":
				body += "{}"
			case "externally signed":
				copy := a
				copy.Signature = "client-forged"
				b, _ := json.Marshal(copy)
				body = string(b)
			case "missing reviewer":
				copy := a
				copy.ReviewerID = ""
				b, _ := json.Marshal(copy)
				body = string(b)
			case "body budget":
				body = strings.Repeat("x", MaxReviewFileBytes+1)
			}
			file := filepath.Join(t.TempDir(), "review.json")
			require.NoError(t, os.WriteFile(file, []byte(body), 0o600))
			loaded := 0
			var out, diagnostic bytes.Buffer
			require.NotZero(t, run(t.Context(), []string{"submit-review", "--file", file}, &out, &diagnostic, func() (dependencies, error) { loaded++; return dependencies{}, nil }))
			require.Zero(t, loaded)
			require.Empty(t, out.String())
			require.Contains(t, diagnostic.String(), "REVIEW_FILE_INVALID")
			require.NotContains(t, diagnostic.String(), a.ReleaseQuote)
		})
	}
}

func TestRejectedReviewReportsOnlySafeFailureCode(t *testing.T) {
	file := filepath.Join(t.TempDir(), "review.json")
	raw, _ := json.Marshal(fixtureApproval())
	require.NoError(t, os.WriteFile(file, raw, 0o600))
	var out, diagnostic bytes.Buffer
	require.NotZero(t, run(t.Context(), []string{"submit-review", "--file", file}, &out, &diagnostic, func() (dependencies, error) {
		return dependencies{Review: &fakeReviewer{fail: true}, Sign: func(a libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyReviewApproval, error) {
			return a, nil
		}}, nil
	}))
	require.Empty(t, out.String())
	require.Contains(t, diagnostic.String(), "REVIEW_REJECTED")
	require.NotContains(t, diagnostic.String(), "private quote")
}
