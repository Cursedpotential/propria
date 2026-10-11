// Byline: Claude Code · Sonnet (agent) · 2026-07-20
// Byline: Codex · GPT-5.6-Sol · 2026-08-30 (monitored Atomic Tools surface)
import { SourcePinnedAction } from "@/components/tools/source-pinned-action";
import { ToolExplorer } from "@/components/tools/tool-explorer";

export default function ToolsPage() {
  return (
    <div className="mx-auto w-full max-w-[1600px] space-y-5 px-5 py-6 lg:px-8 lg:py-8">
      <header className="border-b pb-5">
        <p className="platform-kicker">Source actions and catalog</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">Tools</h1>
        <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">
          Run an approved source-file action with its verified hash, or browse the governed tool catalog.
        </p>
      </header>
      <SourcePinnedAction />
      <details className="border bg-card p-4">
        <summary className="cursor-pointer text-sm font-semibold">Browse published tools</summary>
        <div className="mt-4"><ToolExplorer /></div>
      </details>
    </div>
  );
}
