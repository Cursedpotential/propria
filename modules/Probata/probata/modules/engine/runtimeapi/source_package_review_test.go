package runtimeapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/sourcemeta"
	"github.com/stretchr/testify/require"
)

// nativePackageStoreStub records native writes alongside the established legacy metadata seam.
// Inputs: source-version review. Output: receipt; effects: in-memory capture only.
type nativePackageStoreStub struct {
	sourceMetadataStoreStub
	native postgres.PackageReviewSpec
}

// PersistPackageReview captures the attributed native request without any database mutation.
// Input: reviewed assertion. Output: stable receipt; choose for route boundary tests.
func (s *nativePackageStoreStub) PersistPackageReview(_ context.Context, spec postgres.PackageReviewSpec) (sourcemeta.Receipt, error) {
	s.native = spec
	return sourcemeta.Receipt{CorrectionRef: "real-correction", ReceiptRef: "metadata-correction://real-correction"}, s.err
}

// TestNativeReviewRouteUsesExistingActorAndWriter verifies native actions bypass no existing actor contract.
// Inputs: tailnet/service-authenticated HTTP requests. Outputs: exact source pin, actor and conflict checks.
// Effects: in-memory HTTP only; legacy source/chunk read routes remain independent.
func TestNativeReviewRouteUsesExistingActorAndWriter(t *testing.T) {
	store := &nativePackageStoreStub{}
	handler, err := NewSourceMetadataHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	path := "/reference-import/context/packages/11111111-1111-4111-8111-111111111111/review"
	body := []byte(`{"operation":"edit","item_id":"native:3","revision":"","fields":{"label":"Corrected"},"source_refs":[{"source_ref":"b2://case/unit.xml","source_version_ref":"11111111-1111-4111-8111-111111111111","locator":{"ordinal":3}}],"reason":"Owner correction"}`)
	req := newPreviewRequest(http.MethodPost, path, body)
	req.Header.Set("Idempotency-Key", "native-click")
	response := servePreviewRequest(handler.Routes(), req)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.Equal(t, "11111111-1111-4111-8111-111111111111", store.native.SourceVersionRef)
	require.Equal(t, "native-click", store.native.IdempotencyKey)
	require.NotEmpty(t, store.native.ActorSubjectUID)
	store.err = sourcemeta.ErrStaleRevision
	req = newPreviewRequest(http.MethodPost, path, body)
	req.Header.Set("Idempotency-Key", "stale-click")
	require.Equal(t, http.StatusConflict, servePreviewRequest(handler.Routes(), req).Code)
}
