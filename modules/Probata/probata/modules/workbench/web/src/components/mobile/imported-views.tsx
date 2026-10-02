// Byline: Claude Code · Sonnet · 2026-10-02
// Imported (sources -> threads -> messages) and Calls, for the mobile shell. Read-only.
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useLayoutEffect, useMemo, useRef, useState } from "react";

import { Chip, Empty, ErrorBox, LoadMore, Loading, PageBar, StatusBadge } from "@/components/mobile/mobile-ui";
import { formatCount, formatDate, formatDateTime, formatDuration, formatRange, formatTime, statusLabel } from "@/components/mobile/mobile-format";
import { WhoIsThis } from "@/components/identity/who-is-this";
import { importedApi, type ImportedMessage, type ImportedSource, type ImportedThread, type SourceStatus } from "@/lib/imported-client";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { cn } from "@/lib/utils";

const FORMATS = ["All", "SMS", "Calls", "Facebook"] as const;

function Counts({ raw, normalized, committed }: { raw: number; normalized: number; committed: number }) {
  const cells: [string, number][] = [["Raw", raw], ["Normalized", normalized], ["Committed", committed]];
  return (
    <dl className="grid grid-cols-3 gap-2 text-center">
      {cells.map(([label, value]) => (
        <div key={label} className="rounded-md bg-muted/70 px-1 py-2">
          <dd className="text-base font-semibold tabular-nums">{formatCount(value)}</dd>
          <dt className="text-[11px] text-muted-foreground">{label}</dt>
        </div>
      ))}
    </dl>
  );
}

function SourceCard({ source }: { source: ImportedSource }) {
  const who = [source.format, source.device, source.owner].filter(Boolean).join(" · ");
  return (
    <AppLink href={`/m/source/${source.id}`} className="block rounded-xl border border-border bg-card p-4 active:bg-muted">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="break-all text-sm font-semibold leading-snug">{source.file_name}</p>
          <p className="mt-1 text-xs text-muted-foreground">{who}</p>
        </div>
        <StatusBadge status={source.status} />
      </div>
      <div className="mt-3">
        <Counts raw={source.raw} normalized={source.normalized} committed={source.committed} />
      </div>
      <p className="mt-3 text-xs text-muted-foreground">
        {formatCount(source.files)} {source.files === 1 ? "file" : "files"} · {formatRange(source.first_at, source.last_at)}
      </p>
    </AppLink>
  );
}

export function SourcesView() {
  const [format, setFormat] = useState<(typeof FORMATS)[number]>("All");
  const summary = useQuery({ queryKey: ["m-summary"], queryFn: ({ signal }) => importedApi.summary(signal), staleTime: 15_000 });
  const query = useInfiniteQuery({
    queryKey: ["m-sources", format],
    queryFn: ({ pageParam, signal }) => importedApi.sources(pageParam, format === "All" ? undefined : format, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const totals = summary.data?.totals;
  const byStatus = (summary.data?.files_by_status ?? {}) as Partial<Record<SourceStatus, number>>;
  return (
    <div>
      <PageBar title="Imported" subtitle="What the machine put into context, newest first" />
      {totals ? (
        <section className="space-y-3 p-4 pb-2">
          <Counts raw={totals.raw} normalized={totals.normalized} committed={totals.committed} />
          <div className="flex flex-wrap gap-2">
            {(Object.entries(byStatus) as [SourceStatus, number][]).map(([status, count]) => (
              <Chip key={status}>{formatCount(count)} {statusLabel(status).toLowerCase()}</Chip>
            ))}
          </div>
          <p className="text-xs text-muted-foreground">
            {formatCount(totals.sources)} sources split into {formatCount(totals.files)} files · {formatCount(totals.messages)} messages · {formatCount(totals.calls)} calls
            {summary.data && !summary.data.run_state_available ? " · run status still loading" : ""}
          </p>
        </section>
      ) : null}
      <div className="flex gap-2 overflow-x-auto px-4 py-2" role="tablist" aria-label="Format">
        {FORMATS.map((name) => (
          <button
            key={name}
            type="button"
            role="tab"
            aria-selected={format === name}
            onClick={() => setFormat(name)}
            className={cn(
              "h-11 shrink-0 rounded-full border px-5 text-sm font-semibold active:opacity-80",
              format === name ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card",
            )}
          >
            {name}
          </button>
        ))}
      </div>
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <div className="space-y-3 p-4 pt-2">
          {items.length === 0 ? <Empty>Nothing imported in this format yet.</Empty> : items.map((source) => <SourceCard key={source.id} source={source} />)}
        </div>
      )}
      {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
    </div>
  );
}

function PartyChip({ party }: { party: ImportedThread["party"] }) {
  const text = party === "first_party" ? "First-party" : party === "third_party" ? "Third-party" : party === "mixed" ? "First + third party" : "Not sorted yet";
  const tone = party === "third_party"
    ? "bg-violet-100 text-violet-900 dark:bg-violet-950 dark:text-violet-200"
    : party === "first_party"
      ? "bg-sky-100 text-sky-900 dark:bg-sky-950 dark:text-sky-200"
      : "";
  return <Chip className={cn("shrink-0", tone)}>{text}</Chip>;
}

export function SourceThreadsView({ sourceId }: { sourceId: string }) {
  const query = useInfiniteQuery({
    queryKey: ["m-threads", sourceId],
    queryFn: ({ pageParam, signal }) => importedApi.threads(sourceId, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (last) => last.next_offset ?? undefined,
  });
  const source = query.data?.pages[0]?.source;
  const threads = query.data?.pages.flatMap((page) => page.items) ?? [];
  return (
    <div>
      <PageBar
        title={source?.file_name ?? "Source"}
        subtitle={source ? [source.format, source.device, source.owner].filter(Boolean).join(" · ") : undefined}
        back="/m"
      />
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          {source ? (
            <section className="space-y-3 p-4 pb-2">
              <div className="flex items-center justify-between gap-3">
                <StatusBadge status={source.status} />
                <span className="text-xs text-muted-foreground">{formatCount(source.files)} files</span>
              </div>
              <Counts raw={source.raw} normalized={source.normalized} committed={source.committed} />
              <div className="flex flex-wrap gap-2">
                {(Object.entries(source.status_counts) as [SourceStatus, number][]).map(([status, count]) => (
                  <Chip key={status}>{formatCount(count)} {statusLabel(status).toLowerCase()}</Chip>
                ))}
              </div>
              <p className="break-all text-[11px] text-muted-foreground">{source.casevault_key}</p>
            </section>
          ) : null}
          <ul className="space-y-3 p-4">
            {threads.length === 0 ? <Empty>No conversations with records in this source yet.</Empty> : threads.map((thread) => (
              <li key={thread.id}>
                <AppLink href={`/m/thread/${thread.id}`} className="block rounded-xl border border-border bg-card p-4 active:bg-muted">
                  <div className="flex items-start justify-between gap-3">
                    <p className="min-w-0 text-sm font-semibold leading-snug">{thread.title}</p>
                    <PartyChip party={thread.party} />
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">{thread.participants.map((p) => p.label).join(", ")}</p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    {formatCount(thread.messages || thread.records)} {thread.messages ? "messages" : "records"} · {formatRange(thread.first_at, thread.last_at)}
                  </p>
                </AppLink>
              </li>
            ))}
          </ul>
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
        </>
      )}
    </div>
  );
}

function Bubble({ message, focused }: { message: ImportedMessage; focused: boolean }) {
  return (
    <div id={`msg-${message.id}`} className={cn("flex flex-col", message.outgoing ? "items-end" : "items-start")}>
      <div
        className={cn(
          "max-w-[85%] rounded-2xl px-3.5 py-2.5 text-[15px] leading-snug",
          message.outgoing ? "rounded-br-md bg-primary text-primary-foreground" : "rounded-bl-md border border-border bg-card",
          focused && "ring-4 ring-amber-400",
        )}
      >
        {message.body
          ? <p className="whitespace-pre-wrap break-words">{message.body}</p>
          : <p className="italic opacity-70">No text (media or empty message)</p>}
        {message.attachments > 0 ? <p className="mt-1 text-xs opacity-80">{message.attachments} attachment{message.attachments === 1 ? "" : "s"}</p> : null}
      </div>
      <p className="mt-1 px-1 text-[11px] text-muted-foreground">
        {message.sender.label} · {formatTime(message.at)}
        {message.party ? ` · ${message.party === "first_party" ? "first-party" : "third-party"}` : ""}
      </p>
      {message.sender.number && !message.outgoing ? (
        <div className="mt-1 px-1">
          <WhoIsThis number={message.sender.number} entityId={message.sender.entity_id} context="a text message" />
        </div>
      ) : null}
    </div>
  );
}

export function ThreadView({ threadId }: { threadId: string }) {
  const params = useBrowserSearchParams();
  const focus = params.get("focus");
  const query = useInfiniteQuery({
    queryKey: ["m-messages", threadId, focus],
    queryFn: ({ pageParam, signal }) => importedApi.messages(threadId, { cursor: pageParam, around: pageParam ? null : focus }, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.older_cursor ?? undefined,
  });
  // Pages arrive newest first; each page is oldest-first inside, like a chat.
  const messages = useMemo(() => [...(query.data?.pages ?? [])].reverse().flatMap((page) => page.items), [query.data]);
  const head = query.data?.pages[0];
  const title = useMemo(() => {
    const others = [...new Set(messages.filter((m) => !m.sender.mine).map((m) => m.sender.label))];
    return others.length ? others.slice(0, 3).join(", ") : head?.conversation ?? "Conversation";
  }, [messages, head]);
  const bottomRef = useRef<HTMLDivElement>(null);
  const pageCount = query.data?.pages.length ?? 0;
  const scrolledFor = useRef<string | null>(null);

  // Open at the newest message, or at the search hit, once per thread view; older pages loaded
  // later must not move the view.
  useLayoutEffect(() => {
    const key = `${threadId}:${focus ?? ""}`;
    if (pageCount !== 1 || scrolledFor.current === key) return;
    if (focus) document.getElementById(`msg-${focus}`)?.scrollIntoView({ block: "center" });
    else bottomRef.current?.scrollIntoView({ block: "end" });
    scrolledFor.current = key;
  }, [pageCount, focus, threadId]);

  const rows = messages.map((message, index) => ({
    message,
    day: formatDate(message.at),
    header: index === 0 || formatDate(message.at) !== formatDate(messages[index - 1].at),
  }));
  return (
    <div>
      <PageBar
        title={title}
        subtitle={head ? [head.source.format, head.source.device, head.source.file_name].filter(Boolean).join(" · ") : undefined}
        back={head ? `/m/source/${head.source.id}` : "/m"}
      />
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <div className="space-y-2 px-3 py-3">
          {query.hasNextPage
            ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load earlier messages" />
            : <p className="py-2 text-center text-xs text-muted-foreground">Start of this conversation</p>}
          {rows.map(({ message, day, header }) => (
            <div key={message.id} className="space-y-2">
              {header ? <p className="pt-2 text-center text-xs font-semibold text-muted-foreground">{day}</p> : null}
              <Bubble message={message} focused={message.id === focus} />
            </div>
          ))}
          {focus && head?.newer_cursor ? (
            <div className="p-2 text-center">
              <AppLink href={`/m/thread/${threadId}`} className="inline-flex h-12 items-center rounded-lg border border-border bg-card px-5 text-sm font-semibold active:bg-muted">
                Jump to the latest messages
              </AppLink>
            </div>
          ) : null}
          <div ref={bottomRef} />
        </div>
      )}
    </div>
  );
}

export function CallsView() {
  const query = useInfiniteQuery({
    queryKey: ["m-calls"],
    queryFn: ({ pageParam, signal }) => importedApi.calls(pageParam, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });
  const summary = query.data?.pages[0]?.summary;
  const calls = query.data?.pages.flatMap((page) => page.items) ?? [];
  const cells: [string, number][] = summary
    ? [["Calls", summary.total], ["Missed", summary.missed], ["In", summary.incoming], ["Out", summary.outgoing]]
    : [];
  return (
    <div>
      <PageBar title="Calls" subtitle={query.data ? `Read from ${query.data.pages[0].read_from}` : undefined} />
      <AppLink href="/m/unknown" className="mx-4 mt-3 flex min-h-12 items-center justify-between rounded-lg border border-amber-500/60 bg-amber-100 px-4 text-sm font-semibold text-amber-950 active:opacity-80 dark:bg-amber-950 dark:text-amber-100">
        <span>Unnamed numbers, most frequent first</span>
        <span aria-hidden="true">&rsaquo;</span>
      </AppLink>
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <>
          {summary ? (
            <section className="p-4 pb-2">
              <dl className="grid grid-cols-4 gap-2 text-center">
                {cells.map(([label, value]) => (
                  <div key={label} className="rounded-md bg-muted/70 px-1 py-2">
                    <dd className="text-base font-semibold tabular-nums">{formatCount(value)}</dd>
                    <dt className="text-[11px] text-muted-foreground">{label}</dt>
                  </div>
                ))}
              </dl>
              <p className="mt-2 text-xs text-muted-foreground">{formatRange(summary.first_at, summary.last_at)}</p>
            </section>
          ) : null}
          <ul className="divide-y divide-border">
            {calls.length === 0 ? <Empty>No calls imported yet.</Empty> : calls.map((call) => (
              <li key={call.id} className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate text-[15px] font-semibold">{call.with.label}</p>
                  <p className={cn("text-xs", call.missed ? "font-semibold text-destructive" : "text-muted-foreground")}>
                    {call.missed ? "Missed" : call.direction === "incoming" ? "Incoming" : call.direction === "outgoing" ? "Outgoing" : "Call"}
                    {!call.missed && call.duration_s !== null ? ` · ${formatDuration(call.duration_s)}` : ""}
                    {call.device ? ` · ${call.device}` : ""}
                  </p>
                </div>
                <div className="flex shrink-0 flex-col items-end gap-1">
                  <p className="text-right text-xs text-muted-foreground">{formatDateTime(call.at)}</p>
                  {call.with.number ? <WhoIsThis number={call.with.number} entityId={call.with.entity_id} context="a call" /> : null}
                </div>
              </li>
            ))}
          </ul>
          {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} /> : null}
        </>
      )}
    </div>
  );
}
