// Byline: Codex · GPT-6 · 2026-10-04.
package smsthreads

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// newCreateOnlyTestStore wires the real S3 SDK client to an in-process HTTP fixture.
// Inputs: test handle and fixture handler; outputs: an S3Store using throwaway credentials and a cleanup-registered server.
// Side effects: opens only a loopback test server. Choose for exercising serialized S3 conditions without network transfer.
// Byline: Codex · GPT-6 · 2026-10-04.
func newCreateOnlyTestStore(t *testing.T, handler http.Handler) S3Store {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := s3.New(s3.Options{
		Region: "us-east-1", BaseEndpoint: aws.String(server.URL), UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
		HTTPClient:  server.Client(),
	})
	return S3Store{Client: client}
}

// TestS3StorePutIfAbsentUsesConditionalHeader proves the adapter transmits a provider-enforced create-only condition.
// Inputs: one small in-memory object and HTTP fixture; outputs: request/header assertions.
// Side effects: one loopback PUT only. Choose to verify SDK serialization without contacting B2 or writing an object.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestS3StorePutIfAbsentUsesConditionalHeader(t *testing.T) {
	putCount := 0
	store := newCreateOnlyTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method %s", r.Method)
		}
		putCount++
		if got := r.Header.Get("If-None-Match"); got != "*" {
			t.Errorf("If-None-Match = %q, want *", got)
		}
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	if err := store.PutIfAbsent(context.Background(), "salem-data", "consignatio/casevault/recovery/library-sources/a.zip", strings.NewReader("whole zip"), int64(len("whole zip")), "application/zip"); err != nil {
		t.Fatal(err)
	}
	if putCount != 1 {
		t.Fatalf("PUT count = %d, want 1", putCount)
	}
}

// TestS3StorePutIfAbsentReplaysOnlyMatchingBytes proves a known 412 causes bounded remote hashing before reuse.
// Inputs: conditional-conflict and remote-GET fixture responses; outputs: matching acceptance and mismatch refusal.
// Side effects: loopback PUT and at most one loopback GET; no external object is created.
// Choose to validate idempotent retries without treating every service error as an existing object.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestS3StorePutIfAbsentReplaysOnlyMatchingBytes(t *testing.T) {
	for _, test := range []struct {
		name    string
		remote  string
		wantErr error
	}{
		{name: "identical", remote: "whole zip"},
		{name: "different", remote: "other bytes", wantErr: ErrConditionalObjectMismatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			putCount, getCount := 0, 0
			store := newCreateOnlyTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPut:
					putCount++
					if r.Header.Get("If-None-Match") != "*" {
						t.Errorf("missing create-only header: %v", r.Header)
					}
					_, _ = io.Copy(io.Discard, r.Body)
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusPreconditionFailed)
					_, _ = io.WriteString(w, "<Error><Code>PreconditionFailed</Code><Message>exists</Message></Error>")
				case http.MethodGet:
					getCount++
					w.Header().Set("Content-Length", strconv.Itoa(len(test.remote)))
					w.Header().Set("ETag", `"fixture-etag"`)
					_, _ = io.WriteString(w, test.remote)
				default:
					t.Errorf("unexpected method %s", r.Method)
					http.Error(w, "unexpected", http.StatusMethodNotAllowed)
				}
			}))
			err := store.PutIfAbsent(context.Background(), "salem-data", "archive.zip", strings.NewReader("whole zip"), int64(len("whole zip")), "application/zip")
			if test.wantErr == nil && err != nil {
				t.Fatalf("matching conflict replay failed: %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("mismatch error = %v, want errors.Is(%v)", err, test.wantErr)
			}
			if putCount != 1 || getCount != 1 {
				t.Fatalf("requests PUT=%d GET=%d, want one each", putCount, getCount)
			}
		})
	}
}

// TestS3StorePutIfAbsentDoesNotReplayUnrelatedErrors proves only known condition conflicts trigger readback.
// Inputs: an AccessDenied response; outputs: an error and zero GET requests.
// Side effects: one loopback PUT only. Choose to guard against masking credentials or provider failures as retries.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestS3StorePutIfAbsentDoesNotReplayUnrelatedErrors(t *testing.T) {
	getCount := 0
	store := newCreateOnlyTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getCount++
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "<Error><Code>AccessDenied</Code></Error>")
	}))
	err := store.PutIfAbsent(context.Background(), "salem-data", "archive.zip", strings.NewReader("whole zip"), int64(len("whole zip")), "application/zip")
	if err == nil {
		t.Fatal("AccessDenied was accepted")
	}
	if getCount != 0 {
		t.Fatalf("unrelated error triggered %d readback GETs", getCount)
	}
}

// TestS3StorePutIfAbsentBoundsBodySize rejects oversized and falsely sized bodies before any HTTP call.
// Inputs: oversized or shorter-than-declared readers; outputs: validation errors.
// Side effects: none because the local SDK endpoint increments a counter only if called.
// Choose to prove create-only upload memory and declared-size ceilings.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestS3StorePutIfAbsentBoundsBodySize(t *testing.T) {
	requests := 0
	store := newCreateOnlyTestStore(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	if err := store.PutIfAbsent(context.Background(), "bucket", "key", strings.NewReader("short"), 6, "application/zip"); err == nil {
		t.Fatal("false declared size was accepted")
	}
	if err := store.PutIfAbsent(context.Background(), "bucket", "key", strings.NewReader("x"), maxConditionalPutBytes+1, "application/zip"); err == nil {
		t.Fatal("oversized conditional PUT was accepted")
	}
	if requests != 0 {
		t.Fatalf("invalid bodies triggered %d requests", requests)
	}
}
