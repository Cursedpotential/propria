// Byline: Claude Code · Opus 5.5 · 2026-09-25 (Review Actions: run source-context read-back)
package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourcecontext"
)

type sourceContextReaderStub struct {
	sourceContextValidatorStub
	registration        *sourcecontext.Registration
	current             *sourcecontext.Revision
	requestID, sourceID string
}

func (s *sourceContextReaderStub) CurrentSourceContext(_ context.Context, requestID, sourceRef string) (sourcecontext.Revision, bool, error) {
	s.requestID, s.sourceID = requestID, sourceRef
	if s.current == nil {
		return sourcecontext.Revision{}, false, nil
	}
	return *s.current, true, nil
}

func (s *sourceContextReaderStub) SourceRegistration(_ context.Context, requestID string) (sourcecontext.Registration, bool, error) {
	if s.registration == nil {
		return sourcecontext.Registration{}, false, nil
	}
	return *s.registration, true, nil
}

func readerTestHandler(t *testing.T, reader sourcecontext.Validator) (*PreviewHTTPHandler, string) {
	t.Helper()
	store := NewMemoryPreviewStore(&countingEntropy{next: 1})
	workflow := &previewWorkflowStub{state: proffer.PreviewState{Phase: proffer.PhaseAwaitingDecision}}
	handler, err := NewPreviewHTTPHandler(workflow, store, store, store, bytes.Repeat([]byte("k"), 32), serviceTokenPath(t), reader)
	require.NoError(t, err)
	return handler, startPreview(t, handler)
}

func TestRunSourceContextReturnsNewestRevisionAndRegistration(t *testing.T) {
	sha := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	bytesRetained := int64(4096)
	contextRef := "33333333-3333-3333-3333-333333333333"
	reader := &sourceContextReaderStub{
		registration: &sourcecontext.Registration{
			SourceVersionRef: "44444444-4444-4444-4444-444444444444", DeclaredFormat: "sms_xml",
			SourceContextRef: &contextRef, OriginalSHA256: &sha, OriginalBytes: &bytesRetained,
		},
		current: &sourcecontext.Revision{
			SourceContextRef: "55555555-5555-5555-5555-555555555555", Revision: 2,
			ObservedSource: sourcecontext.ObservedSource{Key: "sms.xml", Name: "sms.xml", ByteLength: 4096, ETag: "sha256:" + sha, PreviewSHA256: sha, VerificationState: "preview_only"},
			Assertions:     sourcecontext.HumanAssertions{SourceClass: "first_party", Context: "Phone backup"},
			ChangeReason:   "Corrected the date range", ActorUsername: "operator",
			ReceiptRef: "proffer-source-context://55555555-5555-5555-5555-555555555555", RecordedAt: time.Unix(20, 0).UTC(),
		},
	}
	handler, handle := readerTestHandler(t, reader)

	recorder := servePreview(handler.Routes(), http.MethodGet, "/reference-import/previews/"+handle+"/source-context", nil)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "request-1", reader.requestID)
	require.Equal(t, "upload://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", reader.sourceID)
	var response RunSourceContext
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, handle, response.PreviewHandle)
	require.Equal(t, "request-1", response.RequestID)
	require.Equal(t, proffer.Ref("options-1"), response.ParserOptionsRef)
	require.NotNil(t, response.Registration)
	require.Equal(t, "sms_xml", response.Registration.DeclaredFormat)
	require.Equal(t, sha, *response.Registration.OriginalSHA256)
	require.NotNil(t, response.Current)
	require.Equal(t, 2, response.Current.Revision)
	require.Equal(t, "Phone backup", response.Current.Assertions.Context)
	require.Equal(t, "sms.xml", response.Current.ObservedSource.Key)
}

func TestRunSourceContextWithoutOperatorContextIsAnOrdinaryAnswer(t *testing.T) {
	handler, handle := readerTestHandler(t, &sourceContextReaderStub{})

	recorder := servePreview(handler.Routes(), http.MethodGet, "/reference-import/previews/"+handle+"/source-context", nil)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Nil(t, response["current"])
	require.Nil(t, response["registration"])
}

func TestRunSourceContextFailsClosedWithoutAReaderOrABinding(t *testing.T) {
	handler, _ := readerTestHandler(t, sourceContextValidatorStub{})
	recorder := servePreview(handler.Routes(), http.MethodGet, "/reference-import/previews/"+"unknown_handle_abcdefghijklmnopqrstuvwxyz"+"/source-context", nil)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())

	readable, _ := readerTestHandler(t, &sourceContextReaderStub{})
	missing := servePreview(readable.Routes(), http.MethodGet, "/reference-import/previews/"+"unknown_handle_abcdefghijklmnopqrstuvwxyz"+"/source-context", nil)
	require.Equal(t, http.StatusNotFound, missing.Code, missing.Body.String())
}
