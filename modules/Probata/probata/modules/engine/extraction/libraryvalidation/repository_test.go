// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/stretchr/testify/require"
)

type recordReaderFunc func(context.Context, string) (Record, error)

func (f recordReaderFunc) Read(ctx context.Context, id string) (Record, error) { return f(ctx, id) }

func TestProposalPreparationRetriesDispatchRaceAndUsesServerImmutableDigest(t *testing.T) {
	for _, thrash := range []bool{false, true} {
		t.Run(map[bool]string{false: "one dispatch race", true: "bounded conflicting observations"}[thrash], func(t *testing.T) {
			_, fixture, _ := serviceFixture(t)
			raw, _ := json.Marshal(fixture.proposal)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &fields))
			reads := 0
			fullBefore, fullAfter := "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Params []json.RawMessage `json:"params"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				var sql string
				require.NoError(t, json.Unmarshal(request.Params[0], &sql))
				require.Equal(t, proposalVersionSQL, sql)
				require.NotContains(t, sql, "UPSERT")
				_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{map[string]any{"status": "OK", "result": map[string]any{"full": fullAfter, "immutable": fixture.proposal.Version}}}})
			}))
			t.Cleanup(server.Close)
			r := HTTPRepository{Reader: recordReaderFunc(func(_ context.Context, id string) (Record, error) {
				reads++
				version := fullAfter
				if reads == 1 || thrash {
					version = fullBefore
				}
				return Record{ID: id, Version: version, Fields: fields}, nil
			}), DB: SQLClient{Config: surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "fixture", AuthLevel: "database", User: "validator", Password: "fixture"}, HTTP: server.Client()}}
			p, err := r.Proposal(t.Context(), fixture.proposal.ID)
			if thrash {
				require.ErrorContains(t, err, "bounded preparation reads")
				require.Equal(t, 3, reads)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, reads)
			require.Equal(t, fixture.proposal.Version, p.Version)
			require.Equal(t, fixture.proposal.ProposedRecord, p.ProposedRecord)
		})
	}
}

func TestCaseRecordHTTPContractAndBoundedErrors(t *testing.T) {
	for _, mode := range []string{"json", "sse", "missing", "tool_error", "over_budget"} {
		t.Run(mode, func(t *testing.T) {
			id := "library_proposal:11111111-2222-3333-4444-555555555555"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				var request struct {
					Method string `json:"method"`
					Params struct {
						Name      string            `json:"name"`
						Arguments map[string]string `json:"arguments"`
					} `json:"params"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "tools/call", request.Method)
				require.Equal(t, "case_record", request.Params.Name)
				require.Equal(t, id, request.Params.Arguments["id"])
				if mode == "over_budget" {
					_, _ = w.Write([]byte(strings.Repeat("x", int(MaxArtifactBytes+1))))
					return
				}
				envelope := map[string]any{"available": true, "found": mode != "missing", "id": id, "version": "sha256:" + strings.Repeat("a", 64), "record": map[string]any{"personal_note": "full private body"}}
				body, _ := json.Marshal(envelope)
				result, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{"isError": mode == "tool_error", "content": []any{map[string]any{"type": "text", "text": string(body)}}}})
				if mode == "sse" {
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = w.Write(append(append([]byte("event: message\ndata: "), result...), []byte("\n\n")...))
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(result)
				}
			}))
			defer server.Close()
			reader := MCPRecords{Endpoint: server.URL, Token: "fixture-token", HTTP: server.Client()}
			record, err := reader.Read(t.Context(), id)
			if mode == "tool_error" || mode == "over_budget" {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "private body")
			} else {
				require.NoError(t, err)
				if mode == "missing" {
					require.Empty(t, record.ID)
				} else {
					require.Equal(t, id, record.ID)
					require.Contains(t, string(record.Fields["personal_note"]), "full private body")
				}
			}
		})
	}
}

func TestSQLClientSelectsActualThrowBeforeNotExecuted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rpc", r.URL.Path)
		user, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "fixture-validator", user)
		require.Equal(t, "literal-fixture-secret", password)
		require.Equal(t, "fct-fixture", r.Header.Get("surreal-ns"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"status":"ERR","result":"NotExecuted"},{"status":"ERR","result":"An error occurred: Library source changed"},{"status":"ERR","result":"The query was not executed due to a failed transaction"}]}`))
	}))
	defer server.Close()
	db := SQLClient{Config: surrealsink.Config{URL: server.URL, Namespace: "fct-fixture", Database: "isolated", User: "fixture-validator", Password: "literal-fixture-secret", AuthLevel: "database"}, HTTP: server.Client()}
	_, err := db.Query(t.Context(), "fixture SQL", nil)
	require.ErrorContains(t, err, "Library source changed")
	require.NotContains(t, err.Error(), "NotExecuted")
	require.NotContains(t, err.Error(), "literal-fixture-secret")
}

func TestTrustedCommitUsesRecordAndDatetimeCastsAndReadback(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	check := runClaim(t, s, plan, 0)
	_, err = s.Finish(t.Context(), FinishInput{Plan: plan, Checks: []StepResult{check}})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		var sql string
		require.NoError(t, json.Unmarshal(payload.Params[0], &sql))
		require.Contains(t, sql, "proposal_id: $p.id")
		require.Contains(t, sql, "<datetime> $check.evidence_time")
		require.Contains(t, sql, "<datetime> $receipt.completed_at")
		require.Contains(t, sql, "Library source changed")
		require.Contains(t, sql, "$self")
		require.NotContains(t, sql, "UPSERT $p.target")
		var vars map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload.Params[1], &vars))
		var receipt Receipt
		require.NoError(t, json.Unmarshal(vars["receipt"], &receipt))
		require.Equal(t, repo.receipt, receipt)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{map[string]any{"status": "OK", "result": map[string]any{"id": "library_validation:" + strings.TrimPrefix(receipt.ProposalID, "library_proposal:"), "signature": receipt.Signature}}}})
	}))
	defer server.Close()
	trusted := HTTPRepository{SigningKey: fixtureKey, DB: SQLClient{Config: surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "fixture", User: "validator", Password: "fixture", AuthLevel: "database"}, HTTP: server.Client()}}
	id, err := trusted.Commit(t.Context(), repo.receipt)
	require.NoError(t, err)
	require.Contains(t, id, "library_validation:")
	tampered := repo.receipt
	tampered.ProposedHash = "sha256:" + strings.Repeat("9", 64)
	_, err = trusted.Commit(t.Context(), tampered)
	require.ErrorContains(t, err, "signature")
}

func TestCredentialFilesAreLiteralBoundedAndNeverEvaluated(t *testing.T) {
	for _, test := range []struct {
		name, content, want string
		bad                 bool
	}{
		{"raw password with equals", "secret-with=padding", "secret-with=padding", false},
		{"literal assignment", "PASSWORD='$(never-run) literal'\n", "$(never-run) literal", false},
		{"duplicate key", "PASSWORD=first\nPASSWORD=second\n", "", true},
		{"wrong assignment", "OTHER=value\n", "", true},
		{"multiline raw", "first\nsecond", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "mounted.env")
			require.NoError(t, os.WriteFile(file, []byte(test.content), 0o600))
			got, err := ReadCredentialFile(file, "PASSWORD")
			if test.bad {
				require.Error(t, err)
				require.NotContains(t, err.Error(), test.content)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, got)
			}
		})
	}
}
