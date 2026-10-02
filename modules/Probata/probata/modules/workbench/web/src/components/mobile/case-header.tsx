// Byline: Claude Code · Sonnet · 2026-10-02
// Read-only case header for the mobile shell: the live case and its people, from the registry (the one identity store).
import { useQuery } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

import { getCaseIdentity } from "@/lib/case-identity-client";
import { cn } from "@/lib/utils";

// Wire value of the live case on the existing routes; the screen only ever says "live".
const LIVE_MODE = "REAL" as const;

const ROLE_LABEL: Record<string, string> = { user: "You", co_parent: "Co-parent" };

export function CaseHeader() {
  const [open, setOpen] = useState(false);
  const { data, isPending, isError } = useQuery({
    queryKey: ["m-case"],
    queryFn: () => getCaseIdentity(LIVE_MODE),
    staleTime: 5 * 60_000,
  });
  const title = data?.matter?.title ?? "Live case";
  const caption = data?.court_case?.caption;
  const docket = data?.court_case?.docket_number;
  return (
    <header className="sticky top-0 z-30 bg-nav pt-[env(safe-area-inset-top)] text-nav-foreground">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        className="flex min-h-14 w-full items-center gap-3 px-4 text-left active:bg-white/5"
      >
        <span className="rounded-full bg-emerald-500/20 px-2.5 py-1 text-xs font-bold uppercase tracking-wide text-emerald-300">live</span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold leading-tight">{isPending ? "Loading case" : title}</span>
          <span className="block truncate text-xs text-nav-foreground/70">
            {isError ? "Case details unavailable" : docket ? `No. ${docket}` : "Probata"}
          </span>
        </span>
        <ChevronDown className={cn("size-5 shrink-0 transition-transform", open && "rotate-180")} />
      </button>
      {open ? (
        <div className="space-y-3 border-t border-white/10 px-4 pb-4 pt-3 text-sm">
          {caption ? <p className="font-medium">{caption}</p> : null}
          {data?.court_case?.court_name ? <p className="text-xs text-nav-foreground/70">{data.court_case.court_name}</p> : null}
          {data?.court_case?.presiding_judge ? <p className="text-xs text-nav-foreground/70">Judge {data.court_case.presiding_judge}</p> : null}
          <ul className="space-y-2">
            {(data?.people ?? []).map((person) => (
              <li key={person.id} className="flex items-baseline justify-between gap-3 rounded-md bg-white/5 px-3 py-2">
                <span className="font-medium">{person.display_name}</span>
                <span className="text-xs text-nav-foreground/70">
                  {ROLE_LABEL[person.role_in_case ?? ""] ?? person.role_in_case ?? "Person"}
                </span>
              </li>
            ))}
          </ul>
          <p className="text-xs text-nav-foreground/60">Read only. Edit people and numbers on the desktop Case page.</p>
        </div>
      ) : null}
    </header>
  );
}
