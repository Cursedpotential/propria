// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package cfjobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// memCatalog is an in-memory Catalog: a candidate list plus what the jobs have recorded.
type memCatalog struct {
	mu         sync.Mutex
	candidates map[Job][]WorkItem
	sniff      map[string]SniffRow
	archives   map[string]ZipArchive
	members    map[string][]ZipMember
	hashes     map[string]HashResult
}

func newMemCatalog() *memCatalog {
	return &memCatalog{
		candidates: map[Job][]WorkItem{}, sniff: map[string]SniffRow{}, archives: map[string]ZipArchive{},
		members: map[string][]ZipMember{}, hashes: map[string]HashResult{},
	}
}

func (m *memCatalog) done(job Job, key string) bool {
	switch job {
	case JobSniff:
		r, ok := m.sniff[key]
		return ok && r.Error == ""
	case JobZip:
		a, ok := m.archives[key]
		return ok && a.Error == "" && !a.Truncated
	default:
		h, ok := m.hashes[key]
		return ok && h.Error == ""
	}
}

func (m *memCatalog) list(job Job, scope Scope, after string) []WorkItem {
	var out []WorkItem
	for _, item := range m.candidates[job] {
		if item.Key > after && !m.done(job, item.Key) && (len(scope.Keys) == 0 || contains(scope.Keys, item.Key)) {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (m *memCatalog) NextBatch(_ context.Context, in NextBatchInput) (NextBatchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := m.list(in.Job, in.Scope, in.After)
	if len(items) > in.Limit {
		items = items[:in.Limit]
	}
	res := NextBatchResult{Items: items}
	if n := len(items); n > 0 {
		res.Last = items[n-1].Key
	}
	if res.Items == nil {
		res.Items = []WorkItem{}
	}
	return res, nil
}

func (m *memCatalog) Plan(_ context.Context, in PlanInput) (int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	items := m.list(in.Job, in.Scope, "")
	if in.MaxItems > 0 && len(items) > in.MaxItems {
		items = items[:in.MaxItems]
	}
	var bytes int64
	for _, it := range items {
		bytes += it.Size
	}
	return int64(len(items)), bytes, nil
}

func (m *memCatalog) UpsertSniff(_ context.Context, _ string, _ Scope, _ string, rows []SniffRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range rows {
		m.sniff[r.Key] = r
	}
	return nil
}

func (m *memCatalog) UpsertZipMembers(_ context.Context, _ string, _ Scope, key string, rows []ZipMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.members[key]
	for _, r := range rows {
		for int64(len(cur)) <= r.Index {
			cur = append(cur, ZipMember{Index: -1})
		}
		cur[r.Index] = r
	}
	m.members[key] = cur
	return nil
}

func (m *memCatalog) UpsertZipArchive(_ context.Context, _ string, _ Scope, a ZipArchive) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.Error == "" && int64(len(m.members[a.Key])) > a.EntriesListed {
		m.members[a.Key] = m.members[a.Key][:a.EntriesListed]
	}
	m.archives[a.Key] = a
	return nil
}

func (m *memCatalog) UpsertHash(_ context.Context, _ string, _ Scope, r HashResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hashes[r.Key] = r
	return nil
}

// workerServer is a stand-in Worker: it checks the bearer token and answers the three Workers' wire formats.
func workerServer(t *testing.T, requests *[]map[string]any) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		*requests = append(*requests, body)
		mu.Unlock()
		keys := body["keys"].([]any)
		switch r.URL.Path {
		case "/sniff":
			var results []map[string]any
			for _, k := range keys {
				key := k.(map[string]any)["key"].(string)
				if strings.Contains(key, "missing") {
					results = append(results, map[string]any{"key": key, "bytes_read": 0, "error": "not found"})
					continue
				}
				results = append(results, map[string]any{"key": key, "bytes_read": 100, "format": "ai_markdown_transcript",
					"signature_kind": "ai_speaker_markers_v1", "confidence": 0.78, "rule_source": "handler_ai_chat_signature.go"})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ruleset": body["ruleset"], "results": results})
		case "/list":
			for _, k := range keys {
				key := k.(map[string]any)["key"].(string)
				if strings.Contains(key, "bad") {
					fmt.Fprintf(w, `{"type":"error","key":%q,"error":"not a zip: no end-of-central-directory record"}`+"\n", key)
					continue
				}
				fmt.Fprintf(w, `{"type":"members","key":%q,"rows":[{"index":0,"name":"a.json","comp_size":5,"size":9,"crc32":"deadbeef","method":8},{"index":1,"name":"b/","comp_size":0,"size":0,"crc32":"00000000","method":0,"is_dir":true}]}`+"\n", key)
				fmt.Fprintf(w, `{"type":"archive","key":%q,"size":999,"entries_declared":2,"entries_listed":2,"cd_offset":900,"cd_size":99,"zip64":false,"truncated":false,"requests":2,"bytes_read":70000}`+"\n", key)
			}
			fmt.Fprintln(w, `{"type":"done","archives":1,"errors":0}`)
		case "/hash":
			for _, k := range keys {
				key := k.(string)
				fmt.Fprintf(w, `{"type":"progress","key":%q,"bytes":10}`+"\n", key)
				if strings.Contains(key, "gone") {
					fmt.Fprintf(w, `{"type":"error","key":%q,"error":"not found"}`+"\n", key)
					continue
				}
				fmt.Fprintf(w, `{"type":"result","key":%q,"size":42,"sha1":%q,"sha256":%q,"bytes_hashed":42,"b2_content_sha1":"none","b2_file_id":"f1","b2_sha1_mismatch":false,"ms":5}`+"\n",
					key, strings.Repeat("a", 40), strings.Repeat("b", 64))
			}
			fmt.Fprintln(w, `{"type":"done","hashed":1,"errors":0}`)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newActivities(t *testing.T, cat Catalog, srv *httptest.Server, token string) *Activities {
	c := &WorkerClient{BaseURL: srv.URL, Token: token}
	return &Activities{Catalog: cat, Sniffer: c, ZipLister: c, B2Hasher: c}
}

func TestSniffFormatsRecordsRowsAndErrors(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	a := newActivities(t, cat, srv, "secret")
	counts, err := a.SniffFormats(context.Background(), SniffBatch{
		RunID: "run1", Items: []WorkItem{{"chat/one.txt", 100}, {"chat/missing.txt", 5}}, HeadBytes: 16384,
	})
	require.NoError(t, err)
	require.Equal(t, Counts{Objects: 2, Failed: 1, Bytes: 100}, counts)
	require.Equal(t, "ai_markdown_transcript", cat.sniff["chat/one.txt"].Format)
	require.EqualValues(t, 100, *cat.sniff["chat/one.txt"].Size, "size comes from the batch when the Worker leaves it null")
	require.Equal(t, "not found", cat.sniff["chat/missing.txt"].Error)
	require.Equal(t, "b2", reqs[0]["provider"])
	require.Equal(t, "salem-data", reqs[0]["bucket"])
	require.Equal(t, "proffer-v1", reqs[0]["ruleset"])
	require.EqualValues(t, 16384, reqs[0]["head_bytes"])
}

func TestSniffFormatsRejectsBadBatchesAndBadToken(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	a := newActivities(t, newMemCatalog(), srv, "secret")
	_, err := a.SniffFormats(context.Background(), SniffBatch{})
	require.ErrorContains(t, err, "sniff batch must hold")
	big := make([]WorkItem, SniffBatchMax+1)
	_, err = a.SniffFormats(context.Background(), SniffBatch{Items: big})
	require.ErrorContains(t, err, "sniff batch must hold")
	wrong := newActivities(t, newMemCatalog(), srv, "nope")
	_, err = wrong.SniffFormats(context.Background(), SniffBatch{Items: []WorkItem{{"a", 1}}})
	require.ErrorContains(t, err, "HTTP 401")
	require.Empty(t, reqs, "a refused call reached no handler")
}

func TestListZipMembersStreamsIntoCatalog(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	a := newActivities(t, cat, srv, "secret")
	counts, err := a.ListZipMembers(context.Background(), ZipBatch{RunID: "r", Items: []WorkItem{{"z/a.zip", 999}, {"z/bad.zip", 50}}, MaxMembers: 100})
	require.NoError(t, err)
	require.Equal(t, 2, counts.Objects)
	require.Equal(t, 1, counts.Failed)
	require.Equal(t, 2, counts.Members)
	require.Len(t, cat.members["z/a.zip"], 2)
	require.Equal(t, "b/", cat.members["z/a.zip"][1].Name)
	require.True(t, cat.members["z/a.zip"][1].IsDir)
	require.Equal(t, int64(70000), cat.archives["z/a.zip"].BytesRead)
	require.Contains(t, cat.archives["z/bad.zip"].Error, "not a zip")
	require.EqualValues(t, 100, reqs[0]["max_members"])
}

func TestListZipMembersFailsWhenTheStreamIsCut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"type":"archive","key":"z/a.zip","size":1,"entries_declared":0,"entries_listed":0}`) // no done line
	}))
	defer srv.Close()
	a := newActivities(t, newMemCatalog(), srv, "")
	_, err := a.ListZipMembers(context.Background(), ZipBatch{Items: []WorkItem{{"z/a.zip", 1}}})
	require.ErrorContains(t, err, "without its done line")
}

func TestHashB2ObjectsRecordsDigestsAndErrors(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	a := newActivities(t, cat, srv, "secret")
	counts, err := a.HashB2Objects(context.Background(), HashBatch{RunID: "r", Items: []WorkItem{{"big/one.zip", 42}, {"big/gone.zip", 7}}})
	require.NoError(t, err)
	require.Equal(t, Counts{Objects: 2, Failed: 1, Bytes: 42}, counts)
	require.Equal(t, strings.Repeat("a", 40), cat.hashes["big/one.zip"].SHA1)
	require.Equal(t, strings.Repeat("b", 64), cat.hashes["big/one.zip"].SHA256)
	require.Equal(t, "not found", cat.hashes["big/gone.zip"].Error)
	_, err = a.HashB2Objects(context.Background(), HashBatch{Scope: Scope{Provider: "r2"}, Items: []WorkItem{{"x", 1}}})
	require.ErrorContains(t, err, "only reads B2")
	_, err = a.HashB2Objects(context.Background(), HashBatch{Items: make([]WorkItem, HashBatchMax+1)})
	require.ErrorContains(t, err, "hash batch must hold")
}

func TestPlanCountsBatches(t *testing.T) {
	cat := newMemCatalog()
	for i := 0; i < 250; i++ {
		cat.candidates[JobSniff] = append(cat.candidates[JobSniff], WorkItem{fmt.Sprintf("k%04d", i), 10})
	}
	a := &Activities{Catalog: cat}
	plan, err := a.Plan(context.Background(), PlanInput{Job: JobSniff, BatchSize: 100})
	require.NoError(t, err)
	require.Equal(t, Plan{Job: JobSniff, Objects: 250, Bytes: 2500, Batches: 3, BatchSize: 100, WorkerRequests: 3}, plan)
	_, err = a.Plan(context.Background(), PlanInput{Job: JobSniff})
	require.Error(t, err)
}

// register wires the workflows and Activities into a Temporal test environment.
func register(env *testsuite.TestWorkflowEnvironment, a *Activities) {
	env.RegisterWorkflowWithOptions(SniffFormatsWorkflow, workflow.RegisterOptions{Name: SniffFormatsWorkflowName})
	env.RegisterWorkflowWithOptions(ListZipMembersWorkflow, workflow.RegisterOptions{Name: ListZipMembersWorkflowName})
	env.RegisterWorkflowWithOptions(BackfillB2HashesWorkflow, workflow.RegisterOptions{Name: BackfillB2HashesWorkflowName})
	env.RegisterActivityWithOptions(a.NextBatch, activity.RegisterOptions{Name: NextBatchActivityName})
	env.RegisterActivityWithOptions(a.Plan, activity.RegisterOptions{Name: PlanActivityName})
	env.RegisterActivityWithOptions(a.SniffFormats, activity.RegisterOptions{Name: SniffFormatsActivityName})
	env.RegisterActivityWithOptions(a.ListZipMembers, activity.RegisterOptions{Name: ListZipMembersActivityName})
	env.RegisterActivityWithOptions(a.HashB2Objects, activity.RegisterOptions{Name: HashB2ObjectsActivityName})
}

func TestSniffWorkflowPagesTheListAndResumes(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	for i := 0; i < 450; i++ {
		cat.candidates[JobSniff] = append(cat.candidates[JobSniff], WorkItem{fmt.Sprintf("chat/%04d.txt", i), 1000})
	}
	cat.candidates[JobSniff] = append(cat.candidates[JobSniff], WorkItem{"chat/zz-missing.txt", 10})
	a := newActivities(t, cat, srv, "secret")

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(SniffFormatsWorkflowName, SniffInput{JobInput: JobInput{BatchSize: 100}})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res JobResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.EqualValues(t, 451, res.Plan.Objects)
	require.EqualValues(t, 5, res.Plan.Batches)
	require.Equal(t, 5, res.Batches)
	require.Equal(t, 451, res.Totals.Objects)
	require.Equal(t, 1, res.Totals.Failed)
	require.Len(t, cat.sniff, 451)

	// Running it again resumes: the only object still without a clean row is the one that failed.
	reqs = nil
	env2 := suite.NewTestWorkflowEnvironment()
	register(env2, a)
	env2.ExecuteWorkflow(SniffFormatsWorkflowName, SniffInput{JobInput: JobInput{BatchSize: 100}})
	require.NoError(t, env2.GetWorkflowError())
	var again JobResult
	require.NoError(t, env2.GetWorkflowResult(&again))
	require.EqualValues(t, 1, again.Plan.Objects)
	require.Len(t, reqs, 1)
}

func TestWorkflowsDryRunCallNoWorkerAndWriteNothing(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	cat.candidates[JobZip] = []WorkItem{{"z/a.zip", 100}, {"z/b.zip", 200}}
	cat.candidates[JobHash] = []WorkItem{{"big/x.bin", 1 << 30}}
	a := newActivities(t, cat, srv, "secret")
	var suite testsuite.WorkflowTestSuite

	env := suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(ListZipMembersWorkflowName, ZipInput{JobInput: JobInput{DryRun: true}})
	require.NoError(t, env.GetWorkflowError())
	var res JobResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.True(t, res.DryRun)
	require.EqualValues(t, 2, res.Plan.Objects)
	require.EqualValues(t, 300, res.Plan.Bytes)
	require.EqualValues(t, 1, res.Plan.WorkerRequests)
	require.Zero(t, res.Batches)

	env = suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(BackfillB2HashesWorkflowName, HashInput{JobInput: JobInput{DryRun: true}})
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(&res))
	require.EqualValues(t, 1<<30, res.Plan.Bytes)
	require.EqualValues(t, 1, res.Plan.Batches)

	require.Empty(t, reqs, "a dry run never calls a Worker")
	require.Empty(t, cat.archives)
	require.Empty(t, cat.hashes)
}

func TestZipAndHashWorkflowsRunToTheCatalog(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	cat.candidates[JobZip] = []WorkItem{{"z/a.zip", 999}, {"z/b.zip", 999}, {"z/c.zip", 999}}
	cat.candidates[JobHash] = []WorkItem{{"big/1", 42}, {"big/2", 42}}
	a := newActivities(t, cat, srv, "secret")
	var suite testsuite.WorkflowTestSuite

	env := suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(ListZipMembersWorkflowName, ZipInput{JobInput: JobInput{BatchSize: 2, MaxItems: 3}})
	require.NoError(t, env.GetWorkflowError())
	var res JobResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, 2, res.Batches)
	require.Len(t, cat.archives, 3)

	env = suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(BackfillB2HashesWorkflowName, HashInput{})
	require.NoError(t, env.GetWorkflowError())
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, 2, res.Batches, "one object per Worker call by default")
	require.Len(t, cat.hashes, 2)
	require.EqualValues(t, 84, res.Totals.Bytes)
}

func TestMaxItemsStopsTheJob(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	for i := 0; i < 50; i++ {
		cat.candidates[JobSniff] = append(cat.candidates[JobSniff], WorkItem{fmt.Sprintf("k%03d.txt", i), 10})
	}
	a := newActivities(t, cat, srv, "secret")
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(SniffFormatsWorkflowName, SniffInput{JobInput: JobInput{BatchSize: 20, MaxItems: 30}})
	require.NoError(t, env.GetWorkflowError())
	var res JobResult
	require.NoError(t, env.GetWorkflowResult(&res))
	require.Equal(t, 30, res.Totals.Objects)
	require.Len(t, cat.sniff, 30)
}

func TestWorkflowFailsOnAWrongToken(t *testing.T) {
	var reqs []map[string]any
	srv := workerServer(t, &reqs)
	defer srv.Close()
	cat := newMemCatalog()
	cat.candidates[JobSniff] = []WorkItem{{"a.txt", 1}}
	a := newActivities(t, cat, srv, "wrong")
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	register(env, a)
	env.ExecuteWorkflow(SniffFormatsWorkflowName, SniffInput{})
	require.Error(t, env.GetWorkflowError())
	require.Contains(t, env.GetWorkflowError().Error(), "HTTP 401")
	require.Empty(t, cat.sniff)
}

func TestLoadConfigReportsEverythingMissing(t *testing.T) {
	for _, name := range []string{"TEMPORAL_HOST_PORT", "TEMPORAL_NAMESPACE", "CASEBIBLE_DATABASE_URL_FILE", "CF_SNIFFER_URL", "CF_SNIFFER_TOKEN_FILE",
		"CF_ZIP_LISTER_URL", "CF_ZIP_LISTER_TOKEN_FILE", "CF_B2_HASHER_URL", "CF_B2_HASHER_TOKEN_FILE", "CF_TASK_QUEUE"} {
		t.Setenv(name, "")
	}
	_, err := LoadConfig()
	require.ErrorContains(t, err, "TEMPORAL_HOST_PORT is required")
	require.ErrorContains(t, err, "CF_B2_HASHER_TOKEN_FILE is required")
	for _, name := range []string{"TEMPORAL_HOST_PORT", "TEMPORAL_NAMESPACE", "CASEBIBLE_DATABASE_URL_FILE", "CF_SNIFFER_URL", "CF_SNIFFER_TOKEN_FILE",
		"CF_ZIP_LISTER_URL", "CF_ZIP_LISTER_TOKEN_FILE", "CF_B2_HASHER_URL", "CF_B2_HASHER_TOKEN_FILE"} {
		t.Setenv(name, "x")
	}
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, QueueName, cfg.TaskQueue)
}
