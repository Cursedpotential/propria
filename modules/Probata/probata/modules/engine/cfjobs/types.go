// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

// Package cfjobs drives the Case Bible's three Cloudflare bulk-read Workers from Temporal: the format sniffer, the ZIP
// member lister and the B2 hash backfill (casebible/tools/cf_workers/).
//
// The Workers read next to the data (ranged GETs, central directories, streamed hashes) and keep nothing. Each Activity
// here is one Worker call: a batch of object keys goes in, the Worker's answer is written to the Case Bible catalog
// (PostgreSQL database `casebible`, schema `raw_duck`, tables cf_*_20261002) with idempotent upserts, and only counts
// come back, so no result rows travel through Temporal history. Each workflow is one job: it pages its work list out of
// the catalog, runs the batches, and can be started as a dry run that only plans.
//
// This package is its own worker (queue QueueName). The proffer worker's queue is deliberately not shared: a worker that
// registers only part of a queue's Activities must not poll it (engine/temporal/worker.go).
package cfjobs

import "time"

// QueueName is the Temporal task queue of the cf-jobs worker.
const QueueName = "casebible-cf-v1"

// Workflow type names, as registered with Temporal.
const (
	SniffFormatsWorkflowName     = "SniffFormatsWorkflow"
	ListZipMembersWorkflowName   = "ListZipMembersWorkflow"
	BackfillB2HashesWorkflowName = "BackfillB2HashesWorkflow"
)

// Activity names, as registered with Temporal.
const (
	NextBatchActivityName      = "cf_next_batch_activity"
	PlanActivityName           = "cf_plan_activity"
	SniffFormatsActivityName   = "cf_sniff_formats_activity"
	ListZipMembersActivityName = "cf_list_zip_members_activity"
	HashB2ObjectsActivityName  = "cf_hash_b2_objects_activity"
)

// Job names which of the three work lists an Activity reads.
type Job string

// The three jobs.
const (
	JobSniff Job = "sniff"
	JobZip   Job = "zip"
	JobHash  Job = "hash"
)

// Defaults and limits. The Worker caps are the ones enforced in cf_workers/*/src/index.js.
const (
	DefaultProvider = "b2"
	DefaultBucket   = "salem-data"
	DefaultRuleset  = "proffer-v1"

	SniffBatchDefault = 100
	SniffBatchMax     = 200
	ZipBatchDefault   = 10
	ZipBatchMax       = 50
	HashBatchDefault  = 1
	HashBatchMax      = 4

	sniffTimeout = 5 * time.Minute
	zipTimeout   = 20 * time.Minute
	hashTimeout  = 6 * time.Hour
	// hashHeartbeat is longer than the Worker's 15 s progress line, so a live hash never times out its heartbeat.
	hashHeartbeat = 2 * time.Minute
)

// WorkItem is one object of a work list: its key and the size the catalog lists for it.
type WorkItem struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
}

// Scope names the objects a job looks at.
type Scope struct {
	// Provider is "b2" or "r2". Empty means DefaultProvider.
	Provider string `json:"provider,omitempty"`
	// Bucket is empty for DefaultBucket.
	Bucket string `json:"bucket,omitempty"`
	// Keys limits the job to these object keys; empty means the job's whole catalog work list.
	Keys []string `json:"keys,omitempty"`
}

func (s Scope) withDefaults() Scope {
	if s.Provider == "" {
		s.Provider = DefaultProvider
	}
	if s.Bucket == "" {
		s.Bucket = DefaultBucket
	}
	return s
}

// JobInput is what every job workflow takes.
type JobInput struct {
	Scope
	// DryRun plans the job (counts, bytes, batches, requests) and stops: no Worker is called and the catalog is not written.
	DryRun bool `json:"dry_run,omitempty"`
	// BatchSize is the objects per Worker call; 0 means the job's default, and the Worker's cap applies.
	BatchSize int `json:"batch_size,omitempty"`
	// Parallel is the Worker calls in flight; 0 means 3 (1 for the hash job).
	Parallel int `json:"parallel,omitempty"`
	// MaxItems stops the job after this many objects; 0 means the whole list.
	MaxItems int `json:"max_items,omitempty"`
}

// SniffInput starts SniffFormatsWorkflow.
type SniffInput struct {
	JobInput
	// Ruleset is "proffer-v1" (default) or "casebible-probe-v1".
	Ruleset string `json:"ruleset,omitempty"`
	// HeadBytes is the bytes read from the start of each object, 8192..262144; 0 means 65536.
	HeadBytes int `json:"head_bytes,omitempty"`
}

// ZipInput starts ListZipMembersWorkflow.
type ZipInput struct {
	JobInput
	// MaxMembers bounds the members listed per archive; 0 means the Worker's default of 200000.
	MaxMembers int `json:"max_members,omitempty"`
}

// HashInput starts BackfillB2HashesWorkflow.
type HashInput struct {
	JobInput
}

// Plan is what a dry run reports, and what every run starts with.
type Plan struct {
	Job     Job   `json:"job"`
	Objects int64 `json:"objects"`
	// Bytes is the sum of the listed object sizes (what the job would read for a hash; far less for a sniff or a ZIP list).
	Bytes     int64 `json:"bytes"`
	Batches   int64 `json:"batches"`
	BatchSize int   `json:"batch_size"`
	// WorkerRequests is the number of Worker calls the run makes (one per batch).
	WorkerRequests int64 `json:"worker_requests"`
}

// NextBatchInput asks for the next page of a job's work list, keyset-paged by key.
type NextBatchInput struct {
	Job   Job    `json:"job"`
	Scope Scope  `json:"scope"`
	After string `json:"after"`
	Limit int    `json:"limit"`
	// Ruleset only matters to the sniff job: an object is done per ruleset.
	Ruleset string `json:"ruleset,omitempty"`
}

// NextBatchResult is one page; an empty Items means the list is exhausted.
type NextBatchResult struct {
	Items []WorkItem `json:"items"`
	// Last is the key to pass as After for the following page.
	Last string `json:"last"`
}

// PlanInput asks the catalog how big a job's work list is.
type PlanInput struct {
	Job       Job    `json:"job"`
	Scope     Scope  `json:"scope"`
	BatchSize int    `json:"batch_size"`
	Ruleset   string `json:"ruleset,omitempty"`
	MaxItems  int    `json:"max_items,omitempty"`
}

// Counts is what a Worker-call Activity returns: how many objects it recorded, and how many of those carry an error.
type Counts struct {
	Objects int `json:"objects"`
	// Failed objects were recorded with their error text in the catalog; the batch itself succeeded.
	Failed int `json:"failed"`
	// Members is the ZIP members recorded (zip job only).
	Members int `json:"members,omitempty"`
	// Bytes is the object bytes the Worker reported reading or hashing.
	Bytes int64 `json:"bytes,omitempty"`
}

func (c *Counts) add(o Counts) {
	c.Objects += o.Objects
	c.Failed += o.Failed
	c.Members += o.Members
	c.Bytes += o.Bytes
}

// SniffBatch is one sniffer Worker call.
type SniffBatch struct {
	RunID     string     `json:"run_id"`
	Scope     Scope      `json:"scope"`
	Ruleset   string     `json:"ruleset"`
	HeadBytes int        `json:"head_bytes"`
	Items     []WorkItem `json:"items"`
}

// ZipBatch is one ZIP lister Worker call.
type ZipBatch struct {
	RunID      string     `json:"run_id"`
	Scope      Scope      `json:"scope"`
	MaxMembers int        `json:"max_members"`
	Items      []WorkItem `json:"items"`
}

// HashBatch is one B2 hasher Worker call.
type HashBatch struct {
	RunID string     `json:"run_id"`
	Scope Scope      `json:"scope"`
	Items []WorkItem `json:"items"`
}

// JobResult is what a job workflow returns.
type JobResult struct {
	RunID  string `json:"run_id"`
	DryRun bool   `json:"dry_run"`
	Plan   Plan   `json:"plan"`
	// Batches is the Worker calls made (0 for a dry run).
	Batches int    `json:"batches"`
	Totals  Counts `json:"totals"`
}
