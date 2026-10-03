// Byline: Claude Code · Sonnet · 2026-10-02
package contacts

import (
	"reflect"
	"testing"
)

const vcf = "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Jordan Reyes\r\nTEL;TYPE=CELL:(810) 555-0142\r\nTEL:+1 313 555 0199\r\nEMAIL:Jordan@Example.com\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nVERSION:3.0\r\nN:Ng;Sam;;;\r\nTEL:810.555.0142\r\nEND:VCARD\r\n"

func TestVCardNamesPhonesAndEmails(t *testing.T) {
	cards, err := ParseVCard([]byte(vcf), "k1", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 || cards[0].Name != "Jordan Reyes" || cards[1].Name != "Sam Ng" {
		t.Fatalf("cards = %+v", cards)
	}
	if !reflect.DeepEqual(cards[0].Phones, []string{"3135550199", "8105550142"}) || !reflect.DeepEqual(cards[0].Emails, []string{"jordan@example.com"}) {
		t.Fatalf("first card = %+v", cards[0])
	}
}

func TestCSVGoogleAndOutlookHeaders(t *testing.T) {
	google, err := ParseCSV([]byte("First Name,Last Name,Phone 1 - Value,E-mail 1 - Value\nJordan,Reyes,(810) 555-0142,j@example.com\n"), "k", "")
	if err != nil || len(google) != 1 || google[0].Name != "Jordan Reyes" || google[0].Phones[0] != "8105550142" || google[0].Emails[0] != "j@example.com" {
		t.Fatalf("google = %+v %v", google, err)
	}
	outlook, err := ParseCSV([]byte("First Name,Last Name,Mobile Phone,E-mail Address\nSam,Ng,810-555-0142,\n"), "k", "")
	if err != nil || len(outlook) != 1 || outlook[0].Name != "Sam Ng" || outlook[0].Phones[0] != "8105550142" {
		t.Fatalf("outlook = %+v %v", outlook, err)
	}
}

func TestSocialJSONAtAnyDepth(t *testing.T) {
	cards, err := ParseSocialJSON([]byte(`{"phone_contacts":[{"first_name":"Jordan","last_name":"Reyes","contact_point":"+18105550142"}],"x":{"y":[{"name":"Lee","email":"lee@example.com"}]}}`), "fb", "")
	if err != nil || len(cards) != 2 {
		t.Fatalf("cards = %+v %v", cards, err)
	}
	if cards[0].Name != "Jordan Reyes" || cards[0].Phones[0] != "8105550142" || cards[1].Name != "Lee" || len(cards[1].Phones) != 0 {
		t.Fatalf("cards = %+v", cards)
	}
}

func TestContactsSharingANumberAreOnePersonNamedByTheMostRecentExport(t *testing.T) {
	old, _ := ParseVCard([]byte(vcf), "b2://x/old.vcf", "2026-01-01T00:00:00Z")
	newer, _ := ParseCSV([]byte("Name,Phone\nJ. Reyes,810-555-0142\nSolo Person,419-555-0100\n"), "b2://x/new.csv", "2026-09-01T00:00:00Z")
	people := BuildPeople(append(old, newer...))
	if len(people) != 2 {
		t.Fatalf("people = %+v", people)
	}
	var merged, solo Person
	for _, person := range people {
		if person.DisplayName == "J. Reyes" {
			merged = person
		} else {
			solo = person
		}
	}
	if !reflect.DeepEqual(merged.Numbers, []string{"3135550199", "8105550142"}) || !reflect.DeepEqual(merged.CandidateNames, []string{"J. Reyes", "Jordan Reyes", "Sam Ng"}) ||
		merged.Source != "b2://x/new.csv" || !reflect.DeepEqual(merged.Emails, []string{"jordan@example.com"}) {
		t.Fatalf("merged = %+v", merged)
	}
	if solo.DisplayName != "Solo Person" || !reflect.DeepEqual(solo.Numbers, []string{"4195550100"}) {
		t.Fatalf("solo = %+v", solo)
	}
}

func TestAnEmailOnlyContactIsStillAPerson(t *testing.T) {
	cards, _ := ParseCSV([]byte("Name,E-mail\nEmail Only,e@example.com\n"), "k", "")
	people := BuildPeople(cards)
	if len(people) != 1 || len(people[0].Numbers) != 0 || people[0].Emails[0] != "e@example.com" {
		t.Fatalf("people = %+v", people)
	}
}
