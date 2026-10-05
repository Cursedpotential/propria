// Byline: Codex · GPT-6.1 · 2026-10-05. Synthetic fixtures only; no real case records or provider writes.
package librarysync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

var fixtureNow = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
var fixtureScope = Scope{AccountScope: "synthetic-account", Bucket: "synthetic-bucket", LegalRoot: "KnowledgeBase/legal/"}

const fixtureOperation = "01234567-89ab-4cde-8f01-23456789abcd"

type memoryArtifacts struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (m *memoryArtifacts) Put(_ context.Context, key string, raw []byte, _ string) (libraryvalidation.ArtifactRef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hash := libraryvalidation.Hash(raw)
	uri := "b2://synthetic-artifacts/" + key + "/" + hash + "?versionId=fixture"
	m.data[uri] = append([]byte(nil), raw...)
	return libraryvalidation.ArtifactRef{URI: uri, VersionID: "fixture", SHA256: hash, Bytes: int64(len(raw))}, nil
}
func (m *memoryArtifacts) Read(_ context.Context, ref libraryvalidation.ArtifactRef, max int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.data[ref.URI]
	if !ok || int64(len(raw)) > max || int64(len(raw)) != ref.Bytes || libraryvalidation.Hash(raw) != ref.SHA256 {
		return nil, errors.New("fixture artifact integrity")
	}
	return append([]byte(nil), raw...), nil
}

type storedVersion struct {
	Object Object
	Raw    []byte
}
type memoryStorage struct {
	mu           sync.Mutex
	versions     []storedVersion
	head         string
	putCalls     int
	lostReply    bool
	partial      bool
	hashFails    map[string]bool
	headOverride *Object
	headError    bool
	listFunc     func(string, Cursor) (Page, error)
}

func (m *memoryStorage) List(_ context.Context, child string, c Cursor) (Page, error) {
	if m.listFunc != nil {
		return m.listFunc(child, c)
	}
	return Page{Objects: []Object{}, Complete: true}, nil
}
func (m *memoryStorage) Head(_ context.Context, _ string) (*Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.headError {
		return nil, errors.New("fixture HEAD failure")
	}
	if m.headOverride != nil {
		v := *m.headOverride
		return &v, nil
	}
	for _, v := range m.versions {
		if v.Object.VersionID == m.head && !v.Object.Hidden {
			o := v.Object
			return &o, nil
		}
	}
	return nil, nil
}
func (m *memoryStorage) Read(_ context.Context, obj Object, max int64) ([]byte, Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.versions {
		if v.Object.VersionID == obj.VersionID && v.Object.Bucket == obj.Bucket && v.Object.Key == obj.Key && int64(len(v.Raw)) == obj.Size && obj.Size <= max && !v.Object.Hidden {
			o := obj
			o.ContentType = v.Object.ContentType
			o.OperationID = v.Object.OperationID
			o.IntentID = v.Object.IntentID
			return append([]byte(nil), v.Raw...), o, nil
		}
	}
	return nil, obj, errors.New("fixture pinned read failure")
}
func (m *memoryStorage) Hash(ctx context.Context, obj Object) (Object, error) {
	if m.hashFails[obj.VersionID] {
		return obj, errors.New("fixture hash failure")
	}
	raw, out, err := m.Read(ctx, obj, MaxOriginalBytes)
	if err == nil {
		out.SHA256 = digest(raw)
	}
	return out, err
}
func (m *memoryStorage) Put(_ context.Context, op Operation, raw []byte, intent string) (Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putCalls++
	for i := range m.versions {
		m.versions[i].Object.Latest = false
	}
	obj := Object{Bucket: op.Bucket, Key: op.Key, VersionID: fmt.Sprintf("written-%d", m.putCalls), Size: int64(len(raw)), ContentType: op.ContentType, OperationID: op.OperationID, IntentID: intent, UploadedAt: fixtureNow.Add(-time.Second), Latest: true}
	m.versions = append(m.versions, storedVersion{obj, append([]byte(nil), raw...)})
	m.head = obj.VersionID
	if m.lostReply {
		return Object{}, errors.New("fixture lost PUT reply")
	}
	return obj, nil
}
func (m *memoryStorage) History(_ context.Context, _ string) (Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := Page{Objects: []Object{}, Complete: !m.partial}
	for _, v := range m.versions {
		p.Objects = append(p.Objects, v.Object)
	}
	return p, nil
}
func (m *memoryStorage) add(raw []byte, obj Object) {
	obj.Size = int64(len(raw))
	obj.SHA256 = ""
	m.versions = append(m.versions, storedVersion{obj, append([]byte(nil), raw...)})
	if obj.Latest {
		for i := 0; i < len(m.versions)-1; i++ {
			m.versions[i].Object.Latest = false
		}
		m.head = obj.VersionID
	}
}

type memoryBackend struct {
	mu              sync.Mutex
	claim           Claim
	payload         []byte
	completed       Completion
	observations    []Observation
	binding         Binding
	intentCalls     int
	failureCalls    int
	seen            bool
	recordChanged   bool
	forgeSuccess    bool
	loseIntentReply bool
	observeStatus   string
}

func (m *memoryBackend) Seen(context.Context, string) (bool, error) { return m.seen, nil }
func (m *memoryBackend) Claim(_ context.Context, in OperationInput) (Claim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if in.OperationID != m.claim.Operation.OperationID {
		return Claim{}, errors.New("fixture unknown operation")
	}
	return m.claim, nil
}
func (m *memoryBackend) Payload(context.Context, Claim) ([]byte, error) {
	return append([]byte(nil), m.payload...), nil
}
func (m *memoryBackend) BeginWrite(_ context.Context, c Claim, _ string) (Intent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.intentCalls++
	if c.Fence != m.claim.Fence || m.recordChanged {
		return Intent{}, errors.New("fixture stale gate")
	}
	if m.claim.WriteIntentID != "" {
		return Intent{MayWrite: false, IntentID: m.claim.WriteIntentID}, nil
	}
	m.claim.WriteIntentID = "intent-fixture"
	if m.loseIntentReply {
		return Intent{}, errors.New("fixture lost intent reply")
	}
	return Intent{MayWrite: true, IntentID: m.claim.WriteIntentID}, nil
}
func (m *memoryBackend) Observe(_ context.Context, in Observation) (Outcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, old := range m.observations {
		if old.ObservationID == in.ObservationID {
			a, _ := json.Marshal(old)
			b, _ := json.Marshal(in)
			if string(a) != string(b) {
				return Outcome{}, errors.New("fixture evidence collision")
			}
			return Outcome{Status: in.Status, BindingID: in.BindingID}, nil
		}
	}
	m.observations = append(m.observations, in)
	status := in.Status
	if m.observeStatus != "" {
		status = m.observeStatus
	}
	return Outcome{Status: status, BindingID: in.BindingID}, nil
}
func (m *memoryBackend) Complete(_ context.Context, c Claim, in Completion) (Outcome, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed = in
	status := in.Status
	if m.forgeSuccess {
		status = Synced
	} else if m.recordChanged || c.Fence != m.claim.Fence || c.Operation.RecordVersion != m.claim.Operation.RecordVersion || in.BasePointerRevision != m.claim.Operation.BasePointerRevision {
		status = Conflicted
	} else if status == Synced && (in.Written == nil || in.Coverage != "complete" || in.IntentID == "" || in.IntentID != m.claim.WriteIntentID || in.Written.SHA256 != m.claim.Operation.PayloadSHA256 || in.Written.Size != m.claim.Operation.PayloadSize) {
		return Outcome{}, errors.New("fixture independent completion gate")
	}
	return Outcome{Status: status, OperationID: c.Operation.OperationID}, nil
}
func (m *memoryBackend) Failure(_ context.Context, c Claim, status, code string) (Outcome, error) {
	m.failureCalls++
	return Outcome{Status: status, OperationID: c.Operation.OperationID, Code: code}, nil
}
func (m *memoryBackend) Original(context.Context, string, string) (Binding, error) {
	return m.binding, nil
}

type fixtureExtractor struct{ bad bool }

func (f fixtureExtractor) Extract(_ context.Context, s libraryvalidation.Snapshot) (libraryvalidation.Extracted, error) {
	x := libraryvalidation.Extracted{Pages: []string{"Synthetic original exact text"}, InputSHA256: s.Raw.SHA256[len("sha256:"):], VersionID: s.Raw.VersionID, Extractor: "existing-fixture", ExtractorVersion: "fixture/1"}
	if f.bad {
		x.InputSHA256 = digest([]byte("different bytes"))
	}
	return x, nil
}

func fixtureService(t *testing.T) (*Service, *memoryStorage, *memoryBackend) {
	t.Helper()
	key := fixtureScope.LegalRoot + "reference-data/fixture.md"
	binding, _ := fixtureScope.BindingID(key)
	payload := []byte("# Synthetic document\nPrivate context preserved: fixture only.\n")
	op := Operation{Contract: ContractVersion, OperationID: fixtureOperation, BindingID: binding, RecordID: "source:fixture", RecordVersion: "sha256:" + digest([]byte("record")), RevisionRef: "record_revision:" + fixtureOperation, Bucket: fixtureScope.Bucket, Key: key, PayloadSHA256: digest(payload), PayloadSize: int64(len(payload)), ContentType: "text/markdown", CodecVersion: MarkdownCodec, BasePointerRevision: "pointer-0"}
	store := &memoryStorage{hashFails: map[string]bool{}}
	back := &memoryBackend{claim: Claim{Operation: op, LeaseID: "lease-fixture", Fence: 1, ExpiresAt: fixtureNow.Add(time.Hour), Status: Pending}, payload: payload}
	svc, err := NewService(fixtureScope, store, back, &memoryArtifacts{data: map[string][]byte{}}, fixtureExtractor{}, []byte("synthetic-signing-key-at-least-32-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	svc.Now = func() time.Time { return fixtureNow }
	return svc, store, back
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func prepareWrite(t *testing.T, s *Service) (Handle, Handle) {
	t.Helper()
	claim, err := s.ClaimOperation(context.Background(), OperationInput{fixtureOperation, "attempt-1"})
	requireNoError(t, err)
	prepared, err := s.PreparePayload(context.Background(), claim)
	requireNoError(t, err)
	return claim, prepared
}
func reconcile(t *testing.T, s *Service, claim Handle, edit func(*AckInput)) Outcome {
	t.Helper()
	fresh, err := s.RefreshOperation(context.Background(), RecoveryInput{claim, "attempt-1"})
	requireNoError(t, err)
	plan, err := s.PlanReconciliation(context.Background(), fresh)
	requireNoError(t, err)
	in := AckInput{Plan: plan.Handle, Checks: []Handle{}}
	for i := 0; i < plan.Count; i++ {
		h, err := s.HashReconciliation(context.Background(), HashInput{plan.Handle, i})
		requireNoError(t, err)
		in.Checks = append(in.Checks, h)
	}
	in.Current, err = s.CheckCurrent(context.Background(), plan.Handle)
	requireNoError(t, err)
	if edit != nil {
		edit(&in)
	}
	out, err := s.Acknowledge(context.Background(), in)
	requireNoError(t, err)
	return out
}
