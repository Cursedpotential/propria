// Byline: Claude Code · Opus 5.5 · 2026-09-26
package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/contextreview"
	"github.com/Cursedpotential/probata/engine/sourcemeta"
)

const (
	overlayTestHandle  = "overlayhandle_0123456789abcdefABCDEF"
	overlayTestMessage = "0190a000-0000-7000-8000-0000000000e1"
	overlayTestSHA     = "abababababababababababababababababababababababababababababababab"
)

type contextReviewStoreStub struct {
	readHorizon   contextreview.Horizon
	readCalls     int
	leakFlag      bool
	review        contextreview.ReviewSpec
	foreshadowing contextreview.ForeshadowingSpec
	err           error
}

func (s *contextReviewStoreStub) PersistReview(_ context.Context, spec contextreview.ReviewSpec) (contextreview.Receipt, error) {
	s.review = spec
	return contextreview.Receipt{Ref: "r1", ReceiptRef: "context-review://r1", Revision: 1, Horizon: contextreview.HorizonAsLived}, s.err
}

func (s *contextReviewStoreStub) PersistForeshadowing(_ context.Context, spec contextreview.ForeshadowingSpec) (contextreview.Receipt, error) {
	s.foreshadowing = spec
	return contextreview.Receipt{Ref: "f1", ReceiptRef: "foreshadowing://f1", Revision: 1, Horizon: contextreview.HorizonHindsight}, s.err
}

func (s *contextReviewStoreStub) Read(_ context.Context, subject contextreview.Subject, horizon contextreview.Horizon) (contextreview.View, error) {
	s.readCalls++
	s.readHorizon = horizon
	view := contextreview.View{PreviewHandle: subject.PreviewHandle, MessageID: subject.MessageID, Horizon: horizon}
	// A misbehaving store that returns the flag on every path: the handler
	// must still strip it from an as-lived answer.
	if s.leakFlag || horizon == contextreview.HorizonHindsight {
		flags := []contextreview.ForeshadowingRevision{{FlagRef: "f1", Revision: 1, Foreshadowing: true, Horizon: contextreview.HorizonHindsight}}
		view.Foreshadowing = &flags
	}
	return view, s.err
}

func contextReviewPath(suffix string) string {
	return "/reference-import/previews/" + overlayTestHandle + "/messages/" + overlayTestMessage + suffix
}

func TestContextReviewReadDefaultsToAsLivedAndNeverCarriesTheFlag(t *testing.T) {
	store := &contextReviewStoreStub{leakFlag: true}
	handler, err := NewContextReviewHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)

	recorder := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, contextReviewPath("/context-review"), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, contextreview.HorizonAsLived, store.readHorizon, "an unspecified horizon must read as-lived")
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.NotContains(t, body, "foreshadowing", "an as-lived answer must not carry the foreshadowing member")
	require.Equal(t, "as_lived", body["horizon"])
	require.Equal(t, []any{}, body["reviews"])

	recorder = servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, contextReviewPath("/context-review?horizon=hindsight"), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, contextreview.HorizonHindsight, store.readHorizon)
	require.Contains(t, recorder.Body.String(), `"foreshadowing":[`)

	recorder = servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, contextReviewPath("/context-review?horizon=everything"), nil))
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
}

func TestContextReviewRoutesRequireTailnetAndServiceToken(t *testing.T) {
	store := &contextReviewStoreStub{}
	handler, err := NewContextReviewHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	outside := newPreviewRequest(http.MethodGet, contextReviewPath("/context-review"), nil)
	outside.RemoteAddr = "203.0.113.9:4444"
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(handler.Routes(), outside).Code)
	wrongToken := newPreviewRequest(http.MethodGet, contextReviewPath("/context-review"), nil)
	wrongToken.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(handler.Routes(), wrongToken).Code)
	require.Zero(t, store.readCalls)
}

func TestContextReviewWriteIsActorBoundValidatedAndDigested(t *testing.T) {
	store := &contextReviewStoreStub{}
	handler, err := NewContextReviewHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	body := []byte(`{"supersedes_ref":"","addressed_to":[{"label":"Recipient A","entity_id":null}],
		"about":[{"label":"The child","entity_id":null}],"about_child":"yes","relevant":true,"change_reason":"read the thread"}`)

	missingKey := newPreviewRequest(http.MethodPost, contextReviewPath("/context-review"), body)
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(handler.Routes(), missingKey).Code)

	req := newPreviewRequest(http.MethodPost, contextReviewPath("/context-review"), body)
	req.Header.Set("Idempotency-Key", "review-1")
	recorder := servePreviewRequest(handler.Routes(), req)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	require.Equal(t, "authentik-user-1", store.review.ActorSubjectUID)
	require.Equal(t, "operator", store.review.ActorUsername)
	require.Equal(t, "review-1", store.review.IdempotencyKey)
	require.Equal(t, contextreview.ReviewDigest(store.review), store.review.ContentDigest)
	require.Len(t, store.review.Assertions.AddressedTo, 1)

	for name, bad := range map[string]string{
		"bad about_child": `{"about_child":"maybe","change_reason":"x"}`,
		"missing reason":  `{"relevant":true,"change_reason":" "}`,
		"bad supersedes":  `{"supersedes_ref":"nope","change_reason":"x"}`,
	} {
		req := newPreviewRequest(http.MethodPost, contextReviewPath("/context-review"), []byte(bad))
		req.Header.Set("Idempotency-Key", "bad-"+name)
		require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(handler.Routes(), req).Code, name)
	}
	unknownField := newPreviewRequest(http.MethodPost, contextReviewPath("/context-review"), []byte(`{"foreshadowing":true,"change_reason":"x"}`))
	unknownField.Header.Set("Idempotency-Key", "unknown-field")
	require.Equal(t, http.StatusBadRequest, servePreviewRequest(handler.Routes(), unknownField).Code,
		"the review body cannot smuggle the hindsight flag")
}

func TestForeshadowingWriteAndErrorMapping(t *testing.T) {
	store := &contextReviewStoreStub{}
	handler, err := NewContextReviewHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	req := newPreviewRequest(http.MethodPost, contextReviewPath("/foreshadowing"), []byte(`{"foreshadowing":true,"note":"known later","change_reason":"hindsight"}`))
	req.Header.Set("Idempotency-Key", "flag-1")
	recorder := servePreviewRequest(handler.Routes(), req)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"horizon":"hindsight"`)
	require.True(t, store.foreshadowing.Foreshadowing)
	require.Equal(t, contextreview.ForeshadowingDigest(store.foreshadowing), store.foreshadowing.ContentDigest)

	noValue := newPreviewRequest(http.MethodPost, contextReviewPath("/foreshadowing"), []byte(`{"change_reason":"x"}`))
	noValue.Header.Set("Idempotency-Key", "flag-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(handler.Routes(), noValue).Code)

	for sentinel, status := range map[error]int{
		contextreview.ErrStaleRevision:       http.StatusConflict,
		contextreview.ErrScopeMissing:        http.StatusConflict,
		contextreview.ErrIdempotencyConflict: http.StatusConflict,
		contextreview.ErrSubjectNotFound:     http.StatusNotFound,
		contextreview.ErrNotInstalled:        http.StatusServiceUnavailable,
	} {
		store.err = sentinel
		req := newPreviewRequest(http.MethodPost, contextReviewPath("/foreshadowing"), []byte(`{"foreshadowing":false,"change_reason":"x"}`))
		req.Header.Set("Idempotency-Key", "flag-err")
		require.Equal(t, status, servePreviewRequest(handler.Routes(), req).Code, sentinel.Error())
	}
}

type sourceMetadataStoreStub struct {
	subject    string
	correction sourcemeta.CorrectionSpec
	err        error
}

func (s *sourceMetadataStoreStub) Read(_ context.Context, handle, subject string) (sourcemeta.View, error) {
	s.subject = subject
	return sourcemeta.View{PreviewHandle: handle, SubjectKind: sourcemeta.SubjectSource, SubjectSHA256: overlayTestSHA,
		Metadata: []sourcemeta.MetadataRow{}, Members: []sourcemeta.Member{}, Hashes: []sourcemeta.HashReceipt{},
		Corrections: []sourcemeta.Correction{}, CorrectionsAvailable: true}, s.err
}

func (s *sourceMetadataStoreStub) PersistCorrection(_ context.Context, spec sourcemeta.CorrectionSpec) (sourcemeta.Receipt, error) {
	s.correction = spec
	return sourcemeta.Receipt{CorrectionRef: "c1", ReceiptRef: "metadata-correction://c1", Revision: 1, RecordedAt: time.Now()}, s.err
}

func TestSourceMetadataReadAndCorrection(t *testing.T) {
	store := &sourceMetadataStoreStub{}
	handler, err := NewSourceMetadataHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	base := "/reference-import/previews/" + overlayTestHandle + "/metadata"

	recorder := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, base+"?subject_sha256="+overlayTestSHA, nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, overlayTestSHA, store.subject)
	require.Equal(t, http.StatusUnprocessableEntity,
		servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, base+"?subject_sha256=ABC", nil)).Code)

	body := []byte(`{"subject_sha256":"` + overlayTestSHA + `","field_key":"embedded:EXIF:DateTimeOriginal","supersedes_ref":"",
		"action":"correct","source_value":"2021:05:04 10:11:12","corrected_value":"2021:05:04 22:11:12","change_reason":"clock was off"}`)
	req := newPreviewRequest(http.MethodPost, base+"/corrections", body)
	req.Header.Set("Idempotency-Key", "correction-1")
	recorder = servePreviewRequest(handler.Routes(), req)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	require.Equal(t, overlayTestHandle, store.correction.PreviewHandle)
	require.Equal(t, sourcemeta.CorrectionDigest(store.correction), store.correction.ContentDigest)

	for name, bad := range map[string]string{
		"retract without a target": `{"subject_sha256":"` + overlayTestSHA + `","field_key":"embedded:EXIF:Make","action":"retract","change_reason":"x"}`,
		"correct without a value":  `{"subject_sha256":"` + overlayTestSHA + `","field_key":"embedded:EXIF:Make","action":"correct","change_reason":"x"}`,
		"field key without origin": `{"subject_sha256":"` + overlayTestSHA + `","field_key":"Make","action":"correct","corrected_value":"x","change_reason":"x"}`,
	} {
		req := newPreviewRequest(http.MethodPost, base+"/corrections", []byte(bad))
		req.Header.Set("Idempotency-Key", "bad-"+name)
		require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(handler.Routes(), req).Code, name)
	}

	store.err = sourcemeta.ErrSubjectNotInRun
	req = newPreviewRequest(http.MethodPost, base+"/corrections", body)
	req.Header.Set("Idempotency-Key", "correction-2")
	require.Equal(t, http.StatusNotFound, servePreviewRequest(handler.Routes(), req).Code)
	store.err = errors.New("database down")
	require.Equal(t, http.StatusServiceUnavailable,
		servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, base, nil)).Code)
}
