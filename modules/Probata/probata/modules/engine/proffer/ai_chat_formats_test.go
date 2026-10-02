// Byline: Claude Code · Sonnet · 2026-10-02
package proffer

import "testing"

func TestIsAIChatFormat(t *testing.T) {
	for _, f := range []string{"chatgpt_official_json", "chatgpt_json_array", " ChatGPT_Official_JSON ", "claude_export_json", "gemini_takeout_json", "bard_takeout_json"} {
		if !IsAIChatFormat(f) {
			t.Errorf("%q must be an AI-chat format", f)
		}
	}
	for _, f := range []string{"", "json", "ndjson", "csv", "facebook_messenger_json", "smsbackuprestore_xml", "messages_transcript"} {
		if IsAIChatFormat(f) {
			t.Errorf("%q must not be an AI-chat format", f)
		}
	}
}
