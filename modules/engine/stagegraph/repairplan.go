// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// The repair-plan Activities (owner ratification 2026-09-25 19:09, option A:
// docs/pending-review/2026-09-25-repair-workflow-builder.md).
//
// These are NOT stages of the 26-stage ProfferWorkflow. They are the steps an
// owner composes into a repair plan in the Workbench; RepairPlanWorkflow runs
// them in the order the owner chose and then re-enters an ordinary Proffer run
// on the result. They are registered here because a new capability is a new
// Activity registered in the stage graph (AGENTS.md ATOMICITY rule 7); keeping
// them in their own list preserves the base graph's "every listed stage runs"
// invariant and leaves ProfferWorkflow untouched.
//
// The three repair tools are named by their registry id so the Workbench, the
// plan JSON, Temporal history and the receipt ledger all spell one name.

package stagegraph

const (
	// RepairFindOtherVersion looks the source's file name up in the catalog and
	// names another existing copy (different hash or larger). Read-only.
	RepairFindOtherVersion StageID = "repair.find_other_version"
	// RepairSalvageTruncatedXML keeps every complete record of a cut-off XML
	// backup and publishes a new, hashed derived copy. Never writes the original.
	RepairSalvageTruncatedXML StageID = "repair.salvage_truncated_xml"
	// RepairLenientDecode decodes an SMS backup with the SBV decoder in lenient
	// mode into derived NDJSON threads. Never writes the original.
	RepairLenientDecode StageID = "repair.lenient_decode"
	// RepairValidatePlan re-runs the fail-closed plan validator inside the
	// worker, against the worker's own storage configuration, before any step.
	RepairValidatePlan StageID = "repair_validate_plan_activity"
	// RepairRecordStepReceipt writes one append-only receipt per plan step.
	RepairRecordStepReceipt StageID = "repair_record_step_receipt_activity"
)

// RepairPlanActivities is the reviewable registry of every Activity a repair
// plan run may schedule besides the existing re-entry Activities. Each carries
// exactly one responsibility bit, like the base stages.
var RepairPlanActivities = []Descriptor{
	{
		ID:             RepairValidatePlan,
		Responsibility: RespValidate,
		Result:         "validated plan: anchor run, source type, typed steps, re-entry route",
	},
	{
		ID:             RepairFindOtherVersion,
		Responsibility: RespLocate,
		Result:         "locator of another existing copy of the source (re-points the plan)",
		DependsOn:      []StageID{RepairValidatePlan},
	},
	{
		ID:             RepairSalvageTruncatedXML,
		Responsibility: RespDerive,
		Result:         "locator and sha256 of a salvaged derived XML object",
		DependsOn:      []StageID{RepairValidatePlan},
	},
	{
		ID:             RepairLenientDecode,
		Responsibility: RespDerive,
		Result:         "locator and sha256 of a lenient derived NDJSON thread manifest",
		DependsOn:      []StageID{RepairValidatePlan},
	},
	{
		ID:             RepairRecordStepReceipt,
		Responsibility: RespPersist,
		Result:         "append-only step receipt reference",
		DependsOn:      []StageID{RepairValidatePlan},
	},
}
