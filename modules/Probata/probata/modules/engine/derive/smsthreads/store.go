// Byline: Claude Code · Fable 5.1 · 2026-09-20

package smsthreads

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectStore is the whole storage surface the derive unit needs: stream one
// source object in, publish new objects beside it, and refuse to overwrite.
type ObjectStore interface {
	Open(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Put(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64, contentType string) error
	Exists(ctx context.Context, bucket, key string) (bool, error)
}

// S3Store serves any S3-compatible provider (B2, R2) through one client.
type S3Store struct{ Client *s3.Client }

// Open streams the object; a stream that breaks mid-object resumes with a
// ranged GET pinned to the object's ETag (resume.go).
// Byline: Claude Code · Opus 5.5 · 2026-10-02
func (s S3Store) Open(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	return openResuming(ctx, func(ctx context.Context, from int64, etag string) (io.ReadCloser, string, int64, error) {
		input := &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}
		if from > 0 {
			input.Range = aws.String(fmt.Sprintf("bytes=%d-", from))
		}
		if etag != "" {
			input.IfMatch = aws.String(etag)
		}
		out, err := s.Client.GetObject(ctx, input)
		if err != nil {
			return nil, "", -1, err
		}
		total := int64(-1)
		if out.ContentLength != nil {
			total = from + *out.ContentLength
		}
		return out.Body, aws.ToString(out.ETag), total, nil
	})
}

func (s S3Store) Put(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64, contentType string) error {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: body,
		ContentLength: aws.Int64(size), ContentType: aws.String(contentType),
	})
	return err
}

func (s S3Store) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err == nil {
		return true, nil
	}
	var notFound *s3types.NotFound
	var noKey *s3types.NoSuchKey
	if errors.As(err, &notFound) || errors.As(err, &noKey) {
		return false, nil
	}
	return false, err
}
