// Byline: Claude Code · Opus 5.5 · 2026-10-02
package activities

import (
	"reflect"
	"testing"

	"github.com/Cursedpotential/probata/engine/contextsearch"
)

func TestContextSearchRecordKindRoutesOnlyApprovedKinds(t *testing.T) {
	cases := []struct {
		recordType string
		aiChat     bool
		want       string
		wantErr    bool
	}{
		{"message", false, contextsearch.RecordKindMessage, false},
		{"message", true, contextsearch.RecordKindAIChat, false},
		{"call", false, contextsearch.RecordKindCall, false},
		{"call", true, "", true},
		{"document", false, contextsearch.RecordKindDocument, false},
		{"event", false, "", true},
		{"media", false, "", true},
		{"other", false, "", true},
	}
	for _, c := range cases {
		got, err := contextSearchRecordKind(c.recordType, c.aiChat)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%s (ai=%v): got %q, %v", c.recordType, c.aiChat, got, err)
		}
	}
}

func TestStatedParticipantsCountsTheDeviceOwnerOnCalls(t *testing.T) {
	sender, recipients := statedParticipants(ContextSearchRecord{RecordType: "message", Participants: []ContextSearchParticipant{
		{Role: "sender", Identifier: "self"}, {Role: "recipient", Identifier: "+18105550199"}, {Role: "unknown", Identifier: "self"},
	}})
	if sender != "self" || !reflect.DeepEqual(recipients, []string{"+18105550199", "self"}) {
		t.Errorf("message: sender %q recipients %v", sender, recipients)
	}
	sender, recipients = statedParticipants(ContextSearchRecord{RecordType: "call", Participants: []ContextSearchParticipant{
		{Role: "unknown", Identifier: "+18105550199"},
	}})
	if sender != "" || !reflect.DeepEqual(recipients, []string{"+18105550199", "self"}) {
		t.Errorf("call: sender %q recipients %v", sender, recipients)
	}
}
