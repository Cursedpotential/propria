// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"encoding/json"
	"testing"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// Shapes copied from the live platform database (context.repair_assessment,
// read-only probe 2026-09-25): repair.detect output and a repair.preview report.
const (
	liveDetection     = `{"detection": {"fmt": "xml", "notes": [], "engine": "lxml-recover", "had_bom": false, "encoding": "utf-8", "confidence": 0.99}, "cloud_only": false}`
	liveDamagedReport = `{"clean": false, "lossy": 4, "format": "unknown", "by_kind": {"xml_recovery_error": 4}, "repairs": 4, "encoding": "", "chunks_ok": 21304, "truncated": false, "chunks_failed": 0}`
	truncatedReport   = `{"clean": false, "lossy": 0, "truncated": true, "chunks_ok": 900, "chunks_failed": 0}`
	cleanReport       = `{"clean": true, "lossy": 0, "truncated": false, "chunks_ok": 609, "chunks_failed": 0}`
)

func TestInferSourceTypePrefersTheStrongestEvidence(t *testing.T) {
	for _, tc := range []struct {
		evidence TypeEvidence
		want     string
	}{
		{TypeEvidence{DetectedFormat: "smsbackuprestore_xml", FileName: "x.bin"}, TypeSMSBackupXML},
		{TypeEvidence{DetectedFormat: "callsbackuprestore_xml"}, TypeSMSBackupXML},
		{TypeEvidence{DetectedFormat: "xml", FileName: "notes.xml"}, TypeXML},
		{TypeEvidence{DetectedFormat: "xml", FileName: "sms-20250617122400.xml"}, TypeSMSBackupXML},
		{TypeEvidence{DetectedFormat: "pdf", DeclaredFormat: "smsbackuprestore_xml"}, "pdf"},
		{TypeEvidence{DeclaredFormat: "sms_export_xml"}, TypeSMSBackupXML},
		{TypeEvidence{DetectionFmt: "xml", FileName: "calls-2026.xml"}, TypeSMSBackupXML},
		{TypeEvidence{DetectionFmt: "xml", FileName: "export.xml"}, TypeXML},
		{TypeEvidence{FileName: "sms-1.XML"}, TypeSMSBackupXML},
		{TypeEvidence{FileName: "photo.jpg"}, TypeUnknown},
		{TypeEvidence{}, TypeUnknown},
	} {
		if got, basis := InferSourceType(tc.evidence); got != tc.want || basis == "" {
			t.Fatalf("%+v -> %q (%s), want %q", tc.evidence, got, basis, tc.want)
		}
	}
}

func TestRepairEvidenceReadsEveryStoredShape(t *testing.T) {
	merged := ParseRepairEvidence(json.RawMessage(liveDetection), json.RawMessage(`{"report":`+truncatedReport+`}`))
	if merged.DetectionFmt != "xml" || merged.Condition() != ConditionTruncated {
		t.Fatalf("merged = %+v", merged)
	}
	if bare := ParseRepairEvidence(json.RawMessage(liveDamagedReport)); bare.Condition() != ConditionDamaged || bare.Lossy != 4 {
		t.Fatalf("bare damaged = %+v", bare)
	}
	if clean := ParseRepairEvidence(json.RawMessage(cleanReport)); clean.Condition() != ConditionClean {
		t.Fatalf("clean = %+v", clean)
	}
	if none := ParseRepairEvidence(nil, json.RawMessage(`not json`), json.RawMessage(liveDetection)); none.Condition() != ConditionUnassessed {
		t.Fatalf("detection alone is not an assessment: %+v", none)
	}
}

func proposalActivities(proposals []Proposal) [][]string {
	out := make([][]string, 0, len(proposals))
	for _, proposal := range proposals {
		var steps []string
		for _, step := range proposal.Steps {
			steps = append(steps, step.Activity)
		}
		out = append(out, steps)
	}
	return out
}

// The brief's example: a truncated SMS XML is offered find another version,
// salvage, lenient decode, and wait — in that order, each a single step.
func TestTruncatedSMSBackupGetsTheFourRatifiedOptions(t *testing.T) {
	anchor := &Anchor{DeclaredFormat: "smsbackuprestore_xml", RepairDetection: json.RawMessage(liveDetection), RepairReport: json.RawMessage(truncatedReport)}
	signature, sourceType, _ := Signature(ProposalEvidence{SourceRef: "b2://bkt/v/sms-2025.xml", Anchor: anchor})
	if signature != "sms_backup_xml:truncated" || sourceType != TypeSMSBackupXML {
		t.Fatalf("signature = %q", signature)
	}
	proposals, covered := TableProposals(signature)
	got := proposalActivities(proposals)
	want := [][]string{
		{string(stagegraph.RepairFindOtherVersion)},
		{string(stagegraph.RepairSalvageTruncatedXML)},
		{string(stagegraph.RepairLenientDecode)},
		nil,
	}
	if !covered || len(got) != len(want) {
		t.Fatalf("proposals = %v", got)
	}
	for index := range want {
		if len(got[index]) != len(want[index]) || (len(want[index]) == 1 && got[index][0] != want[index][0]) {
			t.Fatalf("proposal %d = %v, want %v", index, got[index], want[index])
		}
	}
	for _, proposal := range proposals {
		if proposal.By != ByRule || proposal.AgentAvailable || proposal.Signature != signature || proposal.Rationale == "" {
			t.Fatalf("proposal metadata = %+v", proposal)
		}
		for _, step := range proposal.Steps {
			if step.StepID != "s1" || string(step.Params) != `{}` {
				t.Fatalf("step = %+v", step)
			}
		}
	}
}

func TestSignatureTableIsOneEntryPerSignature(t *testing.T) {
	damaged := &Anchor{DeclaredFormat: "smsbackuprestore_xml", RepairReport: json.RawMessage(liveDamagedReport)}
	signature, _, _ := Signature(ProposalEvidence{SourceRef: "b2://bkt/v/sms-1.xml", Anchor: damaged})
	proposals, _ := TableProposals(signature)
	if signature != "sms_backup_xml:damaged" || proposals[0].Steps[0].Activity != string(stagegraph.RepairLenientDecode) {
		t.Fatalf("damaged SMS backup: %q %v", signature, proposalActivities(proposals))
	}
	clean := &Anchor{DeclaredFormat: "smsbackuprestore_xml", RepairReport: json.RawMessage(cleanReport)}
	signature, _, _ = Signature(ProposalEvidence{SourceRef: "b2://bkt/v/sms-1.xml", Anchor: clean})
	if proposals, covered := TableProposals(signature); !covered || len(proposals) != 0 {
		t.Fatalf("a clean source has nothing to repair: %q %v", signature, proposals)
	}
	// A signature the table does not list gets nothing — never another
	// signature's proposals (no fallback ladder).
	pdf := &Anchor{DetectedFormat: "pdf", RepairReport: json.RawMessage(truncatedReport)}
	signature, _, _ = Signature(ProposalEvidence{SourceRef: "b2://bkt/v/a.pdf", Anchor: pdf})
	if proposals, covered := TableProposals(signature); covered || len(proposals) != 0 || signature != "pdf:truncated" {
		t.Fatalf("uncovered signature %q produced %v", signature, proposals)
	}
	// A caller-supplied report is used only when the run has none.
	signature, _, _ = Signature(ProposalEvidence{SourceRef: "b2://bkt/v/x.xml", Anchor: &Anchor{DeclaredFormat: "xml"},
		DetectReport: json.RawMessage(`{"report":` + truncatedReport + `}`)})
	if signature != "xml:truncated" {
		t.Fatalf("detect_report fallback = %q", signature)
	}
	signature, _, _ = Signature(ProposalEvidence{SourceRef: "b2://bkt/v/x.xml",
		Anchor:       &Anchor{DeclaredFormat: "xml", RepairReport: json.RawMessage(cleanReport)},
		DetectReport: json.RawMessage(`{"report":` + truncatedReport + `}`)})
	if signature != "xml:clean" {
		t.Fatalf("the run's own assessment must win over detect_report: %q", signature)
	}
}
