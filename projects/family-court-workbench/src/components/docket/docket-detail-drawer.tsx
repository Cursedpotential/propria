// Byline: Claude Code · Sonnet 5 · 2026-09-07
import { ExternalLink, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { DocketEntry } from "@/types/store";

/** `record.source` shape is owner-defined per row (see mcp-app store.ts's
 * `CaseSourceResult`) — we only know it MAY carry `path` (local file) and/or
 * `r2_path`. Treat both defensively rather than assuming one is always present. */
function readSource(record: Record<string, unknown>): { path: string | null; r2Path: string | null } {
  const source = record.source && typeof record.source === "object" ? (record.source as Record<string, unknown>) : null;
  const path = typeof source?.path === "string" ? source.path : null;
  const r2Path = typeof source?.r2_path === "string" ? source.r2_path : null;
  return { path, r2Path };
}

async function openLocalFile(path: string) {
  try {
    const { openPath } = await import("@tauri-apps/plugin-opener");
    await openPath(path);
  } catch (err) {
    // Not running under Tauri (plain browser dev) — nothing sane to do
    // beyond telling the user; there is no window.open() equivalent for
    // an arbitrary local filesystem path from a browser tab.
    console.error("openPath unavailable outside Tauri", err);
  }
}

export function DocketDetailDrawer({ entry, onClose }: { entry: DocketEntry; onClose: () => void }) {
  const { path, r2Path } = readSource(entry.record);
  const inForce = entry.table === "order" ? entry.record.in_force : undefined;

  return (
    <div role="dialog" aria-label={`Detail: ${entry.title}`} className="fixed inset-y-0 right-0 z-30 w-96 border-l border-border bg-surface">
      <div className="flex h-10 items-center justify-between border-b border-border px-3">
        <span className="text-sm font-medium text-text-primary">Docket detail</span>
        <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close">
          <X className="size-4" />
        </Button>
      </div>
      <div className="space-y-3 p-3 text-sm">
        <div>
          <div className="text-xs text-text-tertiary">Title</div>
          <div>{entry.title}</div>
        </div>
        <div className="flex gap-4">
          <div>
            <div className="text-xs text-text-tertiary">Table</div>
            <Badge tone="neutral">{entry.table}</Badge>
          </div>
          <div>
            <div className="text-xs text-text-tertiary">Status</div>
            <Badge tone={inForce ? "good" : "neutral"}>{entry.status ?? "—"}</Badge>
          </div>
          {inForce !== undefined && (
            <div>
              <div className="text-xs text-text-tertiary">In force</div>
              <Badge tone={inForce ? "good" : "critical"}>{inForce ? "yes" : "no"}</Badge>
            </div>
          )}
        </div>
        <div>
          <div className="text-xs text-text-tertiary">Date</div>
          <div className="font-mono tabular-nums">{entry.date ?? "—"}</div>
        </div>
        <div>
          <div className="text-xs text-text-tertiary">Source</div>
          {path ? (
            <button onClick={() => void openLocalFile(path)} className="flex items-center gap-1 text-accent-text underline underline-offset-2">
              <ExternalLink className="size-3.5" aria-hidden />
              Open file
            </button>
          ) : r2Path ? (
            <span className="font-mono text-xs text-text-secondary">{r2Path}</span>
          ) : (
            <span className="text-text-tertiary">No source recorded</span>
          )}
        </div>
      </div>
    </div>
  );
}
