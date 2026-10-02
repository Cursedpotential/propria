package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/investigation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InvestigationRequestStore struct {
	db       DB
	identity *CaseIdentityStore
}

func NewInvestigationRequestStore(db DB) (*InvestigationRequestStore, error) {
	identity, e := NewCaseIdentityStore(db)
	if e != nil {
		return nil, e
	}
	return &InvestigationRequestStore{db, identity}, nil
}

var _ investigation.Store = (*InvestigationRequestStore)(nil)

const investigationColumns = `request_id::text,payload,status,created_at,updated_at,results,payload_hash`

func scanInvestigation(row pgx.Row) (investigation.Receipt, string, error) {
	var r investigation.Receipt
	var payload, results []byte
	var hash string
	e := row.Scan(&r.RequestID, &payload, &r.Status, &r.CreatedAt, &r.UpdatedAt, &results, &hash)
	if e != nil {
		return r, "", e
	}
	if e = json.Unmarshal(payload, &r.Request); e != nil {
		return r, "", e
	}
	e = json.Unmarshal(results, &r.Results)
	return r, hash, e
}
func (s *InvestigationRequestStore) selected(ctx context.Context, q queryer, scope investigation.Scope) error {
	matter, e := s.identity.matterID(ctx, q, scope.Mode)
	if e != nil {
		return e
	}
	if matter == "" || matter != scope.MatterID {
		return investigation.ErrScope
	}
	court, e := scanCourtCase(q.QueryRow(ctx, caseCourtCaseSQL, matter))
	if e != nil {
		return e
	}
	if court == nil || court.ID != scope.CourtCaseID {
		return investigation.ErrScope
	}
	return nil
}

// Create serializes receipt admission only; no workflow or network call is made.
func (s *InvestigationRequestStore) Create(ctx context.Context, r investigation.Request, a investigation.Actor) (investigation.Receipt, error) {
	if e := investigation.Validate(r); e != nil {
		return investigation.Receipt{}, e
	}
	if e := investigation.ValidateActor(a); e != nil {
		return investigation.Receipt{}, e
	}
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return investigation.Receipt{}, e
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout = '8s'"); e != nil {
		return investigation.Receipt{}, e
	}
	// Fixed ordering prevents cross-key/correlation deadlocks. A global admission
	// lock is intentionally short and avoids first-insert races under READ COMMITTED.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('ops.legal_investigation_request/admission',0))`); e != nil {
		return investigation.Receipt{}, e
	}
	if e = s.selected(ctx, tx, r.Scope); e != nil {
		return investigation.Receipt{}, e
	}
	payload := investigation.CanonicalPayload(r)
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	alias, _ := json.Marshal([]map[string]string{{"actor_uid": a.UID, "key": a.Key}})
	existing, oldHash, e := scanInvestigation(tx.QueryRow(ctx, `SELECT `+investigationColumns+` FROM ops.legal_investigation_request WHERE (actor_uid=$1 AND idempotency_key=$2::uuid) OR idempotency_aliases @> $3::jsonb`, a.UID, a.Key, alias))
	if errors.Is(e, pgx.ErrNoRows) {
		existing, oldHash, e = scanInvestigation(tx.QueryRow(ctx, `SELECT `+investigationColumns+` FROM ops.legal_investigation_request WHERE mode=$1 AND matter_id=$2::uuid AND court_case_id=$3::uuid AND legal_matter_id=$4::uuid AND claim_id=$5::uuid AND followup_id=$6::uuid`, r.Mode, r.MatterID, r.CourtCaseID, r.LegalMatterID, r.ClaimID, r.FollowupID))
	}
	if e == nil {
		if oldHash != hash {
			return investigation.Receipt{}, investigation.ErrConflict
		}
		if _, e = tx.Exec(ctx, `UPDATE ops.legal_investigation_request SET idempotency_aliases=CASE WHEN idempotency_aliases @> $2::jsonb THEN idempotency_aliases ELSE idempotency_aliases || $2::jsonb END WHERE request_id=$1::uuid`, existing.RequestID, alias); e != nil {
			return investigation.Receipt{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return investigation.Receipt{}, e
		}
		return existing, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return investigation.Receipt{}, e
	}
	// Freshness is checked only after replay, using the same native read functions
	// as /legal-context/records; registry entities do not imply case participation.
	for _, source := range r.Sources {
		matched := false
		if source.Kind == "entity" {
			people, e := s.identity.readPeople(ctx, tx)
			if e != nil {
				return investigation.Receipt{}, e
			}
			for _, person := range people {
				if person.ID == source.RecordID {
					record, e := legalPersonRecord(person)
					if e != nil {
						return investigation.Receipt{}, e
					}
					matched = record.Origin.RecordVersion == source.RecordVersion
					break
				}
			}
		} else {
			records, _, e := fetchLegalEvents(ctx, tx, r.MatterID, r.CourtCaseID, LegalContextQuery{Mode: r.Mode, Kind: "event", RecordID: source.RecordID, Limit: 1})
			if e != nil {
				return investigation.Receipt{}, e
			}
			matched = len(records) == 1 && records[0].Origin.RecordVersion == source.RecordVersion
		}
		if !matched {
			return investigation.Receipt{}, investigation.ErrSource
		}
	}
	receipt, _, e := scanInvestigation(tx.QueryRow(ctx, `INSERT INTO ops.legal_investigation_request(request_id,mode,matter_id,court_case_id,legal_matter_id,claim_id,followup_id,actor_uid,actor_username,idempotency_key,payload_hash,payload) VALUES($1::uuid,$2,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10::uuid,$11,$12::jsonb) RETURNING `+investigationColumns, uuid.NewString(), r.Mode, r.MatterID, r.CourtCaseID, r.LegalMatterID, r.ClaimID, r.FollowupID, a.UID, a.Username, a.Key, hash, payload))
	if e != nil {
		return investigation.Receipt{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return investigation.Receipt{}, e
	}
	return receipt, nil
}
func (s *InvestigationRequestStore) Read(ctx context.Context, id string, scope investigation.Scope) (investigation.Receipt, error) {
	if !investigation.ValidID(id) {
		return investigation.Receipt{}, investigation.ErrNotFound
	}
	if e := investigation.ValidateScope(scope); e != nil {
		return investigation.Receipt{}, e
	}
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return investigation.Receipt{}, e
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout = '8s'"); e != nil {
		return investigation.Receipt{}, e
	}
	if e = s.selected(ctx, tx, scope); e != nil {
		return investigation.Receipt{}, e
	}
	r, _, e := scanInvestigation(tx.QueryRow(ctx, `SELECT `+investigationColumns+` FROM ops.legal_investigation_request WHERE request_id=$1::uuid`, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return r, investigation.ErrNotFound
	}
	if e != nil {
		return r, e
	}
	if r.Scope != scope {
		return investigation.Receipt{}, investigation.ErrScope
	}
	return r, nil
}

// Transition is an internal future executor persistence seam. It neither
// schedules execution nor exposes owner HTTP mutation of lifecycle state.
func (s *InvestigationRequestStore) Transition(ctx context.Context, id, expected, next string, results []investigation.Result) (investigation.Receipt, error) {
	if !investigation.ValidID(id) {
		return investigation.Receipt{}, investigation.ErrNotFound
	}
	if e := investigation.ValidateTransition(expected, next, results); e != nil {
		return investigation.Receipt{}, e
	}
	for i := range results {
		if results[i].Sources == nil {
			results[i].Sources = []investigation.Source{}
		}
	}
	if results == nil {
		results = []investigation.Result{}
	}
	raw, _ := json.Marshal(results)
	tx, e := s.db.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return investigation.Receipt{}, e
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout = '8s'"); e != nil {
		return investigation.Receipt{}, e
	}
	receipt, _, e := scanInvestigation(tx.QueryRow(ctx, `UPDATE ops.legal_investigation_request SET status=$3,results=$4::jsonb,updated_at=clock_timestamp() WHERE request_id=$1::uuid AND status=$2 RETURNING `+investigationColumns, id, expected, next, raw))
	if errors.Is(e, pgx.ErrNoRows) {
		return investigation.Receipt{}, investigation.ErrConflict
	}
	if e != nil {
		return investigation.Receipt{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return investigation.Receipt{}, e
	}
	return receipt, nil
}
