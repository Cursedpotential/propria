// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// ProposedBy values.
const (
	ByRule  = "rule"
	ByAgent = "agent"
)

// A signature is "<source type>:<condition>", for example
// "sms_backup_xml:truncated". The table holds one entry per signature, never
// a fallback from one signature to another (the router is signature-based,
// owner standing rule): a signature it does not list gets no rule proposals.
type tableProposal struct {
	rationale  string
	activities []stagegraph.StageID // empty: the "wait" option, nothing to run
}

const (
	rationaleFind = "Look in the Case Bible catalog for another copy with the same file name that is larger or has a different " +
		"hash. If one exists and can be opened, it becomes the source and is imported instead; this copy is left as it is."
	rationaleFindTwin = "The bytes of this file are scrambled: they look random and hold no readable markup, so no repair can read them. " +
		"Look in the Case Bible catalog for another copy with the same file name; the copy of the same size and a different hash " +
		"is the intact twin, and it becomes the source and is imported instead. This copy is left as it is."
	rationaleSalvage = "Keep every record up to the last complete one before the cut-off, close the document, and import that " +
		"salvaged copy. Nothing after the cut-off can be recovered this way; the original stays untouched."
	rationaleLenient = "Decode the backup leniently: records that decode are kept, records that do not are set aside as rejects, " +
		"and if the damage stops the decoder everything before it is kept. The decoded threads are imported as their own runs."
	rationaleWait = "Wait: do not repair yet. Continue this run with the original (Review, repair decision, use the original) " +
		"so parsing reports exactly what is damaged, then come back and repair with that detail. Nothing runs from here."
)

var (
	findTwin = tableProposal{rationale: rationaleFindTwin, activities: []stagegraph.StageID{stagegraph.RepairFindOtherVersion}}
	find     = tableProposal{rationale: rationaleFind, activities: []stagegraph.StageID{stagegraph.RepairFindOtherVersion}}
	salvage  = tableProposal{rationale: rationaleSalvage, activities: []stagegraph.StageID{stagegraph.RepairSalvageTruncatedXML}}
	lenient  = tableProposal{rationale: rationaleLenient, activities: []stagegraph.StageID{stagegraph.RepairLenientDecode}}
	wait     = tableProposal{rationale: rationaleWait}
)

// signatureTable is the whole rule proposer. Order inside an entry is the
// order the Workbench shows; the owner picks.
var signatureTable = map[string][]tableProposal{
	TypeSMSBackupXML + ":" + ConditionTruncated:  {find, salvage, lenient, wait},
	TypeSMSBackupXML + ":" + ConditionDamaged:    {lenient, find, wait},
	TypeSMSBackupXML + ":" + ConditionUnassessed: {wait, find, lenient, salvage},
	TypeSMSBackupXML + ":" + ConditionClean:      {},
	TypeXML + ":" + ConditionTruncated:           {find, salvage, wait},
	TypeXML + ":" + ConditionDamaged:             {find, wait},
	TypeXML + ":" + ConditionUnassessed:          {wait, find},
	TypeXML + ":" + ConditionClean:               {},
	// HTML (2026-10-02). There is no markup repair engine worth running on scrambled bytes; the way
	// forward is another copy of the same file.
	TypeHTML + ":" + ConditionUnreadable: {findTwin, wait},
	TypeHTML + ":" + ConditionTruncated:  {find, wait},
	TypeHTML + ":" + ConditionDamaged:    {find, wait},
	TypeHTML + ":" + ConditionUnassessed: {wait, find},
	TypeHTML + ":" + ConditionClean:      {},
}

// ProposalEvidence is everything the proposer reads about one source.
type ProposalEvidence struct {
	SourceRef string
	// Anchor is the source's Review run, when one exists.
	Anchor *Anchor
	// DetectReport is optional caller-supplied repair.detect/preview output;
	// the anchor's own persisted assessment is preferred.
	DetectReport json.RawMessage
}

// Signature computes "<type>:<condition>" and the type's basis.
func Signature(evidence ProposalEvidence) (signature, sourceType, basis string) {
	// The run's own persisted assessment is authoritative; a caller-supplied
	// report is used only when the run has none.
	documents := []json.RawMessage{evidence.DetectReport}
	typeEvidence := TypeEvidence{FileName: SourceFileName(evidence.SourceRef)}
	if evidence.Anchor != nil {
		typeEvidence.DetectedFormat = evidence.Anchor.DetectedFormat
		typeEvidence.DeclaredFormat = evidence.Anchor.DeclaredFormat
		if len(evidence.Anchor.RepairDetection) > 0 || len(evidence.Anchor.RepairReport) > 0 {
			documents = []json.RawMessage{evidence.Anchor.RepairDetection, evidence.Anchor.RepairReport}
		}
	}
	report := ParseRepairEvidence(documents...)
	typeEvidence.DetectionFmt = report.DetectionFmt
	sourceType, basis = InferSourceType(typeEvidence)
	return sourceType + ":" + report.Condition(), sourceType, basis
}

// TableProposals returns the rule proposals for a signature, and whether the
// table lists the signature at all.
func TableProposals(signature string) ([]Proposal, bool) {
	entry, covered := signatureTable[signature]
	if !covered {
		return []Proposal{}, false
	}
	proposals := make([]Proposal, 0, len(entry))
	for _, candidate := range entry {
		steps := make([]Step, 0, len(candidate.activities))
		for index, activity := range candidate.activities {
			steps = append(steps, Step{
				StepID: fmt.Sprintf("s%d", index+1), Activity: string(activity), Params: json.RawMessage(`{}`),
			})
		}
		proposals = append(proposals, Proposal{
			Signature: signature, Rationale: candidate.rationale, Steps: steps, By: ByRule,
		})
	}
	return proposals, true
}

// SourceFileName is the base name of an object-store locator's key.
func SourceFileName(sourceRef string) string {
	parsed, err := url.Parse(strings.TrimSpace(sourceRef))
	if err != nil {
		return ""
	}
	key, err := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/"))
	if err != nil || key == "" {
		return ""
	}
	return path.Base(key)
}
