// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contacts"
)

type contactsFakeCatalog struct{ files, zips []contacts.File }

func (f contactsFakeCatalog) ContactFiles(context.Context) ([]contacts.File, error) {
	return f.files, nil
}
func (f contactsFakeCatalog) ZipFiles(context.Context) ([]contacts.File, error) { return f.zips, nil }

// contactsFakeFetcher serves objects by "provider/bucket/key"; anything else is NoSuchKey, like a catalog row whose
// object has moved. Archives list their members from members and serve member bodies by "key!member".
type contactsFakeFetcher struct {
	bodies  map[string]string
	members map[string][]ZipMember
	reads   []string
}

func (f *contactsFakeFetcher) Fetch(_ context.Context, src contacts.Source, w io.Writer) (int64, string, string, error) {
	where := src.Provider + "/" + src.Bucket + "/" + src.Key
	f.reads = append(f.reads, where)
	body, ok := f.bodies[where]
	if !ok {
		return 0, "", "", fmt.Errorf("%s/%s: NoSuchKey", src.Provider, src.Bucket)
	}
	_, _ = io.WriteString(w, body)
	return int64(len(body)), "sha1-of-" + src.Key, "", nil
}
func (f *contactsFakeFetcher) ListZipMembers(_ context.Context, src contacts.Source, _ int64) ([]ZipMember, error) {
	if members, ok := f.members[src.Key]; ok {
		return members, nil
	}
	return nil, fmt.Errorf("%s/%s: NoSuchKey", src.Provider, src.Bucket)
}
func (f *contactsFakeFetcher) FetchZipMember(_ context.Context, src contacts.Source, _ int64, member string, w io.Writer) (int64, string, error) {
	body, ok := f.bodies[src.Key+"!"+member]
	if !ok {
		return 0, "", fmt.Errorf("%s/%s: member %q is not in the archive", src.Provider, src.Bucket, member)
	}
	_, _ = io.WriteString(w, body)
	return int64(len(body)), "member-sha1-" + member, nil
}

type contactsFakeRegistry struct {
	unlinked []string
	carried  map[string]string
	owner    string
	relinked bool
	dry      bool
}

func (r *contactsFakeRegistry) UnlinkedNumbers(context.Context) ([]string, error) {
	return r.unlinked, nil
}
func (r *contactsFakeRegistry) CarriedEntities(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if entity, ok := r.carried[id]; ok {
			out[id] = entity
		}
	}
	return out, nil
}
func (r *contactsFakeRegistry) OwnerEntity(context.Context) (string, error) { return r.owner, nil }
func (r *contactsFakeRegistry) RelinkSweep(_ context.Context, dry bool) (map[string]int64, error) {
	r.relinked, r.dry = true, dry
	return map[string]int64{"linked_call_from": 3}, nil
}

type contactsFakeStore struct {
	people       []caseidentity.ContactPeopleSpec
	placeholders []caseidentity.PlaceholderSpec
	identifiers  []caseidentity.IdentifierSpec
	actors       []caseidentity.Actor
}

func (s *contactsFakeStore) AddContactPeople(_ context.Context, spec caseidentity.ContactPeopleSpec, a caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateContactPeople(spec); err != nil {
		return caseidentity.Receipt{}, err // the real store refuses an oversized person; the step must never send one
	}
	s.people, s.actors = append(s.people, spec), append(s.actors, a)
	return caseidentity.Receipt{Detail: map[string]any{"created": len(spec.People), "aliases": 2 * len(spec.People), "linked": map[string]int64{"call_from": 4}}}, nil
}
func (s *contactsFakeStore) AddPlaceholders(_ context.Context, spec caseidentity.PlaceholderSpec, a caseidentity.Actor) (caseidentity.Receipt, error) {
	s.placeholders, s.actors = append(s.placeholders, spec), append(s.actors, a)
	return caseidentity.Receipt{Detail: map[string]any{"created": len(spec.Numbers)}}, nil
}
func (s *contactsFakeStore) AddIdentifier(_ context.Context, spec caseidentity.IdentifierSpec, a caseidentity.Actor) (caseidentity.Receipt, error) {
	s.identifiers, s.actors = append(s.identifiers, spec), append(s.actors, a)
	return caseidentity.Receipt{}, nil
}

const vcardJordan = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Jordan Reyes\r\nTEL:810-555-0142\r\nTEL:313-555-0199\r\nEND:VCARD\r\n"

type chain struct {
	a        ContactsActivities
	store    *contactsFakeStore
	registry *contactsFakeRegistry
	fetcher  *contactsFakeFetcher
	receipts map[string]contacts.Receipt
}

func newChain(t *testing.T, catalog contactsFakeCatalog, fetcher *contactsFakeFetcher, registry *contactsFakeRegistry) *chain {
	t.Helper()
	store := &contactsFakeStore{}
	return &chain{
		a:     ContactsActivities{Catalog: catalog, Fetcher: fetcher, Registry: registry, Store: store, WorkRoot: t.TempDir()},
		store: store, registry: registry, fetcher: fetcher, receipts: map[string]contacts.Receipt{},
	}
}

func (c *chain) run(t *testing.T, dry bool) {
	t.Helper()
	base := contacts.StepRequest{RunID: "run-1", DryRun: dry, Actor: contacts.Actor{SubjectUID: "uid", Username: "matt"}, Refs: map[string]string{}}
	step := func(name string, fn func(context.Context, contacts.StepRequest) (contacts.Receipt, error), carry string) {
		receipt, err := fn(context.Background(), base)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		c.receipts[name] = receipt
		if carry != "" {
			base.Refs[carry] = receipt.Ref
		}
	}
	step(contacts.ManifestActivity, c.a.Manifest, "manifest")
	step(contacts.ZipMembersActivity, c.a.ZipMembers, "members")
	step(contacts.FetchActivity, c.a.Fetch, "files")
	step(contacts.ParseActivity, c.a.Parse, "people")
	step(contacts.PeopleActivity, c.a.People, "")
	step(contacts.PlaceholdersActivity, c.a.Placeholders, "")
	step(contacts.RelinkActivity, c.a.Relink, "")
}

func defaultChain(t *testing.T) *chain {
	return newChain(t,
		contactsFakeCatalog{files: []contacts.File{{Provider: "b2", Bucket: "salem-data", Key: "contacts/a.vcf", Size: 120, SHA1: "sha1-of-contacts/a.vcf", ListedAt: "2026-09-01T00:00:00Z"}}},
		&contactsFakeFetcher{bodies: map[string]string{"b2/salem-data/contacts/a.vcf": vcardJordan}},
		&contactsFakeRegistry{
			unlinked: []string{"8105550142", "4195550999"}, // 8105550142 is named by a contact, 4195550999 by nobody
			carried:  map[string]string{"3135550199": "entity-existing"}, owner: "entity-owner",
		})
}

func TestChainNamesPeopleFirstThenPlaceholdersOnlyForTheRest(t *testing.T) {
	c := defaultChain(t)
	c.run(t, false)
	if c.receipts[contacts.ManifestActivity].Counts["files"] != 1 || c.receipts[contacts.FetchActivity].Counts["fetched"] != 1 {
		t.Fatalf("manifest/fetch = %+v / %+v", c.receipts[contacts.ManifestActivity], c.receipts[contacts.FetchActivity])
	}
	parse := c.receipts[contacts.ParseActivity]
	// 313-555-0199 is already a person's, so it is not a grouping key: the one new person carries only 810-555-0142.
	if parse.Counts["contacts"] != 1 || parse.Counts["people"] != 1 || parse.Counts["distinct_numbers"] != 1 || parse.Digest == "" || parse.Counts["identifiers_already_carried"] != 1 {
		t.Fatalf("parse = %+v", parse)
	}
	if len(c.store.people) != 1 || c.store.people[0].DryRun || c.store.people[0].People[0].DisplayName != "Jordan Reyes" ||
		len(c.store.people[0].People[0].Numbers) != 1 || c.store.people[0].People[0].Source != "contacts/a.vcf" {
		t.Fatalf("people spec = %+v", c.store.people)
	}
	if len(c.store.identifiers) != 1 || c.store.identifiers[0].EntityID != "entity-existing" || c.store.identifiers[0].Status != "candidate" ||
		!strings.HasPrefix(c.store.identifiers[0].Basis, "from contacts: ") {
		t.Fatalf("a person that already carries a contact number only gets the name as a candidate: %+v", c.store.identifiers)
	}
	if len(c.store.placeholders) != 1 || c.store.placeholders[0].DryRun || len(c.store.placeholders[0].Numbers) != 2 {
		t.Fatalf("placeholders = %+v (the engine skips carried numbers)", c.store.placeholders)
	}
	if !c.registry.relinked || c.registry.dry || c.receipts[contacts.RelinkActivity].Counts["linked_call_from"] != 3 {
		t.Fatalf("re-link = %+v %+v", c.registry, c.receipts[contacts.RelinkActivity])
	}
	seen := map[string]bool{}
	for _, actor := range c.store.actors {
		if actor.Username != "matt" || actor.SubjectUID != "uid" || actor.IdempotencyKey == "" || seen[actor.IdempotencyKey] {
			t.Fatalf("every governed call needs the owner as actor and its own idempotency key: %+v", c.store.actors)
		}
		seen[actor.IdempotencyKey] = true
	}
}

func TestDryRunRollsBackAndDoesNotCountNumbersTheContactsWouldName(t *testing.T) {
	c := defaultChain(t)
	c.run(t, true)
	if !c.store.people[0].DryRun || !c.store.placeholders[0].DryRun || !c.registry.dry {
		t.Fatalf("every write step must be a dry run: %+v %+v %v", c.store.people, c.store.placeholders, c.registry.dry)
	}
	if len(c.store.identifiers) != 0 || c.receipts[contacts.PeopleActivity].Counts["names_would_be_added_to_existing_people"] != 1 {
		t.Fatalf("a dry run must not write candidate names: %+v", c.store.identifiers)
	}
	placeholders := c.receipts[contacts.PlaceholdersActivity]
	if placeholders.Counts["named_by_contacts_in_this_run"] != 1 || placeholders.Counts["placeholders_needed"] != 1 ||
		len(c.store.placeholders[0].Numbers) != 1 || c.store.placeholders[0].Numbers[0] != "4195550999" {
		t.Fatalf("placeholders = %+v %+v", placeholders, c.store.placeholders)
	}
}

func TestTheOwnersOwnNumberNeverChainsContactsIntoOneCluster(t *testing.T) {
	const owner = "8102959302"
	var cards strings.Builder
	for i := 0; i < 30; i++ { // thirty unrelated people whose cards all also list the owner's own number ("Me")
		fmt.Fprintf(&cards, "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Person %02d\r\nTEL:810-555-%04d\r\nTEL:%s\r\nEND:VCARD\r\n", i, 1000+i, owner)
	}
	c := newChain(t,
		contactsFakeCatalog{files: []contacts.File{{Provider: "b2", Bucket: "b", Key: "all.vcf", SHA1: "sha1-of-all.vcf"}}},
		&contactsFakeFetcher{bodies: map[string]string{"b2/b/all.vcf": cards.String()}},
		&contactsFakeRegistry{carried: map[string]string{owner: "entity-owner"}, owner: "entity-owner"})
	c.run(t, false)
	parse := c.receipts[contacts.ParseActivity]
	if parse.Counts["people"] != 30 || parse.Counts["held_clusters"] != 0 || parse.Counts["cluster_max_numbers"] != 1 {
		t.Fatalf("the owner's number must not chain people: %+v", parse.Counts)
	}
	if len(c.store.identifiers) != 0 || parse.Counts["names_on_owner_identifiers_ignored"] != 1 {
		t.Fatalf("names on the owner's own number must not become candidates of the owner: %+v %+v", c.store.identifiers, parse.Counts)
	}
}

func TestAnOversizedClusterIsHeldBackListedAndNeverFailsTheStep(t *testing.T) {
	var cards strings.Builder
	for i := 0; i < 25; i++ { // a shared line chains 25 cards, each with its own number, into one cluster of 26 numbers
		fmt.Fprintf(&cards, "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Family %02d\r\nTEL:810-555-%04d\r\nTEL:248-555-0000\r\nEND:VCARD\r\n", i, 2000+i)
	}
	cards.WriteString(vcardJordan)
	c := newChain(t,
		contactsFakeCatalog{files: []contacts.File{{Provider: "b2", Bucket: "b", Key: "big.vcf", SHA1: "sha1-of-big.vcf", Alternates: nil}}},
		&contactsFakeFetcher{bodies: map[string]string{"b2/b/big.vcf": cards.String()}},
		&contactsFakeRegistry{owner: "entity-owner", carried: map[string]string{}})
	c.run(t, false)
	parse := c.receipts[contacts.ParseActivity]
	if parse.Counts["held_clusters"] != 1 || len(parse.Held) != 1 || parse.Held[0].NumberCount != 26 || parse.Held[0].Contacts != 25 ||
		len(parse.Held[0].Names) != 20 || len(parse.Held[0].Sources) == 0 || !strings.Contains(parse.Held[0].Reason, "numbers") {
		t.Fatalf("held = %+v counts = %+v", parse.Held, parse.Counts)
	}
	if parse.Counts["people"] != 1 || len(c.store.people) != 1 || len(c.store.people[0].People) != 1 || c.store.people[0].People[0].DisplayName != "Jordan Reyes" {
		t.Fatalf("only the cluster that fits is created: %+v", c.store.people)
	}
	if parse.Counts["cluster_max_numbers"] != 26 || parse.Counts["clusters_by_numbers_21-100"] != 1 {
		t.Fatalf("cluster sizes must be reported: %+v", parse.Counts)
	}
}

func TestFetchFallsBackToAnotherLocationAndRecordsTheRealErrorText(t *testing.T) {
	good := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Lee Park\r\nTEL:419-555-0100\r\nEND:VCARD\r\n"
	files := []contacts.File{
		{Provider: "b2", Bucket: "salem-data", Key: "vault/Contacts (2019 UTC).vcf", SHA1: "sha1-of-vault/Contacts (2019 UTC).vcf",
			Alternates: []contacts.Source{{Provider: "r2", Bucket: "casebible-raw", Key: "_backup_import/CSV/Contacts (2019 UTC).vcf"}}},
		{Provider: "b2", Bucket: "salem-data", Key: "vault/gone.vcf", SHA1: "sha1-of-vault/gone.vcf",
			Alternates: []contacts.Source{{Provider: "r2", Bucket: "casebible-raw", Key: "also-gone.vcf"}}},
	}
	fetcher := &contactsFakeFetcher{bodies: map[string]string{"r2/casebible-raw/_backup_import/CSV/Contacts (2019 UTC).vcf": good}}
	// the alternate's bytes hash differently from the key-derived fake, so give the catalog no hash to compare for it
	files[0].SHA1 = ""
	c := newChain(t, contactsFakeCatalog{files: files}, fetcher, &contactsFakeRegistry{owner: "o", carried: map[string]string{}})
	req := contacts.StepRequest{RunID: "run-2", Actor: contacts.Actor{SubjectUID: "u", Username: "m"}, Refs: map[string]string{}}
	manifest, err := c.a.Manifest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.Refs["manifest"] = manifest.Ref
	receipt, err := c.a.Fetch(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Counts["fetched"] != 1 || receipt.Counts["fetched_from_alternate"] != 1 || receipt.Counts["failed"] != 1 {
		t.Fatalf("counts = %+v", receipt.Counts)
	}
	joined := strings.Join(receipt.Notes, "\n")
	if !strings.Contains(joined, "vault/gone.vcf: b2/salem-data: NoSuchKey; r2/casebible-raw: NoSuchKey") || !strings.Contains(joined, "read from an alternate location") {
		t.Fatalf("the receipt must name each location's actual error: %q", joined)
	}
}

func TestContactFilesInsideZipArchivesAreListedFetchedAndParsed(t *testing.T) {
	zipKey := "vault/Takeout/takeout-001.zip"
	member := "Takeout/Contacts/All Contacts/All Contacts.vcf"
	fetcher := &contactsFakeFetcher{
		members: map[string][]ZipMember{zipKey: {
			{Name: member, Size: 90}, {Name: "Takeout/Photos/IMG_1.jpg", Size: 5_000_000},
			{Name: "Takeout/Contacts/Unknown contact 2025.json", Size: 120}, {Name: "Takeout/Drive/notes.txt", Size: 10},
		}},
		bodies: map[string]string{zipKey + "!" + member: vcardJordan},
	}
	c := newChain(t,
		contactsFakeCatalog{zips: []contacts.File{{Provider: "b2", Bucket: "salem-data", Key: zipKey, Size: 10 << 30, ListedAt: "2026-09-01T00:00:00Z"}}},
		fetcher, &contactsFakeRegistry{owner: "o", carried: map[string]string{}, unlinked: nil})
	c.run(t, false)
	zip := c.receipts[contacts.ZipMembersActivity]
	if zip.Counts["zips_listed"] != 1 || zip.Counts["members_matched"] != 1 {
		t.Fatalf("zip = %+v", zip)
	}
	if c.receipts[contacts.FetchActivity].Counts["zip_members_fetched"] != 1 {
		t.Fatalf("fetch = %+v", c.receipts[contacts.FetchActivity])
	}
	parse := c.receipts[contacts.ParseActivity]
	if parse.Counts["contacts"] != 1 || parse.Counts["people"] != 1 {
		t.Fatalf("parse = %+v", parse)
	}
	if got := c.store.people[0].People[0].Source; got != zipKey+"!"+member {
		t.Fatalf("a member's mark names the archive and the member: %q", got)
	}
}

func TestAnUnreadableArchiveIsNamedNotFatal(t *testing.T) {
	c := newChain(t,
		contactsFakeCatalog{zips: []contacts.File{{Provider: "b2", Bucket: "salem-data", Key: "vault/missing.zip", Size: 100}}},
		&contactsFakeFetcher{}, &contactsFakeRegistry{owner: "o", carried: map[string]string{}})
	req := contacts.StepRequest{RunID: "run-3", Actor: contacts.Actor{SubjectUID: "u", Username: "m"}, Refs: map[string]string{}}
	if _, err := c.a.Manifest(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	receipt, err := c.a.ZipMembers(context.Background(), req)
	if err != nil || receipt.Counts["zips_unreadable"] != 1 || !strings.Contains(strings.Join(receipt.Notes, " "), "vault/missing.zip: b2/salem-data: NoSuchKey") {
		t.Fatalf("receipt = %+v err = %v", receipt, err)
	}
}

func TestFetchRefusesAFileWhoseHashDiffersFromTheCatalog(t *testing.T) {
	a := ContactsActivities{
		Catalog: contactsFakeCatalog{files: []contacts.File{{Provider: "b2", Bucket: "b", Key: "x.vcf", Size: 3, SHA1: "not-the-hash"}}},
		Fetcher: &contactsFakeFetcher{bodies: map[string]string{"b2/b/x.vcf": "abc"}}, WorkRoot: t.TempDir(),
	}
	req := contacts.StepRequest{RunID: "run-4", Actor: contacts.Actor{SubjectUID: "u", Username: "m"}, Refs: map[string]string{}}
	manifest, err := a.Manifest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.Refs["manifest"] = manifest.Ref
	if _, err := a.Fetch(context.Background(), req); err == nil {
		t.Fatal("a fetch where nothing verifies must fail rather than parse unverified bytes")
	}
	if _, err := os.Stat(manifest.Ref); err != nil {
		t.Fatal(err)
	}
}

func TestPersonLimitsMatchTheGovernedStore(t *testing.T) {
	person := caseidentity.ContactPerson{DisplayName: "A", Source: "s"}
	for i := 0; i <= contacts.MaxPersonNumbers; i++ {
		person.Numbers = append(person.Numbers, fmt.Sprintf("81055500%02d", i))
	}
	spec := caseidentity.ContactPeopleSpec{ChangeReason: "r", People: []caseidentity.ContactPerson{person}}
	if caseidentity.ValidateContactPeople(spec) == nil {
		t.Fatal("the store must refuse more numbers than contacts.MaxPersonNumbers, so the step holds such a cluster back first")
	}
	person.Numbers = person.Numbers[:contacts.MaxPersonNumbers]
	if err := caseidentity.ValidateContactPeople(caseidentity.ContactPeopleSpec{ChangeReason: "r", People: []caseidentity.ContactPerson{person}}); err != nil {
		t.Fatalf("exactly the limit must be accepted: %v", err)
	}
}
