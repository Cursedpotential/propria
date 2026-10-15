package sourceformat

import (
	"strings"
	"testing"
)

// TestGeminiExporterProviderBeatsSharedHeadings checks the actual exporter format's provider identity.
// Inputs: synthetic preamble URLs and role headings. Outputs: format classifications.
// Effects: none; choose to prevent Gemini exports being placed under ChatGPT.
func TestGeminiExporterProviderBeatsSharedHeadings(t *testing.T) {
	for _, path := range []string{"app/thread_12", "share/thread-34/"} {
		text := "# Export\n[Conversation](https://gemini.google.com/" + path + ")\n\n## Prompt:\nHello\n## Response:\nAnswer\n"
		format, signature := detectAIMarkdown([]byte(text))
		if format != GeminiMarkdown || signature != "gemini_prompt_response_markdown_v1" {
			t.Fatalf("provider-grounded export missed: %q %q", format, signature)
		}
		format, _ = detectAIMarkdown([]byte(strings.ReplaceAll(strings.ToLower(text), "\n", "\r\n")))
		if format != GeminiMarkdown {
			t.Fatal("case-insensitive CRLF exporter roles lost Gemini identity")
		}
	}
	for _, preamble := range []string{"", "https://gemini.google.com.evil.test/app/thread", "https://gemini.google.com/help", "```\nhttps://gemini.google.com/app/thread\n```"} {
		format, _ := detectAIMarkdown([]byte(preamble + "\n## Prompt:\nHello\n## Response:\nAnswer"))
		if format == GeminiMarkdown {
			t.Fatalf("unverified provider accepted: %q", preamble)
		}
	}
	for _, body := range []string{"## Response:\nAnswer", "### Prompt:\nHello\n### Response:\nAnswer", "## Prompt:\nHello\n## Prompt:\nAgain\n## Response:\nAnswer"} {
		format, _ := detectAIMarkdown([]byte("https://gemini.google.com/app/thread\n" + body))
		if format != "" {
			t.Fatalf("malformed provider export classified as %q", format)
		}
	}
}

// TestGeminiMarkdownSignature requires paired native labels outside fenced code, including inline bodies.
// Inputs: exact original Markdown snippets. Outputs: Gemini format only for a real paired transcript.
// Effects: none. Pick for format admission before native span verification.
func TestGeminiMarkdownSignature(t *testing.T) {
	for _, text := range []string{"**You:**\nHello 🧭\n\n**Gemini:**\nAnswer", "**You:** Hello 🧭\n**Gemini:** Answer"} {
		format, signature := detectAIMarkdown([]byte(text))
		if format != GeminiMarkdown || signature != "gemini_you_gemini_markdown_v1" {
			t.Fatalf("Gemini transcript missed: %q %q", format, signature)
		}
	}
	for _, text := range []string{"```markdown\n**You:** Hello\n**Gemini:** Answer\n```", "**Gemini:** Answer", "**You:** Hello\n**Gemini:**"} {
		format, _ := detectAIMarkdown([]byte(text))
		if format == GeminiMarkdown {
			t.Fatalf("nontranscript admitted: %q", text)
		}
	}
}
