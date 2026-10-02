// Byline: Claude Code · Opus 5.5 · 2026-10-01
//
// Package caseidentity is the read model and the owner's edits behind the
// Workbench Case page: the case header, the people of the case, every
// identifier each person used (names, phones, emails, accounts) and what the
// platform has extracted for each of them.
//
// Owner order 2026-10-01 07:56/07:57: registry is the ONE identity store; the
// Case page edits it; ingest, search, the legal desk and the toolkit read it;
// edits version, never overwrite. An identifier edit is a new
// registry.entity_alias row that supersedes exactly the current row of its
// chain; a case-header or person edit updates the registry row and appends its
// before/after to registry.identity_change in the same transaction. Nothing is
// ever deleted: retiring an identifier is a status.
package caseidentity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Limits shared by validation and the schema's CHECK constraints.
const (
	MaxValueBytes     = 512
	MaxPeriodBytes    = 200
	MaxTextBytes      = 4000
	MaxKeyBytes       = 200
	MaxHistoryItems   = 200
	MaxUnknownItems   = 100
	MaxCountKeys      = 2000
	MaxShortNameRunes = 64
)

// Errors the HTTP layer maps to status codes.
var (
	ErrNotFound            = errors.New("case identity record not found")
	ErrStale               = errors.New("the record changed since it was read; reload and try again")
	ErrIdempotencyConflict = errors.New("this Idempotency-Key was already used for a different edit")
	ErrNotInstalled        = errors.New("the case identity registry objects are not installed")
	ErrRejected            = errors.New("the registry rejected the value")
)

// Mode names which case the page shows: the Workbench's TEST (DEV sentinel)
// or REAL (go-live) identity.
type Mode string

const (
	ModeTest Mode = "TEST"
	ModeReal Mode = "REAL"
)

// ParseMode accepts exactly TEST or REAL.
func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.TrimSpace(raw)) {
	case ModeTest:
		return ModeTest, nil
	case ModeReal:
		return ModeReal, nil
	}
	return "", errors.New("mode must be TEST or REAL")
}

// Matter is registry.matter.
type Matter struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Description       *string   `json:"description"`
	Status            string    `json:"status"`
	VerificationState string    `json:"verification_state"`
	CreatedBy         string    `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CourtCase is the matter's primary registry.court_case.
type CourtCase struct {
	ID                string    `json:"id"`
	MatterID          string    `json:"matter_id"`
	Caption           string    `json:"caption"`
	DocketNumber      *string   `json:"docket_number"`
	CourtName         *string   `json:"court_name"`
	Jurisdiction      *string   `json:"jurisdiction"`
	CaseType          *string   `json:"case_type"`
	PresidingJudge    *string   `json:"presiding_judge"`
	Status            string    `json:"status"`
	FiledOn           *string   `json:"filed_on"`
	ClosedOn          *string   `json:"closed_on"`
	VerificationState string    `json:"verification_state"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// IdentifierVersion is one row of an alias chain.
type IdentifierVersion struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	Period       *string   `json:"period"`
	Basis        *string   `json:"basis"`
	ChangeReason *string   `json:"change_reason"`
	RecordedBy   string    `json:"recorded_by"`
	RecordedAt   time.Time `json:"recorded_at"`
	SupersedesID *string   `json:"supersedes_id"`
}

// Identifier is the current row of one alias chain plus its earlier rows.
type Identifier struct {
	IdentifierVersion
	EntityID   string              `json:"entity_id"`
	RawValue   string              `json:"raw_value"`
	Kind       string              `json:"kind"`
	Normalized string              `json:"normalized"`
	History    []IdentifierVersion `json:"history"`
}

// Person is registry.person joined to its registry.entity.
type Person struct {
	ID                string       `json:"id"`
	DisplayName       string       `json:"display_name"`
	CanonicalName     *string      `json:"canonical_name"`
	ShortName         *string      `json:"short_name"`
	RoleInCase        *string      `json:"role_in_case"`
	ConnectionTo      *string      `json:"connection_to"`
	RelationshipType  *string      `json:"relationship_type"`
	IsMinor           bool         `json:"is_minor"`
	IsParty           bool         `json:"is_party"`
	Notes             *string      `json:"notes"`
	VerificationState string       `json:"verification_state"`
	Identifiers       []Identifier `json:"identifiers"`
}

// Change is one registry.identity_change row.
type Change struct {
	ID           string          `json:"id"`
	SubjectTable string          `json:"subject_table"`
	SubjectID    string          `json:"subject_id"`
	Before       json.RawMessage `json:"before"`
	After        json.RawMessage `json:"after"`
	ChangeReason string          `json:"change_reason"`
	RecordedBy   string          `json:"recorded_by"`
	RecordedAt   time.Time       `json:"recorded_at"`
}

// Count is what Probata's working tables hold for one key (an identifier's
// normalized form, or a person's entity id for resolved participants).
type Count struct {
	Key     string     `json:"key"`
	Source  string     `json:"source"`
	Events  int64      `json:"events"`
	FirstAt *time.Time `json:"first_at"`
	LastAt  *time.Time `json:"last_at"`
}

// Unknown is an identifier Probata saw on a participant that no person carries.
type Unknown struct {
	Normalized string     `json:"normalized"`
	RawValue   string     `json:"raw_value"`
	Events     int64      `json:"events"`
	FirstAt    *time.Time `json:"first_at"`
	LastAt     *time.Time `json:"last_at"`
}

// Dismissal is an identifier the owner set aside.
type Dismissal struct {
	Normalized string    `json:"normalized"`
	RawValue   string    `json:"raw_value"`
	Basis      string    `json:"basis"`
	RecordedBy string    `json:"recorded_by"`
	RecordedAt time.Time `json:"recorded_at"`
}

// View is the whole Case page read.
type View struct {
	Mode       Mode        `json:"mode"`
	Matter     *Matter     `json:"matter"`
	CourtCase  *CourtCase  `json:"court_case"`
	People     []Person    `json:"people"`
	History    []Change    `json:"history"`
	Counts     []Count     `json:"probata_counts"`
	Unknowns   []Unknown   `json:"probata_unknowns"`
	Dismissed  []Dismissal `json:"dismissed"`
	CountStore string      `json:"count_store"`
}

// Actor is the authenticated owner act.
type Actor struct {
	SubjectUID     string
	Username       string
	IdempotencyKey string
}

// StoredKey is the idempotency key persisted for one act: scoped to the actor
// and the operation, so two people or two kinds of edit can never collide.
func (a Actor) StoredKey(operation string) string {
	sum := sha256.Sum256([]byte(a.SubjectUID + "\x00" + operation + "\x00" + a.IdempotencyKey))
	return operation + ":" + hex.EncodeToString(sum[:])
}

// IdentifierSpec adds an identifier to a person (SupersedesID empty) or writes
// a new version of an existing chain (SupersedesID = the chain's current row).
type IdentifierSpec struct {
	EntityID     string  `json:"entity_id"`
	RawValue     string  `json:"raw_value"`
	Kind         string  `json:"kind"`
	Status       string  `json:"status"`
	Period       *string `json:"period"`
	Basis        string  `json:"basis"`
	ChangeReason string  `json:"change_reason"`
	SupersedesID string  `json:"supersedes_id"`
}

// HeaderSpec edits the matter or its court case. Fields maps a column to its
// new value; nil clears a nullable column.
type HeaderSpec struct {
	Target       string             `json:"target"`
	ID           string             `json:"id"`
	Fields       map[string]*string `json:"fields"`
	ChangeReason string             `json:"change_reason"`
	ExpectedAt   *time.Time         `json:"expected_updated_at"`
}

// PersonSpec edits one person. Fields maps a column to its new value.
type PersonSpec struct {
	ID           string             `json:"id"`
	Fields       map[string]*string `json:"fields"`
	ChangeReason string             `json:"change_reason"`
}

// NewPersonSpec adds a person to the registry (third parties, children,
// counsel). The two parties are minted by the go-live step, not here.
type NewPersonSpec struct {
	DisplayName  string  `json:"display_name"`
	ShortName    *string `json:"short_name"`
	RoleInCase   string  `json:"role_in_case"`
	ConnectionTo string  `json:"connection_to"`
	IsMinor      bool    `json:"is_minor"`
	Notes        *string `json:"notes"`
	ChangeReason string  `json:"change_reason"`
}

// TriageSpec records the owner's decision on an identifier tied to nobody.
type TriageSpec struct {
	RawValue string `json:"raw_value"`
	Decision string `json:"decision"`
	Basis    string `json:"basis"`
}

// Receipt answers every write.
type Receipt struct {
	Ref        string    `json:"ref"`
	Kind       string    `json:"kind"`
	RecordedAt time.Time `json:"recorded_at"`
	Replayed   bool      `json:"replayed"`
}

// Store is the registry boundary.
type Store interface {
	Read(ctx context.Context, mode Mode) (View, error)
	WriteIdentifier(ctx context.Context, spec IdentifierSpec, actor Actor) (Receipt, error)
	EditHeader(ctx context.Context, mode Mode, spec HeaderSpec, actor Actor) (Receipt, error)
	EditPerson(ctx context.Context, spec PersonSpec, actor Actor) (Receipt, error)
	AddPerson(ctx context.Context, spec NewPersonSpec, actor Actor) (Receipt, error)
	Triage(ctx context.Context, spec TriageSpec, actor Actor) (Receipt, error)
	Lookup(ctx context.Context, values []string) ([]Match, error)
}

// Match answers "who used this identifier?" for another application.
type Match struct {
	Query      string  `json:"query"`
	Normalized string  `json:"normalized"`
	EntityID   *string `json:"entity_id"`
	Person     *string `json:"person"`
	RawValue   *string `json:"raw_value"`
	Kind       *string `json:"kind"`
	Status     *string `json:"status"`
	Period     *string `json:"period"`
}

// Identifier kinds the page writes. The registry also admits the extraction
// lane's finer name kinds (nickname, legal, ...), which are read as names.
var identifierKinds = map[string]bool{"name": true, "phone": true, "email": true, "account": true, "handle": true, "nickname": true, "legal": true, "maiden": true, "misspelling": true, "other": true}

var identifierStatuses = map[string]bool{"confirmed": true, "candidate": true, "retired": true}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateUUID requires a canonical UUID.
func ValidateUUID(label, value string) error {
	if !uuidPattern.MatchString(value) {
		return fmt.Errorf("%s must be a UUID", label)
	}
	return nil
}

// ValidateActor requires the Authentik actor and a bounded Idempotency-Key.
func ValidateActor(actor Actor) error {
	for _, value := range []string{actor.SubjectUID, actor.Username} {
		if strings.TrimSpace(value) == "" || len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("Authentik actor identity is required")
		}
	}
	key := actor.IdempotencyKey
	if strings.TrimSpace(key) == "" || len(key) > MaxKeyBytes || strings.ContainsAny(key, "\x00\r\n") {
		return fmt.Errorf("a bounded Idempotency-Key of at most %d bytes is required", MaxKeyBytes)
	}
	return nil
}

func validText(label, value string, required bool, limit int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", label)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", label)
	}
	if len(value) > limit {
		return fmt.Errorf("%s is longer than %d bytes", label, limit)
	}
	for _, r := range value {
		if r == 0 || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return fmt.Errorf("%s cannot hold control characters", label)
		}
	}
	return nil
}

// ValidateIdentifier checks an identifier write before it reaches PostgreSQL.
// The raw spelling is kept exactly as given (no trimming of inner spacing):
// only surrounding whitespace is refused, so what the owner typed is what is
// stored.
func ValidateIdentifier(spec IdentifierSpec) error {
	if err := ValidateUUID("entity_id", spec.EntityID); err != nil {
		return err
	}
	if spec.SupersedesID != "" {
		if err := ValidateUUID("supersedes_id", spec.SupersedesID); err != nil {
			return err
		}
	}
	if err := validText("raw_value", spec.RawValue, true, MaxValueBytes); err != nil {
		return err
	}
	if strings.TrimSpace(spec.RawValue) != spec.RawValue || strings.ContainsAny(spec.RawValue, "\n\t") {
		return errors.New("raw_value cannot start or end with spaces or hold line breaks")
	}
	if !identifierKinds[spec.Kind] {
		return errors.New("kind must be name, phone, email, account, handle, nickname, legal, maiden, misspelling or other")
	}
	if !identifierStatuses[spec.Status] {
		return errors.New("status must be confirmed, candidate or retired")
	}
	if spec.Period != nil {
		if err := validText("period", *spec.Period, false, MaxPeriodBytes); err != nil {
			return err
		}
	}
	if err := validText("basis", spec.Basis, true, MaxTextBytes); err != nil {
		return err
	}
	if spec.SupersedesID != "" {
		return validText("change_reason", spec.ChangeReason, true, MaxTextBytes)
	}
	return validText("change_reason", spec.ChangeReason, false, MaxTextBytes)
}

// HeaderColumns are the editable columns of the case header, per target.
var HeaderColumns = map[string]map[string]bool{
	"matter":     {"title": true, "description": true, "status": true, "verification_state": true},
	"court_case": {"caption": true, "docket_number": true, "court_name": true, "jurisdiction": true, "case_type": true, "presiding_judge": true, "status": true, "filed_on": true, "closed_on": true, "verification_state": true},
}

// requiredHeaderColumns cannot be cleared.
var requiredHeaderColumns = map[string]bool{"title": true, "caption": true, "status": true, "verification_state": true}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateHeader checks a case-header edit.
func ValidateHeader(spec HeaderSpec) error {
	columns, ok := HeaderColumns[spec.Target]
	if !ok {
		return errors.New("target must be matter or court_case")
	}
	if err := ValidateUUID("id", spec.ID); err != nil {
		return err
	}
	if len(spec.Fields) == 0 {
		return errors.New("at least one field must change")
	}
	for column, value := range spec.Fields {
		if !columns[column] {
			return fmt.Errorf("%s is not an editable %s field", column, spec.Target)
		}
		if value == nil {
			if requiredHeaderColumns[column] {
				return fmt.Errorf("%s cannot be empty", column)
			}
			continue
		}
		if err := validText(column, *value, requiredHeaderColumns[column], MaxTextBytes); err != nil {
			return err
		}
		if (column == "filed_on" || column == "closed_on") && !datePattern.MatchString(*value) {
			return fmt.Errorf("%s must be a date (YYYY-MM-DD)", column)
		}
	}
	return validText("change_reason", spec.ChangeReason, true, MaxTextBytes)
}

// PersonColumns are the editable person fields. display_name and
// canonical_name live on registry.entity; the rest on registry.person.
var PersonColumns = map[string]string{
	"display_name": "entity", "canonical_name": "entity",
	"short_name": "person", "role_in_case": "person", "connection_to": "person",
	"relationship_type": "person", "notes": "person", "is_minor": "person", "verification_state": "person",
}

// ValidatePerson checks a person edit.
func ValidatePerson(spec PersonSpec) error {
	if err := ValidateUUID("id", spec.ID); err != nil {
		return err
	}
	if len(spec.Fields) == 0 {
		return errors.New("at least one field must change")
	}
	for column, value := range spec.Fields {
		if _, ok := PersonColumns[column]; !ok {
			return fmt.Errorf("%s is not an editable person field", column)
		}
		if value == nil {
			if column == "display_name" || column == "is_minor" || column == "verification_state" {
				return fmt.Errorf("%s cannot be empty", column)
			}
			continue
		}
		if err := validText(column, *value, column == "display_name", MaxTextBytes); err != nil {
			return err
		}
		if column == "is_minor" && *value != "true" && *value != "false" {
			return errors.New("is_minor must be true or false")
		}
		if column == "short_name" && utf8.RuneCountInString(*value) > MaxShortNameRunes {
			return fmt.Errorf("short_name is longer than %d characters", MaxShortNameRunes)
		}
	}
	return validText("change_reason", spec.ChangeReason, true, MaxTextBytes)
}

// ValidateNewPerson checks a new person.
func ValidateNewPerson(spec NewPersonSpec) error {
	if err := validText("display_name", spec.DisplayName, true, MaxValueBytes); err != nil {
		return err
	}
	if strings.TrimSpace(spec.DisplayName) != spec.DisplayName {
		return errors.New("display_name cannot start or end with spaces")
	}
	if spec.ShortName != nil && (strings.TrimSpace(*spec.ShortName) == "" || utf8.RuneCountInString(*spec.ShortName) > MaxShortNameRunes) {
		return fmt.Errorf("short_name must be 1-%d characters", MaxShortNameRunes)
	}
	for label, value := range map[string]string{"role_in_case": spec.RoleInCase, "connection_to": spec.ConnectionTo} {
		if err := validText(label, value, true, 64); err != nil {
			return err
		}
	}
	if spec.Notes != nil {
		if err := validText("notes", *spec.Notes, false, MaxTextBytes); err != nil {
			return err
		}
	}
	return validText("change_reason", spec.ChangeReason, true, MaxTextBytes)
}

// ValidateTriage checks an unknowns-queue decision.
func ValidateTriage(spec TriageSpec) error {
	if err := validText("raw_value", spec.RawValue, true, MaxValueBytes); err != nil {
		return err
	}
	if spec.Decision != "dismissed" && spec.Decision != "reopened" {
		return errors.New("decision must be dismissed or reopened")
	}
	return validText("basis", spec.Basis, true, MaxTextBytes)
}

// ValidateLookup bounds an identifier lookup.
func ValidateLookup(values []string) error {
	if len(values) == 0 || len(values) > 200 {
		return errors.New("between 1 and 200 identifiers are required")
	}
	for _, value := range values {
		if err := validText("identifier", value, true, MaxValueBytes); err != nil {
			return err
		}
	}
	return nil
}

// NormIdentifier is registry.norm_identifier in Go, for callers that must
// agree with the database without a round trip (and for tests). The database
// function is the authority; TestNormIdentifierMatchesTheSQLRule pins the two.
func NormIdentifier(p string) string {
	if strings.Trim(p, " ") == "" {
		return ""
	}
	var digits strings.Builder
	for _, r := range p {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	nanp := func(s string) bool { return len(s) == 10 && s[0] >= '2' && s[0] <= '9' }
	hasLetterOrAt := strings.IndexFunc(p, func(r rune) bool { return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '@' }) >= 0
	trimmed := strings.TrimLeft(p, " \t\n\r\f\v")
	switch {
	case len(d) == 11 && d[0] == '1' && nanp(d[1:]):
		return d[1:]
	case nanp(d):
		return d
	case len(d) > 11 && nanp(d[len(d)-10:]) && (strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#")):
		return d[len(d)-10:]
	case d != "" && !hasLetterOrAt:
		return d
	}
	return strings.ToLower(strings.Trim(p, " "))
}
