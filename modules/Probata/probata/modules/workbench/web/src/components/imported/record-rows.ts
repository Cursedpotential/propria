// Byline: Claude Code · Sonnet · 2026-10-02
// Adapters from an imported message or call to the row shapes the SBV-derived Review components already
// render (MessageBubble takes a PreviewMessageRow, CallsTable takes CallRow), so the imported views reuse
// those components unchanged instead of drawing their own.
import { formatDuration, formatTime } from "@/components/mobile/mobile-format";
import type { CallRow } from "@/components/sbv/calls-table";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { ImportedCall, ImportedMessage } from "@/lib/imported-client";

export function toMessageRow(message: ImportedMessage, ordinal: number): PreviewMessageRow {
  return {
    message: {
      message_id: message.id, ordinal, sent_at: message.at, sender_participant_id: message.sender.id, body: message.body,
      participant_ids: [], attachments: [], source_locator_ref: "",
    },
    senderName: message.sender.label,
    senderAddress: message.sender.number ?? null,
    direction: message.outgoing ? "out" : "in",
    timeLabel: formatTime(message.at),
    bodyLine: message.body,
    attachmentCount: message.attachments,
    missingPayloadCount: 0,
  };
}

export function toCallRow(call: ImportedCall, ordinal: number): CallRow {
  return {
    recordId: call.id, ordinal, whenLabel: call.at ? new Date(call.at).toLocaleString() : "—",
    direction: call.direction === "incoming" || call.direction === "outgoing" ? call.direction : "unknown",
    missed: call.missed,
    // A number nobody has named is shown as the number so the row can offer "Who is this?"; a named person by name.
    number: call.with.number ?? call.with.label,
    durationLabel: call.duration_s === null ? "—" : formatDuration(call.duration_s),
    disposition: call.missed ? "missed" : (call.kind || "—"),
  };
}
