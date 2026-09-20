// Byline: OpenAI Codex · GPT-5 · 2026-09-12
// SDA-04: compact, truthful cross-surface context without merging product authority.
import { ShieldCheck } from "lucide-react";

const CONTEXT = [
  { term: "Surface", detail: "Family Court Console" },
  { term: "Scope", detail: "Local case workspace" },
  { term: "Authority", detail: "Planning aid" },
] as const;

export function ContextStrip() {
  return (
    <section
      aria-label="Workspace context"
      className="flex min-h-[var(--pr-context-strip-min-height)] flex-wrap items-center gap-x-5 gap-y-2 border-b border-border bg-bg-inset px-4 py-2 text-[13px]"
    >
      <ShieldCheck className="size-4 shrink-0 text-accent" aria-hidden />
      <dl className="flex min-w-0 flex-1 flex-wrap items-center gap-x-5 gap-y-1">
        {CONTEXT.map(({ term, detail }) => (
          <div key={term} className="flex min-w-0 items-baseline gap-1.5">
            <dt className="text-text-tertiary">{term}</dt>
            <dd className="m-0 font-medium text-text-primary">{detail}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
