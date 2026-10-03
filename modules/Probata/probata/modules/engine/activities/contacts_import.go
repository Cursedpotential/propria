// Byline: Claude Code · Sonnet · 2026-10-02
//
// The six Activities of ContactsImportWorkflow (package contacts). Each does ONE thing, takes and returns
// references and counts (never contact payloads through Temporal history), reports heartbeats, and
// returns a Receipt. The registry is only ever written through caseidentity.Store, the governed boundary
// the Workbench's Case page uses, so every change is validated, attributed to the owner who started the
// run, and appended to registry.identity_change. Dry run rolls the writes back.
package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contacts"
)

// ContactsCatalog lists the contact exports in the Case Bible catalog (raw_duck.b2_objects), one key per
// sha1 (the most recently listed). Read-only.
type ContactsCatalog interface {
	ContactFiles(ctx context.Context) ([]contacts.File, error)
}

// ContactsFetcher copies one catalog object into w and returns its size and its SHA-1 and SHA-256 (computed
// while copying), so the catalog's own hash can be checked whichever it holds.
type ContactsFetcher interface {
	Fetch(ctx context.Context, bucket, key string, w io.Writer) (size int64, sha1Hex, sha256Hex string, err error)
}

// ContactsRegistry is the SQL boundary of the placeholder and re-link steps. UnlinkedNumbers are the numbers on
// imported rows with a NULL entity column that NO person carries; CarriedEntities answers who carries
// numbers now; RelinkSweep fills only NULL entity columns for numbers a person carries (dryRun rolls back).
type ContactsRegistry interface {
	UnlinkedNumbers(ctx context.Context) ([]string, error)
	CarriedEntities(ctx context.Context, numbers []string) (map[string]string, error)
	RelinkSweep(ctx context.Context, dryRun bool) (map[string]int64, error)
}

// ContactsStore is the governed case-identity boundary the steps write through.
type ContactsStore interface {
	AddContactPeople(ctx context.Context, spec caseidentity.ContactPeopleSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
	AddPlaceholders(ctx context.Context, spec caseidentity.PlaceholderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
	AddIdentifier(ctx context.Context, spec caseidentity.IdentifierSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
}

// ContactsActivities implements the six Activities.
type ContactsActivities struct {
	Catalog  ContactsCatalog
	Fetcher  ContactsFetcher
	Registry ContactsRegistry
	Store    ContactsStore
	// WorkRoot is a shared folder; each run keeps its manifest, files and people under <WorkRoot>/<run_id>.
	WorkRoot string
	Now      func() time.Time
	// Heartbeat reports progress to Temporal; nil outside a worker.
	Heartbeat func(ctx context.Context, detail string)
}

// NewContactsActivities binds the heartbeat and clock to Temporal.
func NewContactsActivities(c ContactsCatalog, f ContactsFetcher, r ContactsRegistry, s ContactsStore, workRoot string) ContactsActivities {
	return ContactsActivities{
		Catalog: c, Fetcher: f, Registry: r, Store: s, WorkRoot: workRoot,
		Now:       func() time.Time { return time.Now().UTC() },
		Heartbeat: func(ctx context.Context, detail string) { activity.RecordHeartbeat(ctx, detail) },
	}
}

// RegisterContactsActivities installs the six Activities under their exact workflow names.
func RegisterContactsActivities(registrar ActivityRegistrar, a ContactsActivities) {
	for name, fn := range map[string]any{
		contacts.ManifestActivity:     a.Manifest,
		contacts.FetchActivity:        a.Fetch,
		contacts.ParseActivity:        a.Parse,
		contacts.PeopleActivity:       a.People,
		contacts.PlaceholdersActivity: a.Placeholders,
		contacts.RelinkActivity:       a.Relink,
	} {
		registrar.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
}

const (
	contactReasonPeople       = "contacts import: named by a contact export, unconfirmed (owner 2026-10-02)"
	contactReasonPlaceholders = "contacts import: number named by no contact; placeholder person so no party is ever empty (owner 2026-10-02)"
	contactPeopleBatch        = 100
	contactPlaceholderBatch   = 200
)

func contactsPermanent(err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), "ContactsPermanent", err)
}

func (a ContactsActivities) beat(ctx context.Context, detail string) {
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, detail)
	}
}

func (a ContactsActivities) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now().UTC()
}

func (a ContactsActivities) runDir(req contacts.StepRequest) (string, error) {
	if strings.TrimSpace(a.WorkRoot) == "" {
		return "", contactsPermanent(errors.New("contacts import: no work folder is configured on this worker"))
	}
	if err := contacts.ValidateInput(contacts.Input{RunID: req.RunID, Actor: req.Actor}); err != nil {
		return "", contactsPermanent(err)
	}
	dir := filepath.Join(a.WorkRoot, req.RunID)
	return dir, os.MkdirAll(dir, 0o750)
}

func (a ContactsActivities) actor(req contacts.StepRequest, operation string, n int) caseidentity.Actor {
	return caseidentity.Actor{SubjectUID: req.Actor.SubjectUID, Username: req.Actor.Username, IdempotencyKey: fmt.Sprintf("%s-%s-%05d", req.RunID, operation, n)}
}

func (a ContactsActivities) receipt(step string, req contacts.StepRequest, ref string, digest string, counts map[string]int64, notes ...string) contacts.Receipt {
	return contacts.Receipt{Step: step, DryRun: req.DryRun, Status: "success", Ref: ref, Digest: digest, Counts: counts, Notes: notes, At: a.now()}
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

var hexID = regexp.MustCompile(`^[0-9a-f]{20,64}$`)

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func kindOf(key string) string {
	lowered := strings.ToLower(key)
	switch {
	case strings.HasSuffix(lowered, ".vcf"):
		return "vcard"
	case strings.Contains(lowered, "imported_contacts"), strings.Contains(lowered, "synced_contacts"):
		return "facebook_instagram"
	}
	return "csv_json"
}

// Manifest is contacts_manifest_activity: list the contact exports in the catalog and write manifest.jsonl.
func (a ContactsActivities) Manifest(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	dir, err := a.runDir(req)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if a.Catalog == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: the catalog connection is not configured on this worker"))
	}
	files, err := a.Catalog.ContactFiles(ctx)
	if err != nil {
		return contacts.Receipt{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Key < files[j].Key })
	counts := map[string]int64{"files": int64(len(files))}
	var lines strings.Builder
	for _, file := range files {
		counts["bytes"] += file.Size
		counts["kind_"+kindOf(file.Key)]++
		encoded, err := json.Marshal(file)
		if err != nil {
			return contacts.Receipt{}, err
		}
		lines.Write(encoded)
		lines.WriteByte('\n')
	}
	path := filepath.Join(dir, "manifest.jsonl")
	if err := os.WriteFile(path, []byte(lines.String()), 0o640); err != nil {
		return contacts.Receipt{}, err
	}
	digest, err := fileDigest(path)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if len(files) == 0 {
		return contacts.Receipt{Step: contacts.ManifestActivity, DryRun: req.DryRun, Status: "not_applicable", Ref: path, Digest: digest, Counts: counts,
			Notes: []string{"the catalog lists no contact exports; check the key patterns"}, At: a.now()}, nil
	}
	return a.receipt(contacts.ManifestActivity, req, path, digest, counts), nil
}

func readManifest(path string) ([]contacts.File, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var files []contacts.File
	for _, line := range strings.Split(string(body), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var file contacts.File
		if err := json.Unmarshal([]byte(line), &file); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

// Fetch is contacts_fetch_activity: copy every manifest object from B2 into the work folder, checking each
// object's SHA-1 against the catalog's. A mismatch is removed and counted, never parsed.
func (a ContactsActivities) Fetch(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	dir, err := a.runDir(req)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if a.Fetcher == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: no object store is configured on this worker"))
	}
	manifest := req.Refs["manifest"]
	if manifest == "" || !strings.HasPrefix(filepath.Clean(manifest), filepath.Clean(dir)) {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts fetch requires this run's manifest"))
	}
	files, err := readManifest(manifest)
	if err != nil {
		return contacts.Receipt{}, err
	}
	bucket := strings.TrimSpace(req.Bucket)
	if bucket == "" {
		bucket = "salem-data"
	}
	strip := req.KeyStrip
	if strip == "" {
		strip = "b2://" + bucket + "/"
	}
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0o750); err != nil {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"files": int64(len(files))}
	var notes []string
	var kept []contacts.File
	for index, file := range files {
		a.beat(ctx, fmt.Sprintf("fetching %d of %d", index+1, len(files)))
		id := file.ContentID()
		if !hexID.MatchString(id) { // a catalog value is never trusted as a file name
			id = fmt.Sprintf("%x", sha256.Sum256([]byte(file.Bucket+"/"+file.Key)))
		}
		target := filepath.Join(filesDir, id+strings.ToLower(filepath.Ext(file.Key)))
		fileBucket := firstText(file.Bucket, bucket)
		if info, statErr := os.Stat(target); statErr == nil && (file.Size == 0 || info.Size() == file.Size) {
			counts["reused"]++
			file.Path = target
			kept = append(kept, file)
			continue
		}
		out, err := os.OpenFile(target+".part", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
		if err != nil {
			return contacts.Receipt{}, err
		}
		size, sum1, sum256, fetchErr := a.Fetcher.Fetch(ctx, fileBucket, strings.TrimPrefix(file.Key, strip), out)
		closeErr := out.Close()
		switch {
		case ctx.Err() != nil:
			_ = os.Remove(target + ".part")
			return contacts.Receipt{}, ctx.Err()
		case fetchErr != nil || closeErr != nil:
			_ = os.Remove(target + ".part")
			counts["failed"]++
			notes = append(notes, "fetch failed: "+file.Key)
		case (file.SHA1 != "" && !strings.EqualFold(sum1, file.SHA1)) || (file.SHA1 == "" && file.SHA256 != "" && !strings.EqualFold(sum256, file.SHA256)):
			_ = os.Remove(target + ".part")
			counts["sha1_mismatch"]++
			notes = append(notes, "sha1 differs from the catalog: "+file.Key)
		default:
			if err := os.Rename(target+".part", target); err != nil {
				return contacts.Receipt{}, err
			}
			counts["fetched"]++
			counts["bytes"] += size
			file.Path = target
			kept = append(kept, file)
		}
	}
	if len(kept) == 0 && len(files) > 0 {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts fetch: no contact export could be fetched and verified"))
	}
	listing := filepath.Join(dir, "fetched.jsonl")
	var lines strings.Builder
	for _, file := range kept {
		encoded, _ := json.Marshal(file)
		lines.Write(encoded)
		lines.WriteByte('\n')
	}
	if err := os.WriteFile(listing, []byte(lines.String()), 0o640); err != nil {
		return contacts.Receipt{}, err
	}
	digest, err := fileDigest(listing)
	if err != nil {
		return contacts.Receipt{}, err
	}
	return a.receipt(contacts.FetchActivity, req, listing, digest, counts, notes...), nil
}

// Parse is contacts_parse_activity: parse every fetched export and group contacts into people.
func (a ContactsActivities) Parse(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	dir, err := a.runDir(req)
	if err != nil {
		return contacts.Receipt{}, err
	}
	listing := req.Refs["files"]
	if listing == "" || !strings.HasPrefix(filepath.Clean(listing), filepath.Clean(dir)) {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts parse requires this run's fetched files"))
	}
	files, err := readManifest(listing)
	if err != nil {
		return contacts.Receipt{}, err
	}
	var all []contacts.Contact
	counts := map[string]int64{"files": int64(len(files))}
	var notes []string
	seenSHA := map[string]bool{}
	for index, file := range files {
		a.beat(ctx, fmt.Sprintf("parsing %d of %d", index+1, len(files)))
		if id := file.ContentID(); id != "" && seenSHA[id] {
			counts["duplicates_by_hash"]++
			continue
		} else if id != "" {
			seenSHA[id] = true
		}
		body, err := os.ReadFile(file.Path)
		if err != nil {
			return contacts.Receipt{}, err
		}
		parsed, parseErr := contacts.ParseFile(file.Path, body, file.Key, file.ListedAt)
		if parseErr != nil {
			counts["unreadable_files"]++
			notes = append(notes, fmt.Sprintf("%s: %v", file.Key, parseErr))
		}
		counts["contacts"] += int64(len(parsed))
		all = append(all, parsed...)
	}
	people := contacts.BuildPeople(all)
	numbers := map[string]bool{}
	for _, person := range people {
		for _, number := range person.Numbers {
			numbers[number] = true
		}
		if len(person.CandidateNames) > 1 {
			counts["people_with_several_names"]++
		}
		if len(person.Numbers) == 0 {
			counts["people_with_email_only"]++
		}
	}
	counts["people"] = int64(len(people))
	counts["distinct_numbers"] = int64(len(numbers))
	encoded, err := json.Marshal(people)
	if err != nil {
		return contacts.Receipt{}, err
	}
	path := filepath.Join(dir, "people.json")
	if err := os.WriteFile(path, encoded, 0o640); err != nil {
		return contacts.Receipt{}, err
	}
	digest, err := fileDigest(path)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if len(people) == 0 {
		return contacts.Receipt{Step: contacts.ParseActivity, DryRun: req.DryRun, Status: "not_applicable", Ref: path, Digest: digest, Counts: counts, Notes: notes, At: a.now()}, nil
	}
	return a.receipt(contacts.ParseActivity, req, path, digest, counts, notes...), nil
}

func readPeople(path string) ([]contacts.Person, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var people []contacts.Person
	return people, json.Unmarshal(body, &people)
}

func detailInt(detail map[string]any, key string) int64 {
	switch value := detail[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	}
	return 0
}

func addLinked(counts map[string]int64, detail map[string]any) {
	switch linked := detail["linked"].(type) {
	case map[string]int64:
		for name, count := range linked {
			counts["linked_"+name] += count
		}
	case map[string]any:
		for name, value := range linked {
			counts["linked_"+name] += detailInt(map[string]any{"v": value}, "v")
		}
	}
}

// People is contacts_people_activity: create the unconfirmed people through the governed store, and add the
// contacts' names as candidates to people that already carry one of their numbers.
func (a ContactsActivities) People(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	if _, err := a.runDir(req); err != nil {
		return contacts.Receipt{}, err
	}
	if a.Store == nil || a.Registry == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: the registry is not configured on this worker"))
	}
	people, err := readPeople(req.Refs["people"])
	if err != nil {
		return contacts.Receipt{}, err
	}
	var all []string
	for _, person := range people {
		all = append(all, person.Numbers...)
	}
	carried, err := a.Registry.CarriedEntities(ctx, all)
	if err != nil {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"people": int64(len(people))}
	for index := 0; index < len(people); index += contactPeopleBatch {
		end := min(index+contactPeopleBatch, len(people))
		spec := caseidentity.ContactPeopleSpec{ChangeReason: contactReasonPeople, DryRun: req.DryRun}
		for _, person := range people[index:end] {
			spec.People = append(spec.People, caseidentity.ContactPerson{
				DisplayName: person.DisplayName, Numbers: person.Numbers, Emails: person.Emails,
				CandidateNames: person.CandidateNames, Source: person.Source,
			})
		}
		a.beat(ctx, fmt.Sprintf("people %d-%d of %d", index+1, end, len(people)))
		receipt, err := a.Store.AddContactPeople(ctx, spec, a.actor(req, "people", index/contactPeopleBatch))
		if err != nil {
			return contacts.Receipt{}, contactsStoreError(err)
		}
		counts["created"] += detailInt(receipt.Detail, "created")
		counts["skipped_all_carried"] += detailInt(receipt.Detail, "skipped_all_carried")
		counts["aliases"] += detailInt(receipt.Detail, "aliases")
		addLinked(counts, receipt.Detail)
	}
	// Names for people who already carry a contact's number: candidates, never a rename.
	n := 0
	for _, person := range people {
		entities := map[string]bool{}
		for _, number := range person.Numbers {
			if entity := carried[number]; entity != "" {
				entities[entity] = true
			}
		}
		for entity := range entities {
			for _, name := range person.CandidateNames {
				if req.DryRun {
					counts["names_would_be_added_to_existing_people"]++
					continue
				}
				_, err := a.Store.AddIdentifier(ctx, caseidentity.IdentifierSpec{
					EntityID: entity, RawValue: name, Kind: "name", Status: "candidate",
					Basis: "from contacts: " + person.Source, ChangeReason: "contacts import: candidate name from a contact export (unconfirmed)",
				}, a.actor(req, "alias", n))
				n++
				switch {
				case err == nil:
					counts["names_added_to_existing_people"]++
				case errors.Is(err, caseidentity.ErrStale):
					counts["names_already_present"]++
				default:
					return contacts.Receipt{}, contactsStoreError(err)
				}
			}
		}
	}
	if counts["created"] == 0 && counts["names_added_to_existing_people"] == 0 && !req.DryRun && len(people) > 0 {
		return contacts.Receipt{Step: contacts.PeopleActivity, DryRun: req.DryRun, Status: "not_applicable", Counts: counts,
			Notes: []string{"every contact number was already carried by a person"}, At: a.now()}, nil
	}
	return a.receipt(contacts.PeopleActivity, req, "", "", counts), nil
}

// contactsStoreError makes a refused request permanent (retrying cannot change the answer) and leaves
// infrastructure errors retryable.
func contactsStoreError(err error) error {
	if errors.Is(err, caseidentity.ErrRejected) || errors.Is(err, caseidentity.ErrNotFound) || errors.Is(err, caseidentity.ErrIdempotencyConflict) {
		return contactsPermanent(err)
	}
	return err
}

// Placeholders is contacts_placeholders_activity: a placeholder person only for numbers still carried by no
// person. In a dry run the people the earlier step only simulated are excluded, so the count is honest.
func (a ContactsActivities) Placeholders(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	if _, err := a.runDir(req); err != nil {
		return contacts.Receipt{}, err
	}
	if a.Store == nil || a.Registry == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: the registry is not configured on this worker"))
	}
	numbers, err := a.Registry.UnlinkedNumbers(ctx)
	if err != nil {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"numbers_without_a_person": int64(len(numbers))}
	if req.DryRun && req.Refs["people"] != "" {
		people, err := readPeople(req.Refs["people"])
		if err != nil {
			return contacts.Receipt{}, err
		}
		named := map[string]bool{}
		for _, person := range people {
			for _, number := range person.Numbers {
				named[number] = true
			}
		}
		var rest []string
		for _, number := range numbers {
			if !named[number] {
				rest = append(rest, number)
			}
		}
		counts["named_by_contacts_in_this_run"] = int64(len(numbers) - len(rest))
		numbers = rest
	}
	counts["placeholders_needed"] = int64(len(numbers))
	for index := 0; index < len(numbers); index += contactPlaceholderBatch {
		end := min(index+contactPlaceholderBatch, len(numbers))
		a.beat(ctx, fmt.Sprintf("placeholders %d-%d of %d", index+1, end, len(numbers)))
		receipt, err := a.Store.AddPlaceholders(ctx, caseidentity.PlaceholderSpec{Numbers: numbers[index:end], ChangeReason: contactReasonPlaceholders, DryRun: req.DryRun},
			a.actor(req, "placeholders", index/contactPlaceholderBatch))
		if err != nil {
			return contacts.Receipt{}, contactsStoreError(err)
		}
		counts["created"] += detailInt(receipt.Detail, "created")
		counts["already_carried"] += detailInt(receipt.Detail, "already_carried")
		addLinked(counts, receipt.Detail)
	}
	if len(numbers) == 0 {
		return contacts.Receipt{Step: contacts.PlaceholdersActivity, DryRun: req.DryRun, Status: "not_applicable", Counts: counts,
			Notes: []string{"every number on the imported rows has a person"}, At: a.now()}, nil
	}
	return a.receipt(contacts.PlaceholdersActivity, req, "", "", counts), nil
}

// Relink is contacts_relink_activity: a sweep that fills every NULL entity column whose number a person now
// carries (rows imported while the run was going, or numbers a person gained). Only NULLs; never overwrites.
func (a ContactsActivities) Relink(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	if _, err := a.runDir(req); err != nil {
		return contacts.Receipt{}, err
	}
	if a.Registry == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: the registry is not configured on this worker"))
	}
	a.beat(ctx, "re-linking")
	linked, err := a.Registry.RelinkSweep(ctx, req.DryRun)
	if err != nil {
		return contacts.Receipt{}, err
	}
	return a.receipt(contacts.RelinkActivity, req, "", "", linked), nil
}
