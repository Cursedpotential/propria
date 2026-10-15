package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

// TestBatchUnitRegistrationRequiresExactExternalInventoryBeforePersistence protects immutable package provenance.
// Inputs: retained synthetic manifest and source registration mutations. Outputs: pointer-only metadata or refusal.
// Effects: retained fixture metadata files and no database calls.
func TestBatchUnitRegistrationRequiresExactExternalInventoryBeforePersistence(t *testing.T) {
	base := os.Getenv("BATCH_UNIT_TEST_ROOT")
	if base == "" {
		t.Skip("requires retained VPS fixture root")
	}
	root, err := os.MkdirTemp(base, "unit-registration-*")
	require.NoError(t, err)
	req := proffer.BatchUnitRequest{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", BatchID: "batch-registration", UnitRoot: "b2://fixture/exports/unit/", DeclaredFormat: "unknown_binary"}
	a := activities.BatchImportActivities{UnitManifestRoot: root, UnitAuthorize: func(string) error { return nil }, Lister: func(context.Context, string, string, string, string, int32) ([]string, string, error) {
		return []string{"exports/unit/index.html", "exports/unit/attachments/photo.jpg"}, "", nil
	}, UnitHead: func(_ context.Context, ref string) (proffer.BatchUnitMember, error) {
		return proffer.BatchUnitMember{SourceRef: ref, ProviderVersionID: "v1", ETag: "e1", ByteLength: 20}, nil
	}, UnitRead: func(context.Context, proffer.BatchUnitMember) ([]byte, error) {
		return []byte(`<html><div class="bubble from-me"><span class="meta">synthetic</span></div></html>`), nil
	}}
	frozen, err := a.FreezeBatchUnit(context.Background(), req)
	require.NoError(t, err)
	repo := &SourceLifecycleRepository{unitManifestRoot: root}
	spec := activities.SourceRegistrationSpec{RequestID: req.BatchID + "-00000", MatterID: req.MatterID, CourtCaseID: req.CourtCaseID, AcquisitionRef: frozen.PrimaryRef, DeclaredFormat: frozen.SelectedFormat, Attempt: 1, UnitRootRef: proffer.Ref(req.UnitRoot), UnitManifestRef: frozen.UnitManifestRef, UnitManifestSHA256: frozen.ManifestSHA256, UnitMemberCount: frozen.MemberCount}
	raw, err := repo.batchUnitRegistrationMetadata(spec)
	require.NoError(t, err)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(raw, &metadata))
	require.Equal(t, float64(2), metadata["member_count"])
	require.NotContains(t, metadata, "members")
	require.Equal(t, string(frozen.PrimaryRef), metadata["primary_ref"])
	for _, mutation := range []func(*activities.SourceRegistrationSpec){func(s *activities.SourceRegistrationSpec) {
		s.AcquisitionRef = "b2://fixture/exports/unit/index.html?versionId=v2"
	}, func(s *activities.SourceRegistrationSpec) { s.UnitMemberCount++ }, func(s *activities.SourceRegistrationSpec) { s.UnitRootRef = "b2://fixture/other/" }, func(s *activities.SourceRegistrationSpec) { s.DeclaredFormat = "document" }, func(s *activities.SourceRegistrationSpec) { s.MatterID = "11111111-1111-4111-8111-111111111111" }} {
		changed := spec
		mutation(&changed)
		_, err := repo.batchUnitRegistrationMetadata(changed)
		require.Error(t, err)
	}
}
