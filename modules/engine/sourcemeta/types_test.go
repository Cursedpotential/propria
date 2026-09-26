// Byline: Claude Code · Opus 5.5 · 2026-09-26
package sourcemeta

import (
	"encoding/json"
	"strings"
	"testing"
)

func validCorrection() CorrectionSpec {
	return CorrectionSpec{
		PreviewHandle: strings.Repeat("h", 32), SubjectSHA256: strings.Repeat("ab", 32),
		FieldKey: "embedded:EXIF:DateTimeOriginal", Action: ActionCorrect,
		SourceValue: json.RawMessage(`"2021:05:04 10:11:12"`), CorrectedValue: json.RawMessage(`"2021:05:04 22:11:12"`),
		ChangeReason: "clock was off", ActorSubjectUID: "uid", ActorUsername: "owner", IdempotencyKey: "key",
	}
}

func TestValidateCorrection(t *testing.T) {
	if err := ValidateCorrection(validCorrection()); err != nil {
		t.Fatal(err)
	}
	retract := validCorrection()
	retract.Action, retract.CorrectedValue, retract.SupersedesRef = ActionRetract, nil, "0190a000-0000-7000-8000-0000000000c1"
	if err := ValidateCorrection(retract); err != nil {
		t.Fatalf("retract: %v", err)
	}
	cases := map[string]func(*CorrectionSpec){
		"uppercase digest":     func(s *CorrectionSpec) { s.SubjectSHA256 = strings.Repeat("AB", 32) },
		"no origin":            func(s *CorrectionSpec) { s.FieldKey = "DateTimeOriginal" },
		"control in key":       func(s *CorrectionSpec) { s.FieldKey = "embedded:EXIF:\u0001" },
		"key too long":         func(s *CorrectionSpec) { s.FieldKey = "embedded:" + strings.Repeat("x", MaxFieldKeyLen) },
		"json null correction": func(s *CorrectionSpec) { s.CorrectedValue = json.RawMessage(`null`) },
		"invalid json":         func(s *CorrectionSpec) { s.CorrectedValue = json.RawMessage(`{`) },
		"oversized value": func(s *CorrectionSpec) {
			s.CorrectedValue = json.RawMessage(`"` + strings.Repeat("x", MaxValueBytes) + `"`)
		},
		"retract with value": func(s *CorrectionSpec) {
			s.Action = ActionRetract
			s.SupersedesRef = "0190a000-0000-7000-8000-0000000000c1"
		},
		"retract with no target": func(s *CorrectionSpec) { s.Action, s.CorrectedValue = ActionRetract, nil },
		"unknown action":         func(s *CorrectionSpec) { s.Action = "delete" },
		"blank reason":           func(s *CorrectionSpec) { s.ChangeReason = "  " },
		"no actor":               func(s *CorrectionSpec) { s.ActorUsername = "" },
		"no key":                 func(s *CorrectionSpec) { s.IdempotencyKey = "" },
		"bad supersedes":         func(s *CorrectionSpec) { s.SupersedesRef = "nope" },
	}
	for name, mutate := range cases {
		spec := validCorrection()
		mutate(&spec)
		if err := ValidateCorrection(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCorrectionDigestIgnoresJSONWhitespaceButNotContent(t *testing.T) {
	a := validCorrection()
	b := validCorrection()
	b.CorrectedValue = json.RawMessage(` "2021:05:04 22:11:12" `)
	if CorrectionDigest(a) != CorrectionDigest(b) {
		t.Fatal("whitespace changed the digest")
	}
	b.CorrectedValue = json.RawMessage(`"2021:05:04 23:11:12"`)
	if CorrectionDigest(a) == CorrectionDigest(b) {
		t.Fatal("a different value kept the digest")
	}
}

func TestValidateSHA256(t *testing.T) {
	if err := ValidateSHA256("", false); err != nil {
		t.Fatal("empty optional digest rejected")
	}
	if err := ValidateSHA256("", true); err == nil {
		t.Fatal("empty required digest accepted")
	}
	if _, err := DecodeSHA256(strings.Repeat("0f", 32)); err != nil {
		t.Fatal(err)
	}
}
