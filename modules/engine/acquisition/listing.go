// Byline: Claude Code · Opus 5 · 2026-09-21
//
// Paged object listing for batch-by-folder intake (owner 2026-09-20 23:52:
// "it gets batched by folder. Being careful not to overload any systems and
// process them one at a time").
//
// One page per call, never a whole listing in memory: a vault folder can hold
// tens of thousands of objects, and the caller is a Temporal Activity whose
// result crosses workflow history.

package acquisition

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectListingPage is one page of object keys under a prefix.
type ObjectListingPage struct {
	// Keys are full object keys, already filtered of directory placeholders.
	Keys []string
	// NextCursor is empty when the listing is complete.
	NextCursor string
}

// ObjectLister is the narrow seam a batch uses; S3Lister satisfies it and a
// test supplies its own.
type ObjectLister interface {
	ListObjectsPage(ctx context.Context, bucket, prefix, cursor string, limit int32) (ObjectListingPage, error)
}

// S3Lister lists any S3-compatible store through one client.
type S3Lister struct{ Client *s3.Client }

// ListObjectsPage returns at most limit keys under prefix, recursively (no
// delimiter): a folder's members include everything beneath it, which is what
// "batch by folder" means for a vault tree. A key that ends in "/" is a
// directory placeholder and is dropped.
func (l S3Lister) ListObjectsPage(ctx context.Context, bucket, prefix, cursor string, limit int32) (ObjectListingPage, error) {
	if l.Client == nil {
		return ObjectListingPage{}, errors.New("acquisition: object lister requires an S3 client")
	}
	if strings.TrimSpace(bucket) == "" {
		return ObjectListingPage{}, errors.New("acquisition: object listing requires a bucket")
	}
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(limit),
	}
	if strings.TrimSpace(cursor) != "" {
		input.ContinuationToken = aws.String(cursor)
	}
	out, err := l.Client.ListObjectsV2(ctx, input)
	if err != nil {
		return ObjectListingPage{}, fmt.Errorf("acquisition: list %s/%s: %w", bucket, prefix, err)
	}
	page := ObjectListingPage{Keys: make([]string, 0, len(out.Contents))}
	for _, object := range out.Contents {
		key := aws.ToString(object.Key)
		if key == "" || strings.HasSuffix(key, "/") {
			continue
		}
		page.Keys = append(page.Keys, key)
	}
	if aws.ToBool(out.IsTruncated) {
		page.NextCursor = aws.ToString(out.NextContinuationToken)
	}
	return page, nil
}
