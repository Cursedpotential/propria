// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"context"
	"crypto/sha1" //nolint:gosec // matches the catalog's sha1 column; a content identity check, not a security control
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ContactsFetcher reads one object from an S3-compatible store (Backblaze B2) and hashes it while copying.
// It only reads; nothing is written to the bucket.
type S3ContactsFetcher struct{ Client *s3.Client }

// Fetch implements ContactsFetcher.
func (f S3ContactsFetcher) Fetch(ctx context.Context, bucket, key string, w io.Writer) (int64, string, string, error) {
	if f.Client == nil {
		return 0, "", "", errors.New("contacts fetch: no object store client")
	}
	out, err := f.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return 0, "", "", errors.New("contacts fetch: object unavailable")
	}
	defer out.Body.Close()
	one, two := sha1.New(), sha256.New() //nolint:gosec
	n, err := io.Copy(io.MultiWriter(w, one, two), out.Body)
	if err != nil {
		return n, "", "", errors.New("contacts fetch: read interrupted")
	}
	return n, hex.EncodeToString(one.Sum(nil)), hex.EncodeToString(two.Sum(nil)), nil
}
