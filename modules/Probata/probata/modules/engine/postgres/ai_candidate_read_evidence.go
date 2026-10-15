package postgres

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

type nativeCandidateReadBundle struct {
	ContractVersion string                   `json:"contract_version"`
	Stage           string                   `json:"stage"`
	Source          aicontextsource.Identity `json:"source"`
	PreparedRef     string                   `json:"prepared_ref"`
	Candidates      []struct {
		Kind        string     `json:"kind"`
		Name        string     `json:"name"`
		EntityType  string     `json:"entity_type"`
		EventType   string     `json:"event_type"`
		Predicate   string     `json:"predicate"`
		Statement   string     `json:"statement"`
		Quote       string     `json:"quote"`
		Status      string     `json:"status"`
		Confidence  *float64   `json:"confidence"`
		OccurredAt  *time.Time `json:"occurred_at"`
		Occurrences []struct {
			NativeJSONPointer   string     `json:"native_json_pointer"`
			Start               int        `json:"start"`
			End                 int        `json:"end"`
			Unit                string     `json:"unit"`
			SourceAvailableFrom *time.Time `json:"source_available_from"`
		} `json:"occurrences"`
	} `json:"candidates"`
}

type nativeReadEvidence struct {
	pin       service.AISourcePin
	bundleSHA string
	quotes    map[string]string
}
type nativeReadPrepared struct {
	bundle   aicontextsource.Prepared
	original []byte
	document any
}
type nativeCandidateReadPage struct {
	bundles      map[string]nativeReadEvidence
	prepared     map[string]nativeReadPrepared
	bytesRead    int
	readFile     func(string, string) ([]byte, string, error)
	readPrepared func(string) (aicontextsource.Prepared, []byte, error)
}

// newNativeCandidateReadPage creates an isolated bounded evidence cache for one candidate response page.
// Inputs: none. Outputs: read-only bundle/original adapters and empty page-local caches.
// Effects: none; choose instead of rereading the same immutable evidence for each displayed row.
func newNativeCandidateReadPage() *nativeCandidateReadPage {
	return &nativeCandidateReadPage{bundles: map[string]nativeReadEvidence{}, prepared: map[string]nativeReadPrepared{}, readFile: aicontextsource.ReadFile, readPrepared: aicontextsource.ReadPrepared}
}

// hydrate restores only a verified external quote in the returned native candidate without altering its saved digest.
// Inputs: requested registered source and PostgreSQL-safe review row. Outputs: response-only candidate JSON or refusal.
// Effects: bounded read-only external file access; no database adapter is available to this helper.
// Choose for native SourceRef candidates; legacy retained-object rows pass through byte-for-byte.
func (p *nativeCandidateReadPage) hydrate(sourceVersionID string, row service.AIReviewRow) (service.AIReviewRow, error) {
	var candidate service.AICandidate
	if err := json.Unmarshal(row.Candidate, &candidate); err != nil {
		return row, service.ErrInvalid{Err: errors.New("candidate review metadata is malformed")}
	}
	if candidate.SourceRef == "" {
		return row, nil
	}
	invalid := func() (service.AIReviewRow, error) {
		return row, service.ErrInvalid{Err: errors.New("native candidate external evidence does not match its saved source or content")}
	}
	digest := service.AICandidateDigest(candidate)
	if candidate.SourceVersionID != sourceVersionID || candidate.SourceObjectID != "" || candidate.PreparedRef == "" || candidate.EvidenceQuote != "" || candidate.EvidenceBundleRef == "" || !aicontextsource.ValidDigest(candidate.EvidenceBundleSHA256) || hex.EncodeToString(digest[:]) != row.ContentSHA256 || candidate.Kind != row.Kind || candidate.ReportedKind != row.ReportedKind || candidate.ReviewDomain != row.ReviewDomain {
		return invalid()
	}
	evidence, ok := p.bundles[candidate.EvidenceBundleRef]
	if !ok {
		var err error
		evidence, err = p.load(candidate)
		if err != nil {
			return row, service.ErrInvalid{Err: err}
		}
		p.bundles[candidate.EvidenceBundleRef] = evidence
	}
	if evidence.bundleSHA != candidate.EvidenceBundleSHA256 || !reflect.DeepEqual(evidence.pin, candidate.AISourcePin) {
		return invalid()
	}
	quote, ok := evidence.quotes[row.ContentSHA256]
	if !ok {
		return invalid()
	}
	candidate.EvidenceQuote = quote
	response, err := json.Marshal(candidate)
	if err != nil {
		return row, err
	}
	row.Candidate = response
	return row, nil
}

// load verifies one immutable native candidate bundle against its exact copied original and builds a digest-bound quote index.
// Inputs: saved source/prepared/bundle pins. Outputs: only verified short quote mappings.
// Effects: read-only mounted files, at most eight distinct bundles and 64 MiB of source/bundle bytes per page.
// Choose once per unique bundle on a response page; it never changes the persisted projection or decision digest.
func (p *nativeCandidateReadPage) load(candidate service.AICandidate) (nativeReadEvidence, error) {
	out := nativeReadEvidence{pin: candidate.AISourcePin, bundleSHA: candidate.EvidenceBundleSHA256, quotes: map[string]string{}}
	if len(p.bundles) >= 8 {
		return out, errors.New("candidate read page exceeds its external evidence bundle bound")
	}
	raw, bundlePath, err := p.readFile(candidate.EvidenceBundleRef, "candidates.json")
	if err != nil {
		return out, errors.New("native candidate evidence bundle is unavailable")
	}
	p.bytesRead += len(raw)
	if aicontextsource.Hash(raw) != candidate.EvidenceBundleSHA256 {
		return out, errors.New("native candidate evidence bundle SHA256 differs")
	}
	prepared, ok := p.prepared[candidate.PreparedRef]
	if !ok {
		if len(p.prepared) >= 8 {
			return out, errors.New("candidate read page exceeds its prepared-source bound")
		}
		prepared.bundle, prepared.original, err = p.readPrepared(candidate.PreparedRef)
		if err != nil {
			return out, errors.New("native candidate prepared original is unavailable or changed")
		}
		p.bytesRead += len(prepared.original)
		if p.bytesRead > 64<<20 {
			return out, errors.New("candidate read page exceeds its external evidence byte bound")
		}
		switch prepared.bundle.Source.SourceFormat {
		case "gemini_markdown", "chatgpt_markdown", "claude_markdown":
			if !utf8.Valid(prepared.original) {
				return out, errors.New("native candidate original text is not UTF-8")
			}
			prepared.document = string(prepared.original)
		case "chatgpt", "claude", "claude_ai_export_json", "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json":
			if json.Unmarshal(prepared.original, &prepared.document) != nil {
				return out, errors.New("native candidate original JSON is malformed")
			}
		default:
			return out, errors.New("native candidate original format is unsupported")
		}
		p.prepared[candidate.PreparedRef] = prepared
	}
	if p.bytesRead > 64<<20 {
		return out, errors.New("candidate read page exceeds its external evidence byte bound")
	}
	preparedURL, err := url.Parse(candidate.PreparedRef)
	if err != nil || filepath.Dir(bundlePath) != filepath.Dir(filepath.FromSlash(preparedURL.Path)) {
		return out, errors.New("native candidate bundle and prepared source directories differ")
	}
	var bundle nativeCandidateReadBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return out, errors.New("native candidate evidence bundle is malformed")
	}
	if bundle.ContractVersion != "ai-context-v1" || bundle.Stage != "candidates" || bundle.PreparedRef != candidate.PreparedRef || !reflect.DeepEqual(bundle.Source, prepared.bundle.Source) || bundle.Source.SourceVersionID != candidate.SourceVersionID || bundle.Source.SourceRef != candidate.SourceRef || !sameVersionID(bundle.Source.ProviderVersionID, candidate.VersionID) || prepared.bundle.OriginalSHA256 != candidate.SourceSHA256 || (bundle.Source.SourceSHA256 != "" && bundle.Source.SourceSHA256 != candidate.SourceSHA256) || len(bundle.Candidates) > 500 {
		return out, errors.New("native candidate bundle source/version/prepared pins differ")
	}
	occurrences := 0
	verified := []service.AICandidate{}
	for _, item := range bundle.Candidates {
		if item.Kind != "entity" && item.Kind != "event" || item.Kind == "event" && item.OccurredAt == nil {
			continue
		}
		if item.Status != "unreviewed_context_candidate" || item.Confidence == nil || len(item.Quote) < 1 || len(item.Quote) > 1000 {
			continue
		}
		for _, occ := range item.Occurrences {
			occurrences++
			if occurrences > 500 {
				return out, errors.New("native candidate evidence occurrence count exceeds bound")
			}
			spanSHA := aicontextsource.Hash([]byte(item.Quote))
			full := service.AICandidate{AISourcePin: candidate.AISourcePin, NativeJSONPointer: occ.NativeJSONPointer, SourceSpan: service.AISourceSpan{Start: occ.Start, End: occ.End, SHA256: spanSHA}, SpanUnit: occ.Unit, Kind: item.Kind, ReportedKind: item.Kind, ReviewDomain: "ai_chat_content", Name: item.Name, EntityType: item.EntityType, EventType: item.EventType, Predicate: item.Predicate, Statement: item.Statement, EvidenceQuote: item.Quote, EvidenceBundleRef: candidate.EvidenceBundleRef, EvidenceBundleSHA256: candidate.EvidenceBundleSHA256, OccurredAt: item.OccurredAt, SourceAvailableFrom: occ.SourceAvailableFrom, Confidence: *item.Confidence}
			if err := service.ValidateAICandidate(full); err != nil {
				continue
			}
			verified = append(verified, full)
			persisted, err := aiCandidateForPersistence(full)
			if err != nil {
				return out, err
			}
			hash := service.AICandidateDigest(persisted)
			out.quotes[hex.EncodeToString(hash[:])] = item.Quote
		}
	}
	if err := verifyAISpans(prepared.document, verified); err != nil {
		return out, errors.New("native candidate quote differs from its exact original pointer/span/hash")
	}
	return out, nil
}
