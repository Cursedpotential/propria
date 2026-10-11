// Byline: Codex · GPT-6.1-sol · 2026-10-11
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
)

const discoveryTestID = "11111111-1111-4111-8111-111111111111"
const discoveryTestGeneration = "22222222-2222-4222-8222-222222222222"

type discoveryTestRow struct {
	values []any
	err    error
}

func (r discoveryTestRow) Scan(dst ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, v := range r.values {
		switch d := dst[i].(type) {
		case *[]byte:
			*d = v.([]byte)
		case *bool:
			*d = v.(bool)
		case *int:
			*d = v.(int)
		}
	}
	return nil
}

type discoveryTestTx struct {
	pgx.Tx
	run             *discovery.Run
	feedback        *discovery.Feedback
	candidate       discovery.Candidate
	batch           discovery.Batch
	writes          int
	commits         int
	generationCount int
	queryCalls      int
}

func (x *discoveryTestTx) Commit(context.Context) error   { x.commits++; return nil }
func (x *discoveryTestTx) Rollback(context.Context) error { return nil }
func (x *discoveryTestTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "context.") || strings.Contains(sql, "working.") || strings.Contains(sql, "evidence.") {
		return pgconn.CommandTag{}, errors.New("attempted canonical input mutation")
	}
	x.writes++
	if strings.HasPrefix(sql, "INSERT INTO analysis.discovery_run") && x.run == nil {
		var r discovery.Run
		_ = json.Unmarshal(args[6].([]byte), &r)
		x.run = &r
	}
	if strings.HasPrefix(sql, "UPDATE analysis.discovery_run") {
		var r discovery.Run
		_ = json.Unmarshal(args[6].([]byte), &r)
		if x.run.Revision != args[8].(int64) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		x.run = &r
	}
	if strings.HasPrefix(sql, "INSERT INTO analysis.discovery_feedback") {
		var f discovery.Feedback
		_ = json.Unmarshal(args[4].([]byte), &f)
		x.feedback = &f
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (x *discoveryTestTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.HasPrefix(sql, "SELECT count(*) FROM context.normalized_generation") {
		return discoveryTestRow{values: []any{x.generationCount}}
	}
	if strings.HasPrefix(sql, discoveryRunRead) {
		if x.run == nil || x.run.Request.RunID != args[0].(string) || x.run.Request.Scope.MatterID != args[1].(string) || x.run.Request.Scope.CourtCaseID != args[2].(string) || x.run.Request.Scope.Mode != args[3].(string) {
			return discoveryTestRow{err: pgx.ErrNoRows}
		}
		raw, _ := json.Marshal(x.run)
		return discoveryTestRow{values: []any{raw}}
	}
	if strings.HasPrefix(sql, "SELECT payload FROM analysis.discovery_feedback") {
		if x.feedback == nil {
			return discoveryTestRow{err: pgx.ErrNoRows}
		}
		raw, _ := json.Marshal(x.feedback)
		return discoveryTestRow{values: []any{raw}}
	}
	if strings.HasPrefix(sql, "SELECT run_id=") {
		return discoveryTestRow{values: []any{x.run.Request.RunID == args[1].(string)}}
	}
	if strings.HasPrefix(sql, "SELECT c.payload,i.payload") {
		if args[2].(string) != x.candidate.CandidateID {
			return discoveryTestRow{err: pgx.ErrNoRows}
		}
		c, _ := json.Marshal(x.candidate)
		b, _ := json.Marshal(x.batch)
		return discoveryTestRow{values: []any{c, b}}
	}
	return discoveryTestRow{err: errors.New("unexpected SQL: " + sql)}
}
func (x *discoveryTestTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	x.queryCalls++
	return nil, errors.New("unexpected source query")
}

type discoveryTestDB struct{ *discoveryTestTx }

func (d discoveryTestDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return d.discoveryTestTx, nil
}
func discoveryFixture() (discovery.RunRequest, discovery.Dataset) {
	scope := discovery.Scope{Mode: "LIVE", MatterID: authoritativeMatterID, CourtCaseID: authoritativeCourtCaseID}
	r := discovery.RunRequest{RunID: discoveryTestID, Scope: scope, Actor: discovery.Actor{SubjectUID: "owner", Username: "owner"}, InputBackend: "postgresql", DatasetID: "test", SelectionRevision: "v1", GenerationIDs: []string{discoveryTestGeneration}, Filters: discovery.Filters{Perspective: "hindsight"}, MaxRecords: 100, MaxCandidates: 10}
	d := discovery.Dataset{Scope: scope, InputBackend: r.InputBackend, DatasetID: r.DatasetID, Label: "test", SelectionRevision: r.SelectionRevision, GenerationIDs: r.GenerationIDs}
	return r, d
}
func TestDiscoveryRunIdempotencyCrossScopeAndDevFence(t *testing.T) {
	ctx := context.Background()
	r, d := discoveryFixture()
	x := &discoveryTestTx{}
	s, _ := NewDiscoveryStore(discoveryTestDB{x}, []discovery.Dataset{d})
	first, e := s.CreateRun(ctx, r)
	if e != nil || !first.DispatchPending || first.Revision != 1 {
		t.Fatal(e, first)
	}
	second, e := s.CreateRun(ctx, r)
	if e != nil || second.Request.RunID != first.Request.RunID {
		t.Fatal("stable replay lost", e)
	}
	r.Actor.Username = "different"
	if _, e = s.CreateRun(ctx, r); !errors.Is(e, discovery.ErrConflict) {
		t.Fatal("different idempotent request accepted", e)
	}
	foreign := first.Request.Scope
	foreign.MatterID = discoveryTestGeneration
	if _, e = s.ReadRun(ctx, r.RunID, foreign); !errors.Is(e, discovery.ErrNotFound) {
		t.Fatal("foreign scope read leaked", e)
	}
	r, d = discoveryFixture()
	r.Scope.Mode = "DEV"
	before := x.writes
	if _, e = s.CreateRun(ctx, r); e == nil || x.writes != before {
		t.Fatal("DEV canonical write admitted", e)
	}
}
func TestDiscoveryFeedbackCASReplayAndWrongPersonNeutralTaxonomy(t *testing.T) {
	ctx := context.Background()
	r, d := discoveryFixture()
	x := &discoveryTestTx{}
	s, _ := NewDiscoveryStore(discoveryTestDB{x}, []discovery.Dataset{d})
	run, e := s.CreateRun(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	x.run.ScoreRevision = 1
	x.candidate = discovery.Candidate{CandidateID: "candidate", EpisodeID: "episode", Labels: []string{"threat"}}
	x.batch = discovery.Batch{Episodes: []discovery.Episode{{EpisodeID: "episode", RecordIDs: []string{discoveryTestID}}}, Records: []discovery.SourceRecord{{RecordID: discoveryTestID, Text: "original"}}}
	in := discovery.FeedbackRequest{FeedbackID: discoveryTestGeneration, CandidateID: "candidate", ExpectedRevision: run.Revision, Kind: "wrong_person", AttributedActorIDs: []string{"actual"}}
	saved, e := s.SubmitFeedback(ctx, r.RunID, r.Scope, in, r.Actor)
	if e != nil || saved.Run.FeedbackRevision != 1 || !saved.Run.DispatchPending || saved.Feedback.Kind != "wrong_person" || len(saved.Feedback.Labels) != 0 || len(saved.Feedback.PreviousLabels) != 1 || saved.Feedback.PreviousLabels[0] != "threat" {
		t.Fatal("attribution conflated with labels", e, saved)
	}
	before := x.writes
	replay, e := s.SubmitFeedback(ctx, r.RunID, r.Scope, in, r.Actor)
	if e != nil || x.writes != before || replay.Feedback.FeedbackID != saved.Feedback.FeedbackID {
		t.Fatal("replay double applied", e)
	}
	in.FeedbackID = discoveryTestID
	in.Kind = "skip"
	x.feedback = nil
	if _, e = s.SubmitFeedback(ctx, r.RunID, r.Scope, in, r.Actor); !errors.Is(e, discovery.ErrConflict) || x.writes != before {
		t.Fatal("stale CAS wrote state", e)
	}
	in.ExpectedRevision = saved.Run.Revision
	neutral, e := s.SubmitFeedback(ctx, r.RunID, r.Scope, in, r.Actor)
	if e != nil || neutral.Feedback.Kind != "skip" || len(neutral.Feedback.Labels) != 0 {
		t.Fatal("skip became label negative", e)
	}
}
func TestDiscoveryReaderRejectsCrossCaseGenerationBeforeBodies(t *testing.T) {
	r, d := discoveryFixture()
	x := &discoveryTestTx{generationCount: 0}
	reader, _ := NewDiscoverySourceReader(discoveryTestDB{x}, []discovery.Dataset{d}, nil)
	if _, _, e := reader.ReadSources(context.Background(), r); !errors.Is(e, discovery.ErrScope) || x.queryCalls != 0 {
		t.Fatal("foreign generation reached body reader", e)
	}
	for _, term := range []string{"UPDATE ", "INSERT ", "DELETE ", "evidence."} {
		if strings.Contains(DiscoverySourceSQL, term) {
			t.Fatal("reader admits input mutation/custody", term)
		}
	}
	filter := strings.Index(DiscoverySourceSQL, "source_available_from")
	limit := strings.LastIndex(DiscoverySourceSQL, " LIMIT ")
	if filter < 0 || filter > limit {
		t.Fatal("historical availability filter follows retrieval limit")
	}
}

func TestDiscoveryMergedMembershipIsScopedExplicitAndHorizonFiltered(t *testing.T) {
	for _, required := range []string{"working.message_occurrence", "message.derived_from_record_id", "membership.message_id=message.id", "thread.matter_id=$1::uuid AND thread.court_case_id=$2::uuid", "version.knowledge_available_from <= $6", "membership.source_available_from <= $6", "n.occurred_at <= $6"} {
		if !strings.Contains(DiscoverySourceSQL, required) {
			t.Fatal("missing explicit scoped mapping", required)
		}
	}
	raw := []byte(`{"record_id":"11111111-1111-4111-8111-111111111111","explicit_episode_id":"thread-version","group_revision":"recorded-assertion","working_message_id":"actual-message","source_available_from":"2026-01-01T00:00:00Z","payload":{"content":{"body":"hello","native_metadata":{"conversation_id":"imported-thread"}}}}`)
	record, e := decodeDiscoverySource(raw)
	if e != nil || record.ExplicitEpisodeID != "thread-version" || record.ConversationID != "imported-thread" || record.WorkingMessageID != "actual-message" || record.SourceAvailableFrom == nil {
		t.Fatal("canonical metadata lost", record, e)
	}
}
func TestDiscoveryFailurePersistsRestartAndRejectsStaleRevision(t *testing.T) {
	ctx := context.Background()
	request, dataset := discoveryFixture()
	state := &discoveryTestTx{}
	store, _ := NewDiscoveryStore(discoveryTestDB{state}, []discovery.Dataset{dataset})
	run, e := store.CreateRun(ctx, request)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.MarkFailed(ctx, request.RunID, request.Scope, 0, "scoring"); e != nil {
		t.Fatal(e)
	}
	restarted, _ := NewDiscoveryStore(discoveryTestDB{state}, []discovery.Dataset{dataset})
	saved, e := restarted.ReadRun(ctx, request.RunID, request.Scope)
	if e != nil || saved.Status != "failed" || saved.DispatchPending || saved.Revision != run.Revision+1 {
		t.Fatal("restart lost failure", saved, e)
	}
	before := state.writes
	if e = restarted.MarkFailed(ctx, request.RunID, request.Scope, 0, "scoring"); e != nil || state.writes != before {
		t.Fatal("failure replay double applied", e)
	}
	state.run.FeedbackRevision = 1
	if e = restarted.MarkFailed(ctx, request.RunID, request.Scope, 0, "scoring"); !errors.Is(e, discovery.ErrConflict) || state.writes != before {
		t.Fatal("stale execution overwrote correction", e)
	}
}

func TestDiscoveryDispatchPreservesCompletedAndFailedState(t *testing.T) {
	for _, status := range []string{"ready", "ready_partial", "failed"} {
		ctx := context.Background()
		request, dataset := discoveryFixture()
		state := &discoveryTestTx{}
		store, _ := NewDiscoveryStore(discoveryTestDB{state}, []discovery.Dataset{dataset})
		if _, e := store.CreateRun(ctx, request); e != nil {
			t.Fatal(e)
		}
		state.run.Status = status
		state.run.DispatchPending = false
		if e := store.RecordDispatch(ctx, request.RunID, request.Scope, 0, "real-workflow", "real-run"); e != nil || state.run.Status != status || state.run.DispatchPending || state.run.WorkflowID != "real-workflow" {
			t.Fatal("late dispatch replaced terminal state", status, e)
		}
	}
}

func TestDiscoveryImportedNavigationMatchesAttestedReaderIdentity(t *testing.T) {
	raw := []byte(`{"record_id":"11111111-1111-4111-8111-111111111111","record_type":"message","working_message_id":"different-working-id","imported_export_key":"/exports/phone.xml","imported_conversation_key":"+18005551234","payload":{"content":{"body":"hello","native_metadata":{"message_id":"native-external"}}}}`)
	record, e := decodeDiscoverySource(raw)
	if e != nil || record.ImportedMessageID != record.RecordID || record.ImportedSourceID != "L2V4cG9ydHMvcGhvbmUueG1s" || record.ImportedThreadID != "L2V4cG9ydHMvcGhvbmUueG1sCisxODAwNTU1MTIzNA" {
		t.Fatal("Imported reader navigation differs", record, e)
	}
	unlinked, e := decodeDiscoverySource([]byte(`{"record_id":"11111111-1111-4111-8111-111111111111","record_type":"message","imported_message_id":"external-spoof","payload":{"content":{"native_metadata":{"message_id":"native-id"}}}}`))
	if e != nil || unlinked.ImportedMessageID != "" || unlinked.ImportedThreadID != "" {
		t.Fatal("unattested navigation admitted", unlinked, e)
	}
	if !strings.Contains(DiscoverySourceSQL, "regexp_replace(s.source_key, '\\.derived/.*$', '')") || !strings.Contains(DiscoverySourceSQL, "regexp_replace(s.source_key, '^.*/', '')") {
		t.Fatal("source-key repository derivation missing")
	}
}
func TestDiscoveryHistoricalGroupingIgnoresUnknownAndFutureLinks(t *testing.T) {
	cutoff := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	future := cutoff.Add(time.Hour)
	prior := cutoff.Add(-time.Hour)
	filters := discovery.Filters{Perspective: "as_lived", Cutoff: &cutoff}
	for _, clock := range []*time.Time{nil, &future} {
		record := discovery.SourceRecord{ExplicitEpisodeID: "later-link", GroupRevision: "later-revision", ConversationID: "native-conversation", GroupKnowledgeAvailableFrom: clock}
		applyDiscoverySourceHorizon(&record, filters)
		if record.ExplicitEpisodeID != "" || record.GroupRevision != "" || record.ConversationID != "native-conversation" || len(record.ContextUncertainty) == 0 {
			t.Fatal("future assertion changed early episode", record)
		}
	}
	record := discovery.SourceRecord{ExplicitEpisodeID: "prior-version", GroupKnowledgeAvailableFrom: &prior}
	applyDiscoverySourceHorizon(&record, filters)
	if record.ExplicitEpisodeID != "prior-version" {
		t.Fatal("eligible explicit link lost")
	}
	if !strings.Contains(DiscoverySourceSQL, "later.supersedes_id=version.id AND ($6::timestamptz IS NULL OR later.knowledge_available_from <= $6)") {
		t.Fatal("future supersession hid eligible prior version")
	}
}
