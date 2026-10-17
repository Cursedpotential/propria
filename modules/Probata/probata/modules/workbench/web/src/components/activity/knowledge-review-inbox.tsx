import { useQuery } from "@tanstack/react-query";
import { useBrowserSearchParams } from "@/lib/router-compat";
import { KnowledgeReviewDetail } from "./knowledge-review-detail";
import { AlertTriangle, ArrowUpRight, CheckCircle2, RefreshCcw } from "lucide-react";

import { AppLink } from "@/lib/router-compat";

type Review = {
  review_ref: string;
  severity: "low" | "medium" | "high" | "critical";
  status: "queued" | "deferred";
  kind: string;
  scope: string;
  summary: string;
  source_count: number;
  review_href: string | null;
};

type ReviewInbox = {
  schema: "propria-knowledge-review-inbox/v1";
  status: "ready";
  total: number;
  items: Review[];
  truncated: boolean;
};

/** Read the canonical knowledge-conflict queue via same-origin Workbench auth.
 * Input: existing authenticated Workbench browser session. Output: bounded review metadata.
 * Effects: GET only; all review decisions remain on the existing owner-authorized route.
 */
async function loadInbox({ signal }: { signal?: AbortSignal }): Promise<ReviewInbox> {
  const response = await fetch("/api/knowledge-review/inbox", {
    method: "GET",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal
  });
  if (!response.ok) {
    throw new Error(response.status === 503
      ? "Knowledge reconciliation review is not connected yet."
      : `Knowledge review could not be loaded (HTTP ${response.status}).`);
  }
  const value: unknown = await response.json();
  if (!value || typeof value !== "object" || !("schema" in value)
      || value.schema !== "propria-knowledge-review-inbox/v1")
    throw new Error("Knowledge review returned an unexpected response.");
  return value as ReviewInbox;
}

/** Add knowledge-review alerts to the existing Activity surface, never a second approval queue.
 * Input: read-only Workbench inbox API. Output: compact, status-aware review cards.
 * Side effects: polling only; does not approve, edit, archive, or supersede any records.
 */
export function KnowledgeReviewInbox() {
  const params = useBrowserSearchParams();
  const selected = params.get("knowledge_review");
  const query = useQuery({
    queryKey: ["knowledge-review-inbox"],
    queryFn: ({ signal }) => loadInbox({ signal }),
    refetchInterval: 30_000,
    staleTime: 15_000,
    retry: false,
  });
  const reviews = query.data?.items ?? [];
  const critical = reviews.filter((item) =>
    item.severity === "critical" || item.severity === "high").length;

  return (
    <section className="space-y-3 border bg-card p-4" aria-label="Knowledge review alerts">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold">Knowledge review</h2>
          <p className="mt-1 text-xs text-muted-foreground">
            Conflicting, stale, or superseded memories and records needing a human decision.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs">
          {query.data ? <span aria-label={`${query.data.total} queued knowledge reviews`}
              className="rounded border px-2 py-1 text-muted-foreground">
              {query.data.total} pending
            </span> : null}
          {critical > 0 ? <span className="rounded border border-amber-500 px-2 py-1 text-amber-700 dark:text-amber-300">
            {critical} priority
          </span> : null}
          <button type="button" onClick={() => void query.refetch()}
            disabled={query.isFetching}
            className="inline-flex items-center gap-1 rounded border px-2 py-1 disabled:opacity-50">
            <RefreshCcw size={13} aria-hidden="true" /> Refresh
          </button>
        </div>
      </div>

      {query.isPending ? <p role="status" className="text-xs text-muted-foreground">Loading reviews…</p> : null}
      {query.isError ? <p role="alert" className="flex items-center gap-2 text-xs text-amber-700 dark:text-amber-300">
        <AlertTriangle size={15} aria-hidden="true" /> {query.error.message}
      </p> : null}
      {query.data && reviews.length === 0 ? <p className="flex items-center gap-2 text-xs text-muted-foreground">
        <CheckCircle2 size={14} aria-hidden="true" /> No actionable knowledge conflicts are currently queued.
      </p> : null}

      {reviews.length > 0 ? (
        <ul className="space-y-2">
          {reviews.map((item) => (
            <li key={item.review_ref} className="flex flex-wrap items-start justify-between gap-3 border-l-2 border-primary/40 bg-muted/20 px-3 py-2">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-semibold">{item.kind.replaceAll("_", " ")}</span>
                  <span className="text-xs text-muted-foreground">{item.severity} · {item.status}</span>
                </div>
                <p className="mt-1 text-sm">{item.summary}</p>
                <p className="mt-1 break-all text-xs text-muted-foreground">
                  {item.scope} · {item.source_count} sources · {item.review_ref}
                </p>
              </div>
              {item.review_href ? <AppLink className="inline-flex items-center gap-1 rounded border px-2 py-1 text-xs"
                href={item.review_href}>Open review <ArrowUpRight size={12} aria-hidden="true" /></AppLink>
                : <span className="text-xs text-muted-foreground">Decision link unavailable</span>}
            </li>
          ))}
        </ul>
      ) : null}
      {selected ? <KnowledgeReviewDetail key={selected} reviewRef={selected} /> : null}
      {query.data?.truncated ? <p className="text-xs text-muted-foreground">
        More reviews exist. Use the authoritative review queue for the full list.
      </p> : null}
    </section>
  );
}
