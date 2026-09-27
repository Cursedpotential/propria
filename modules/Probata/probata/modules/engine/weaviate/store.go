// Package weaviate is the HTTP storage boundary for the pre-approval context
// search surface. It owns every Weaviate call; engine/contextsearch owns the
// object shape and its identifiers, and activities/publish_context_search.go
// owns the Activity that drives both. This split is the one already
// established by activities + engine/postgres: the Activity computes and
// validates, the Store performs I/O.
//
// It speaks Weaviate's REST API over net/http deliberately, rather than
// vendoring github.com/weaviate/weaviate-go-client. engine/ is a vendored
// module and the client would pull a large transitive tree in for three
// endpoints (schema, batch objects, graphql). Nothing here needs more than
// encoding/json.
//
// Semantics below were verified live against the deployed instance on
// 2026-09-26, not assumed from documentation:
//
//   - POST /v1/objects with an id that already exists returns HTTP 422
//     "id ... already exists" -- it is NOT an upsert.
//   - POST /v1/batch/objects with an id that already exists returns per-object
//     status SUCCESS and REPLACES the object. The collection count does not
//     grow.
//
// The batch endpoint is therefore the only write path used here: it is both
// the idempotent one and the batched one.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
package weaviate

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/contextsearch"
)

// DefaultBatchSize bounds one batch request. It keeps a single request well
// inside Weaviate's gRPC/HTTP message ceiling while still amortizing the round
// trip.
const DefaultBatchSize = 100

// Store publishes context search objects to one Weaviate collection.
//
// Collection has NO default on purpose. Vector-store collection names are the
// owner's to approve (standing rule, 2026-09-24), so a caller must pass one
// explicitly; an empty Collection fails closed rather than inventing a
// production name.
type Store struct {
	// BaseURL is the instance root, for example http://host:8082.
	BaseURL string
	// Collection is the Weaviate class this Store reads and writes.
	Collection string
	// APIKey is optional. When set it is sent as a bearer token. It is never
	// logged and never appears in an error message.
	APIKey string
	// HTTP is optional; a bounded default client is used when nil.
	HTTP *http.Client
	// BatchSize is optional; DefaultBatchSize is used when zero.
	BatchSize int
}

func (s Store) validate() error {
	if strings.TrimSpace(s.BaseURL) == "" {
		return errors.New("weaviate store requires a base url")
	}
	if err := validateCollectionName(s.Collection); err != nil {
		return err
	}
	return nil
}

// validateCollectionName enforces Weaviate's class-name rule (it must begin
// with an upper-case letter) before a request is built, so a bad name fails
// with a clear message here instead of as an opaque 422.
func validateCollectionName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("weaviate store requires an explicit collection name; vector-store collection names are owner-approved and have no default")
	}
	first := trimmed[0]
	if first < 'A' || first > 'Z' {
		return fmt.Errorf("weaviate collection %q must begin with an upper-case letter", trimmed)
	}
	return nil
}

func (s Store) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

func (s Store) batchSize() int {
	if s.BatchSize > 0 {
		return s.BatchSize
	}
	return DefaultBatchSize
}

func (s Store) do(ctx context.Context, method, path string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("encode %s %s request: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.BaseURL, "/")+path, reader)
	if err != nil {
		return 0, nil, fmt.Errorf("build %s %s request: %w", method, path, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(s.APIKey) != "" {
		request.Header.Set("Authorization", "Bearer "+s.APIKey)
	}
	response, err := s.client().Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, fmt.Errorf("read %s %s response: %w", method, path, err)
	}
	return response.StatusCode, payload, nil
}

// Ready reports whether the instance is serving.
func (s Store) Ready(ctx context.Context) error {
	if strings.TrimSpace(s.BaseURL) == "" {
		return errors.New("weaviate store requires a base url")
	}
	status, _, err := s.do(ctx, http.MethodGet, "/v1/.well-known/ready", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("weaviate readiness returned HTTP %d", status)
	}
	return nil
}

// property describes one collection property.
type property struct {
	Name         string   `json:"name"`
	DataType     []string `json:"dataType"`
	Tokenization string   `json:"tokenization,omitempty"`
}

// exact marks an identifier or digest property that must match whole-value
// rather than being split into words, so a filter on a uuid or a hex digest
// behaves the way a reader expects.
func exact(name string) property {
	return property{Name: name, DataType: []string{"text"}, Tokenization: "field"}
}

func text(name string) property {
	return property{Name: name, DataType: []string{"text"}}
}

// collectionProperties is the exact schema of the pre-approval search object.
// It mirrors contextsearch.Object field for field. There is deliberately no
// content/body property: D-149 item 8 keeps the text in PostgreSQL.
func collectionProperties() []property {
	return []property{
		exact("dedup_key"),
		exact("dedup_key_construction"),
		exact("object_id_construction"),

		exact("pg_schema"),
		exact("pg_table"),
		exact("pg_row_id"),
		exact("source_version_id"),
		exact("artifact_id"),
		exact("normalized_generation_id"),
		{Name: "member_ids", DataType: []string{"text[]"}, Tokenization: "field"},

		exact("source_object_sha256"),
		exact("content_sha256"),
		exact("template_id"),
		exact("template_version"),
		exact("parser_id"),
		exact("parser_version"),
		exact("request_id"),
		exact("extraction_attempt_ref"),
		{Name: "attempt", DataType: []string{"int"}},

		text("record_type"),
		text("source"),
		exact("conversation_id"),

		{Name: "occurred_at", DataType: []string{"date"}},
		{Name: "knowledge_time", DataType: []string{"date"}},
		exact("disclosure_tier"),
	}
}

// EnsureCollection creates the collection when it is absent and otherwise
// leaves the existing one exactly as it is. It never mutates or replaces a
// collection that already exists, so pointing a Store at a populated
// collection cannot destroy it.
//
// The collection is created with vectorizer "none": vectors are supplied by
// the caller, so Weaviate never needs the text and never calls out to an
// embedding provider on its own.
func (s Store) EnsureCollection(ctx context.Context) (created bool, err error) {
	if err := s.validate(); err != nil {
		return false, err
	}
	status, _, err := s.do(ctx, http.MethodGet, "/v1/schema/"+s.Collection, nil)
	if err != nil {
		return false, err
	}
	switch status {
	case http.StatusOK:
		return false, nil
	case http.StatusNotFound:
		// fall through to create
	default:
		return false, fmt.Errorf("inspect weaviate collection %s returned HTTP %d", s.Collection, status)
	}
	body := map[string]any{
		"class":       s.Collection,
		"description": "Pre-approval context search surface: extraction output made searchable before any canonical PostgreSQL commit. Vector, PostgreSQL coordinates, member ids and provenance only -- canonical text stays in PostgreSQL.",
		"vectorizer":  "none",
		"properties":  collectionProperties(),
	}
	status, payload, err := s.do(ctx, http.MethodPost, "/v1/schema", body)
	if err != nil {
		return false, err
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("create weaviate collection %s returned HTTP %d: %s", s.Collection, status, truncate(payload))
	}
	return true, nil
}

// PublishOutcome is the compact durable result of one publish pass.
type PublishOutcome struct {
	// Requested is how many objects the caller handed over; Written is how
	// many Weaviate reported SUCCESS for. Fail-closed means these are equal
	// on a nil error.
	Requested int
	Written   int
	// ObjectIDs are the deterministic ids, in the order supplied.
	ObjectIDs []string
}

// PublishObjects writes every object under its deterministic id.
//
// It is idempotent by construction: the id comes from the dedup key, and the
// batch endpoint replaces an existing id rather than adding a row, so
// re-publishing the same set leaves the collection count unchanged.
//
// It is fail-closed: any per-object error fails the whole call rather than
// reporting a partial success, because a silently missing object is a message
// the owner would never see in search.
func (s Store) PublishObjects(ctx context.Context, objects []contextsearch.Object) (PublishOutcome, error) {
	outcome := PublishOutcome{Requested: len(objects)}
	if err := s.validate(); err != nil {
		return outcome, err
	}
	if len(objects) == 0 {
		return outcome, errors.New("publish context search objects requires at least one object")
	}

	encoded := make([]map[string]any, 0, len(objects))
	outcome.ObjectIDs = make([]string, 0, len(objects))
	for index, object := range objects {
		id, err := object.ObjectID()
		if err != nil {
			return outcome, fmt.Errorf("context search object %d: %w", index, err)
		}
		record := map[string]any{
			"class":      s.Collection,
			"id":         id.String(),
			"properties": objectProperties(object),
		}
		if len(object.Vector) > 0 {
			record["vector"] = object.Vector
		}
		encoded = append(encoded, record)
		outcome.ObjectIDs = append(outcome.ObjectIDs, id.String())
	}

	for start := 0; start < len(encoded); start += s.batchSize() {
		end := min(start+s.batchSize(), len(encoded))
		written, err := s.publishBatch(ctx, encoded[start:end])
		outcome.Written += written
		if err != nil {
			return outcome, err
		}
	}
	if outcome.Written != outcome.Requested {
		return outcome, fmt.Errorf("weaviate accepted %d of %d context search objects", outcome.Written, outcome.Requested)
	}
	return outcome, nil
}

// batchResult is the per-object status Weaviate returns for a batch write.
type batchResult struct {
	ID     string `json:"id"`
	Result struct {
		Status string `json:"status"`
		Errors *struct {
			Error []struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"errors"`
	} `json:"result"`
}

func (s Store) publishBatch(ctx context.Context, records []map[string]any) (int, error) {
	status, payload, err := s.do(ctx, http.MethodPost, "/v1/batch/objects", map[string]any{"objects": records})
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("weaviate batch write returned HTTP %d: %s", status, truncate(payload))
	}
	var results []batchResult
	if err := json.Unmarshal(payload, &results); err != nil {
		return 0, fmt.Errorf("decode weaviate batch response: %w", err)
	}
	if len(results) != len(records) {
		return 0, fmt.Errorf("weaviate batch response covered %d of %d objects", len(results), len(records))
	}
	written := 0
	for _, result := range results {
		if !strings.EqualFold(result.Result.Status, "SUCCESS") {
			message := result.Result.Status
			if result.Result.Errors != nil && len(result.Result.Errors.Error) > 0 {
				message = result.Result.Errors.Error[0].Message
			}
			return written, fmt.Errorf("weaviate rejected context search object %s: %s", result.ID, message)
		}
		written++
	}
	return written, nil
}

// objectProperties flattens one object into Weaviate properties. occurred_at
// is omitted entirely when absent rather than sent as a zero time, so a
// record whose as-lived timestamp could not be recovered reads as null
// instead of as year 1.
func objectProperties(object contextsearch.Object) map[string]any {
	members := make([]string, 0, len(object.MemberIDs))
	for _, member := range object.MemberIDs {
		members = append(members, member.String())
	}
	properties := map[string]any{
		"dedup_key":              object.DedupKey,
		"dedup_key_construction": contextsearch.DedupKeyConstruction,
		"object_id_construction": contextsearch.ObjectIDConstruction,

		"pg_schema":                object.Coordinates.Schema,
		"pg_table":                 object.Coordinates.Table,
		"pg_row_id":                object.Coordinates.RowID.String(),
		"source_version_id":        object.Coordinates.SourceVersionID.String(),
		"artifact_id":              object.Coordinates.ArtifactID.String(),
		"normalized_generation_id": object.Coordinates.NormalizedGenerationID.String(),
		"member_ids":               members,

		"source_object_sha256":   hex.EncodeToString(object.Provenance.SourceObjectSHA256),
		"content_sha256":         hex.EncodeToString(object.ContentSHA256),
		"template_id":            object.Provenance.TemplateID,
		"template_version":       object.Provenance.TemplateVersion,
		"parser_id":              object.Provenance.ParserID,
		"parser_version":         object.Provenance.ParserVersion,
		"request_id":             object.Provenance.RequestID,
		"extraction_attempt_ref": object.Provenance.ExtractionAttemptRef,
		"attempt":                object.Provenance.Attempt,

		"record_type":     object.RecordType,
		"source":          object.Source,
		"conversation_id": object.ConversationID,

		"knowledge_time":  object.Temporal.KnowledgeTime.UTC().Format(time.RFC3339Nano),
		"disclosure_tier": object.Temporal.DisclosureTier,
	}
	if object.Temporal.OccurredAt != nil {
		properties["occurred_at"] = object.Temporal.OccurredAt.UTC().Format(time.RFC3339Nano)
	}
	return properties
}

// StoredObject is one object read back from Weaviate, for verification.
type StoredObject struct {
	ID         string         `json:"id"`
	Class      string         `json:"class"`
	Properties map[string]any `json:"properties"`
	Vector     []float32      `json:"vector"`
}

// GetObject reads one object back by id. A missing object is reported as a
// nil StoredObject with a nil error so a caller can distinguish "absent" from
// "the read failed".
func (s Store) GetObject(ctx context.Context, id string) (*StoredObject, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("read context search object requires an id")
	}
	status, payload, err := s.do(ctx, http.MethodGet, "/v1/objects/"+s.Collection+"/"+id+"?include=vector", nil)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		var stored StoredObject
		if err := json.Unmarshal(payload, &stored); err != nil {
			return nil, fmt.Errorf("decode context search object %s: %w", id, err)
		}
		return &stored, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("read context search object %s returned HTTP %d: %s", id, status, truncate(payload))
	}
}

// CountObjects returns the collection's object count. This is the measurement
// that proves idempotency: publishing the same set twice must leave it
// unchanged.
func (s Store) CountObjects(ctx context.Context) (int64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	query := fmt.Sprintf("{Aggregate{%s{meta{count}}}}", s.Collection)
	status, payload, err := s.do(ctx, http.MethodPost, "/v1/graphql", map[string]any{"query": query})
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("count context search objects returned HTTP %d: %s", status, truncate(payload))
	}
	var decoded struct {
		Data struct {
			Aggregate map[string][]struct {
				Meta struct {
					Count int64 `json:"count"`
				} `json:"meta"`
			} `json:"Aggregate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return 0, fmt.Errorf("decode context search count: %w", err)
	}
	if len(decoded.Errors) > 0 {
		return 0, fmt.Errorf("count context search objects: %s", decoded.Errors[0].Message)
	}
	buckets, ok := decoded.Data.Aggregate[s.Collection]
	if !ok || len(buckets) == 0 {
		return 0, fmt.Errorf("count context search objects: collection %s absent from the aggregate response", s.Collection)
	}
	return buckets[0].Meta.Count, nil
}

// DropCollection removes a collection outright. It exists so a live test can
// purge the collection it created -- test data must never become canonical.
//
// It is guarded: confirm must repeat the collection name exactly. That makes
// an accidental or mistyped call impossible to execute against a production
// collection, which is the failure this guard is here to prevent. A caller
// that wants to remove owner data does not belong in this Store at all.
func (s Store) DropCollection(ctx context.Context, confirm string) error {
	if err := s.validate(); err != nil {
		return err
	}
	if confirm != s.Collection {
		return fmt.Errorf("refusing to drop weaviate collection %s: confirmation did not repeat the collection name exactly", s.Collection)
	}
	status, payload, err := s.do(ctx, http.MethodDelete, "/v1/schema/"+s.Collection, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("drop weaviate collection %s returned HTTP %d: %s", s.Collection, status, truncate(payload))
	}
	return nil
}

// truncate bounds an error body so a large Weaviate payload cannot flood a
// Temporal failure message.
func truncate(payload []byte) string {
	const limit = 512
	trimmed := strings.TrimSpace(string(payload))
	if len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "..."
}
