// Byline: Claude Code · Opus 5 · 2026-09-20 (cursor-paged message rows for the Review message browser)
"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { getProfferPreviewMessages, type ProfferPreviewMessageFilters } from "@/lib/api-client";
import type {
  MatterMode,
  ProfferPreviewMessage,
  ProfferPreviewParticipant,
} from "@/lib/shared/types";

export type { ProfferPreviewMessageFilters };

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
  /** Attachments this source names but carries no bytes for. */
  missingPayloadCount: number;
}

function singleLine(body: string) {
  return body.replace(/\s+/gu, " ").trim();
}

function timeLabel(sentAt: string | null | undefined) {
  if (!sentAt) return "Timestamp unavailable";
  const parsed = new Date(sentAt);
  return Number.isNaN(parsed.getTime()) ? sentAt : parsed.toLocaleString();
}

const EMPTY_FILTERS: ProfferPreviewMessageFilters = {};

/**
 * Fetches message rows for one preview attempt. `filters` is part of the query key: a
 * cursor is bound to the filter that minted it, so changing any filter value restarts
 * paging from the first page rather than reusing a stale cursor (the server 422s on that).
 */
export function useProfferPreviewMessages(
  previewHandle: string,
  mode: MatterMode,
  filters: ProfferPreviewMessageFilters = EMPTY_FILTERS,
  enabled = true,
) {
  return useInfiniteQuery({
    queryKey: [
      "proffer",
      "preview-messages",
      mode,
      previewHandle,
      filters.q ?? "",
      filters.hasAttachments ?? false,
      filters.sender ?? "",
      filters.from ?? "",
      filters.to ?? "",
    ] as const,
    enabled: Boolean(previewHandle) && enabled,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam, signal }) =>
      getProfferPreviewMessages(previewHandle, mode, pageParam, MESSAGE_PAGE_SIZE, signal, filters),
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
  });
}

/** Reads the server-reported match/total counts off the most recent loaded page.
 * `-1` (or a missing field, on an older engine) means "unavailable" and callers should
 * fall back to the count of rows actually loaded in this browser session. */
export function usePreviewMessageTotals(
  pages: Array<{ total_matches?: number | null; total_messages?: number | null }> | undefined,
) {
  return useMemo(() => {
    const last = pages && pages.length > 0 ? pages[pages.length - 1] : undefined;
    const totalMatches = last?.total_matches;
    const totalMessages = last?.total_messages;
    return {
      totalMatches: totalMatches !== undefined && totalMatches !== null && totalMatches >= 0 ? totalMatches : null,
      totalMessages: totalMessages !== undefined && totalMessages !== null && totalMessages >= 0 ? totalMessages : null,
    };
  }, [pages]);
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
          missingPayloadCount: message.attachments.filter((attachment) => attachment.payload_missing).length,
        };
      });

    return { rows, participants };
  }, [pages]);
}

