// Byline: Codex · GPT-6.1 · 2026-10-04; B2-only source/snapshot admission regressions.
package libraryvalidation

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestR2SourceCoordinatesBlockBeforeFetchAndPreserveHistory(t *testing.T) {
	for _, self := range []bool{false, true} {
		for _, field := range []string{"storage_ref", "source_locator", "source", "primary_url"} {
			s, repo, _ := serviceFixture(t)
			var raw json.RawMessage
			switch field {
			case "storage_ref", "source_locator":
				raw = json.RawMessage(`"r2://retired/source?versionId=old"`)
			case "source":
				raw = json.RawMessage(`{"locator":"r2://retired/source","migration_history":{"uri":"r2://old/history"}}`)
			case "primary_url":
				raw = json.RawMessage(`"https://account.r2.cloudflarestorage.com/source"`)
			}
			if self {
				repo.proposal.Target = repo.proposal.Citations[0].SourceID
				repo.proposal.ExpectedVersion = "absent"
				repo.proposal.Citations[0].SourceVersion = "absent"
				repo.proposal.ProposedRecord[field] = raw
				repo.proposal.ProposedRecord["primary_url"] = json.RawMessage(`"` + fixtureURL + `"`)
				if field == "primary_url" {
					repo.proposal.ProposedRecord[field] = raw
				}
				delete(repo.records, repo.proposal.Target)
			} else {
				repo.records[repo.proposal.Citations[0].SourceID].Fields[field] = raw
			}
			before, _ := json.Marshal(map[string]any{"proposal": repo.proposal, "records": repo.records})
			fetches := 0
			s.Fetcher = fixtureFetcher(t, func(http.ResponseWriter, *http.Request) { fetches++ })
			plan, err := s.Prepare(t.Context(), repo.proposal.ID)
			require.NoError(t, err)
			snapshot, err := s.SourceSnapshot(t.Context(), ClaimInput{Plan: plan, Index: 0})
			require.NoError(t, err)
			require.Equal(t, Blocked, snapshot.Status)
			var retained Snapshot
			require.NoError(t, readSigned(t.Context(), s.Artifacts, snapshot.Ref, &retained, s.SigningKey))
			require.Equal(t, "R2_SOURCE_STORAGE_RETIRED", retained.FailureCode)
			require.Empty(t, retained.Raw.URI)
			require.Zero(t, fetches, "must not fetch official URL as a silent substitute for an active R2 source")
			after, _ := json.Marshal(map[string]any{"proposal": repo.proposal, "records": repo.records})
			require.Equal(t, before, after)
			check, err := s.VerifyClaim(t.Context(), ClaimInput{Plan: plan, Index: 0, SnapshotRef: snapshot.Ref})
			require.NoError(t, err)
			result, err := s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
			require.NoError(t, err)
			require.Equal(t, Blocked, result.Status)
			require.True(t, result.DatabaseCommitted)
			require.Equal(t, "R2_SOURCE_STORAGE_RETIRED", repo.receipt.ClaimChecks[0].FailureCode)
			require.Zero(t, fetches)
		}
	}
	s, repo, _ := serviceFixture(t)
	record := repo.records[repo.proposal.Citations[0].SourceID]
	record.Fields["migration_history"] = json.RawMessage(`{"storage_ref":"r2://retired/history"}`)
	record.Fields["legacy_storage_ref"] = json.RawMessage(`"r2://retired/history"`)
	record.Fields["personal_note"] = json.RawMessage(`"r2://retired/a preserved historical statement"`)
	before, _ := json.Marshal(record.Fields)
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	check := runClaim(t, s, plan, 0)
	require.Equal(t, Verified, check.Status)
	after, _ := json.Marshal(record.Fields)
	require.Equal(t, before, after)
}

func TestB2EndpointAdmissionRejectsRetiredOrAliasedProvidersWithoutPrintingSecrets(t *testing.T) {
	for _, endpoint := range []string{"https://s3.us-west-004.backblazeb2.com", "https://s3.eu-central-003.backblazeb2.com:443/"} {
		require.NoError(t, ValidateB2StorageEndpoint(endpoint))
	}
	for _, endpoint := range []string{"https://account.r2.cloudflarestorage.com", "https://backblazeb2.com.attacker.invalid", "http://s3.us-west-004.backblazeb2.com", "https://private-secret@s3.us-west-004.backblazeb2.com", "https://s3.us-west-004.backblazeb2.com:8443"} {
		err := ValidateB2StorageEndpoint(endpoint)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private-secret")
	}
	file := filepath.Join(t.TempDir(), "storage.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"endpoint_url":"https://account.r2.cloudflarestorage.com","secret_access_key":"private-secret"}`), 0600))
	err := validateB2ConfigFile(file)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-secret")
}

func TestRetiredArtifactCannotBeReadOrUsedAsTrustedCurrencyEvidence(t *testing.T) {
	provider := &fixtureVersionStore{version: "v1"}
	store := B2Artifacts{Store: provider, Bucket: "derivatives", Prefix: "library-validation"}
	ref, err := store.Put(t.Context(), "source.html", []byte(fixtureText), "text/html")
	require.NoError(t, err)
	provider.readVersion = ""
	ref.URI = strings.Replace(ref.URI, "b2://", "r2://", 1)
	_, err = store.Read(t.Context(), ref, MaxSourceBytes)
	require.Error(t, err)
	require.Empty(t, provider.readVersion)
	require.False(t, validPinnedRef(ref))
	check := (ClaimVerifier{}).Verify(t.Context(), Snapshot{Status: Fetched, Raw: ref}, Extracted{})
	require.Equal(t, Blocked, check.Status)
	require.Equal(t, "SNAPSHOT_STORAGE_NOT_B2", check.FailureCode)
	for _, fields := range []map[string]json.RawMessage{
		{"storage": json.RawMessage(`{"provider":"r2"}`)},
		{"source": json.RawMessage(`{"storage":{"endpoint":"https://account.r2.cloudflarestorage.com"}}`)},
	} {
		require.True(t, activeR2Source(fields))
	}
}
