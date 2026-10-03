// Byline: Claude Code · Sonnet · 2026-10-02
//
// Grouping contacts into people, safely. Contacts that share a number or an email are one person, but that
// relation is transitive, so a single widely shared identifier (the owner's own number on every "Me" card, a
// family line on dozens of cards) chains thousands of contacts into one giant cluster. This file fixes that in
// three ways: identifiers a person ALREADY carries are never used as grouping keys (their names are kept
// separately as candidates for that person), a cluster that is still too large to become one person is HELD
// BACK for the owner to review instead of failing the step, and the size of every cluster is reported so the
// real shape of the data is visible.
package contacts

import (
	"sort"
	"strings"
)

// Limits on what one created person may carry. They mirror caseidentity.ValidateContactPeople (a test pins
// that they agree): a cluster beyond them is held back, never sent.
const (
	MaxPersonNumbers = 20
	MaxPersonEmails  = 20
	MaxPersonNames   = 20
)

// Held is a cluster that is too large to be one person. It is listed for the owner and never created.
type Held struct {
	// Reason says which limit it exceeds.
	Reason string `json:"reason"`
	// Contacts is how many cards the cluster chains together.
	Contacts int `json:"contacts"`
	// Counts are the full sizes; the lists below are cut to a readable length.
	NumberCount int      `json:"number_count"`
	EmailCount  int      `json:"email_count"`
	NameCount   int      `json:"name_count"`
	Names       []string `json:"names"`
	Numbers     []string `json:"numbers"`
	Emails      []string `json:"emails"`
	Sources     []string `json:"sources"`
}

// ExistingName is a name a contact gave to an identifier that a person already carries. It becomes a
// candidate alias of that person; it is never a grouping key and never renames anyone.
type ExistingName struct {
	// Identifier is the 10-digit number or lower-case email.
	Identifier string   `json:"identifier"`
	Names      []string `json:"names"`
	Sources    []string `json:"sources"`
}

// Sizes summarises the clusters the contacts formed, before any limit was applied.
type Sizes struct {
	Clusters    int            `json:"clusters"`
	MaxNumbers  int            `json:"max_numbers"`
	MaxEmails   int            `json:"max_emails"`
	MaxNames    int            `json:"max_names"`
	MaxContacts int            `json:"max_contacts"`
	Numbers     map[string]int `json:"numbers_histogram"`
	Names       map[string]int `json:"names_histogram"`
}

// Grouping is the result of Group.
type Grouping struct {
	// People are the clusters small enough to become one unconfirmed person each.
	People []Person `json:"people"`
	// Held are the clusters that were too large; none of them is created.
	Held []Held `json:"held"`
	// Existing are names for identifiers a person already carries, keyed by identifier.
	Existing []ExistingName `json:"existing"`
	Sizes    Sizes          `json:"sizes"`
}

// bucketFor names the histogram bucket of a count.
func bucketFor(n int) string {
	switch {
	case n <= 1:
		return "1"
	case n <= 5:
		return "2-5"
	case n <= 20:
		return "6-20"
	case n <= 100:
		return "21-100"
	}
	return "over 100"
}

func head(values []string, n int) []string {
	if len(values) > n {
		return values[:n]
	}
	return values
}

// Group clusters contacts into people. skip reports identifiers that must not be grouping keys (a number or
// email some person already carries, the owner's own among them); it may be nil. A contact whose every
// identifier is skipped contributes only to Existing. Output order is deterministic.
func Group(contacts []Contact, skip func(identifier string) bool) Grouping {
	if skip == nil {
		skip = func(string) bool { return false }
	}
	parent := map[string]string{}
	var find func(x string) string
	find = func(x string) string {
		if _, ok := parent[x]; !ok {
			parent[x] = x
		}
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	existing := map[string]*ExistingName{}
	remember := func(identifier, name, source string) {
		entry := existing[identifier]
		if entry == nil {
			entry = &ExistingName{Identifier: identifier}
			existing[identifier] = entry
		}
		if !contains(entry.Names, name) {
			entry.Names = append(entry.Names, name)
		}
		if !contains(entry.Sources, source) {
			entry.Sources = append(entry.Sources, source)
		}
	}
	type kept struct {
		contact Contact
		ids     []string
	}
	var cards []kept
	for _, contact := range contacts {
		if contact.Name == "" {
			continue
		}
		var ids []string
		for _, identifier := range append(append([]string{}, contact.Phones...), contact.Emails...) {
			if skip(identifier) {
				remember(identifier, contact.Name, contact.Source)
				continue
			}
			ids = append(ids, identifier)
		}
		if len(ids) == 0 {
			continue
		}
		root := find(ids[0])
		for _, other := range ids[1:] {
			parent[find(other)] = root
		}
		cards = append(cards, kept{contact: contact, ids: ids})
	}
	groups := map[string][]kept{}
	for _, card := range cards {
		root := find(card.ids[0])
		groups[root] = append(groups[root], card)
	}

	newer := func(a, b Contact) bool {
		if a.ListedAt != b.ListedAt {
			return a.ListedAt > b.ListedAt
		}
		return a.Source > b.Source
	}
	result := Grouping{Sizes: Sizes{Numbers: map[string]int{}, Names: map[string]int{}}}
	for _, group := range groups {
		sort.SliceStable(group, func(i, j int) bool { return newer(group[i].contact, group[j].contact) })
		person := Person{DisplayName: group[0].contact.Name, Source: group[0].contact.Source}
		seenName, seenNumber, seenEmail, seenSource := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, card := range group {
			if !seenName[card.contact.Name] {
				seenName[card.contact.Name] = true
				person.CandidateNames = append(person.CandidateNames, card.contact.Name)
			}
			for _, number := range card.contact.Phones {
				if !skip(number) && !seenNumber[number] {
					seenNumber[number] = true
					person.Numbers = append(person.Numbers, number)
				}
			}
			for _, email := range card.contact.Emails {
				if !skip(email) && !seenEmail[email] {
					seenEmail[email] = true
					person.Emails = append(person.Emails, email)
				}
			}
			if !seenSource[card.contact.Source] {
				seenSource[card.contact.Source] = true
				person.Sources = append(person.Sources, card.contact.Source)
			}
		}
		sort.Strings(person.Numbers)
		sort.Strings(person.Emails)
		sort.Strings(person.Sources)

		sizes := &result.Sizes
		sizes.Clusters++
		sizes.Numbers[bucketFor(len(person.Numbers))]++
		sizes.Names[bucketFor(len(person.CandidateNames))]++
		sizes.MaxNumbers = max(sizes.MaxNumbers, len(person.Numbers))
		sizes.MaxEmails = max(sizes.MaxEmails, len(person.Emails))
		sizes.MaxNames = max(sizes.MaxNames, len(person.CandidateNames))
		sizes.MaxContacts = max(sizes.MaxContacts, len(group))

		var reasons []string
		if len(person.Numbers) > MaxPersonNumbers {
			reasons = append(reasons, "more than 20 numbers")
		}
		if len(person.Emails) > MaxPersonEmails {
			reasons = append(reasons, "more than 20 emails")
		}
		if len(person.CandidateNames) > MaxPersonNames {
			reasons = append(reasons, "more than 20 names")
		}
		if len(reasons) > 0 {
			result.Held = append(result.Held, Held{
				Reason: strings.Join(reasons, ", "), Contacts: len(group),
				NumberCount: len(person.Numbers), EmailCount: len(person.Emails), NameCount: len(person.CandidateNames),
				Names: head(person.CandidateNames, 20), Numbers: head(person.Numbers, 30), Emails: head(person.Emails, 10), Sources: head(person.Sources, 10),
			})
			continue
		}
		result.People = append(result.People, person)
	}
	sort.Slice(result.People, func(i, j int) bool {
		a, b := result.People[i], result.People[j]
		if strings.ToLower(a.DisplayName) != strings.ToLower(b.DisplayName) {
			return strings.ToLower(a.DisplayName) < strings.ToLower(b.DisplayName)
		}
		return strings.Join(a.Numbers, ",") < strings.Join(b.Numbers, ",")
	})
	sort.Slice(result.Held, func(i, j int) bool {
		if result.Held[i].Contacts != result.Held[j].Contacts {
			return result.Held[i].Contacts > result.Held[j].Contacts
		}
		return strings.Join(result.Held[i].Numbers, ",") < strings.Join(result.Held[j].Numbers, ",")
	})
	for _, entry := range existing {
		sort.Strings(entry.Sources)
		result.Existing = append(result.Existing, *entry)
	}
	sort.Slice(result.Existing, func(i, j int) bool { return result.Existing[i].Identifier < result.Existing[j].Identifier })
	return result
}

// BuildPeople groups contacts with no identifier skipped and returns only the people that fit the limits. It
// is Group without the owner/existing-person exclusion, kept for callers that have no registry to ask.
func BuildPeople(contacts []Contact) []Person { return Group(contacts, nil).People }

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
