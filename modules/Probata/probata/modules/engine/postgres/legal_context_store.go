// Legal context is a read-through of native identities and committed context
// events. It never promotes context to evidence or creates a second store.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
)

type LegalContextQuery struct {
	Mode     caseidentity.Mode
	Kind     string
	Search   string
	RecordID string
	Limit    int
}

type LegalContextOrigin struct {
	System        string `json:"system"`
	Kind          string `json:"kind"`
	RecordID      string `json:"record_id"`
	RecordVersion string `json:"record_version"`
}

type LegalContextRecord struct {
	Origin LegalContextOrigin `json:"origin"`
	Title  string             `json:"title"`
	Record json.RawMessage    `json:"record"`
}

type LegalContextResult struct {
	Available   bool                 `json:"available"`
	Mode        caseidentity.Mode    `json:"mode"`
	MatterID    string               `json:"matter_id"`
	CourtCaseID string               `json:"court_case_id"`
	Records     []LegalContextRecord `json:"records"`
	Truncated   bool                 `json:"truncated"`
}

type LegalContextStore struct {
	db       DB
	identity *CaseIdentityStore
}

func NewLegalContextStore(db DB) (*LegalContextStore, error) {
	identity, err := NewCaseIdentityStore(db)
	if err != nil {
		return nil, err
	}
	return &LegalContextStore{db: db, identity: identity}, nil
}

func (s *LegalContextStore) ReadLegalContext(ctx context.Context, query LegalContextQuery) (LegalContextResult, error) {
	mode, modeErr := caseidentity.ParseMode(string(query.Mode))
	result := LegalContextResult{Mode: mode, Records: []LegalContextRecord{}}
	if modeErr != nil {
		return result, modeErr
	}
	query.Mode = mode
	if (query.Kind != "entity" && query.Kind != "event") || query.Limit < 1 || query.Limit > 100 {
		return result, errors.New("invalid legal context query")
	}
	if query.Kind == "entity" {
		view, err := s.identity.Read(ctx, query.Mode)
		if err != nil {
			return result, err
		}
		if view.Matter != nil {
			result.MatterID = view.Matter.ID
		}
		if view.CourtCase != nil {
			result.CourtCaseID = view.CourtCase.ID
		}
		// Registry people are deliberately a shared identity catalog, as on the
		// Case page. Presence is not participation or case evidence.
		for _, person := range view.People {
			if query.RecordID != "" && query.RecordID != person.ID {
				continue
			}
			record, err := legalPersonRecord(person)
			if err != nil {
				return result, err
			}
			if query.Search != "" && !strings.Contains(strings.ToLower(string(record.Record)), strings.ToLower(query.Search)) {
				continue
			}
			result.Records = append(result.Records, record)
			if len(result.Records) > query.Limit {
				result.Truncated = true
				result.Records = result.Records[:query.Limit]
				break
			}
		}
		result.Available = true
		return result, nil
	}
	// Resolve the admitted case and its events in one repeatable read-only
	// snapshot, rather than accept an arbitrary client matter or collection.
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err = tx.Exec(ctx, "SET LOCAL statement_timeout = '8s'"); err != nil {
		return result, err
	}
	result.MatterID, err = s.identity.matterID(ctx, tx, query.Mode)
	if err != nil {
		return result, err
	}
	if result.MatterID == "" {
		result.Available = true
		return result, nil
	}
	court, err := scanCourtCase(tx.QueryRow(ctx, caseCourtCaseSQL, result.MatterID))
	if err != nil {
		return result, err
	}
	if court == nil {
		result.Available = true
		return result, nil
	}
	result.CourtCaseID = court.ID
	result.Records, result.Truncated, err = fetchLegalEvents(ctx, tx, result.MatterID, court.ID, query)
	if err != nil {
		return result, err
	}
	result.Available = true
	return result, nil
}

func legalPersonRecord(person caseidentity.Person) (LegalContextRecord, error) {
	raw, err := json.Marshal(struct {
		Scope  string              `json:"scope"`
		Person caseidentity.Person `json:"person"`
	}{"registry", person})
	if err != nil {
		return LegalContextRecord{}, err
	}
	// This is explicitly a view fingerprint, not a fabricated native version.
	digest := sha256.Sum256(raw)
	return LegalContextRecord{Origin: LegalContextOrigin{System: "probata", Kind: "entity", RecordID: person.ID, RecordVersion: "view-sha256:" + hex.EncodeToString(digest[:])}, Title: person.DisplayName, Record: raw}, nil
}

const legalEventsSQL = `SELECT e.id::text, e.source_record_version, e.display_summary,
 jsonb_build_object('scope','case','matter_id',$1::text,'court_case_id',$2::text,
 'id',e.id,'source_system',e.source_system,'source_record_id',e.source_record_id,
 'source_record_version',e.source_record_version,'source_locator',e.source_locator,
 'extraction_run_id',e.extraction_run_id,'temporal_precision',e.temporal_precision,
 'occurred_at',e.occurred_at,'valid_from',e.valid_from,'valid_to',e.valid_to,
 'display_summary',e.display_summary,'event_type',e.event_type,'entity_refs',e.entity_refs,
 'authority','candidate_context','created_at',e.created_at,
 'memberships',(SELECT jsonb_agg(jsonb_build_object('member_id',m.id,'collection_id',m.collection_id) ORDER BY m.id)
 FROM timeline.timeline_member m WHERE m.candidate_id=e.id AND m.included AND m.member_authority='candidate_context'))
 FROM timeline.event_candidate e
 WHERE e.source_system='probata.context' AND e.source_record_version IS NOT NULL AND e.source_record_version<>''
 AND e.source_locator->>'schema'='probata.event-source/v1'
 AND EXISTS (SELECT 1 FROM timeline.timeline_member m WHERE m.candidate_id=e.id AND m.included AND m.member_authority='candidate_context')
 AND EXISTS (SELECT 1 FROM context.proffer_preview_snapshot sn
 JOIN context.source_version sv ON sv.id=sn.source_version_id
 WHERE sn.preview_handle=e.source_locator->>'preview_handle'
 AND sn.normalized_generation_id::text=e.source_locator->>'normalized_generation_id'
 AND sv.matter_id=$1::uuid AND sv.court_case_id=$2::uuid)
 AND ($3::text='' OR e.display_summary ILIKE $4 OR COALESCE(e.source_locator->>'description','') ILIKE $4)
 AND ($5::text='' OR e.id::text=$5)
 ORDER BY e.occurred_at DESC NULLS LAST,e.id LIMIT $6`

func fetchLegalEvents(ctx context.Context, db queryer, matter, court string, query LegalContextQuery) ([]LegalContextRecord, bool, error) {
	rows, err := db.Query(ctx, legalEventsSQL, matter, court, query.Search, "%"+query.Search+"%", query.RecordID, query.Limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	records := []LegalContextRecord{}
	for rows.Next() {
		record := LegalContextRecord{Origin: LegalContextOrigin{System: "probata", Kind: "event"}}
		if err := rows.Scan(&record.Origin.RecordID, &record.Origin.RecordVersion, &record.Title, &record.Record); err != nil {
			return nil, false, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := len(records) > query.Limit
	if truncated {
		records = records[:query.Limit]
	}
	return records, truncated, nil
}
