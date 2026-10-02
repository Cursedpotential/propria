// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"
)

// The repair type vocabulary. A step's input and output types are checked
// against it so a plan can never hand a step something it cannot read.
const (
	// TypeSMSBackupXML is an SMS Backup & Restore <smses> or <calls> XML.
	TypeSMSBackupXML = "sms_backup_xml"
	// TypeXML is any other XML document.
	TypeXML = "xml"
	// TypeHTML is an HTML document: whichever HTML content signature the handler selection found
	// (facebook_messenger_html, generic_html_document), or bytes that sit under an HTML name but carry no
	// readable markup (scrambled files, 2026-10-02).
	TypeHTML = "html"
	// TypeDerivedThreads is a published per-thread NDJSON derivation: a
	// manifest plus a threads/ folder of chunks, re-entered as a batch.
	TypeDerivedThreads = "derived_ndjson_threads"
	// TypeAny accepts every source type.
	TypeAny = "any"
	// TypeSameAsInput is the output type of a tool that keeps its input type.
	TypeSameAsInput = "same-as-input"
	// TypeUnknown is a source nothing identified.
	TypeUnknown = "unknown"
)

// DerivedThreadsDeclaredFormat is what each derived chunk declares when it
// re-enters Proffer: the DuckDB ndjson_v1 template reads a thread line as a
// message row (the same value derive_sms_threads_activity hands on).
const DerivedThreadsDeclaredFormat = "ndjson"

// backupFileName matches the names SMS Backup & Restore gives its files.
var backupFileName = regexp.MustCompile(`(?i)^(sms|calls)-[^/]*\.xml$`)

var htmlFormats = map[string]bool{
	"html": true, "facebook_messenger_html": true, "generic_html_document": true,
}

var smsBackupFormats = map[string]bool{
	"smsbackuprestore_xml": true, "callsbackuprestore_xml": true,
	"sms_xml": true, "sms_export_xml": true, "calls_xml": true,
}

// TypeEvidence is everything known about a source's type, strongest first.
type TypeEvidence struct {
	// DetectedFormat is the handler selection's content signature
	// (context.handler_detected_format), when the run got that far.
	DetectedFormat string
	// DeclaredFormat is what the run declared at start.
	DeclaredFormat string
	// DetectionFmt is repair.detect's structural family ("xml", "json", ...).
	DetectionFmt string
	// FileName is the source object's base name.
	FileName string
}

// InferSourceType names the source's repair type and what decided it. The
// content signature wins; the declared format, the repair detector's family
// and the file name follow in that order. It never guesses beyond them.
func InferSourceType(evidence TypeEvidence) (string, string) {
	detected := strings.ToLower(strings.TrimSpace(evidence.DetectedFormat))
	declared := strings.ToLower(strings.TrimSpace(evidence.DeclaredFormat))
	family := strings.ToLower(strings.TrimSpace(evidence.DetectionFmt))
	name := strings.TrimSpace(evidence.FileName)
	switch {
	case smsBackupFormats[detected]:
		return TypeSMSBackupXML, "content signature " + detected
	case detected == "xml":
		return xmlByName(name), "content signature xml"
	case htmlFormats[detected]:
		return TypeHTML, "content signature " + detected
	case detected == "binary" && (family == "html" || isHTMLName(name)):
		// The handler found opaque bytes where the name and the repair detector say HTML.
		return TypeHTML, "opaque content under an html name"
	case detected != "":
		return detected, "content signature " + detected
	case smsBackupFormats[declared]:
		return TypeSMSBackupXML, "declared format " + declared
	case declared == "xml":
		return xmlByName(name), "declared format xml"
	case htmlFormats[declared]:
		return TypeHTML, "declared format " + declared
	case family == "html":
		return TypeHTML, "repair detector family html"
	case family == "xml":
		return xmlByName(name), "repair detector family xml"
	case strings.EqualFold(path.Ext(name), ".xml"):
		return xmlByName(name), "file name " + name
	case declared != "":
		return declared, "declared format " + declared
	case family != "" && family != "unknown":
		return family, "repair detector family " + family
	}
	return TypeUnknown, "nothing identified the source"
}

func isHTMLName(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm", ".xhtml":
		return true
	}
	return false
}

// xmlByName keeps a generic XML finding generic unless the file carries the
// backup app's own naming.
func xmlByName(name string) string {
	if backupFileName.MatchString(name) {
		return TypeSMSBackupXML
	}
	return TypeXML
}

// Conditions a repair report can put a source in.
const (
	ConditionTruncated = "truncated"
	ConditionDamaged   = "damaged"
	ConditionClean     = "clean"
	// ConditionUnreadable is a source whose content the detector could not recognize at all
	// (confidence below 0.5, "content inconclusive") and whose repair preview is lossy or missing:
	// scrambled bytes rather than damaged markup.
	ConditionUnreadable = "unreadable"
	ConditionUnassessed = "unassessed"
)

// RepairReport is the part of repair.preview's report the proposer reads.
type RepairReport struct {
	Present      bool
	Clean        bool
	Truncated    bool
	Lossy        int64
	ChunksFailed int64
	DetectionFmt string
	// ContentInconclusive is repair.detect saying the content gave no format evidence and the answer
	// fell back to the extension hint.
	ContentInconclusive bool
}

// ParseRepairEvidence reads repair.detect / repair.preview output in any of
// the shapes the platform stores or relays: {"detection":{...}},
// {"report":{...}}, or both merged. Unknown shapes yield an absent report.
func ParseRepairEvidence(documents ...json.RawMessage) RepairReport {
	var out RepairReport
	for _, raw := range documents {
		if len(raw) == 0 {
			continue
		}
		var payload struct {
			Detection *struct {
				Fmt        string   `json:"fmt"`
				Confidence *float64 `json:"confidence"`
				Notes      []string `json:"notes"`
			} `json:"detection"`
			Report *struct {
				Clean        *bool `json:"clean"`
				Truncated    bool  `json:"truncated"`
				Lossy        int64 `json:"lossy"`
				ChunksFailed int64 `json:"chunks_failed"`
			} `json:"report"`
			// A bare report object (the `report` value on its own).
			Clean        *bool `json:"clean"`
			Truncated    *bool `json:"truncated"`
			Lossy        int64 `json:"lossy"`
			ChunksFailed int64 `json:"chunks_failed"`
		}
		if json.Unmarshal(raw, &payload) != nil {
			continue
		}
		if payload.Detection != nil && out.DetectionFmt == "" {
			out.DetectionFmt = strings.TrimSpace(payload.Detection.Fmt)
		}
		if payload.Detection != nil {
			for _, note := range payload.Detection.Notes {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(note)), "content inconclusive") {
					out.ContentInconclusive = true
				}
			}
			if payload.Detection.Confidence != nil && *payload.Detection.Confidence >= 0.5 {
				out.ContentInconclusive = false
			}
		}
		switch {
		case payload.Report != nil && payload.Report.Clean != nil:
			out.Present, out.Clean = true, *payload.Report.Clean
			out.Truncated, out.Lossy, out.ChunksFailed = payload.Report.Truncated, payload.Report.Lossy, payload.Report.ChunksFailed
		case payload.Clean != nil:
			out.Present, out.Clean = true, *payload.Clean
			out.Lossy, out.ChunksFailed = payload.Lossy, payload.ChunksFailed
			if payload.Truncated != nil {
				out.Truncated = *payload.Truncated
			}
		}
	}
	return out
}

// Condition names what the repair report says about the source.
func (r RepairReport) Condition() string {
	switch {
	case r.ContentInconclusive && (!r.Present || !r.Clean):
		return ConditionUnreadable
	case !r.Present:
		return ConditionUnassessed
	case r.Truncated:
		return ConditionTruncated
	case !r.Clean:
		return ConditionDamaged
	default:
		return ConditionClean
	}
}

// accepts reports whether a step declaring inputTypes can read sourceType.
func accepts(inputTypes []string, sourceType string) bool {
	for _, candidate := range inputTypes {
		if candidate == TypeAny || candidate == sourceType {
			return true
		}
	}
	return false
}
