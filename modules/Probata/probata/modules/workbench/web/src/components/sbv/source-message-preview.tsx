"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { getSourceMessagePreview, type PreviewSelection } from "@/lib/source-message-preview-client";

/** Render one bounded original-file message page using the SBV donor bubble layout.
 * Input: HEAD-inspected original source. Output: readable messages and page controls.
 * Effects: Go read-preview calls only, cancelled on selection change; no XML parsing,
 * extraction, indexing, case admission or attachment downloads. Pick before Process.
 */
export function SourceMessagePreview({ selection }: { selection: PreviewSelection }) {
  const [cursors, setCursors] = useState<(string | null)[]>([null]);
  const [pageIndex, setPageIndex] = useState(0);
  const [firstPage, setFirstPage] = useState(1);
  const cursor = cursors[pageIndex] ?? null;
  const query = useQuery({
    queryKey: ["source-message-preview", selection.source_ref, selection.etag, cursor],
    queryFn: ({ signal }) => getSourceMessagePreview(selection, cursor, signal),
    gcTime: 0, retry: false, staleTime: Infinity,
  });
  return <section className="space-y-3" aria-label="Original message preview">
    <div className="flex items-center justify-between gap-2">
      <h2 className="text-sm font-semibold">Messages</h2><span className="text-xs text-muted-foreground">Original file preview</span>
    </div>
    <nav className="flex items-center justify-between gap-2" aria-label="Preview pages">
      <Button size="sm" variant="outline" disabled={pageIndex === 0 || query.isFetching} onClick={() => setPageIndex(pageIndex - 1)}>Previous messages</Button>
      <span className="text-xs" role="status">Page {firstPage + pageIndex}</span>
      <Button size="sm" variant="outline" disabled={!query.data?.next_cursor || query.isFetching} onClick={() => {
        const next = query.data?.next_cursor;
        if (next) {
          const history = [...cursors.slice(0, pageIndex + 1), next];
          const dropped = Math.max(0, history.length - 64);
          setCursors(history.slice(dropped)); setFirstPage(firstPage + dropped); setPageIndex(history.length - 1 - dropped);
        }
      }}>Next messages</Button>
    </nav>
    {query.isPending ? <p role="status" className="text-sm text-muted-foreground">Loading a small page of messages…</p> : query.isError ?
      <div><p role="alert" className="text-sm text-destructive">{query.error.message}</p><Button size="sm" variant="outline" onClick={() => void query.refetch()}>Retry preview</Button></div> :
      <div className="max-h-[65dvh] space-y-3 overflow-auto" aria-label="Message preview page">
        {query.data.messages.map((message) => {
          const outgoing = ["outgoing", "sent", "out"].includes(message.direction);
          const date = typeof message.date_ms === "number" ? new Date(message.date_ms) : null;
          return <article key={message.id} className={`flex ${outgoing ? "justify-end" : "justify-start"}`}>
            <div className="max-w-[90%] space-y-1">
              <p className="break-words text-xs text-muted-foreground">{outgoing ? "You" : message.participants.join(" · ") || "Unknown participant"}</p>
              <div className={`space-y-2 rounded-2xl px-3 py-2 text-sm ${outgoing ? "bg-primary text-primary-foreground" : "border bg-card"}`}>
                <p className="whitespace-pre-wrap break-words">{message.body}</p>
                {message.body_truncated && <p className="text-xs">Message shortened for this preview.</p>}
                {message.metadata_truncated && <p className="text-xs">Some details shortened for this preview.</p>}
                {message.attachments.map((attachment, index) => <p key={index} className="break-words text-xs">Attachment: {attachment.name || attachment.content_type || "unnamed"} · available after processing</p>)}
              </div>
              <p className="text-[10px] text-muted-foreground">{date && !Number.isNaN(date.getTime()) ? date.toLocaleString() : "Time unknown"}</p>
            </div>
          </article>;
        })}
        {!query.data.messages.length && <p className="text-sm text-muted-foreground">No messages in this section of the file.</p>}
        {query.data.truncation_reason && <p role="status" className="text-xs text-muted-foreground">This page reached its preview limit. Continue with Next messages when available.</p>}
        {!query.data.next_cursor && <p className="text-xs text-muted-foreground">End of preview.</p>}
      </div>}
  </section>;
}
