// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Shown when the store itself reports `{ available: false, reason }` — a
// genuine runtime condition (the SurrealDB native driver module failed to
// load, the shared server at ws://127.0.0.1:8471 is unreachable, etc.),
// distinct from "not implemented yet" (which no longer exists anywhere in
// this app — see types/store.ts).
import { AlertTriangle } from "lucide-react";

export function UnavailableNotice({ reason }: { reason: string }) {
  return (
    <div className="flex items-start gap-2 rounded-[var(--radius-md)] border border-warn-border bg-warn-fill px-3 py-2.5 text-sm text-warn-text">
      <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div>
        <div className="font-medium">Case store unavailable</div>
        <p className="mt-0.5 text-xs opacity-90">{reason}</p>
      </div>
    </div>
  );
}
