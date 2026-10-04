// Byline: Codex · GPT-6 · 2026-10-04.
package smsthreads

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const maxConditionalPutBytes = int64(5_000_000_000)

// ErrConditionalObjectMismatch means a create-only conflict found different remote bytes.
// Inputs: none; outputs: a stable sentinel suitable for errors.Is.
// Side effects: none. Choose this over retrying a key whose pre-existing bytes differ.
// Byline: Codex · GPT-6 · 2026-10-04.
var ErrConditionalObjectMismatch = errors.New("conditional object already exists with different bytes")

// PutIfAbsent conditionally creates one bounded S3 object and verifies matching bytes after a create conflict.
// Inputs: bucket/key, seekable object bytes, exact size, and content type; outputs: nil after creation or verified identical replay.
// Side effects: issues PutObject with If-None-Match: *; on known precondition conflicts, reads back and hashes the existing object.
// Choose instead of ObjectStore.Put when overwriting any pre-existing key is forbidden.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) PutIfAbsent(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64, contentType string) error {
	if s.Client == nil {
		return errors.New("conditional S3 create requires a client")
	}
	if bucket == "" || key == "" || body == nil || size <= 0 || size > maxConditionalPutBytes {
		return errors.New("conditional S3 create requires bucket, key, body, and size within the 5 GB single-PUT limit")
	}
	expected, err := hashCreateOnlyBody(ctx, body, size)
	if err != nil {
		return err
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind conditional S3 body: %w", err)
	}
	_, err = s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: body,
		ContentLength: aws.Int64(size), ContentType: aws.String(contentType), IfNoneMatch: aws.String("*"),
	})
	if err == nil {
		return nil
	}
	if !isConditionalCreateConflict(err) {
		return fmt.Errorf("conditional S3 create failed: %w", err)
	}
	if err = s.verifyExistingCreateOnly(ctx, bucket, key, expected, size); err != nil {
		if errors.Is(err, ErrConditionalObjectMismatch) {
			return err
		}
		return fmt.Errorf("conditional S3 conflict readback failed: %w", err)
	}
	return nil
}

// hashCreateOnlyBody validates and hashes an exactly sized upload without retaining its content.
// Inputs: context, seekable body and declared size; outputs: SHA-256 after exactly size bytes are read.
// Side effects: reads and rewinds the local source with cancellation checks. Choose before a conditional upload to authenticate replay bytes.
// Byline: Codex · GPT-6 · 2026-10-04.
func hashCreateOnlyBody(ctx context.Context, body io.ReadSeeker, size int64) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return zero, fmt.Errorf("seek conditional S3 body: %w", err)
	}
	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	var count int64
	for {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		want := int64(len(buffer))
		if size-count+1 < want {
			want = size - count + 1
		}
		n, readErr := body.Read(buffer[:int(want)])
		count += int64(n)
		if count > size {
			return zero, fmt.Errorf("conditional S3 body exceeds declared size %d", size)
		}
		if n > 0 {
			_, _ = hash.Write(buffer[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return zero, fmt.Errorf("hash conditional S3 body: %w", readErr)
		}
		if n == 0 {
			return zero, io.ErrNoProgress
		}
	}
	if count != size {
		return zero, fmt.Errorf("conditional S3 body size mismatch: read %d bytes, declared %d", count, size)
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return zero, fmt.Errorf("rewind conditional S3 body after hash: %w", err)
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest, nil
}

// isConditionalCreateConflict recognizes only S3's bounded conditional-write conflict codes.
// Inputs: one PutObject error; outputs: true for PreconditionFailed or ConditionalRequestConflict only.
// Side effects: none. Choose narrow error-code matching so auth, transport, and service failures are not replayed as conflicts.
// Byline: Codex · GPT-6 · 2026-10-04.
func isConditionalCreateConflict(err error) bool {
	var apiError smithy.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	switch apiError.ErrorCode() {
	case "PreconditionFailed", "ConditionalRequestConflict":
		return true
	default:
		return false
	}
}

// verifyExistingCreateOnly streams the existing object and accepts a conflict only for exact byte identity.
// Inputs: bucket/key, expected digest and size; outputs: nil for identical content or a stable mismatch/error.
// Side effects: one bounded GET stream. Choose after a recognized precondition response to make retries idempotent.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) verifyExistingCreateOnly(ctx context.Context, bucket, key string, expected [sha256.Size]byte, expectedSize int64) error {
	object, err := s.Open(ctx, bucket, key)
	if err != nil {
		return err
	}
	defer object.Close()
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(object, expectedSize+1))
	if err != nil {
		return fmt.Errorf("read existing conditional S3 object: %w", err)
	}
	if count != expectedSize {
		return ErrConditionalObjectMismatch
	}
	actual := hash.Sum(nil)
	if !equalSHA256(actual, expected[:]) {
		return ErrConditionalObjectMismatch
	}
	return nil
}

// equalSHA256 compares fixed-size digest bytes without converting them to display strings.
// Inputs: actual and expected digest slices; outputs: true only for equal 32-byte digests.
// Side effects: none. Choose for readback comparisons inside the create-only adapter.
// Byline: Codex · GPT-6 · 2026-10-04.
func equalSHA256(actual, expected []byte) bool {
	if len(actual) != sha256.Size || len(expected) != sha256.Size {
		return false
	}
	var difference byte
	for i := range actual {
		difference |= actual[i] ^ expected[i]
	}
	return difference == 0
}
