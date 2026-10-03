// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Cross-language vectors for ConversationKey. server/context_chunks/generation.py ports the key so the conversation
// chunks, which are cut BEFORE the commit from the normalized records, group messages exactly as the first-party
// context import does. tests/test_context_chunks.py asserts the same vectors against the Python port; change one,
// change both.
package smsthreads

import (
	"fmt"
	"reflect"
	"testing"
)

func TestConversationKeyVectorsSharedWithThePythonPort(t *testing.T) {
	long := []string{}
	longParticipants := []string{}
	for i := 0; i < 9; i++ {
		long = append(long, fmt.Sprintf("+1810555%04d", i))
		longParticipants = append(longParticipants, fmt.Sprintf("810555%04d", i))
	}
	cases := []struct {
		in           []string
		key          string
		participants []string
	}{
		{[]string{"+18102959303", "self"}, "8102959303", []string{"8102959303"}},
		{[]string{"(810) 295-9303", "+18102959302", "SELF"}, "8102959302_8102959303", []string{"8102959302", "8102959303"}},
		{[]string{"Katrina.Kinzel@Example.com", "self"}, "katrina.kinzel@example.com", []string{"katrina.kinzel@example.com"}},
		{[]string{"Katrina Kinzel", "self"}, "katrinakinzel", []string{"katrinakinzel"}},
		{[]string{"self", "null", ""}, "unknown", []string{}},
		{[]string{"12345", "insert-address-token"}, "12345", []string{"12345"}},
		{long, "group-9-785bb6e5b107", longParticipants},
	}
	for _, c := range cases {
		key, participants := ConversationKey(c.in)
		if participants == nil {
			participants = []string{}
		}
		if key != c.key || !reflect.DeepEqual(participants, c.participants) {
			t.Errorf("ConversationKey(%q) = %q %q, want %q %q", c.in, key, participants, c.key, c.participants)
		}
	}
}
