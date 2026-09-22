// Byline: Claude Code · Opus 5 · 2026-09-22
// LEFT region of Sources: the folder tree over the server-allowlisted B2 roots.
//
// Reuses the listing logic the Intake source explorer proved (server-scoped
// prefix browsing through GET /api/proffer/sources, never a client-side filter,
// never a caller-supplied bucket) without its page layout: this is a tree in a
// resizable panel, not a full-width card with a six-step rail above it.
"use client";

import { ChevronRight, FolderOpen, Home, Loader2 } from "lucide-react";

import type { ProfferSourcePrefix, ProfferSourceRoot } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

function breadcrumbParts(prefix: string) {
  const parts = prefix.split("/").filter(Boolean);
  return parts.map((label, index) => ({ label, prefix: `${parts.slice(0, index + 1).join("/")}/` }));
}

export function SourceTree({
  roots,
  activeRootId,
  prefix,
  prefixes,
  loading,
  unitRoots,
  onRootChange,
  onPrefixChange,
  onSelectFolder,
  selectedFolder,
}: {
  roots: readonly ProfferSourceRoot[];
  activeRootId: string;
  prefix: string;
  prefixes: readonly ProfferSourcePrefix[];
  loading: boolean;
  /** Folder prefixes known to be a unit — catalog-detected or hand-marked. */
  unitRoots: ReadonlyMap<string, string>;
  onRootChange: (rootId: string) => void;
  onPrefixChange: (prefix: string) => void;
  onSelectFolder: (prefix: string) => void;
  selectedFolder: string | null;
}) {
  const parts = breadcrumbParts(prefix);

  return (
    <div className="flex h-full min-h-0 flex-col border-r bg-card" aria-label="Source folders">
      <div className="border-b px-3 py-2">
        <label className="grid gap-1 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
          Location
          <select
            value={activeRootId}
            onChange={(event) => onRootChange(event.target.value)}
            className="h-8 border bg-background px-2 text-xs font-normal text-foreground"
            disabled={!roots.length}
            aria-label="Source location"
          >
            {!roots.length && <option value="">Loading locations</option>}
            {roots.map((root) => (
              <option key={root.root_id} value={root.root_id}>
                {root.label}
                {root.temporary ? " (temporary)" : ""}
              </option>
            ))}
          </select>
        </label>
      </div>

      <nav className="flex flex-wrap items-center gap-0.5 border-b px-3 py-1.5 text-[11px]" aria-label="Folder path">
        <button type="button" className="flex items-center gap-1 px-1 py-0.5 font-semibold hover:text-primary" onClick={() => onPrefixChange("")}>
          <Home className="h-3 w-3" /> Root
        </button>
        {parts.map((part) => (
          <span key={part.prefix} className="flex min-w-0 items-center">
            <ChevronRight className="h-3 w-3 shrink-0 text-muted-foreground" />
            <button
              type="button"
              className="max-w-40 truncate px-1 py-0.5 hover:text-primary"
              title={part.label}
              onClick={() => onPrefixChange(part.prefix)}
            >
              {part.label}
            </button>
          </span>
        ))}
      </nav>

      <div className="min-h-0 flex-1 overflow-auto" aria-busy={loading}>
        {loading && !prefixes.length ? (
          <p className="flex items-center gap-2 px-3 py-4 text-xs text-muted-foreground" role="status">
            <Loader2 className="h-3.5 w-3.5 animate-spin" /> Loading folders
          </p>
        ) : !prefixes.length ? (
          <p className="px-3 py-4 text-xs text-muted-foreground">No folders here. Files are listed in the centre.</p>
        ) : (
          <ul className="py-1">
            {prefixes.map((entry) => {
              const unitType = unitRoots.get(entry.prefix.replace(/\/$/, ""));
              const selected = selectedFolder === entry.prefix;
              return (
                <li key={entry.prefix}>
                  <div
                    className={cn(
                      "flex w-full items-center gap-1 px-2 text-xs",
                      selected && "bg-accent text-accent-foreground",
                    )}
                  >
                    <button
                      type="button"
                      className="flex min-w-0 flex-1 items-center gap-1.5 py-1.5 text-left hover:text-primary"
                      onClick={() => onSelectFolder(entry.prefix)}
                      title={entry.prefix}
                    >
                      <FolderOpen className="h-3.5 w-3.5 shrink-0" />
                      <span className="truncate">{entry.name}</span>
                      {unitType && (
                        <span className="shrink-0 border border-[#4051b9] bg-[#e9ecfb] px-1 text-[9px] font-semibold uppercase text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]">
                          {unitType.replaceAll("_", " ")}
                        </span>
                      )}
                    </button>
                    <button
                      type="button"
                      className="shrink-0 px-1 py-1.5 text-muted-foreground hover:text-primary"
                      aria-label={`Open folder ${entry.name}`}
                      onClick={() => onPrefixChange(entry.prefix)}
                    >
                      <ChevronRight className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
