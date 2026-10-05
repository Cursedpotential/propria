// Byline: Codex · GPT-6.1 · 2026-10-04.
package librarysync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage exposes bounded retained-version primitives, with no delete/hide API.
// Inputs: admitted metadata and opaque bytes; outputs: exact provider identities. Effects: separate list/read/PUT calls.
type Storage interface {
	List(context.Context, string, Cursor) (Page, error)
	Head(context.Context, string) (*Object, error)
	Read(context.Context, Object, int64) ([]byte, Object, error)
	Hash(context.Context, Object) (Object, error)
	Put(context.Context, Operation, []byte, string) (Object, error)
	History(context.Context, string) (Page, error)
}

// S3API is the existing AWS SDK's minimal B2 surface; inputs/outputs: SDK values; effects: only allowed storage operations.
type S3API interface {
	ListObjectVersions(context.Context, *s3.ListObjectVersionsInput, ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// B2Config holds mounted S3-compatible credentials, never returned or logged.
// Inputs: existing literal JSON config; outputs: B2 client construction. Effects: none in this type.
type B2Config struct {
	Endpoint        string `json:"endpoint_url"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

// B2Storage uses existing AWS SDK dependencies with bounded requests and exact-version checks.
// Inputs: client and explicit scope; outputs: Storage primitives. Effects: no runtime/deployment changes.
type B2Storage struct {
	Client S3API
	Scope  Scope
}

// NewB2Storage constructs a B2-only client with transport deadlines and redirect refusal.
// Inputs: current mounted config/scope; outputs: storage adapter. Effects: none until an operation is invoked.
// Choose instead of acquisition imports, which would introduce the existing activities dependency cycle.
func NewB2Storage(cfg B2Config, scope Scope) (*B2Storage, error) {
	if libraryvalidation.ValidateB2StorageEndpoint(cfg.Endpoint) != nil || cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("sync B2 config missing or not current Backblaze storage")
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(aws.Config{Region: cfg.Region, Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""), HTTPClient: boundedHTTPClient()}, func(o *s3.Options) { o.BaseEndpoint = aws.String(cfg.Endpoint); o.UsePathStyle = true })
	return &B2Storage{Client: client, Scope: scope}, nil
}

func (b B2Storage) ready() error {
	if b.Client == nil {
		return errors.New("B2 storage not configured")
	}
	return b.Scope.Validate()
}

// List observes one legal child version page including hide markers; inputs: child/cursor; outputs: explicit continuation.
// Effects: one version-list request. Choose over catalog/key-only listing for actual version identity.
func (b B2Storage) List(ctx context.Context, child string, cursor Cursor) (Page, error) {
	if child != "case-law" && child != "benchbooks" && child != "reference-data" {
		return Page{}, errors.New("invalid legal child")
	}
	return b.list(ctx, b.Scope.LegalRoot+child+"/", cursor, false)
}

func (b B2Storage) list(ctx context.Context, prefix string, cursor Cursor, exact bool) (Page, error) {
	if err := b.ready(); err != nil {
		return Page{}, err
	}
	if cursor.Key != "" && (!strings.HasPrefix(cursor.Key, prefix) || !safeKey(cursor.Key) || (!exact && b.Scope.Admit(b.Scope.Bucket, cursor.Key) != nil)) {
		return Page{}, errors.New("invalid version cursor")
	}
	if cursor.VersionID != "" && !validVersion(cursor.VersionID) {
		return Page{}, errors.New("invalid version cursor")
	}
	ctx, cancel := context.WithTimeout(ctx, IOTimeout)
	defer cancel()
	in := &s3.ListObjectVersionsInput{Bucket: aws.String(b.Scope.Bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(MaxPage)}
	if cursor.Key != "" {
		in.KeyMarker = aws.String(cursor.Key)
	}
	if cursor.VersionID != "" {
		in.VersionIdMarker = aws.String(cursor.VersionID)
	}
	out, err := b.Client.ListObjectVersions(ctx, in)
	if err != nil || out == nil {
		return Page{}, errors.New("B2 version listing unavailable")
	}
	if len(out.Versions)+len(out.DeleteMarkers) > MaxPage {
		return Page{}, errors.New("B2 version listing exceeded page budget")
	}
	p := Page{Objects: []Object{}, Complete: !aws.ToBool(out.IsTruncated)}
	for _, v := range out.Versions {
		k := aws.ToString(v.Key)
		if exact && k != prefix {
			continue
		}
		if b.Scope.Admit(b.Scope.Bucket, k) != nil || !strings.HasPrefix(k, prefix) || !validVersion(aws.ToString(v.VersionId)) || aws.ToInt64(v.Size) < 0 || v.LastModified == nil {
			return Page{}, errors.New("B2 version listing identity invalid")
		}
		p.Objects = append(p.Objects, Object{Bucket: b.Scope.Bucket, Key: k, VersionID: aws.ToString(v.VersionId), Size: aws.ToInt64(v.Size), Latest: aws.ToBool(v.IsLatest), UploadedAt: *v.LastModified})
	}
	for _, v := range out.DeleteMarkers {
		k := aws.ToString(v.Key)
		if exact && k != prefix {
			continue
		}
		if b.Scope.Admit(b.Scope.Bucket, k) != nil || !strings.HasPrefix(k, prefix) || !validVersion(aws.ToString(v.VersionId)) || v.LastModified == nil {
			return Page{}, errors.New("B2 hide listing identity invalid")
		}
		p.Objects = append(p.Objects, Object{Bucket: b.Scope.Bucket, Key: k, VersionID: aws.ToString(v.VersionId), Latest: aws.ToBool(v.IsLatest), Hidden: true, UploadedAt: *v.LastModified})
	}
	if !p.Complete {
		p.Next = Cursor{Key: aws.ToString(out.NextKeyMarker), VersionID: aws.ToString(out.NextVersionIdMarker)}
		if p.Next.Key == "" || p.Next == cursor {
			return Page{}, errors.New("B2 listing returned unusable continuation")
		}
	}
	return p, nil
}

// History observes bounded exact-key version history; inputs: admitted key; outputs: complete or explicitly partial coverage.
// Effects: at most five pages. No version is deleted; partial history cannot establish absence of a racing writer.
func (b B2Storage) History(ctx context.Context, key string) (Page, error) {
	if err := b.Scope.Admit(b.Scope.Bucket, key); err != nil {
		return Page{}, err
	}
	result := Page{Objects: []Object{}}
	cursor := Cursor{}
	visited := map[Cursor]bool{}
	for n := 0; n < MaxHistoryPages; n++ {
		if visited[cursor] {
			return result, errors.New("B2 history repeated a visited cursor")
		}
		visited[cursor] = true
		p, err := b.list(ctx, key, cursor, true)
		if err != nil {
			return result, err
		}
		result.Objects = append(result.Objects, p.Objects...)
		result.Next = p.Next
		if p.Complete {
			result.Complete = true
			break
		}
		cursor = p.Next
	}
	sort.Slice(result.Objects, func(i, j int) bool { return result.Objects[i].UploadedAt.After(result.Objects[j].UploadedAt) })
	return result, nil
}

// Head identifies the current exact version without claiming byte integrity; inputs: key; outputs: metadata or absence.
// Effects: one HEAD. Only explicit missing-object responses mean absence; permission/network errors remain failures.
func (b B2Storage) Head(ctx context.Context, key string) (*Object, error) {
	if err := b.ready(); err != nil {
		return nil, err
	}
	if err := b.Scope.Admit(b.Scope.Bucket, key); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, IOTimeout)
	defer cancel()
	out, err := b.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(b.Scope.Bucket), Key: aws.String(key)})
	if err != nil {
		var api interface{ ErrorCode() string }
		if errors.As(err, &api) && (api.ErrorCode() == "NotFound" || api.ErrorCode() == "NoSuchKey") {
			return nil, nil
		}
		return nil, errors.New("B2 current version unavailable")
	}
	if out == nil || !validVersion(aws.ToString(out.VersionId)) || out.ContentLength == nil || *out.ContentLength < 0 {
		return nil, errors.New("B2 HEAD has no valid retained identity")
	}
	v := Object{Bucket: b.Scope.Bucket, Key: key, VersionID: *out.VersionId, Size: *out.ContentLength, ContentType: aws.ToString(out.ContentType), Latest: true, OperationID: out.Metadata["toolkit-operation-id"], IntentID: out.Metadata["toolkit-intent-id"]}
	if out.LastModified != nil {
		v.UploadedAt = *out.LastModified
	}
	return &v, nil
}

func (b B2Storage) open(ctx context.Context, obj Object, max int64) (*s3.GetObjectOutput, error) {
	if err := b.ready(); err != nil {
		return nil, err
	}
	if err := b.Scope.Admit(obj.Bucket, obj.Key); err != nil {
		return nil, err
	}
	if obj.Hidden || !validVersion(obj.VersionID) || obj.Size < 0 || max <= 0 || max > MaxOriginalBytes || obj.Size > max {
		return nil, errors.New("pinned read identity or body budget invalid")
	}
	out, err := b.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(obj.Bucket), Key: aws.String(obj.Key), VersionId: aws.String(obj.VersionID)})
	if err != nil {
		return nil, errors.New("B2 pinned version unavailable")
	}
	if out == nil || out.Body == nil {
		return nil, errors.New("B2 pinned body absent")
	}
	if aws.ToString(out.VersionId) != obj.VersionID || out.ContentLength == nil || *out.ContentLength != obj.Size {
		out.Body.Close()
		return nil, errors.New("B2 pinned descriptor mismatch")
	}
	return out, nil
}

// Read returns complete exact-version bytes under a caller budget; inputs: metadata/max; outputs: bytes plus provider metadata.
// Effects: one bounded GET. Use for original proxy or the existing extractor; does not declare legal verification.
func (b B2Storage) Read(ctx context.Context, obj Object, max int64) ([]byte, Object, error) {
	ctx, cancel := context.WithTimeout(ctx, IOTimeout)
	defer cancel()
	out, err := b.open(ctx, obj, max)
	if err != nil {
		return nil, obj, err
	}
	defer out.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(out.Body, obj.Size+1))
	if err != nil || int64(len(raw)) != obj.Size {
		return nil, obj, errors.New("B2 pinned body truncated or exceeded size")
	}
	obj.ContentType = aws.ToString(out.ContentType)
	obj.OperationID = out.Metadata["toolkit-operation-id"]
	obj.IntentID = out.Metadata["toolkit-intent-id"]
	return raw, obj, nil
}

// Hash computes a full-byte SHA-256 independently of parsing; inputs: exact object; outputs: attested metadata only.
// Effects: streaming exact-version GET capped at 20 MiB. Never substitute SHA-1/ETag or truncate a large document.
func (b B2Storage) Hash(ctx context.Context, obj Object) (Object, error) {
	ctx, cancel := context.WithTimeout(ctx, IOTimeout)
	defer cancel()
	out, err := b.open(ctx, obj, MaxOriginalBytes)
	if err != nil {
		return obj, err
	}
	defer out.Body.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(out.Body, obj.Size+1))
	if err != nil || n != obj.Size {
		return obj, errors.New("B2 hashing failed complete-byte budget")
	}
	obj.SHA256 = hex.EncodeToString(h.Sum(nil))
	obj.ContentType = aws.ToString(out.ContentType)
	obj.OperationID = out.Metadata["toolkit-operation-id"]
	obj.IntentID = out.Metadata["toolkit-intent-id"]
	return obj, nil
}

// Put appends one explicitly authorized retained version with operation metadata and no conditional-CAS claim.
// Inputs: valid immutable operation, complete verified payload and backend intent; outputs: provider VersionId only.
// Effects: one SDK PUT with retries disabled. Readback/reconciliation are separately tracked; missing replies remain write_unknown.
func (b B2Storage) Put(ctx context.Context, op Operation, raw []byte, intent string) (Object, error) {
	if err := b.ready(); err != nil {
		return Object{}, err
	}
	if err := op.validate(b.Scope); err != nil {
		return Object{}, err
	}
	if intent == "" || len(intent) > 200 || strings.ContainsAny(intent, "\r\n\x00") {
		return Object{}, errors.New("outbox write intent invalid")
	}
	if err := ValidatePayload(op, raw); err != nil {
		return Object{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, IOTimeout)
	defer cancel()
	out, err := b.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(op.Bucket), Key: aws.String(op.Key), Body: bytes.NewReader(raw), ContentLength: aws.Int64(op.PayloadSize), ContentType: aws.String(op.ContentType), Metadata: map[string]string{"toolkit-operation-id": op.OperationID, "toolkit-payload-sha256": op.PayloadSHA256, "toolkit-intent-id": intent}}, func(o *s3.Options) { o.RetryMaxAttempts = 1 })
	if err != nil || out == nil || !validVersion(aws.ToString(out.VersionId)) {
		return Object{}, errors.New("B2 PUT outcome unknown; reconcile before retry")
	}
	return Object{Bucket: op.Bucket, Key: op.Key, VersionID: *out.VersionId, Size: op.PayloadSize, ContentType: op.ContentType, OperationID: op.OperationID, IntentID: intent}, nil
}
