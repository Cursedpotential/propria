// Byline: Claude Code · Opus 5.5 · 2026-09-25

package smsthreads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lowcarbdev/sbv/pkg/parseonly"
)

// realDecodeSMSBackup is captured before any test substitutes the decoder.
var realDecodeSMSBackup = decodeSMSBackup

// stoppingDecoder stands in for the SBV importer: it runs the real decoder
// over the first `keep` bytes of the source, then fails the way the real one
// does on a span it cannot represent.
func stoppingDecoder(t *testing.T, keep int, stop error) {
	t.Helper()
	t.Cleanup(func() { decodeSMSBackup = realDecodeSMSBackup })
	decodeSMSBackup = func(ctx context.Context, source io.Reader, uri string, sink parseonly.ImmutableArtifactSink, emit func(context.Context, parseonly.Record) error) error {
		head := make([]byte, keep)
		if _, err := io.ReadFull(source, head); err != nil {
			return err
		}
		complete := append(append([]byte(nil), head...), []byte("\n</smses>\n")...)
		if err := realDecodeSMSBackup(ctx, bytes.NewReader(complete), uri, sink, emit); err != nil {
			return err
		}
		return stop
	}
}

func cutAfterSecondSMS(source string) int {
	marker := `body="reply to A" read="1" status="-1" contact_name="A" />`
	return strings.Index(source, marker) + len(marker)
}

func TestLenientDeriveKeepsWhatDecodedBeforeADecoderStop(t *testing.T) {
	source := sampleBackup([]byte("png-bytes"))
	stoppingDecoder(t, cutAfterSecondSMS(source), errors.New("SBV smsbackuprestore_xml parse: rejected span lacks complete raw bytes"))
	store := &memoryStore{objects: map[string][]byte{"bkt/vault/sms-cut.xml": []byte(source)}}

	if _, _, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "vault/sms-cut.xml", ScratchRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("a strict derivation must fail when the decoder stops")
	}

	manifest, location, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "vault/sms-cut.xml", ScratchRoot: t.TempDir(),
		Lenient: true, Variant: "lenient",
	})
	if err != nil {
		t.Fatalf("lenient derive: %v", err)
	}
	if manifest.Records != 2 || !manifest.Lenient || manifest.Variant != "lenient" || manifest.DecodedBytes == 0 ||
		!strings.Contains(manifest.StreamError, "lacks complete raw bytes") || !strings.HasSuffix(manifest.Decoder, " lenient") {
		t.Fatalf("lenient manifest = %+v", manifest)
	}
	if manifest.SourceBytes != int64(len(source)) {
		t.Fatalf("source digest must still cover the whole object: %d of %d bytes", manifest.SourceBytes, len(source))
	}
	if location.Prefix != "vault/sms-cut.xml.derived/lenient/" {
		t.Fatalf("lenient output published at %q", location.Prefix)
	}
	if _, ok := store.objects["bkt/vault/sms-cut.xml.derived/manifest.json"]; ok {
		t.Fatal("a lenient derivation must never occupy the strict manifest location")
	}
	report, err := ValidateVariant(context.Background(), store, nil, "lenient", "b2", "bkt", "vault/sms-cut.xml", 0)
	if err != nil || !report.OK || report.Records != 2 {
		t.Fatalf("lenient derivation must validate: %+v err=%v", report, err)
	}
	if !bytes.Equal(store.objects["bkt/vault/sms-cut.xml"], []byte(source)) {
		t.Fatal("the original object was modified")
	}
	// Retry-safe: the published lenient manifest is reused, not re-derived.
	again, _, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "vault/sms-cut.xml", ScratchRoot: t.TempDir(),
		Lenient: true, Variant: "lenient", ReuseExisting: true,
	})
	if err != nil || again.Records != manifest.Records || again.StreamError != manifest.StreamError {
		t.Fatalf("reuse = %+v err=%v", again, err)
	}
}

// Storage, scratch-disk, network and cancellation failures are never a
// "decoder stop": lenient mode must fail on them exactly as strict mode does.
func TestLenientDeriveNeverToleratesEnvironmentFailures(t *testing.T) {
	source := sampleBackup([]byte("png-bytes"))
	for name, stop := range map[string]error{
		"object store":  environmental(errors.New("PUT media: 503 slow down")),
		"scratch disk":  &fs.PathError{Op: "write", Path: "/scratch/x", Err: errors.New("no space left on device")},
		"wrapped store": errors.Join(errors.New("persist attachment 0"), environmental(errors.New("PUT failed"))),
	} {
		stoppingDecoder(t, cutAfterSecondSMS(source), stop)
		store := &memoryStore{objects: map[string][]byte{"bkt/v/sms.xml": []byte(source)}}
		if _, _, err := Derive(context.Background(), Options{
			Store: store, Scheme: "b2", Bucket: "bkt", Key: "v/sms.xml", ScratchRoot: t.TempDir(),
			Lenient: true, Variant: "lenient",
		}); err == nil {
			t.Fatalf("%s failure was tolerated as a decoder stop", name)
		}
	}
}

type failingReadStore struct{ memoryStore }

func (f *failingReadStore) Open(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	body, err := f.memoryStore.Open(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(io.MultiReader(io.LimitReader(body, 200), errReader{errors.New("connection reset by peer")})), nil
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

func TestLenientDeriveFailsOnASourceReadError(t *testing.T) {
	store := &failingReadStore{memoryStore{objects: map[string][]byte{"bkt/v/sms.xml": []byte(sampleBackup([]byte("x")))}}}
	if _, _, err := Derive(context.Background(), Options{
		Store: store, Scheme: "b2", Bucket: "bkt", Key: "v/sms.xml", ScratchRoot: t.TempDir(),
		Lenient: true, Variant: "lenient",
	}); err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("a network failure must fail the lenient derivation, got %v", err)
	}
}

func TestVariantLocationsAreSeparateAndValidated(t *testing.T) {
	store := &memoryStore{objects: map[string][]byte{"bkt/v/a.xml.derived/manifest.json": []byte(`{}`)}}
	location, done, err := ResolvePublishedVariant(context.Background(), store, nil, "lenient", "b2", "bkt", "v/a.xml")
	if err != nil || done || location.Prefix != "v/a.xml.derived/lenient/" {
		t.Fatalf("a strict manifest must not satisfy a lenient lookup: %+v done=%t err=%v", location, done, err)
	}
	for _, bad := range []string{"Lenient", "../x", "a/b", "x y", strings.Repeat("a", 40)} {
		if _, _, err := ResolvePublishedVariant(context.Background(), store, nil, bad, "b2", "bkt", "v/a.xml"); err == nil {
			t.Fatalf("variant %q accepted", bad)
		}
	}
	roots, err := ParseDerivedRoots(`[{"source":"b2://bkt/v/","derived":"b2://bkt/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	mapped, _, err := ResolvePublishedVariant(context.Background(), store, roots, "lenient", "b2", "bkt", "v/a.xml")
	if err != nil || mapped.Prefix != "derived/a.xml/lenient/" {
		t.Fatalf("mapped variant = %+v err=%v", mapped, err)
	}
}

// multipartRecorder is an in-memory multipart API.
type multipartRecorder struct {
	mu        sync.Mutex
	parts     map[int32][]byte
	completed [][]byte
	aborted   bool
	failPart  int32
}

func (m *multipartRecorder) CreateMultipartUpload(context.Context, *s3.CreateMultipartUploadInput, ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	m.parts = map[int32][]byte{}
	return &s3.CreateMultipartUploadOutput{UploadId: aws.String("upload-1")}, nil
}

func (m *multipartRecorder) UploadPart(_ context.Context, in *s3.UploadPartInput, _ ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	if *in.PartNumber == m.failPart {
		return nil, errors.New("part rejected")
	}
	body, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) != *in.ContentLength {
		return nil, errors.New("content length mismatch")
	}
	m.mu.Lock()
	m.parts[*in.PartNumber] = body
	m.mu.Unlock()
	return &s3.UploadPartOutput{ETag: aws.String("etag")}, nil
}

func (m *multipartRecorder) CompleteMultipartUpload(_ context.Context, in *s3.CompleteMultipartUploadInput, _ ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	for index, part := range in.MultipartUpload.Parts {
		if *part.PartNumber != int32(index+1) {
			return nil, errors.New("parts out of order")
		}
		m.completed = append(m.completed, m.parts[*part.PartNumber])
	}
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (m *multipartRecorder) AbortMultipartUpload(context.Context, *s3.AbortMultipartUploadInput, ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	m.aborted = true
	return &s3.AbortMultipartUploadOutput{}, nil
}

func TestMultipartUploadSendsEveryByteInOrder(t *testing.T) {
	body := bytes.Repeat([]byte("0123456789"), 1003) // 10,030 bytes
	api := &multipartRecorder{}
	if err := putMultipart(context.Background(), api, "bkt", "k", bytes.NewReader(body), int64(len(body)), 1000, "application/xml"); err != nil {
		t.Fatal(err)
	}
	if len(api.completed) != 11 || !bytes.Equal(bytes.Join(api.completed, nil), body) || api.aborted {
		t.Fatalf("parts=%d aborted=%t", len(api.completed), api.aborted)
	}
}

func TestMultipartUploadAbortsOnAFailedPart(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 5000)
	api := &multipartRecorder{failPart: 3}
	err := putMultipart(context.Background(), api, "bkt", "k", bytes.NewReader(body), int64(len(body)), 1000, "application/xml")
	if err == nil || !api.aborted || len(api.completed) != 0 {
		t.Fatalf("err=%v aborted=%t completed=%d", err, api.aborted, len(api.completed))
	}
}
