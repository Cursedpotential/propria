package postgres

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestOperationPinsSurviveLaterHumanPublicationFailure(t *testing.T) {
	source, raw, normalized := uuid.New(), uuid.New(), uuid.New()
	result := operationResultPins(previewmodel.Binding{SourceVersionID: source}, []previewmodel.OperationStage{
		{Stage: "persist_raw_generation_activity", Status: "success", Ref: raw.String(), ReceiptRef: "raw-receipt"},
		{Stage: "persist_normalized_generation_activity", Status: "success", Ref: normalized.String(), ReceiptRef: "normalized-receipt"},
		{Stage: "publish_context_search_activity", Status: "failed", Reason: "projection unavailable"},
	})
	if result.RawGenerationID != raw || result.NormalizedGenerationID != normalized || result.NormalizedReceiptRef != "normalized-receipt" || result.PreviewAvailable {
		t.Fatalf("saved artifacts were hidden or implied preview publication: %+v", result)
	}
}

func TestAIResultRouteNeverReadsPGChatGenerations(t *testing.T) {
	source, raw, normalized := uuid.New(), uuid.New(), uuid.New()
	sourceRef, format := "registered-source", "chatgpt_official_json"
	var absentUUID *uuid.UUID
	db := &operationResultTestDB{replies: [][]any{
		{"handle", "request", proffer.Ref("b2://synthetic/conversations.json"), "workflow", "run", proffer.Ref(""), proffer.Ref("options"), &source, &raw, &normalized, time.Now(), absentUUID, absentUUID, ""},
		{1, &sourceRef, true, &format},
	}}
	// The registration identity is a real UUID, while the stored source format independently identifies AI.
	sourceRef = source.String()
	result, err := (&ProfferPreviewStore{db: db}).ResolveOperationResults(context.Background(), "handle")
	if err != nil || result.SourceVersionID != source || result.Status != "not_queried" || result.ContentStore != "weaviate" || result.SourceKind != "ai_chat" || result.RawGenerationID != uuid.Nil || result.NormalizedGenerationID != uuid.Nil {
		t.Fatalf("AI route=%+v %v", result, err)
	}
	if len(db.queries) != 2 {
		t.Fatalf("AI route performed extra generation/receipt reads: %d", len(db.queries))
	}
	for _, query := range db.queries {
		if strings.Contains(query, "FROM context.raw_generation") || strings.Contains(query, "FROM context.normalized_generation") {
			t.Fatal("AI result route required PG chat rows")
		}
	}
	for _, generic := range []string{"json", "message_export_json", "generic_message", "smsbackuprestore_xml"} {
		if _, ai := aiOperationResults(previewmodel.Binding{SourceVersionID: source}, generic); ai {
			t.Fatalf("generic format %q inferred AI", generic)
		}
	}
}

func TestOperationPinsRejectAmbiguousOrMismatchedGeneration(t *testing.T) {
	raw, normalized := uuid.New(), uuid.New()
	for _, existing := range []uuid.UUID{uuid.Nil, normalized} {
		result := operationResultPins(previewmodel.Binding{SourceVersionID: uuid.New(), RawGenerationID: raw, NormalizedGenerationID: existing}, []previewmodel.OperationStage{
			{Stage: "persist_normalized_generation_activity", Status: "success", Ref: normalized.String(), ReceiptRef: "one"},
			{Stage: "persist_normalized_generation_activity", Status: "success", Ref: uuid.NewString(), ReceiptRef: "two"},
		})
		if result.Status != "inconsistent" || !strings.Contains(result.Reason, "two") {
			t.Fatalf("ambiguous pins accepted: %+v", result)
		}
	}
	missing := operationResultPins(previewmodel.Binding{}, nil)
	if missing.Status != "missing" || missing.Reason == "" {
		t.Fatalf("missing result became empty/success: %+v", missing)
	}
}

type operationResultTestDB struct {
	DB
	replies [][]any
	queries []string
	args    [][]any
}

func (db *operationResultTestDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	if len(db.replies) == 0 {
		return operationResultTestRow{}
	}
	values := db.replies[0]
	db.replies = db.replies[1:]
	return operationResultTestRow{values: values}
}

type operationResultTestRow struct{ values []any }

func (row operationResultTestRow) Scan(dest ...any) error {
	if row.values == nil {
		return pgx.ErrNoRows
	}
	for i, value := range row.values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func TestExactResultLineageRejectsMismatchedSourceAndRawGeneration(t *testing.T) {
	source, raw, norm := uuid.New(), uuid.New(), uuid.New()
	for _, tc := range []struct {
		name    string
		replies [][]any
		want    string
	}{
		{"wrong request", [][]any{{false}}, "operation request"},
		{"wrong raw source", [][]any{{true}, {uuid.New()}}, "different source"},
		{"wrong normalized source", [][]any{{true}, {source}, {uuid.New(), raw}}, "source lineage"},
		{"wrong normalized raw", [][]any{{true}, {source}, {source, uuid.New()}}, "source lineage"},
		{"missing normalized", [][]any{{true}, {source}, nil}, "no matching generation"},
		{"valid persisted", [][]any{{true}, {source}, {source, raw}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &operationResultTestDB{replies: tc.replies}
			store := &ProfferPreviewStore{db: db}
			result, err := store.validateOperationResultLineage(context.Background(), previewmodel.OperationResults{SourceVersionID: source, RawGenerationID: raw, NormalizedGenerationID: norm}, "request-not-workflow", "workflow-not-request")
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if result.Status != "available" {
					t.Fatalf("valid result unavailable: %+v", result)
				}
			} else if result.Status != "inconsistent" || !strings.Contains(result.Reason, tc.want) {
				t.Fatalf("bad lineage accepted: %+v", result)
			}
			if db.args[0][1] != "request-not-workflow" {
				t.Fatal("source was resolved using an assumed Temporal ID")
			}
			for _, query := range db.queries {
				if strings.Contains(query, "ORDER BY") {
					t.Fatal("result reader selected a latest generation")
				}
			}
		})
	}
}

func TestFingerprintUsesExactSourceVersionAndSourceKind(t *testing.T) {
	source := uuid.New()
	computed := time.Now().UTC()
	digest := strings.Repeat("a", 64)
	retained := digest
	db := &operationResultTestDB{replies: [][]any{{"context_source_fingerprint", "sha256", digest, "context-source-fingerprint-v1", &computed, "fixture-hasher", "hash-receipt", "activity-receipt", &retained}}}
	result, err := (&ProfferPreviewStore{db: db}).SourceFingerprint(context.Background(), previewmodel.Binding{SourceVersionID: source, RequestID: "request-not-workflow"})
	if err != nil || result.Status != "available" || result.SourceVersionID != source || result.Digest != digest {
		t.Fatalf("fingerprint=%+v err=%v", result, err)
	}
	if db.args[0][0] != source || db.args[0][1] != "request-not-workflow" || !strings.Contains(db.queries[0], "hash.hash_kind='context_source_fingerprint'") {
		t.Fatal("fingerprint was not pinned to selected source bytes")
	}
	retained = strings.Repeat("b", 64)
	db.replies = [][]any{{"context_source_fingerprint", "sha256", digest, "context-source-fingerprint-v1", &computed, "fixture-hasher", "hash-receipt", "activity-receipt", &retained}}
	result, err = (&ProfferPreviewStore{db: db}).SourceFingerprint(context.Background(), previewmodel.Binding{SourceVersionID: source, RequestID: "request-not-workflow"})
	if err != nil || result.Status != "inconsistent" {
		t.Fatalf("conflicting original digest accepted: %+v %v", result, err)
	}
}
