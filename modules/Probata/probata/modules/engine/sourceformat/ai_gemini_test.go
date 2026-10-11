package sourceformat

import "testing"

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
