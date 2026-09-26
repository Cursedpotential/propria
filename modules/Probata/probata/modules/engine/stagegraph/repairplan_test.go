// Byline: Claude Code · Opus 5.5 · 2026-09-25

package stagegraph

import (
	"math/bits"
	"testing"
)

// The repair-plan Activities are their own registry: every one carries exactly
// one responsibility, none collides with a Proffer stage name, and none was
// slipped into the 26-stage graph (which must stay exactly as ratified).
func TestRepairPlanActivitiesAreAtomicAndSeparateFromTheProfferGraph(t *testing.T) {
	proffer := map[StageID]bool{}
	for _, d := range Stages {
		proffer[d.ID] = true
	}
	for _, d := range OptionalStages {
		proffer[d.ID] = true
	}
	seen := map[StageID]bool{}
	known := map[StageID]bool{}
	for _, d := range RepairPlanActivities {
		known[d.ID] = true
	}
	for _, d := range RepairPlanActivities {
		if seen[d.ID] {
			t.Fatalf("repair activity %q registered twice", d.ID)
		}
		seen[d.ID] = true
		if proffer[d.ID] {
			t.Fatalf("repair activity %q collides with a Proffer stage", d.ID)
		}
		if n := bits.OnesCount32(uint32(d.Responsibility)); n != 1 {
			t.Fatalf("repair activity %q has %d responsibility bits, want exactly 1", d.ID, n)
		}
		for _, dep := range d.DependsOn {
			if !known[dep] {
				t.Fatalf("repair activity %q depends on unknown %q", d.ID, dep)
			}
		}
	}
	for _, id := range []StageID{RepairFindOtherVersion, RepairSalvageTruncatedXML, RepairLenientDecode, RepairValidatePlan, RepairRecordStepReceipt} {
		if !seen[id] {
			t.Fatalf("repair activity %q is missing from RepairPlanActivities", id)
		}
	}
	if len(Stages) != 26 {
		t.Fatalf("the Proffer graph changed size: %d stages", len(Stages))
	}
}

// find_other_version writes nothing; the two derive tools publish derived
// objects. The responsibility bit is how that promise is reviewed.
func TestRepairToolResponsibilitiesMatchTheirWrites(t *testing.T) {
	want := map[StageID]Responsibility{
		RepairFindOtherVersion:    RespLocate,
		RepairSalvageTruncatedXML: RespDerive,
		RepairLenientDecode:       RespDerive,
	}
	for _, d := range RepairPlanActivities {
		if expected, ok := want[d.ID]; ok && d.Responsibility != expected {
			t.Fatalf("%q responsibility = %v, want %v", d.ID, d.Responsibility, expected)
		}
	}
}
