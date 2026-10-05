// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
)

// Rule names, in the order the Workbench shows them. Every rule in the
// ratified design is one named check; anchor_resolves and params_valid make
// explicit two preconditions the others rely on.
const (
	RuleAnchor         = "anchor_resolves"
	RuleRegistered     = "registered_activity"
	RuleParams         = "params_valid"
	RuleTypeChain      = "type_chain"
	RuleOriginalNotHit = "original_never_written"
	RuleReentry        = "reentry_ingestible"
	RuleDestination    = "destination_resolves"
	RuleTestLive       = "test_live_mode"
	RuleBounded        = "bounded"
	maxPlanSteps       = 12
	reentrySingleRun   = "single_run"
	reentryBatch       = "batch"
	salvagedSubfolder  = "salvaged/"
	lenientVariant     = "lenient"
)

// Re-entry kinds.
const (
	ReentrySingleRun = reentrySingleRun
	ReentryBatch     = reentryBatch
)

// SalvagedSubfolder and LenientVariant name where the two derive tools write
// beneath a source's derived location. The validator states them and the
// Activities use them, so the promise and the behaviour cannot drift apart.
const (
	SalvagedSubfolder = salvagedSubfolder
	LenientVariant    = lenientVariant
)

// Anchor is the Review run a plan belongs to: its matter, court case, parser
// options and declared format scope the re-entry run, and its source version
// anchors every step receipt.
type Anchor struct {
	OperatingMode    string `json:"operating_mode"`
	PreviewHandle    string `json:"preview_handle"`
	RequestID        string `json:"request_id"`
	WorkflowID       string `json:"workflow_id"`
	SourceRef        string `json:"source_ref"`
	SourceVersionID  string `json:"source_version_id"`
	DeclaredFormat   string `json:"declared_format"`
	DetectedFormat   string `json:"detected_format,omitempty"`
	ParserOptionsRef string `json:"parser_options_ref"`
	MatterID         string `json:"matter_id,omitempty"`
	CourtCaseID      string `json:"court_case_id,omitempty"`
	// RepairDetection and RepairReport are the run's persisted repair.detect
	// output and repair.preview report. They inform type and signature only
	// and never enter workflow history.
	RepairDetection json.RawMessage `json:"repair_detection,omitempty"`
	RepairReport    json.RawMessage `json:"repair_report,omitempty"`
}

// ErrAnchorNotFound reports that no Review run anchors the plan.
var ErrAnchorNotFound = errors.New("repairplan: no Review run anchors this plan")

// AnchorResolver finds a plan's Review run: by preview handle when one is
// given, else the newest run bound to the source reference.
type AnchorResolver interface {
	ResolveAnchor(ctx context.Context, sourceRef, previewHandle string) (Anchor, error)
}

// Environment is the configuration one process validates against. The
// starter validates for the Workbench; the worker validates again with its
// own configuration before any step runs, so a plan never writes where the
// writer's configuration would refuse.
type Environment struct {
	Registry     *Registry
	Anchors      AnchorResolver
	DerivedRoots smsthreads.DerivedRoots
	Stores       objectstores.Stores
	SourceRoots  objectstores.Roots
	// IdentityAdmitted verifies the one case pair and never derives operating mode.
	IdentityAdmitted func(matterID, courtCaseID string) bool
}

// ValidatedPlan is the validator's full result. When OK it is exactly what
// RepairPlanWorkflow executes.
type ValidatedPlan struct {
	OK         bool           `json:"ok"`
	Checks     []Check        `json:"checks"`
	PlanID     string         `json:"plan_id"`
	MatterMode string         `json:"matter_mode"`
	SourceRef  string         `json:"source_ref"`
	SourceType string         `json:"source_type"`
	Anchor     Anchor         `json:"anchor"`
	Steps      []ResolvedStep `json:"steps"`
	Reentry    Reentry        `json:"reentry"`
}

// ResolvedStep is one typed, bounded step ready to schedule.
type ResolvedStep struct {
	StepID              string          `json:"step_id"`
	Activity            string          `json:"activity"`
	Params              json.RawMessage `json:"params"`
	InputType           string          `json:"input_type"`
	OutputType          string          `json:"output_type"`
	OutputKind          string          `json:"output_kind"`
	NeedsN8N            bool            `json:"needs_n8n"`
	FlowName            string          `json:"flow_name,omitempty"`
	StartToCloseSeconds int64           `json:"start_to_close_seconds"`
	HeartbeatSeconds    int64           `json:"heartbeat_seconds,omitempty"`
	MaxAttempts         int32           `json:"max_attempts"`
}

// Reentry is how the plan's result re-enters Proffer.
type Reentry struct {
	Kind             string `json:"kind"`
	DeclaredFormat   string `json:"declared_format"`
	ParserOptionsRef string `json:"parser_options_ref"`
	MatterID         string `json:"matter_id"`
	CourtCaseID      string `json:"court_case_id"`
	// From the plan's reentry options (empty when the plan names none).
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	OwnerPersonID       string `json:"owner_person_id,omitempty"`
	PerspectivePersonID string `json:"perspective_person_id,omitempty"`
	AutoApproval        string `json:"auto_approval,omitempty"`
}

// Response projects the result onto the contract.
func (v ValidatedPlan) Response() ValidateResponse {
	checks := v.Checks
	if checks == nil {
		checks = []Check{}
	}
	return ValidateResponse{OK: v.OK, Checks: checks}
}

// FailedSummary names every failed rule with its reason, for a 422 detail.
func (v ValidatedPlan) FailedSummary() string {
	var failed []string
	for _, check := range v.Checks {
		if check.Status != CheckPass {
			failed = append(failed, check.Rule+": "+check.Reason)
		}
	}
	if len(failed) == 0 {
		return ""
	}
	return "plan failed validation — " + strings.Join(failed, " | ")
}

// Validate resolves the plan's anchor and runs every rule. It returns an error
// only when the anchor could not be read at all (an outage, not a verdict).
func (e Environment) Validate(ctx context.Context, plan Plan) (ValidatedPlan, error) {
	if e.Anchors == nil {
		return ValidatedPlan{}, errors.New("repairplan: no anchor resolver is configured")
	}
	anchor, err := e.Anchors.ResolveAnchor(ctx, plan.SourceRef, plan.Handle())
	switch {
	case err == nil:
		return e.ValidateAgainst(plan, &anchor), nil
	case errors.Is(err, ErrAnchorNotFound):
		return e.ValidateAgainst(plan, nil), nil
	default:
		return ValidatedPlan{}, err
	}
}

// ValidateAgainst runs every rule against an already-resolved anchor (nil
// when none exists). It is pure given the Environment's configuration.
func (e Environment) ValidateAgainst(plan Plan, anchor *Anchor) ValidatedPlan {
	registry := e.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	out := ValidatedPlan{PlanID: plan.PlanID, MatterMode: plan.MatterMode, SourceRef: plan.SourceRef, Steps: []ResolvedStep{}}
	if anchor != nil {
		out.Anchor = *anchor
		out.Anchor.RepairDetection, out.Anchor.RepairReport = nil, nil
	}
	specs := make([]*ToolSpec, len(plan.Steps))
	for index, step := range plan.Steps {
		if spec, ok := registry.Lookup(step.Activity); ok {
			specCopy := spec
			specs[index] = &specCopy
		}
	}
	sourceType, basis := planSourceType(plan, anchor)
	out.SourceType = sourceType

	add := func(rule string, reason string, pass bool) {
		status := CheckFail
		if pass {
			status = CheckPass
		}
		out.Checks = append(out.Checks, Check{Rule: rule, Status: status, Reason: reason})
	}
	reason, pass := checkAnchor(plan, anchor)
	add(RuleAnchor, reason, pass)
	reason, pass = checkRegistered(plan, specs)
	add(RuleRegistered, reason, pass)
	reason, pass = checkParams(plan, specs)
	add(RuleParams, reason, pass)
	reason, pass, types := checkTypeChain(plan, specs, sourceType, basis)
	add(RuleTypeChain, reason, pass)
	location, source, locationErr := e.derivedLocation(plan.SourceRef)
	reason, pass = checkOriginal(plan, specs, location, locationErr)
	add(RuleOriginalNotHit, reason, pass)
	reentry, reason, pass := checkReentry(plan, specs, anchor, types)
	add(RuleReentry, reason, pass)
	reason, pass = e.checkDestination(plan, specs, location, locationErr, source)
	add(RuleDestination, reason, pass)
	reason, pass = e.checkTestLive(plan, anchor)
	add(RuleTestLive, reason, pass)
	reason, pass = checkBounded(plan, specs)
	add(RuleBounded, reason, pass)

	out.OK = true
	for _, check := range out.Checks {
		if check.Status != CheckPass {
			out.OK = false
		}
	}
	out.Reentry = reentry
	for index, step := range plan.Steps {
		spec := specs[index]
		if spec == nil {
			continue
		}
		resolved := ResolvedStep{
			StepID: step.StepID, Activity: step.Activity, Params: normalizedParams(step.Params),
			OutputKind: spec.OutputKind, NeedsN8N: spec.NeedsN8N, FlowName: spec.FlowName,
			StartToCloseSeconds: int64(spec.StartToClose.Seconds()), HeartbeatSeconds: int64(spec.Heartbeat.Seconds()),
			MaxAttempts: spec.MaxAttempts,
		}
		if index < len(types) {
			resolved.InputType, resolved.OutputType = types[index][0], types[index][1]
		}
		out.Steps = append(out.Steps, resolved)
	}
	return out
}

func normalizedParams(params json.RawMessage) json.RawMessage {
	return json.RawMessage(canonicalParams(params))
}

func planSourceType(plan Plan, anchor *Anchor) (string, string) {
	evidence := TypeEvidence{FileName: SourceFileName(plan.SourceRef)}
	var documents []json.RawMessage
	if anchor != nil {
		evidence.DetectedFormat, evidence.DeclaredFormat = anchor.DetectedFormat, anchor.DeclaredFormat
		documents = append(documents, anchor.RepairDetection, anchor.RepairReport)
	}
	evidence.DetectionFmt = ParseRepairEvidence(documents...).DetectionFmt
	return InferSourceType(evidence)
}

func checkAnchor(plan Plan, anchor *Anchor) (string, bool) {
	if err := plan.ShapeError(); err != nil {
		return "The plan does not match the contract: " + err.Error() + ".", false
	}
	if anchor == nil {
		if handle := plan.Handle(); handle != "" {
			return fmt.Sprintf("Review run %s was not found. A repair plan belongs to a Review run, which supplies the matter, "+
				"court case and parser options the repaired copy re-enters with.", handle), false
		}
		return fmt.Sprintf("No Review run exists for %s. Start the source in Review first: the run supplies the matter, "+
			"court case and parser options the repaired copy re-enters with.", plan.SourceRef), false
	}
	switch {
	case anchor.SourceRef != plan.SourceRef:
		return fmt.Sprintf("Review run %s imported %s, not %s.", anchor.PreviewHandle, anchor.SourceRef, plan.SourceRef), false
	case anchor.SourceVersionID == "":
		return fmt.Sprintf("Review run %s has not registered its source yet, so there is no source version to record step receipts against.", anchor.PreviewHandle), false
	case anchor.DeclaredFormat == "" || anchor.ParserOptionsRef == "":
		return fmt.Sprintf("Review run %s has no declared format or parser options to re-enter with.", anchor.PreviewHandle), false
	}
	return fmt.Sprintf("Anchored to Review run %s (source version %s, declared format %s). Every step receipt is recorded "+
		"against that source version.", anchor.PreviewHandle, anchor.SourceVersionID, anchor.DeclaredFormat), true
}

func checkRegistered(plan Plan, specs []*ToolSpec) (string, bool) {
	if len(plan.Steps) == 0 {
		return "The plan has no steps.", false
	}
	var unknown, known []string
	for index, step := range plan.Steps {
		if specs[index] == nil {
			unknown = append(unknown, fmt.Sprintf("%s (%q)", step.StepID, step.Activity))
			continue
		}
		known = append(known, fmt.Sprintf("%s: %s (%s → %s)", step.StepID, step.Activity,
			strings.Join(specs[index].InputTypes, "|"), strings.Join(specs[index].OutputTypes, "|")))
	}
	if len(unknown) > 0 {
		return "Not a registered repair Activity: " + strings.Join(unknown, ", ") + ".", false
	}
	return "Every step is a registered repair Activity with declared types — " + strings.Join(known, "; ") + ".", true
}

func checkParams(plan Plan, specs []*ToolSpec) (string, bool) {
	var problems []string
	checked := 0
	for index, step := range plan.Steps {
		if specs[index] == nil {
			continue
		}
		checked++
		if err := validateParams(specs[index].ParamsSchema, step.Params); err != nil {
			problems = append(problems, fmt.Sprintf("%s (%s): %v", step.StepID, step.Activity, err))
		}
	}
	if len(problems) > 0 {
		return "Parameters rejected — " + strings.Join(problems, "; ") + ".", false
	}
	if checked != len(plan.Steps) || checked == 0 {
		return "Parameters cannot be checked for steps that are not registered Activities.", false
	}
	return "Every step's parameters satisfy its Activity's params_schema.", true
}

// checkTypeChain threads the source type through the steps and returns each
// step's (input, output) type pair.
func checkTypeChain(plan Plan, specs []*ToolSpec, sourceType, basis string) (string, bool, [][2]string) {
	types := make([][2]string, 0, len(plan.Steps))
	current := sourceType
	trail := []string{fmt.Sprintf("source %s (%s)", sourceType, basis)}
	for index, step := range plan.Steps {
		spec := specs[index]
		if spec == nil {
			return fmt.Sprintf("Step %s is not a registered Activity, so its types are unknown.", step.StepID), false, types
		}
		if !accepts(spec.InputTypes, current) {
			from := "the source"
			if index > 0 {
				from = "step " + plan.Steps[index-1].StepID
			}
			return fmt.Sprintf("Step %s (%s) reads %s but receives %s from %s.", step.StepID, step.Activity,
				strings.Join(spec.InputTypes, " or "), current, from), false, types
		}
		output := spec.OutputTypes[0]
		if spec.PreservesType {
			output = current
		}
		types = append(types, [2]string{current, output})
		trail = append(trail, fmt.Sprintf("%s %s → %s", step.StepID, step.Activity, output))
		current = output
	}
	if len(plan.Steps) == 0 {
		return "No steps to type-check.", false, types
	}
	return "Type-checked: " + strings.Join(trail, " → ") + ".", true, types
}

// sourceCoordinate is a parsed object-store locator.
type sourceCoordinate struct {
	scheme, bucket, key string
	ok                  bool
}

func parseObjectLocator(ref string) sourceCoordinate {
	parsed, err := url.Parse(strings.TrimSpace(ref))
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return sourceCoordinate{}
	}
	key, err := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
	if err != nil || key == "" || strings.HasSuffix(key, "/") {
		return sourceCoordinate{}
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == ".." || segment == "." {
			return sourceCoordinate{}
		}
	}
	return sourceCoordinate{scheme: strings.ToLower(parsed.Scheme), bucket: parsed.Host, key: key, ok: true}
}

// derivedLocation resolves where derived output for the plan's own source
// goes, exactly as the derive Activities resolve it.
func (e Environment) derivedLocation(sourceRef string) (smsthreads.DerivedLocation, sourceCoordinate, error) {
	source := parseObjectLocator(sourceRef)
	if !source.ok {
		return smsthreads.DerivedLocation{}, source, fmt.Errorf("%s is not an object-store locator (<scheme>://<bucket>/<key>)", sourceRef)
	}
	if !e.Stores.Has(source.scheme) {
		return smsthreads.DerivedLocation{}, source, fmt.Errorf("scheme %q is not a configured object store on this engine (configured: %v)",
			source.scheme, e.Stores.Schemes())
	}
	location, err := e.DerivedRoots.Locate(source.scheme, source.bucket, source.key)
	return location, source, err
}

func checkOriginal(plan Plan, specs []*ToolSpec, location smsthreads.DerivedLocation, locationErr error) (string, bool) {
	var writers []string
	for index, step := range plan.Steps {
		spec := specs[index]
		if spec == nil {
			return fmt.Sprintf("Step %s is not a registered Activity, so what it writes cannot be proven.", step.StepID), false
		}
		switch spec.Writes {
		case WritesNone:
		case WritesDerived:
			writers = append(writers, step.StepID)
		default:
			return fmt.Sprintf("Step %s (%s) declares write class %q.", step.StepID, step.Activity, spec.Writes), false
		}
	}
	if len(writers) == 0 {
		return "No step writes anything; the original is only read.", true
	}
	where := "a derived location beside the source (<key>.derived/) or under its configured derived root"
	if locationErr == nil {
		where = location.URI()
	}
	return fmt.Sprintf("Steps %s write only new derived objects under %s (salvage: %s<name>; lenient decode: %s/). "+
		"The original is only read, and every derived object is hashed.", strings.Join(writers, ", "), where,
		salvagedSubfolder, lenientVariant), true
}

func checkReentry(plan Plan, specs []*ToolSpec, anchor *Anchor, types [][2]string) (Reentry, string, bool) {
	if len(plan.Steps) == 0 {
		return Reentry{}, "Nothing to re-enter: the plan has no steps.", false
	}
	last := specs[len(specs)-1]
	if last == nil || len(types) != len(plan.Steps) {
		return Reentry{}, "The last step's output cannot be proven ingestible (unregistered step or broken type chain).", false
	}
	if anchor == nil {
		return Reentry{}, "Re-entry needs the Review run's matter, court case and parser options, and no run anchors this plan.", false
	}
	reentry := Reentry{
		ParserOptionsRef: anchor.ParserOptionsRef, MatterID: anchor.MatterID, CourtCaseID: anchor.CourtCaseID,
	}
	if options := plan.Reentry; options != nil {
		reentry.OwnerPersonID, reentry.PerspectivePersonID, reentry.AutoApproval =
			options.OwnerPersonID, options.PerspectivePersonID, options.AutoApproval
	}
	terminal := types[len(types)-1][1]
	switch last.OutputKind {
	case OutputDerivedChunkFolder:
		reentry.Kind, reentry.DeclaredFormat = reentryBatch, DerivedThreadsDeclaredFormat
	case OutputDerivedObject, OutputExistingObject:
		reentry.Kind, reentry.DeclaredFormat = reentrySingleRun, anchor.DeclaredFormat
	default:
		return Reentry{}, fmt.Sprintf("Step %s produces %q, which no Proffer route ingests.", plan.Steps[len(plan.Steps)-1].StepID, last.OutputKind), false
	}
	if reentry.DeclaredFormat == "" || reentry.ParserOptionsRef == "" || reentry.MatterID == "" || reentry.CourtCaseID == "" {
		return Reentry{}, "The re-entry run would lack a declared format, parser options, matter or court case.", false
	}
	what := "the repaired copy"
	switch last.OutputKind {
	case OutputExistingObject:
		what = "the other version found"
	case OutputDerivedObject:
		what = "the salvaged derived copy"
	}
	if reentry.Kind == reentryBatch {
		return reentry, fmt.Sprintf("After the last step, one Proffer batch starts over the derived threads/ folder (%s), every chunk "+
			"declared %s, under the Review run's matter: each chunk runs validation, routing, handler selection, extraction, "+
			"storage and completeness — no stage is skipped.", terminal, reentry.DeclaredFormat), true
	}
	return reentry, fmt.Sprintf("After the last step, a new Proffer run starts on %s (%s) declared %s with the Review run's "+
		"parser options and matter: validation, routing, handler selection, extraction, storage and completeness all run — "+
		"no stage is skipped.", what, terminal, reentry.DeclaredFormat), true
}

func (e Environment) checkDestination(plan Plan, specs []*ToolSpec, location smsthreads.DerivedLocation, locationErr error, source sourceCoordinate) (string, bool) {
	if locationErr != nil {
		return "Derived output cannot be placed: " + locationErr.Error() + ". Repairs publish derived copies in the source's own object store.", false
	}
	if _, admitted := e.SourceRoots.Match(source.scheme, source.bucket, source.key); !admitted {
		return fmt.Sprintf("%s is outside every configured source root (SOURCE_ROOTS_JSON), so nothing derived from it could re-enter Proffer.", plan.SourceRef), false
	}
	firstWriter, repointed := -1, false
	for index, spec := range specs {
		if spec == nil {
			return "Destinations cannot be resolved for unregistered steps.", false
		}
		if spec.RepointsSource && firstWriter < 0 {
			repointed = true
		}
		if spec.Writes == WritesDerived && firstWriter < 0 {
			firstWriter = index
		}
	}
	placement := "beside the original (no DERIVED_ROOTS_JSON rule matches this source)"
	if location.Mapped {
		placement = "under the configured derived root for " + location.MatchedSource
	}
	sentinel := location.Prefix + "x"
	root, admitted := e.SourceRoots.Match(location.Scheme, location.Bucket, sentinel)
	if firstWriter >= 0 && !repointed && !admitted {
		return fmt.Sprintf("Derived output would go to %s (%s), which is outside every configured source root, so the repaired "+
			"copy could not re-enter Proffer.", location.URI(), placement), false
	}
	switch {
	case firstWriter < 0:
		return fmt.Sprintf("No step writes. The re-entry run starts from the other version found, which the step only accepts "+
			"when it exists now and lies inside a configured source root (%d configured).", len(e.SourceRoots)), true
	case repointed:
		return fmt.Sprintf("Derived output is placed after the source is re-pointed, by the same rule: under the chosen copy's "+
			"derived location in store %s, which the step refuses unless it lies inside a configured source root.", source.scheme), true
	}
	return fmt.Sprintf("Derived output goes to %s, %s, in configured object store %s, inside source root %q — so the "+
		"re-entry run may start from it.", location.URI(), placement, location.Scheme, root.ID), true
}

func (e Environment) checkTestLive(plan Plan, anchor *Anchor) (string, bool) {
	if plan.MatterMode != ModeDev && plan.MatterMode != ModeLive {
		return "matter_mode must be DEV or LIVE.", false
	}
	if anchor == nil {
		return "Operating mode cannot be proven without the Review run.", false
	}
	if e.IdentityAdmitted == nil {
		return "No admitted case identity predicate is configured.", false
	}
	if !e.IdentityAdmitted(anchor.MatterID, anchor.CourtCaseID) {
		return "Review run is not under the approved case identity.", false
	}
	if anchor.OperatingMode != plan.MatterMode {
		return "The plan does not match the Review run's explicit operating_mode receipt.", false
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(anchor.OperatingMode)); err != nil {
		return err.Error(), false
	}
	return "LIVE re-entry preserves the approved case identity and existing safety gates.", true
}

func checkBounded(plan Plan, specs []*ToolSpec) (string, bool) {
	count := len(plan.Steps)
	if count == 0 || count > maxPlanSteps {
		return fmt.Sprintf("A plan has 1 to %d steps; this one has %d.", maxPlanSteps, count), false
	}
	ids := map[string]bool{}
	repeated := map[string]string{}
	repoints := 0
	for index, step := range plan.Steps {
		if !stepIDPattern.MatchString(step.StepID) {
			return fmt.Sprintf("Step %d has an invalid step_id %q.", index+1, step.StepID), false
		}
		if ids[step.StepID] {
			return fmt.Sprintf("step_id %q appears twice.", step.StepID), false
		}
		ids[step.StepID] = true
		key := step.Activity + "\x00" + canonicalParams(step.Params)
		if earlier, seen := repeated[key]; seen {
			return fmt.Sprintf("Step %s repeats step %s exactly (%s with the same parameters): a repeated step is a loop.", step.StepID, earlier, step.Activity), false
		}
		repeated[key] = step.StepID
		spec := specs[index]
		if spec == nil {
			return fmt.Sprintf("Step %s is not registered, so its memory bound is unknown.", step.StepID), false
		}
		if spec.RepointsSource {
			repoints++
		}
		if !spec.Streaming {
			return fmt.Sprintf("Step %s (%s) does not stream, so its memory is unbounded.", step.StepID, step.Activity), false
		}
	}
	if repoints > 1 {
		return "More than one step re-points the source; a second re-point could lead back to the first source (a cycle).", false
	}
	return fmt.Sprintf("%d of at most %d steps, unique ids, no repeated step, at most one re-point (no cycles); every step "+
		"streams with bounded memory, and a batch re-entry is bounded by the batch workflow.", count, maxPlanSteps), true
}
