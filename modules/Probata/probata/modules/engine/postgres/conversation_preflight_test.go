package postgres

import "testing"

// TestConversationDeclaredFormatRoutesAIWithoutGenerationLookup preserves store boundaries.
// Inputs: declared formats. Output: exact discriminator assertions; effects: none.
// Pick for human preflight routing; generic JSON never implies AI or normalization.
func TestConversationDeclaredFormatRoutesAIWithoutGenerationLookup(t *testing.T) {
	for _, format := range []string{"chatgpt_official_json", "chatgpt", "claude", "gemini_markdown"} {
		if !aiConversationDeclaredFormat(format) {
			t.Errorf("AI format %s admitted to human lane", format)
		}
	}
	for _, format := range []string{"json", "message_export_json", "sbv_chatgpt_official_json", "sms_backup_restore_xml"} {
		if aiConversationDeclaredFormat(format) {
			t.Errorf("unrecognized format %s guessed as AI", format)
		}
	}
}
