// Byline: Claude Code · Opus 5 · 2026-09-20 (selection-driven message detail; no per-row detail button)
// Byline: Claude Code · Opus 5.5 · 2026-09-26 ("Mark as event worth recalling" on the selected message; owner 2026-09-25)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-03 (long bodies fold; an attachment-only message shows its attachment)
"use client";

import { MessageSquareText } from "lucide-react";

import { MarkEventButton } from "@/components/entities/mark-event-button";
import { AttachmentPreview } from "@/components/sbv/attachment-preview";
import { CollapsibleText, DETAIL_TEXT_LIMIT } from "@/components/sbv/collapsible-text";
import { Badge } from "@/components/ui/badge";
import { ContextReviewPanel } from "@/components/review/context-review";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPreviewParticipant } from "@/lib/shared/types";

interface MessageDetailPanelProps {
  row: PreviewMessageRow | null;
  participants: Map<string, ProfferPreviewParticipant>;
  previewHandle: string;
  mode: MatterMode;
}

export function MessageDetailPanel({ row, participants, previewHandle, mode }: MessageDetailPanelProps) {
  if (!row) {
    return (
      <div className="grid h-full place-content-center px-6 text-center">
        <MessageSquareText className="mx-auto size-7 text-muted-foreground" aria-hidden="true" />
        <p className="mt-3 text-sm font-medium">Select a message row</p>
        <p className="mt-1 text-xs text-muted-foreground">Arrow keys move the selection; this panel follows it.</p>
      </div>
    );
  }

  const { message } = row;
  const otherParticipants = [...new Set(message.participant_ids)]
    .filter((id) => id !== message.sender_participant_id)
    .map((id) => participants.get(id)?.display_name ?? id);

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="message-detail-panel">
      <header className="border-b px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="text-sm font-semibold">{row.senderName}</h3>
          <Badge variant={row.direction === "out" ? "default" : "outline"}>{row.direction === "out" ? "Outbound" : "Inbound"}</Badge>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          <time dateTime={message.sent_at ?? undefined}>{row.timeLabel}</time>
          <span className="ml-2 font-mono">#{message.ordinal}</span>
        </p>
        {row.senderAddress && <p className="mt-1 break-all font-mono text-[11px] text-muted-foreground">{row.senderAddress}</p>}
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        {message.body ? (
          <CollapsibleText text={message.body} limit={DETAIL_TEXT_LIMIT} className="whitespace-pre-wrap break-words text-sm leading-6" />
        ) : (
          <p className="text-sm text-muted-foreground">{message.attachments.length ? "(no text — attachment only)" : "(no message body)"}</p>
        )}
        {message.attachments.length > 0 && (
          <div className="mt-3 space-y-1.5" data-testid="message-detail-attachments">
            {message.attachments.map((attachment) => (
              <AttachmentPreview key={attachment.attachment_id} attachment={attachment} previewHandle={previewHandle} mode={mode} />
            ))}
          </div>
        )}
        {/* Byline: Claude Code · Opus 5.5 · 2026-09-26 — context review for the selected message (owner 2026-09-25). */}
        <div className="-mx-4 mt-3">
          <ContextReviewPanel key={message.message_id} previewHandle={previewHandle} mode={mode} messageId={message.message_id} />
        </div>
      </div>

      <footer className="space-y-1 border-t px-4 py-3 font-mono text-[10px] text-muted-foreground">
        <div className="break-all">message {message.message_id}</div>
        <div className="break-all">
          other participants {otherParticipants.length ? otherParticipants.join(", ") : "none recorded"}
        </div>
        <MarkEventButton key={message.message_id} previewHandle={previewHandle} mode={mode} recordId={message.message_id} compact />
      </footer>
    </div>
  );
}
