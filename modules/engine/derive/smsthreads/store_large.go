// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Two storage operations the repair tools need beyond ObjectStore: a HEAD
// that reports size and ETag (repair.find_other_version confirms a catalog
// candidate still exists; repair.salvage_truncated_xml detects a source that
// changed under a published salvage), and an upload that does not stop at
// the 5 GiB single-PUT ceiling S3-compatible stores impose (a salvaged
// multi-gigabyte backup).

package smsthreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectInfo is what a HEAD request reports about one object.
type ObjectInfo struct {
	Size         int64
	ETag         string
	LastModified time.Time
}

// ObjectStatter reports whether an object exists and, if so, what it is.
type ObjectStatter interface {
	Stat(ctx context.Context, bucket, key string) (ObjectInfo, bool, error)
}

// LargeObjectPutter uploads an object of any size from a random-access body.
type LargeObjectPutter interface {
	PutLarge(ctx context.Context, bucket, key string, body io.ReaderAt, size int64, contentType string) error
}

const (
	// singlePutLimit keeps a single PUT well under the 5 GiB ceiling.
	singlePutLimit = 4 << 30
	// multipartPartSize is the default part size; it grows only when an
	// object would otherwise need more than maxMultipartParts parts.
	multipartPartSize = 64 << 20
	maxMultipartParts = 10_000
)

// Stat issues one HEAD request.
func (s S3Store) Stat(ctx context.Context, bucket, key string) (ObjectInfo, bool, error) {
	out, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		var notFound *s3types.NotFound
		var noKey *s3types.NoSuchKey
		if errors.As(err, &notFound) || errors.As(err, &noKey) {
			return ObjectInfo{}, false, nil
		}
		return ObjectInfo{}, false, err
	}
	info := ObjectInfo{ETag: strings.Trim(aws.ToString(out.ETag), `"`)}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.LastModified != nil {
		info.LastModified = out.LastModified.UTC()
	}
	return info, true, nil
}

// PutLarge uploads body in one PUT when it fits and in parts otherwise.
func (s S3Store) PutLarge(ctx context.Context, bucket, key string, body io.ReaderAt, size int64, contentType string) error {
	if s.Client == nil {
		return errors.New("smsthreads: S3 store has no client")
	}
	if size <= singlePutLimit {
		return s.Put(ctx, bucket, key, io.NewSectionReader(body, 0, size), size, contentType)
	}
	return putMultipart(ctx, s.Client, bucket, key, body, size, multipartPartSize, contentType)
}

// multipartAPI is the slice of the S3 client a multipart upload uses; the
// real *s3.Client satisfies it and tests substitute a recorder.
type multipartAPI interface {
	CreateMultipartUpload(context.Context, *s3.CreateMultipartUploadInput, ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(context.Context, *s3.UploadPartInput, ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CompleteMultipartUpload(context.Context, *s3.CompleteMultipartUploadInput, ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(context.Context, *s3.AbortMultipartUploadInput, ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}

// putMultipart uploads body in sequential parts. Memory stays at one part's
// section reader; nothing is buffered. A failed upload is aborted so no
// orphaned parts are billed.
func putMultipart(ctx context.Context, api multipartAPI, bucket, key string, body io.ReaderAt, size, partSize int64, contentType string) error {
	if size <= 0 {
		return errors.New("smsthreads: multipart upload needs a positive size")
	}
	if partSize <= 0 {
		partSize = multipartPartSize
	}
	if parts := (size + partSize - 1) / partSize; parts > maxMultipartParts {
		partSize = (size + maxMultipartParts - 1) / maxMultipartParts
	}
	created, err := api.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket), Key: aws.String(key), ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("smsthreads: start multipart upload of %s: %w", key, err)
	}
	uploadID := created.UploadId
	abort := func(cause error) error {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if _, abortErr := api.AbortMultipartUpload(cleanup, &s3.AbortMultipartUploadInput{
			Bucket: aws.String(bucket), Key: aws.String(key), UploadId: uploadID,
		}); abortErr != nil {
			return errors.Join(cause, fmt.Errorf("smsthreads: abort multipart upload of %s: %w", key, abortErr))
		}
		return cause
	}
	var completed []s3types.CompletedPart
	for offset, number := int64(0), int32(1); offset < size; offset, number = offset+partSize, number+1 {
		if err := ctx.Err(); err != nil {
			return abort(err)
		}
		length := partSize
		if offset+length > size {
			length = size - offset
		}
		part, err := api.UploadPart(ctx, &s3.UploadPartInput{
			Bucket: aws.String(bucket), Key: aws.String(key), UploadId: uploadID,
			PartNumber: aws.Int32(number), Body: io.NewSectionReader(body, offset, length),
			ContentLength: aws.Int64(length),
		})
		if err != nil {
			return abort(fmt.Errorf("smsthreads: upload part %d of %s: %w", number, key, err))
		}
		completed = append(completed, s3types.CompletedPart{ETag: part.ETag, PartNumber: aws.Int32(number)})
	}
	if _, err := api.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: aws.String(bucket), Key: aws.String(key), UploadId: uploadID,
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: completed},
	}); err != nil {
		return abort(fmt.Errorf("smsthreads: complete multipart upload of %s: %w", key, err))
	}
	return nil
}
