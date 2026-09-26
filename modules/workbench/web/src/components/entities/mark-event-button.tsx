// Byline: Claude Code · Opus 5.5 · 2026-09-25 ("event worth recalling" on one record)
"use client";

import { CalendarPlus, Check } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useEntityActions } from "@/hooks/use-entity-extraction";
import type { MatterMode } from "@/lib/shared/types";

/**
 * Marks one record of the run as an event worth recalling. The event is
 * dated by the record's own time (never by now) and proposed like any other
 * event; it reaches the timeline only when the owner runs the workflow.
 * Drop-in for the Review message detail:
 *   <MarkEventButton previewHandle={handle} mode={mode} recordId={message.message_id} />
 */
export function MarkEventButton({
  previewHandle,
  mode,
  recordId,
  compact = false,
  onMarked,
}: {
  previewHandle: string;
  mode: MatterMode;
  recordId: string;
  compact?: boolean;
  onMarked?: () => void;
}) {
  const actions = useEntityActions(previewHandle, mode);
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (done) {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-[#2f9d67]" role="status">
        <Check className="size-3.5" /> Proposed as an event
      </span>
    );
  }
  if (!open) {
    return (
      <Button type="button" variant="outline" size={compact ? "xs" : "sm"} onClick={() => setOpen(true)} data-testid="mark-event-button">
        <CalendarPlus /> Mark as event worth recalling
      </Button>
    );
  }
  const submit = async () => {
    setError(null);
    try {
      await actions.markEvent(recordId, title.trim() || undefined);
      setDone(true);
      onMarked?.();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    }
  };
  return (
    <form
      className="flex flex-wrap items-center gap-1"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <Input
        aria-label="Event title"
        className="h-7 w-56 text-xs"
        placeholder="What happened (optional)"
        maxLength={200}
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        autoFocus
      />
      <Button type="submit" size="xs" disabled={actions.mark.isPending}>Propose event</Button>
      <Button type="button" size="xs" variant="ghost" onClick={() => setOpen(false)}>Cancel</Button>
      {error && <span className="basis-full text-xs text-destructive" role="alert">{error}</span>}
    </form>
  );
}
