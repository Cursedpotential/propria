// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// S3Store.ReadRange against an S3-shaped HTTP server: the request on the wire
// carries a Range header, and the reader never takes more than the range even
// from a server that ignores it. Unit test — not a read from real B2.

package smsthreads

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func rangeTestStore(t *testing.T, handler http.HandlerFunc) S3Store {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return S3Store{Client: s3.New(s3.Options{
		BaseEndpoint: aws.String(server.URL), UsePathStyle: true, Region: "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("test-id", "test-secret", ""),
	})}
}

func TestReadRangeSendsARangedGet(t *testing.T) {
	object := bytes.Repeat([]byte("s"), 200_000)
	var gotRange, gotPath, gotMethod string
	store := rangeTestStore(t, func(w http.ResponseWriter, r *http.Request) {
		gotRange, gotPath, gotMethod = r.Header.Get("Range"), r.URL.Path, r.Method
		w.Header().Set("Content-Range", "bytes 0-65535/200000")
		w.Header().Set("Content-Length", "65536")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(object[:65536])
	})
	data, err := store.ReadRange(context.Background(), "salem-data", "consignatio/vault/v1/sms-1.xml", 0, 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotRange != "bytes=0-65535" || gotPath != "/salem-data/consignatio/vault/v1/sms-1.xml" {
		t.Fatalf("request = %s %s Range=%q", gotMethod, gotPath, gotRange)
	}
	if len(data) != 65536 {
		t.Fatalf("read %d bytes, want 65536", len(data))
	}
}

func TestReadRangeNeverReadsPastTheRange(t *testing.T) {
	object := bytes.Repeat([]byte("s"), 200_000)
	store := rangeTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
		// A server that ignores Range and sends the whole object.
		w.Header().Set("Content-Length", "200000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(object)
	})
	data, err := store.ReadRange(context.Background(), "b", "k", 0, 64<<10)
	if err != nil || len(data) != 64<<10 {
		t.Fatalf("read %d bytes err=%v, want exactly the 64 KiB range", len(data), err)
	}
}

func TestReadRangeOfAnEmptyObjectIsEmpty(t *testing.T) {
	store := rangeTestStore(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>InvalidRange</Code><Message>The requested range is not satisfiable</Message></Error>`))
	})
	data, err := store.ReadRange(context.Background(), "b", "k", 0, 64<<10)
	if err != nil || len(data) != 0 {
		t.Fatalf("empty object read = %d bytes err=%v", len(data), err)
	}
}

func TestReadRangeRejectsUnboundedRequests(t *testing.T) {
	store := S3Store{Client: s3.New(s3.Options{Region: "us-east-1"})}
	for _, bad := range [][2]int64{{-1, 10}, {0, 0}, {0, maxRangeRead + 1}} {
		if _, err := store.ReadRange(context.Background(), "b", "k", bad[0], bad[1]); err == nil {
			t.Fatalf("range %v accepted", bad)
		}
	}
}
