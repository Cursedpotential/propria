// Byline: Claude Code · Sonnet · 2026-10-02
// "Where does this number appear?" - the source the owner needs to see before he can say whether a pending
// number is who it looks like. Every imported message and call that carries the number, newest first, drawn
// with the existing SBV-derived MessageBubble and CallsTable, each under the file (casevault key) and
// conversation it came from, with a tap target that opens that conversation at that message. Read-only.
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { formatCount, prettyNumber } from "@/components/mobile/mobile-format";
import { toCallRow, toMessageRow } from "@/components/imported/record-rows";

import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { CallsTable } from "@/components/sbv/calls-table";
import { MessageBubble } from "@/components/sbv/message-bubble";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { importedApi, type NumberRecord } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";

function SourceCaption({ record }: { record: NumberRecord }) {
  const { source } = record;
  return (
    <div className="mb-1 rounded-md bg-muted/60 px-3 py-2 text-xs">
      <p className="break-all font-semibold">{source.file_name}</p>
      <p className="break-all text-muted-foreground">{source.casevault_key}</p>
      <p className="mt-1 flex flex-wrap items-center gap-1.5 text-muted-foreground">
        <Badge variant="outline" className="text-[11px]">{source.format}</Badge>
        {source.device ? <span>phone {source.device}</span> : null}
        {source.owner ? <span>· {source.owner}</span> : null}
        <span>· conversation {source.conversation}</span>
      </p>
    </div>
  );
}

export function NumberSourceSheet({ open, onOpenChange, number, onNavigate }: {
  open: boolean; onOpenChange: (open: boolean) => void; number: string; onNavigate?: () => void;
}) {
  const query = useInfiniteQuery({
    queryKey: ["number-records", number],
    queryFn: ({ pageParam, signal }) => importedApi.numberRecords(number, pageParam, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
    enabled: open,
  });
  const items = useMemo(() => query.data?.pages.flatMap((page) => page.items) ?? [], [query.data]);
  const counts = query.data?.pages[0]?.counts;
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="mx-auto max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-2xl pb-[env(safe-area-inset-bottom)]">
        <SheetHeader>
          <SheetTitle className="text-lg">Where {prettyNumber(number)} appears</SheetTitle>
          <SheetDescription>
            {counts ? `${formatCount(counts.messages)} messages · ${formatCount(counts.calls)} calls, newest first.` : "Every imported message and call that carries this number."}
          </SheetDescription>
        </SheetHeader>
        {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
          <div className="space-y-4 px-4 pb-6">
            {items.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">No imported record carries this number.</p> : null}
            {items.map((record, index) => (
              <div key={record.id} data-testid="number-record">
                <SourceCaption record={record} />
                {record.type === "message" ? (
                  <>
                    <MessageBubble row={toMessageRow(record.message, index)} previewHandle="" mode="REAL" showSenderLabel />
                    <Button asChild variant="outline" size="sm" className="mt-1 h-10">
                      <AppLink href={`/m/thread/${record.source.thread_id}?focus=${record.id}`} onClick={() => onNavigate?.()}>Open this conversation at this message</AppLink>
                    </Button>
                  </>
                ) : (
                  <CallsTable rows={[toCallRow({ id: record.id, at: record.at, device: record.source.device, ...record.call }, index)]} />
                )}
              </div>
            ))}
            {query.hasNextPage ? (
              <Button type="button" variant="outline" onClick={() => void query.fetchNextPage()} disabled={query.isFetchingNextPage} className="h-12 w-full">
                {query.isFetchingNextPage ? "Loading" : "Load more"}
              </Button>
            ) : null}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
