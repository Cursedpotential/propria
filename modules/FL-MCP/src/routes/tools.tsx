// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { createFileRoute } from "@tanstack/react-router";
import { ToolFormCard } from "@/components/tools/tool-form-card";

export const Route = createFileRoute("/tools")({
  component: ToolsPage,
});

function ToolsPage() {
  return (
    <div className="space-y-3">
      <ToolFormCard
        title="Survival Guide"
        description="Ask for guidance from the plugin's survival_guide MCP tool (custody-guide content, Genesee County specific)."
        placeholder="e.g. What should I do before a parenting-time exchange that's likely to be tense?"
        buildPrompt={(input) => `Use the survival_guide tool to answer this: ${input}`}
      />
      <ToolFormCard
        title="Court-Safe Language Review"
        description="Ask the plugin's court_language_review MCP tool to rewrite text for a filing, message, or note into court-safe, non-inflammatory language."
        placeholder="Paste the text you want reviewed for court-safe language…"
        buildPrompt={(input) => `Use the court_language_review tool to review the following text for court-safe language:\n\n${input}`}
      />
    </div>
  );
}
