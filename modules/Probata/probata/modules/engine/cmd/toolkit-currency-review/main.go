// Command toolkit-currency-review captures official releases and submits an operator-authorized currency review.
// Inputs: capture-release --url HTTPS_URL or submit-review --file ABSOLUTE_JSON_FILE plus separate mounted credential files.
// Outputs: safe pinned references/status only. Effects: official/B2 reads and derivative writes, currency-only scoped DB commit.
// Choose from an authenticated SSH/operator session; never expose this command or approval key through MCP or a browser.
// A submit file must contain a real reviewer/decision, exact snapshots/release quote/pinpoint and explicit current-review horizon.
// No command invents release evidence, fills a review horizon, invokes NIM for approval or publishes a proposal.
// Byline: Codex · GPT-6.1 · 2026-10-04.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

const MaxReviewFileBytes = 128 << 10

// reviewer exposes only the two existing protected operations; inputs: URL/approved decision; outputs: retained evidence.
type reviewer interface {
	CaptureRelease(context.Context, string) (libraryvalidation.ArtifactRef, error)
	CommitApprovedReview(context.Context, libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyEvidence, error)
}

// dependencies separates local operator authorization/signing from the existing verification/commit service.
// Inputs: protected composition; outputs: narrow CLI seams; effects: none. Choose for fake-dependency tests without network or keys.
type dependencies struct {
	Review reviewer
	Sign   func(libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyReviewApproval, error)
}
type factory func() (dependencies, error)

// main runs one bounded operator command and prints only normalized failure codes on error.
// Inputs: argv/mounted files; outputs: exit status and safe JSON; effects: explicit selected operation only.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, fromEnv))
}

// run validates a literal request before loading secrets, signing the explicit operator submission and calling one operation.
// Inputs: arguments/writers/factory; outputs: process exit code. Effects: described in command docstring; error bodies stay private.
func run(ctx context.Context, args []string, out, diagnostic io.Writer, makeDeps factory) int {
	fail := func(code string) int {
		_ = json.NewEncoder(diagnostic).Encode(map[string]string{"status": libraryvalidation.Blocked, "failure_code": code})
		return 1
	}
	if len(args) == 0 {
		return fail("USAGE_CAPTURE_RELEASE_OR_SUBMIT_REVIEW")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var url, file string
	switch args[0] {
	case "capture-release":
		flags.StringVar(&url, "url", "", "official HTTPS primary release URL")
	case "submit-review":
		flags.StringVar(&file, "file", "", "absolute bounded literal CurrencyReviewApproval JSON file; unsigned")
	default:
		return fail("USAGE_CAPTURE_RELEASE_OR_SUBMIT_REVIEW")
	}
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
		return fail("INPUT_INVALID")
	}
	var approval libraryvalidation.CurrencyReviewApproval
	if args[0] == "capture-release" {
		if _, err := libraryvalidation.PrimaryURL(url); err != nil {
			return fail("OFFICIAL_URL_REQUIRED")
		}
	} else {
		if readApproval(file, &approval) != nil {
			return fail("REVIEW_FILE_INVALID")
		}
	}
	deps, err := makeDeps()
	if err != nil || deps.Review == nil || deps.Sign == nil {
		return fail("PROTECTED_CONFIGURATION_INVALID")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if args[0] == "capture-release" {
		ref, err := deps.Review.CaptureRelease(ctx, url)
		if err != nil {
			return fail("RELEASE_CAPTURE_FAILED")
		}
		if json.NewEncoder(out).Encode(map[string]any{"operation": "capture-release", "status": libraryvalidation.Fetched, "snapshot_ref": ref}) != nil {
			return fail("OUTPUT_FAILED")
		}
		return 0
	}
	// SSH/host operator execution is the authorization boundary. Sign only this explicitly submitted decision.
	approval, err = deps.Sign(approval)
	if err != nil {
		return fail("REVIEW_SIGNING_FAILED")
	}
	evidence, err := deps.Review.CommitApprovedReview(ctx, approval)
	if err != nil || evidence.Status != libraryvalidation.Cleared || !strings.HasPrefix(evidence.SourceID, "source:") {
		return fail("REVIEW_REJECTED")
	}
	result := map[string]any{"operation": "submit-review", "status": evidence.Status, "currency_ref": "library_currency:" + strings.TrimPrefix(evidence.SourceID, "source:"), "approval_ref": evidence.ApprovalRef, "source_snapshot_ref": evidence.SourceSnapshotRef, "release_snapshot_ref": evidence.ReleaseSnapshotRef, "expires_at": evidence.ExpiresAt}
	if json.NewEncoder(out).Encode(result) != nil {
		return fail("OUTPUT_FAILED")
	}
	return 0
}

// readApproval admits one bounded regular literal JSON object with explicit review identities, evidence and horizon.
// Inputs: absolute unsigned review file; outputs: unchanged decision fields; effects: file read only, never shell evaluation.
func readApproval(path string, approval *libraryvalidation.CurrencyReviewApproval) error {
	info, err := os.Lstat(path)
	if !filepath.IsAbs(path) || err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxReviewFileBytes {
		return errors.New("review file unavailable or unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("review file unavailable")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxReviewFileBytes+1))
	if err != nil || len(raw) > MaxReviewFileBytes {
		return errors.New("review file budget exceeded")
	}
	check := json.NewDecoder(strings.NewReader(string(raw)))
	if jsonValue(check, 0) != nil {
		return errors.New("ambiguous review JSON")
	}
	if _, err = check.Token(); err != io.EOF {
		return errors.New("trailing review JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(approval) != nil || approval.Signature != "" || approval.ReviewScope != libraryvalidation.CurrencyReviewScope || strings.TrimSpace(approval.ReviewerID) == "" || strings.TrimSpace(approval.DecisionID) == "" || strings.TrimSpace(approval.ReviewID) == "" || approval.ReleaseVersion == "" || approval.ReleasePinpoint == "" || strings.TrimSpace(approval.ReleaseQuote) == "" || len(approval.ReleaseQuote) > libraryvalidation.MaxPassageBytes || approval.EffectiveAt.IsZero() || approval.ReviewedThrough.IsZero() || approval.ApprovedAt.IsZero() || approval.ExpiresAt.IsZero() {
		return errors.New("review decision incomplete or externally signed")
	}
	for _, ref := range []libraryvalidation.ArtifactRef{approval.SourceSnapshotRef, approval.ReleaseSnapshotRef} {
		if ref.URI == "" || ref.VersionID == "" || ref.Bytes <= 0 || len(ref.SHA256) != 71 || !strings.HasPrefix(ref.SHA256, "sha256:") {
			return errors.New("review requires exact retained references")
		}
	}
	return nil
}

// jsonValue rejects duplicate keys and excessive nesting before a typed JSON decode can silently accept ambiguity.
// Inputs: bounded decoder/depth; outputs: validation error; effects: none. Choose for operator files intended for literal review.
func jsonValue(decoder *json.Decoder, depth int) error {
	if depth > 8 {
		return errors.New("JSON depth budget")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delim == '{' {
		seen := map[string]bool{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate JSON key")
			}
			seen[name] = true
			if err = jsonValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else if delim == '[' {
		for decoder.More() {
			if err = jsonValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else {
		return errors.New("invalid JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}
