// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/stretchr/testify/require"
)

func TestLiteralRPCBindingPreservesValuesResultsAndServerFailures(t *testing.T) {
	values := map[string]any{"digest": "sha256:" + strings.Repeat("a", 64), "source": "source:official", "body": "Exact quote with " + string(rune(92)) + " and a ' character.", "when": "2026-10-04T00:00:00Z"}
	raw, err := json.Marshal(map[string]any{"id": 1, "method": "query", "params": []any{"RETURN $body;", values}})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "https://fixture.invalid/rpc", bytes.NewReader(raw))
	require.NoError(t, err)
	count, err := bindLiteralRPC(req)
	require.NoError(t, err)
	require.Equal(t, len(values), count)
	var bound struct {
		Params []json.RawMessage `json:"params"`
	}
	require.NoError(t, json.NewDecoder(req.Body).Decode(&bound))
	var variables map[string]string
	require.NoError(t, json.Unmarshal(bound.Params[1], &variables))
	var restored map[string]any
	require.NoError(t, json.Unmarshal([]byte(variables["toolkit_bound_json"]), &restored))
	require.Equal(t, values, restored)
	var sql string
	require.NoError(t, json.Unmarshal(bound.Params[0], &sql))
	require.NotContains(t, sql, values["body"])
	require.True(t, strings.HasSuffix(sql, "RETURN $body;"))
	for _, failed := range []bool{false, true} {
		entries := []any{}
		for i := 0; i < count; i++ {
			entries = append(entries, map[string]any{"status": "OK", "result": nil})
		}
		if failed {
			entries = append(entries, map[string]any{"status": "ERR", "result": "Library proposal changed"}, map[string]any{"status": "ERR", "result": "NotExecuted"})
		} else {
			entries = append(entries, map[string]any{"status": "OK", "result": "original result"})
		}
		raw, _ = json.Marshal(map[string]any{"id": 1, "result": entries})
		response, err := trimBindingResults(&http.Response{Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(raw))}, count)
		require.NoError(t, err)
		var reply struct {
			Result []struct {
				Status string
				Result string
			}
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&reply))
		if failed {
			require.Len(t, reply.Result, 2)
			require.Equal(t, "Library proposal changed", reply.Result[0].Result)
		} else {
			require.Len(t, reply.Result, 1)
			require.Equal(t, "original result", reply.Result[0].Result)
		}
	}
}

func TestAccessSQLClientUsesBoundRecordTokenCachesAndRefreshes(t *testing.T) {
	signins, queries := 0, 0
	now := time.Now().UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signin":
			signins++
			require.Empty(t, r.Header.Get("Authorization"))
			var payload map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, ValidatorAccess, payload["AC"])
			require.Equal(t, "mounted-record-name", payload["username"])
			require.Equal(t, "mounted-record-password", payload["password"])
			claims, _ := json.Marshal(map[string]any{"NS": "fct", "DB": "fixture", "AC": ValidatorAccess, "ID": ValidatorPrincipal, "exp": now.Add(time.Minute).Unix()})
			token := "fixture-header." + base64.RawURLEncoding.EncodeToString(claims) + ".fixture-signature"
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "token": token})
		case "/rpc":
			queries++
			require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "))
			require.Empty(t, r.Header.Get("surreal-auth-ns"))
			require.Empty(t, r.Header.Get("surreal-auth-db"))
			require.Equal(t, "fct", r.Header.Get("surreal-ns"))
			_, _ = w.Write([]byte(`{"result":[{"status":"OK","result":true}]}`))
		default:
			t.Fatal("unexpected credential endpoint")
		}
	}))
	defer server.Close()
	client, err := NewAccessSQLClient(AccessConfig{DB: surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "fixture", User: "mounted-record-name", Password: "mounted-record-password", AuthLevel: "database"}, Access: ValidatorAccess, Principal: ValidatorPrincipal}, server.Client().Transport)
	require.NoError(t, err)
	client.HTTP.Transport.(*accessTransport).now = func() time.Time { return now }
	_, err = client.Query(t.Context(), "RETURN true;", nil)
	require.NoError(t, err)
	_, err = client.Query(t.Context(), "RETURN true;", nil)
	require.NoError(t, err)
	require.Equal(t, 1, signins)
	now = now.Add(time.Minute)
	_, err = client.Query(t.Context(), "RETURN true;", nil)
	require.NoError(t, err)
	require.Equal(t, 2, signins)
	require.Equal(t, 3, queries)
}

func TestRecordAuthRejectsSystemWrongPrincipalRedirectAndBudgetWithoutQuery(t *testing.T) {
	for _, mode := range []string{"system token", "wrong principal", "expired", "redirect", "body budget"} {
		t.Run(mode, func(t *testing.T) {
			queries := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/signin" {
					queries++
					t.Error("must not send query after failed signin")
					return
				}
				if mode == "redirect" {
					http.Redirect(w, r, "/rpc", http.StatusTemporaryRedirect)
					return
				}
				if mode == "body budget" {
					_, _ = w.Write([]byte(strings.Repeat("x", 16385)))
					return
				}
				claims := map[string]any{"NS": "fct", "DB": "fixture", "AC": ValidatorAccess, "ID": ValidatorPrincipal, "exp": time.Now().Add(time.Hour).Unix()}
				if mode == "system token" {
					delete(claims, "AC")
					delete(claims, "ID")
				}
				if mode == "wrong principal" {
					claims["ID"] = CurrencyPrincipal
				}
				if mode == "expired" {
					claims["exp"] = time.Now().Add(-time.Hour).Unix()
				}
				raw, _ := json.Marshal(claims)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "token": "header." + base64.RawURLEncoding.EncodeToString(raw) + ".signature"})
			}))
			defer server.Close()
			cfg := AccessConfig{DB: surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "fixture", User: "fixture-user", Password: "never-print-fixture-secret", AuthLevel: "database"}, Access: ValidatorAccess, Principal: ValidatorPrincipal}
			client, err := NewAccessSQLClient(cfg, server.Client().Transport)
			require.NoError(t, err)
			_, err = client.Query(t.Context(), "RETURN true;", nil)
			require.Error(t, err)
			require.NotContains(t, err.Error(), cfg.DB.Password)
			require.Zero(t, queries)
			cfg.DB.AuthLevel = "root"
			_, err = NewAccessSQLClient(cfg, nil)
			require.Error(t, err)
		})
	}
}
