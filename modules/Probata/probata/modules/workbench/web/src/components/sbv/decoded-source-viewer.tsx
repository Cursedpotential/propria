// Byline: Claude Code · Fable 5.1 · 2026-09-21
// Reading surface for existing processed backup messages. Layout and behavior
// follow SBV's viewer (ConversationList +
// MessageThread + LazyMedia; MIT, Copyright (c) 2025 lowcarbdev): conversations
// on the left, the thread on the right as chat bubbles, pictures inline. SBV's
// viewer consumes processed message/media outputs; it never imports, extracts
// or indexes the source. Oldest first, pages forward (owner, 2026-09-20).
"use client";

import { useQuery } from "@tanstack/react-query";
import { Loader2, Paperclip, Search, Users } from "lucide-react";
import { useMemo, useState } from "react";

import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  type DecodedAttachment,
  type DecodedMessage,
  DecodedSourceError,
  type DecodedThread,
  getDecodedManifest,
  getDecodedMediaUrl,
  getDecodedThreadPage,
} from "@/lib/decoded-source-client";

const SELF = "self";

function party(value: string): string {
  const digits = value.replace(/\D/g, "").replace(/^1(?=\d{10}$)/, "");
  return /^\d{10}$/.test(digits) ? `(${digits.slice(0, 3)}) ${digits.slice(3, 6)}-${digits.slice(6)}` : value;
}

function others(participants: string[]): string[] {
  return [...new Set(participants.filter((value) => value && value !== SELF))];
}

function when(value: string | null, withDate = true): string {
  if (!value) return "time unknown";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(
    undefined,
    withDate
      ? { year: "numeric", month: "short", day: "numeric", hour: "numeric", minute: "2-digit" }
      : { hour: "numeric", minute: "2-digit" },
  );
}

function dayKey(value: string | null): string {
  const date = value ? new Date(value) : null;
  return date && !Number.isNaN(date.getTime()) ? date.toDateString() : "";
}

function Attachment({ sourceRef, attachment }: { sourceRef: string; attachment: DecodedAttachment }) {
  const [broken, setBroken] = useState(false);
  const url = getDecodedMediaUrl(sourceRef, attachment.sha256);
  const type = attachment.media_type ?? "";
  if (url && !broken && type.startsWith("image/")) {
    return (
      <a href={url} target="_blank" rel="noreferrer" className="block overflow-hidden rounded-md border">
        <img src={url} alt={attachment.name ?? "Picture"} loading="lazy" className="max-h-64 w-auto object-contain" onError={() => setBroken(true)} />
      </a>
    );
  }
  if (url && !broken && type.startsWith("video/")) {
    return <video src={url} controls playsInline preload="metadata" className="max-h-64 w-full rounded-md border" onError={() => setBroken(true)} />;
  }
  if (url && !broken && type.startsWith("audio/")) {
    return <audio src={url} controls className="w-full" onError={() => setBroken(true)} />;
  }
  return (
    <a
      href={url ?? undefined}
      target="_blank"
      rel="noreferrer"
      className="flex items-center gap-2 rounded-md border bg-background/60 px-2 py-1 text-xs text-foreground"
    >
      <Paperclip className="size-3.5 shrink-0" aria-hidden="true" />
      <span className="truncate">{attachment.name ?? type ?? "Attachment"}</span>
    </a>
  );
}

function Bubble({ sourceRef, message, group }: { sourceRef: string; message: DecodedMessage; group: boolean }) {
  const outgoing = message.sender === SELF;
  return (
    <article className={`flex ${outgoing ? "justify-end" : "justify-start"}`} aria-label={`${outgoing ? "You" : party(message.sender ?? "Unknown")}, ${when(message.occurred_at)}`}>
      <div className="max-w-[min(68ch,72%)] space-y-1">
        {group && !outgoing && <p className="px-1 text-[11px] font-medium text-muted-foreground">{party(message.sender ?? "Unknown")}</p>}
        <div
          className={`space-y-2 rounded-2xl px-3 py-2 text-sm ${outgoing ? "rounded-br-sm bg-primary text-primary-foreground" : "rounded-bl-sm border bg-card"}`}
          title={when(message.occurred_at)}
        >
          {message.body && <p className="whitespace-pre-wrap break-words">{message.body}</p>}
          {message.attachments.map((attachment) => (
            <Attachment key={attachment.ordinal} sourceRef={sourceRef} attachment={attachment} />
          ))}
          {message.missing_attachments > 0 && (
            <p className={`text-[11px] ${outgoing ? "text-primary-foreground/80" : "text-muted-foreground"}`}>
              {message.missing_attachments} attachment{message.missing_attachments === 1 ? "" : "s"} named in the backup with no file
            </p>
          )}
        </div>
        <p className={`px-1 text-[10px] text-muted-foreground ${outgoing ? "text-right" : ""}`}>{when(message.occurred_at, false)}</p>
      </div>
    </article>
  );
}

function Thread({ sourceRef, thread }: { sourceRef: string; thread: DecodedThread }) {
  const [offset, setOffset] = useState(0);
  const query = useQuery({
    queryKey: ["decoded-thread-page", sourceRef, thread.file, offset],
    queryFn: ({ signal }) => getDecodedThreadPage(sourceRef, thread.file, offset, signal),
    gcTime: 0,
    retry: false,
  });
  const messages = query.data?.messages ?? [];
  const group = others(thread.participants).length > 1;

  if (query.isPending) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> Loading conversation…
      </div>
    );
  }
  if (query.isError) {
    return <div className="space-y-2 p-4"><p className="text-sm text-destructive" role="alert">{query.error.message}</p>
      <Button variant="outline" size="sm" onClick={() => void query.refetch()}>Retry messages</Button></div>;
  }
  return (
    <div className="flex h-full min-h-0 flex-col">
    <nav className="flex items-center justify-between gap-2 border-b px-4 py-2" aria-label="Message pages">
      <Button size="sm" variant="outline" disabled={offset === 0 || query.isFetching} onClick={() => setOffset(Math.max(0, offset - 200))}>Previous messages</Button>
      <span className="text-xs text-muted-foreground" role="status">{messages.length ? `${offset + 1}–${offset + messages.length} of ${query.data!.total_records.toLocaleString()}` : "No messages on this page"}</span>
      <Button size="sm" variant="outline" disabled={typeof query.data?.next_offset !== "number" || query.isFetching} onClick={() => { if (typeof query.data?.next_offset === "number") setOffset(query.data.next_offset); }}>Next messages</Button>
    </nav>
    <div key={offset} className="min-h-0 flex-1 space-y-2 overflow-y-auto px-4 py-3" role="log" aria-label={`Conversation with ${others(thread.participants).map(party).join(", ")}`}>
      {messages.map((message, index) => {
        const newDay = dayKey(message.occurred_at) !== dayKey(messages[index - 1]?.occurred_at ?? null);
        return (
          <div key={message.ordinal} className="space-y-2">
            {newDay && message.occurred_at && (
              <p role="separator" className="platform-kicker py-1 text-center">
                {new Date(message.occurred_at).toLocaleDateString(undefined, { weekday: "short", year: "numeric", month: "short", day: "numeric" })}
              </p>
            )}
            <Bubble sourceRef={sourceRef} message={message} group={group} />
          </div>
        );
      })}
      {typeof query.data?.next_offset !== "number" && <p className="py-2 text-center text-[11px] text-muted-foreground">End of this conversation file</p>}
    </div>
    </div>
  );
}

/** Read existing processed message pages with the SBV donor conversation layout.
 * Input: exact original source reference. Output: conversations, messages and media.
 * Effects: bounded read API calls only; never starts extraction, indexing or promotion.
 * Pick for a readable backup at Sources or the full-size Read entry point.
 */
export function DecodedSourceViewer({ sourceRef }: { sourceRef: string }) {
  const [selected, setSelected] = useState<string | null>(null);
  const [needle, setNeedle] = useState("");
  const manifest = useQuery({
    queryKey: ["decoded-manifest", sourceRef],
    queryFn: ({ signal }) => getDecodedManifest(sourceRef, signal),
    retry: false,
    staleTime: 60_000,
  });

  const threads = useMemo(() => {
    const all = [...(manifest.data?.threads ?? [])].sort((a, b) => b.records - a.records);
    const digits = needle.replace(/\D/g, "");
    if (!needle.trim()) return all;
    return all.filter((thread) => (digits ? thread.thread.includes(digits) : thread.thread.toLowerCase().includes(needle.trim().toLowerCase())));
  }, [manifest.data, needle]);
  const active = threads.find((thread) => thread.file === selected) ?? threads[0] ?? null;

  if (manifest.isPending) {
    return (
      <p className="flex items-center gap-2 px-4 py-3 text-xs text-muted-foreground">
        <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" /> Checking for decoded messages…
      </p>
    );
  }
  if (manifest.isError) {
    const notDecoded = manifest.error instanceof DecodedSourceError && manifest.error.status === 404;
    return <div className="space-y-2 px-4 py-3">
      <p className="text-xs text-muted-foreground" role={notDecoded ? "status" : "alert"}>{notDecoded ? "Processed messages are not available yet." : manifest.error.message}</p>
      <Button size="sm" variant="outline" disabled={manifest.isFetching} onClick={() => void manifest.refetch()}>Check again</Button>
    </div>;
  }

  const data = manifest.data;
  return (
    <section className="platform-panel overflow-hidden" aria-label="Decoded messages">
      <header className="flex flex-wrap items-center gap-x-4 gap-y-1 border-b px-4 py-2 text-xs text-muted-foreground">
        <strong className="text-sm font-semibold text-foreground">Decoded messages</strong>
        <span className="tabular-nums">{data.records.toLocaleString()} messages</span>
        <span className="tabular-nums">{data.threads.length.toLocaleString()} conversation files</span>
        <span className="tabular-nums">{data.media_objects.toLocaleString()} pictures and files</span>
        {data.rejected > 0 && <span className="tabular-nums text-destructive">{data.rejected.toLocaleString()} unreadable records</span>}
        {data.decoded_at && <span className="ml-auto">decoded {when(data.decoded_at)}</span>}
      </header>
      <div className="grid h-[70dvh] min-h-96 grid-cols-[minmax(9rem,20rem)_minmax(0,1fr)]">
        <aside className="flex min-h-0 flex-col border-r">
          <label className="relative m-2">
            <span className="sr-only">Find a conversation by number</span>
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input value={needle} onChange={(event) => setNeedle(event.target.value)} placeholder="Find a number" className="h-8 pl-7 text-xs" />
          </label>
          <ul className="min-h-0 flex-1 divide-y overflow-y-auto" aria-label="Conversations">
            {threads.map((thread) => {
              const people = others(thread.participants);
              const isActive = active?.file === thread.file;
              return (
                <li key={thread.file}>
                  <button
                    type="button"
                    aria-pressed={isActive}
                    onClick={() => setSelected(thread.file)}
                    className={`grid w-full grid-cols-[minmax(0,1fr)_auto] gap-x-2 px-3 py-1.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring ${isActive ? "bg-accent" : "hover:bg-accent/40"}`}
                  >
                    <span className="flex min-w-0 items-center gap-1.5 truncate text-sm font-medium">
                      {people.length > 1 && <Users className="size-3.5 shrink-0 text-muted-foreground" aria-label="Group" />}
                      <span className="truncate">{people.map(party).join(" · ") || "Unknown"}</span>
                    </span>
                    <span className="text-xs tabular-nums text-muted-foreground">{thread.records.toLocaleString()}</span>
                    <span className="col-span-2 truncate text-[11px] text-muted-foreground">
                      {when(thread.first_occurred_at)} → {when(thread.last_occurred_at)}
                      {thread.chunk > 1 ? ` · part ${thread.chunk}` : ""}
                    </span>
                  </button>
                </li>
              );
            })}
            {threads.length === 0 && <li className="px-3 py-4 text-xs text-muted-foreground">No conversation matches that number.</li>}
          </ul>
        </aside>
        <div className="min-h-0">{active && <Thread key={active.file} sourceRef={sourceRef} thread={active} />}</div>
      </div>
    </section>
  );
}
