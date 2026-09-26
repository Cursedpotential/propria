// Byline: Claude Code · Opus 5.5 · 2026-09-25

package activities

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// memStore mirrors the SQL semantics of postgres.EntityExtractionStore:
// inserts are ignored on an existing id, supersede is guarded, and a replay
// of already-written rows is a no-op.
type memStore struct {
	run          flow.RunRef
	participants []entities.ParticipantAggregate
	messages     []entities.MessageView
	runs         map[string]string
	entities     map[string]entities.Proposal
	entityOrder  []string
	events       map[string]events.Proposal
	eventOrder   []string
	registry     map[string]entities.RegistryEntity
	written      map[string]map[string]any
	collection   string
}

func newMemStore() *memStore {
	at := func(value string) *time.Time { parsed, _ := time.Parse(time.RFC3339, value); return &parsed }
	available := at("2026-09-21T01:14:39Z")
	store := &memStore{
		run:      flow.RunRef{PreviewHandle: "handle_abcdefghijklmnopqrstuvwxyz0123", GenerationID: "11111111-1111-4111-8111-111111111111", SourceVersionID: "22222222-2222-4222-8222-222222222222", MatterMode: "REAL"},
		runs:     map[string]string{},
		entities: map[string]entities.Proposal{}, events: map[string]events.Proposal{},
		registry: map[string]entities.RegistryEntity{}, written: map[string]map[string]any{},
	}
	bodies := []struct{ from, to, body, when string }{
		{"self", "+18105550101", "Morning Katherine, can you get Emma at 3?", "2025-06-01T14:46:00Z"},
		{"+18105550101", "self", "Yes. Court is on 2025-07-02", "2025-06-01T15:02:00Z"},
		{"self", "+18105550101", "Thanks Kat", "2025-06-01T15:10:00Z"},
	}
	for i, line := range bodies {
		store.messages = append(store.messages, entities.MessageView{
			RecordID: "3333333" + string(rune('0'+i)) + "-3333-4333-8333-333333333333", Ordinal: int64(i),
			OccurredAt: at(line.when), SourceAvailableFrom: available, Body: line.body,
			Participants: []entities.MessageAddressee{{Role: "sender", Identifier: line.from}, {Role: "recipient", Identifier: line.to}},
		})
	}
	store.participants = []entities.ParticipantAggregate{
		{Identifier: "+18105550101", AddressKind: "phone", NormalizedAddress: "+18105550101", MessageCount: 3, TotalMessages: 3, SenderCount: 1, RecipientCount: 2},
		{Identifier: "self", AddressKind: "self", NormalizedAddress: "self", MessageCount: 3, TotalMessages: 3, SenderCount: 2, RecipientCount: 1},
	}
	return store
}

func (s *memStore) write(table, id string, row any) bool {
	if s.written[table] == nil {
		s.written[table] = map[string]any{}
	}
	if _, ok := s.written[table][id]; ok {
		return false
	}
	s.written[table][id] = row
	return true
}

func (s *memStore) ResolveRun(_ context.Context, handle string) (flow.RunRef, error) {
	if handle != s.run.PreviewHandle {
		return flow.RunRef{}, service.ErrNotFound
	}
	run := s.run
	run.MatterMode = ""
	return run, nil
}
func (s *memStore) ParticipantAggregates(context.Context, string) ([]entities.ParticipantAggregate, error) {
	return s.participants, nil
}
func (s *memStore) MessagePage(_ context.Context, _ string, after int64, limit int) ([]entities.MessageView, error) {
	var out []entities.MessageView
	for _, message := range s.messages {
		if message.Ordinal > after && len(out) < limit {
			out = append(out, message)
		}
	}
	return out, nil
}
func (s *memStore) MessageByID(_ context.Context, _ string, id string) (entities.MessageView, error) {
	for _, message := range s.messages {
		if message.RecordID == id {
			return message, nil
		}
	}
	return entities.MessageView{}, service.ErrNotFound
}
func (s *memStore) RecordsMissing(_ context.Context, _ string, ids []string) ([]string, error) {
	var missing []string
	for _, id := range ids {
		if _, err := s.MessageByID(context.Background(), "", id); err != nil {
			missing = append(missing, id)
		}
	}
	return missing, nil
}
func (s *memStore) BeginRun(_ context.Context, run service.RunRow) error {
	if _, ok := s.runs[run.ID]; !ok {
		s.runs[run.ID] = "running"
	}
	return nil
}
func (s *memStore) FinishRun(_ context.Context, id, status string, _ map[string]any, _ string) error {
	if s.runs[id] == "running" {
		s.runs[id] = status
	}
	return nil
}
func (s *memStore) ExtractionRuns(context.Context, string) ([]service.RunSummary, error) {
	return nil, nil
}
func (s *memStore) StageEntities(ctx context.Context, runID string, proposals []entities.Proposal) ([]string, error) {
	ids := make([]string, len(proposals))
	for i, proposal := range proposals {
		ids[i] = flow.DeterministicID("candidate", runID, proposal.ContentHex())
	}
	return ids, s.ReplaceEntities(ctx, runID, ids, proposals, nil)
}
func (s *memStore) StageEvents(ctx context.Context, runID string, proposals []events.Proposal) ([]string, error) {
	ids := make([]string, len(proposals))
	for i, proposal := range proposals {
		ids[i] = flow.DeterministicID("candidate_event", runID, proposal.ContentHex())
	}
	return ids, s.ReplaceEvents(ctx, runID, ids, proposals, nil)
}
func (s *memStore) ReplaceEntities(_ context.Context, runID string, ids []string, insert []entities.Proposal, supersede []string) error {
	inserted := 0
	for i, proposal := range insert {
		if _, ok := s.entities[ids[i]]; ok {
			continue
		}
		proposal.CandidateID, proposal.ExtractionRunID = ids[i], runID
		if proposal.ReviewState == "" {
			proposal.ReviewState = entities.StatePending
		}
		s.entities[ids[i]] = proposal
		s.entityOrder = append(s.entityOrder, ids[i])
		inserted++
	}
	if len(insert) > 0 && inserted == 0 {
		return nil
	}
	for _, id := range supersede {
		current, ok := s.entities[id]
		if !ok || (current.ReviewState != entities.StatePending && current.ReviewState != entities.StateRejected) {
			return entities.ErrConflict
		}
		current.ReviewState = entities.StateSuperseded
		s.entities[id] = current
	}
	return nil
}
func (s *memStore) ReplaceEvents(_ context.Context, runID string, ids []string, insert []events.Proposal, supersede []string) error {
	inserted := 0
	for i, event := range insert {
		if _, ok := s.events[ids[i]]; ok {
			continue
		}
		event.CandidateID, event.ExtractionRunID = ids[i], runID
		if event.ReviewState == "" {
			event.ReviewState = entities.StatePending
		}
		s.events[ids[i]] = event
		s.eventOrder = append(s.eventOrder, ids[i])
		inserted++
	}
	if len(insert) > 0 && inserted == 0 {
		return nil
	}
	for _, id := range supersede {
		current, ok := s.events[id]
		if !ok || (current.ReviewState != entities.StatePending && current.ReviewState != entities.StateRejected) {
			return entities.ErrConflict
		}
		current.ReviewState = entities.StateSuperseded
		s.events[id] = current
	}
	return nil
}
func (s *memStore) CurrentEntities(context.Context, string) ([]entities.Proposal, error) {
	var out []entities.Proposal
	for _, id := range s.entityOrder {
		if p := s.entities[id]; p.ReviewState != entities.StateSuperseded {
			out = append(out, p)
		}
	}
	return out, nil
}
func (s *memStore) CurrentEvents(context.Context, string) ([]events.Proposal, error) {
	var out []events.Proposal
	for _, id := range s.eventOrder {
		if e := s.events[id]; e.ReviewState != entities.StateSuperseded {
			out = append(out, e)
		}
	}
	return out, nil
}
func (s *memStore) PendingEntitiesOfRuns(_ context.Context, runIDs []string) ([]entities.Proposal, error) {
	var out []entities.Proposal
	for _, id := range s.entityOrder {
		p := s.entities[id]
		for _, runID := range runIDs {
			if p.ExtractionRunID == runID && p.ReviewState == entities.StatePending {
				out = append(out, p)
			}
		}
	}
	return out, nil
}
func (s *memStore) Registry(_ context.Context, lookup service.RegistryLookup) ([]entities.RegistryEntity, error) {
	var out []entities.RegistryEntity
	for _, entity := range s.registry {
		out = append(out, entity)
	}
	return out, nil
}
func (s *memStore) SearchRegistry(context.Context, string, int) ([]entities.RegistryEntity, error) {
	return nil, nil
}
func (s *memStore) WriteEntities(_ context.Context, rows []service.EntityRow) (int, error) {
	n := 0
	for _, row := range rows {
		if s.write("entity", row.ID, row) {
			n++
			s.registry[row.ID] = entities.RegistryEntity{ID: row.ID, DisplayName: row.Name, RegistryType: entities.RegistryType(row.RegistryType), NormalizedName: row.NormalizedName}
		}
	}
	return n, nil
}
func (s *memStore) WriteAliases(_ context.Context, rows []service.AliasRow) (int, error) {
	n := 0
	for _, row := range rows {
		if _, ok := s.registry[row.EntityID]; !ok {
			return n, errors.New("alias of a missing entity (FK)")
		}
		if s.write("alias", row.ID, row) {
			n++
		}
	}
	return n, nil
}
func (s *memStore) WriteMentions(_ context.Context, mentions []service.MentionRow, resolutions []service.ResolutionRow, _ service.Reviewer, _ string) (int, int, error) {
	m, r := 0, 0
	for _, row := range mentions {
		if s.write("mention", row.ID, row) {
			m++
		}
	}
	for _, row := range resolutions {
		if s.write("resolution", row.ID, row) {
			r++
		}
	}
	return m, r, nil
}
func (s *memStore) WriteEventCandidates(_ context.Context, rows []service.EventCandidateRow, _ service.Reviewer) (int, error) {
	n := 0
	for _, row := range rows {
		if s.write("event_candidate", row.ID, row) {
			n++
		}
		s.write("anchor", row.AnchorID, row.AvailableFrom)
	}
	return n, nil
}
func (s *memStore) EnsureCollection(_ context.Context, slug, _ string) (string, error) {
	if s.collection == "" {
		s.collection = flow.DeterministicID("timeline_collection", slug)
	}
	return s.collection, nil
}
func (s *memStore) WriteTimelineMembers(_ context.Context, rows []service.TimelineMemberRow) (int, error) {
	n := 0
	for _, row := range rows {
		if s.write("member", row.ID, row) {
			n++
		}
	}
	return n, nil
}
func (s *memStore) FinalizeCommit(_ context.Context, receipt service.CommitReceipt, entityPromotions, eventPromotions []service.Promotion) error {
	s.write("receipt", receipt.ID, receipt)
	for _, promotion := range entityPromotions {
		if p := s.entities[promotion.CandidateID]; p.ReviewState == entities.StatePending {
			p.ReviewState, p.PromotedToID = entities.StateApproved, promotion.TargetID
			s.entities[promotion.CandidateID] = p
		}
	}
	for _, promotion := range eventPromotions {
		if e := s.events[promotion.CandidateID]; e.ReviewState == entities.StatePending {
			e.ReviewState, e.PromotedToID = entities.StateApproved, promotion.TargetID
			s.events[promotion.CandidateID] = e
		}
	}
	return nil
}

type scriptedModel struct{ reply string }

func (m scriptedModel) Complete(context.Context, []model.Message, model.CallOptions) (model.Completion, error) {
	return model.Completion{Content: m.reply, FinishReason: "stop"}, nil
}

const modelReply = `{
 "people": [
  {"name": "Katherine", "aliases": ["Kat"], "participant": "P2", "mentions": [{"message": "m1", "text": "Katherine"}, {"message": "m3", "text": "Kat"}]},
  {"name": "Emma", "aliases": [], "participant": null, "mentions": [{"message": "m1", "text": "Emma"}]}
 ],
 "places": [], "organizations": [],
 "events": [
  {"title": "Court date", "description": "P2 says court is on 2025-07-02", "message": "m2", "date": "2025-07-02", "when": null, "type": "court", "people": ["Katherine"], "places": [], "organizations": []}
 ]
}`

func activitiesFor(store *memStore) EntityExtractionActivities {
	return EntityExtractionActivities{Store: store, Model: scriptedModel{reply: modelReply}, ModelID: "test-model"}
}

var testActor = entities.Actor{SubjectUID: "uid-owner", Username: "owner"}

func extract(t *testing.T, store *memStore) {
	t.Helper()
	acts := activitiesFor(store)
	request := flow.ExtractionRequest{ExtractionID: "ext-1", Run: store.run, Actor: testActor, UseModel: true}
	rules, err := acts.ProposeEntitiesRules(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	modelResult, err := acts.ExtractEntitiesEventsModel(context.Background(), request)
	if err != nil || modelResult.Skipped {
		t.Fatalf("model: %v %+v", err, modelResult)
	}
	if _, err := acts.ReconcileEntityProposals(context.Background(), flow.ReconcileRequest{Extraction: request, RunIDs: []string{rules.ExtractionRunID, modelResult.ExtractionRunID}}); err != nil {
		t.Fatal(err)
	}
}

func currentByName(t *testing.T, store *memStore) map[string]entities.Proposal {
	t.Helper()
	current, _ := store.CurrentEntities(context.Background(), "")
	out := map[string]entities.Proposal{}
	for _, proposal := range current {
		out[proposal.Name] = proposal
	}
	return out
}

func TestExtractionProposesGroupsAndStagesEvents(t *testing.T) {
	store := newMemStore()
	extract(t, store)
	byName := currentByName(t, store)
	katherine, ok := byName["Katherine"]
	if !ok || len(byName) != 3 {
		t.Fatalf("want Katherine (number merged in), Emma, device owner; got %v", keys(byName))
	}
	var number, kat bool
	for _, alias := range katherine.Aliases {
		number = number || alias.Normalized == "+18105550101"
		kat = kat || alias.Text == "Kat"
	}
	if !number || !kat || katherine.MentionCount < 4 {
		t.Fatalf("Katherine must carry her number and nickname with supporting mentions: %+v", katherine)
	}
	currentEvents, _ := store.CurrentEvents(context.Background(), "")
	if len(currentEvents) != 1 || currentEvents[0].Title != "Court date" || !strings.Contains(currentEvents[0].Description, "+1 (810) 555-0101") {
		t.Fatalf("event not staged with its label resolved: %+v", currentEvents)
	}
	// Re-running the same extraction is idempotent: no new current rows.
	before := len(store.entityOrder)
	extract(t, store)
	if len(currentByName(t, store)) != 3 {
		t.Fatalf("a repeated extraction must not duplicate proposals: %v", keys(currentByName(t, store)))
	}
	_ = before
}

func keys(values map[string]entities.Proposal) []string {
	var out []string
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func TestCorrectionsAndOwnerMarkThroughTheService(t *testing.T) {
	store := newMemStore()
	extract(t, store)
	emma := currentByName(t, store)["Emma"]
	now := time.Date(2026, 9, 25, 21, 0, 0, 0, time.UTC)
	envelope := service.CorrectionEnvelope{Target: "entity", Entity: &entities.CorrectionRequest{Op: entities.OpRename, CandidateIDs: []string{emma.CandidateID}, Name: "Emma D."}}
	if err := service.Correct(context.Background(), store, store.run, envelope, testActor, now, "digest-rename"); err != nil {
		t.Fatal(err)
	}
	// The same request again (a retried HTTP call) is a no-op, not a conflict.
	if err := service.Correct(context.Background(), store, store.run, envelope, testActor, now, "digest-rename"); err != nil {
		t.Fatalf("a replayed correction must succeed: %v", err)
	}
	renamed, ok := currentByName(t, store)["Emma D."]
	if !ok || renamed.Correction == nil || renamed.Correction.Actor != testActor || !renamed.Correction.At.Equal(now) {
		t.Fatalf("rename not attributed: %+v", renamed)
	}
	if _, stale := currentByName(t, store)["Emma"]; stale {
		t.Fatal("the renamed row must supersede the original")
	}
	marked, err := service.MarkEvent(context.Background(), store, store.run, events.MarkRequest{RecordID: store.messages[0].RecordID, Title: "Pickup at 3"}, testActor, now, "digest-mark")
	if err != nil {
		t.Fatal(err)
	}
	if marked.DetectedBy != entities.DetectedOwner || !marked.OccurredAt.Equal(*store.messages[0].OccurredAt) {
		t.Fatalf("owner mark must be dated by its message: %+v", marked)
	}
	again, err := service.MarkEvent(context.Background(), store, store.run, events.MarkRequest{RecordID: store.messages[0].RecordID, Title: "Pickup at 3"}, testActor, now, "digest-mark")
	if err != nil || again.CandidateID != marked.CandidateID {
		t.Fatalf("a replayed mark returns the same event: %v %s %s", err, again.CandidateID, marked.CandidateID)
	}
}

func TestCommitIsIdempotentGuardedAndPromotes(t *testing.T) {
	store := newMemStore()
	extract(t, store)
	acts := activitiesFor(store)
	report, err := service.Validate(context.Background(), store, store.run)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("validation failed: %+v", report.Checks)
	}
	request := flow.CommitRequest{CommitID: "commit-1", Run: store.run, Actor: testActor, Digest: report.Digest, RequestedAt: time.Now().Add(-time.Minute), CollectionSlug: "primary"}
	validated, err := acts.ValidateExtractionCommit(context.Background(), request)
	if err != nil || !validated.Report.OK || validated.Report.Digest != report.Digest {
		t.Fatalf("activity validation disagrees: %v %+v", err, validated.Report)
	}
	steps := []func(context.Context, flow.CommitRequest) (flow.CommitStepResult, error){
		acts.CommitEntities, acts.CommitEntityAliases, acts.CommitEntityMentions, acts.CommitEventCandidates, acts.CommitTimelineMembers,
	}
	first := make([]int, len(steps))
	for i, step := range steps {
		result, err := step(context.Background(), request)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		first[i] = result.Written
	}
	if first[0] != 3 || first[1] < 2 || first[2] < 7 || first[3] != 1 || first[4] != 1 {
		t.Fatalf("first run wrote %v (entities, aliases, mentions, events, members)", first)
	}
	for i, step := range steps {
		result, err := step(context.Background(), request)
		if err != nil || result.Written != 0 {
			t.Fatalf("retry of step %d must write nothing: %v %+v", i, err, result)
		}
	}
	// The event names Katherine; its entity ref is her committed registry id.
	for _, row := range store.written["event_candidate"] {
		event := row.(service.EventCandidateRow)
		if len(event.EntityRefs) != 1 || !strings.HasPrefix(event.EntityRefs[0], service.EntityRefPrefix) {
			t.Fatalf("event entity refs = %v", event.EntityRefs)
		}
		if !event.AvailableFrom.Equal(*store.messages[1].SourceAvailableFrom) {
			t.Fatalf("event availability must be its source's: %v", event.AvailableFrom)
		}
	}
	// "self" never becomes a registry alias.
	for _, row := range store.written["alias"] {
		if strings.EqualFold(row.(service.AliasRow).Text, "self") {
			t.Fatal(`"self" was written as a registry alias`)
		}
	}
	if _, err := acts.FinalizeExtractionCommit(context.Background(), flow.FinalizeRequest{Commit: request, Outcome: flow.OutcomeCommitted, Promote: true}); err != nil {
		t.Fatal(err)
	}
	for _, proposal := range currentByName(t, store) {
		if proposal.ReviewState != entities.StateApproved || proposal.PromotedToID == "" {
			t.Fatalf("committed proposals must be promoted: %+v", proposal)
		}
	}
	if len(store.written["receipt"]) != 1 {
		t.Fatalf("one receipt per commit: %d", len(store.written["receipt"]))
	}
}

func TestCommitRefusesProposalsChangedAfterValidation(t *testing.T) {
	store := newMemStore()
	extract(t, store)
	report, _ := service.Validate(context.Background(), store, store.run)
	emma := currentByName(t, store)["Emma"]
	envelope := service.CorrectionEnvelope{Target: "entity", Entity: &entities.CorrectionRequest{Op: entities.OpReject, CandidateIDs: []string{emma.CandidateID}}}
	if err := service.Correct(context.Background(), store, store.run, envelope, testActor, time.Now(), "late-edit"); err != nil {
		t.Fatal(err)
	}
	_, err := activitiesFor(store).CommitEntities(context.Background(), flow.CommitRequest{Run: store.run, Digest: report.Digest, Actor: testActor})
	if err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("a commit must refuse an edited proposal set, got %v", err)
	}
}

func TestTestModeRunValidatesButCannotCommit(t *testing.T) {
	store := newMemStore()
	extract(t, store)
	run := store.run
	run.MatterMode = "TEST"
	report, err := service.Validate(context.Background(), store, run)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatal("TEST-mode proposals must not validate for commit")
	}
	var failed []string
	for _, check := range report.Checks {
		if check.Status == commitcheck.Fail {
			failed = append(failed, check.Rule)
		}
	}
	if strings.Join(failed, ",") != "live_mode" {
		t.Fatalf("only the mode rule should fail: %v", failed)
	}
}

func TestModelActivitySkipsWithoutAModel(t *testing.T) {
	store := newMemStore()
	result, err := EntityExtractionActivities{Store: store, ModelUnavailable: "ENTITY_MODEL_API_KEY_FILE is not mounted"}.ExtractEntitiesEventsModel(context.Background(), flow.ExtractionRequest{ExtractionID: "x", Run: store.run})
	if err != nil || !result.Skipped || !strings.Contains(result.Reason, "not mounted") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
