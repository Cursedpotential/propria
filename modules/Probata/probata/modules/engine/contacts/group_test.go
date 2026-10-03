// Byline: Claude Code · Sonnet · 2026-10-02
package contacts

import (
	"fmt"
	"testing"
)

// TestTolerantVCardSurvivesBOMBareCRJunkAndOneBadCard covers the file that used to fail with "no BEGIN field":
// a byte-order mark, bare carriage returns, junk before the first card and one unreadable card.
func TestTolerantVCardSurvivesBOMBareCRJunkAndOneBadCard(t *testing.T) {
	raw := "\xef\xbb\xbfjunk line\rBEGIN:VCARD\rVERSION:3.0\rFN:Lee Park\rTEL:419-555-0100\rEND:VCARD\r" +
		"BEGIN:VCARD\rthis is not a card\rEND:VCARD\r" +
		"BEGIN:VCARD\nVERSION:3.0\nFN:Sam Ng\nTEL:810-555-0142\nEND:VCARD\n"
	cards, err := ParseVCard([]byte(raw), "k", "")
	if err != nil || len(cards) != 2 || cards[0].Name != "Lee Park" || cards[1].Name != "Sam Ng" {
		t.Fatalf("cards = %+v err = %v", cards, err)
	}
	if _, err := ParseVCard([]byte("no cards here"), "k", ""); err == nil {
		t.Fatal("a file with no readable card must still be reported")
	}
}

// TestGroupSkipsCarriedIdentifiersAndHoldsBackOversizedClusters checks the guarantees of Group: skipped
// identifiers never chain contacts, an over-limit cluster is held (not returned as a person), and sizes are reported.
func TestGroupSkipsCarriedIdentifiersAndHoldsBackOversizedClusters(t *testing.T) {
	var cards []Contact
	for i := 0; i < 5; i++ {
		cards = append(cards, Contact{Name: fmt.Sprintf("P%d", i), Phones: []string{fmt.Sprintf("81055501%02d", i), "8102959302"}, Source: "s.vcf"})
	}
	if chained := Group(cards, nil); len(chained.People) != 1 {
		t.Fatalf("without a skip the shared number chains them: %+v", chained.People)
	}
	skipped := Group(cards, func(id string) bool { return id == "8102959302" })
	if len(skipped.People) != 5 || len(skipped.Held) != 0 || len(skipped.Existing) != 1 || len(skipped.Existing[0].Names) != 5 {
		t.Fatalf("skipped = %+v", skipped)
	}
	var many []Contact
	for i := 0; i < MaxPersonNumbers+1; i++ {
		many = append(many, Contact{Name: "Q", Phones: []string{fmt.Sprintf("41955500%02d", i), "2485550000"}, Source: "big.vcf"})
	}
	held := Group(many, nil)
	if len(held.People) != 0 || len(held.Held) != 1 || held.Held[0].NumberCount != MaxPersonNumbers+2 || len(held.Held[0].Sources) == 0 {
		t.Fatalf("held = %+v", held)
	}
}
