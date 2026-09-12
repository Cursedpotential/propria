// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — non-blocking, readable store state.
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { CalendarClock, Gavel, Users } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { summaryQuery } from "@/lib/queries";
import { isUnavailable } from "@/types/store";

export const Route = createFileRoute("/")({
  // Warm the cache without blocking the application shell. If the external
  // case store is offline, the operator still gets navigation and a useful
  // recovery state instead of an empty graphite window.
  loader: ({ context }) => {
    void context.queryClient.prefetchQuery(summaryQuery());
  },
  component: CaseStatusPage,
});

function CaseStatusPage() {
  const { data, error, isPending, refetch, isFetching } = useQuery(summaryQuery());

  if (isPending) {
    return <CaseStoreState title="Connecting to the case store" detail="Loading the current case summary and custody-safe workspace state." />;
  }

  if (error || !data) {
    return (
      <CaseStoreState
        title="Case store unavailable"
        detail={error instanceof Error ? error.message : "The local case store did not return a summary."}
        action={
          <button
            type="button"
            onClick={() => void refetch()}
            disabled={isFetching}
            className="min-h-9 rounded-[var(--radius-sm)] border border-accent-border bg-accent-fill px-3 py-1.5 text-[13px] font-semibold text-accent-text transition-colors hover:bg-surface-hover disabled:cursor-wait disabled:opacity-60"
          >
            {isFetching ? "Trying again…" : "Try again"}
          </button>
        }
      />
    );
  }

  if (isUnavailable(data)) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Case store unavailable</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-text-secondary">{data.reason}</CardContent>
      </Card>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Gavel className="size-4" aria-hidden /> Court
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 text-sm">
          <Row label="County" value={data.county} />
          <Row label="Court" value={data.court} />
          <Row label="Judge" value={data.judge} />
          <Row label="Referee" value={data.referee} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CalendarClock className="size-4" aria-hidden /> Upcoming
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <Row label="Next hearing" value={data.next_hearing} mono />
          <div className="pt-1">
            <div className="mb-1 text-xs font-medium text-text-tertiary">Deadlines</div>
            {data.deadlines.length === 0 && <p className="text-xs text-text-tertiary">None on record.</p>}
            <ul className="space-y-1">
              {data.deadlines.slice(0, 6).map((d, i) => (
                <li key={i} className="flex items-center justify-between gap-2">
                  <span>{d.label}</span>
                  <span className="font-mono text-xs tabular-nums text-text-secondary">{d.due}</span>
                </li>
              ))}
            </ul>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="size-4" aria-hidden /> Parties & children
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex flex-wrap gap-1">
            {data.parties.map((p, i) => (
              <Badge key={i} tone="neutral">
                {p}
              </Badge>
            ))}
          </div>
          <p className="text-xs text-text-tertiary">{data.children.count} child(ren) on record.</p>
          {data.flags.length > 0 && (
            <div className="flex flex-wrap gap-1 pt-1">
              {data.flags.map((f, i) => (
                <Badge key={i} tone="warn">
                  {f}
                </Badge>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="lg:col-span-3">
        <CardHeader>
          <CardTitle>Controlling orders</CardTitle>
        </CardHeader>
        <CardContent>
          {data.controlling_orders.length === 0 ? (
            <p className="text-sm text-text-tertiary">None on record.</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-text-tertiary">
                  <th className="py-1.5 font-medium">Title</th>
                  <th className="py-1.5 font-medium">Entered</th>
                  <th className="py-1.5 font-medium">Served</th>
                </tr>
              </thead>
              <tbody>
                {data.controlling_orders.map((o, i) => (
                  <tr key={i} className="border-b border-border last:border-0">
                    <td className="py-1.5">{o.title}</td>
                    <td className="py-1.5 font-mono text-xs tabular-nums text-text-secondary">{o.entered}</td>
                    <td className="py-1.5 font-mono text-xs tabular-nums text-text-secondary">{o.served ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function CaseStoreState({ title, detail, action }: { title: string; detail: string; action?: React.ReactNode }) {
  return (
    <section className="max-w-2xl border-l-2 border-accent px-4 py-3" aria-live="polite">
      <p className="text-[12px] font-semibold leading-5 text-accent-text">Workspace status</p>
      <h1 className="mt-1 text-lg font-semibold leading-6 text-text-primary">{title}</h1>
      <p className="mt-1 max-w-prose text-sm leading-6 text-text-secondary">{detail}</p>
      {action && <div className="mt-3">{action}</div>}
    </section>
  );
}

function Row({ label, value, mono }: { label: string; value: string | null; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-text-tertiary">{label}</span>
      <span className={mono ? "font-mono tabular-nums text-text-secondary" : "text-text-primary"}>{value ?? "—"}</span>
    </div>
  );
}
