package activities

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

// batchUnitFixture constructs retained synthetic package metadata without touching any live object store.
// Inputs: explicit VPS fixture root. Outputs: injected Activity and case-bound request. Effects: retained metadata files only.
func batchUnitFixture(t *testing.T) (BatchImportActivities, proffer.BatchUnitRequest) {
	t.Helper()
	base := os.Getenv("BATCH_UNIT_TEST_ROOT")
	if base == "" {
		t.Skip("requires explicit retained VPS fixture root")
	}
	root, err := os.MkdirTemp(base, "batch-unit-*")
	require.NoError(t, err)
	req := proffer.BatchUnitRequest{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", BatchID: "batch-fixture", UnitRoot: "b2://fixture/exports/one/", DeclaredFormat: "unknown_binary"}
	a := BatchImportActivities{UnitManifestRoot: root, UnitAuthorize: func(string) error { return nil },
		Lister: func(context.Context, string, string, string, string, int32) ([]string, string, error) {
			return []string{"exports/one/index.html", "exports/one/attachments/photo.jpg", "exports/one/attachments/.keep", "exports/one/notes.derived/info.json"}, "", nil
		},
		UnitHead: func(_ context.Context, ref string) (proffer.BatchUnitMember, error) {
			return proffer.BatchUnitMember{SourceRef: ref, ProviderVersionID: "immutable-1", ETag: "etag-1", ByteLength: 100}, nil
		},
		UnitRead: func(context.Context, proffer.BatchUnitMember) ([]byte, error) {
			return []byte(`<html><div class="bubble from-me"><span class="meta">fixture</span></div></html>`), nil
		}}
	return a, req
}

// TestFreezeBatchUnitPreservesMembershipAndReusesImmutablePins proves one primary and lossless external attachment inventory.
// Inputs: synthetic HTML plus non-HTML members. Outputs: four exact members and stable retry identity.
// Effects: retained local fixture files only; no model calls, database rows or provider writes.
func TestFreezeBatchUnitPreservesMembershipAndReusesImmutablePins(t *testing.T) {
	a, req := batchUnitFixture(t)
	result, err := a.FreezeBatchUnit(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 4, result.MemberCount)
	require.Equal(t, "imessage_html", result.SelectedFormat)
	require.Equal(t, proffer.Ref("b2://fixture/exports/one/index.html?versionId=immutable-1"), result.PrimaryRef)
	manifest, sha, err := ReadBatchUnitManifest(a.UnitManifestRoot, result.UnitManifestRef, result.ManifestSHA256)
	require.NoError(t, err)
	require.Equal(t, result.ManifestSHA256, sha)
	require.Equal(t, []string{"attachments/.keep", "attachments/photo.jpg", "index.html", "notes.derived/info.json"}, []string{manifest.Members[0].RelativePath, manifest.Members[1].RelativePath, manifest.Members[2].RelativePath, manifest.Members[3].RelativePath})
	a.Lister = func(context.Context, string, string, string, string, int32) ([]string, string, error) {
		t.Fatal("retry relisted mutable membership")
		return nil, "", nil
	}
	a.UnitHead = func(context.Context, string) (proffer.BatchUnitMember, error) {
		t.Fatal("retry inspected a newer provider version")
		return proffer.BatchUnitMember{}, nil
	}
	repeated, err := a.FreezeBatchUnit(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, result, repeated)
	_, _, err = ReadBatchUnitManifest(a.UnitManifestRoot, result.UnitManifestRef, strings.Repeat("a", 64))
	require.ErrorContains(t, err, "SHA")
}

// TestFreezeBatchUnitRefusesAmbiguousUnversionedOrChangingPackages protects bounded and exact inventory admission.
// Inputs: synthetic conflicting metadata. Outputs: explicit refusal and no successful unit manifest.
// Effects: retained test directories only; the workflow is responsible for continuing other units.
func TestFreezeBatchUnitRefusesAmbiguousUnversionedOrChangingPackages(t *testing.T) {
	for _, kind := range []string{"ambiguous", "unversioned", "changed", "outside", "authority", "not-html"} {
		t.Run(kind, func(t *testing.T) {
			a, req := batchUnitFixture(t)
			switch kind {
			case "ambiguous":
				a.Lister = func(context.Context, string, string, string, string, int32) ([]string, string, error) {
					return []string{"exports/one/index.html", "exports/one/index.htm"}, "", nil
				}
			case "unversioned":
				a.UnitHead = func(_ context.Context, ref string) (proffer.BatchUnitMember, error) {
					return proffer.BatchUnitMember{SourceRef: ref, ETag: "x", ByteLength: 3}, nil
				}
			case "changed":
				calls := 0
				a.Lister = func(context.Context, string, string, string, string, int32) ([]string, string, error) {
					calls++
					if calls == 1 {
						return []string{"exports/one/index.html"}, "", nil
					}
					return []string{"exports/one/index.html", "exports/one/new.jpg"}, "", nil
				}
			case "outside":
				a.Lister = func(context.Context, string, string, string, string, int32) ([]string, string, error) {
					return []string{"exports/other/index.html"}, "", nil
				}
			case "authority":
				a.UnitAuthorize = func(string) error { return errors.New("unauthorized unit root") }
			case "not-html":
				a.UnitRead = func(context.Context, proffer.BatchUnitMember) ([]byte, error) {
					return []byte(`{"name":"index.html is not proof"}`), nil
				}
			}
			_, err := a.FreezeBatchUnit(context.Background(), req)
			require.Error(t, err)
		})
	}
}
