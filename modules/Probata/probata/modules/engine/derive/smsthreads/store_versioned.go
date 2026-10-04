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
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrRecoveredVersionMismatch means a recovery key or exact version contains different bytes.
// Inputs: none; outputs: a stable sentinel for errors.Is checks. Side effects: none.
// Choose this over creating another version when a key already identifies different recovery bytes.
// Byline: Codex · GPT-6 · 2026-10-04.
var ErrRecoveredVersionMismatch = errors.New("recovered object version has different bytes")

// ErrRecoveredVersionNotFound means the requested key or exact version is absent.
// Inputs: none; outputs: a stable sentinel for errors.Is checks. Side effects: none.
// Choose this to distinguish a missing object from provider, permission, or transport failures.
// Byline: Codex · GPT-6 · 2026-10-04.
var ErrRecoveredVersionNotFound = errors.New("recovered object version not found")

// ObjectVersion identifies one provider-retained version and its reported size.
// Inputs: one successful HEAD response; outputs: the returned VersionId and ContentLength.
// Side effects: none. Choose this value to pin subsequent reads to the observed version.
// Byline: Codex · GPT-6 · 2026-10-04.
type ObjectVersion struct {
	VersionID string
	Size      int64
}

// PutRecoveredVersion reuses identical existing bytes or writes and verifies one provider-retained version.
// Inputs: bucket/key, seekable bytes, exact size, content type and SHA-256; outputs: a nonempty provider VersionId.
// Side effects: HEAD, at most one ordinary PutObject in this explicitly versioned method, and exact-version GET readback.
// A pre-existing mismatch is refused; races may create multiple retained versions, so callers must use a fresh operation namespace.
// Choose only for explicitly authorized versioned recovery when the provider rejects conditional create.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) PutRecoveredVersion(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64, contentType, expectedSHA256 string, progress func(int64)) (string, error) {
	if s.Client == nil || bucket == "" || key == "" || body == nil || size < 0 || size > maxConditionalPutBytes || !validSHA256Hex(expectedSHA256) {
		return "", errors.New("versioned recovery requires a client, coordinates, bounded body, and lowercase SHA-256")
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek versioned recovery body: %w", err)
	}
	digest, err := hashCreateOnlyBody(ctx, body, size)
	if err != nil {
		return "", fmt.Errorf("hash versioned recovery body: %w", err)
	}
	if got := fmt.Sprintf("%x", digest); got != expectedSHA256 {
		return "", errors.New("versioned recovery source digest differs from its expected SHA-256")
	}

	current, err := s.HeadVersion(ctx, bucket, key)
	if err == nil {
		if err = s.verifyRecoveredVersion(ctx, bucket, key, current, expectedSHA256, size, progress); err != nil {
			return "", err
		}
		return current.VersionID, nil
	}
	if !errors.Is(err, ErrRecoveredVersionNotFound) {
		return "", fmt.Errorf("preflight versioned recovery key: %w", err)
	}

	if _, err = body.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind versioned recovery body: %w", err)
	}
	out, putErr := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: body,
		ContentLength: aws.Int64(size), ContentType: aws.String(contentType),
	})
	if putErr != nil {
		// A retry may follow a lost response. Reuse only a freshly observed version whose full bytes match.
		latest, headErr := s.HeadVersion(ctx, bucket, key)
		if headErr == nil && s.verifyRecoveredVersion(ctx, bucket, key, latest, expectedSHA256, size, progress) == nil {
			return latest.VersionID, nil
		}
		return "", fmt.Errorf("versioned recovery PUT failed; exact matching version was not confirmed: %w", putErr)
	}
	versionID := aws.ToString(out.VersionId)
	if !validObjectVersionID(versionID) {
		return "", errors.New("provider accepted versioned recovery PUT without a valid retained VersionId")
	}
	version := ObjectVersion{VersionID: versionID, Size: size}
	if err = s.verifyRecoveredVersion(ctx, bucket, key, version, expectedSHA256, size, progress); err != nil {
		return "", fmt.Errorf("new provider version failed exact readback: %w", err)
	}
	return versionID, nil
}

// OpenVersion opens one object by its exact provider VersionId.
// Inputs: bucket/key and nonempty VersionId; outputs: a streaming reader pinned to that version.
// Side effects: one remote GET. Choose over S3Store.Open when retryable recovery receipts identify a specific version.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) OpenVersion(ctx context.Context, bucket, key, versionID string) (io.ReadCloser, error) {
	if s.Client == nil || bucket == "" || key == "" || !validObjectVersionID(versionID) {
		return nil, errors.New("exact-version GET requires a client, coordinates, and VersionId")
	}
	out, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), VersionId: aws.String(versionID),
	})
	if err != nil {
		if isVersionNotFound(err) {
			return nil, ErrRecoveredVersionNotFound
		}
		return nil, err
	}
	if !validObjectVersionID(aws.ToString(out.VersionId)) || aws.ToString(out.VersionId) != versionID {
		_ = out.Body.Close()
		return nil, errors.New("provider GET response VersionId differs from requested version")
	}
	return out.Body, nil
}

// HeadVersion identifies the current object version without reading its payload.
// Inputs: bucket/key; outputs: exact nonempty VersionId and reported size, or ErrRecoveredVersionNotFound.
// Side effects: one remote HEAD. Choose before an idempotent retry, then hash through OpenVersion before reuse.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) HeadVersion(ctx context.Context, bucket, key string) (ObjectVersion, error) {
	if s.Client == nil || bucket == "" || key == "" {
		return ObjectVersion{}, errors.New("versioned HEAD requires a client and object coordinates")
	}
	out, err := s.Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		if isVersionNotFound(err) {
			return ObjectVersion{}, ErrRecoveredVersionNotFound
		}
		return ObjectVersion{}, err
	}
	version := ObjectVersion{VersionID: aws.ToString(out.VersionId), Size: aws.ToInt64(out.ContentLength)}
	if !validObjectVersionID(version.VersionID) || version.Size < 0 {
		return ObjectVersion{}, errors.New("provider HEAD returned no valid VersionId or object size")
	}
	return version, nil
}

// verifyRecoveredVersion hashes one exact provider version and compares its full size and SHA-256.
// Inputs: fixed coordinates, ObjectVersion, expected digest and size; outputs: nil only for complete byte identity.
// Side effects: one bounded streaming GET. Choose before accepting pre-existing or newly uploaded versions.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s S3Store) verifyRecoveredVersion(ctx context.Context, bucket, key string, version ObjectVersion, expectedSHA256 string, expectedSize int64, progress func(int64)) error {
	if !validObjectVersionID(version.VersionID) || version.Size != expectedSize {
		return ErrRecoveredVersionMismatch
	}
	stream, err := s.OpenVersion(ctx, bucket, key, version.VersionID)
	if err != nil {
		return err
	}
	defer stream.Close()
	actual, count, err := hashCreateOnlyReader(ctx, stream, expectedSize, progress)
	if err != nil {
		return err
	}
	if count != expectedSize || fmt.Sprintf("%x", actual) != expectedSHA256 {
		return ErrRecoveredVersionMismatch
	}
	return nil
}

// validObjectVersionID accepts bounded, provider-issued version identifiers and rejects suspended-bucket "null" identities.
// Inputs: SDK VersionId text; outputs: true only for a nonempty non-null identifier of at most 2048 bytes.
// Side effects: none. Choose at every write, head and exact-read boundary before treating an object as version-pinned.
// Byline: Codex · GPT-6 · 2026-10-04.
func validObjectVersionID(value string) bool {
	return value != "" && value != "null" && len(value) <= 2048
}

// validSHA256Hex accepts exactly one canonical lowercase SHA-256 digest.
// Inputs: caller digest; outputs: true only for 64 lowercase hexadecimal characters. Side effects: none.
// Choose before any versioned write to bind retries to an explicit content identity.
// Byline: Codex · GPT-6 · 2026-10-04.
func validSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// hashCreateOnlyReader hashes at most the expected object size plus one byte and observes cancellation.
// Inputs: context, stream and expected nonnegative size; outputs: digest and observed size or a read error.
// Side effects: reads only the supplied stream. Choose for exact-version readback so oversized objects are rejected promptly.
// Byline: Codex · GPT-6 · 2026-10-04.
func hashCreateOnlyReader(ctx context.Context, reader io.Reader, expectedSize int64, progress func(int64)) ([32]byte, int64, error) {
	var zero [32]byte
	if expectedSize < 0 {
		return zero, 0, errors.New("expected version size cannot be negative")
	}
	if err := ctx.Err(); err != nil {
		return zero, 0, err
	}
	hash := sha256.New()
	var count int64
	buffer := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return zero, count, err
		}
		readSize := int64(len(buffer))
		if expectedSize-count+1 < readSize {
			readSize = expectedSize - count + 1
		}
		n, readErr := versionContextReader{ctx: ctx, reader: reader}.Read(buffer[:int(readSize)])
		if n > 0 {
			count += int64(n)
			_, _ = hash.Write(buffer[:n])
			if progress != nil {
				progress(count)
			}
			if count > expectedSize {
				return zero, count, ErrRecoveredVersionMismatch
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return zero, count, readErr
		}
		if n == 0 {
			return zero, count, io.ErrNoProgress
		}
	}
	if count != expectedSize {
		return zero, count, ErrRecoveredVersionMismatch
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return digest, count, nil
}

// versionContextReader checks cancellation between each read from a remote object stream.
// Inputs: Activity/request context and stream; outputs: stream bytes or the context error.
// Side effects: advances only the supplied reader. Choose for potentially long exact-version verification streams.
// Byline: Codex · GPT-6 · 2026-10-04.
type versionContextReader struct {
	ctx    context.Context
	reader io.Reader
}

// Read forwards one exact-version stream read after checking cancellation.
// Inputs: a caller buffer; outputs: bytes or a context/read error. Side effects: advances the wrapped stream.
// Choose for hashCreateOnlyReader so cancellation remains responsive on large recovery objects.
// Byline: Codex · GPT-6 · 2026-10-04.
func (r versionContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// isVersionNotFound recognizes only S3 missing-object/version response types.
// Inputs: one SDK error; outputs: true for modeled not-found errors. Side effects: none.
// Choose narrow modeled-error checks so permission and service failures never look like an empty key.
// Byline: Codex · GPT-6 · 2026-10-04.
func isVersionNotFound(err error) bool {
	var notFound *s3types.NotFound
	var noKey *s3types.NoSuchKey
	return errors.As(err, &notFound) || errors.As(err, &noKey)
}
