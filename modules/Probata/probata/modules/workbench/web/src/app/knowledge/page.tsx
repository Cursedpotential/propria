// Byline: Codex · GPT-5 · 2026-08-15 (case-scoped Knowledge MVP)
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-24: Graphiti pane removed)
import { KnowledgeBrowser } from "@/components/knowledge/knowledge-browser";

export default function KnowledgePage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Knowledge</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Search the evidence knowledge projection and browse canonical case knowledge.
        </p>
      </div>
      <KnowledgeBrowser />
    </div>
  );
}
