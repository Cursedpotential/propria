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

// Source is one place the same content can be read from: a provider ("b2" or "r2"), a bucket and a key.
type Source struct {
	Provider string `json:"provider"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
}

// File is one contact export in the catalog (raw_duck.bucket_objects_current) and, once fetched, where it is.
// A loose file is read from Provider/Bucket/Key; when that object is gone the same content is tried at each
// of Alternates (the catalog lists the same name and size, or the same SHA-1, in other places). A ZIP member
// (Member set) is read out of the archive at Provider/Bucket/Key with ranged reads.
type File struct {
	// Provider is "b2" or "r2"; the catalog lists both.
	Provider string `json:"provider,omitempty"`
	// Bucket is the bucket the object lives in; the catalog lists several.
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key"`
	// Member is the path of the file inside the ZIP at Key ("" for a loose file).
	Member   string `json:"member,omitempty"`
	Size     int64  `json:"size"`
	SHA1     string `json:"sha1,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	ListedAt string `json:"listed_at"`
	Path     string `json:"path,omitempty"`
	// ArchiveSize is the size of the ZIP at Key for a member (ranged reads need it); 0 for a loose file.
	ArchiveSize int64 `json:"archive_size,omitempty"`
	// Alternates are other places holding the same content, best first.
	Alternates []Source `json:"alternates,omitempty"`
}

// DisplayKey names the file in receipts and in the "from contacts" mark: the key, or key!member for a ZIP member.
func (f File) DisplayKey() string {
	if f.Member != "" {
		return f.Key + "!" + f.Member
	}
	return f.Key
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

var vcardBlock = regexp.MustCompile(`(?is)BEGIN:VCARD.*?END:VCARD`)

// ParseVCard reads every card of a vCard 3.0/4.0 file.
//
// It is tolerant of what phones and exporters really write: a byte-order mark, NULs, bare-CR or bare-LF line
// ends and text before the first BEGIN:VCARD are normalised, and each BEGIN..END block is decoded on its
// own, so one damaged card is skipped instead of losing the whole file ("no BEGIN field" on a
// real Android export). It returns an error only when no card can be read at all.
func ParseVCard(body []byte, source, listedAt string) ([]Contact, error) {
	cleaned := bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	cleaned = bytes.ReplaceAll(cleaned, []byte{0}, nil)
	cleaned = bytes.ReplaceAll(cleaned, []byte{'\r', '\n'}, []byte{'\n'})
	cleaned = bytes.ReplaceAll(cleaned, []byte{'\r'}, []byte{'\n'})
	cleaned = bytes.ReplaceAll(cleaned, []byte{'\n'}, []byte{'\r', '\n'})
	blocks := vcardBlock.FindAll(cleaned, -1)
	if len(blocks) == 0 {
		return nil, errors.New("no vCard found")
	}
	var out []Contact
	var firstErr error
	for _, block := range blocks {
		card, err := vcard.NewDecoder(bytes.NewReader(append(append([]byte{}, block...), '\r', '\n'))).Decode()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		name := cleanName(card.PreferredValue(vcard.FieldFormattedName))
		if name == "" {
			if n := card.Name(); n != nil {
				name = cleanName(n.GivenName + " " + n.FamilyName)
			}
		}
		contact := newContact(name, card.Values(vcard.FieldTelephone), card.Values(vcard.FieldEmail), source, listedAt)
		if contact.Name == "" && len(contact.Phones) == 0 && len(contact.Emails) == 0 {
			continue // a card the decoder accepted but that carries nothing usable
		}
		out = append(out, contact)
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
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
