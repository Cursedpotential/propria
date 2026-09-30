// Byline: Claude Code · Opus 5.5 · 2026-09-25 (click a mention -> the message it came from)
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-27: "Mark as event" lives on the selected message's detail panel only)
"use client";

import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";

import { formatWhen } from "@/components/entities/format";
import { Button } from "@/components/ui/button";
import { getExtractionRecord } from "@/lib/entity-extraction-client";
import type { MatterMode } from "@/lib/shared/types";

/** Shows one record of the run with the mentioned span highlighted. */
export function RecordPeek({
  previewHandle,
  mode,
  recordId,
  highlight,
  onClose,
}: {
  previewHandle: string;
  mode: MatterMode;
  recordId: string;
  highlight?: { start?: number; end?: number };
  onClose: () => void;
}) {
  const record = useQuery({
    queryKey: ["entities", "record", mode, previewHandle, recordId] as const,
    queryFn: ({ signal }) => getExtractionRecord(previewHandle, mode, recordId, signal),
  });
  const body = record.data?.body ?? "";
  const runes = [...body];
  const start = highlight?.start;
  const end = highlight?.end;
  const marked = start !== undefined && end !== undefined && end <= runes.length && start < end;
  return (
    <div className="mt-1 border-l-2 border-primary bg-accent/30 p-2 text-xs" data-testid="record-peek">
      <div className="flex items-start justify-between gap-2">
        <p className="text-muted-foreground">
          {record.data ? `#${record.data.ordinal} · ${formatWhen(record.data.occurred_at)}` : "Loading message…"}
          {record.data?.participants
            .filter((participant) => participant.role === "sender")
            .map((participant) => ` · from ${participant.display_name || participant.identifier}`)}
        </p>
        <Button type="button" size="icon-xs" variant="ghost" aria-label="Close message" onClick={onClose}><X /></Button>
      </div>
      {record.error && <p className="text-destructive" role="alert">{(record.error as Error).message}</p>}
      {record.data && (
        <p className="mt-1 whitespace-pre-wrap break-words leading-5">
          {marked ? (
            <>
              {runes.slice(0, start).join("")}
              <mark className="bg-[#ffe9a8] px-0.5 dark:bg-[#6b5413]">{runes.slice(start, end).join("")}</mark>
              {runes.slice(end).join("")}
            </>
          ) : body || "(no message body)"}
        </p>
      )}
    </div>
  );
}
