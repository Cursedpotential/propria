// Byline: Claude Code · Sonnet · 2026-10-02
package proffer

import "strings"

// AIChatSearchOnlyType is the Temporal application-error type of an AI-chat refusal.
const AIChatSearchOnlyType = "ai_chat_search_only"

// aiChatFormats are the source formats that are AI assistant exports. Owner
// order 2026-10-02: AI chats are search-only, in Weaviate (AiChatEvents20260918)
// through the Case Bible. They are never normalized into working.* and never
// become evidence. Only chatgpt_official_json / chatgpt_json_array are detected
// today; the other ids are reserved so a future detector cannot slip past.
var aiChatFormats = map[string]struct{}{
	"chatgpt_official_json": {}, "chatgpt_json_array": {},
	"claude_export_json": {}, "gemini_takeout_json": {}, "bard_takeout_json": {},
}

// IsAIChatFormat reports whether format names an AI-chat export format.
func IsAIChatFormat(format string) bool {
	_, ok := aiChatFormats[strings.ToLower(strings.TrimSpace(format))]
	return ok
}

// AIChatRefusalMessage is the operator-facing refusal text.
func AIChatRefusalMessage(format string) string {
	return "AI chats are search-only and are not imported by Proffer (format " + format +
		"): they are searched from Weaviate (AiChatEvents20260918), loaded through the Case Bible, and never become evidence"
}
