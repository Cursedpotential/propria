// Byline: Codex · GPT-6.1 · 2026-10-05.
package librarysync

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type sdkFixture struct {
	list func(*s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error)
	head func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	get  func(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
	put  func(*s3.PutObjectInput, []func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (f sdkFixture) ListObjectVersions(_ context.Context, in *s3.ListObjectVersionsInput, _ ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	if f.list != nil {
		return f.list(in)
	}
	return &s3.ListObjectVersionsOutput{}, nil
}
func (f sdkFixture) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if f.head != nil {
		return f.head(in)
	}
	return nil, errors.New("unexpected HEAD")
}
func (f sdkFixture) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.get != nil {
		return f.get(in)
	}
	return nil, errors.New("unexpected GET")
}
func (f sdkFixture) PutObject(_ context.Context, in *s3.PutObjectInput, options ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.put != nil {
		return f.put(in, options)
	}
	return nil, errors.New("unexpected PUT")
}

func TestPinnedStorageChecksVersionLengthAndCompleteHash(t *testing.T) {
	for _, kind := range []string{"valid", "wrong-version", "wrong-length", "overflow", "truncated", "over-budget"} {
		t.Run(kind, func(t *testing.T) {
			raw := "%PDF-1.7 synthetic"
			obj := Object{Bucket: fixtureScope.Bucket, Key: fixtureScope.LegalRoot + "case-law/a.pdf", VersionID: "exact-version", Size: int64(len(raw))}
			calls := 0
			b := B2Storage{Scope: fixtureScope, Client: sdkFixture{get: func(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				calls++
				if aws.ToString(in.VersionId) != obj.VersionID {
					t.Fatal("unpinned GET")
				}
				version := obj.VersionID
				size := obj.Size
				body := raw
				switch kind {
				case "wrong-version":
					version = "latest-other"
				case "wrong-length":
					size++
				case "overflow":
					body += "extra"
				case "truncated":
					body = body[:3]
				}
				return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(body)), VersionId: aws.String(version), ContentLength: aws.Int64(size), ContentType: aws.String("application/pdf"), Metadata: map[string]string{"toolkit-operation-id": fixtureOperation, "toolkit-intent-id": "intent-1"}}, nil
			}}}
			if kind == "over-budget" {
				obj.Size = MaxOriginalBytes + 1
			}
			out, err := b.Hash(context.Background(), obj)
			if kind == "valid" {
				requireNoError(t, err)
				if out.SHA256 != digest([]byte(raw)) || out.IntentID != "intent-1" || out.OperationID != fixtureOperation {
					t.Fatal("full-byte or intent evidence lost")
				}
				bytes, meta, err := b.Read(context.Background(), obj, MaxOriginalBytes)
				requireNoError(t, err)
				if string(bytes) != raw || meta.IntentID != "intent-1" {
					t.Fatal("pinned read mismatch")
				}
			} else if err == nil {
				t.Fatalf("%s accepted", kind)
			}
			if kind == "over-budget" && calls != 0 {
				t.Fatal("budget checked after GET")
			}
		})
	}
}

func TestHistoryDetectsNonAdjacentCursorCyclesAndNeighborBudget(t *testing.T) {
	for _, kind := range []string{"cycle", "neighbors"} {
		t.Run(kind, func(t *testing.T) {
			key := fixtureScope.LegalRoot + "reference-data/a.md"
			calls := 0
			b := B2Storage{Scope: fixtureScope, Client: sdkFixture{list: func(in *s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
				calls++
				if aws.ToInt32(in.MaxKeys) != MaxPage || aws.ToString(in.Prefix) != key {
					t.Fatal("unbounded history query")
				}
				next := ""
				if kind == "cycle" {
					switch calls {
					case 1:
						next = key + "A"
					case 2:
						next = key + "B"
					default:
						next = key + "A"
					}
				} else {
					next = key + strings.Repeat("x", calls)
				}
				return &s3.ListObjectVersionsOutput{IsTruncated: aws.Bool(true), NextKeyMarker: aws.String(next), NextVersionIdMarker: aws.String("cursor-version"), Versions: []s3types.ObjectVersion{{Key: aws.String(key + "neighbor"), VersionId: aws.String("neighbor-v"), Size: aws.Int64(4), LastModified: &fixtureNow}}}, nil
			}}}
			p, err := b.History(context.Background(), key)
			if kind == "cycle" {
				if err == nil || calls != 3 {
					t.Fatalf("cursor cycle not bounded %d %v", calls, err)
				}
			} else {
				requireNoError(t, err)
				if p.Complete || calls != MaxHistoryPages || len(p.Objects) != 0 {
					t.Fatal("neighbor pages fabricated complete exact-key absence")
				}
			}
		})
	}
}

func TestListRetainsHideMarkersAndRejectsBadPagination(t *testing.T) {
	key := fixtureScope.LegalRoot + "benchbooks/a.pdf"
	b := B2Storage{Scope: fixtureScope, Client: sdkFixture{list: func(*s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
		return &s3.ListObjectVersionsOutput{Versions: []s3types.ObjectVersion{{Key: aws.String(key), VersionId: aws.String("original"), Size: aws.Int64(12), LastModified: &fixtureNow}}, DeleteMarkers: []s3types.DeleteMarkerEntry{{Key: aws.String(key), VersionId: aws.String("hide"), IsLatest: aws.Bool(true), LastModified: &fixtureNow}}}, nil
	}}}
	p, err := b.List(context.Background(), "benchbooks", Cursor{})
	requireNoError(t, err)
	if !p.Complete || len(p.Objects) != 2 || !p.Objects[1].Hidden {
		t.Fatal("hide/version evidence lost")
	}
	b.Client = sdkFixture{list: func(*s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
		return &s3.ListObjectVersionsOutput{IsTruncated: aws.Bool(true)}, nil
	}}
	if _, err = b.List(context.Background(), "benchbooks", Cursor{}); err == nil {
		t.Fatal("missing cursor accepted")
	}
	if _, err = b.List(context.Background(), "other", Cursor{}); err == nil {
		t.Fatal("unapproved child listed")
	}
}

func TestPUTDisablesSDKRetriesAndPreservesExactIntent(t *testing.T) {
	_, _, back := fixtureService(t)
	calls := 0
	b := B2Storage{Scope: fixtureScope, Client: sdkFixture{put: func(in *s3.PutObjectInput, options []func(*s3.Options)) (*s3.PutObjectOutput, error) {
		calls++
		config := s3.Options{RetryMaxAttempts: 3}
		for _, apply := range options {
			apply(&config)
		}
		if config.RetryMaxAttempts != 1 || in.Metadata["toolkit-intent-id"] != "intent-1" || in.Metadata["toolkit-operation-id"] != fixtureOperation {
			t.Fatal("unsafe retry or intent missing")
		}
		raw, err := io.ReadAll(in.Body)
		requireNoError(t, err)
		if string(raw) != string(back.payload) {
			t.Fatal("export bytes changed")
		}
		return nil, errors.New("synthetic lost reply")
	}}}
	if _, err := b.Put(context.Background(), back.claim.Operation, back.payload, "intent-1"); err == nil || calls != 1 {
		t.Fatal("PUT uncertainty not visible")
	}
	op := back.claim.Operation
	op.Key = strings.TrimSuffix(op.Key, ".md") + ".pdf"
	if _, err := b.Put(context.Background(), op, back.payload, "intent-1"); err == nil || calls != 1 {
		t.Fatal("format-changing write reached provider")
	}
}

func TestHEADPreservesIntentAndRejectsUnversionedDescriptor(t *testing.T) {
	key := fixtureScope.LegalRoot + "reference-data/a.md"
	b := B2Storage{Scope: fixtureScope, Client: sdkFixture{head: func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{VersionId: aws.String("v"), ContentLength: aws.Int64(12), ContentType: aws.String("text/markdown"), Metadata: map[string]string{"toolkit-operation-id": fixtureOperation, "toolkit-intent-id": "intent-1"}}, nil
	}}}
	obj, err := b.Head(context.Background(), key)
	requireNoError(t, err)
	if obj.IntentID != "intent-1" {
		t.Fatal("HEAD lost exact intent")
	}
	b.Client = sdkFixture{head: func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{ContentLength: aws.Int64(12)}, nil
	}}
	if _, err = b.Head(context.Background(), key); err == nil {
		t.Fatal("unversioned HEAD accepted")
	}
}
