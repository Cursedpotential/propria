// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

const (
	testHandle = "HandleHandleHandleHandleHandle0123456789_-"
	testSource = "b2://salem-data/consignatio/vault/v1/sms-20250617122400.xml"
	testMatter = "deadbeef-dead-beef-dead-beefdeadbeef"
	testCourt  = "cafebabe-cafe-babe-cafe-babecafebabe"
	realMatter = "01a03136-c5cc-71c7-ac77-5c00a29a2ea8"
	realCourt  = "01a03136-c5cc-76f9-98df-702058d423d9"
)

var (
	findID    = string(stagegraph.RepairFindOtherVersion)
	salvageID = string(stagegraph.RepairSalvageTruncatedXML)
	lenientID = string(stagegraph.RepairLenientDecode)
)

type fakeAnchors struct {
	anchor Anchor
	err    error
}

func (f *fakeAnchors) ResolveAnchor(_ context.Context, sourceRef, handle string) (Anchor, error) {
	if f.err != nil {
		return Anchor{}, f.err
	}
	if handle != "" && handle != f.anchor.PreviewHandle {
		return Anchor{}, ErrAnchorNotFound
	}
	if handle == "" && sourceRef != f.anchor.SourceRef {
		return Anchor{}, ErrAnchorNotFound
	}
	return f.anchor, nil
}

func testAnchor() Anchor {
	return Anchor{
		PreviewHandle: testHandle, RequestID: "req-1", WorkflowID: "req-1", SourceRef: testSource,
		SourceVersionID: "0199aaaa-0000-7000-8000-000000000001", DeclaredFormat: "smsbackuprestore_xml",
		ParserOptionsRef: "pending-handler-selection/v1", MatterID: testMatter, CourtCaseID: testCourt,
		RepairDetection: json.RawMessage(liveDetection), RepairReport: json.RawMessage(truncatedReport),
	}
}

func testModes(matterID, courtCaseID string) (string, bool) {
	switch {
	case matterID == testMatter && courtCaseID == testCourt:
		return ModeTest, true
	case matterID == realMatter && courtCaseID == realCourt:
		return ModeReal, true
	}
	return "", false
}

func testEnv(t *testing.T) Environment {
	t.Helper()
	roots, err := objectstores.ParseRoots(`[{"id":"b2-bucket","label":"B2","url":"b2://salem-data/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	return Environment{
		Registry: DefaultRegistry(), Anchors: &fakeAnchors{anchor: testAnchor()},
		Stores: objectstores.Stores{"b2": "/run/secrets/b2.json"}, SourceRoots: roots, MatterMode: testModes,
	}
}

func testPlan(activities ...string) Plan {
	handle := testHandle
	plan := Plan{PlanID: "plan-0001-abcd", SourceRef: testSource, PreviewHandle: &handle, MatterMode: ModeTest, Steps: []Step{}}
	for index, activity := range activities {
		plan.Steps = append(plan.Steps, Step{StepID: fmt.Sprintf("s%d", index+1), Activity: activity, Params: json.RawMessage(`{}`)})
	}
	return plan
}

func validate(t *testing.T, env Environment, plan Plan) ValidatedPlan {
	t.Helper()
	validated, err := env.Validate(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	return validated
}

func rule(t *testing.T, validated ValidatedPlan, name string) Check {
	t.Helper()
	for _, check := range validated.Checks {
		if check.Rule == name {
			return check
		}
	}
	t.Fatalf("no %q check in %+v", name, validated.Checks)
	return Check{}
}

func requirePass(t *testing.T, validated ValidatedPlan, name string) {
	t.Helper()
	if check := rule(t, validated, name); check.Status != CheckPass || check.Reason == "" {
		t.Fatalf("%s should pass: %+v", name, check)
	}
}

func requireFail(t *testing.T, validated ValidatedPlan, name, reasonContains string) {
	t.Helper()
	check := rule(t, validated, name)
	if check.Status != CheckFail || !strings.Contains(check.Reason, reasonContains) {
		t.Fatalf("%s should fail with %q: %+v", name, reasonContains, check)
	}
	if validated.OK {
		t.Fatalf("a plan with a failed %s check was OK", name)
	}
}

func TestAValidPlanPassesEveryNamedRule(t *testing.T) {
	validated := validate(t, testEnv(t), testPlan(findID, salvageID))
	wantRules := []string{RuleAnchor, RuleRegistered, RuleParams, RuleTypeChain, RuleOriginalNotHit, RuleReentry, RuleDestination, RuleTestLive, RuleBounded}
	if !validated.OK || len(validated.Checks) != len(wantRules) {
		t.Fatalf("validated = %+v", validated.Checks)
	}
	for index, name := range wantRules {
		if validated.Checks[index].Rule != name || validated.Checks[index].Status != CheckPass {
			t.Fatalf("check %d = %+v, want %s pass", index, validated.Checks[index], name)
		}
	}
	if validated.SourceType != TypeSMSBackupXML || len(validated.Steps) != 2 ||
		validated.Steps[1].InputType != TypeSMSBackupXML || validated.Steps[1].OutputKind != OutputDerivedObject ||
		validated.Steps[1].StartToCloseSeconds != 4*3600 || validated.Steps[1].HeartbeatSeconds != 120 {
		t.Fatalf("resolved steps = %+v", validated.Steps)
	}
	if validated.Reentry != (Reentry{Kind: ReentrySingleRun, DeclaredFormat: "smsbackuprestore_xml",
		ParserOptionsRef: "pending-handler-selection/v1", MatterID: testMatter, CourtCaseID: testCourt}) {
		t.Fatalf("reentry = %+v", validated.Reentry)
	}
	if validated.Anchor.RepairDetection != nil || validated.Anchor.RepairReport != nil {
		t.Fatal("assessment bodies must not travel into workflow history")
	}
	response := validated.Response()
	if !response.OK || len(response.Checks) != len(wantRules) {
		t.Fatalf("response = %+v", response)
	}
}

func TestAnchorRule(t *testing.T) {
	env := testEnv(t)
	plan := testPlan(salvageID)
	stranger := "StrangerStrangerStrangerStranger0123456789"
	plan.PreviewHandle = &stranger
	requireFail(t, validate(t, env, plan), RuleAnchor, "was not found")

	plan = testPlan(salvageID)
	plan.PreviewHandle = nil
	plan.SourceRef = "b2://salem-data/elsewhere/sms-1.xml"
	requireFail(t, validate(t, env, plan), RuleAnchor, "No Review run exists")

	mismatched := testAnchor()
	mismatched.SourceRef = "b2://salem-data/other.xml"
	env.Anchors = &fakeAnchors{anchor: mismatched}
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleAnchor, "imported b2://salem-data/other.xml")

	unregistered := testAnchor()
	unregistered.SourceVersionID = ""
	env.Anchors = &fakeAnchors{anchor: unregistered}
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleAnchor, "no source version")

	requirePass(t, validate(t, testEnv(t), testPlan(salvageID)), RuleAnchor)

	env.Anchors = &fakeAnchors{err: errors.New("database unavailable")}
	if _, err := env.Validate(context.Background(), testPlan(salvageID)); err == nil {
		t.Fatal("an unreadable anchor is an outage, not a verdict")
	}
}

func TestRegisteredActivityRule(t *testing.T) {
	requireFail(t, validate(t, testEnv(t), testPlan("repair.write-original")), RuleRegistered, "Not a registered repair Activity")
	requireFail(t, validate(t, testEnv(t), testPlan()), RuleRegistered, "no steps")
	requirePass(t, validate(t, testEnv(t), testPlan(lenientID)), RuleRegistered)
}

func TestParamsRule(t *testing.T) {
	plan := testPlan(findID)
	plan.Steps[0].Params = json.RawMessage(`{"max_candidates":99}`)
	requireFail(t, validate(t, testEnv(t), plan), RuleParams, "at most 20")
	plan = testPlan(salvageID)
	plan.Steps[0].Params = json.RawMessage(`{"path":"/r2/original.xml"}`)
	requireFail(t, validate(t, testEnv(t), plan), RuleParams, "not a parameter")
	plan = testPlan(findID)
	plan.Steps[0].Params = json.RawMessage(`{"max_candidates":3,"require_larger":true}`)
	requirePass(t, validate(t, testEnv(t), plan), RuleParams)
}

func TestTypeChainRule(t *testing.T) {
	// Lenient decode emits derived threads; salvage reads XML.
	requireFail(t, validate(t, testEnv(t), testPlan(lenientID, salvageID)), RuleTypeChain,
		"reads sms_backup_xml or xml but receives derived_ndjson_threads from step s1")
	// Lenient decode needs an SMS backup; a generic XML source is refused.
	env := testEnv(t)
	generic := testAnchor()
	generic.SourceRef, generic.DeclaredFormat = "b2://salem-data/v/export.xml", "xml"
	env.Anchors = &fakeAnchors{anchor: generic}
	plan := testPlan(lenientID)
	plan.SourceRef = generic.SourceRef
	requireFail(t, validate(t, env, plan), RuleTypeChain, "receives xml from the source")
	// Find keeps the type, so find → salvage → lenient type-checks.
	validated := validate(t, testEnv(t), testPlan(findID, salvageID, lenientID))
	requirePass(t, validated, RuleTypeChain)
	if validated.Steps[2].OutputType != TypeDerivedThreads {
		t.Fatalf("chain types = %+v", validated.Steps)
	}
}

func TestOriginalNeverWrittenRule(t *testing.T) {
	requireFail(t, validate(t, testEnv(t), testPlan(salvageID, "repair.overwrite")), RuleOriginalNotHit, "cannot be proven")
	validated := validate(t, testEnv(t), testPlan(salvageID))
	requirePass(t, validated, RuleOriginalNotHit)
	if reason := rule(t, validated, RuleOriginalNotHit).Reason; !strings.Contains(reason, "sms-20250617122400.xml.derived/") {
		t.Fatalf("the reason must name the derived location: %s", reason)
	}
	requirePass(t, validate(t, testEnv(t), testPlan(findID)), RuleOriginalNotHit)
}

func TestReentryRule(t *testing.T) {
	requireFail(t, validate(t, testEnv(t), testPlan(salvageID, "repair.unknown")), RuleReentry, "cannot be proven ingestible")
	env := testEnv(t)
	env.Anchors = &fakeAnchors{err: ErrAnchorNotFound}
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleReentry, "no run anchors this plan")
	batch := validate(t, testEnv(t), testPlan(lenientID))
	requirePass(t, batch, RuleReentry)
	if batch.Reentry.Kind != ReentryBatch || batch.Reentry.DeclaredFormat != DerivedThreadsDeclaredFormat {
		t.Fatalf("lenient decode re-enters as a batch of ndjson chunks: %+v", batch.Reentry)
	}
	single := validate(t, testEnv(t), testPlan(findID))
	requirePass(t, single, RuleReentry)
	if single.Reentry.Kind != ReentrySingleRun {
		t.Fatalf("find re-enters as one run: %+v", single.Reentry)
	}
}

func TestDestinationRule(t *testing.T) {
	env := testEnv(t)
	upload := testAnchor()
	upload.SourceRef = "upload://" + strings.Repeat("a", 64)
	env.Anchors = &fakeAnchors{anchor: upload}
	plan := testPlan(salvageID)
	plan.SourceRef = upload.SourceRef
	requireFail(t, validate(t, env, plan), RuleDestination, "not an object-store locator")

	env = testEnv(t)
	r2 := testAnchor()
	r2.SourceRef = "r2://casebible-raw/sms-1.xml"
	env.Anchors = &fakeAnchors{anchor: r2}
	plan = testPlan(salvageID)
	plan.SourceRef = r2.SourceRef
	requireFail(t, validate(t, env, plan), RuleDestination, `scheme "r2" is not a configured object store`)

	env = testEnv(t)
	roots, err := smsthreads.ParseDerivedRoots(`[{"source":"b2://salem-data/consignatio/vault/v1/","derived":"b2://other-bucket/derived/"}]`)
	if err != nil {
		t.Fatal(err)
	}
	env.DerivedRoots = roots
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleDestination, "outside every configured source root")

	validated := validate(t, testEnv(t), testPlan(salvageID))
	requirePass(t, validated, RuleDestination)
	if reason := rule(t, validated, RuleDestination).Reason; !strings.Contains(reason, "beside the original") || !strings.Contains(reason, `"b2-bucket"`) {
		t.Fatalf("destination reason = %s", reason)
	}
}

// Live vault keys carry raw spaces ("sms /", "Phone Records/"); they are
// ordinary locators here, exactly as the batch workflow spells them.
func TestAVaultKeyWithSpacesValidates(t *testing.T) {
	env := testEnv(t)
	spaced := testAnchor()
	spaced.SourceRef = "b2://salem-data/consignatio/vault/v1/sms /Phone Records/sms-20260609173028.xml"
	env.Anchors = &fakeAnchors{anchor: spaced}
	plan := testPlan(salvageID)
	plan.SourceRef = spaced.SourceRef
	validated := validate(t, env, plan)
	if !validated.OK {
		t.Fatalf("checks = %+v", validated.Checks)
	}
	if reason := rule(t, validated, RuleDestination).Reason; !strings.Contains(reason, "sms /Phone Records/sms-20260609173028.xml.derived/") {
		t.Fatalf("destination = %s", reason)
	}
}

func TestTestLiveRule(t *testing.T) {
	plan := testPlan(salvageID)
	plan.MatterMode = ModeReal
	requireFail(t, validate(t, testEnv(t), plan), RuleTestLive, "The plan says REAL but Review run")

	env := testEnv(t)
	stray := testAnchor()
	stray.MatterID = "00000000-0000-0000-0000-000000000000"
	env.Anchors = &fakeAnchors{anchor: stray}
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleTestLive, "neither the TEST nor the REAL matter")

	env = testEnv(t)
	env.MatterMode = nil
	requireFail(t, validate(t, env, testPlan(salvageID)), RuleTestLive, "no TEST/REAL matter identities")

	requirePass(t, validate(t, testEnv(t), testPlan(salvageID)), RuleTestLive)
	env = testEnv(t)
	real := testAnchor()
	real.MatterID, real.CourtCaseID = realMatter, realCourt
	env.Anchors = &fakeAnchors{anchor: real}
	plan = testPlan(salvageID)
	plan.MatterMode = ModeReal
	requirePass(t, validate(t, env, plan), RuleTestLive)
}

func TestBoundedRule(t *testing.T) {
	many := make([]string, 13)
	for index := range many {
		many[index] = findID
	}
	requireFail(t, validate(t, testEnv(t), testPlan(many...)), RuleBounded, "1 to 12 steps")

	plan := testPlan(salvageID, lenientID)
	plan.Steps[1].StepID = "s1"
	requireFail(t, validate(t, testEnv(t), plan), RuleBounded, "appears twice")

	requireFail(t, validate(t, testEnv(t), testPlan(salvageID, salvageID)), RuleBounded, "a repeated step is a loop")

	plan = testPlan(findID, findID)
	plan.Steps[1].Params = json.RawMessage(`{"max_candidates":6}`)
	requireFail(t, validate(t, testEnv(t), plan), RuleBounded, "More than one step re-points")

	requirePass(t, validate(t, testEnv(t), testPlan(findID, salvageID, lenientID)), RuleBounded)
}

func TestFailedSummaryNamesEveryFailedRule(t *testing.T) {
	plan := testPlan(lenientID, salvageID)
	plan.MatterMode = ModeReal
	validated := validate(t, testEnv(t), plan)
	summary := validated.FailedSummary()
	for _, name := range []string{RuleTypeChain, RuleTestLive} {
		if !strings.Contains(summary, name+":") {
			t.Fatalf("summary %q does not name %s", summary, name)
		}
	}
	if strings.Contains(summary, RuleAnchor+":") {
		t.Fatalf("summary names a passing rule: %q", summary)
	}
}
