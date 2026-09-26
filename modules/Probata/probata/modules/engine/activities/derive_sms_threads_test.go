// Byline: Claude Code · Fable 5.1 · 2026-09-20
// Byline: Claude Code · Opus 5 · 2026-09-21 (merged with the deleted
// derive_structured_text_test.go when the duplicate Activity was removed)
//
// In-memory stores only. These are unit tests: nothing here talks to B2, R2,
// PostgreSQL or Temporal, and none of them is evidence that the deployed
// worker works.

package activities

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/stretchr/testify/require"
)

type memoryObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    int
}

func newMemoryObjectStore(seed map[string][]byte) *memoryObjectStore {
	objects := map[string][]byte{}
	for key, value := range seed {
		objects[key] = value
	}
	return &memoryObjectStore{objects: objects}
}

func (m *memoryObjectStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("no such object: " + key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memoryObjectStore) Put(_ context.Context, bucket, key string, body io.ReadSeeker, _ int64, _ string) error {
	data, err := io.ReadAll(body)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[bucket+"/"+key] = data
	m.puts++
	return err
}

func (m *memoryObjectStore) Exists(_ context.Context, bucket, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[bucket+"/"+key]
	return ok, nil
}

func (m *memoryObjectStore) keys(prefix string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			out = append(out, key)
		}
	}
	return out
}

type fixedLocator struct {
	locator DeriveSourceLocator
	err     error
}

func (f *fixedLocator) ResolveDeriveSource(context.Context, proffer.StageRequest) (DeriveSourceLocator, error) {
	return f.locator, f.err
}

type recordingReceipts struct {
	specs []DeriveReceiptSpec
	// results is keyed by the idempotency coordinate the real store uses, so
	// a second call with the same manifest digest returns the same refs.
	results map[string][2]proffer.Ref
	err     error
}

func (r *recordingReceipts) PersistDerivedGeneration(_ context.Context, spec DeriveReceiptSpec) (proffer.Ref, proffer.Ref, error) {
	if r.err != nil {
		return "", "", r.err
	}
	r.specs = append(r.specs, spec)
	if r.results == nil {
		r.results = map[string][2]proffer.Ref{}
	}
	key := spec.ManifestURI + ":" + spec.ManifestSHA256
	if existing, ok := r.results[key]; ok {
		return existing[0], existing[1], nil
	}
	refs := [2]proffer.Ref{
		proffer.Ref("result-" + string(rune('a'+len(r.results)))),
		proffer.Ref("receipt-" + string(rune('a'+len(r.results)))),
	}
	r.results[key] = refs
	return refs[0], refs[1], nil
}

func smsBackupFixture() string {
	image := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nreal-bytes"))
	return `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms protocol="0" address="+1 (810) 555-0101" date="1700000000000" type="1" body="hello from A" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550202" date="1700000060000" type="2" body="hello from B" read="1" status="-1" contact_name="B" />
  <mms date="1700000180000" msg_box="1" address="8105550101" m_type="132" contact_name="A">
    <parts>
      <part seq="0" ct="image/png" name="pic.png" cl="pic.png" data="` + image + `" />
      <part seq="1" ct="text/plain" name="null" text="see picture" />
    </parts>
    <addrs><addr address="8105550101" type="137" charset="106" /></addrs>
  </mms>
</smses>
`
}

func deriveRequest() proffer.StageRequest {
	refs := map[string]proffer.Ref{"original": "11111111-1111-1111-1111-111111111111"}
	for _, name := range handlerDecisionRefs {
		refs[name] = proffer.Ref("ref-" + name)
	}
	return proffer.StageRequest{
		RequestID: "derive-test-1", SourceVersionRef: "22222222-2222-2222-2222-222222222222",
		DeclaredFormat: "smsbackuprestore_xml", Refs: refs,
	}
}

// newDeriveFixture wires the Activity with the source key the owner's live
// vault actually uses shape-wise: a prefix with a space in it.
func newDeriveFixture(t *testing.T, roots smsthreads.DerivedRoots) (DeriveSMSThreadsActivities, *memoryObjectStore, *recordingReceipts) {
	t.Helper()
	store := newMemoryObjectStore(map[string][]byte{"bkt/vault/v1/sms /sms.xml": []byte(smsBackupFixture())})
	receipts := &recordingReceipts{}
	return DeriveSMSThreadsActivities{
		Locators: &fixedLocator{locator: DeriveSourceLocator{Scheme: "b2", Bucket: "bkt", Key: "vault/v1/sms /sms.xml"}},
		Stores: func(scheme string) (smsthreads.ObjectStore, error) {
			if scheme != "b2" {
				return nil, errors.New("unconfigured scheme " + scheme)
			}
			return store, nil
		},
		Receipts: receipts, DerivedRoots: roots, ScratchRoot: t.TempDir(),
	}, store, receipts
}

func mappedRoots() smsthreads.DerivedRoots {
	return smsthreads.DerivedRoots{{Source: "b2://bkt/vault/v1/", Derived: "b2://bkt/derived/v1/"}}
}

func TestDeriveSMSThreadsPublishesUnderTheMappedDerivedRoot(t *testing.T) {
	activity, store, receipts := newDeriveFixture(t, mappedRoots())

	result, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.NoError(t, err)

	require.Equal(t, stagegraph.DeriveSMSThreads, result.Result.Stage)
	require.Equal(t, proffer.StatusSuccess, result.Result.Status)
	require.NotEmpty(t, result.Result.Ref)
	require.NotEmpty(t, result.Result.ReceiptRef)
	require.False(t, result.Reused)

	const prefix = "b2://bkt/derived/v1/sms /sms.xml/"
	require.Equal(t, prefix, result.DerivedPrefix)
	require.Equal(t, prefix+"manifest.json", result.ManifestURI)
	require.Equal(t, prefix+"threads/", result.ThreadsPrefix)
	require.Len(t, result.ManifestSHA256, 64)
	require.Equal(t, uint64(3), result.Records)
	require.Equal(t, 2, result.ThreadCount)
	require.Equal(t, len(result.Chunks), result.ChunkCount)
	require.False(t, result.ChunksTruncated)

	// Nothing landed beside the original, and the original is untouched.
	require.Empty(t, store.keys("bkt/vault/v1/sms /sms.xml."))
	require.Equal(t, smsBackupFixture(), string(store.objects["bkt/vault/v1/sms /sms.xml"]))
	require.NotEmpty(t, store.keys("bkt/derived/v1/sms /sms.xml/media/"))

	// The manifest digest must be of the bytes the store actually serves, and
	// the manifest must record the source locator.
	manifest, digest, err := smsthreads.LoadManifestWithDigest(
		context.Background(), store, "bkt", "derived/v1/sms /sms.xml/manifest.json")
	require.NoError(t, err)
	require.Equal(t, digest, result.ManifestSHA256)
	require.Equal(t, "b2://bkt/vault/v1/sms /sms.xml", manifest.Source)
	require.True(t, json.Valid(store.objects["bkt/derived/v1/sms /sms.xml/manifest.json"]))

	for _, chunk := range result.Chunks {
		require.True(t, strings.HasPrefix(chunk.URI, prefix+"threads/"), chunk.URI)
		require.Len(t, chunk.SHA256, 64)
		require.Equal(t, DerivedChunkDeclaredFormat, chunk.DeclaredFormat)
		require.Greater(t, chunk.Records, uint64(0))
	}

	require.Len(t, receipts.specs, 1)
	require.Equal(t, "b2://bkt/vault/v1/sms /sms.xml", receipts.specs[0].SourceLocator)
	require.Equal(t, int32(1), receipts.specs[0].Attempt)
}

// An unconfigured DERIVED_ROOTS_JSON must not break the route: it falls back
// to the old beside-the-original placement.
func TestDeriveSMSThreadsFallsBackBesideTheOriginalWhenNoRootMatches(t *testing.T) {
	activity, store, _ := newDeriveFixture(t, smsthreads.DerivedRoots{
		{Source: "b2://other/vault/", Derived: "b2://other/derived/"},
	})

	result, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.NoError(t, err)
	require.Equal(t, "b2://bkt/vault/v1/sms /sms.xml.derived/", result.DerivedPrefix)
	require.NotEmpty(t, store.keys("bkt/vault/v1/sms /sms.xml.derived/"))
}

// A second identical call republishes nothing, reports Reused, and returns the
// same durable references — the retry-safety proof.
func TestDeriveSMSThreadsIsIdempotentOnRetry(t *testing.T) {
	activity, store, receipts := newDeriveFixture(t, mappedRoots())

	first, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.NoError(t, err)
	putsAfterFirst := store.puts
	require.Greater(t, putsAfterFirst, 1)

	second, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.NoError(t, err)

	require.True(t, second.Reused, "a finished derivation must be reused, not recomputed")
	require.Equal(t, putsAfterFirst, store.puts, "a retry must publish no new objects")
	require.Equal(t, first.ManifestSHA256, second.ManifestSHA256)
	require.Equal(t, first.Result.Ref, second.Result.Ref)
	require.Equal(t, first.Result.ReceiptRef, second.Result.ReceiptRef)
	require.Equal(t, first.ChunkCount, second.ChunkCount)
	require.Len(t, receipts.results, 1, "the same derivation must not mint a second result reference")
}

func TestDeriveSMSThreadsHeartbeatsWhileStreaming(t *testing.T) {
	activity, _, _ := newDeriveFixture(t, mappedRoots())
	var mu sync.Mutex
	var beats []Progress
	activity.Heartbeat = func(_ context.Context, progress Progress) {
		mu.Lock()
		defer mu.Unlock()
		beats = append(beats, progress)
	}

	_, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, beats, "a long stream must report liveness")
	for _, beat := range beats {
		require.Equal(t, stagegraph.DeriveSMSThreads, beat.Stage)
	}
	require.Greater(t, beats[len(beats)-1].MembersComplete, int64(0))
}

func TestDeriveSMSThreadsFailsClosedOnMissingInputs(t *testing.T) {
	base, _, _ := newDeriveFixture(t, mappedRoots())

	incomplete := deriveRequest()
	delete(incomplete.Refs, "handler_validation")
	_, err := base.DeriveSMSThreads(context.Background(), incomplete)
	require.ErrorContains(t, err, "all-or-none")
	require.True(t, nonRetryable(err), "a malformed request must not be retried")

	noOriginal := deriveRequest()
	delete(noOriginal.Refs, "original")
	_, err = base.DeriveSMSThreads(context.Background(), noOriginal)
	require.ErrorContains(t, err, "retained original reference")

	noRequest := deriveRequest()
	noRequest.RequestID = ""
	_, err = base.DeriveSMSThreads(context.Background(), noRequest)
	require.ErrorContains(t, err, "request and source version references")

	unwired := DeriveSMSThreadsActivities{}
	_, err = unwired.DeriveSMSThreads(context.Background(), deriveRequest())
	require.ErrorContains(t, err, "source locator store is required")
}

// A source reachable only through a worker-local copy, or through a scheme the
// worker has no store for, is a permanent failure: retrying cannot help.
func TestDeriveSMSThreadsRejectsUnusableLocatorsWithoutRetrying(t *testing.T) {
	for name, locator := range map[string]DeriveSourceLocator{
		"no scheme":           {Bucket: "bkt", Key: "vault/sms.xml"},
		"no bucket":           {Scheme: "b2", Key: "vault/sms.xml"},
		"no key":              {Scheme: "b2", Bucket: "bkt"},
		"unconfigured scheme": {Scheme: "gs", Bucket: "bkt", Key: "vault/sms.xml"},
	} {
		activity, _, _ := newDeriveFixture(t, mappedRoots())
		activity.Locators = &fixedLocator{locator: locator}
		_, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
		require.Truef(t, nonRetryable(err), "%s must be a permanent failure, got %v", name, err)
	}

	// A missing object may be a storage hiccup or a late upload: keep retrying.
	activity, _, _ := newDeriveFixture(t, mappedRoots())
	activity.Locators = &fixedLocator{locator: DeriveSourceLocator{Scheme: "b2", Bucket: "bkt", Key: "absent.xml"}}
	_, err := activity.DeriveSMSThreads(context.Background(), deriveRequest())
	require.Error(t, err)
	require.False(t, nonRetryable(err))
}

// TestDeriveResultBoundsChunkReferences proves history stays compact: beyond
// the bound the manifest is the only listing, and the slice is never nil.
func TestDeriveResultBoundsChunkReferences(t *testing.T) {
	var empty proffer.DeriveResult
	empty.BoundChunks()
	require.NotNil(t, empty.Chunks)
	encoded, err := json.Marshal(empty)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"chunks":[]`, "a nil slice must encode as [], never null")

	large := proffer.DeriveResult{Chunks: make([]proffer.DerivedChunkRef, 1000)}
	large.BoundChunks()
	require.True(t, large.ChunksTruncated)
	require.LessOrEqual(t, len(large.Chunks), 256)
}
