// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type handlerAdmissionDB struct {
	content      []byte
	transactions int
	reads        int
}
type handlerAdmissionRow struct {
	content []byte
	digest  bool
}

// BeginTx records forbidden recommendation persistence in credential admission tests.
// Inputs: context/options. Output: fixed error. Side effects: counter update. Pick for the admission test fake.
func (db *handlerAdmissionDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	db.transactions++
	return nil, errors.New("unexpected recommendation persistence")
}

// Query rejects unexpected broad queries in the credential admission test fake.
// Inputs: query. Output: fixed error. Side effects: none. Pick for the admission test fake.
func (db *handlerAdmissionDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

// QueryRow supplies only retained digest and original bytes for the recommendation admission test.
// Inputs: query. Output: fake row. Side effects: read count update. Pick for this test's read-only setup.
func (db *handlerAdmissionDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	db.reads++
	return handlerAdmissionRow{content: db.content, digest: strings.Contains(query, "object.content_sha256")}
}

// Scan assigns either the existing custody digest or inline retained content to test destinations.
// Inputs: destination pointers. Output: nil. Side effects: assigns fixture data. Pick in the test fake.
func (row handlerAdmissionRow) Scan(dest ...any) error {
	if row.digest {
		*dest[0].(*[]byte) = make([]byte, 32)
		return nil
	}
	*dest[0].(*string) = "inline"
	*dest[1].(*string) = ""
	*dest[2].(*[]byte) = row.content
	return nil
}

// TestRecommendHandlerRejectsCredentialContentBeforeRecommendation exercises the real engine recommendation boundary.
// Input: retained inline synthetic credentials and valid durable references. Output: fixed policy exclusion error.
// Side effects: fake read counts only. Pick to prove no recommendation/transaction precedes the credential gate.
func TestRecommendHandlerRejectsCredentialContentBeforeRecommendation(t *testing.T) {
	for _, content := range []string{
		"API_KEY=Ab12Cd34Ef56Gh78Ij90",
		`[{"uuid":"c1","name":"Chat","chat_messages":[{"uuid":"m1","sender":"human","text":"Here is the config\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90"}]}]`,
		"name,url,username,password\nSite,https://site.test,user,Ab12Cd34\n",
		strings.Repeat("x", int(handlerSignatureReadLimit)+512) + "\nAPI_KEY=Ab12Cd34Ef56Gh78Ij90",
	} {
		db := &handlerAdmissionDB{content: []byte(content)}
		store, err := NewHandlerSelectionStore(db, nil)
		if err != nil {
			t.Fatal(err)
		}
		result, err := store.RecommendHandler(context.Background(), proffer.StageRequest{RequestID: "credential-admission-test", SourceVersionRef: proffer.Ref(uuid.NewString()), Refs: map[string]proffer.Ref{"original": proffer.Ref(uuid.NewString())}}, 1)
		if err == nil || !strings.Contains(err.Error(), "retained source excluded by source_admission_v1") {
			t.Fatalf("expected policy exclusion, got %+v %v", result, err)
		}
		if strings.Contains(err.Error(), "Ab12") || result.RecommendationRef != "" || db.transactions != 0 || db.reads != 2 {
			t.Fatalf("credential leaked or recommendation persisted: reads=%d transactions=%d", db.reads, db.transactions)
		}
	}
	format, _, err := admittedHandlerContent([]byte("# Credential handling\nOAuth client_secret and API_KEY values must be protected."))
	if err != nil || format != "text" {
		t.Fatalf("ordinary documentation rejected: %s %v", format, err)
	}
}
