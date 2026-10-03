// Byline: Claude Code · Sonnet · 2026-10-02
package activities

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/contacts"
)

type contactsFakeCatalog struct{ files []contacts.File }

func (f contactsFakeCatalog) ContactFiles(context.Context) ([]contacts.File, error) {
	return f.files, nil
}

type contactsFakeFetcher struct{ bodies map[string]string }

func (f contactsFakeFetcher) Fetch(_ context.Context, _, key string, w io.Writer) (int64, string, string, error) {
	body := f.bodies[key]
	_, _ = io.WriteString(w, body)
	return int64(len(body)), "sha1-of-" + key, "", nil
}

type contactsFakeRegistry struct {
	unlinked []string
	carried  map[string]string
	relinked bool
	dry      bool
}

func (r *contactsFakeRegistry) UnlinkedNumbers(context.Context) ([]string, error) {
	return r.unlinked, nil
}
func (r *contactsFakeRegistry) CarriedEntities(context.Context, []string) (map[string]string, error) {
	return r.carried, nil
}
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

func runChain(t *testing.T, dry bool) (*contactsFakeStore, *contactsFakeRegistry, map[string]contacts.Receipt) {
	t.Helper()
	store := &contactsFakeStore{}
	registry := &contactsFakeRegistry{
		unlinked: []string{"8105550142", "4195550999"}, // 8105550142 is named by a contact, 4195550999 by nobody
		carried:  map[string]string{"3135550199": "entity-existing"},
	}
	a := ContactsActivities{
		Catalog:  contactsFakeCatalog{files: []contacts.File{{Bucket: "salem-data", Key: "contacts/a.vcf", Size: 120, SHA1: "sha1-of-contacts/a.vcf", ListedAt: "2026-09-01T00:00:00Z"}}},
		Fetcher:  contactsFakeFetcher{bodies: map[string]string{"contacts/a.vcf": "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Jordan Reyes\r\nTEL:810-555-0142\r\nTEL:313-555-0199\r\nEND:VCARD\r\n"}},
		Registry: registry, Store: store, WorkRoot: t.TempDir(),
	}
	base := contacts.StepRequest{RunID: "run-1", DryRun: dry, Actor: contacts.Actor{SubjectUID: "uid", Username: "matt"}, Refs: map[string]string{}}
	receipts := map[string]contacts.Receipt{}
	step := func(name string, fn func(context.Context, contacts.StepRequest) (contacts.Receipt, error), carry string) {
		receipt, err := fn(context.Background(), base)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		receipts[name] = receipt
		if carry != "" {
			base.Refs[carry] = receipt.Ref
		}
	}
	step(contacts.ManifestActivity, a.Manifest, "manifest")
	step(contacts.FetchActivity, a.Fetch, "files")
	step(contacts.ParseActivity, a.Parse, "people")
	step(contacts.PeopleActivity, a.People, "")
	step(contacts.PlaceholdersActivity, a.Placeholders, "")
	step(contacts.RelinkActivity, a.Relink, "")
	return store, registry, receipts
}

func TestChainNamesPeopleFirstThenPlaceholdersOnlyForTheRest(t *testing.T) {
	store, registry, receipts := runChain(t, false)
	if receipts[contacts.ManifestActivity].Counts["files"] != 1 || receipts[contacts.FetchActivity].Counts["fetched"] != 1 {
		t.Fatalf("manifest/fetch = %+v / %+v", receipts[contacts.ManifestActivity], receipts[contacts.FetchActivity])
	}
	parse := receipts[contacts.ParseActivity]
	if parse.Counts["contacts"] != 1 || parse.Counts["people"] != 1 || parse.Counts["distinct_numbers"] != 2 || parse.Digest == "" {
		t.Fatalf("parse = %+v", parse)
	}
	if len(store.people) != 1 || store.people[0].DryRun || store.people[0].People[0].DisplayName != "Jordan Reyes" || store.people[0].People[0].Source != "contacts/a.vcf" {
		t.Fatalf("people spec = %+v", store.people)
	}
	if len(store.identifiers) != 1 || store.identifiers[0].EntityID != "entity-existing" || store.identifiers[0].Status != "candidate" ||
		!strings.HasPrefix(store.identifiers[0].Basis, "from contacts: ") {
		t.Fatalf("a person that already carries a contact number only gets the name as a candidate: %+v", store.identifiers)
	}
	if len(store.placeholders) != 1 || store.placeholders[0].DryRun || len(store.placeholders[0].Numbers) != 2 {
		t.Fatalf("placeholders = %+v (the engine skips carried numbers)", store.placeholders)
	}
	if !registry.relinked || registry.dry || receipts[contacts.RelinkActivity].Counts["linked_call_from"] != 3 {
		t.Fatalf("re-link = %+v %+v", registry, receipts[contacts.RelinkActivity])
	}
	seen := map[string]bool{}
	for _, actor := range store.actors {
		if actor.Username != "matt" || actor.SubjectUID != "uid" || actor.IdempotencyKey == "" || seen[actor.IdempotencyKey] {
			t.Fatalf("every governed call needs the owner as actor and its own idempotency key: %+v", store.actors)
		}
		seen[actor.IdempotencyKey] = true
	}
}

func TestDryRunRollsBackAndDoesNotCountNumbersTheContactsWouldName(t *testing.T) {
	store, registry, receipts := runChain(t, true)
	if !store.people[0].DryRun || !store.placeholders[0].DryRun || !registry.dry {
		t.Fatalf("every write step must be a dry run: %+v %+v %v", store.people, store.placeholders, registry.dry)
	}
	if len(store.identifiers) != 0 || receipts[contacts.PeopleActivity].Counts["names_would_be_added_to_existing_people"] != 1 {
		t.Fatalf("a dry run must not write candidate names: %+v", store.identifiers)
	}
	placeholders := receipts[contacts.PlaceholdersActivity]
	if placeholders.Counts["named_by_contacts_in_this_run"] != 1 || placeholders.Counts["placeholders_needed"] != 1 ||
		len(store.placeholders[0].Numbers) != 1 || store.placeholders[0].Numbers[0] != "4195550999" {
		t.Fatalf("placeholders = %+v %+v", placeholders, store.placeholders)
	}
}

func TestFetchRefusesAFileWhoseHashDiffersFromTheCatalog(t *testing.T) {
	a := ContactsActivities{
		Catalog: contactsFakeCatalog{files: []contacts.File{{Bucket: "b", Key: "x.vcf", Size: 3, SHA1: "not-the-hash"}}},
		Fetcher: contactsFakeFetcher{bodies: map[string]string{"x.vcf": "abc"}}, WorkRoot: t.TempDir(),
	}
	req := contacts.StepRequest{RunID: "run-2", Actor: contacts.Actor{SubjectUID: "u", Username: "m"}, Refs: map[string]string{}}
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
