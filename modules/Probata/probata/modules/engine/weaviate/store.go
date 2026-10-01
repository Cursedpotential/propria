// Package weaviate is the HTTP storage boundary for the pre-approval search
// surface. It owns every Weaviate call; engine/contextsearch owns the object
// shape and its identifiers, and activities/publish_context_search.go owns the
// Activity that drives both.
//
// It speaks Weaviate's REST API over net/http deliberately rather than
// vendoring the Go client: three endpoints (schema, batch objects, graphql)
// need nothing beyond encoding/json.
//
// Semantics verified live on 2026-09-26: POST /v1/objects with an existing id
// is HTTP 422 (not an upsert); POST /v1/batch/objects with an existing id
// REPLACES the object and the count does not grow. The batch endpoint is the
// only write path used here.
//
// The target is the owner-approved MsgEvents20260918 (OD-06, 2026-10-01). This
// Store never creates, drops or alters an existing property of that
// collection. Its only schema action is to ADD a property the collection does
// not yet have (POST /v1/schema/{class}/properties), the same additive path the
// Case Bible writer uses (elt_run.py ensure_schema).
//
// Byline: Claude Code · Opus 5 · 2026-09-26
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (MsgEvents20260918 mapping, named
// vector, additive-only schema, no collection creation)
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

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/contextsearch"
)

// DefaultBatchSize bounds one batch request.
const DefaultBatchSize = 100

// Store publishes search objects to one existing Weaviate collection.
//
// Collection has NO default on purpose: vector-store collection names are the
// owner's to approve (standing rule, 2026-09-24).
type Store struct {
	// BaseURL is the instance root, for example http://100.91.190.107:8082.
	BaseURL string
	// Collection is the Weaviate class this Store writes.
	Collection string
	// VectorName is the named vector objects are written under (text_nim on
	// MsgEvents20260918). Required: the collection has no default vector.
	VectorName string
	// EmbedModel is recorded on every object as embed_model.
	EmbedModel string
	// APIKey is optional; when set it is sent as a bearer token and never
	// logged.
	APIKey string
	// HTTP is optional; a bounded default client is used when nil.
	HTTP *http.Client
	// BatchSize is optional; DefaultBatchSize is used when zero.
	BatchSize int
	// Now is injectable for tests; it stamps indexed_at.
	Now func() time.Time
}

func (s Store) validate() error {
	if strings.TrimSpace(s.BaseURL) == "" {
		return errors.New("weaviate store requires a base url")
	}
	if err := validateCollectionName(s.Collection); err != nil {
		return err
	}
	if strings.TrimSpace(s.VectorName) == "" {
		return errors.New("weaviate store requires the named vector to write (text_nim)")
	}
	return nil
}

func validateCollectionName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("weaviate store requires an explicit collection name; vector-store collection names are owner-approved and have no default")
	}
	if first := trimmed[0]; first < 'A' || first > 'Z' {
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

func (s Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
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

// Property describes one collection property.
type Property struct {
	Name         string   `json:"name"`
	DataType     []string `json:"dataType"`
	Tokenization string   `json:"tokenization,omitempty"`
}

func exact(name string) Property {
	return Property{Name: name, DataType: []string{"text"}, Tokenization: "field"}
}

func word(name string) Property {
	return Property{Name: name, DataType: []string{"text"}, Tokenization: "word"}
}

// ExistingProperties are the MsgEvents20260918 properties this Store writes
// that the collection already had (read live 2026-10-01). They are reused,
// never redefined; EnsureCollection fails closed if one is missing or carries
// a different data type, because that would mean the target is not the
// collection the owner approved.
func ExistingProperties() []Property {
	return []Property{
		word("dedup_key"), word("body"), word("search_text"), word("sender"),
		{Name: "participants", DataType: []string{"text[]"}}, {Name: "recipients", DataType: []string{"text[]"}},
		word("contact_name"), word("source_format"), word("event_kind"), word("record_kind"),
		word("direction"), word("ts_original"), word("extractor"), word("ingest_run_id"),
		word("embed_model"), {Name: "provenance", DataType: []string{"text[]"}},
		{Name: "sort_ts", DataType: []string{"date"}}, {Name: "event_ts", DataType: []string{"date"}},
		{Name: "indexed_at", DataType: []string{"date"}}, {Name: "n_sources", DataType: []string{"int"}},
	}
}

// AddedProperties are the Probata properties this Store adds to the
// collection, additively, the first time it runs. Identifiers and digests use
// field tokenization so a filter matches the whole value.
func AddedProperties() []Property {
	return []Property{
		exact("origin_system"),
		exact("dedup_key_construction"),
		exact("object_id_construction"),
		exact("pg_schema"),
		exact("pg_table"),
		exact("pg_row_id"),
		exact("source_version_id"),
		exact("original_object_id"),
		exact("normalized_generation_id"),
		exact("matter_id"),
		exact("source_object_sha256"),
		exact("content_sha256"),
		exact("parser_id"),
		exact("parser_version"),
		exact("normalizer_id"),
		exact("normalizer_version"),
		exact("extraction_attempt_ref"),
		{Name: "attempt", DataType: []string{"int"}},
		{Name: "occurred_at", DataType: []string{"date"}},
		{Name: "knowledge_time", DataType: []string{"date"}},
		exact("disclosure_tier"),
		exact("disclosure_tier_basis"),
		exact("provenance_class"),
		exact("timestamp_certainty"),
		exact("timestamp_granularity"),
	}
}

type schemaClass struct {
	Class        string                     `json:"class"`
	Properties   []Property                 `json:"properties"`
	VectorConfig map[string]json.RawMessage `json:"vectorConfig"`
}

// EnsureCollection checks the collection exists with the expected properties
// and named vector, and adds any AddedProperties it lacks. It returns the
// names it added. It never creates the collection: an absent collection is an
// error, because creating one would invent a collection the owner never
// approved.
func (s Store) EnsureCollection(ctx context.Context) ([]string, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	status, payload, err := s.do(ctx, http.MethodGet, "/v1/schema/"+s.Collection, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, fmt.Errorf("weaviate collection %s does not exist; refusing to create it (collection names are owner-approved)", s.Collection)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("inspect weaviate collection %s returned HTTP %d: %s", s.Collection, status, truncate(payload))
	}
	var class schemaClass
	if err := json.Unmarshal(payload, &class); err != nil {
		return nil, fmt.Errorf("decode weaviate collection %s schema: %w", s.Collection, err)
	}
	if _, ok := class.VectorConfig[s.VectorName]; !ok {
		return nil, fmt.Errorf("weaviate collection %s has no named vector %q", s.Collection, s.VectorName)
	}
	have := make(map[string]Property, len(class.Properties))
	for _, p := range class.Properties {
		have[p.Name] = p
	}
	for _, want := range ExistingProperties() {
		got, ok := have[want.Name]
		if !ok {
			return nil, fmt.Errorf("weaviate collection %s lacks expected property %s; it is not the collection this writer was mapped onto", s.Collection, want.Name)
		}
		if !sameDataType(got.DataType, want.DataType) {
			return nil, fmt.Errorf("weaviate collection %s property %s is %v, want %v", s.Collection, want.Name, got.DataType, want.DataType)
		}
	}
	var added []string
	for _, want := range AddedProperties() {
		if got, ok := have[want.Name]; ok {
			if !sameDataType(got.DataType, want.DataType) {
				return added, fmt.Errorf("weaviate collection %s property %s is %v, want %v; refusing to write into a conflicting property", s.Collection, want.Name, got.DataType, want.DataType)
			}
			continue
		}
		status, payload, err := s.do(ctx, http.MethodPost, "/v1/schema/"+s.Collection+"/properties", want)
		if err != nil {
			return added, err
		}
		if status != http.StatusOK {
			return added, fmt.Errorf("add property %s to weaviate collection %s returned HTTP %d: %s", want.Name, s.Collection, status, truncate(payload))
		}
		added = append(added, want.Name)
	}
	return added, nil
}

func sameDataType(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PublishOutcome is the compact result of one publish call.
type PublishOutcome struct {
	Requested int
	Written   int
	ObjectIDs []string
}

// PublishObjects writes every object under its deterministic id. It is
// idempotent (the batch endpoint replaces an existing id) and fail-closed
// (any per-object error fails the call).
func (s Store) PublishObjects(ctx context.Context, objects []contextsearch.Object) (PublishOutcome, error) {
	outcome := PublishOutcome{Requested: len(objects)}
	if err := s.validate(); err != nil {
		return outcome, err
	}
	if len(objects) == 0 {
		return outcome, errors.New("publish search objects requires at least one object")
	}
	indexedAt := s.now().UTC().Format(time.RFC3339Nano)
	encoded := make([]map[string]any, 0, len(objects))
	outcome.ObjectIDs = make([]string, 0, len(objects))
	for index, object := range objects {
		id, err := object.ObjectID()
		if err != nil {
			return outcome, fmt.Errorf("search object %d: %w", index, err)
		}
		if len(object.Vector) == 0 {
			return outcome, fmt.Errorf("search object %d has no %s vector; the collection is searched by that vector", index, s.VectorName)
		}
		encoded = append(encoded, map[string]any{
			"class":      s.Collection,
			"id":         id.String(),
			"properties": s.objectProperties(object, indexedAt),
			"vectors":    map[string]any{s.VectorName: object.Vector},
		})
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
		return outcome, fmt.Errorf("weaviate accepted %d of %d search objects", outcome.Written, outcome.Requested)
	}
	return outcome, nil
}

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
		if result.Result.Errors != nil && len(result.Result.Errors.Error) > 0 {
			return written, fmt.Errorf("weaviate rejected search object %s: %s", result.ID, result.Result.Errors.Error[0].Message)
		}
		if result.Result.Status != "" && !strings.EqualFold(result.Result.Status, "SUCCESS") {
			return written, fmt.Errorf("weaviate rejected search object %s: %s", result.ID, result.Result.Status)
		}
		written++
	}
	return written, nil
}

// objectProperties maps one object onto MsgEvents20260918. Existing
// properties keep the meaning the Case Bible writer gave them; vault_key,
// sha1 and catalog_path are deliberately left unset, because the Case Bible's
// publish_bundle.py retires objects by vault_key and must never match a
// Probata object.
func (s Store) objectProperties(object contextsearch.Object, indexedAt string) map[string]any {
	properties := map[string]any{
		// existing
		"dedup_key":     object.DedupKey,
		"search_text":   object.SearchText,
		"source_format": object.Provenance.SourceFormat,
		"event_kind":    object.RecordKind,
		"record_kind":   object.RecordKind,
		"extractor":     object.Provenance.ParserID + "@" + object.Provenance.ParserVersion,
		"ingest_run_id": object.Provenance.RequestID,
		"indexed_at":    indexedAt,
		"n_sources":     1,
		"provenance": []string{fmt.Sprintf("%s|%s|%s|%s.%s/%s",
			contextsearch.OriginSystem, object.Coordinates.SourceVersionID, object.Provenance.ParserID,
			object.Coordinates.Schema, object.Coordinates.Table, object.Coordinates.RowID)},

		// added
		"origin_system":            contextsearch.OriginSystem,
		"dedup_key_construction":   contextsearch.DedupKeyConstruction,
		"object_id_construction":   contextsearch.ObjectIDConstruction,
		"pg_schema":                object.Coordinates.Schema,
		"pg_table":                 object.Coordinates.Table,
		"pg_row_id":                object.Coordinates.RowID.String(),
		"source_version_id":        object.Coordinates.SourceVersionID.String(),
		"normalized_generation_id": object.Coordinates.NormalizedGenerationID.String(),
		"source_object_sha256":     hex.EncodeToString(object.Provenance.SourceObjectSHA256),
		"content_sha256":           hex.EncodeToString(object.ContentSHA256),
		"parser_id":                object.Provenance.ParserID,
		"parser_version":           object.Provenance.ParserVersion,
		"normalizer_id":            object.Provenance.NormalizerID,
		"normalizer_version":       object.Provenance.NormalizerVersion,
		"extraction_attempt_ref":   object.Provenance.ExtractionAttemptRef,
		"attempt":                  object.Provenance.Attempt,
		"knowledge_time":           object.Temporal.KnowledgeTime.UTC().Format(time.RFC3339Nano),
		"disclosure_tier":          object.Temporal.DisclosureTier,
		"disclosure_tier_basis":    object.Temporal.DisclosureTierBasis,
	}
	setText := func(name, value string) {
		if strings.TrimSpace(value) != "" {
			properties[name] = value
		}
	}
	setList := func(name string, values []string) {
		kept := make([]string, 0, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				kept = append(kept, value)
			}
		}
		if len(kept) > 0 {
			properties[name] = kept
		}
	}
	setText("embed_model", s.EmbedModel)
	setText("body", object.Body)
	setText("sender", object.People.Sender)
	setText("direction", object.Direction)
	setText("ts_original", object.Temporal.OccurredAtRaw)
	setText("contact_name", strings.Join(object.People.ContactNames, "; "))
	setText("provenance_class", object.ProvenanceClass)
	setText("timestamp_certainty", object.Temporal.TimestampCertainty)
	setText("timestamp_granularity", object.Temporal.TimestampGranularity)
	setList("participants", object.People.Participants)
	setList("recipients", object.People.Recipients)
	if object.Coordinates.OriginalObjectID != uuid.Nil {
		properties["original_object_id"] = object.Coordinates.OriginalObjectID.String()
	}
	if object.Coordinates.MatterID != uuid.Nil {
		properties["matter_id"] = object.Coordinates.MatterID.String()
	}
	if object.Temporal.OccurredAt != nil {
		stamp := object.Temporal.OccurredAt.UTC().Format(time.RFC3339Nano)
		properties["occurred_at"] = stamp
		properties["sort_ts"] = stamp
		properties["event_ts"] = stamp
	}
	return properties
}

// StoredObject is one object read back from Weaviate, for verification.
type StoredObject struct {
	ID         string               `json:"id"`
	Class      string               `json:"class"`
	Properties map[string]any       `json:"properties"`
	Vectors    map[string][]float32 `json:"vectors"`
}

// GetObject reads one object back by id; absent is (nil, nil).
func (s Store) GetObject(ctx context.Context, id string) (*StoredObject, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("read search object requires an id")
	}
	status, payload, err := s.do(ctx, http.MethodGet, "/v1/objects/"+s.Collection+"/"+id+"?include=vector", nil)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		var stored StoredObject
		if err := json.Unmarshal(payload, &stored); err != nil {
			return nil, fmt.Errorf("decode search object %s: %w", id, err)
		}
		return &stored, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("read search object %s returned HTTP %d: %s", id, status, truncate(payload))
	}
}

// CountRunObjects counts this collection's objects written by one Probata
// run, with a Weaviate where filter (a dict filter, never a FilterExpr).
func (s Store) CountRunObjects(ctx context.Context, requestID string) (int64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	where := fmt.Sprintf(`{operator:And,operands:[{path:["origin_system"],operator:Equal,valueText:%q},{path:["ingest_run_id"],operator:Equal,valueText:%q}]}`,
		contextsearch.OriginSystem, requestID)
	query := fmt.Sprintf("{Aggregate{%s(where:%s){meta{count}}}}", s.Collection, where)
	status, payload, err := s.do(ctx, http.MethodPost, "/v1/graphql", map[string]any{"query": query})
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, fmt.Errorf("count search objects returned HTTP %d: %s", status, truncate(payload))
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
		return 0, fmt.Errorf("decode search object count: %w", err)
	}
	if len(decoded.Errors) > 0 {
		return 0, fmt.Errorf("count search objects: %s", decoded.Errors[0].Message)
	}
	buckets := decoded.Data.Aggregate[s.Collection]
	if len(buckets) == 0 {
		return 0, fmt.Errorf("count search objects: collection %s absent from the aggregate response", s.Collection)
	}
	return buckets[0].Meta.Count, nil
}

func truncate(payload []byte) string {
	const limit = 512
	trimmed := strings.TrimSpace(string(payload))
	if len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "..."
}
