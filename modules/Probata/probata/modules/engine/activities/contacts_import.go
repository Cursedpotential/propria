// Byline: Claude Code · Sonnet · 2026-10-02
//
// The seven Activities of ContactsImportWorkflow (package contacts). Each does ONE thing, takes and returns
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

// ContactsCatalog lists the contact exports in the Case Bible catalog (raw_duck.bucket_objects_current, every
// provider and bucket it covers). Read-only.
type ContactsCatalog interface {
	// ContactFiles returns the loose contact files (vCard, contact CSV/JSON, Facebook/Instagram contact lists),
	// one entry per distinct content: the best place to read it first, the other places holding the same
	// content as Alternates.
	ContactFiles(ctx context.Context) ([]contacts.File, error)
	// ZipFiles returns the archives that may hold contact files inside (Google Takeout, phone exports), one
	// entry per distinct archive, with Alternates.
	ZipFiles(ctx context.Context) ([]contacts.File, error)
}

// ZipMember is one file inside an archive, as its central directory lists it.
type ZipMember struct {
	// Name is the member's path inside the archive.
	Name string
	// Size is the member's uncompressed size in bytes.
	Size int64
}

// ContactsFetcher reads from the object stores. Every method takes the place to read (provider, bucket, key)
// and fails with a short, secret-free reason such as "b2/salem-data: NoSuchKey".
type ContactsFetcher interface {
	// Fetch copies one object into w and returns its size and its SHA-1 and SHA-256 (computed while copying),
	// so the catalog's own hash can be checked whichever it holds.
	Fetch(ctx context.Context, src contacts.Source, w io.Writer) (size int64, sha1Hex, sha256Hex string, err error)
	// ListZipMembers lists the members of the ZIP at src using ranged reads of its directory only.
	ListZipMembers(ctx context.Context, src contacts.Source, size int64) ([]ZipMember, error)
	// FetchZipMember copies one member of the ZIP at src into w with ranged reads, returning its size and SHA-1.
	FetchZipMember(ctx context.Context, src contacts.Source, size int64, member string, w io.Writer) (n int64, sha1Hex string, err error)
}

// ContactsRegistry is the SQL boundary of the parse, placeholder and re-link steps. UnlinkedNumbers are the
// numbers on imported rows with a NULL entity column that NO person carries; CarriedEntities answers who
// carries numbers and emails now; OwnerEntity is the perspective person (role "user"), whose own identifiers
// must never chain contacts together; RelinkSweep fills only NULL entity columns for numbers a person carries
// (dryRun rolls back).
type ContactsRegistry interface {
	UnlinkedNumbers(ctx context.Context) ([]string, error)
	CarriedEntities(ctx context.Context, identifiers []string) (map[string]string, error)
	OwnerEntity(ctx context.Context) (string, error)
	RelinkSweep(ctx context.Context, dryRun bool) (map[string]int64, error)
}

// ContactsStore is the governed case-identity boundary the steps write through.
type ContactsStore interface {
	AddContactPeople(ctx context.Context, spec caseidentity.ContactPeopleSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
	AddPlaceholders(ctx context.Context, spec caseidentity.PlaceholderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
	AddIdentifier(ctx context.Context, spec caseidentity.IdentifierSpec, actor caseidentity.Actor) (caseidentity.Receipt, error)
}

// ContactsActivities implements the seven Activities of ContactsImportWorkflow.
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

// RegisterContactsActivities installs the seven Activities under their exact workflow names.
func RegisterContactsActivities(registrar ActivityRegistrar, a ContactsActivities) {
	for name, fn := range map[string]any{
		contacts.ManifestActivity:     a.Manifest,
		contacts.ZipMembersActivity:   a.ZipMembers,
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

const (
	// contactMaxZips bounds how many archives one run lists the inside of.
	contactMaxZips = 400
	// contactMemberMaxBytes bounds one ZIP member that is read out (a contact list is never this large).
	contactMemberMaxBytes = 64 << 20
	// contactNotesMax bounds the free-text notes a Receipt carries.
	contactNotesMax      = 40
	contactHeldInReceipt = 50
)

var (
	zipMemberContactName = regexp.MustCompile(`(?i)(^|/)[^/]*contacts?[^/]*\.(csv|json)$`)
	zipMemberVCard       = regexp.MustCompile(`(?i)\.vcf$`)
	zipMemberSocial      = regexp.MustCompile(`(?i)(imported_contacts|synced_contacts|contacts_uploaded|your_contacts)[^/]*\.json$`)
	zipMemberNoise       = regexp.MustCompile(`(?i)unknown contact|cube acr|contacts_sync_settings|blocked`)
)

// wantedZipMember reports whether a file inside an archive is a contact list worth reading: a vCard, a CSV or
// JSON named for contacts, or a Facebook/Instagram imported/synced contacts file. Call-recorder notes
// ("Unknown contact ..."), sync settings and block lists are not contact lists.
func wantedZipMember(name string) bool {
	if zipMemberNoise.MatchString(name) {
		return false
	}
	return zipMemberVCard.MatchString(name) || zipMemberContactName.MatchString(name) || zipMemberSocial.MatchString(name)
}

// writeLines writes one JSON document per line to path and returns its SHA-256.
func writeLines[T any](path string, items []T) (string, error) {
	var lines strings.Builder
	for _, item := range items {
		encoded, err := json.Marshal(item)
		if err != nil {
			return "", err
		}
		lines.Write(encoded)
		lines.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(lines.String()), 0o640); err != nil {
		return "", err
	}
	return fileDigest(path)
}

// Manifest is contacts_manifest_activity: list the contact files and the archives that may hold some, from the
// catalog's current listing of every bucket, and write manifest.jsonl and zips.jsonl in the run folder.
// Inputs: the run id. Output: a Receipt whose Ref is manifest.jsonl, with counts of files, bytes, archives
// and files by kind. It reads the catalog only; it fetches nothing.
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
	zips, err := a.Catalog.ZipFiles(ctx)
	if err != nil {
		return contacts.Receipt{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Key < files[j].Key })
	sort.Slice(zips, func(i, j int) bool { return zips[i].Key < zips[j].Key })
	counts := map[string]int64{"files": int64(len(files)), "zip_candidates": int64(len(zips))}
	for _, file := range files {
		counts["bytes"] += file.Size
		counts["kind_"+kindOf(file.Key)]++
		counts["provider_"+file.Provider]++
		if len(file.Alternates) > 0 {
			counts["files_with_fallback_locations"]++
		}
	}
	if _, err := writeLines(filepath.Join(dir, "zips.jsonl"), zips); err != nil {
		return contacts.Receipt{}, err
	}
	path := filepath.Join(dir, "manifest.jsonl")
	digest, err := writeLines(path, files)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if len(files) == 0 && len(zips) == 0 {
		return contacts.Receipt{Step: contacts.ManifestActivity, DryRun: req.DryRun, Status: "not_applicable", Ref: path, Digest: digest, Counts: counts,
			Notes: []string{"the catalog lists no contact exports; check the key patterns"}, At: a.now()}, nil
	}
	return a.receipt(contacts.ManifestActivity, req, path, digest, counts), nil
}

// ZipMembers is contacts_zip_members_activity: list the contact files INSIDE the candidate archives (Google
// Takeout, phone exports) and write members.jsonl. Only each archive's central directory is read, with ranged
// requests, so a 10 GB Takeout part costs a few megabytes; nothing is downloaded whole. An archive that cannot
// be read from its first location is tried at its alternates; one that cannot be read at all is counted and
// named in the notes, and never stops the run.
func (a ContactsActivities) ZipMembers(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	dir, err := a.runDir(req)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if a.Fetcher == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: no object store is configured on this worker"))
	}
	zips, err := readManifest(filepath.Join(dir, "zips.jsonl"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"zip_candidates": int64(len(zips))}
	var notes []string
	if len(zips) > contactMaxZips {
		counts["zips_over_cap"] = int64(len(zips) - contactMaxZips)
		zips = zips[:contactMaxZips]
	}
	var members []contacts.File
	for index, zipFile := range zips {
		a.beat(ctx, fmt.Sprintf("listing archive %d of %d", index+1, len(zips)))
		var listed []ZipMember
		var used contacts.Source
		var reasons []string
		for _, src := range append([]contacts.Source{{Provider: zipFile.Provider, Bucket: zipFile.Bucket, Key: zipFile.Key}}, zipFile.Alternates...) {
			found, listErr := a.Fetcher.ListZipMembers(ctx, src, zipFile.Size)
			if listErr == nil {
				listed, used = found, src
				break
			}
			reasons = append(reasons, listErr.Error())
		}
		if used.Key == "" {
			counts["zips_unreadable"]++
			if len(notes) < contactNotesMax {
				notes = append(notes, fmt.Sprintf("archive not listed: %s: %s", zipFile.Key, strings.Join(reasons, "; ")))
			}
			continue
		}
		counts["zips_listed"]++
		for _, member := range listed {
			if !wantedZipMember(member.Name) {
				continue
			}
			if member.Size > contactMemberMaxBytes {
				counts["members_over_size_cap"]++
				continue
			}
			members = append(members, contacts.File{
				Provider: used.Provider, Bucket: used.Bucket, Key: used.Key, Member: member.Name, Size: member.Size,
				ListedAt: zipFile.ListedAt, ArchiveSize: zipFile.Size, Alternates: zipFile.Alternates,
			})
		}
	}
	counts["members_matched"] = int64(len(members))
	path := filepath.Join(dir, "members.jsonl")
	digest, err := writeLines(path, members)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if len(members) == 0 {
		return contacts.Receipt{Step: contacts.ZipMembersActivity, DryRun: req.DryRun, Status: "not_applicable", Ref: path, Digest: digest, Counts: counts,
			Notes: append(notes, "no contact file was found inside the archives"), At: a.now()}, nil
	}
	return a.receipt(contacts.ZipMembersActivity, req, path, digest, counts, notes...), nil
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

// fetchTo copies one file (or ZIP member) to target, trying its first location and then each alternate until one
// is read and verified. It returns the location that worked and the computed SHA-1 and SHA-256, or every
// location's reason for failing. A hash from the catalog is checked; a member has only the archive's own CRC,
// which the ZIP reader checks while reading.
func (a ContactsActivities) fetchTo(ctx context.Context, file contacts.File, bucket string, target string) (used contacts.Source, size int64, sum1, sum256 string, err error) {
	sources := append([]contacts.Source{{Provider: firstText(file.Provider, "b2"), Bucket: firstText(file.Bucket, bucket), Key: file.Key}}, file.Alternates...)
	var reasons []string
	for _, src := range sources {
		out, openErr := os.OpenFile(target+".part", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
		if openErr != nil {
			return contacts.Source{}, 0, "", "", openErr
		}
		var fetchErr error
		if file.Member != "" {
			size, sum1, fetchErr = a.Fetcher.FetchZipMember(ctx, src, file.ArchiveSize, file.Member, out)
		} else {
			size, sum1, sum256, fetchErr = a.Fetcher.Fetch(ctx, src, out)
		}
		closeErr := out.Close()
		switch {
		case ctx.Err() != nil:
			_ = os.Remove(target + ".part")
			return contacts.Source{}, 0, "", "", ctx.Err()
		case fetchErr != nil:
			reasons = append(reasons, fetchErr.Error())
		case closeErr != nil:
			reasons = append(reasons, closeErr.Error())
		case file.Member == "" && file.SHA1 != "" && !strings.EqualFold(sum1, file.SHA1):
			reasons = append(reasons, fmt.Sprintf("%s/%s: content differs from the catalog's SHA-1", src.Provider, src.Bucket))
		case file.Member == "" && file.SHA1 == "" && file.SHA256 != "" && !strings.EqualFold(sum256, file.SHA256):
			reasons = append(reasons, fmt.Sprintf("%s/%s: content differs from the catalog's SHA-256", src.Provider, src.Bucket))
		default:
			if renameErr := os.Rename(target+".part", target); renameErr != nil {
				return contacts.Source{}, 0, "", "", renameErr
			}
			return src, size, sum1, sum256, nil
		}
		_ = os.Remove(target + ".part")
	}
	return contacts.Source{}, 0, "", "", errors.New(strings.Join(reasons, "; "))
}

// Fetch is contacts_fetch_activity: copy every manifest file and ZIP member into the run folder, checking each
// loose file's hash against the catalog's. When a file's first location is gone, the same content is tried at
// its alternate locations (another bucket, the other provider), and a file is dropped only after every location
// failed; the receipt records each location's actual error text. Nothing unverified is ever parsed.
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
	if members := req.Refs["members"]; members != "" && strings.HasPrefix(filepath.Clean(members), filepath.Clean(dir)) {
		more, err := readManifest(members)
		if err != nil {
			return contacts.Receipt{}, err
		}
		files = append(files, more...)
	}
	bucket := firstText(req.Bucket, "salem-data")
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0o750); err != nil {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"files": int64(len(files))}
	var notes []string
	note := func(text string) {
		if len(notes) < contactNotesMax {
			notes = append(notes, text)
		}
	}
	var kept []contacts.File
	for index, file := range files {
		a.beat(ctx, fmt.Sprintf("fetching %d of %d", index+1, len(files)))
		id := file.ContentID()
		if file.Member != "" || !hexID.MatchString(id) { // a catalog value is never trusted as a file name
			id = fmt.Sprintf("%x", sha256.Sum256([]byte(file.Bucket+"/"+file.DisplayKey())))
		}
		extension := strings.ToLower(filepath.Ext(firstText(file.Member, file.Key)))
		target := filepath.Join(filesDir, id+extension)
		if info, statErr := os.Stat(target); statErr == nil && (file.Size == 0 || info.Size() == file.Size) {
			counts["reused"]++
			file.Path = target
			kept = append(kept, file)
			continue
		}
		used, size, sum1, sum256, fetchErr := a.fetchTo(ctx, file, bucket, target)
		if fetchErr != nil {
			if ctx.Err() != nil {
				return contacts.Receipt{}, ctx.Err()
			}
			counts["failed"]++
			note(fmt.Sprintf("fetch failed: %s: %s", file.DisplayKey(), fetchErr.Error()))
			continue
		}
		counts["fetched"]++
		counts["bytes"] += size
		if used.Key != file.Key || used.Provider != firstText(file.Provider, "b2") || used.Bucket != firstText(file.Bucket, bucket) {
			counts["fetched_from_alternate"]++
			note(fmt.Sprintf("read from an alternate location: %s <- %s/%s/%s", file.DisplayKey(), used.Provider, used.Bucket, used.Key))
		}
		if file.Member != "" {
			counts["zip_members_fetched"]++
		}
		file.Path = target
		if file.SHA1 == "" {
			file.SHA1 = sum1 // the content's own SHA-1, so duplicate members and r2-only copies are recognised in parse
		}
		if file.SHA256 == "" {
			file.SHA256 = sum256
		}
		kept = append(kept, file)
	}
	if len(kept) == 0 && len(files) > 0 {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts fetch: no contact export could be fetched and verified"))
	}
	listing := filepath.Join(dir, "fetched.jsonl")
	digest, err := writeLines(listing, kept)
	if err != nil {
		return contacts.Receipt{}, err
	}
	return a.receipt(contacts.FetchActivity, req, listing, digest, counts, notes...), nil
}

// ownerAndCarried resolves who already carries each identifier and which of them is the owner (perspective
// person), so grouping can treat both as fixed points instead of chaining contacts through them.
func (a ContactsActivities) ownerAndCarried(ctx context.Context, all []contacts.Contact) (owner string, carried map[string]string, err error) {
	if a.Registry == nil {
		return "", map[string]string{}, nil
	}
	if owner, err = a.Registry.OwnerEntity(ctx); err != nil {
		return "", nil, err
	}
	seen := map[string]bool{}
	var ids []string
	for _, c := range all {
		for _, id := range append(append([]string{}, c.Phones...), c.Emails...) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	carried = map[string]string{}
	for i := 0; i < len(ids); i += 5000 {
		part, err := a.Registry.CarriedEntities(ctx, ids[i:min(i+5000, len(ids))])
		if err != nil {
			return "", nil, err
		}
		for id, entity := range part {
			carried[id] = entity
		}
	}
	return owner, carried, nil
}

// ExistingNames is one identifier a person already carries with the names contacts gave it. Parse writes it
// to existing.json and People adds the names as candidates; the owner's own entries are dropped.
type ExistingNames struct {
	Identifier string   `json:"identifier"`
	Entity     string   `json:"entity"`
	Names      []string `json:"names"`
	Sources    []string `json:"sources"`
}

// Parse is contacts_parse_activity: parse every fetched file and group the contacts into people.
//
// Identifiers a person already carries (the owner's own numbers and emails first among them) are never used to
// chain contacts together; the names contacts gave them are kept as candidates for that person instead. A
// cluster still too large to be one person is HELD BACK: it is listed in the receipt (names, numbers, source
// files) and in held.json for the owner to review, and never created and never fails the step. Cluster sizes
// are reported in the counts. Outputs in the run folder: people.json (clusters that fit), held.json,
// existing.json (names for people who already exist).
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
	seenHash := map[string]bool{}
	for index, file := range files {
		a.beat(ctx, fmt.Sprintf("parsing %d of %d", index+1, len(files)))
		if id := file.ContentID(); id != "" && seenHash[id] {
			counts["duplicates_by_hash"]++
			continue
		} else if id != "" {
			seenHash[id] = true
		}
		body, err := os.ReadFile(file.Path)
		if err != nil {
			return contacts.Receipt{}, err
		}
		parsed, parseErr := contacts.ParseFile(firstText(file.Member, file.Key), body, file.DisplayKey(), file.ListedAt)
		if parseErr != nil {
			counts["unreadable_files"]++
			if len(notes) < contactNotesMax {
				notes = append(notes, fmt.Sprintf("%s: %v", file.DisplayKey(), parseErr))
			}
		}
		counts["contacts"] += int64(len(parsed))
		all = append(all, parsed...)
	}

	owner, carried, err := a.ownerAndCarried(ctx, all)
	if err != nil {
		return contacts.Receipt{}, err
	}
	grouping := contacts.Group(all, func(identifier string) bool { _, ok := carried[identifier]; return ok })
	numbers := map[string]bool{}
	for _, person := range grouping.People {
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
	var existing []ExistingNames
	for _, entry := range grouping.Existing {
		entity := carried[entry.Identifier]
		if entity == "" || entity == owner {
			counts["names_on_owner_identifiers_ignored"]++
			continue
		}
		existing = append(existing, ExistingNames{Identifier: entry.Identifier, Entity: entity, Names: head(entry.Names, 20), Sources: head(entry.Sources, 5)})
	}
	counts["people"] = int64(len(grouping.People))
	counts["distinct_numbers"] = int64(len(numbers))
	counts["identifiers_already_carried"] = int64(len(carried))
	counts["people_already_carrying_a_contact_identifier"] = int64(len(existing))
	counts["clusters"] = int64(grouping.Sizes.Clusters)
	counts["held_clusters"] = int64(len(grouping.Held))
	counts["cluster_max_numbers"] = int64(grouping.Sizes.MaxNumbers)
	counts["cluster_max_emails"] = int64(grouping.Sizes.MaxEmails)
	counts["cluster_max_names"] = int64(grouping.Sizes.MaxNames)
	counts["cluster_max_contacts"] = int64(grouping.Sizes.MaxContacts)
	for bucket, n := range grouping.Sizes.Numbers {
		counts["clusters_by_numbers_"+bucket] = int64(n)
	}
	for bucket, n := range grouping.Sizes.Names {
		counts["clusters_by_names_"+bucket] = int64(n)
	}
	for name, value := range map[string]any{"held.json": grouping.Held, "existing.json": existing} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return contacts.Receipt{}, err
		}
		if err := os.WriteFile(filepath.Join(dir, name), encoded, 0o640); err != nil {
			return contacts.Receipt{}, err
		}
	}
	encoded, err := json.Marshal(grouping.People)
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
	receipt := a.receipt(contacts.ParseActivity, req, path, digest, counts, notes...)
	receipt.Held = head(grouping.Held, contactHeldInReceipt)
	if len(grouping.People) == 0 {
		receipt.Status = "not_applicable"
	}
	return receipt, nil
}

func head[T any](values []T, n int) []T {
	if len(values) > n {
		return values[:n]
	}
	return values
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
// contacts' names as candidates to people that already carry a contact's identifier. A cluster over the limits
// of one person is skipped here too (and counted), so a stale people.json can never fail the step.
func (a ContactsActivities) People(ctx context.Context, req contacts.StepRequest) (contacts.Receipt, error) {
	dir, err := a.runDir(req)
	if err != nil {
		return contacts.Receipt{}, err
	}
	if a.Store == nil || a.Registry == nil {
		return contacts.Receipt{}, contactsPermanent(errors.New("contacts import: the registry is not configured on this worker"))
	}
	all, err := readPeople(req.Refs["people"])
	if err != nil {
		return contacts.Receipt{}, err
	}
	counts := map[string]int64{"people": int64(len(all))}
	var people []contacts.Person
	for _, person := range all {
		if len(person.Numbers) > contacts.MaxPersonNumbers || len(person.Emails) > contacts.MaxPersonEmails || len(person.CandidateNames) > contacts.MaxPersonNames {
			counts["held_back_at_write"]++
			continue
		}
		people = append(people, person)
	}
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
	// Names for people who already carry a contact's identifier: candidates, never a rename.
	var existing []ExistingNames
	if body, readErr := os.ReadFile(filepath.Join(dir, "existing.json")); readErr == nil {
		if err := json.Unmarshal(body, &existing); err != nil {
			return contacts.Receipt{}, err
		}
	}
	n := 0
	for _, entry := range existing {
		for _, name := range entry.Names {
			if req.DryRun {
				counts["names_would_be_added_to_existing_people"]++
				continue
			}
			_, err := a.Store.AddIdentifier(ctx, caseidentity.IdentifierSpec{
				EntityID: entry.Entity, RawValue: name, Kind: "name", Status: "candidate",
				Basis: "from contacts: " + firstText(strings.Join(head(entry.Sources, 2), "; "), "contact export"), ChangeReason: "contacts import: candidate name from a contact export (unconfirmed)",
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
	if counts["created"] == 0 && counts["names_added_to_existing_people"] == 0 && !req.DryRun && len(people) > 0 {
		return contacts.Receipt{Step: contacts.PeopleActivity, DryRun: req.DryRun, Status: "not_applicable", Counts: counts,
			Notes: []string{"every contact identifier was already carried by a person"}, At: a.now()}, nil
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
