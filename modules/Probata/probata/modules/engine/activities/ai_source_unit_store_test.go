// Byline: Codex · GPT-6.1 · 2026-10-07.
package activities

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestAISourceUnitS3CopyPinsVersionAndPreservesNativeMetadata checks the real SDK request and response contract.
// Inputs: local synthetic S3 HTTP responder
// Outputs: exact encoded source version and returned destination version
// Effects: VPS loopback only, no real store
// Choose: to prove server-side PUT-copy does not become payload upload or latest-key copy.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitS3CopyPinsVersionAndPreservesNativeMetadata(t *testing.T) {
	calls := 0
	key := "consignatio/vault/v1/export/a +%?#é.json"
	version := "source-v+/?#"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" {
			t.Errorf("copy invoked %s", r.Method)
		}
		u, e := url.Parse(r.Header.Get("X-Amz-Copy-Source"))
		if e != nil || u.Path != "salem-data/"+key || u.Query().Get("versionId") != version {
			t.Errorf("unpinned/incorrect copy source %q: %v", r.Header.Get("X-Amz-Copy-Source"), e)
		}
		if r.Header.Get("X-Amz-Metadata-Directive") != "COPY" || r.Header.Get("X-Amz-Copy-Source-If-Match") != "\"etag\"" {
			t.Error("source metadata admission lost")
		}
		if r.ContentLength > 0 {
			t.Error("copy uploaded a payload")
		}
		w.Header().Set("X-Amz-Version-Id", "destination-v1")
		w.Header().Set("X-Amz-Copy-Source-Version-Id", version)
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<CopyObjectResult><ETag>"etag"</ETag><LastModified>2026-10-07T00:00:00Z</LastModified></CopyObjectResult>`)
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-west-004", Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "fixture", SecretAccessKey: "fixture"}, nil
	}), HTTPClient: server.Client()}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	store := aiSourceUnitS3{toolkitPlacementS3{smsthreads.S3Store{Client: client}}}
	id, e := store.CopyExact(context.Background(), "salem-data", key, version, aiSourceUnitPrefix+"export/a.json", "\"etag\"")
	if e != nil || id != "destination-v1" || calls != 1 {
		t.Fatalf("copy contract failed id=%q calls=%d err=%v", id, calls, e)
	}
	_, e = store.CopyExact(context.Background(), "salem-data", key, version, "consignatio/casevault/KnowledgeBase/ai-chats/chatgpt/export/a.json", "\"etag\"")
	if e == nil || calls != 1 {
		t.Fatalf("provider-qualified destination escaped neutral route: err=%v calls=%d", e, calls)
	}
}

// TestAISourceUnitS3CopyRejectsMissingAndWrongVersionResponses rejects ambiguous HTTP success responses.
// Inputs: synthetic copy responses with invalid version binding
// Outputs: errors without retries
// Effects: VPS loopback only
// Choose: to retain uncertain operations instead of guessing matching latest.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitS3CopyRejectsMissingAndWrongVersionResponses(t *testing.T) {
	for _, sourceVersion := range []string{"", "wrong-source"} {
		t.Run("source-"+sourceVersion, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Amz-Version-Id", "destination-v1")
				w.Header().Set("X-Amz-Copy-Source-Version-Id", sourceVersion)
				w.Header().Set("Content-Type", "application/xml")
				fmt.Fprint(w, `<CopyObjectResult><ETag>"e"</ETag><LastModified>2026-10-07T00:00:00Z</LastModified></CopyObjectResult>`)
			}))
			defer server.Close()
			client := s3.NewFromConfig(aws.Config{Region: "us-west-004", Credentials: aws.AnonymousCredentials{}, HTTPClient: server.Client()}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
			store := aiSourceUnitS3{toolkitPlacementS3{smsthreads.S3Store{Client: client}}}
			_, e := store.CopyExact(context.Background(), "salem-data", "consignatio/vault/v1/source.json", "source-v1", aiSourceUnitPrefix+"export/a.json", "e")
			if e == nil || calls != 1 {
				t.Fatalf("ambiguous response accepted/retried: %v %d", e, calls)
			}
		})
	}
}

// TestAISourceUnitS3HeadExactRejectsChangedVersion checks exact-version metadata admission independently of body hashes.
// Inputs: synthetic HEAD with a different returned version
// Outputs: error
// Effects: VPS loopback only
// Choose: to catch accidental unversioned metadata reads.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitS3HeadExactRejectsChangedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" || r.URL.Query().Get("versionId") != "source-v1" {
			t.Error("HEAD not version-pinned")
		}
		w.Header().Set("X-Amz-Version-Id", "different")
		w.Header().Set("ETag", "\"e\"")
		w.Header().Set("Content-Length", "7")
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-west-004", Credentials: aws.AnonymousCredentials{}, HTTPClient: server.Client()}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	store := aiSourceUnitS3{toolkitPlacementS3{smsthreads.S3Store{Client: client}}}
	_, e := store.HeadExact(context.Background(), "salem-data", "consignatio/vault/v1/source.json", "source-v1")
	if e == nil || !strings.Contains(e.Error(), "identity") {
		t.Fatal("wrong HEAD version admitted", e)
	}
}

// TestAISourceUnitS3TruncatedMembershipFailsClosed rejects incomplete destination listings.
// Inputs: truncated synthetic S3 response
// Outputs: explicit coverage error
// Effects: VPS loopback only
// Choose: before complete-unit membership claims.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitS3TruncatedMembershipFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<ListBucketResult><IsTruncated>true</IsTruncated></ListBucketResult>`)
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-west-004", Credentials: aws.AnonymousCredentials{}, HTTPClient: server.Client()}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	store := aiSourceUnitS3{toolkitPlacementS3{smsthreads.S3Store{Client: client}}}
	_, e := store.UnitKeys(context.Background(), "salem-data", aiSourceUnitPrefix+"export/")
	if e == nil {
		t.Fatal("truncated membership accepted")
	}
}
