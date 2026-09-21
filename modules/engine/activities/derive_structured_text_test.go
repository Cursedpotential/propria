// Byline: Claude Code · Opus 5 · 2026-09-20

package activities

import (
	"bytes"
	"context"
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

// memoryObjectStore is the same in-memory shape derive/smsthreads tests use:
// a map of bucket/key to bytes, so this Activity is exercised against real
// derive output rather than a stubbed manifest.
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

func (m *memoryObjectStore) StoreForScheme(scheme string) (smsthreads.ObjectStore, error) {
	if scheme != "b2" {
		return nil, errors.New("unconfigured scheme " + scheme)
	}
	return m, nil
}

type fixedLocator struct {
	locator DeriveSourceLocator
	err     error
	calls   int
}

func (f *fixedLocator) ResolveDeriveSource(context.Context, proffer.StageRequest) (DeriveSourceLocator, error) {
	f.calls++
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

const deriveTestBackup = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms protocol="0" address="+1 (810) 555-0101" date="1700000000000" type="1" body="hello from A" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550101" date="1700000060000" type="2" body="reply to A" read="1" status="-1" contact_name="A" />
  <sms protocol="0" address="8105550202" date="1700000120000" type="1" body="hello from B" read="1" status="-1" contact_name="B" />
</smses>
`

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

func newDeriveFixture(t *testing.T) (DeriveActivities, *memoryObjectStore, *recordingReceipts) {
	t.Helper()
	store := newMemoryObjectStore(map[string][]byte{"bkt/vault/sms.xml": []byte(deriveTestBackup)})
	receipts := &recordingReceipts{}
	return DeriveActivities{
		Locators: &fixedLocator{locator: DeriveSourceLocator{Scheme: "b2", Bucket: "bkt", Key: "vault/sms.xml"}},
		Stores:   store, Receipts: receipts, ScratchRoot: t.TempDir(),
	}, store, receipts
}

func TestDeriveStructuredTextPublishesBesideSourceAndReturnsAReceipt(t *testing.T) {
	activity, store, receipts := newDeriveFixture(t)

	result, err := activity.DeriveStructuredText(context.Background(), deriveRequest())
	require.NoError(t, err)

	require.Equal(t, stagegraph.DeriveStructuredText, result.Result.Stage)
	require.Equal(t, proffer.StatusSuccess, result.Result.Status)
	require.NotEmpty(t, result.Result.Ref)
	require.NotEmpty(t, result.Result.ReceiptRef)
	require.False(t, result.Reused)

	require.Equal(t, "b2://bkt/vault/sms.xml.derived/manifest.json", result.ManifestURI)
	require.Len(t, result.ManifestSHA256, 64)
	require.Equal(t, "b2://bkt/vault/sms.xml.derived/", result.DerivedPrefix)
	require.Equal(t, uint64(3), result.Records)
	require.Equal(t, 2, result.ThreadCount)
	require.Equal(t, len(result.Chunks), result.ChunkCount)
	require.False(t, result.ChunksTruncated)

	// The manifest digest must be of the bytes the store actually serves.
	published, ok := store.objects["bkt/vault/sms.xml.derived/manifest.json"]
	require.True(t, ok)
	_, digest, err := smsthreads.LoadManifestWithDigest(context.Background(), store, "bkt", "vault/sms.xml.derived/manifest.json")
	require.NoError(t, err)
	require.Equal(t, digest, result.ManifestSHA256)
	require.True(t, json.Valid(published))

	// The original is never modified.
	require.Equal(t, deriveTestBackup, string(store.objects["bkt/vault/sms.xml"]))

	// Every chunk is a locator a successor run can declare as ndjson.
	for _, chunk := range result.Chunks {
		require.True(t, strings.HasPrefix(chunk.URI, "b2://bkt/vault/sms.xml.derived/threads/"), chunk.URI)
		require.Len(t, chunk.SHA256, 64)
		require.Equal(t, DerivedChunkDeclaredFormat, chunk.DeclaredFormat)
		require.Greater(t, chunk.Records, uint64(0))
	}

	require.Len(t, receipts.specs, 1)
	require.Equal(t, "b2://bkt/vault/sms.xml", receipts.specs[0].SourceLocator)
	require.Equal(t, int32(1), receipts.specs[0].Attempt)
}

// TestDeriveStructuredTextIsIdempotentOnRetry is the retry-safety proof: a
// second identical call republishes nothing, reports Reused, and returns the
// same durable references.
func TestDeriveStructuredTextIsIdempotentOnRetry(t *testing.T) {
	activity, store, receipts := newDeriveFixture(t)

	first, err := activity.DeriveStructuredText(context.Background(), deriveRequest())
	require.NoError(t, err)
	putsAfterFirst := store.puts
	require.Greater(t, putsAfterFirst, 1)

	second, err := activity.DeriveStructuredText(context.Background(), deriveRequest())
	require.NoError(t, err)

	require.True(t, second.Reused, "a finished derivation must be reused, not recomputed")
	require.Equal(t, putsAfterFirst, store.puts, "a retry must publish no new objects")
	require.Equal(t, first.ManifestSHA256, second.ManifestSHA256)
	require.Equal(t, first.Result.Ref, second.Result.Ref)
	require.Equal(t, first.Result.ReceiptRef, second.Result.ReceiptRef)
	require.Equal(t, first.ChunkCount, second.ChunkCount)
	require.Len(t, receipts.results, 1, "the same derivation must not mint a second result reference")
}

func TestDeriveStructuredTextHeartbeatsWhileStreaming(t *testing.T) {
	activity, _, _ := newDeriveFixture(t)
	var mu sync.Mutex
	var beats []Progress
	activity.Heartbeat = func(_ context.Context, progress Progress) {
		mu.Lock()
		defer mu.Unlock()
		beats = append(beats, progress)
	}

	_, err := activity.DeriveStructuredText(context.Background(), deriveRequest())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, beats, "a long stream must report liveness")
	for _, beat := range beats {
		require.Equal(t, stagegraph.DeriveStructuredText, beat.Stage)
	}
	require.Greater(t, beats[len(beats)-1].MembersComplete, int64(0))
}

func TestDeriveStructuredTextFailsClosedOnMissingInputs(t *testing.T) {
	base, _, _ := newDeriveFixture(t)

	incomplete := deriveRequest()
	delete(incomplete.Refs, "handler_validation")
	_, err := base.DeriveStructuredText(context.Background(), incomplete)
	require.ErrorContains(t, err, "all-or-none")

	noOriginal := deriveRequest()
	delete(noOriginal.Refs, "original")
	_, err = base.DeriveStructuredText(context.Background(), noOriginal)
	require.ErrorContains(t, err, "retained original reference")

	noRequest := deriveRequest()
	noRequest.RequestID = ""
	_, err = base.DeriveStructuredText(context.Background(), noRequest)
	require.ErrorContains(t, err, "request and source version references")

	unwired := DeriveActivities{}
	_, err = unwired.DeriveStructuredText(context.Background(), deriveRequest())
	require.ErrorContains(t, err, "source locator store is required")
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
