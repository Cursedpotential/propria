// Byline: Codex · GPT-6.1-sol · 2026-10-06
package postgres

import (
	"context"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
)

const scopeStatementTimeoutSQL = `SET LOCAL statement_timeout = '1500ms'`
const scopeCleanupTimeout = 250 * time.Millisecond

var _ caseidentity.ScopeReader = (*CaseIdentityStore)(nil)

// readIdentityHeader reads the same two registry rows used by the full Case page.
// Inputs: transaction queryer and validated mode. Outputs: matter and approved court rows.
// Side effects: two SELECTs only. Use for both full-page and bounded scope reads.
func (s *CaseIdentityStore) readIdentityHeader(ctx context.Context, q queryer, mode caseidentity.Mode) (*caseidentity.Matter, *caseidentity.CourtCase, error) {
	matterID, err := s.matterID(ctx, q, mode)
	if err != nil {
		return nil, nil, err
	}
	var matter *caseidentity.Matter
	var court *caseidentity.CourtCase
	if matterID != "" {
		matter, err = scanMatter(q.QueryRow(ctx, `SELECT `+caseMatterColumns+` FROM registry.matter m WHERE m.id = $1::uuid`, matterID))
		if err != nil {
			return matter, nil, err
		}
		court, err = scanCourtCase(q.QueryRow(ctx, caseCourtCaseSQL, matterID))
		if err != nil {
			return matter, court, err
		}
	}
	if matter == nil || court == nil {
		return matter, court, caseidentity.ErrNotFound
	}
	return matter, court, nil
}

// ReadScope reads the authoritative identity in a short read-only registry transaction.
// Inputs: request context and DEV/LIVE mode (omitted defaults LIVE at the parser boundary).
// Outputs: only registry IDs, or an error when rows are missing, foreign or unavailable.
// Side effects: bounded pool acquisition and two SELECTs; no people, history or counts.
// Choose over Read for fresh scope approval; it never substitutes configuration for rows.
func (s *CaseIdentityStore) ReadScope(ctx context.Context, mode caseidentity.Mode) (caseidentity.ScopeView, error) {
	mode, err := caseidentity.ParseMode(string(mode))
	if err != nil {
		return caseidentity.ScopeView{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, caseidentity.ScopeReadTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return caseidentity.ScopeView{}, caseIdentityError(err)
	}
	defer func() {
		cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), scopeCleanupTimeout)
		defer cleanupCancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err := tx.Exec(ctx, scopeStatementTimeoutSQL); err != nil {
		return caseidentity.ScopeView{}, caseIdentityError(err)
	}
	matter, court, err := s.readIdentityHeader(ctx, tx, mode)
	if err != nil {
		return caseidentity.ScopeView{}, err
	}
	if err := ctx.Err(); err != nil {
		return caseidentity.ScopeView{}, err
	}
	if !caseidentity.AdmittedIdentity(matter.ID, court.ID) || court.MatterID != matter.ID {
		return caseidentity.ScopeView{}, caseidentity.ErrNotFound
	}
	return caseidentity.ScopeView{Mode: mode, Matter: caseidentity.ScopeMatter{ID: matter.ID}, CourtCase: caseidentity.ScopeCourtCase{ID: court.ID, MatterID: court.MatterID}}, nil
}
