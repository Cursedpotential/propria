// Byline: Codex · GPT-6.1 · 2026-10-05. Isolated HTTPS fixtures only; no real B2 writes.
package librarysync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestSDKTransportPinsVersionPreservesIntentAndNeverRetriesPUT(t *testing.T) {
	_, _, backend := fixtureService(t)
	op := backend.claim.Operation
	raw := backend.payload
	var puts atomic.Int32
	var gets atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+op.Bucket+"/"+op.Key || r.Header.Get("Authorization") == "" {
			t.Error("SDK request lost scoped key or server-side signing")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodPut {
			puts.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Equal(body, raw) || r.Header.Get("X-Amz-Meta-Toolkit-Operation-Id") != op.OperationID || r.Header.Get("X-Amz-Meta-Toolkit-Intent-Id") != "durable-intent" || r.Header.Get("X-Amz-Meta-Toolkit-Payload-Sha256") != op.PayloadSHA256 {
				t.Error("PUT lost exact bytes or immutable operation/intent evidence")
			}
			// Simulate a retryable failed reply after the server has consumed the body.
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `<Error><Code>InternalError</Code><Message>synthetic failure</Message></Error>`)
			return
		}
		if r.Method != http.MethodGet || r.URL.Query().Get("versionId") != "exact/version+identity" {
			t.Error("GET was not pinned to the original provider VersionId")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gets.Add(1)
		w.Header().Set("X-Amz-Version-Id", "exact/version+identity")
		w.Header().Set("X-Amz-Meta-Toolkit-Operation-Id", op.OperationID)
		w.Header().Set("X-Amz-Meta-Toolkit-Intent-Id", "durable-intent")
		w.Header().Set("Content-Type", op.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
		w.Write(raw)
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-west-004", HTTPClient: server.Client(), Credentials: credentials.NewStaticCredentialsProvider("synthetic-key", "synthetic-secret", "")}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	storage := B2Storage{Client: client, Scope: fixtureScope}
	if _, err := storage.Put(context.Background(), op, raw, "durable-intent"); err == nil || puts.Load() != 1 {
		t.Fatalf("uncertain PUT must stay visible and single-attempt: count=%d error=%v", puts.Load(), err)
	}
	obj := Object{Bucket: op.Bucket, Key: op.Key, VersionID: "exact/version+identity", Size: int64(len(raw))}
	hashed, err := storage.Hash(context.Background(), obj)
	requireNoError(t, err)
	read, actual, err := storage.Read(context.Background(), obj, MaxPayloadBytes)
	requireNoError(t, err)
	if gets.Load() != 2 || hashed.SHA256 != op.PayloadSHA256 || hashed.IntentID != "durable-intent" || actual.OperationID != op.OperationID || actual.IntentID != "durable-intent" || !bytes.Equal(read, raw) {
		t.Fatal(fmt.Sprintf("pinned byte/metadata evidence lost: reads=%d", gets.Load()))
	}
}
