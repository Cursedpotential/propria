package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Cursedpotential/probata/engine/discovery"
	"time"
)

// DiscoverySourceReader reads bounded canonical normalized inputs; optional Surreal chooses a validated source-linked subset.
type DiscoverySourceReader struct {
	db       DB
	datasets []discovery.Dataset
	surreal  discovery.SurrealReader
}

// NewDiscoverySourceReader binds the existing DB and server definitions without querying or mutating sources.
func NewDiscoverySourceReader(db DB, ds []discovery.Dataset, surreal discovery.SurrealReader) (*DiscoverySourceReader, error) {
	if db == nil || discovery.ValidateDatasets(ds) != nil {
		return nil, discovery.ErrInvalid
	}
	for _, d := range ds {
		if d.InputBackend == "surrealdb" && surreal == nil {
			return nil, discovery.ErrUnavailable
		}
	}
	return &DiscoverySourceReader{db, append([]discovery.Dataset{}, ds...), surreal}, nil
}

// DiscoverySourceSQL prefilters scope, selected generations, recorded clocks and participant/platform filters before LIMIT.
// Inputs: parameterized request. Outputs: read query. Effects: none; canonical bytes and custody tables are never read or changed.
const DiscoverySourceSQL = `SELECT jsonb_build_object(
 'record_id',n.id::text,'source_version_id',n.source_version_id::text,'generation_id',n.normalized_generation_id::text,
 'ordinal',n.record_ordinal,'record_type',n.record_type,'occurred_at',n.occurred_at,'record_revision',n.created_at,
 'explicit_episode_id',merged.thread_version_id,'group_revision',merged.group_revision,
 'source_available_from',COALESCE(NULLIF(n.normalized_payload->>'source_available_from','')::timestamptz,merged.source_available_from),
 'group_knowledge_available_from',merged.knowledge_available_from,'working_message_id',merged.message_id,'imported_message_id',merged.external_id,
 'payload',n.normalized_payload,'source_kind',s.provenance_class,'declared_format',v.declared_format,
 'source_ref',o.object_uri,'source_sha256',encode(o.content_sha256,'hex'),'source_object_id',o.id::text,
 'locators',COALESCE((SELECT jsonb_agg(jsonb_build_object('locator_id',l.id::text,'source_version_id',l.source_version_id::text,
 'coordinate_system',l.coordinate_system,'range_start',l.range_start,'range_end',l.range_end,'projection',l.locator_projection,
 'exact_slice_sha256',encode(l.exact_slice_sha256,'hex'),'verification_activity_receipt_id',l.verification_activity_receipt_id::text) ORDER BY l.id)
 FROM context.normalized_record_range_locator link JOIN context.source_range_locator l ON l.id=link.source_range_locator_id AND l.source_version_id=n.source_version_id
 WHERE link.normalized_record_id=n.id AND link.source_version_id=n.source_version_id),'[]'::jsonb))
 FROM context.normalized_record_identity n
 JOIN context.normalized_generation g ON g.id=n.normalized_generation_id AND g.source_version_id=n.source_version_id
 JOIN context.source_version v ON v.id=n.source_version_id
 JOIN context.source s ON s.id=v.source_id
 LEFT JOIN context.retained_object o ON o.id=v.original_object_id
 LEFT JOIN LATERAL (
 SELECT version.id::text AS thread_version_id,encode(version.assertion_digest,'hex') AS group_revision,
 membership.source_available_from,version.knowledge_available_from,message.id::text AS message_id,message.external_id
 FROM working.message message
 LEFT JOIN working.message_occurrence occurrence ON occurrence.normalized_record_id=n.id AND occurrence.source_version_id=n.source_version_id
 JOIN working.first_party_context_thread_message membership ON membership.message_id=message.id
 JOIN working.first_party_context_thread_version version ON version.id=membership.thread_version_id AND version.context_thread_id=membership.context_thread_id
 JOIN working.first_party_context_thread thread ON thread.context_thread_id=version.context_thread_id
 WHERE (message.id=COALESCE(occurrence.primary_record_id,n.id) OR message.derived_from_record_id=COALESCE(occurrence.primary_record_id,n.id))
 AND thread.matter_id=$1::uuid AND thread.court_case_id=$2::uuid
 AND NOT EXISTS(SELECT 1 FROM working.first_party_context_thread_version later WHERE later.supersedes_id=version.id)
 AND ($6::timestamptz IS NULL OR (version.knowledge_available_from <= $6 AND membership.source_available_from <= $6))
 ORDER BY version.version_ordinal DESC,version.id,membership.thread_ordinal LIMIT 1
 ) merged ON true
 WHERE v.matter_id=$1::uuid AND v.court_case_id=$2::uuid AND n.normalized_generation_id=ANY($3::uuid[])
 AND ($4::timestamptz IS NULL OR n.occurred_at >= $4) AND ($5::timestamptz IS NULL OR n.occurred_at <= $5)
 AND ($6::timestamptz IS NULL OR (COALESCE(NULLIF(n.normalized_payload->>'source_available_from','')::timestamptz,merged.source_available_from) <= $6 AND n.occurred_at <= $6))
 AND (cardinality($7::text[])=0 OR EXISTS(SELECT 1 FROM jsonb_array_elements(COALESCE(n.normalized_payload->'participants','[]'::jsonb)) p WHERE p->>'identifier'=ANY($7::text[])))
 AND (cardinality($8::text[])=0 OR COALESCE(n.normalized_payload->'content'->'native_metadata'->>'platform',n.normalized_payload->'content'->'native_fields'->>'platform',v.declared_format)=ANY($8::text[]))
 AND ($10::uuid[] IS NULL OR n.id=ANY($10::uuid[]))
 ORDER BY n.occurred_at NULLS LAST,n.normalized_generation_id,n.record_ordinal,n.id LIMIT $9`

type discoverySourceRow struct {
	discovery.SourceRecord
	Payload        json.RawMessage `json:"payload"`
	DeclaredFormat string          `json:"declared_format"`
}

func decodeDiscoverySource(raw []byte) (discovery.SourceRecord, error) {
	var row discoverySourceRow
	if e := json.Unmarshal(raw, &row); e != nil {
		return discovery.SourceRecord{}, e
	}
	var p struct {
		SourceAvailableFrom *time.Time              `json:"source_available_from"`
		Participants        []discovery.Participant `json:"participants"`
		Content             struct {
			Body           string                     `json:"body"`
			NativeFields   map[string]json.RawMessage `json:"native_fields"`
			NativeMetadata map[string]json.RawMessage `json:"native_metadata"`
		} `json:"content"`
	}
	if e := json.Unmarshal(row.Payload, &p); e != nil {
		return discovery.SourceRecord{}, e
	}
	r := row.SourceRecord
	if parsed, e := time.Parse(time.RFC3339Nano, r.RecordRevision); e == nil {
		r.RecordRevision = parsed.UTC().Format(time.RFC3339Nano)
	}
	r.Text = p.Content.Body
	r.Participants = p.Participants
	r.NativeSourceAvailableFrom = p.SourceAvailableFrom
	if p.SourceAvailableFrom != nil {
		r.SourceAvailableFrom = p.SourceAvailableFrom
	}
	r.InputRevision = discovery.DigestBytes(row.Payload)
	get := func(key string) string {
		for _, m := range []map[string]json.RawMessage{p.Content.NativeMetadata, p.Content.NativeFields} {
			var v string
			if json.Unmarshal(m[key], &v) == nil && v != "" {
				return v
			}
		}
		return ""
	}
	r.Platform = get("platform")
	if r.Platform == "" {
		r.Platform = row.DeclaredFormat
	}
	r.ConversationID = get("conversation_id")
	if r.ConversationID == "" {
		r.ConversationID = get("thread_id")
	}
	if r.ExplicitEpisodeID == "" {
		r.ExplicitEpisodeID = get("explicit_episode_id")
	}
	if r.ExplicitEpisodeID == "" {
		r.ExplicitEpisodeID = get("episode_group_id")
	}
	if r.ImportedMessageID == "" {
		r.ImportedMessageID = get("message_id")
		if r.ImportedMessageID == "" {
			r.ImportedMessageID = get("external_id")
		}
	}
	r.SpeechForm = get("speech_form")
	r.AttributionBasis = get("attribution_basis")
	for _, m := range []map[string]json.RawMessage{p.Content.NativeMetadata, p.Content.NativeFields} {
		if len(r.AttributedActorIDs) == 0 {
			_ = json.Unmarshal(m["attributed_actor_ids"], &r.AttributedActorIDs)
		}
	}
	if r.Participants == nil {
		r.Participants = []discovery.Participant{}
	}
	if r.Locators == nil {
		r.Locators = []discovery.Locator{}
	}
	return r, nil
}

// ReadSources resolves one bounded, explicitly selected input set and checks every Surreal projection against fresh PG pins.
// Inputs: server-bound run. Outputs: immutable-source records and explicit partial coverage. Effects: read-only PG/Surreal I/O.
// Choose before discovery assembly; no evidence hash, retention status, review or promotion gate is required.
func (s *DiscoverySourceReader) ReadSources(ctx context.Context, in discovery.RunRequest) ([]discovery.SourceRecord, bool, error) {
	if in.Validate() != nil {
		return nil, false, discovery.ErrInvalid
	}
	d, e := discovery.Selection(in, s.datasets)
	if e != nil {
		return nil, false, e
	}
	var generationCount int
	if e = s.db.QueryRow(ctx, `SELECT count(*) FROM context.normalized_generation g JOIN context.source_version v ON v.id=g.source_version_id WHERE g.id=ANY($1::uuid[]) AND v.matter_id=$2::uuid AND v.court_case_id=$3::uuid`, in.GenerationIDs, in.Scope.MatterID, in.Scope.CourtCaseID).Scan(&generationCount); e != nil {
		return nil, false, e
	}
	if generationCount != len(in.GenerationIDs) {
		return nil, false, discovery.ErrScope
	}
	var projection []discovery.ProjectionRecord
	var ids []string
	if in.InputBackend == "surrealdb" {
		projection, e = s.surreal.ReadSelection(ctx, d, in, discovery.MaxRecords+1)
		if e != nil {
			return nil, false, e
		}
		if len(projection) == 0 {
			return []discovery.SourceRecord{}, false, nil
		}
		seen := map[string]bool{}
		allow := map[string]bool{}
		for _, id := range in.GenerationIDs {
			allow[id] = true
		}
		for _, p := range projection {
			if !discovery.ID(p.RecordID) || !discovery.ID(p.SourceVersionID) || !allow[p.GenerationID] || p.InputRevision == "" || p.RecordRevision == "" || seen[p.RecordID] {
				return nil, false, discovery.ErrScope
			}
			seen[p.RecordID] = true
			ids = append(ids, p.RecordID)
		}
	}
	var cutoff *time.Time
	if in.Filters.Perspective == "as_lived" {
		cutoff = in.Filters.Cutoff
	}
	participants := in.Filters.Participants
	if participants == nil {
		participants = []string{}
	}
	platforms := in.Filters.Platforms
	if platforms == nil {
		platforms = []string{}
	}
	rows, e := s.db.Query(ctx, DiscoverySourceSQL, in.Scope.MatterID, in.Scope.CourtCaseID, in.GenerationIDs, in.Filters.StartAt, in.Filters.EndAt, cutoff, participants, platforms, in.MaxRecords+1, ids)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []discovery.SourceRecord{}
	byID := map[string]discovery.SourceRecord{}
	total := 0
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return nil, false, e
		}
		total += len(raw)
		if total > discovery.MaxBytes {
			return nil, false, fmt.Errorf("%w: source batch exceeds finite byte budget", discovery.ErrInvalid)
		}
		r, e := decodeDiscoverySource(raw)
		if e != nil {
			return nil, false, e
		}
		if !discovery.ID(r.RecordID) || !discovery.ID(r.SourceVersionID) {
			return nil, false, discovery.ErrInvalid
		}
		out = append(out, r)
		byID[r.RecordID] = r
	}
	if e = rows.Err(); e != nil {
		return nil, false, e
	}
	if in.InputBackend == "surrealdb" {
		// Validate all projected IDs before filtering/scoring. A second unfiltered canonical selection prevents hidden foreign/missing rows.
		verify, e := s.db.Query(ctx, `SELECT n.id::text,n.source_version_id::text,n.normalized_generation_id::text,n.created_at,n.normalized_payload FROM context.normalized_record_identity n JOIN context.source_version v ON v.id=n.source_version_id WHERE n.id=ANY($1::uuid[]) AND v.matter_id=$2::uuid AND v.court_case_id=$3::uuid`, ids, in.Scope.MatterID, in.Scope.CourtCaseID)
		if e != nil {
			return nil, false, e
		}
		defer verify.Close()
		pins := map[string]discovery.ProjectionRecord{}
		for verify.Next() {
			var p discovery.ProjectionRecord
			var created time.Time
			var payload []byte
			if e = verify.Scan(&p.RecordID, &p.SourceVersionID, &p.GenerationID, &created, &payload); e != nil {
				return nil, false, e
			}
			p.RecordRevision = created.UTC().Format(time.RFC3339Nano)
			p.InputRevision = discovery.DigestBytes(payload)
			pins[p.RecordID] = p
		}
		if e = verify.Err(); e != nil {
			return nil, false, e
		}
		for _, p := range projection {
			pin, ok := pins[p.RecordID]
			if !ok || pin != p {
				return nil, false, discovery.ErrScope
			}
		}
	}
	truncated := len(out) > in.MaxRecords
	if truncated {
		out = out[:in.MaxRecords]
	}
	return out, truncated, nil
}
