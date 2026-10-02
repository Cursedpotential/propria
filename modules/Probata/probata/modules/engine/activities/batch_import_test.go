// Byline: Claude Code · Opus 5 · 2026-09-21
//
// In-memory stores only. These are unit tests; nothing here talks to an
// object store, PostgreSQL or Temporal.

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

type memoryBindingStore struct {
	created  []previewmodel.Binding
	bySource map[proffer.Ref][]previewmodel.Binding
	err      error
}

func (m *memoryBindingStore) Create(_ context.Context, binding previewmodel.Binding) (previewmodel.Binding, error) {
	if m.err != nil {
		return previewmodel.Binding{}, m.err
	}
	for _, existing := range m.created {
		if existing.RequestID == binding.RequestID {
			return existing, nil
		}
	}
	binding.Handle = "handle-" + binding.RequestID
	m.created = append(m.created, binding)
	return binding, nil
}

func (m *memoryBindingStore) BindingsBySourceRef(_ context.Context, sourceRef proffer.Ref, _ int) ([]previewmodel.Binding, error) {
	return m.bySource[sourceRef], nil
}

type fixedOperations struct {
	state proffer.OperationState
	err   error
}

func (f fixedOperations) Operation(context.Context, string) (proffer.OperationState, error) {
	return f.state, f.err
}

func TestListBatchFolderReturnsOnePageAndNeverNil(t *testing.T) {
	var gotLimit int32
	batch := BatchImportActivities{
		Lister: func(_ context.Context, scheme, bucket, prefix, cursor string, limit int32) ([]string, string, error) {
			gotLimit = limit
			require.Equal(t, "b2", scheme)
			require.Equal(t, "bucket", bucket)
			require.Equal(t, "vault/v1/sms /", prefix)
			require.Equal(t, "page-2", cursor)
			return []string{"vault/v1/sms /a.xml"}, "page-3", nil
		},
	}
	result, err := batch.ListBatchFolder(context.Background(), ListBatchFolderRequest{
		Scheme: "b2", Bucket: "bucket", Prefix: "vault/v1/sms /", Cursor: "page-2", Limit: 9000,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"vault/v1/sms /a.xml"}, result.Keys)
	require.Equal(t, "page-3", result.NextCursor)
	require.Equal(t, int32(maxBatchListingPage), gotLimit, "a caller may not ask for an unbounded page")

	empty := BatchImportActivities{
		Lister: func(context.Context, string, string, string, string, int32) ([]string, string, error) {
			return nil, "", nil
		},
	}
	page, err := empty.ListBatchFolder(context.Background(), ListBatchFolderRequest{
		Scheme: "b2", Bucket: "bucket", Prefix: "vault/",
	})
	require.NoError(t, err)
	require.NotNil(t, page.Keys, "a nil slice marshals to null and the BFF rejects it")
}

// Byline: Claude Code · Opus 5.5 · 2026-10-02
func TestListBatchFolderSkipsDerivedOutputsBelowThePrefix(t *testing.T) {
	keys := []string{
		"cv/8102959302/sms-a.xml",
		"cv/8102959302/sms-a.xml.derived/manifest.json",
		"cv/8102959302/sms-a.xml.derived/media/0a.png",
		"cv/8102959302/sms-b.xml",
	}
	batch := BatchImportActivities{
		Lister: func(_ context.Context, _, _, prefix, _ string, _ int32) ([]string, string, error) {
			out := []string{}
			for _, k := range keys {
				if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
					out = append(out, k)
				}
			}
			return out, "", nil
		},
	}
	folder, err := batch.ListBatchFolder(context.Background(), ListBatchFolderRequest{Scheme: "b2", Bucket: "b", Prefix: "cv/8102959302/"})
	require.NoError(t, err)
	require.Equal(t, []string{"cv/8102959302/sms-a.xml", "cv/8102959302/sms-b.xml"}, folder.Keys)

	derived, err := batch.ListBatchFolder(context.Background(), ListBatchFolderRequest{Scheme: "b2", Bucket: "b", Prefix: "cv/8102959302/sms-a.xml.derived/media/"})
	require.NoError(t, err)
	require.Equal(t, []string{"cv/8102959302/sms-a.xml.derived/media/0a.png"}, derived.Keys, "a batch over a derived folder still lists its files")
}

func TestListBatchFolderRejectsAnIncompleteLocatorWithoutRetrying(t *testing.T) {
	batch := BatchImportActivities{
		Lister: func(context.Context, string, string, string, string, int32) ([]string, string, error) {
			t.Fatal("the lister must not be reached")
			return nil, "", nil
		},
	}
	_, err := batch.ListBatchFolder(context.Background(), ListBatchFolderRequest{Scheme: "b2", Bucket: "bucket"})
	require.True(t, nonRetryable(err))

	unwired := BatchImportActivities{}
	_, err = unwired.ListBatchFolder(context.Background(), ListBatchFolderRequest{Scheme: "b2", Bucket: "b", Prefix: "p/"})
	require.ErrorContains(t, err, "object lister is required")
}

// A retried bind Activity must return the first handle, never mint a second.
func TestBindImportOperationIsIdempotentOnRequestID(t *testing.T) {
	store := &memoryBindingStore{}
	batch := BatchImportActivities{Bindings: store}
	req := BindImportOperationRequest{
		RequestID: "batch-1-00001", SourceRef: "b2://bucket/a.xml",
		WorkflowID: "batch-1-00001", RunID: "run-1", ParserOptionsRef: "options-1",
	}
	first, err := batch.BindImportOperation(context.Background(), req)
	require.NoError(t, err)
	second, err := batch.BindImportOperation(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, first.PreviewHandle, second.PreviewHandle)
	require.Len(t, store.created, 1)

	_, err = batch.BindImportOperation(context.Background(), BindImportOperationRequest{RequestID: "only"})
	require.True(t, nonRetryable(err))
}

// An unreadable run is "unknown", never "finished": reporting it as terminal
// would release the batch's in-flight slot while the run is still going.
func TestReadImportOperationReportsUnavailableRatherThanFailing(t *testing.T) {
	batch := BatchImportActivities{Operations: fixedOperations{err: errors.New("temporal is unreachable")}}
	result, err := batch.ReadImportOperation(context.Background(), ReadImportOperationRequest{WorkflowID: "w"})
	require.NoError(t, err)
	require.False(t, result.Available)
	require.False(t, result.Terminal)

	live := BatchImportActivities{Operations: fixedOperations{state: proffer.OperationState{
		Lifecycle: proffer.OperationCompleted, Terminal: true, SourceVersionRef: "source-version-1",
	}}}
	result, err = live.ReadImportOperation(context.Background(), ReadImportOperationRequest{WorkflowID: "w"})
	require.NoError(t, err)
	require.True(t, result.Available)
	require.True(t, result.Terminal)
	require.Equal(t, string(proffer.OperationCompleted), result.Lifecycle)
	// Repair re-entry binds a run to Review only once this is set.
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	require.Equal(t, "source-version-1", result.SourceVersionRef)

	_, err = live.ReadImportOperation(context.Background(), ReadImportOperationRequest{})
	require.True(t, nonRetryable(err))
}

func TestFindImportBindingsNeverReturnsNil(t *testing.T) {
	store := &memoryBindingStore{bySource: map[proffer.Ref][]previewmodel.Binding{
		"b2://bucket/a.xml": {{Handle: "h1", RequestID: "r1", WorkflowID: "w1"}},
	}}
	batch := BatchImportActivities{Bindings: store}

	found, err := batch.FindImportBindings(context.Background(), FindImportBindingsRequest{SourceRef: "b2://bucket/a.xml"})
	require.NoError(t, err)
	require.Len(t, found.Bindings, 1)
	require.Equal(t, "h1", found.Bindings[0].PreviewHandle)

	none, err := batch.FindImportBindings(context.Background(), FindImportBindingsRequest{SourceRef: "b2://bucket/absent.xml"})
	require.NoError(t, err)
	require.NotNil(t, none.Bindings)
	require.Empty(t, none.Bindings)

	_, err = batch.FindImportBindings(context.Background(), FindImportBindingsRequest{})
	require.True(t, nonRetryable(err))
}
