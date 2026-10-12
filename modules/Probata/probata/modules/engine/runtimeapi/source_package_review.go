package runtimeapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/sourcemeta"
)

// NativePackageReviewStore extends metadata corrections to registered native version coordinates.
// Inputs: an explicit review action. Output: durable receipt; effects: existing correction ledger only.
// Choose without requiring retained objects or manufacturing preview bindings.
type NativePackageReviewStore interface {
	PersistPackageReview(context.Context, postgres.PackageReviewSpec) (sourcemeta.Receipt, error)
}

// correctPackage records one owner review action through SourceMetadataStore's native seam.
// Inputs: source version path, actor/key and bounded review JSON. Output: durable receipt or conflict.
// Effects: atomic review revision plus existing canonical working-candidate writer; no source/body mutation or evidence promotion.
func (h *SourceMetadataHTTPHandler) correctPackage(w http.ResponseWriter, r *http.Request) {
	store, ok := h.store.(NativePackageReviewStore)
	if !ok {
		h.fail(w, errors.New("native metadata corrections unavailable"))
		return
	}
	var spec postgres.PackageReviewSpec
	if err := decodeOverlayJSON(w, r, &spec); err != nil {
		previewError(w, 422, err)
		return
	}
	uid, user, key, err := overlayActor(r)
	if err != nil {
		previewError(w, 401, err)
		return
	}
	spec.SourceVersionRef = r.PathValue("source_version_ref")
	spec.ActorSubjectUID = uid
	spec.ActorUsername = user
	spec.IdempotencyKey = key
	if err = postgres.ValidatePackageReview(spec); err != nil {
		previewError(w, 422, err)
		return
	}
	receipt, err := store.PersistPackageReview(r.Context(), spec)
	if err != nil {
		if errors.Is(err, entities.ErrConflict) {
			previewError(w, 409, err)
			return
		}
		var invalid service.ErrInvalid
		if errors.As(err, &invalid) {
			previewError(w, 422, err)
			return
		}
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusCreated, receipt)
}
