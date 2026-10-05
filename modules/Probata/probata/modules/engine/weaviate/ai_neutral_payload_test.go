// Byline: Codex · GPT-6-Sol · 2026-10-05.
package weaviate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/google/uuid"
)

// neutralWireObject supplies a complete source-linked AI object with genuine fixture dates and role labels.
// Inputs: none. Outputs: synthetic AI search object. Effects: none. Choose for actual HTTP serialization/readback tests.
func neutralWireObject() contextsearch.Object {
	at := time.Date(2020, 2, 1, 4, 5, 0, 0, time.UTC)
	return contextsearch.Object{DedupKey: "synthetic-neutral-ai", Coordinates: contextsearch.Coordinates{Schema: "context", Table: "normalized_record_identity", RowID: uuid.New(), SourceVersionID: uuid.New(), NormalizedGenerationID: uuid.New(), OriginalObjectID: uuid.New()}, Provenance: contextsearch.Provenance{SourceObjectSHA256: bytes.Repeat([]byte{3}, 32), SourceFormat: "json", RawFormatID: "chatgpt_official_json", ParserID: "p", ParserVersion: "1", NormalizerID: "n", NormalizerVersion: "1", RequestID: "synthetic-run", Attempt: 1, ExtractionAttemptRef: "synthetic-attempt"}, Temporal: contextsearch.Temporal{OccurredAt: &at, OccurredAtRaw: at.Format(time.RFC3339), KnowledgeTime: at.Add(time.Hour), TimestampCertainty: "exact", TimestampGranularity: "second"}, People: contextsearch.People{Sender: "assistant", Participants: []string{"assistant", "user"}, RoleLabels: []string{"assistant", "user"}}, RecordKind: contextsearch.RecordKindAIChat, Body: "synthetic AI body", SearchText: "synthetic AI body", ContentSHA256: bytes.Repeat([]byte{4}, 32), Vector: []float32{.1, .2}}
}

// TestAINeutralWirePayloadSchemaAndReadback exercises schema validation and REST publication/readback without live writes.
// Inputs: synthetic object and an in-memory Weaviate-shaped HTTP server. Outputs: exact payload assertions.
// Effects: transient test memory only. Choose instead of writing synthetic objects into production collections.
func TestAINeutralWirePayloadSchemaAndReadback(t *testing.T) {
	var stored StoredObject
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/schema/AiChatEvents20260918":
			_ = json.NewEncoder(w).Encode(schemaClass{Class: "AiChatEvents20260918", Properties: append(ExistingProperties(), AddedProperties()...), VectorConfig: map[string]json.RawMessage{"text_nim": json.RawMessage(`{}`)}})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/batch/objects":
			var batch struct {
				Objects []StoredObject `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if len(batch.Objects) != 1 {
				t.Errorf("batch objects=%d", len(batch.Objects))
				w.WriteHeader(400)
				return
			}
			stored = batch.Objects[0]
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": stored.ID, "result": map[string]any{"status": "SUCCESS"}}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/objects/AiChatEvents20260918/"+stored.ID:
			_ = json.NewEncoder(w).Encode(stored)
		default:
			t.Errorf("unexpected schema mutation/request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	s := Store{BaseURL: server.URL, Collection: "AiChatEvents20260918", VectorName: "text_nim", EmbedModel: "fixture-embed"}
	if added, err := s.EnsureCollection(context.Background()); err != nil || len(added) != 0 {
		t.Fatalf("schema=%v error=%v", added, err)
	}
	o := neutralWireObject()
	outcome, err := s.PublishObjects(context.Background(), []contextsearch.Object{o})
	if err != nil || outcome.Written != 1 {
		t.Fatalf("publication=%+v error=%v", outcome, err)
	}
	back, err := s.GetObject(context.Background(), outcome.ObjectIDs[0])
	if err != nil || back == nil {
		t.Fatalf("readback=%+v error=%v", back, err)
	}
	for _, absent := range []string{"disclosure_tier", "disclosure_tier_basis", "owner_person_id", "perspective_person_id", "participant_entity_ids"} {
		if _, ok := back.Properties[absent]; ok {
			t.Errorf("AI payload has human property %s", absent)
		}
	}
	for name, want := range map[string]string{"raw_format_id": "chatgpt_official_json", "source_format": "json", "record_kind": "ai_chat", "occurred_at": o.Temporal.OccurredAt.Format(time.RFC3339Nano), "knowledge_time": o.Temporal.KnowledgeTime.Format(time.RFC3339Nano), "timestamp_certainty": "exact", "timestamp_granularity": "second", "source_version_id": o.Coordinates.SourceVersionID.String(), "normalized_generation_id": o.Coordinates.NormalizedGenerationID.String(), "pg_row_id": o.Coordinates.RowID.String(), "body": o.Body} {
		if back.Properties[name] != want {
			t.Errorf("%s=%v want %q", name, back.Properties[name], want)
		}
	}
	if !reflect.DeepEqual(back.Properties["participant_roles"], []any{"assistant", "user"}) || !reflect.DeepEqual(back.Properties["participants"], []any{"assistant", "user"}) {
		t.Fatalf("roles changed: %+v", back.Properties)
	}
}

// TestAINeutralValidationCannotWaiveHumanTierOrSourceProof checks the narrowly scoped domain validation exception.
// Inputs: valid AI object mutated across kind, provenance and temporal boundaries. Outputs: assertions. Effects: none.
// Choose as the strict-human/source-proof regression counterpart of neutral AI serialization.
func TestAINeutralValidationCannotWaiveHumanTierOrSourceProof(t *testing.T) {
	for _, mutate := range []func(*contextsearch.Object){
		func(o *contextsearch.Object) { o.RecordKind = contextsearch.RecordKindMessage },
		func(o *contextsearch.Object) { o.Provenance.RawFormatID = "smsbackuprestore_xml" },
		func(o *contextsearch.Object) { o.Temporal.DisclosureTier = contextsearch.TierDiscovered },
		func(o *contextsearch.Object) { o.Temporal.DisclosureTierBasis = "invented" },
		func(o *contextsearch.Object) { o.Temporal.KnowledgeTime = time.Time{} },
		func(o *contextsearch.Object) { zero := time.Time{}; o.Temporal.OccurredAt = &zero },
	} {
		o := neutralWireObject()
		mutate(&o)
		if err := o.Validate(); err == nil {
			t.Fatalf("invalid object passed: %+v", o)
		}
	}
	o := neutralWireObject()
	o.RecordKind = contextsearch.RecordKindMessage
	o.Provenance.SourceFormat = "smsbackuprestore_xml"
	o.Provenance.RawFormatID = "smsbackuprestore_xml"
	o.Temporal.DisclosureTier = contextsearch.TierContemporaneous
	o.Temporal.DisclosureTierBasis = "owner_participant:v1"
	if err := o.Validate(); err != nil {
		t.Fatalf("strict valid human object failed: %v", err)
	}
	props := (Store{}).objectProperties(o, "fixture")
	if props["disclosure_tier"] != contextsearch.TierContemporaneous || props["disclosure_tier_basis"] != "owner_participant:v1" {
		t.Fatalf("human payload lost tier: %v", props)
	}
}
