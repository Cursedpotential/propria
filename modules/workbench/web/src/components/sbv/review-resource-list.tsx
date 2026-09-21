// Byline: Claude Code · Fable 5.1 · 2026-09-20
// Owner 2026-09-20 23:10: "Sources and proposals — this section sucks now, it's
// massive and hard to nav". Every run was a six-line card titled with a
// front-truncated b2:// path (so all titles looked alike), repeating one
// boilerplate paragraph, with failed attempts mixed in. This is a one-line-per-run
// list: file name as the title, failed attempts behind a filter, a name search.
"use client";

import { CircleDot, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { Input } from "@/components/ui/input";
import type { ProfferProposalResource } from "@/lib/shared/types";

type Filter = "reviewable" | "failed" | "all";

interface ReviewResourceListProps {
  resources: ProfferProposalResource[];
  loading: boolean;
  selectedHandle: string | null;
  onSelect: (handle: string) => void;
}

/** The part of a source ref a person recognises: the file, plus its parent when it is a derived chunk. */
function resourceName(sourceRef: string): { name: string; context: string } {
  const path = sourceRef.replace(/^[a-z0-9]+:\/\/[^/]+\//i, "");
  const segments = path.split("/").filter(Boolean);
  const name = segments.at(-1) ?? sourceRef;
  const derivedAt = segments.findIndex((segment) => segment.endsWith(".derived"));
  if (derivedAt >= 0) {
    return { name, context: segments[derivedAt].replace(/\.derived$/, "") };
  }
  return { name, context: segments.slice(-3, -1).join("/") };
}

function isFailed(resource: ProfferProposalResource) {
  return resource.lifecycle === "failed" || resource.lifecycle === "unavailable";
}

function when(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : date.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

const DOT: Record<string, string> = {
  completed: "bg-emerald-500",
  failed: "bg-destructive",
  unavailable: "bg-destructive",
};

export function ReviewResourceList({ resources, loading, selectedHandle, onSelect }: ReviewResourceListProps) {
  const [filter, setFilter] = useState<Filter>("reviewable");
  const [query, setQuery] = useState("");

  const counts = useMemo(() => {
    const failed = resources.filter(isFailed).length;
    return { reviewable: resources.length - failed, failed, all: resources.length };
  }, [resources]);

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return resources.filter((resource) => {
      // The selected run always stays visible, so a deep link to a failed attempt is never hidden.
      if (resource.preview_handle === selectedHandle) return true;
      if (filter === "reviewable" && isFailed(resource)) return false;
      if (filter === "failed" && !isFailed(resource)) return false;
      return !needle || resource.source_ref.toLowerCase().includes(needle);
    });
  }, [resources, filter, query, selectedHandle]);

  if (loading && resources.length === 0) {
    return (
      <div className="flex items-center gap-2 px-4 py-6 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> Loading available sources and proposals…
      </div>
    );
  }
  if (resources.length === 0) {
    return (
      <div className="px-5 py-10 text-center">
        <CircleDot className="mx-auto size-5 text-muted-foreground" />
        <p className="mt-3 text-sm font-semibold">No context proposals are available</p>
        <p className="mt-1 text-xs text-muted-foreground">Start intake to select source material and create the first reviewable attempt.</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex flex-wrap items-center gap-2 border-b px-4 py-2">
        <div role="group" aria-label="Filter runs" className="flex items-center gap-1">
          {(["reviewable", "failed", "all"] as const).map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={filter === value}
              onClick={() => setFilter(value)}
              className={`rounded-md px-2 py-1 text-xs capitalize focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${filter === value ? "bg-accent font-semibold" : "text-muted-foreground hover:bg-accent/40"}`}
            >
              {value} <span className="tabular-nums">{counts[value]}</span>
            </button>
          ))}
        </div>
        <label className="relative ml-auto w-full max-w-xs">
          <span className="sr-only">Find a source by name</span>
          <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Find by file name" className="h-8 pl-7 text-xs" />
        </label>
      </div>
      {visible.length === 0 ? (
        <p className="px-4 py-4 text-xs text-muted-foreground">No runs match this filter.</p>
      ) : (
        <ul className="max-h-56 divide-y overflow-y-auto" aria-label="Sources and proposals">
          {visible.map((resource) => {
            const { name, context } = resourceName(resource.source_ref);
            const selected = resource.preview_handle === selectedHandle;
            return (
              <li key={resource.preview_handle}>
                <button
                  type="button"
                  aria-pressed={selected}
                  onClick={() => onSelect(resource.preview_handle)}
                  title={`${resource.source_ref}\n${resource.representation_detail}`}
                  className={`grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-4 py-1.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring ${selected ? "bg-accent" : "hover:bg-accent/40"}`}
                >
                  <span aria-hidden className={`size-2 rounded-full ${DOT[resource.lifecycle] ?? "bg-amber-500"}`} />
                  <span className="min-w-0">
                    <strong className="block truncate text-sm font-medium">{name}</strong>
                    {context && <span className="block truncate text-[11px] text-muted-foreground">{context}</span>}
                  </span>
                  <span className="whitespace-nowrap text-right text-[11px] capitalize text-muted-foreground">
                    {resource.lifecycle.replaceAll("_", " ")} · {resource.completed_stage_count} stages · {resource.representation_state === "committed_readback" ? "Committed readback" : "Precommit proposal"}
                    <span className="block normal-case">{when(resource.created_at)}</span>
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
