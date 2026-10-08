// Byline: Codex · GPT-6.1 · 2026-10-07.
package activities

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strings"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// AISourceUnitMetadata pins the exact provider metadata observed before copying.
// Inputs: exact-version HEAD
// Outputs: bounded version, size and preserved native metadata
// Effects: none as data
// Choose: over interpreting an ETag as a content hash.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitMetadata struct {
	VersionID          string            `json:"version_id"`
	Bytes              int64             `json:"bytes"`
	ETag               string            `json:"etag"`
	ContentType        string            `json:"content_type,omitempty"`
	ContentDisposition string            `json:"content_disposition,omitempty"`
	ContentEncoding    string            `json:"content_encoding,omitempty"`
	ContentLanguage    string            `json:"content_language,omitempty"`
	CacheControl       string            `json:"cache_control,omitempty"`
	Metadata           map[string]string `json:"metadata"`
}

// aiSourceUnitStore adds exact-version HEAD and server-side copy to the existing resolver seam.
// Inputs: reviewed B2 coordinates
// Outputs: metadata, retained streams and version IDs
// Effects: explicit method-specific B2 reads/copies
// Choose: without changing legal/Markdown placement contracts.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type aiSourceUnitStore interface {
	toolkitPlacementStore
	HeadExact(context.Context, string, string, string) (AISourceUnitMetadata, error)
	CopyExact(context.Context, string, string, string, string, string) (string, error)
	SourceKeys(context.Context, string, string) ([]string, error)
	UnitKeys(context.Context, string, string) ([]string, error)
}

// aiSourceUnitS3 reuses the worker's mounted object-store configuration and S3 client.
// Inputs: existing S3Store
// Outputs: narrowly scoped retained-version capabilities
// Effects: none at construction
// Choose: over credentials or a second transport runtime.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type aiSourceUnitS3 struct{ toolkitPlacementS3 }

// HeadExact admits one pinned version's provider identity without reading its payload.
// Inputs: bucket/key/version
// Outputs: exact returned metadata or error
// Effects: one versioned HEAD
// Choose: before separately scheduled full-byte hashing.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s aiSourceUnitS3) HeadExact(ctx context.Context, bucket, key, version string) (AISourceUnitMetadata, error) {
	if s.Client == nil || bucket != "salem-data" || key == "" || !validToolkitVersionID(version) {
		return AISourceUnitMetadata{}, errors.New("invalid exact source metadata coordinates")
	}
	out, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), VersionId: aws.String(version)})
	if err != nil {
		return AISourceUnitMetadata{}, err
	}
	m := AISourceUnitMetadata{VersionID: aws.ToString(out.VersionId), Bytes: aws.ToInt64(out.ContentLength), ETag: aws.ToString(out.ETag), ContentType: aws.ToString(out.ContentType), ContentDisposition: aws.ToString(out.ContentDisposition), ContentEncoding: aws.ToString(out.ContentEncoding), ContentLanguage: aws.ToString(out.ContentLanguage), CacheControl: aws.ToString(out.CacheControl), Metadata: out.Metadata}
	if m.VersionID != version || m.Bytes < 0 || m.ETag == "" {
		return m, errors.New("exact HEAD identity or metadata unavailable")
	}
	if m.Metadata == nil {
		m.Metadata = map[string]string{}
	}
	return m, nil
}

// aiSourceCopyLocator encodes an exact source version without interpreting literal key characters.
// Inputs: bucket, literal key and provider version
// Outputs: signed CopySource header value
// Effects: none
// Choose: over concatenating an unescaped question mark or version query.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceCopyLocator(bucket, key, version string) string {
	u := url.URL{Path: bucket + "/" + key, RawQuery: url.Values{"versionId": []string{version}}.Encode()}
	return u.EscapedPath() + "?" + u.RawQuery
}

// CopyExact performs one same-bucket server-side copy with a pinned source version and ETag.
// Inputs: source/destination keys, exact source version and admitted ETag
// Outputs: actual response destination version
// Effects: one retained CopyObject, no GET, PUT, move or delete
// Choose: after source hashing and exclusive destination preflight, then check races independently.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s aiSourceUnitS3) CopyExact(ctx context.Context, bucket, source, version, destination, etag string) (string, error) {
	if s.Client == nil || bucket != "salem-data" || !strings.HasPrefix(source, "consignatio/vault/v1/") || !strings.HasPrefix(destination, aiSourceUnitPrefix) || source == destination || !validToolkitVersionID(version) || etag == "" {
		return "", errors.New("invalid pinned same-bucket copy")
	}
	out, err := s.Client.CopyObject(ctx, &s3.CopyObjectInput{Bucket: aws.String(bucket), Key: aws.String(destination), CopySource: aws.String(aiSourceCopyLocator(bucket, source, version)), CopySourceIfMatch: aws.String(etag), MetadataDirective: s3types.MetadataDirectiveCopy}, func(o *s3.Options) { o.RetryMaxAttempts = 1 })
	if err != nil {
		return "", err
	}
	id := aws.ToString(out.VersionId)
	if !validToolkitVersionID(id) || aws.ToString(out.CopySourceVersionId) != version {
		return id, errors.New("copy response does not bind both retained version identities; review uncertain response")
	}
	return id, nil
}

// UnitKeys observes a bounded current destination unit without silently truncating membership.
// Inputs: bucket and unit prefix
// Outputs: exact visible keys
// Effects: one metadata listing
// Choose: to reject merging with unreviewed members and to prove all requested placements at readback.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s aiSourceUnitS3) UnitKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if !strings.HasPrefix(prefix, aiSourceUnitPrefix) || !strings.HasSuffix(prefix, "/") {
		return nil, errors.New("invalid AI destination unit listing")
	}
	return s.aiSourceListKeys(ctx, bucket, prefix)
}

// SourceKeys observes the complete current source prefix before admitting its frozen membership.
// Inputs: bucket and reviewed source prefix
// Outputs: all visible source keys within the bounded listing
// Effects: one metadata listing
// Choose: before copying a unit whose earlier discovery could have missed later-added members.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s aiSourceUnitS3) SourceKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if !strings.HasPrefix(prefix, "consignatio/vault/v1/") || !strings.HasSuffix(prefix, "/") || !toolkitPlacementPath(strings.TrimSuffix(prefix, "/")) {
		return nil, errors.New("invalid source unit listing")
	}
	return s.aiSourceListKeys(ctx, bucket, prefix)
}

// aiSourceListKeys refuses truncated current-key listings for either source or destination units.
// Inputs: B2 bucket and already validated unit prefix
// Outputs: complete visible key set or error
// Effects: one ListObjectsV2 metadata request
// Choose: for bounded membership proof instead of treating the first page as complete.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s aiSourceUnitS3) aiSourceListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if s.Client == nil || bucket != "salem-data" {
		return nil, errors.New("invalid B2 unit listing")
	}
	out, err := s.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(128)})
	if err != nil {
		return nil, err
	}
	if aws.ToBool(out.IsTruncated) {
		return nil, errors.New("unit listing truncated; incomplete coverage refused")
	}
	keys := make([]string, 0, len(out.Contents))
	for _, v := range out.Contents {
		keys = append(keys, aws.ToString(v.Key))
	}
	return keys, nil
}

// aiSourceMetadataPreserved compares native metadata independently of new provider version timestamps and ETags.
// Inputs: source and destination HEAD snapshots
// Outputs: true for equal native metadata and size
// Effects: none
// Choose: beside full-byte SHA comparison, never instead of it.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceMetadataPreserved(a, b AISourceUnitMetadata) bool {
	a.VersionID, b.VersionID = "", ""
	a.ETag, b.ETag = "", ""
	return reflect.DeepEqual(a, b)
}

// sourceUnitStore resolves only the existing B2 adapter and requires retained-version capabilities.
// Inputs: configured resolver
// Outputs: narrow source-unit store
// Effects: resolver construction only
// Choose: without registering new authentication or storage systems.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) sourceUnitStore() (aiSourceUnitStore, error) {
	if a.Stores == nil {
		return nil, errors.New("existing B2 resolver required")
	}
	base, err := a.Stores("b2")
	if err != nil {
		return nil, err
	}
	switch s := base.(type) {
	case smsthreads.S3Store:
		return aiSourceUnitS3{toolkitPlacementS3{s}}, nil
	case *smsthreads.S3Store:
		if s != nil {
			return aiSourceUnitS3{toolkitPlacementS3{*s}}, nil
		}
	case aiSourceUnitStore:
		return s, nil
	}
	return nil, errors.New("B2 resolver lacks exact-version server-copy capabilities")
}
