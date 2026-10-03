// Byline: Claude Code · Sonnet · 2026-10-02
//
// Parsing contact exports and grouping contacts into people. Pure functions: no I/O beyond the reader
// they are given, no registry, no clock. They parse and group; they do nothing more (the engine's
// atomicity rule), so contacts_parse_activity is a thin shell around them.
//
// Formats: vCard (.vcf, off-the-shelf github.com/emersion/go-vcard), CSV (Google, Outlook or generic
// headers) and Facebook / Instagram contact lists (JSON, any nesting).
package contacts

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emersion/go-vcard"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// File is one contact export in the catalog (raw_duck.b2_objects) and, once fetched, where it is.
type File struct {
	// Bucket is the B2 bucket the object lives in; the catalog lists several.
	Bucket   string `json:"bucket,omitempty"`
	Key      string `json:"key"`
	Size     int64  `json:"size"`
	SHA1     string `json:"sha1,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	ListedAt string `json:"listed_at"`
	Path     string `json:"path,omitempty"`
}

// ContentID is the catalog's content hash for the object (sha1, else sha256); duplicates across buckets and
// keys share it. Empty when the catalog holds no hash.
func (f File) ContentID() string {
	if f.SHA1 != "" {
		return strings.ToLower(f.SHA1)
	}
	return strings.ToLower(f.SHA256)
}

// Contact is one card from one export.
type Contact struct {
	Name     string   `json:"name"`
	Phones   []string `json:"phones"`
	Emails   []string `json:"emails"`
	Source   string   `json:"source"`
	ListedAt string   `json:"listed_at"`
}

// Person is one grouped person: contacts that share a number or an email are one person.
type Person struct {
	DisplayName    string   `json:"display_name"`
	Numbers        []string `json:"numbers"`
	Emails         []string `json:"emails"`
	CandidateNames []string `json:"candidate_names"`
	// Source is the key of the most recent export that names the person.
	Source  string   `json:"source"`
	Sources []string `json:"sources"`
}

var whitespace = regexp.MustCompile(`\s+`)

func cleanName(value string) string {
	return strings.TrimSpace(whitespace.ReplaceAllString(value, " "))
}

func newContact(name string, phones, emails []string, source, listedAt string) Contact {
	seenPhone, seenEmail := map[string]bool{}, map[string]bool{}
	contact := Contact{Name: cleanName(name), Source: source, ListedAt: listedAt}
	for _, raw := range phones {
		if number, ok := caseidentity.NormalizePhone(raw); ok && !seenPhone[number] {
			seenPhone[number] = true
			contact.Phones = append(contact.Phones, number)
		}
	}
	for _, raw := range emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if strings.Contains(email, "@") && !seenEmail[email] {
			seenEmail[email] = true
			contact.Emails = append(contact.Emails, email)
		}
	}
	sort.Strings(contact.Phones)
	sort.Strings(contact.Emails)
	return contact
}

// ParseFile parses one export by its extension and returns its contacts. An unknown extension yields none.
func ParseFile(name string, body []byte, source, listedAt string) ([]Contact, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".vcf", ".vcard":
		return ParseVCard(body, source, listedAt)
	case ".csv":
		return ParseCSV(body, source, listedAt)
	case ".json":
		return ParseSocialJSON(body, source, listedAt)
	}
	return nil, nil
}

// ParseVCard reads every card of a vCard 3.0/4.0 file.
func ParseVCard(body []byte, source, listedAt string) ([]Contact, error) {
	decoder := vcard.NewDecoder(bytes.NewReader(body))
	var out []Contact
	for {
		card, err := decoder.Decode()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		name := cleanName(card.PreferredValue(vcard.FieldFormattedName))
		if name == "" {
			if n := card.Name(); n != nil {
				name = cleanName(n.GivenName + " " + n.FamilyName)
			}
		}
		out = append(out, newContact(name, card.Values(vcard.FieldTelephone), card.Values(vcard.FieldEmail), source, listedAt))
	}
}

// ParseCSV reads Google, Outlook or generic contact CSVs.
func ParseCSV(body []byte, source, listedAt string) ([]Contact, error) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for i := range header {
		header[i] = strings.ToLower(strings.TrimSpace(header[i]))
	}
	phoneColumn := regexp.MustCompile(`phone|mobile|cell|tel`)
	mailColumn := regexp.MustCompile(`e-?mail`)
	var out []Contact
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		cell := map[string]string{}
		for i, value := range row {
			if i < len(header) {
				cell[header[i]] = strings.TrimSpace(value)
			}
		}
		name := firstNonEmpty(cell["name"], cell["display name"], cell["full name"])
		if name == "" {
			name = strings.Join(nonEmpty(firstNonEmpty(cell["first name"], cell["given name"]), cell["middle name"],
				firstNonEmpty(cell["last name"], cell["family name"])), " ")
		}
		var phones, emails []string
		for column, value := range cell {
			if value == "" || strings.Contains(column, "type") || strings.Contains(column, "label") {
				continue
			}
			switch {
			case phoneColumn.MatchString(column):
				for _, part := range regexp.MustCompile(`\s*:::\s*|;`).Split(value, -1) {
					phones = append(phones, part)
				}
			case mailColumn.MatchString(column):
				emails = append(emails, value)
			}
		}
		out = append(out, newContact(name, phones, emails, source, listedAt))
	}
}

// ParseSocialJSON reads Facebook and Instagram contact lists: any object that carries a name and a
// phone number or email, at any depth.
func ParseSocialJSON(body []byte, source, listedAt string) ([]Contact, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	var out []Contact
	var walk func(node any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			name := cleanName(str(value["name"]))
			if name == "" {
				name = cleanName(str(value["first_name"]) + " " + str(value["last_name"]))
			}
			point := firstNonEmpty(str(value["contact_point"]), str(value["phone"]), str(value["phone_number"]), str(value["email"]))
			if name != "" && point != "" {
				out = append(out, newContact(name, []string{point}, []string{point}, source, listedAt))
			}
			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(root)
	return out, nil
}

func str(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

// BuildPeople groups contacts that share a number or an email into one person. DisplayName is the name
// from the most recent export (ListedAt, then source key); CandidateNames keeps every name any export
// gave, newest first, so disagreeing exports never lose a name and nothing is a silent pick.
func BuildPeople(contacts []Contact) []Person {
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
	var named []Contact
	for _, contact := range contacts {
		identifiers := append(append([]string{}, contact.Phones...), contact.Emails...)
		if contact.Name == "" || len(identifiers) == 0 {
			continue
		}
		named = append(named, contact)
		root := find(identifiers[0])
		for _, other := range identifiers[1:] {
			parent[find(other)] = root
		}
	}
	groups := map[string][]Contact{}
	for _, contact := range named {
		identifiers := append(append([]string{}, contact.Phones...), contact.Emails...)
		groups[find(identifiers[0])] = append(groups[find(identifiers[0])], contact)
	}
	newer := func(a, b Contact) bool {
		if a.ListedAt != b.ListedAt {
			return a.ListedAt > b.ListedAt
		}
		return a.Source > b.Source
	}
	var people []Person
	for _, cards := range groups {
		sort.SliceStable(cards, func(i, j int) bool { return newer(cards[i], cards[j]) })
		person := Person{DisplayName: cards[0].Name, Source: cards[0].Source}
		seenName, seenNumber, seenEmail, seenSource := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, card := range cards {
			if !seenName[card.Name] {
				seenName[card.Name] = true
				person.CandidateNames = append(person.CandidateNames, card.Name)
			}
			for _, number := range card.Phones {
				if !seenNumber[number] {
					seenNumber[number] = true
					person.Numbers = append(person.Numbers, number)
				}
			}
			for _, email := range card.Emails {
				if !seenEmail[email] {
					seenEmail[email] = true
					person.Emails = append(person.Emails, email)
				}
			}
			if !seenSource[card.Source] {
				seenSource[card.Source] = true
				person.Sources = append(person.Sources, card.Source)
			}
		}
		sort.Strings(person.Numbers)
		sort.Strings(person.Emails)
		sort.Strings(person.Sources)
		people = append(people, person)
	}
	sort.Slice(people, func(i, j int) bool {
		if strings.ToLower(people[i].DisplayName) != strings.ToLower(people[j].DisplayName) {
			return strings.ToLower(people[i].DisplayName) < strings.ToLower(people[j].DisplayName)
		}
		return strings.Join(people[i].Numbers, ",") < strings.Join(people[j].Numbers, ",")
	})
	return people
}
