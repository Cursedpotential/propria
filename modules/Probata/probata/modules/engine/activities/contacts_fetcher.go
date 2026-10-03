// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"archive/zip"
	"context"
	"crypto/sha1" //nolint:gosec // matches the catalog's sha1 column; a content identity check, not a security control
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/Cursedpotential/probata/engine/contacts"
)

// S3ContactsFetcher reads objects from the S3-compatible stores (Backblaze B2 and Cloudflare R2) the catalog
// lists. It only reads; nothing is written to a bucket. Clients is keyed by provider ("b2", "r2").
//
// Every failure is reported as a short, secret-free reason such as "b2/salem-data: NoSuchKey", so a receipt
// can say why an object could not be read (a catalog row whose object is gone is NoSuchKey, a key the login
// may not read is AccessDenied) instead of a generic failure.
type S3ContactsFetcher struct{ Clients map[string]*s3.Client }

const (
	// zipBlockSize is the read-ahead block of the ranged ZIP reader; the central directory of a large archive
	// is read in a few such blocks instead of thousands of tiny requests.
	zipBlockSize = 4 << 20
	// zipBlockCache is how many blocks one ZIP reader keeps.
	zipBlockCache = 6
)

// client returns the S3 client for src's provider.
func (f S3ContactsFetcher) client(src contacts.Source) (*s3.Client, error) {
	client := f.Clients[strings.ToLower(src.Provider)]
	if client == nil {
		return nil, fmt.Errorf("%s/%s: no %s object store is configured on this worker", src.Provider, src.Bucket, src.Provider)
	}
	return client, nil
}

// reason turns an S3 error into a short, secret-free sentence: the API's error code when it has one.
func reason(src contacts.Source, err error) error {
	var api smithy.APIError
	if errors.As(err, &api) {
		return fmt.Errorf("%s/%s: %s", src.Provider, src.Bucket, api.ErrorCode())
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s/%s: %w", src.Provider, src.Bucket, err)
	}
	return fmt.Errorf("%s/%s: object store unreachable", src.Provider, src.Bucket)
}

// Fetch implements ContactsFetcher: copy one object into w, hashing it while copying.
func (f S3ContactsFetcher) Fetch(ctx context.Context, src contacts.Source, w io.Writer) (int64, string, string, error) {
	client, err := f.client(src)
	if err != nil {
		return 0, "", "", err
	}
	out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(src.Bucket), Key: aws.String(src.Key)})
	if err != nil {
		return 0, "", "", reason(src, err)
	}
	defer out.Body.Close()
	one, two := sha1.New(), sha256.New() //nolint:gosec
	n, err := io.Copy(io.MultiWriter(w, one, two), out.Body)
	if err != nil {
		return n, "", "", reason(src, err)
	}
	return n, hex.EncodeToString(one.Sum(nil)), hex.EncodeToString(two.Sum(nil)), nil
}

// blockReaderAt is an io.ReaderAt over an object that reads it in large ranged blocks and keeps the last few,
// which is what makes archive/zip affordable on a 10 GB archive: its directory is read in a handful of
// requests. It is safe for the sequential use archive/zip makes of it.
type blockReaderAt struct {
	ctx    context.Context
	size   int64
	fetch  func(ctx context.Context, start, end int64) ([]byte, error)
	mu     sync.Mutex
	blocks map[int64][]byte
	order  []int64
}

// ReadAt implements io.ReaderAt.
func (b *blockReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}
	n := 0
	for n < len(p) {
		if off >= b.size {
			return n, io.EOF
		}
		index := off / zipBlockSize
		block, err := b.block(index)
		if err != nil {
			return n, err
		}
		within := off - index*zipBlockSize
		copied := copy(p[n:], block[within:])
		n += copied
		off += int64(copied)
	}
	return n, nil
}

// block returns block number index, fetching it with one ranged request when it is not cached.
func (b *blockReaderAt) block(index int64) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if data, ok := b.blocks[index]; ok {
		return data, nil
	}
	start := index * zipBlockSize
	end := min(start+zipBlockSize, b.size) - 1
	data, err := b.fetch(b.ctx, start, end)
	if err != nil {
		return nil, err
	}
	if b.blocks == nil {
		b.blocks = map[int64][]byte{}
	}
	b.blocks[index] = data
	b.order = append(b.order, index)
	if len(b.order) > zipBlockCache {
		delete(b.blocks, b.order[0])
		b.order = b.order[1:]
	}
	return data, nil
}

// openZip opens the archive at src for reading through ranged requests only.
func (f S3ContactsFetcher) openZip(ctx context.Context, src contacts.Source, size int64) (*zip.Reader, error) {
	client, err := f.client(src)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		head, err := client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(src.Bucket), Key: aws.String(src.Key)})
		if err != nil {
			return nil, reason(src, err)
		}
		size = aws.ToInt64(head.ContentLength)
	}
	reader := &blockReaderAt{ctx: ctx, size: size, fetch: func(ctx context.Context, start, end int64) ([]byte, error) {
		out, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(src.Bucket), Key: aws.String(src.Key), Range: aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
		})
		if err != nil {
			return nil, reason(src, err)
		}
		defer out.Body.Close()
		data, err := io.ReadAll(out.Body)
		if err != nil {
			return nil, reason(src, err)
		}
		return data, nil
	}}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		var api smithy.APIError
		if errors.As(err, &api) || strings.Contains(err.Error(), src.Bucket) {
			return nil, err
		}
		return nil, fmt.Errorf("%s/%s: not a readable ZIP (%v)", src.Provider, src.Bucket, err)
	}
	return archive, nil
}

// ListZipMembers implements ContactsFetcher: the members of the archive, from its central directory.
func (f S3ContactsFetcher) ListZipMembers(ctx context.Context, src contacts.Source, size int64) ([]ZipMember, error) {
	archive, err := f.openZip(ctx, src, size)
	if err != nil {
		return nil, err
	}
	members := make([]ZipMember, 0, len(archive.File))
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		members = append(members, ZipMember{Name: file.Name, Size: int64(file.UncompressedSize64)})
	}
	return members, nil
}

// FetchZipMember implements ContactsFetcher: copy one member of the archive into w with ranged reads, hashing it.
// The ZIP reader checks the member's own CRC-32 while reading.
func (f S3ContactsFetcher) FetchZipMember(ctx context.Context, src contacts.Source, size int64, member string, w io.Writer) (int64, string, error) {
	archive, err := f.openZip(ctx, src, size)
	if err != nil {
		return 0, "", err
	}
	for _, file := range archive.File {
		if file.Name != member {
			continue
		}
		if file.UncompressedSize64 > contactMemberMaxBytes {
			return 0, "", fmt.Errorf("%s/%s: member larger than %d bytes", src.Provider, src.Bucket, contactMemberMaxBytes)
		}
		rc, err := file.Open()
		if err != nil {
			return 0, "", fmt.Errorf("%s/%s: member unreadable (%v)", src.Provider, src.Bucket, err)
		}
		defer rc.Close()
		sum := sha1.New() //nolint:gosec
		n, err := io.Copy(io.MultiWriter(w, sum), io.LimitReader(rc, contactMemberMaxBytes+1))
		if err != nil {
			return n, "", fmt.Errorf("%s/%s: member read failed (%v)", src.Provider, src.Bucket, err)
		}
		return n, hex.EncodeToString(sum.Sum(nil)), nil
	}
	return 0, "", fmt.Errorf("%s/%s: member %q is not in the archive", src.Provider, src.Bucket, member)
}
