// Byline: Codex · GPT-6.1-sol · 2026-10-06
package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type scopeRegistryRow struct {
	ctx        context.Context
	id, parent string
	err        error
}

func (r scopeRegistryRow) Scan(dest ...any) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*string)) = r.id
	if len(dest) == 13 {
		*(dest[1].(*string)) = r.parent
	}
	return nil
}

type scopeRegistryTx struct {
	pgx.Tx                                // Any unimplemented transaction method is a tripwire, not a fallback.
	t                                     *testing.T
	deadline                              time.Time
	matter, court, parent                 string
	matterErr, courtErr, execErr          error
	queries, aggregates, execs, rollbacks int
	blockQuery, blockCleanup              bool
	cleanupDeadline                       time.Time
	cleanupWasCanceled                    bool
}

func (tx *scopeRegistryTx) checkContext(ctx context.Context) {
	tx.t.Helper()
	deadline, ok := ctx.Deadline()
	require.True(tx.t, ok)
	require.Equal(tx.t, tx.deadline, deadline) // A statement cannot reset the total timeout.
}

func (tx *scopeRegistryTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.checkContext(ctx)
	tx.execs++
	require.Equal(tx.t, scopeStatementTimeoutSQL, sql)
	require.Empty(tx.t, args)
	return pgconn.CommandTag{}, tx.execErr
}

func (tx *scopeRegistryTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx.checkContext(ctx)
	tx.queries++
	require.Equal(tx.t, []any{authoritativeMatterID}, args)
	if tx.blockQuery {
		<-ctx.Done()
		return scopeRegistryRow{ctx: ctx}
	}
	switch sql {
	case `SELECT ` + caseMatterColumns + ` FROM registry.matter m WHERE m.id = $1::uuid`:
		return scopeRegistryRow{ctx: ctx, id: tx.matter, err: tx.matterErr}
	case caseCourtCaseSQL:
		require.Contains(tx.t, sql, "c.id = '"+authoritativeCourtCaseID+"'::uuid")
		require.Contains(tx.t, sql, "c.matter_id = $1::uuid")
		return scopeRegistryRow{ctx: ctx, id: tx.court, parent: tx.parent, err: tx.courtErr}
	default:
		tx.t.Fatalf("scope reached a non-header query: %s", sql)
		return scopeRegistryRow{ctx: ctx, err: errors.New("unexpected query")}
	}
}

func (tx *scopeRegistryTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	tx.aggregates++
	tx.t.Fatal("scope must never query people, aliases, history, counts, unknowns or dismissals")
	return nil, errors.New("unexpected aggregate")
}

func (tx *scopeRegistryTx) Rollback(ctx context.Context) error {
	tx.rollbacks++
	tx.cleanupDeadline, _ = ctx.Deadline()
	tx.cleanupWasCanceled = ctx.Err() != nil
	remaining := time.Until(tx.cleanupDeadline)
	require.Greater(tx.t, remaining, time.Duration(0))
	require.LessOrEqual(tx.t, remaining, scopeCleanupTimeout)
	if tx.blockCleanup {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

type scopeRegistryDB struct {
	DB         // A pool query outside the transaction is a tripwire.
	t          *testing.T
	tx         *scopeRegistryTx
	begins     int
	blockBegin bool
	beginErr   error
}

func (db *scopeRegistryDB) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	db.begins++
	deadline, ok := ctx.Deadline()
	require.True(db.t, ok, "pool acquisition must already be bounded")
	require.LessOrEqual(db.t, time.Until(deadline), caseidentity.ScopeReadTimeout)
	require.Equal(db.t, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead}, options)
	db.tx.deadline = deadline
	if db.blockBegin {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return db.tx, db.beginErr
}

func newScopeRegistryStore(t *testing.T) (*CaseIdentityStore, *scopeRegistryDB) {
	t.Helper()
	tx := &scopeRegistryTx{t: t, matter: authoritativeMatterID, court: authoritativeCourtCaseID, parent: authoritativeMatterID}
	db := &scopeRegistryDB{t: t, tx: tx}
	store, err := NewCaseIdentityStore(db)
	require.NoError(t, err)
	return store, db
}

func TestCaseScopeStoreFreshModesReadOnlyExactRegistryPair(t *testing.T) {
	store, db := newScopeRegistryStore(t)
	for _, mode := range []caseidentity.Mode{"", caseidentity.ModeDev, caseidentity.ModeLive} {
		view, err := store.ReadScope(context.Background(), mode)
		require.NoError(t, err)
		wantMode := mode
		if wantMode == "" {
			wantMode = caseidentity.ModeLive
		}
		require.Equal(t, caseidentity.ScopeView{Mode: wantMode, Matter: caseidentity.ScopeMatter{ID: db.tx.matter}, CourtCase: caseidentity.ScopeCourtCase{ID: db.tx.court, MatterID: db.tx.parent}}, view)
	}
	require.Equal(t, 3, db.begins)
	require.Equal(t, 3, db.tx.execs)
	require.Equal(t, 6, db.tx.queries)
	require.Equal(t, 3, db.tx.rollbacks)
	require.Zero(t, db.tx.aggregates)
}

func TestCaseScopeStoreUnknownModeNeverAcquiresPool(t *testing.T) {
	store, db := newScopeRegistryStore(t)
	view, err := store.ReadScope(context.Background(), "unknown")
	require.Error(t, err)
	require.Empty(t, view.Matter.ID)
	require.Zero(t, db.begins)
}

func TestCaseScopeStoreMissingForeignAndUnavailableRowsDenyWithoutFallback(t *testing.T) {
	for _, failure := range []string{"matter-absent", "court-absent", "foreign-matter", "foreign-court", "foreign-parent", "begin-unavailable", "statement-unavailable", "matter-unavailable", "court-unavailable"} {
		t.Run(failure, func(t *testing.T) {
			store, db := newScopeRegistryStore(t)
			want := caseidentity.ErrNotFound
			unavailable := errors.New("database unavailable")
			switch failure {
			case "matter-absent":
				db.tx.matterErr = pgx.ErrNoRows
			case "court-absent":
				db.tx.courtErr = pgx.ErrNoRows
			case "foreign-matter":
				db.tx.matter = "11111111-1111-1111-1111-111111111111"
			case "foreign-court":
				db.tx.court = "11111111-1111-1111-1111-111111111111"
			case "foreign-parent":
				db.tx.parent = "11111111-1111-1111-1111-111111111111"
			case "begin-unavailable":
				db.beginErr = unavailable
				want = unavailable
			case "statement-unavailable":
				db.tx.execErr = unavailable
				want = unavailable
			case "matter-unavailable":
				db.tx.matterErr = unavailable
				want = unavailable
			case "court-unavailable":
				db.tx.courtErr = unavailable
				want = unavailable
			}
			view, err := store.ReadScope(context.Background(), caseidentity.ModeLive)
			require.ErrorIs(t, err, want)
			require.Equal(t, caseidentity.ScopeView{}, view)
			require.Zero(t, db.tx.aggregates)
			if db.beginErr == nil {
				require.Equal(t, 1, db.tx.rollbacks)
			}
		})
	}
}

func TestCaseScopeStoreTotalDeadlineIncludesPoolWait(t *testing.T) {
	store, db := newScopeRegistryStore(t)
	db.blockBegin = true
	started := time.Now()
	view, err := store.ReadScope(context.Background(), caseidentity.ModeLive)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, caseidentity.ScopeView{}, view)
	require.WithinDuration(t, started.Add(caseidentity.ScopeReadTimeout), time.Now(), 500*time.Millisecond)
	require.Zero(t, db.tx.execs)
	require.Zero(t, db.tx.queries)
	require.Zero(t, db.tx.rollbacks)
}

func TestCaseScopeStoreQueryDeadlineAndCleanupStayBelowClientTimeout(t *testing.T) {
	store, db := newScopeRegistryStore(t)
	db.tx.blockQuery, db.tx.blockCleanup = true, true
	started := time.Now()
	view, err := store.ReadScope(context.Background(), caseidentity.ModeLive)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, caseidentity.ScopeView{}, view)
	require.WithinDuration(t, started.Add(caseidentity.ScopeReadTimeout+scopeCleanupTimeout), time.Now(), 500*time.Millisecond)
	require.Equal(t, 1, db.tx.queries)
	require.Equal(t, 1, db.tx.rollbacks)
	require.False(t, db.tx.cleanupWasCanceled, "cleanup needs its own short live context after request deadline")
	require.Zero(t, db.tx.aggregates)
}

func TestCaseScopeStoreEarlierCallerDeadlineIsPreserved(t *testing.T) {
	store, db := newScopeRegistryStore(t)
	db.blockBegin = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	_, err := store.ReadScope(ctx, caseidentity.ModeLive)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, deadline, db.tx.deadline)
}

func TestCaseScopeSharedHeaderKeepsFullReadSQLAndWriteGuards(t *testing.T) {
	// The shared unit remains exactly the full-page sibling's original two SELECTs.
	store, db := newScopeRegistryStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), caseidentity.ScopeReadTimeout)
	defer cancel()
	db.tx.deadline, _ = ctx.Deadline()
	matter, court, err := store.readIdentityHeader(ctx, db.tx, caseidentity.ModeDev)
	require.NoError(t, err)
	require.Equal(t, authoritativeMatterID, matter.ID)
	require.Equal(t, authoritativeCourtCaseID, court.ID)
	require.Equal(t, 2, db.tx.queries)
	require.Zero(t, db.tx.execs)
	require.Zero(t, db.tx.aggregates)
	require.True(t, strings.Contains(caseCourtCaseSQL, authoritativeCourtCaseID))
	_, err = store.EditHeader(context.Background(), caseidentity.ModeDev, caseidentity.HeaderSpec{}, caseidentity.Actor{})
	require.ErrorIs(t, err, caseidentity.ErrDevWrite)
	require.Zero(t, db.begins)
}

type fullCaseEmptyRows struct{ pgx.Rows }

func (fullCaseEmptyRows) Next() bool { return false }
func (fullCaseEmptyRows) Close()     {}
func (fullCaseEmptyRows) Err() error { return nil }

type fullCaseRegressionTx struct {
	*scopeRegistryTx
	pageQueries []string
}

func (tx *fullCaseRegressionTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.checkContext(ctx)
	require.Equal(tx.t, `SET LOCAL statement_timeout = '8s'`, sql)
	require.Empty(tx.t, args)
	return pgconn.CommandTag{}, nil
}
func (tx *fullCaseRegressionTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	tx.pageQueries = append(tx.pageQueries, sql)
	return fullCaseEmptyRows{}, nil
}
func (tx *fullCaseRegressionTx) Rollback(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	require.True(tx.t, ok)
	require.LessOrEqual(tx.t, time.Until(deadline), 5*time.Second)
	tx.rollbacks++
	return nil
}

type fullCaseRegressionDB struct {
	DB
	tx *fullCaseRegressionTx
}

func (db *fullCaseRegressionDB) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	require.Equal(db.tx.t, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead}, options)
	db.tx.deadline, _ = ctx.Deadline()
	return db.tx, nil
}

func TestCaseScopeExtractionLeavesFullCasePageReadUnchanged(t *testing.T) {
	_, scopeDB := newScopeRegistryStore(t)
	tx := &fullCaseRegressionTx{scopeRegistryTx: scopeDB.tx}
	store, err := NewCaseIdentityStore(&fullCaseRegressionDB{tx: tx})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	view, err := store.Read(ctx, caseidentity.ModeLive)
	require.NoError(t, err)
	require.Equal(t, authoritativeMatterID, view.Matter.ID)
	require.Equal(t, authoritativeCourtCaseID, view.CourtCase.ID)
	require.Equal(t, 2, tx.queries)
	require.Equal(t, []string{casePeopleSQL, caseHistorySQL, caseUnknownsSQL, caseDismissedSQL}, tx.pageQueries)
	require.Equal(t, "probata", view.CountStore)
	require.NotNil(t, view.People)
	require.NotNil(t, view.History)
	require.NotNil(t, view.Counts)
	require.NotNil(t, view.Unknowns)
	require.NotNil(t, view.Dismissed)
	require.Equal(t, 1, tx.rollbacks)
}
