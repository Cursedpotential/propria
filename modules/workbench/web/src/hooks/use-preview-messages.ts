// Byline: Claude Code · Opus 5 · 2026-09-20 (cursor-paged message rows for the Review message browser)
"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { getProfferPreviewMessages } from "@/lib/api-client";
import type {
  MatterMode,
  ProfferPreviewMessage,
  ProfferPreviewParticipant,
} from "@/lib/shared/types";

/** Page size for one cursor request against the governed messages endpoint. */
export const MESSAGE_PAGE_SIZE = 200;

/** Self-authored messages carry the reserved `self` display name from the parser. */
const SELF_DISPLAY_NAME = "self";

export type MessageDirection = "in" | "out";

/** One flattened grid row. The underlying record stays attached, never copied apart. */
export interface PreviewMessageRow {
  message: ProfferPreviewMessage;
  senderName: string;
  senderAddress: string | null;
  direction: MessageDirection;
  timeLabel: string;
  bodyLine: string;
  attachmentCount: number;
}

function singleLine(body: string) {
  return body.replace(/\s+/gu, " ").trim();
}

function timeLabel(sentAt: string | null | undefined) {
  if (!sentAt) return "Timestamp unavailable";
  const parsed = new Date(sentAt);
  return Number.isNaN(parsed.getTime()) ? sentAt : parsed.toLocaleString();
}

export function useProfferPreviewMessages(previewHandle: string, mode: MatterMode, enabled = true) {
  return useInfiniteQuery({
    queryKey: ["proffer", "preview-messages", mode, previewHandle] as const,
    enabled: Boolean(previewHandle) && enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      getProfferPreviewMessages(previewHandle, mode, pageParam, MESSAGE_PAGE_SIZE, signal),
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
  });
}

/**
 * Collapses every loaded cursor page into ordinal-sorted rows plus the merged
 * participant directory. Duplicate message identities across pages resolve to
 * the newest projection rather than repeating a row.
 */
export function usePreviewMessageRows(
  pages: Array<{ messages: ProfferPreviewMessage[]; participants: ProfferPreviewParticipant[] }> | undefined,
) {
  return useMemo(() => {
    const participants = new Map<string, ProfferPreviewParticipant>();
    const messages = new Map<string, ProfferPreviewMessage>();
    (pages ?? []).forEach((page) => {
      page.participants.forEach((participant) => participants.set(participant.participant_id, participant));
      page.messages.forEach((message) => messages.set(message.message_id, message));
    });

    const rows: PreviewMessageRow[] = [...messages.values()]
      .sort((left, right) => left.ordinal - right.ordinal)
      .map((message) => {
        const participant = message.sender_participant_id
          ? participants.get(message.sender_participant_id)
          : undefined;
        const senderName = participant?.display_name ?? "Unattributed sender";
        return {
          message,
          senderName,
          senderAddress: participant?.canonical_address ?? null,
          direction: senderName === SELF_DISPLAY_NAME ? "out" : "in",
          timeLabel: timeLabel(message.sent_at),
          bodyLine: singleLine(message.body) || "(no message body)",
          attachmentCount: message.attachments.length,
        };
      });

    return { rows, participants };
  }, [pages]);
}

/** Client-side narrowing over the rows already loaded in this browser session. */
export function filterLoadedRows(
  rows: PreviewMessageRow[],
  query: string,
  attachmentsOnly: boolean,
) {
  const needle = query.trim().toLowerCase();
  if (!needle && !attachmentsOnly) return rows;
  return rows.filter((row) => {
    if (attachmentsOnly && row.attachmentCount === 0) return false;
    if (!needle) return true;
    const attachmentNames = row.message.attachments.map((item) => item.filename ?? "").join(" ");
    return `${row.bodyLine} ${row.senderName} ${row.senderAddress ?? ""} ${attachmentNames}`
      .toLowerCase()
      .includes(needle);
  });
}
