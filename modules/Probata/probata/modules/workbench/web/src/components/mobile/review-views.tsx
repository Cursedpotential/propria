// Byline: Claude Code · Sonnet · 2026-10-02
// Review on a phone: the queue of previews waiting for the owner, and Approve / Reject on one of them.
// No new write path: the decision is the existing POST /api/proffer/previews/{handle}/decision
// (decideProffer), with the same gate the desktop Review page uses: the preview must be awaiting a
// decision, its records must carry provenance, and all six receipts must be completed.
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, CircleAlert, CircleDashed } from "lucide-react";
import { useState } from "react";

import { Empty, ErrorBox, LoadMore, Loading, PageBar } from "@/components/mobile/mobile-ui";
import { errorText, formatCount, formatDate, formatDateTime, formatTime } from "@/components/mobile/mobile-format";
import { decideProffer, getProfferPreview, getProfferPreviewContent, getProfferPreviewMessages } from "@/lib/api-client";
import { Badge } from "@/components/ui/badge";
import { importedApi } from "@/lib/imported-client";
import { PROFFER_CONTEXT_CHECKPOINTS } from "@/lib/proffer-context-checkpoints";
import { AppLink } from "@/lib/router-compat";
import type { ProfferPreviewMessage, ProfferPreviewParticipant } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

// Wire value of the live case on the existing Proffer routes; the screen only ever says "live".
const LIVE_MODE = "LIVE" as const;

export function ReviewQueueView() {
  const query = useQuery({ queryKey: ["m-review-queue"], queryFn: ({ signal }) => importedApi.reviewQueue(signal), staleTime: 10_000 });
  return (
    <div>
      <PageBar title="Review" subtitle="Imports waiting for your decision" />
      {query.isPending ? <Loading /> : query.isError ? <ErrorBox error={query.error} onRetry={() => void query.refetch()} /> : (
        <ul className="space-y-3 p-4">
          {query.data.items.length === 0 ? <Empty>Nothing is waiting for a decision.</Empty> : query.data.items.map((item) => (
            <li key={item.preview_handle}>
              <AppLink href={`/m/review/${item.preview_handle}`} className="block rounded-xl border border-border bg-card p-4 active:bg-muted">
                <div className="flex items-start justify-between gap-3">
                  <p className="min-w-0 text-sm font-semibold leading-snug">{item.title}</p>
                  <Badge variant="secondary" className="shrink-0 bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-200">Needs decision</Badge>
                </div>
                <p className="mt-1 break-all text-xs text-muted-foreground">{item.file_name}</p>
                <p className="mt-2 text-xs text-muted-foreground">
                  {[item.format, item.device, item.owner].filter(Boolean).join(" · ")} · {formatCount(item.records)} records · waiting since {formatDateTime(item.waiting_since)}
                </p>
              </AppLink>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function PreviewBubble({ message, participants }: { message: ProfferPreviewMessage; participants: Map<string, ProfferPreviewParticipant> }) {
  const sender = message.sender_participant_id ? participants.get(message.sender_participant_id) : undefined;
  return (
    <div className="rounded-2xl border border-border bg-card px-3.5 py-2.5">
      <p className="text-[11px] font-semibold text-muted-foreground">
        {sender?.display_name ?? "Unknown sender"} · {formatDate(message.sent_at)} {formatTime(message.sent_at)}
      </p>
      {message.body ? <p className="mt-1 whitespace-pre-wrap break-words text-[15px] leading-snug">{message.body}</p> : <p className="mt-1 text-sm italic text-muted-foreground">No text</p>}
      {message.attachments.length > 0 ? <p className="mt-1 text-xs text-muted-foreground">{message.attachments.length} attachment{message.attachments.length === 1 ? "" : "s"} noted</p> : null}
    </div>
  );
}

export function ReviewDetailView({ handle }: { handle: string }) {
  const client = useQueryClient();
  const [confirming, setConfirming] = useState<"approve" | "reject" | null>(null);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);

  const preview = useQuery({ queryKey: ["m-preview", handle], queryFn: ({ signal }) => getProfferPreview(handle, LIVE_MODE, signal) });
  const content = useQuery({
    queryKey: ["m-preview-content", handle],
    queryFn: ({ signal }) => getProfferPreviewContent(handle, LIVE_MODE, undefined, undefined, 100, signal),
  });
  const messages = useInfiniteQuery({
    queryKey: ["m-preview-messages", handle],
    queryFn: ({ pageParam, signal }) => getProfferPreviewMessages(handle, LIVE_MODE, pageParam ?? undefined, 40, signal),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });

  const data = preview.data;
  const records = content.data?.records ?? [];
  const receipts = data?.receipts ?? [];
  const checks = PROFFER_CONTEXT_CHECKPOINTS.map(({ type, label }) => ({
    label,
    ok: receipts.some((receipt) => receipt.receipt_type === type && receipt.status === "completed"),
  }));
  const provenance = records.length > 0 && records.every((record) => Boolean(record.source_locator_ref));
  const awaiting = data?.phase === "awaiting_decision";
  const eligible = Boolean(awaiting && provenance && checks.every((check) => check.ok) && !content.isError);
  const participants = new Map((messages.data?.pages[0]?.participants ?? []).map((participant) => [participant.participant_id, participant]));
  const shown = messages.data?.pages.flatMap((page) => page.messages) ?? [];
  const total = messages.data?.pages[0]?.total_messages;

  async function decide(approved: boolean) {
    if (!eligible || pending) return;
    if (!approved && !reason.trim()) return;
    setPending(true);
    setFailure(null);
    try {
      const result = await decideProffer(handle, LIVE_MODE, { approved, reason: approved ? "" : reason.trim() });
      if (result.preview_handle !== handle) throw new Error("The decision response did not match this preview");
      setDone(approved ? "Approved. Your decision is recorded." : "Rejected. Your decision and reason are recorded.");
      setConfirming(null);
      await Promise.all([
        client.invalidateQueries({ queryKey: ["m-review-queue"] }),
        client.invalidateQueries({ queryKey: ["m-preview", handle] }),
        client.invalidateQueries({ queryKey: ["m-sources"] }),
        client.invalidateQueries({ queryKey: ["m-summary"] }),
      ]);
    } catch (error) {
      setFailure(errorText(error));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className={confirming ? "pb-64" : "pb-24"}>
      <PageBar title="Review import" subtitle={data ? `Status: ${data.phase.replace(/_/g, " ")}` : undefined} back="/m/review" />
      {preview.isPending ? <Loading /> : preview.isError ? <ErrorBox error={preview.error} onRetry={() => void preview.refetch()} /> : (
        <div className="space-y-4 p-4">
          <section className="rounded-xl border border-border bg-card p-4">
            <p className="text-sm font-semibold">Did the machine read this correctly?</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Read the messages below, check the six steps, then approve or reject. Approving commits these records to the case.
            </p>
            <ul className="mt-3 space-y-2">
              {checks.map((check) => (
                <li key={check.label} className="flex items-center gap-2 text-sm">
                  {check.ok ? <Check className="size-5 text-emerald-600" /> : <CircleDashed className="size-5 text-muted-foreground" />}
                  <span className={check.ok ? "" : "text-muted-foreground"}>{check.label}</span>
                </li>
              ))}
              <li className="flex items-center gap-2 text-sm">
                {provenance ? <Check className="size-5 text-emerald-600" /> : <CircleAlert className="size-5 text-amber-600" />}
                <span className={provenance ? "" : "text-muted-foreground"}>
                  {content.isPending ? "Checking where each record came from" : provenance ? `Every record shows where it came from (${formatCount(records.length)} checked)` : "Provenance not confirmed"}
                </span>
              </li>
            </ul>
          </section>

          <section>
            <h2 className="mb-2 text-sm font-semibold">
              Messages {total !== undefined && total !== null && total >= 0 ? `(${formatCount(total)})` : ""}
            </h2>
            {messages.isPending ? <Loading /> : messages.isError ? <ErrorBox error={messages.error} onRetry={() => void messages.refetch()} /> : (
              <div className="space-y-2">
                {shown.length === 0 ? <Empty>No messages in this preview.</Empty> : shown.map((message) => <PreviewBubble key={message.message_id} message={message} participants={participants} />)}
                {messages.hasNextPage ? <LoadMore onClick={() => void messages.fetchNextPage()} loading={messages.isFetchingNextPage} /> : null}
              </div>
            )}
          </section>
        </div>
      )}

      {done ? (
        <div className="fixed inset-x-0 bottom-[calc(4rem+env(safe-area-inset-bottom))] z-20 border-t border-border bg-card p-4 shadow-lg">
          <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">{done}</p>
          <AppLink href="/m/review" className="mt-3 flex h-12 items-center justify-center rounded-lg border border-border font-semibold active:bg-muted">Back to the queue</AppLink>
        </div>
      ) : data ? (
        <div className="fixed inset-x-0 bottom-[calc(4rem+env(safe-area-inset-bottom))] z-20 border-t border-border bg-card p-3 shadow-lg">
          {confirming === "reject" ? (
            <div className="space-y-2">
              <label htmlFor="m-reject-reason" className="text-sm font-semibold">Why are you rejecting this?</label>
              <textarea
                id="m-reject-reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                rows={3}
                className="w-full rounded-lg border border-input bg-background p-3 text-base"
                placeholder="A reason is required"
              />
              <div className="grid grid-cols-2 gap-2">
                <button type="button" onClick={() => setConfirming(null)} className="h-12 rounded-lg border border-border font-semibold active:bg-muted">Cancel</button>
                <button
                  type="button"
                  disabled={pending || !reason.trim()}
                  onClick={() => void decide(false)}
                  className="h-12 rounded-lg bg-destructive font-semibold text-white active:opacity-80 disabled:opacity-50"
                >
                  {pending ? "Recording" : "Confirm reject"}
                </button>
              </div>
            </div>
          ) : confirming === "approve" ? (
            <div className="space-y-2">
              <p className="text-sm font-semibold">Approve this import and commit its records to the case?</p>
              <div className="grid grid-cols-2 gap-2">
                <button type="button" onClick={() => setConfirming(null)} className="h-12 rounded-lg border border-border font-semibold active:bg-muted">Cancel</button>
                <button
                  type="button"
                  disabled={pending}
                  onClick={() => void decide(true)}
                  className="h-12 rounded-lg bg-primary font-semibold text-primary-foreground active:opacity-80 disabled:opacity-50"
                >
                  {pending ? "Recording" : "Confirm approve"}
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              {!awaiting ? (
                <p className="text-sm text-muted-foreground">This import is not waiting for a decision ({data.phase.replace(/_/g, " ")}).</p>
              ) : !eligible ? (
                <p className="text-sm text-muted-foreground">The decision unlocks when every step above shows a check.</p>
              ) : null}
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  disabled={!eligible}
                  onClick={() => { setFailure(null); setConfirming("reject"); }}
                  className={cn("h-14 rounded-lg border border-destructive font-semibold text-destructive active:bg-destructive/10", !eligible && "opacity-40")}
                >
                  Reject
                </button>
                <button
                  type="button"
                  disabled={!eligible}
                  onClick={() => { setFailure(null); setConfirming("approve"); }}
                  className={cn("h-14 rounded-lg bg-primary font-semibold text-primary-foreground active:opacity-80", !eligible && "opacity-40")}
                >
                  Approve
                </button>
              </div>
            </div>
          )}
          {failure ? <p className="mt-2 text-sm font-semibold text-destructive" role="alert">{failure}</p> : null}
        </div>
      ) : null}
    </div>
  );
}
