// Byline: Codex · GPT-6 · 2026-10-04.
package smsthreads

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// versionedS3Fixture models a small versioned S3 endpoint and records requested version IDs.
// Inputs: HTTP methods and object state; outputs: version-aware S3 responses and bounded call evidence.
// Side effects: mutates only in-memory synthetic fixture state. Choose over live B2 access for adapter contract tests.
// Byline: Codex · GPT-6 · 2026-10-04.
type versionedS3Fixture struct {
	mu               sync.Mutex
	versions         map[string]map[string][]byte
	putCount         int
	nextVersion      int
	omitPutVersion   bool
	putVersionValue  string
	headVersionValue string
	losePutReply     bool
	getVersions      []string
}

// ServeHTTP implements only the bounded HEAD, PUT and exact-version GET paths under test.
// Inputs: one SDK-generated request; outputs: S3-compatible status, body and VersionId headers.
// Side effects: mutates synthetic versions/call counters only. Choose as the HTTP peer for S3Store tests.
// Byline: Codex · GPT-6 · 2026-10-04.
func (f *versionedS3Fixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	switch r.Method {
	case http.MethodHead:
		versions := f.versions[key]
		if len(versions) == 0 {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}
		latestID, data := latestFixtureVersion(versions)
		if f.headVersionValue != "" {
			latestID = f.headVersionValue
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.Header().Set("x-amz-version-id", latestID)
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f.putCount++
		f.nextVersion++
		versionID := fmt.Sprintf("version-%d", f.nextVersion)
		if f.versions[key] == nil {
			f.versions[key] = make(map[string][]byte)
		}
		f.versions[key][versionID] = append([]byte(nil), body...)
		if f.losePutReply {
			f.losePutReply = false
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !f.omitPutVersion {
			if f.putVersionValue != "" {
				w.Header().Set("x-amz-version-id", f.putVersionValue)
			} else {
				w.Header().Set("x-amz-version-id", versionID)
			}
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		versionID := r.URL.Query().Get("versionId")
		f.getVersions = append(f.getVersions, versionID)
		data, ok := f.versions[key][versionID]
		if !ok || versionID == "" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.Header().Set("x-amz-version-id", versionID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// latestFixtureVersion returns the highest test-created version and its bytes.
// Inputs: one nonempty fixture version map; outputs: latest ID and body. Side effects: none.
// Choose for HEAD response behavior in this deterministic synthetic endpoint.
// Byline: Codex · GPT-6 · 2026-10-04.
func latestFixtureVersion(versions map[string][]byte) (string, []byte) {
	var latest string
	for id := range versions {
		if id > latest {
			latest = id
		}
	}
	return latest, versions[latest]
}

// newVersionedS3Store builds an S3 SDK client pointed only at a local synthetic HTTP server.
// Inputs: test and fixture handler; outputs: S3Store plus server cleanup registered with testing.
// Side effects: starts and closes one loopback test server. Choose to verify SDK VersionId serialization without network access.
// Byline: Codex · GPT-6 · 2026-10-04.
func newVersionedS3Store(t *testing.T, fixture *versionedS3Fixture) S3Store {
	t.Helper()
	server := httptest.NewServer(fixture)
	t.Cleanup(server.Close)
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(aws.Config{
		Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("synthetic", "synthetic", ""),
	}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint.String())
		options.UsePathStyle = true
		options.Retryer = aws.NopRetryer{}
	})
	return S3Store{Client: client}
}

// TestPutRecoveredVersionReusesMatchingAndLostResponseVersions proves retry paths never create a duplicate version.
// Inputs: synthetic bytes and an in-memory S3 HTTP fixture; outputs: exact reused VersionIds and PUT count.
// Side effects: mutates only synthetic fixture state. Choose to validate preflight and lost-response idempotency.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestPutRecoveredVersionReusesMatchingAndLostResponseVersions(t *testing.T) {
	for _, tc := range []struct {
		name         string
		loseResponse bool
	}{
		{name: "matching replay"},
		{name: "lost response", loseResponse: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &versionedS3Fixture{versions: map[string]map[string][]byte{}, losePutReply: tc.loseResponse}
			store := newVersionedS3Store(t, fixture)
			body := []byte("synthetic complete archive bytes")
			digest := sha256.Sum256(body)
			versionID, err := store.PutRecoveredVersion(context.Background(), "bucket", "recovery/run-a/unit.zip", strings.NewReader(string(body)), int64(len(body)), "application/zip", hex.EncodeToString(digest[:]), nil)
			if err != nil {
				t.Fatal(err)
			}
			if versionID != "version-1" {
				t.Fatalf("unexpected VersionId %q", versionID)
			}
			replayed, err := store.PutRecoveredVersion(context.Background(), "bucket", "recovery/run-a/unit.zip", strings.NewReader(string(body)), int64(len(body)), "application/zip", hex.EncodeToString(digest[:]), nil)
			if err != nil || replayed != versionID {
				t.Fatalf("matching replay got %q, %v; want %q", replayed, err, versionID)
			}
			if fixture.putCount != 1 {
				t.Fatalf("expected one PUT, got %d", fixture.putCount)
			}
			if len(fixture.getVersions) < 1 || fixture.getVersions[0] != versionID {
				t.Fatalf("readback did not request exact VersionId: %v", fixture.getVersions)
			}
		})
	}
}

// TestPutRecoveredVersionRefusesMismatchAndMissingVersionID covers fail-closed provider outcomes.
// Inputs: synthetic existing bytes or a PUT response without VersionId; outputs: explicit errors and no overwrite.
// Side effects: mutates only synthetic fixture state. Choose to prove conflicting keys and unversioned responses are never accepted.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestPutRecoveredVersionRefusesMismatchAndMissingVersionID(t *testing.T) {
	t.Run("existing mismatch", func(t *testing.T) {
		fixture := &versionedS3Fixture{versions: map[string]map[string][]byte{"recovery/existing.zip": {"old-version": []byte("different retained bytes")}}}
		store := newVersionedS3Store(t, fixture)
		body := []byte("new content")
		digest := sha256.Sum256(body)
		_, err := store.PutRecoveredVersion(context.Background(), "bucket", "recovery/existing.zip", strings.NewReader(string(body)), int64(len(body)), "application/zip", hex.EncodeToString(digest[:]), nil)
		if err != ErrRecoveredVersionMismatch {
			t.Fatalf("expected mismatch sentinel, got %v", err)
		}
		if fixture.putCount != 0 {
			t.Fatalf("mismatched key was written: PUT count %d", fixture.putCount)
		}
	})
	t.Run("missing VersionId", func(t *testing.T) {
		fixture := &versionedS3Fixture{versions: map[string]map[string][]byte{}, omitPutVersion: true}
		store := newVersionedS3Store(t, fixture)
		body := []byte("synthetic bytes")
		digest := sha256.Sum256(body)
		if _, err := store.PutRecoveredVersion(context.Background(), "bucket", "recovery/new.zip", strings.NewReader(string(body)), int64(len(body)), "application/zip", hex.EncodeToString(digest[:]), nil); err == nil || !strings.Contains(err.Error(), "valid retained VersionId") {
			t.Fatalf("missing VersionId was not rejected: %v", err)
		}
	})
	t.Run("null VersionId", func(t *testing.T) {
		fixture := &versionedS3Fixture{versions: map[string]map[string][]byte{}, putVersionValue: "null"}
		store := newVersionedS3Store(t, fixture)
		body := []byte("synthetic bytes")
		digest := sha256.Sum256(body)
		if _, err := store.PutRecoveredVersion(context.Background(), "bucket", "recovery/null.zip", strings.NewReader(string(body)), int64(len(body)), "application/zip", hex.EncodeToString(digest[:]), nil); err == nil {
			t.Fatal("null VersionId was accepted")
		}
	})
	t.Run("HEAD and GET reject invalid IDs", func(t *testing.T) {
		fixture := &versionedS3Fixture{
			versions:         map[string]map[string][]byte{"recovery/head-null.zip": {"version-1": []byte("bytes")}},
			headVersionValue: "null",
		}
		store := newVersionedS3Store(t, fixture)
		if _, err := store.HeadVersion(context.Background(), "bucket", "recovery/head-null.zip"); err == nil {
			t.Fatal("HEAD accepted null VersionId")
		}
		if _, err := store.OpenVersion(context.Background(), "bucket", "recovery/head-null.zip", "null"); err == nil {
			t.Fatal("GET accepted null VersionId")
		}
		if _, err := store.OpenVersion(context.Background(), "bucket", "recovery/head-null.zip", strings.Repeat("v", 2049)); err == nil {
			t.Fatal("GET accepted VersionId longer than 2048 bytes")
		}
	})
}

// TestOpenVersionSelectsRequestedVersion proves SDK GETs carry and enforce the caller's exact version ID.
// Inputs: two retained synthetic versions at one key; outputs: requested older body and observed query.
// Side effects: reads only fixture versions. Choose to validate version pinning independent of latest-key state.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestOpenVersionSelectsRequestedVersion(t *testing.T) {
	fixture := &versionedS3Fixture{versions: map[string]map[string][]byte{
		"recovery/versioned.zip": {"version-1": []byte("older archive"), "version-2": []byte("newer archive")},
	}}
	store := newVersionedS3Store(t, fixture)
	stream, err := store.OpenVersion(context.Background(), "bucket", "recovery/versioned.zip", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	got, err := io.ReadAll(stream)
	if err != nil || string(got) != "older archive" {
		t.Fatalf("exact older-version GET got %q, %v", got, err)
	}
	if len(fixture.getVersions) != 1 || fixture.getVersions[0] != "version-1" {
		t.Fatalf("requested VersionId was not selected: %v", fixture.getVersions)
	}
}
