// Byline: Codex · GPT-6 · 2026-10-06
import { useInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { ConversationToolbar } from "@/components/conversations/conversation-actions";
import { toMessageRow } from "@/components/imported/record-rows";
import { formatDate } from "@/components/mobile/mobile-format";
import { Empty, ErrorBox, LoadMore, Loading } from "@/components/mobile/mobile-ui";
import { ReadContext } from "@/components/read/read-context";
import { readHref } from "@/components/read/read-location";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Button } from "@/components/ui/button";
import { importedApi } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";

/** Read a paged conversation with source identity, message permalinks and extracted context.
 * Inputs: original thread/message IDs and route context. Output: readable messages and details.
 * Effects: GETs, initial focus scrolling, and explicitly opened existing conversation tools.
 * Pick for imported records; preview attempts stay in ProfferPreviewClient.
 */
export function ReadConversation({ threadId, around, params }: { threadId: string; around: string | null; params: URLSearchParams }) {
  const query = useInfiniteQuery({
    queryKey: ["m-messages", threadId, around],
    queryFn: ({ pageParam, signal }) => importedApi.messages(threadId, { cursor: pageParam, around: pageParam ? null : around }, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.older_cursor ?? undefined,
  });
  const messages = [...(query.data?.pages ?? [])].reverse().flatMap((page) => page.items);
  const head = query.data?.pages[0];
  const focused = useRef(false);
  const pageCount = query.data?.pages.length ?? 0;
  const focusLoaded = messages.some((message) => message.id === around);
  useEffect(() => {
    if (!around || !focusLoaded || focused.current) return;
    document.getElementById(`read-message-${around}`)?.scrollIntoView({ block: "center" });
    focused.current = true;
  }, [around, focusLoaded, pageCount]);
  if (query.isPending) return <Loading />;
  if (!head) return <ErrorBox error={query.error ?? new Error("Conversation unavailable")} onRetry={() => void query.refetch()} />;
  // The messages response owns source identity, including when search supplied only a thread.
  const readingParams = new URLSearchParams(params);
  readingParams.set("source", head.source.id);
  readingParams.set("thread", threadId);
  return (
    <section className="grid min-w-0 items-start gap-4 2xl:grid-cols-[minmax(0,1fr)_22rem]" aria-label="Conversation reader">
      <div className="min-w-0 rounded-lg border border-border bg-card">
        <header className="space-y-2 border-b border-border p-4">
          <h2 className="break-words text-lg font-semibold">{head.conversation}</h2>
          <p className="break-words text-sm text-muted-foreground">{[head.source.file_name, head.source.format, head.source.device, head.source.owner_name].filter(Boolean).join(" · ")}</p>
          <AppLink href={readHref(readingParams, { around: null })} className="inline-block text-sm underline underline-offset-4">Browse this source’s conversations</AppLink>
          {params.get("source") && params.get("source") !== head.source.id ? <p className="text-sm text-amber-700 dark:text-amber-300">This thread belongs to the source shown here. The source selection in the URL differs; use the link above to align it.</p> : null}
          <details className="text-sm">
            <summary className="cursor-pointer font-medium">Source and conversation details</summary>
            <dl className="mt-2 space-y-1 break-all text-xs text-muted-foreground">
              <dt className="font-semibold">Source ID</dt><dd>{head.source.id}</dd>
              <dt className="font-semibold">Thread ID</dt><dd>{threadId}</dd>
              <dt className="font-semibold">Source owner</dt><dd>{head.source.owner ?? "Not supplied"}</dd>
            </dl>
          </details>
          <details key={threadId} className="text-sm">
            <summary className="cursor-pointer font-medium">Conversation tools</summary>
            <ConversationToolbar threadId={threadId} placement="desktop" />
            <AppLink href={`/conversations?${new URLSearchParams({ source: head.source.id, thread: threadId }).toString()}`} className="mt-3 inline-block underline underline-offset-4">Open bulk conversation tools</AppLink>
          </details>
        </header>
        {query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : null}
        {around && !focusLoaded ? <p className="p-4 text-sm" role="status">The requested message is not in this loaded window. Load earlier messages or jump to the latest messages.</p> : null}
        {query.hasNextPage ? <LoadMore onClick={() => void query.fetchNextPage()} loading={query.isFetchingNextPage} label="Load earlier messages" />
          : <p className="p-3 text-center text-xs text-muted-foreground">Start of this conversation</p>}
        {!messages.length ? <Empty>No message content is available for this conversation.</Empty> : null}
        <div className="space-y-3 p-3" aria-label="Messages">
          {messages.map((message, index) => (
            <article key={message.id} id={`read-message-${message.id}`} aria-label={`Message from ${message.sender.label}`}>
              {index === 0 || formatDate(message.at) !== formatDate(messages[index - 1].at) ? <p className="py-2 text-center text-xs text-muted-foreground">{formatDate(message.at)}</p> : null}
              <MessageBubble row={toMessageRow(message, index)} previewHandle="" mode="LIVE" showSenderLabel={!message.outgoing} highlighted={message.id === around} />
              <details className="mt-1 px-2 text-xs text-muted-foreground">
                <summary className="cursor-pointer">Citation and message details</summary>
                <dl className="mt-2 space-y-1 break-all">
                  <dt className="font-semibold">Original record ID</dt><dd>{message.id}</dd>
                  <dt className="font-semibold">Source file</dt><dd>{head.source.file_name}</dd>
                  <dt className="font-semibold">Party</dt><dd>{message.party?.replaceAll("_", " ") ?? "Unclassified"}</dd>
                  <dt className="font-semibold">Certainty</dt><dd>{message.certainty ?? "Not supplied"}</dd>
                  <dt className="font-semibold">Attachments</dt><dd>{message.attachments}{message.attachments ? " (content is not supplied by this messages API)" : ""}</dd>
                </dl>
                <AppLink href={readHref(readingParams, { around: message.id })} className="mt-2 inline-block underline underline-offset-4">Link to this message</AppLink>
              </details>
            </article>
          ))}
        </div>
        {around ? <div className="p-4"><Button asChild variant="outline"><AppLink href={readHref(readingParams, { around: null })}>Jump to latest messages</AppLink></Button></div> : null}
      </div>
      <ReadContext threadId={threadId} params={readingParams} />
    </section>
  );
}
