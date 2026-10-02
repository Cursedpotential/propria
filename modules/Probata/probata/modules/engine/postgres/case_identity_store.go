// Byline: Claude Code · Opus 5.5 · 2026-10-01
//
// Registry store behind the Workbench Case page (caseidentity.Store).
//
// Reads: the case header, every person, every identifier chain of every
// person, the identity change log, what Probata's working tables hold per
// identifier, and the participants no person carries yet.
//
// Writes never overwrite an identifier: a new version is a new
// registry.entity_alias row whose supersedes_id names the chain's current row
// (UNIQUE(supersedes_id) makes a fork impossible, the append-only trigger
// makes an UPDATE impossible). Case-header and person edits update the
// registry row and append the row's before and after to
// registry.identity_change in the same transaction. Every write carries an
// actor-scoped idempotency key, so a retried click answers with the first
// receipt instead of writing twice.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// CaseIdentityStore reads and edits the case identity registry.
type CaseIdentityStore struct {
	db    DB
	clock func() time.Time
}

// NewCaseIdentityStore requires a database.
func NewCaseIdentityStore(db DB) (*CaseIdentityStore, error) {
	if db == nil {
		return nil, errors.New("case identity store requires a database")
	}
	return &CaseIdentityStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

var _ caseidentity.Store = (*CaseIdentityStore)(nil)

// placeholderMatterWriters are the created_by values of pre-launch placeholder
// identity rows (registry.reseed_dev_case_identity).
var placeholderMatterWriters = []string{"migration-0030", "migration-0069-dev-seed"}

const caseMatterColumns = `m.id::text, m.title, m.description, m.status, m.verification_state, m.created_by, m.updated_at`

// caseMatterIDSQL resolves the matter a mode shows. TEST is the DEV sentinel.
// REAL is the one matter that is not placeholder data; when several exist the
// engine's admitted go-live identity wins, and otherwise the read fails closed
// rather than pick one.
const caseRealMattersSQL = `SELECT m.id::text FROM registry.matter m WHERE m.created_by <> ALL($1::text[]) ORDER BY m.created_at, m.id LIMIT 10`

func (s *CaseIdentityStore) matterID(ctx context.Context, q queryer, mode caseidentity.Mode) (string, error) {
	if mode == caseidentity.ModeTest {
		return devMatterID, nil
	}
	rows, err := q.Query(ctx, caseRealMattersSQL, placeholderMatterWriters)
	if err != nil {
		return "", caseIdentityError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return "", err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", caseIdentityError(err)
	}
	switch len(ids) {
	case 0:
		return "", nil
	case 1:
		return ids[0], nil
	}
	for _, id := range ids {
		if strings.EqualFold(id, authoritativeMatterID) {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: %d non-placeholder matters exist; the case identity is split", caseidentity.ErrStale, len(ids))
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanMatter(row pgx.Row) (*caseidentity.Matter, error) {
	var m caseidentity.Matter
	err := row.Scan(&m.ID, &m.Title, &m.Description, &m.Status, &m.VerificationState, &m.CreatedBy, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, caseIdentityError(err)
	}
	return &m, nil
}

const caseCourtCaseSQL = `SELECT c.id::text, c.matter_id::text, c.caption, c.docket_number, c.court_name, c.jurisdiction,
       c.case_type, c.presiding_judge, c.status, to_char(c.filed_on, 'YYYY-MM-DD'), to_char(c.closed_on, 'YYYY-MM-DD'),
       c.verification_state, c.updated_at
FROM registry.court_case c WHERE c.matter_id = $1::uuid ORDER BY c.is_primary DESC, c.created_at LIMIT 1`

func scanCourtCase(row pgx.Row) (*caseidentity.CourtCase, error) {
	var c caseidentity.CourtCase
	err := row.Scan(&c.ID, &c.MatterID, &c.Caption, &c.DocketNumber, &c.CourtName, &c.Jurisdiction, &c.CaseType, &c.PresidingJudge,
		&c.Status, &c.FiledOn, &c.ClosedOn, &c.VerificationState, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, caseIdentityError(err)
	}
	return &c, nil
}

const casePeopleSQL = `SELECT p.id::text, coalesce(e.display_name::text, e.canonical_name::text, ''), e.canonical_name::text,
       p.short_name, p.role_in_case, p.connection_to, p.relationship_type, p.is_minor, e.is_party, p.notes,
       p.verification_state
FROM registry.person p JOIN registry.entity e ON e.id = p.id
WHERE e.merged_into_id IS NULL
ORDER BY CASE p.role_in_case WHEN 'user' THEN 0 ELSE 1 END, e.created_at, p.id`

const caseAliasesSQL = `SELECT a.id::text, a.entity_id::text, a.alias_text::text, coalesce(a.alias_kind, 'other'),
       coalesce(a.normalized, ''), a.status, a.period, a.basis, a.change_reason, a.recorded_by, a.created_at,
       a.supersedes_id::text
FROM registry.entity_alias a WHERE a.entity_id = ANY($1::uuid[])
ORDER BY a.entity_id, a.created_at, a.id`

type aliasRow struct {
	caseidentity.IdentifierVersion
	entityID, raw, kind, normalized string
}

// chainIdentifiers groups alias rows into chains: the current row of each
// chain (no row supersedes it) carries the earlier rows, newest first.
func chainIdentifiers(rows []aliasRow) map[string][]caseidentity.Identifier {
	byID := make(map[string]aliasRow, len(rows))
	superseded := make(map[string]bool, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
		if row.SupersedesID != nil {
			superseded[*row.SupersedesID] = true
		}
	}
	out := map[string][]caseidentity.Identifier{}
	for _, row := range rows {
		if superseded[row.ID] {
			continue
		}
		identifier := caseidentity.Identifier{
			IdentifierVersion: row.IdentifierVersion, EntityID: row.entityID, RawValue: row.raw,
			Kind: row.kind, Normalized: row.normalized, History: []caseidentity.IdentifierVersion{},
		}
		seen := map[string]bool{row.ID: true}
		for previous := row.SupersedesID; previous != nil; {
			earlier, ok := byID[*previous]
			if !ok || seen[earlier.ID] {
				break
			}
			seen[earlier.ID] = true
			identifier.History = append(identifier.History, earlier.IdentifierVersion)
			previous = earlier.SupersedesID
		}
		out[row.entityID] = append(out[row.entityID], identifier)
	}
	for entity := range out {
		list := out[entity]
		sort.SliceStable(list, func(i, j int) bool {
			if kindRank(list[i].Kind) != kindRank(list[j].Kind) {
				return kindRank(list[i].Kind) < kindRank(list[j].Kind)
			}
			return list[i].Normalized < list[j].Normalized
		})
	}
	return out
}

func kindRank(kind string) int {
	switch kind {
	case "phone":
		return 0
	case "email":
		return 1
	case "account", "handle":
		return 2
	case "other":
		return 4
	}
	return 3
}

const caseHistorySQL = `SELECT c.id::text, c.subject_table, c.subject_id::text, c.before_state, c.after_state,
       c.change_reason, c.recorded_by, c.recorded_at
FROM registry.identity_change c ORDER BY c.recorded_at DESC, c.id DESC LIMIT $1`

// caseCountsSQL counts, per key, what Probata's working tables hold: first-
// and third-party message participants, calls and entity mentions. A key is
// an identifier's normalized form (matched against the participant's raw or
// E.164 value, normalized by the same rule) or, for participants the engine
// already resolved, the person's entity id.
const caseCountsSQL = `
WITH observed AS (
    SELECT registry.norm_identifier(coalesce(nullif(p.participant_e164, ''), p.participant_raw)) AS k,
           p.entity_id::text AS entity, 'first_party_message' AS src, m.ts_utc AS t
    FROM working.message_participant p JOIN working.message m ON m.id = p.message_id
    UNION ALL
    SELECT registry.norm_identifier(coalesce(nullif(p.participant_e164, ''), p.participant_raw)),
           p.entity_id::text, 'third_party_message', m.occurred_at
    FROM working.third_party_message_participant p JOIN working.third_party_message m ON m.id = p.message_id
    UNION ALL
    SELECT registry.norm_identifier(coalesce(nullif(c.from_e164, ''), c.from_raw)), c.from_entity_id::text, 'call', c.started_at
    FROM working.call_log c
    UNION ALL
    SELECT registry.norm_identifier(coalesce(nullif(c.to_e164, ''), c.to_raw)), c.to_entity_id::text, 'call', c.started_at
    FROM working.call_log c
    UNION ALL
    SELECT registry.norm_identifier(e.surface_text::text), r.canonical_entity_id::text, 'mention', e.created_at
    FROM working.entity_mention e
    LEFT JOIN working.entity_resolution r ON r.mention_id = e.id AND upper_inf(r.sys_period)
)
SELECT k, src, count(*), min(t), max(t) FROM observed WHERE k = ANY($1::text[]) GROUP BY k, src
UNION ALL
SELECT entity, src, count(*), min(t), max(t) FROM observed WHERE entity = ANY($2::text[]) GROUP BY entity, src`

// caseUnknownsSQL lists participant identifiers no person carries (and the
// owner has not dismissed), most frequent first.
const caseUnknownsSQL = `
WITH observed AS (
    SELECT coalesce(nullif(p.participant_e164, ''), p.participant_raw) AS raw, m.ts_utc AS t
    FROM working.message_participant p JOIN working.message m ON m.id = p.message_id WHERE p.entity_id IS NULL
    UNION ALL
    SELECT coalesce(nullif(p.participant_e164, ''), p.participant_raw), m.occurred_at
    FROM working.third_party_message_participant p JOIN working.third_party_message m ON m.id = p.message_id
    WHERE p.entity_id IS NULL
    UNION ALL
    SELECT coalesce(nullif(c.from_e164, ''), c.from_raw), c.started_at FROM working.call_log c WHERE c.from_entity_id IS NULL
    UNION ALL
    SELECT coalesce(nullif(c.to_e164, ''), c.to_raw), c.started_at FROM working.call_log c WHERE c.to_entity_id IS NULL
), keyed AS (
    SELECT registry.norm_identifier(raw) AS k, raw, t FROM observed WHERE raw IS NOT NULL AND btrim(raw) <> ''
)
SELECT k, min(raw), count(*), min(t), max(t) FROM keyed
WHERE k IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM registry.entity_alias_current a WHERE a.normalized = keyed.k AND a.status <> 'retired')
  AND NOT EXISTS (SELECT 1 FROM registry.vw_identifier_dismissed d WHERE d.normalized = keyed.k)
GROUP BY k ORDER BY count(*) DESC, k LIMIT $1`

const caseDismissedSQL = `SELECT normalized, raw_value, basis, recorded_by, recorded_at FROM registry.vw_identifier_dismissed ORDER BY recorded_at DESC LIMIT $1`

// Read returns the whole Case page in one read-only transaction so every
// section answers from the same snapshot.
func (s *CaseIdentityStore) Read(ctx context.Context, mode caseidentity.Mode) (caseidentity.View, error) {
	view := caseidentity.View{
		Mode: mode, People: []caseidentity.Person{}, History: []caseidentity.Change{},
		Counts: []caseidentity.Count{}, Unknowns: []caseidentity.Unknown{}, Dismissed: []caseidentity.Dismissal{},
		CountStore: "probata",
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return view, caseIdentityError(err)
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err := tx.Exec(ctx, `SET LOCAL statement_timeout = '8s'`); err != nil {
		return view, caseIdentityError(err)
	}
	matterID, err := s.matterID(ctx, tx, mode)
	if err != nil {
		return view, err
	}
	if matterID != "" {
		if view.Matter, err = scanMatter(tx.QueryRow(ctx, `SELECT `+caseMatterColumns+` FROM registry.matter m WHERE m.id = $1::uuid`, matterID)); err != nil {
			return view, err
		}
		if view.CourtCase, err = scanCourtCase(tx.QueryRow(ctx, caseCourtCaseSQL, matterID)); err != nil {
			return view, err
		}
	}
	if view.People, err = s.readPeople(ctx, tx); err != nil {
		return view, err
	}
	if view.History, err = readHistory(ctx, tx); err != nil {
		return view, err
	}
	var keys, entities []string
	for _, person := range view.People {
		entities = append(entities, person.ID)
		for _, identifier := range person.Identifiers {
			if identifier.Normalized != "" && len(keys) < caseidentity.MaxCountKeys {
				keys = append(keys, identifier.Normalized)
			}
		}
	}
	if view.Counts, err = readCounts(ctx, tx, keys, entities); err != nil {
		return view, err
	}
	if view.Unknowns, err = readUnknowns(ctx, tx); err != nil {
		return view, err
	}
	if view.Dismissed, err = readDismissed(ctx, tx); err != nil {
		return view, err
	}
	return view, nil
}

func (s *CaseIdentityStore) readPeople(ctx context.Context, q queryer) ([]caseidentity.Person, error) {
	rows, err := q.Query(ctx, casePeopleSQL)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	people := []caseidentity.Person{}
	for rows.Next() {
		var p caseidentity.Person
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.CanonicalName, &p.ShortName, &p.RoleInCase, &p.ConnectionTo,
			&p.RelationshipType, &p.IsMinor, &p.IsParty, &p.Notes, &p.VerificationState); err != nil {
			rows.Close()
			return nil, err
		}
		p.Identifiers = []caseidentity.Identifier{}
		people = append(people, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, caseIdentityError(err)
	}
	if len(people) == 0 {
		return people, nil
	}
	ids := make([]string, len(people))
	for i, p := range people {
		ids[i] = p.ID
	}
	aliasRows, err := q.Query(ctx, caseAliasesSQL, ids)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	var all []aliasRow
	for aliasRows.Next() {
		var row aliasRow
		if err := aliasRows.Scan(&row.ID, &row.entityID, &row.raw, &row.kind, &row.normalized, &row.Status, &row.Period,
			&row.Basis, &row.ChangeReason, &row.RecordedBy, &row.RecordedAt, &row.SupersedesID); err != nil {
			aliasRows.Close()
			return nil, err
		}
		all = append(all, row)
	}
	aliasRows.Close()
	if err := aliasRows.Err(); err != nil {
		return nil, caseIdentityError(err)
	}
	chains := chainIdentifiers(all)
	for i := range people {
		if list, ok := chains[people[i].ID]; ok {
			people[i].Identifiers = list
		}
	}
	return people, nil
}

func readHistory(ctx context.Context, q queryer) ([]caseidentity.Change, error) {
	rows, err := q.Query(ctx, caseHistorySQL, caseidentity.MaxHistoryItems)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	defer rows.Close()
	out := []caseidentity.Change{}
	for rows.Next() {
		var c caseidentity.Change
		var before, after []byte
		if err := rows.Scan(&c.ID, &c.SubjectTable, &c.SubjectID, &before, &after, &c.ChangeReason, &c.RecordedBy, &c.RecordedAt); err != nil {
			return nil, err
		}
		c.Before, c.After = json.RawMessage(before), json.RawMessage(after)
		out = append(out, c)
	}
	return out, caseIdentityError(rows.Err())
}

func readCounts(ctx context.Context, q queryer, keys, entities []string) ([]caseidentity.Count, error) {
	out := []caseidentity.Count{}
	if len(keys)+len(entities) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, caseCountsSQL, keys, entities)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var c caseidentity.Count
		if err := rows.Scan(&c.Key, &c.Source, &c.Events, &c.FirstAt, &c.LastAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, caseIdentityError(rows.Err())
}

func readUnknowns(ctx context.Context, q queryer) ([]caseidentity.Unknown, error) {
	rows, err := q.Query(ctx, caseUnknownsSQL, caseidentity.MaxUnknownItems)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	defer rows.Close()
	out := []caseidentity.Unknown{}
	for rows.Next() {
		var u caseidentity.Unknown
		if err := rows.Scan(&u.Normalized, &u.RawValue, &u.Events, &u.FirstAt, &u.LastAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, caseIdentityError(rows.Err())
}

func readDismissed(ctx context.Context, q queryer) ([]caseidentity.Dismissal, error) {
	rows, err := q.Query(ctx, caseDismissedSQL, caseidentity.MaxHistoryItems)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	defer rows.Close()
	out := []caseidentity.Dismissal{}
	for rows.Next() {
		var d caseidentity.Dismissal
		if err := rows.Scan(&d.Normalized, &d.RawValue, &d.Basis, &d.RecordedBy, &d.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, caseIdentityError(rows.Err())
}

// ---- writes ----------------------------------------------------------------

func (s *CaseIdentityStore) begin(ctx context.Context, lock string) (pgx.Tx, func(), error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, caseIdentityError(err)
	}
	rollback := func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }
	if err := tx.QueryRow(ctx, `SELECT 1 FROM (SELECT pg_advisory_xact_lock(hashtextextended($1, 0))) AS lock`,
		"case-identity:"+lock).Scan(new(int)); err != nil {
		rollback()
		return nil, nil, caseIdentityError(err)
	}
	return tx, rollback, nil
}

const identifierByKeySQL = `SELECT a.id::text, a.entity_id::text, a.alias_text::text, coalesce(a.alias_kind, ''), a.status,
       a.period, coalesce(a.basis, ''), coalesce(a.supersedes_id::text, ''), a.created_at
FROM registry.entity_alias a WHERE a.idempotency_key = $1`

// WriteIdentifier appends one identifier row: a new chain, or the next
// version of an existing one.
func (s *CaseIdentityStore) WriteIdentifier(ctx context.Context, spec caseidentity.IdentifierSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateIdentifier(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("identifier")
	tx, rollback, err := s.begin(ctx, "person:"+strings.ToLower(spec.EntityID))
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	var existing struct {
		id, entity, raw, kind, status, basis, supersedes string
		period                                           *string
		at                                               time.Time
	}
	err = tx.QueryRow(ctx, identifierByKeySQL, key).Scan(&existing.id, &existing.entity, &existing.raw, &existing.kind,
		&existing.status, &existing.period, &existing.basis, &existing.supersedes, &existing.at)
	switch {
	case err == nil:
		rollback()
		same := strings.EqualFold(existing.entity, spec.EntityID) && existing.raw == spec.RawValue && existing.kind == spec.Kind &&
			existing.status == spec.Status && existing.basis == spec.Basis && strings.EqualFold(existing.supersedes, spec.SupersedesID) &&
			equalOptional(existing.period, spec.Period)
		if !same {
			return caseidentity.Receipt{}, caseidentity.ErrIdempotencyConflict
		}
		return caseidentity.Receipt{Ref: existing.id, Kind: "registry.entity_alias", RecordedAt: existing.at, Replayed: true}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	var isPerson bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registry.person p JOIN registry.entity e ON e.id = p.id
		WHERE p.id = $1::uuid AND e.merged_into_id IS NULL)`, spec.EntityID).Scan(&isPerson); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	if !isPerson {
		rollback()
		return caseidentity.Receipt{}, fmt.Errorf("%w: no person %s", caseidentity.ErrNotFound, spec.EntityID)
	}
	var supersedes any
	if spec.SupersedesID != "" {
		var raw string
		var current bool
		err := tx.QueryRow(ctx, `SELECT a.alias_text::text,
			NOT EXISTS (SELECT 1 FROM registry.entity_alias s WHERE s.supersedes_id = a.id)
			FROM registry.entity_alias a WHERE a.id = $1::uuid AND a.entity_id = $2::uuid`, spec.SupersedesID, spec.EntityID).Scan(&raw, &current)
		if errors.Is(err, pgx.ErrNoRows) {
			rollback()
			return caseidentity.Receipt{}, fmt.Errorf("%w: identifier %s does not belong to this person", caseidentity.ErrNotFound, spec.SupersedesID)
		}
		if err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if !current {
			rollback()
			return caseidentity.Receipt{}, caseidentity.ErrStale
		}
		if !strings.EqualFold(raw, spec.RawValue) {
			rollback()
			return caseidentity.Receipt{}, fmt.Errorf("%w: a new version keeps the raw spelling %q; add a new identifier for a new spelling", caseidentity.ErrRejected, raw)
		}
		spec.RawValue = raw
		supersedes = spec.SupersedesID
	} else {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registry.entity_alias a
			WHERE a.entity_id = $1::uuid AND lower(a.alias_text::text) = lower($2))`, spec.EntityID, spec.RawValue).Scan(&exists); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if exists {
			rollback()
			return caseidentity.Receipt{}, fmt.Errorf("%w: this person already carries %q; edit that identifier instead", caseidentity.ErrStale, spec.RawValue)
		}
	}
	id, err := uuid.NewV7()
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	at := s.clock()
	var changeReason any
	if strings.TrimSpace(spec.ChangeReason) != "" {
		changeReason = spec.ChangeReason
	}
	if _, err := tx.Exec(ctx, `INSERT INTO registry.entity_alias
		    (id, entity_id, alias_text, alias_kind, status, period, basis, change_reason, recorded_by, idempotency_key,
		     supersedes_id, created_at, provenance)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11::uuid, $12,
		        ARRAY[ROW('postgres', $13::text, 'workbench.case_identity')::ai.source_ref])`,
		id, spec.EntityID, spec.RawValue, spec.Kind, spec.Status, spec.Period, spec.Basis, changeReason,
		actor.Username, key, supersedes, at, id.String()); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: id.String(), Kind: "registry.entity_alias", RecordedAt: at}, nil
}

func equalOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// replayChange answers a retried header/person/new-person act.
func replayChange(ctx context.Context, tx pgx.Tx, key, subjectTable, subjectID string) (caseidentity.Receipt, bool, error) {
	var ref, table, subject string
	var at time.Time
	err := tx.QueryRow(ctx, `SELECT id::text, subject_table, subject_id::text, recorded_at FROM registry.identity_change WHERE idempotency_key = $1`, key).
		Scan(&ref, &table, &subject, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return caseidentity.Receipt{}, false, nil
	}
	if err != nil {
		return caseidentity.Receipt{}, false, caseIdentityError(err)
	}
	if table != subjectTable || (subjectID != "" && !strings.EqualFold(subject, subjectID)) {
		return caseidentity.Receipt{}, false, caseidentity.ErrIdempotencyConflict
	}
	return caseidentity.Receipt{Ref: ref, Kind: "registry.identity_change", RecordedAt: at, Replayed: true}, true, nil
}

func insertChange(ctx context.Context, tx pgx.Tx, table, subjectID string, before, after []byte, reason string, actor caseidentity.Actor, key string, at time.Time) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO registry.identity_change
		    (id, subject_table, subject_id, before_state, after_state, change_reason, recorded_by, recorded_by_uid, idempotency_key, recorded_at)
		VALUES ($1, $2, $3::uuid, $4::jsonb, $5::jsonb, $6, $7, $8, $9, $10)`,
		id, table, subjectID, before, after, reason, actor.Username, actor.SubjectUID, key, at)
	return id.String(), err
}

// headerStateSQL is the editable state of the matter or court case, as JSON.
var headerStateSQL = map[string]string{
	"matter": `SELECT jsonb_build_object('title', title, 'description', description, 'status', status,
		'verification_state', verification_state), updated_at FROM registry.matter WHERE id = $1::uuid`,
	"court_case": `SELECT jsonb_build_object('caption', caption, 'docket_number', docket_number, 'court_name', court_name,
		'jurisdiction', jurisdiction, 'case_type', case_type, 'presiding_judge', presiding_judge, 'status', status, 'filed_on', filed_on,
		'closed_on', closed_on, 'verification_state', verification_state), updated_at
		FROM registry.court_case WHERE id = $1::uuid AND matter_id = $2::uuid`,
}

// EditHeader updates the matter or its court case and logs before/after.
func (s *CaseIdentityStore) EditHeader(ctx context.Context, mode caseidentity.Mode, spec caseidentity.HeaderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateHeader(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	table := "registry." + spec.Target
	key := actor.StoredKey("header")
	tx, rollback, err := s.begin(ctx, table+":"+strings.ToLower(spec.ID))
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	if receipt, replayed, err := replayChange(ctx, tx, key, table, spec.ID); err != nil || replayed {
		rollback()
		return receipt, err
	}
	matterID, err := s.matterID(ctx, tx, mode)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	if matterID == "" || (spec.Target == "matter" && !strings.EqualFold(spec.ID, matterID)) {
		rollback()
		return caseidentity.Receipt{}, fmt.Errorf("%w: %s %s is not this mode's case", caseidentity.ErrNotFound, spec.Target, spec.ID)
	}
	args := []any{spec.ID}
	if spec.Target == "court_case" {
		args = append(args, matterID)
	}
	var before []byte
	var updatedAt time.Time
	if err := tx.QueryRow(ctx, headerStateSQL[spec.Target], args...).Scan(&before, &updatedAt); err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return caseidentity.Receipt{}, fmt.Errorf("%w: %s %s is not this mode's case", caseidentity.ErrNotFound, spec.Target, spec.ID)
		}
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	if spec.ExpectedAt != nil && !spec.ExpectedAt.Equal(updatedAt) {
		rollback()
		return caseidentity.Receipt{}, caseidentity.ErrStale
	}
	columns := sortedKeys(spec.Fields)
	sets := make([]string, 0, len(columns))
	values := []any{spec.ID}
	for _, column := range columns {
		values = append(values, spec.Fields[column])
		cast := ""
		if column == "filed_on" || column == "closed_on" {
			cast = "::date"
		}
		// column comes from caseidentity.HeaderColumns (ValidateHeader), never from input text.
		sets = append(sets, fmt.Sprintf("%s = $%d%s", column, len(values), cast))
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s WHERE id = $1::uuid", table, strings.Join(sets, ", ")), values...); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	var after []byte
	if err := tx.QueryRow(ctx, headerStateSQL[spec.Target], args...).Scan(&after, &updatedAt); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	at := s.clock()
	ref, err := insertChange(ctx, tx, table, spec.ID, before, after, spec.ChangeReason, actor, key, at)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: ref, Kind: "registry.identity_change", RecordedAt: at}, nil
}

const personStateSQL = `SELECT jsonb_build_object('display_name', e.display_name, 'canonical_name', e.canonical_name,
	'short_name', p.short_name, 'role_in_case', p.role_in_case, 'connection_to', p.connection_to,
	'relationship_type', p.relationship_type, 'notes', p.notes, 'is_minor', p.is_minor,
	'verification_state', p.verification_state)
FROM registry.person p JOIN registry.entity e ON e.id = p.id WHERE p.id = $1::uuid AND e.merged_into_id IS NULL`

// EditPerson updates one person's entity/person columns and logs before/after.
func (s *CaseIdentityStore) EditPerson(ctx context.Context, spec caseidentity.PersonSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidatePerson(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("person")
	tx, rollback, err := s.begin(ctx, "person:"+strings.ToLower(spec.ID))
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	if receipt, replayed, err := replayChange(ctx, tx, key, "registry.person", spec.ID); err != nil || replayed {
		rollback()
		return receipt, err
	}
	var before []byte
	if err := tx.QueryRow(ctx, personStateSQL, spec.ID).Scan(&before); err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return caseidentity.Receipt{}, fmt.Errorf("%w: no person %s", caseidentity.ErrNotFound, spec.ID)
		}
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	for _, table := range []string{"entity", "person"} {
		sets, values := []string{}, []any{spec.ID}
		for _, column := range sortedKeys(spec.Fields) {
			if caseidentity.PersonColumns[column] != table {
				continue
			}
			values = append(values, spec.Fields[column])
			cast := ""
			if column == "is_minor" {
				cast = "::boolean"
			}
			// column comes from caseidentity.PersonColumns (ValidatePerson), never from input text.
			sets = append(sets, fmt.Sprintf("%s = $%d%s", column, len(values), cast))
		}
		if len(sets) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE registry.%s SET %s WHERE id = $1::uuid", table, strings.Join(sets, ", ")), values...); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
	}
	var after []byte
	if err := tx.QueryRow(ctx, personStateSQL, spec.ID).Scan(&after); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	at := s.clock()
	ref, err := insertChange(ctx, tx, "registry.person", spec.ID, before, after, spec.ChangeReason, actor, key, at)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: ref, Kind: "registry.identity_change", RecordedAt: at}, nil
}

// AddPerson creates a registry.entity (type person) and its registry.person.
func (s *CaseIdentityStore) AddPerson(ctx context.Context, spec caseidentity.NewPersonSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateNewPerson(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("new-person")
	tx, rollback, err := s.begin(ctx, "new-person")
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	if receipt, replayed, err := replayChange(ctx, tx, key, "registry.person", ""); err != nil || replayed {
		rollback()
		return receipt, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO registry.entity
		    (id, entity_type, display_name, canonical_name, data_tier, provenance, requires_human_review, review_status, safe_for_legal_use)
		VALUES ($1, 'person', $2, $2, 'analytical', ARRAY[ROW('postgres', $3::text, 'workbench.case_identity')::ai.source_ref],
		        false, 'approved', false)`, id, spec.DisplayName, id.String()); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO registry.person (id, role_in_case, connection_to, short_name, is_minor, notes, verification_state)
		VALUES ($1, $2, $3, $4, $5, $6, 'confirmed')`, id, spec.RoleInCase, spec.ConnectionTo, spec.ShortName, spec.IsMinor, spec.Notes); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	var after []byte
	if err := tx.QueryRow(ctx, personStateSQL, id.String()).Scan(&after); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	at := s.clock()
	if _, err := insertChange(ctx, tx, "registry.person", id.String(), []byte(`{}`), after, spec.ChangeReason, actor, key, at); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: id.String(), Kind: "registry.person", RecordedAt: at}, nil
}

// Triage records a dismiss / reopen decision on an identifier tied to nobody.
func (s *CaseIdentityStore) Triage(ctx context.Context, spec caseidentity.TriageSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateTriage(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("triage")
	tx, rollback, err := s.begin(ctx, "triage")
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	var ref, raw, decision string
	var at time.Time
	err = tx.QueryRow(ctx, `SELECT id::text, raw_value, decision, recorded_at FROM registry.identifier_triage WHERE idempotency_key = $1`, key).Scan(&ref, &raw, &decision, &at)
	switch {
	case err == nil:
		rollback()
		if raw != spec.RawValue || decision != spec.Decision {
			return caseidentity.Receipt{}, caseidentity.ErrIdempotencyConflict
		}
		return caseidentity.Receipt{Ref: ref, Kind: "registry.identifier_triage", RecordedAt: at, Replayed: true}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	at = s.clock()
	if _, err := tx.Exec(ctx, `INSERT INTO registry.identifier_triage (id, normalized, raw_value, decision, basis, recorded_by, idempotency_key, recorded_at)
		VALUES ($1, registry.norm_identifier($2), $2, $3, $4, $5, $6, $7)`,
		id, spec.RawValue, spec.Decision, spec.Basis, actor.Username, key, at); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: id.String(), Kind: "registry.identifier_triage", RecordedAt: at}, nil
}

const caseLookupSQL = `SELECT q.value, registry.norm_identifier(q.value), a.entity_id::text, v.person, a.alias_text::text,
       v.kind, a.status, a.period
FROM unnest($1::text[]) WITH ORDINALITY AS q(value, ord)
LEFT JOIN registry.entity_alias_current a ON a.normalized = registry.norm_identifier(q.value) AND a.status <> 'retired'
LEFT JOIN registry.vw_case_identifier v ON v.alias_id = a.id
ORDER BY q.ord, a.status, a.entity_id`

// Lookup answers which person used each identifier (current, not retired).
func (s *CaseIdentityStore) Lookup(ctx context.Context, values []string) ([]caseidentity.Match, error) {
	if err := caseidentity.ValidateLookup(values); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, caseLookupSQL, values)
	if err != nil {
		return nil, caseIdentityError(err)
	}
	defer rows.Close()
	out := []caseidentity.Match{}
	for rows.Next() {
		var m caseidentity.Match
		var normalized *string
		if err := rows.Scan(&m.Query, &normalized, &m.EntityID, &m.Person, &m.RawValue, &m.Kind, &m.Status, &m.Period); err != nil {
			return nil, err
		}
		if normalized != nil {
			m.Normalized = *normalized
		}
		out = append(out, m)
	}
	return out, caseIdentityError(rows.Err())
}

func sortedKeys(fields map[string]*string) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// caseIdentityError maps a missing object to ErrNotInstalled.
func caseIdentityError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42P01", "42703", "42883":
			return fmt.Errorf("%w: %s", caseidentity.ErrNotInstalled, pgErr.Message)
		case "42501":
			return fmt.Errorf("%w: the engine role lacks a registry grant (%s)", caseidentity.ErrNotInstalled, pgErr.Message)
		}
	}
	return err
}

// caseIdentityWriteError maps constraint failures to the page's errors.
func caseIdentityWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return caseidentity.ErrStale
		case "23514", "23502", "22P02", "22007", "22008", "23503", "22001":
			return fmt.Errorf("%w: %s", caseidentity.ErrRejected, pgErr.Message)
		}
	}
	return caseIdentityError(err)
}
