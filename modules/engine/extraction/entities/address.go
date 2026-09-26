// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package entities proposes people, places and organizations from a run's
// normalized messages, groups their aliases, and applies the owner's
// corrections. Everything here is pure: no I/O, no clock, no randomness.
// Activities (modules/engine/activities/entity_extraction.go) read and write;
// this package only decides.
//
// Extraction is not analysis (AGENTS.md): a proposal says "these surfaces
// appear to name one entity", never what the entity did or meant.
package entities

import (
	"strings"
	"unicode"
)

// AddressKind classifies a participant identifier.
type AddressKind string

const (
	AddressPhone  AddressKind = "phone"
	AddressEmail  AddressKind = "email"
	AddressHandle AddressKind = "handle"
	// AddressSelf is the SMS/chat export's name for the device owner. It is
	// relative to the source: "self" in her phone's backup is her, in his
	// backup it is him. It is therefore scoped to one source version and is
	// never written as a global registry alias.
	AddressSelf  AddressKind = "self"
	AddressOther AddressKind = "other"
)

// Address is one normalized participant identifier.
type Address struct {
	Kind       AddressKind `json:"kind"`
	Raw        string      `json:"raw"`
	Normalized string      `json:"normalized"`
}

// DefaultCountryCode is prepended to ten-digit national numbers. The corpus
// is North American (NANP); a number carrying its own "+" keeps its code.
const DefaultCountryCode = "1"

// NormalizeAddress turns one participant identifier into its comparison form:
// E.164 for phone numbers, lower case for e-mail addresses and handles.
func NormalizeAddress(raw string) Address {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	switch {
	case trimmed == "":
		return Address{Kind: AddressOther, Raw: raw}
	case lower == "self" || lower == "me":
		return Address{Kind: AddressSelf, Raw: raw, Normalized: "self"}
	case isEmail(lower):
		return Address{Kind: AddressEmail, Raw: raw, Normalized: lower}
	case strings.HasPrefix(lower, "@") && len(lower) > 1 && !strings.ContainsAny(lower, " \t"):
		return Address{Kind: AddressHandle, Raw: raw, Normalized: lower}
	}
	if phone, ok := normalizePhone(trimmed); ok {
		return Address{Kind: AddressPhone, Raw: raw, Normalized: phone}
	}
	return Address{Kind: AddressOther, Raw: raw, Normalized: strings.Join(strings.Fields(lower), " ")}
}

func isEmail(value string) bool {
	at := strings.IndexByte(value, '@')
	if at <= 0 || at != strings.LastIndexByte(value, '@') || strings.ContainsAny(value, " \t") {
		return false
	}
	domain := value[at+1:]
	dot := strings.LastIndexByte(domain, '.')
	return dot > 0 && dot < len(domain)-1
}

// normalizePhone accepts digits with the usual separators only. Seven to
// fifteen digits is the E.164 envelope; shorter numbers are service short
// codes and stay "other".
func normalizePhone(value string) (string, bool) {
	var digits strings.Builder
	plus := false
	for i, r := range value {
		switch {
		case unicode.IsDigit(r):
			if r > unicode.MaxASCII {
				return "", false
			}
			digits.WriteRune(r)
		case r == '+' && i == 0:
			plus = true
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return "", false
		}
	}
	number := digits.String()
	if len(number) < 7 || len(number) > 15 {
		return "", false
	}
	switch {
	case plus:
		return "+" + number, true
	case len(number) == 10:
		return "+" + DefaultCountryCode + number, true
	case len(number) == 11 && strings.HasPrefix(number, DefaultCountryCode):
		return "+" + number, true
	default:
		// A local number without its country or area code cannot be made
		// E.164 honestly; it stays bare digits so it never collides with a
		// full number that merely shares its last seven digits.
		return number, true
	}
}

// DisplayAddress renders an address for people, e.g. "+1 (810) 555-0101".
func DisplayAddress(address Address) string {
	if address.Kind == AddressSelf {
		return "Device owner (this source)"
	}
	if address.Kind != AddressPhone {
		if address.Normalized != "" {
			return address.Normalized
		}
		return strings.TrimSpace(address.Raw)
	}
	number := address.Normalized
	if strings.HasPrefix(number, "+"+DefaultCountryCode) && len(number) == 12 {
		national := number[2:]
		return "+" + DefaultCountryCode + " (" + national[:3] + ") " + national[3:6] + "-" + national[6:]
	}
	return number
}

// NationalDigits returns the digits a person would type for a phone address
// inside a message body ("8105550101" for "+18105550101"). Empty for
// non-phone addresses.
func NationalDigits(address Address) string {
	if address.Kind != AddressPhone {
		return ""
	}
	number := strings.TrimPrefix(address.Normalized, "+")
	if strings.HasPrefix(number, DefaultCountryCode) && len(number) == 11 {
		return number[1:]
	}
	return number
}
