// Byline: Claude Code · Sonnet · 2026-10-02
//
// ContactsImportWorkflow: from the catalog's contact exports to a person for every number, as ONE traceable
// Temporal workflow (owner 2026-10-02 20:02: "make sure all these things get run as Temporal activities and
// are traceable"). Each step is its own Activity with a Receipt; the workflow only sequences them, carries
// references between them and never touches a byte of contact data. DryRun is a workflow input: the
// read-only steps (manifest, fetch, parse) run for real either way, and the three steps that write the
// registry (people, placeholders, re-link) run inside a transaction that is rolled back.
//
//  1. contacts_manifest_activity      query raw_duck.b2_objects, dedupe by sha1, write manifest.jsonl
//  2. contacts_fetch_activity         copy each object from B2 into the work folder, verifying its sha1
//  3. contacts_parse_activity         parse vCard / CSV / Facebook-Instagram JSON, group into people
//  4. contacts_people_activity        contact-people through the governed case-identity store (unconfirmed
//     people named by the most recent export, other names as candidates)
//  5. contacts_placeholders_activity  a placeholder only for numbers still carried by NO person
//  6. contacts_relink_activity        sweep: fill every NULL entity column whose number a person now carries
//
// Order is the owner's (15:34): contacts first; placeholders only for what no contact names.
package contacts

import (
	"errors"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Names registered with Temporal. They are the trace: each appears in the workflow's history.
const (
	WorkflowName    = "contacts_import_workflow"
	StatusQueryName = "contacts_import_status"

	ManifestActivity     = "contacts_manifest_activity"
	FetchActivity        = "contacts_fetch_activity"
	ParseActivity        = "contacts_parse_activity"
	PeopleActivity       = "contacts_people_activity"
	PlaceholdersActivity = "contacts_placeholders_activity"
	RelinkActivity       = "contacts_relink_activity"
)

// Actor is the authenticated owner who started the run. It becomes the actor of every identity_change row.
type Actor struct {
	SubjectUID string `json:"subject_uid"`
	Username   string `json:"username"`
}

// Input starts one run.
type Input struct {
	// RunID names the run: its work folder, its idempotency keys and its workflow id suffix.
	RunID string `json:"run_id"`
	// DryRun rolls back every registry write and reports the counts the real run would produce.
	DryRun bool  `json:"dry_run"`
	Actor  Actor `json:"actor"`
	// KeyStrip is removed from catalog keys before fetching (default b2://salem-data/).
	KeyStrip string `json:"key_strip,omitempty"`
	// Bucket is the B2 bucket the keys live in (default salem-data).
	Bucket string `json:"bucket,omitempty"`
}

// StepRequest is what every Activity receives: the run, the actor, and the references from earlier steps.
type StepRequest struct {
	RunID    string            `json:"run_id"`
	DryRun   bool              `json:"dry_run"`
	Actor    Actor             `json:"actor"`
	KeyStrip string            `json:"key_strip,omitempty"`
	Bucket   string            `json:"bucket,omitempty"`
	Refs     map[string]string `json:"refs,omitempty"`
}

// Receipt is the durable answer of one step: what it did, in counts, and where its output is.
type Receipt struct {
	Step   string           `json:"step"`
	DryRun bool             `json:"dry_run"`
	Status string           `json:"status"` // success | not_applicable
	Ref    string           `json:"ref,omitempty"`
	Digest string           `json:"digest,omitempty"`
	Counts map[string]int64 `json:"counts,omitempty"`
	Notes  []string         `json:"notes,omitempty"`
	At     time.Time        `json:"at"`
}

// RunStatus is what the status query returns while the run goes on and after it ends.
type RunStatus struct {
	RunID   string    `json:"run_id"`
	DryRun  bool      `json:"dry_run"`
	State   string    `json:"state"` // running | completed | failed
	Current string    `json:"current,omitempty"`
	Steps   []Receipt `json:"steps"`
	Error   string    `json:"error,omitempty"`
}

// Result is the workflow's return value.
type Result struct {
	RunID    string    `json:"run_id"`
	DryRun   bool      `json:"dry_run"`
	Receipts []Receipt `json:"receipts"`
}

func options(timeout, heartbeat time.Duration, attempts int32) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		HeartbeatTimeout:    heartbeat,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 5 * time.Second, BackoffCoefficient: 2, MaximumInterval: 2 * time.Minute, MaximumAttempts: attempts,
			NonRetryableErrorTypes: []string{"ContactsPermanent"},
		},
	}
}

// stepOptions gives each step its own bounded retry policy and timeouts (never the SDK default of unlimited).
var stepOptions = map[string]workflow.ActivityOptions{
	ManifestActivity:     options(5*time.Minute, 0, 3),
	FetchActivity:        options(60*time.Minute, 2*time.Minute, 3),
	ParseActivity:        options(15*time.Minute, 2*time.Minute, 2),
	PeopleActivity:       options(60*time.Minute, 2*time.Minute, 2),
	PlaceholdersActivity: options(60*time.Minute, 2*time.Minute, 2),
	RelinkActivity:       options(30*time.Minute, 2*time.Minute, 2),
}

// ValidateInput refuses a run that cannot be traced to a person.
func ValidateInput(in Input) error {
	if strings.TrimSpace(in.RunID) == "" || len(in.RunID) > 64 {
		return errors.New("contacts import requires a run_id of at most 64 characters")
	}
	for _, r := range in.RunID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return errors.New("run_id may hold only letters, digits, - and _")
		}
	}
	if strings.TrimSpace(in.Actor.SubjectUID) == "" || strings.TrimSpace(in.Actor.Username) == "" {
		return errors.New("contacts import requires the owner's actor (subject_uid and username)")
	}
	return nil
}

// ContactsImportWorkflow is WorkflowName.
func ContactsImportWorkflow(ctx workflow.Context, in Input) (Result, error) {
	if err := ValidateInput(in); err != nil {
		return Result{}, temporal.NewNonRetryableApplicationError(err.Error(), "ContactsPermanent", err)
	}
	status := RunStatus{RunID: in.RunID, DryRun: in.DryRun, State: "running"}
	if err := workflow.SetQueryHandler(ctx, StatusQueryName, func() (RunStatus, error) { return status, nil }); err != nil {
		return Result{}, err
	}
	refs := map[string]string{}
	run := func(step string, carry ...string) (Receipt, error) {
		status.Current = step
		request := StepRequest{RunID: in.RunID, DryRun: in.DryRun, Actor: in.Actor, KeyStrip: in.KeyStrip, Bucket: in.Bucket, Refs: copyRefs(refs)}
		var receipt Receipt
		err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, stepOptions[step]), step, request).Get(ctx, &receipt)
		if err != nil {
			status.State, status.Error = "failed", err.Error()
			return Receipt{}, err
		}
		status.Steps = append(status.Steps, receipt)
		if receipt.Ref != "" && len(carry) > 0 {
			refs[carry[0]] = receipt.Ref
		}
		return receipt, nil
	}
	for _, step := range []struct{ name, carries string }{
		{ManifestActivity, "manifest"},
		{FetchActivity, "files"},
		{ParseActivity, "people"},
		{PeopleActivity, ""},
		{PlaceholdersActivity, ""},
		{RelinkActivity, ""},
	} {
		var carry []string
		if step.carries != "" {
			carry = []string{step.carries}
		}
		if _, err := run(step.name, carry...); err != nil {
			return Result{RunID: in.RunID, DryRun: in.DryRun, Receipts: status.Steps}, err
		}
	}
	status.State, status.Current = "completed", ""
	return Result{RunID: in.RunID, DryRun: in.DryRun, Receipts: status.Steps}, nil
}

func copyRefs(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
