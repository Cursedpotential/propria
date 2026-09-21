// Ported from modules/forks/sbv/frontend/src/components/MessageThread.jsx
// (the per-message <div> block inside the "Unified Message and Call View" render,
// including its sender-label-for-group-conversations rule and incoming/outgoing
// alignment/coloring)
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: Bootstrap utility classes (`d-flex`, `bg-primary text-white`, `card`, …)
// replaced with this app's own Tailwind + shadcn tokens; SBV's `message.type === 2`
// sent/received check replaced with this app's own participant-derived `direction`
// (see use-preview-messages.ts, "self" display name convention); the highlighted/
// deep-linked message ring and audio-full-width special case were carried over.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { AttachmentPreview } from "@/components/sbv/attachment-preview";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { MatterMode } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

interface MessageBubbleProps {
  row: PreviewMessageRow;
  previewHandle: string;
  mode: MatterMode;
  /** True when this preview has more than two distinct participants overall. */
  showSenderLabel: boolean;
  highlighted?: boolean;
  onOpenAttachment?: (attachmentId: string) => void;
}

export function MessageBubble({ row, previewHandle, mode, showSenderLabel, highlighted, onOpenAttachment }: MessageBubbleProps) {
  const { message, direction } = row;
  const isSent = direction === "out";
  const hasAudioAttachment = message.attachments.some((item) => item.media_type?.startsWith("audio/"));

  return (
    <div className={cn("flex", isSent ? "justify-end" : "justify-start")} data-testid="message-bubble" data-direction={direction}>
      <div className={hasAudioAttachment ? "w-[90%]" : "max-w-[70%]"}>
        {showSenderLabel && !isSent && (
          <div className="mb-1 ml-2 text-[11px] text-muted-foreground">{row.senderName}</div>
        )}
        <div
          className={cn(
            "rounded-md border px-3 py-1.5 shadow-sm",
            isSent ? "border-primary/40 bg-primary text-primary-foreground" : "bg-card",
            highlighted && "ring-2 ring-offset-1 ring-[#c69027]",
          )}
        >
          {message.body && (
            <div className="whitespace-pre-wrap break-words text-sm leading-5">{message.body}</div>
          )}
          {message.attachments.length > 0 && (
            <div className="mt-1.5 space-y-1.5">
              {message.attachments.map((attachment) => (
                <AttachmentPreview
                  key={attachment.attachment_id}
                  attachment={attachment}
                  previewHandle={previewHandle}
                  mode={mode}
                  onOpen={onOpenAttachment ? () => onOpenAttachment(attachment.attachment_id) : undefined}
                />
              ))}
            </div>
          )}
          <div
            className={cn(
              "mt-1 flex items-center gap-1 text-[11px]",
              isSent ? "text-primary-foreground/70" : "text-muted-foreground",
            )}
          >
            <time dateTime={message.sent_at ?? undefined}>{row.timeLabel}</time>
            <span className="font-mono opacity-70">#{message.ordinal}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
