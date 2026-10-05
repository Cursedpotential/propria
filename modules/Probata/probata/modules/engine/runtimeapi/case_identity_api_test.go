// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-10-01
package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

type caseIdentityStoreStub struct {
	readMode   caseidentity.Mode
	readCalls  int
	identifier caseidentity.IdentifierSpec
	edit       caseidentity.IdentifierEditSpec
	deleted    caseidentity.IdentifierDeleteSpec
	header     caseidentity.HeaderSpec
	person     caseidentity.PersonSpec
	actor      caseidentity.Actor
	lookup     []string
	replayed   bool
	err        error
}

func (s *caseIdentityStoreStub) receipt() (caseidentity.Receipt, error) {
	return caseidentity.Receipt{Ref: "r1", Kind: "registry.entity_alias", RecordedAt: time.Unix(0, 0).UTC(), Replayed: s.replayed}, s.err
}

func (s *caseIdentityStoreStub) Read(_ context.Context, mode caseidentity.Mode) (caseidentity.View, error) {
	s.readCalls++
	s.readMode = mode
	return caseidentity.View{Mode: mode, People: []caseidentity.Person{}}, s.err
}

func (s *caseIdentityStoreStub) AddIdentifier(_ context.Context, spec caseidentity.IdentifierSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.identifier, s.actor = spec, actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) EditIdentifier(_ context.Context, spec caseidentity.IdentifierEditSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.edit, s.actor = spec, actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) DeleteIdentifier(_ context.Context, spec caseidentity.IdentifierDeleteSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.deleted, s.actor = spec, actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) EditHeader(_ context.Context, _ caseidentity.Mode, spec caseidentity.HeaderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.header, s.actor = spec, actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) EditPerson(_ context.Context, spec caseidentity.PersonSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.person, s.actor = spec, actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) AddPerson(_ context.Context, _ caseidentity.NewPersonSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.actor = actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) AddPlaceholders(_ context.Context, _ caseidentity.PlaceholderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.actor = actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) AddContactPeople(_ context.Context, _ caseidentity.ContactPeopleSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.actor = actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) MergePerson(_ context.Context, _ caseidentity.MergeSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.actor = actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) Triage(_ context.Context, _ caseidentity.TriageSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	s.actor = actor
	return s.receipt()
}

func (s *caseIdentityStoreStub) Lookup(_ context.Context, values []string) ([]caseidentity.Match, error) {
	s.lookup = values
	return []caseidentity.Match{{Query: values[0], Normalized: caseidentity.NormIdentifier(values[0])}}, s.err
}

func newCaseIdentityHandler(t *testing.T) (*caseIdentityStoreStub, http.Handler) {
	t.Helper()
	store := &caseIdentityStoreStub{}
	handler, err := NewCaseIdentityHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	return store, handler.Routes()
}

const caseTestPerson = "01a0f751-e07b-76b6-afcb-63acfbba373e"

func TestCaseIdentityReadRequiresTailnetTokenAndMode(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	outside := newPreviewRequest(http.MethodGet, "/case-identity?mode=REAL", nil)
	outside.RemoteAddr = "203.0.113.9:4444"
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, outside).Code)
	wrongToken := newPreviewRequest(http.MethodGet, "/case-identity?mode=REAL", nil)
	wrongToken.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, wrongToken).Code)
	require.Zero(t, store.readCalls)

	require.Equal(t, http.StatusOK, servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity", nil)).Code)
	recorder := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity?mode=REAL", nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, caseidentity.ModeReal, store.readMode)
}

func TestCaseIdentityWriteIsActorBoundAndValidated(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	body := []byte(`{"entity_id":"` + caseTestPerson + `","raw_value":"810-252-2779","kind":"phone","status":"confirmed",
		"period":"2024-11..12","basis":"her phone saves it as Matthew Salem","change_reason":"owner confirmed"}`)

	missingKey := newPreviewRequest(http.MethodPost, "/case-identity/identifiers", body)
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, missingKey).Code)

	req := newPreviewRequest(http.MethodPost, "/case-identity/identifiers", body)
	req.Header.Set("Idempotency-Key", "confirm-2779")
	recorder := servePreviewRequest(routes, req)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	require.Equal(t, "operator", store.actor.Username)
	require.Equal(t, "authentik-user-1", store.actor.SubjectUID)
	require.Equal(t, "confirm-2779", store.actor.IdempotencyKey)
	require.Equal(t, "810-252-2779", store.identifier.RawValue)

	store.replayed = true
	replay := newPreviewRequest(http.MethodPost, "/case-identity/identifiers", body)
	replay.Header.Set("Idempotency-Key", "confirm-2779")
	require.Equal(t, http.StatusOK, servePreviewRequest(routes, replay).Code, "a replayed key answers 200 with the first receipt")
	store.replayed = false

	for name, bad := range map[string]string{
		"bad status":  `{"entity_id":"` + caseTestPerson + `","raw_value":"1","kind":"phone","status":"sure","basis":"x"}`,
		"no basis":    `{"entity_id":"` + caseTestPerson + `","raw_value":"1","kind":"phone","status":"confirmed","basis":""}`,
		"bad person":  `{"entity_id":"Matt","raw_value":"1","kind":"phone","status":"confirmed","basis":"x"}`,
		"padded name": `{"entity_id":"` + caseTestPerson + `","raw_value":" Matt","kind":"name","status":"confirmed","basis":"x"}`,
	} {
		req := newPreviewRequest(http.MethodPost, "/case-identity/identifiers", []byte(bad))
		req.Header.Set("Idempotency-Key", "bad-"+name)
		require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, req).Code, name)
	}
	unknown := newPreviewRequest(http.MethodPost, "/case-identity/identifiers", []byte(`{"id":"x","entity_id":"`+caseTestPerson+`"}`))
	unknown.Header.Set("Idempotency-Key", "unknown-field")
	require.Equal(t, http.StatusBadRequest, servePreviewRequest(routes, unknown).Code, "unknown fields are refused")
}

func TestCaseIdentityIdentifierEditAndDelete(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	const alias = "01a0f751-e07b-7000-8000-00000000a001"
	edit := newPreviewRequest(http.MethodPost, "/case-identity/identifiers/"+alias,
		[]byte(`{"fields":{"kind":"legal","period":null},"change_reason":"the caption names her"}`))
	edit.Header.Set("Idempotency-Key", "edit-1")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, edit).Code)
	require.Equal(t, alias, store.edit.ID)
	require.Equal(t, "legal", *store.edit.Fields["kind"])
	require.Nil(t, store.edit.Fields["period"])

	noReason := newPreviewRequest(http.MethodPost, "/case-identity/identifiers/"+alias, []byte(`{"fields":{"kind":"legal"}}`))
	noReason.Header.Set("Idempotency-Key", "edit-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, noReason).Code)

	remove := newPreviewRequest(http.MethodPost, "/case-identity/identifiers/"+alias+"/delete", []byte(`{"change_reason":"typed into the wrong person"}`))
	remove.Header.Set("Idempotency-Key", "delete-1")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, remove).Code)
	require.Equal(t, alias, store.deleted.ID)

	badID := newPreviewRequest(http.MethodPost, "/case-identity/identifiers/not-a-uuid/delete", []byte(`{"change_reason":"x"}`))
	badID.Header.Set("Idempotency-Key", "delete-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, badID).Code)
}

func TestCaseIdentityPersonAndHeaderEdits(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	req := newPreviewRequest(http.MethodPost, "/case-identity/people/"+caseTestPerson,
		[]byte(`{"fields":{"short_name":"Matt"},"change_reason":"label the catalog tools use"}`))
	req.Header.Set("Idempotency-Key", "short-name")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, req).Code)
	require.Equal(t, caseTestPerson, store.person.ID)

	header := newPreviewRequest(http.MethodPost, "/case-identity/header?mode=REAL",
		[]byte(`{"target":"court_case","id":"01a0f751-e07b-76a1-a738-eb3e3aa3e68c","fields":{"docket_number":"2025-53985-DC"},"change_reason":"docket from the court"}`))
	header.Header.Set("Idempotency-Key", "docket")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, header).Code)
	require.Equal(t, "court_case", store.header.Target)

	noMode := newPreviewRequest(http.MethodPost, "/case-identity/header",
		[]byte(`{"target":"matter","id":"01a0f751-e07b-75cc-9ad5-63ad9449a8ba","fields":{"title":"x"},"change_reason":"x"}`))
	noMode.Header.Set("Idempotency-Key", "no-mode")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, noMode).Code)
}

func TestCaseIdentityErrorMappingAndLookup(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	for err, want := range map[error]int{
		caseidentity.ErrNotFound:            http.StatusNotFound,
		caseidentity.ErrStale:               http.StatusConflict,
		caseidentity.ErrIdempotencyConflict: http.StatusConflict,
		caseidentity.ErrRejected:            http.StatusUnprocessableEntity,
		caseidentity.ErrNotInstalled:        http.StatusServiceUnavailable,
	} {
		store.err = err
		req := newPreviewRequest(http.MethodPost, "/case-identity/triage", []byte(`{"raw_value":"34428","decision":"dismissed","basis":"short code"}`))
		req.Header.Set("Idempotency-Key", "triage")
		require.Equal(t, want, servePreviewRequest(routes, req).Code, err.Error())
	}
	store.err = nil
	recorder := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity/lookup?value=%2B1+(810)+268-9630", nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var payload struct {
		Matches []caseidentity.Match `json:"matches"`
		Store   string               `json:"store"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "8102689630", payload.Matches[0].Normalized)
	require.Equal(t, "probata.registry", payload.Store)
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity/lookup", nil)).Code)
}

// Byline: Claude Code · Sonnet · 2026-10-02
func TestCaseIdentityPlaceholdersAndMergeAreActorBoundWrites(t *testing.T) {
	store, routes := newCaseIdentityHandler(t)
	batch := []byte(`{"numbers":["810-555-0142"],"change_reason":"seen in imported calls","dry_run":true}`)
	noKey := newPreviewRequest(http.MethodPost, "/case-identity/placeholders", batch)
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, noKey).Code)
	req := newPreviewRequest(http.MethodPost, "/case-identity/placeholders", batch)
	req.Header.Set("Idempotency-Key", "placeholders-1")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, req).Code)
	require.Equal(t, "placeholders-1", store.actor.IdempotencyKey)

	empty := newPreviewRequest(http.MethodPost, "/case-identity/placeholders", []byte(`{"numbers":[],"change_reason":"x"}`))
	empty.Header.Set("Idempotency-Key", "placeholders-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, empty).Code)

	merge := newPreviewRequest(http.MethodPost, "/case-identity/people/"+caseTestPerson+"/merge",
		[]byte(`{"into_id":"01a0f751-e07b-76c7-8c0f-65692ad656b8","change_reason":"her other phone"}`))
	merge.Header.Set("Idempotency-Key", "merge-1")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, merge).Code)
	self := newPreviewRequest(http.MethodPost, "/case-identity/people/"+caseTestPerson+"/merge",
		[]byte(`{"into_id":"`+caseTestPerson+`","change_reason":"x"}`))
	self.Header.Set("Idempotency-Key", "merge-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, self).Code)
}

// Byline: Claude Code · Sonnet · 2026-10-02
func TestCaseIdentityContactPeopleIsAnActorBoundBatch(t *testing.T) {
	_, routes := newCaseIdentityHandler(t)
	body := []byte(`{"people":[{"display_name":"Jordan Reyes","numbers":["8105550142"],"emails":[],"candidate_names":["J. Reyes"],"source":"b2://k/contacts.vcf"}],"change_reason":"contacts import","dry_run":true}`)
	noKey := newPreviewRequest(http.MethodPost, "/case-identity/contact-people", body)
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, noKey).Code)
	req := newPreviewRequest(http.MethodPost, "/case-identity/contact-people", body)
	req.Header.Set("Idempotency-Key", "contacts-1")
	require.Equal(t, http.StatusCreated, servePreviewRequest(routes, req).Code)
	bad := newPreviewRequest(http.MethodPost, "/case-identity/contact-people", []byte(`{"people":[],"change_reason":"x"}`))
	bad.Header.Set("Idempotency-Key", "contacts-2")
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(routes, bad).Code)
}
