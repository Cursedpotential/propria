// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package sourceformat

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// ClaudeAIExportJSON is the native claude.ai conversation export format, with uuid/name/chat_messages.
const ClaudeAIExportJSON = "claude_ai_export_json"

// ChatGPTMarkdown is a Markdown export with ordered Prompt and Response headings and nonempty bodies.
const ChatGPTMarkdown = "chatgpt_markdown"

// ClaudeMarkdown is a Markdown transcript with ordered Human and Assistant headings and nonempty bodies.
const ClaudeMarkdown = "claude_markdown"

// GeminiMarkdown is a native Markdown transcript with paired bold You and Gemini labels.
const GeminiMarkdown = "gemini_markdown"

// claudeConversationSignature checks a complete native conversation without inferring message fields.
// Inputs: a decoded first conversation. Output: whether uuid/name and typed
// messages exist, or its explicitly empty message array is a native envelope.
// Side effects: none. Pick this for complete objects and complete first members of bounded array prefixes.
func claudeConversationSignature(first map[string]json.RawMessage) bool {
	var id, name string
	if json.Unmarshal(first["uuid"], &id) != nil || strings.TrimSpace(id) == "" || json.Unmarshal(first["name"], &name) != nil {
		return false
	}
	var messages []struct {
		UUID        string          `json:"uuid"`
		Sender      string          `json:"sender"`
		Text        *string         `json:"text"`
		Content     json.RawMessage `json:"content"`
		Attachments json.RawMessage `json:"attachments"`
		Files       json.RawMessage `json:"files"`
	}
	if json.Unmarshal(first["chat_messages"], &messages) != nil || messages == nil {
		return false
	}
	if len(messages) == 0 {
		return true
	}
	knownRole := false
	for _, message := range messages {
		if strings.TrimSpace(message.UUID) == "" || strings.TrimSpace(message.Sender) == "" {
			return false
		}
		if message.Sender == "human" || message.Sender == "assistant" {
			knownRole = true
		}
		if message.Text != nil {
			continue
		}
		found := false
		for _, native := range []json.RawMessage{message.Content, message.Attachments, message.Files} {
			var records []json.RawMessage
			if json.Unmarshal(native, &records) == nil && records != nil {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return knownRole
}

var aiHeading = regexp.MustCompile(`^(?:#{1,6}\s+)(Prompt|Response|Human|Assistant):?\s*$`)
var geminiRoleLabel = regexp.MustCompile(`^\*\*(You|Gemini):\*\*\s*(.*)$`)

// detectAIMarkdown recognizes paired role headings outside fenced code, with content for both turns.
// Inputs: UTF-8 bytes. Outputs: ChatGPT, Claude or Gemini Markdown format/signature, otherwise empty IDs.
// Side effects: none. Pick this for exported Markdown; ordinary role prose is not a signature.
func detectAIMarkdown(content []byte) (string, string) {
	var role string
	var body bool
	var completeUser bool
	var family string
	fence := ""
	for _, line := range bytes.Split(content, []byte{'\n'}) {
		text := strings.TrimSpace(string(line))
		if strings.HasPrefix(text, "```") || strings.HasPrefix(text, "~~~") {
			marker := text[:3]
			if fence == "" {
				fence = marker
				if role != "" {
					body = true
				}
			} else if fence == marker {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		match := aiHeading.FindStringSubmatch(text)
		gemini := geminiRoleLabel.FindStringSubmatch(text)
		if match != nil || gemini != nil {
			if role == "Prompt" || role == "Human" || role == "You" {
				completeUser = body
			}
			if (role == "Response" || role == "Assistant" || role == "Gemini") && completeUser && body {
				return markdownIDs(family)
			}
			newRole := ""
			inlineBody := ""
			if gemini != nil {
				newRole, inlineBody = gemini[1], gemini[2]
			} else {
				newRole = match[1]
			}
			newFamily := "claude"
			if newRole == "Prompt" || newRole == "Response" {
				newFamily = "chatgpt"
			} else if newRole == "You" || newRole == "Gemini" {
				newFamily = "gemini"
			}
			if family != "" && newFamily != family {
				return "", ""
			}
			family = newFamily
			if (newRole == "Response" || newRole == "Assistant" || newRole == "Gemini") && !completeUser {
				return "", ""
			}
			role = newRole
			body = strings.TrimSpace(inlineBody) != ""
		} else if text != "" && role != "" {
			body = true
		}
	}
	if (role == "Response" || role == "Assistant" || role == "Gemini") && completeUser && body {
		return markdownIDs(family)
	}
	return "", ""
}

// markdownIDs returns the exact public format/signature pair for a recognized export family.
// Input: chatgpt, claude or gemini. Output: registered format/signature IDs. Side effects: none.
// Pick after validating paired headings outside code fences.
func markdownIDs(family string) (string, string) {
	if family == "chatgpt" {
		return ChatGPTMarkdown, "chatgpt_prompt_response_markdown_v1"
	}
	if family == "gemini" {
		return GeminiMarkdown, "gemini_you_gemini_markdown_v1"
	}
	return ClaudeMarkdown, "claude_human_assistant_markdown_v1"
}
