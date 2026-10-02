// Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02
//
// Case identity routes on the Proffer starter: the registry read behind the
// Workbench Case page and the owner's edits. Same boundary as every other
// starter route: tailnet peer + mounted service token; every write carries the
// Authentik actor headers and an Idempotency-Key.
//
//	GET  /case-identity?mode=TEST|REAL         the whole Case page
//	GET  /case-identity/lookup?value=...       who used these identifiers (read tool for other apps)
//	POST /case-identity/identifiers            add an identifier
//	POST /case-identity/identifiers/{alias_id} fix an identifier in place
//	POST /case-identity/identifiers/{alias_id}/delete  remove an identifier
//	POST /case-identity/header?mode=TEST|REAL  edit the matter or its court case
//	POST /case-identity/people                 add a person
//	POST /case-identity/people/{person_id}     edit a person
//	POST /case-identity/triage                 dismiss / reopen an identifier tied to nobody
package runtimeapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// CaseIdentityHTTPHandler serves the case identity routes.
type CaseIdentityHTTPHandler struct {
	store            caseidentity.Store
	serviceTokenPath string
}

// NewCaseIdentityHTTPHandler validates its store and the service token.
func NewCaseIdentityHTTPHandler(store caseidentity.Store, serviceTokenPath string) (*CaseIdentityHTTPHandler, error) {
	if store == nil {
		return nil, errors.New("case identity handler requires a store")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &CaseIdentityHTTPHandler{store: store, serviceTokenPath: serviceTokenPath}, nil
}

// CaseIdentityRoutePatterns are the exact mux patterns the starter mounts.
var CaseIdentityRoutePatterns = []string{
	"GET /case-identity",
	"GET /case-identity/lookup",
	"POST /case-identity/identifiers",
	"POST /case-identity/identifiers/{alias_id}",
	"POST /case-identity/identifiers/{alias_id}/delete",
	"POST /case-identity/header",
	"POST /case-identity/people",
	"POST /case-identity/people/{person_id}",
	"POST /case-identity/triage",
	"POST /case-identity/placeholders",
	"POST /case-identity/people/{person_id}/merge",
}

// Routes returns the case identity mux.
func (h *CaseIdentityHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	handlers := []http.HandlerFunc{h.read, h.lookup, h.addIdentifier, h.editIdentifier, h.deleteIdentifier, h.editHeader, h.addPerson, h.editPerson, h.triage, h.addPlaceholders, h.mergePerson}
	for i, pattern := range CaseIdentityRoutePatterns {
		mux.HandleFunc(pattern, overlayAuth(h.serviceTokenPath, "case identity", handlers[i]))
	}
	return mux
}

func (h *CaseIdentityHTTPHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, caseidentity.ErrNotFound):
		previewError(w, http.StatusNotFound, err)
	case errors.Is(err, caseidentity.ErrStale), errors.Is(err, caseidentity.ErrIdempotencyConflict):
		previewError(w, http.StatusConflict, err)
	case errors.Is(err, caseidentity.ErrRejected):
		previewError(w, http.StatusUnprocessableEntity, err)
	case errors.Is(err, caseidentity.ErrNotInstalled):
		previewError(w, http.StatusServiceUnavailable, err)
	default:
		previewError(w, http.StatusServiceUnavailable, errors.New("case identity registry is unavailable"))
	}
}

func caseActor(r *http.Request) (caseidentity.Actor, error) {
	uid, username, err := authenticatedActor(r)
	if err != nil {
		return caseidentity.Actor{}, err
	}
	actor := caseidentity.Actor{SubjectUID: uid, Username: username, IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key"))}
	return actor, caseidentity.ValidateActor(actor)
}

func caseMode(w http.ResponseWriter, r *http.Request) (caseidentity.Mode, bool) {
	mode, err := caseidentity.ParseMode(r.URL.Query().Get("mode"))
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return "", false
	}
	return mode, true
}

func (h *CaseIdentityHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	mode, ok := caseMode(w, r)
	if !ok {
		return
	}
	view, err := h.store.Read(r.Context(), mode)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, view)
}

func (h *CaseIdentityHTTPHandler) lookup(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()["value"]
	if err := caseidentity.ValidateLookup(values); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	matches, err := h.store.Lookup(r.Context(), values)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, map[string]any{"matches": matches, "store": "probata.registry"})
}

// write decodes one bounded body, binds the actor, validates and persists.
func caseWrite[T any](h *CaseIdentityHTTPHandler, w http.ResponseWriter, r *http.Request, validate func(T) error,
	persist func(T, caseidentity.Actor) (caseidentity.Receipt, error)) {
	var body T
	if err := decodeOverlayJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, err := caseActor(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	if err := validate(body); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	receipt, err := persist(body, actor)
	if err != nil {
		h.fail(w, err)
		return
	}
	status := http.StatusCreated
	if receipt.Replayed {
		status = http.StatusOK
	}
	previewJSON(w, status, receipt)
}

func (h *CaseIdentityHTTPHandler) addIdentifier(w http.ResponseWriter, r *http.Request) {
	caseWrite(h, w, r, caseidentity.ValidateIdentifier, func(spec caseidentity.IdentifierSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
		return h.store.AddIdentifier(r.Context(), spec, actor)
	})
}

type identifierEditBody struct {
	Fields       map[string]*string `json:"fields"`
	ChangeReason string             `json:"change_reason"`
}

func (h *CaseIdentityHTTPHandler) editIdentifier(w http.ResponseWriter, r *http.Request) {
	aliasID := r.PathValue("alias_id")
	toSpec := func(body identifierEditBody) caseidentity.IdentifierEditSpec {
		return caseidentity.IdentifierEditSpec{ID: aliasID, Fields: body.Fields, ChangeReason: body.ChangeReason}
	}
	caseWrite(h, w, r, func(body identifierEditBody) error { return caseidentity.ValidateIdentifierEdit(toSpec(body)) },
		func(body identifierEditBody, actor caseidentity.Actor) (caseidentity.Receipt, error) {
			return h.store.EditIdentifier(r.Context(), toSpec(body), actor)
		})
}

type identifierDeleteBody struct {
	ChangeReason string `json:"change_reason"`
}

func (h *CaseIdentityHTTPHandler) deleteIdentifier(w http.ResponseWriter, r *http.Request) {
	aliasID := r.PathValue("alias_id")
	toSpec := func(body identifierDeleteBody) caseidentity.IdentifierDeleteSpec {
		return caseidentity.IdentifierDeleteSpec{ID: aliasID, ChangeReason: body.ChangeReason}
	}
	caseWrite(h, w, r, func(body identifierDeleteBody) error { return caseidentity.ValidateIdentifierDelete(toSpec(body)) },
		func(body identifierDeleteBody, actor caseidentity.Actor) (caseidentity.Receipt, error) {
			return h.store.DeleteIdentifier(r.Context(), toSpec(body), actor)
		})
}

func (h *CaseIdentityHTTPHandler) editHeader(w http.ResponseWriter, r *http.Request) {
	mode, ok := caseMode(w, r)
	if !ok {
		return
	}
	caseWrite(h, w, r, caseidentity.ValidateHeader, func(spec caseidentity.HeaderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
		return h.store.EditHeader(r.Context(), mode, spec, actor)
	})
}

func (h *CaseIdentityHTTPHandler) addPerson(w http.ResponseWriter, r *http.Request) {
	caseWrite(h, w, r, caseidentity.ValidateNewPerson, func(spec caseidentity.NewPersonSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
		return h.store.AddPerson(r.Context(), spec, actor)
	})
}

type personEditBody struct {
	Fields       map[string]*string `json:"fields"`
	ChangeReason string             `json:"change_reason"`
}

func (h *CaseIdentityHTTPHandler) editPerson(w http.ResponseWriter, r *http.Request) {
	personID := r.PathValue("person_id")
	toSpec := func(body personEditBody) caseidentity.PersonSpec {
		return caseidentity.PersonSpec{ID: personID, Fields: body.Fields, ChangeReason: body.ChangeReason}
	}
	caseWrite(h, w, r, func(body personEditBody) error { return caseidentity.ValidatePerson(toSpec(body)) },
		func(body personEditBody, actor caseidentity.Actor) (caseidentity.Receipt, error) {
			return h.store.EditPerson(r.Context(), toSpec(body), actor)
		})
}

func (h *CaseIdentityHTTPHandler) addPlaceholders(w http.ResponseWriter, r *http.Request) {
	caseWrite(h, w, r, caseidentity.ValidatePlaceholders, func(spec caseidentity.PlaceholderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
		return h.store.AddPlaceholders(r.Context(), spec, actor)
	})
}

type mergeBody struct {
	IntoID       string `json:"into_id"`
	ChangeReason string `json:"change_reason"`
}

func (h *CaseIdentityHTTPHandler) mergePerson(w http.ResponseWriter, r *http.Request) {
	personID := r.PathValue("person_id")
	toSpec := func(body mergeBody) caseidentity.MergeSpec {
		return caseidentity.MergeSpec{FromID: personID, IntoID: body.IntoID, ChangeReason: body.ChangeReason}
	}
	caseWrite(h, w, r, func(body mergeBody) error { return caseidentity.ValidateMerge(toSpec(body)) },
		func(body mergeBody, actor caseidentity.Actor) (caseidentity.Receipt, error) {
			return h.store.MergePerson(r.Context(), toSpec(body), actor)
		})
}

func (h *CaseIdentityHTTPHandler) triage(w http.ResponseWriter, r *http.Request) {
	caseWrite(h, w, r, caseidentity.ValidateTriage, func(spec caseidentity.TriageSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
		return h.store.Triage(r.Context(), spec, actor)
	})
}
