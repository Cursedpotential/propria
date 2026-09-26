// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"strings"
	"unicode"
)

// RegistryNormalizedName is exactly the rule the governed create path uses
// for registry.entity.normalized_name (server/api/entity_routes.py:
// " ".join(name.casefold().split())), so a committed proposal and a
// hand-created entity collide on uq_entity_norm instead of duplicating.
func RegistryNormalizedName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// NameKey is the comparison form used for grouping: letters and digits only,
// lower case, single spaces. "Jane  O'Neil" and "jane oneil" share it.
func NameKey(name string) string {
	var builder strings.Builder
	space := false
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			space = false
			builder.WriteRune(r)
		case r == '\'' || r == '’' || r == '.':
			// Apostrophes and initials' dots join, they do not separate.
		default:
			space = true
		}
	}
	return builder.String()
}

// Tokens splits a name key into words.
func Tokens(name string) []string {
	return strings.Fields(NameKey(name))
}

// SpellingKey erases only the differences that are spelling, not sound or
// name: "ph" is "f", a hard "c" or "q" is "k", "ck" is "k", a silent "h"
// after a consonant or at the end is dropped, doubled letters collapse, and
// a final "y", "ie", "ey" or "i" is one ending. So "Karina"/"Carina",
// "Katherine"/"Catherine", "Philip"/"Filip", "John"/"Jon", "Sarah"/"Sara",
// "Katie"/"Katy" and "Marc"/"Mark" share a key, while "Maria"/"Mario",
// "Jon"/"Jan" and "Katherine"/"Kathryn" do not — vowels are never erased.
func SpellingKey(word string) string {
	runes := []rune(strings.ReplaceAll(NameKey(word), " ", ""))
	isVowel := func(r rune) bool { return strings.ContainsRune("aeiou", r) }
	var mapped []rune
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		switch {
		case r == 'p' && next == 'h':
			mapped = append(mapped, 'f')
			i++
		case r == 'c' && next == 'k':
			mapped = append(mapped, 'k')
			i++
		case r == 'c' && (next == 'e' || next == 'i' || next == 'y'):
			mapped = append(mapped, 's')
		case r == 'c' || r == 'q':
			mapped = append(mapped, 'k')
		case r == 'h' && i > 0 && !isVowel(runes[i-1]):
			// silent h after a consonant: "Katherine", "Christina"
		case r == 'h' && i > 0 && isVowel(runes[i-1]) && next != 0 && !isVowel(next):
			// silent h between a vowel and a consonant: "John"
		case r == 'h' && i == len(runes)-1 && i > 0:
			// silent final h: "Sarah"
		case r == 'y' && i > 0:
			mapped = append(mapped, 'i')
		default:
			mapped = append(mapped, r)
		}
	}
	var collapsed []rune
	for _, r := range mapped {
		if len(collapsed) > 0 && collapsed[len(collapsed)-1] == r {
			continue
		}
		collapsed = append(collapsed, r)
	}
	key := string(collapsed)
	for _, ending := range []string{"ie", "ei", "ey"} {
		if strings.HasSuffix(key, ending) && len(key) > len(ending) {
			key = strings.TrimSuffix(key, ending) + "i"
			break
		}
	}
	return key
}

// NameRelation says how two names relate, strongest first.
type NameRelation string

const (
	NameUnrelated NameRelation = ""
	// NameSame: identical comparison keys ("Jane" / "jane").
	NameSame NameRelation = "same"
	// NameSpelling: same sound, one or two letters apart ("Karina" /
	// "Carina"). Proposed as alias_kind "misspelling".
	NameSpelling NameRelation = "spelling"
	// NameContained: every word of the shorter name is in the longer one
	// ("Jane" / "Jane Doe"). Proposed as alias_kind "other".
	NameContained NameRelation = "contained"
)

// RelateNames compares two names. It is deliberately conservative: names
// that merely sound alike ("Karina" / "Kathryn", "Maria" / "Mario") are NOT
// related here; the owner merges those by hand.
func RelateNames(a, b string) NameRelation {
	left, right := NameKey(a), NameKey(b)
	if left == "" || right == "" {
		return NameUnrelated
	}
	if left == right {
		return NameSame
	}
	leftTokens, rightTokens := strings.Fields(left), strings.Fields(right)
	if len(leftTokens) == len(rightTokens) {
		spelling := true
		for i := range leftTokens {
			if leftTokens[i] == rightTokens[i] {
				continue
			}
			if !spellingVariant(leftTokens[i], rightTokens[i]) {
				spelling = false
				break
			}
		}
		if spelling {
			return NameSpelling
		}
	}
	shorter, longer := leftTokens, rightTokens
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if len(shorter) < len(longer) && containsAllTokens(longer, shorter) {
		return NameContained
	}
	return NameUnrelated
}

func spellingVariant(a, b string) bool {
	left, right := SpellingKey(a), SpellingKey(b)
	return left != "" && left == right
}

func containsAllTokens(haystack, needles []string) bool {
	for _, needle := range needles {
		found := false
		for _, token := range haystack {
			if token == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// LooksLikeAddress reports whether a "name" is really an identifier, so a
// contact card that stores the number as the display name does not become a
// name alias.
func LooksLikeAddress(name string) bool {
	address := NormalizeAddress(name)
	return address.Kind == AddressPhone || address.Kind == AddressEmail || address.Kind == AddressHandle || address.Kind == AddressSelf
}
