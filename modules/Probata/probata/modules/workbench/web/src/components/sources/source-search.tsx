// Byline: Claude Code · Opus 5 · 2026-09-22
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (Names and paths is the B2 root search; it never waits on Intake)
// Search on Sources, from step one (owner, 2026-09-22 09:06), in four modes.
//
// The modes are not four search engines in Probata. Names and paths is the
// server-scoped search of the B2 root being listed (sources-screen.tsx runSearch);
// contents, meaning and relationships call the Intake
// (Consignatio) discovery service at INTAKE_DISCOVERY_INDEX_URL through the
// same BFF routes the Intake app uses, so nothing here has to be rebuilt when
// the Xplorer-based Intake becomes the front door. A mode whose backend is not
// reachable shows ONE small flag on the mode — never a caveat paragraph.
"use client";

import { Loader2, Search } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import type { DiscoveryCapabilities } from "@/lib/discovery-types";
import { cn } from "@/lib/utils";

export type SearchMode = "names" | "contents" | "meaning" | "relationships";

export const SEARCH_MODES: Array<{ id: SearchMode; label: string; wire: string }> = [
  { id: "names", label: "Names and paths", wire: "filename_substring" },
  { id: "contents", label: "Contents", wire: "contents" },
  { id: "meaning", label: "Meaning", wire: "hybrid" },
  { id: "relationships", label: "Relationships", wire: "graph" },
];

/** True when this mode's backend is reachable right now. */
export function modeAvailable(mode: SearchMode, capabilities: DiscoveryCapabilities | null): boolean {
  if (mode === "names") return true;
  if (!capabilities) return false;
  if (mode === "contents") return Boolean(capabilities.modes?.contents);
  if (mode === "meaning") return Boolean(capabilities.modes?.hybrid);
  return Boolean(capabilities.graph);
}

export function SourceSearch({
  query,
  mode,
  capabilities,
  searching,
  onQueryChange,
  onModeChange,
  onSubmit,
  onClear,
  resultSummary,
}: {
  query: string;
  mode: SearchMode;
  capabilities: DiscoveryCapabilities | null;
  searching: boolean;
  onQueryChange: (value: string) => void;
  onModeChange: (mode: SearchMode) => void;
  onSubmit: () => void;
  onClear: () => void;
  resultSummary: string | null;
}) {
  const [touched, setTouched] = useState(false);

  return (
    <form
      className="flex flex-wrap items-center gap-2 border-b bg-card px-3 py-2"
      onSubmit={(event) => {
        event.preventDefault();
        setTouched(true);
        onSubmit();
      }}
      aria-label="Search sources"
    >
      <span className="relative min-w-0 flex-1">
        <Search className="pointer-events-none absolute left-2.5 top-2 h-4 w-4 text-muted-foreground" />
        <input
          className="h-8 w-full border bg-background pl-8 pr-2 text-xs"
          value={query}
          placeholder="Search the corpus"
          aria-label="Search the corpus"
          onChange={(event) => onQueryChange(event.target.value)}
        />
      </span>
      <div className="flex items-center gap-1" role="group" aria-label="Search mode">
        {SEARCH_MODES.map((entry) => {
          const available = modeAvailable(entry.id, capabilities);
          return (
            <button
              key={entry.id}
              type="button"
              aria-pressed={mode === entry.id}
              onClick={() => onModeChange(entry.id)}
              className={cn(
                "flex min-h-8 items-center gap-1 border px-2 text-[11px] font-semibold",
                mode === entry.id ? "border-primary bg-accent text-accent-foreground" : "hover:bg-accent/50",
              )}
            >
              {entry.label}
              {!available && (
                <span
                  className="border border-[#c58214] bg-[#fff4dd] px-1 text-[9px] font-semibold uppercase text-[#684b18] dark:bg-[#43351f] dark:text-[#ffe0a6]"
                  title="This search mode has no index yet"
                >
                  not available yet
                </span>
              )}
            </button>
          );
        })}
      </div>
      <Button type="submit" size="sm" disabled={searching || !query.trim()}>
        {searching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Search className="h-3.5 w-3.5" />} Search
      </Button>
      {(resultSummary || touched) && (
        <Button type="button" size="sm" variant="outline" onClick={() => { setTouched(false); onClear(); }}>
          Clear
        </Button>
      )}
      {resultSummary && (
        <span className="text-[11px] text-muted-foreground" role="status" aria-live="polite">
          {resultSummary}
        </span>
      )}
    </form>
  );
}
