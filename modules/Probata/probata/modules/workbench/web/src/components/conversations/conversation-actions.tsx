// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Extract and Send to Surreal for checked conversations, on desktop and in /m. Built from shadcn Sheet, Checkbox, Button and
// Badge, and the existing WorkflowSteps list (components/entities/workflow-steps.tsx) for progress; nothing is drawn by hand.
//   ConversationSelectionBar  "N selected" with Extract and Send to Surreal, over a list that has checkboxes
//   ConversationActionSheet   the extractor picker (multi-select, default pre-checked) or the send confirmation, then progress
//   ConversationToolbar       the same actions for the one conversation being read
"use client";

import { useMemo, useState } from "react";

import { WorkflowSteps } from "@/components/entities/workflow-steps";
import { ExtractionsView } from "@/components/conversations/extractions-view";
import { errorText } from "@/components/mobile/mobile-format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { useActionStatus, useExtractors, useStartExtraction, useStartSend } from "@/hooks/use-conversation-actions";
import type { ActionStatus, ExtractorInfo } from "@/lib/conversation-actions-client";
import type { WorkflowProgress } from "@/lib/entity-extraction-client";
import { cn } from "@/lib/utils";

export type ActionKind = "extract" | "send";

/** The engine's lines, as the existing step list takes them: one line per conversation and extractor. */
function toProgress(status: ActionStatus, extractors: readonly ExtractorInfo[]): WorkflowProgress {
  const label = (id?: string) => extractors.find((extractor) => extractor.id === id)?.label ?? (id === "surreal" ? "surreal-case" : id);
  return {
    workflow_id: status.workflow_id,
    outcome: status.outcome,
    steps: status.steps.map((step) => ({
      step: step.conversation ? `${step.conversation} · ${label(step.extractor) ?? ""}` : step.step,
      status: step.status,
      detail: step.detail ?? "",
      counts: step.counts ?? {},
      flags: step.flags ?? [],
    })),
  };
}

function ActionProgress({ workflowId, threadIds, title }: { workflowId: string; threadIds: readonly string[]; title: string }) {
  const status = useActionStatus(workflowId, threadIds);
  const extractors = useExtractors();
  const progress = useMemo(() => (status.data ? toProgress(status.data, extractors.data?.extractors ?? []) : undefined), [status.data, extractors.data]);
  if (status.isError) return <p className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm" role="alert">{errorText(status.error)}</p>;
  if (!progress) return <Skeleton className="h-24 w-full" />;
  return (
    <div className="space-y-2">
      <WorkflowSteps progress={progress} title={title} />
      {status.data?.receipts.length ? (
        <ul className="space-y-1 text-xs text-muted-foreground" aria-label="Receipts">
          {status.data.receipts.map((receipt) => (
            <li key={receipt.thread_id}>
              Receipt · {receipt.verified.messages} of {receipt.plan.messages} messages and {receipt.verified.entities} entities, {receipt.verified.events} events read back from surreal-case
              {receipt.verified.match ? "" : " (counts differ)"}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function ExtractorChoice({ extractor, checked, onChange }: { extractor: ExtractorInfo; checked: boolean; onChange: (next: boolean) => void }) {
  return (
    <label className={cn("flex min-h-14 cursor-pointer items-start gap-3 rounded-md border p-3", checked && "border-primary bg-primary/5")}>
      <Checkbox checked={checked} onCheckedChange={(value) => onChange(value === true)} className="mt-0.5 size-5" aria-label={extractor.label} data-testid={`extractor-${extractor.id}`} />
      <span className="min-w-0 flex-1">
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="text-sm font-semibold">{extractor.label}</span>
          {extractor.default ? <Badge variant="secondary">default</Badge> : null}
          {extractor.compare_only ? <Badge variant="outline">compare only</Badge> : null}
        </span>
        <span className="mt-0.5 block text-xs text-muted-foreground">{extractor.description}</span>
      </span>
    </label>
  );
}

interface ActionSheetProps {
  kind: ActionKind;
  threadIds: readonly string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called after a workflow was started, so a list can clear its checkboxes. */
  onStarted?: () => void;
  side?: "bottom" | "right";
}

/** The extractor picker, or the send confirmation, for these conversations; then the run's progress in place. Callers mount it only while open, so every open starts from the defaults. */
export function ConversationActionSheet({ kind, threadIds, open, onOpenChange, onStarted, side = "bottom" }: ActionSheetProps) {
  const extractors = useExtractors();
  const [chosen, setChosen] = useState<ReadonlySet<string> | null>(null);
  const [includeExtractions, setIncludeExtractions] = useState(true);
  const [started, setStarted] = useState<{ workflowId: string; threadIds: string[] } | null>(null);
  const startExtraction = useStartExtraction();
  const startSend = useStartSend();

  const defaults = useMemo(() => new Set((extractors.data?.extractors ?? []).filter((extractor) => extractor.default).map((extractor) => extractor.id)), [extractors.data]);
  const checked = chosen ?? defaults;
  const pending = startExtraction.isPending || startSend.isPending;
  const error = startExtraction.error ?? startSend.error;

  function toggle(id: string, next: boolean) {
    const copy = new Set(checked);
    if (next) copy.add(id);
    else copy.delete(id);
    setChosen(copy);
  }

  async function run() {
    const ids = [...threadIds];
    const result = kind === "extract"
      ? await startExtraction.mutateAsync({ threadIds: ids, extractors: [...checked] })
      : await startSend.mutateAsync({ threadIds: ids, includeExtractions });
    setStarted({ workflowId: result.workflow_id, threadIds: ids });
    onStarted?.();
  }

  const count = threadIds.length;
  const noun = `${count} conversation${count === 1 ? "" : "s"}`;
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side={side} className={cn("overflow-y-auto", side === "bottom" ? "max-h-[88dvh]" : "w-full sm:max-w-lg")}>
        <SheetHeader>
          <SheetTitle>{kind === "extract" ? "Extract entities and events" : "Send to Surreal"}</SheetTitle>
          <SheetDescription>
            {kind === "extract"
              ? `Choose which extractors read ${noun}. Each result is tagged with the extractor that made it, so you can compare them.`
              : `Send ${noun} to surreal-case, in their own tables. Sending again never makes a second copy.`}
          </SheetDescription>
        </SheetHeader>
        <div className="space-y-3 px-4 pb-2">
          {started ? (
            <ActionProgress workflowId={started.workflowId} threadIds={started.threadIds} title={kind === "extract" ? "Extraction" : "Send to Surreal"} />
          ) : kind === "extract" ? (
            extractors.isPending ? <Skeleton className="h-32 w-full" /> : extractors.isError ? (
              <p className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm" role="alert">{errorText(extractors.error)}</p>
            ) : (
              <div className="space-y-2" role="group" aria-label="Extractors">
                {extractors.data.extractors.map((extractor) => (
                  <ExtractorChoice key={extractor.id} extractor={extractor} checked={checked.has(extractor.id)} onChange={(next) => toggle(extractor.id, next)} />
                ))}
              </div>
            )
          ) : (
            <label className="flex min-h-12 cursor-pointer items-center gap-3 rounded-md border p-3 text-sm">
              <Checkbox checked={includeExtractions} onCheckedChange={(value) => setIncludeExtractions(value === true)} className="size-5" aria-label="Include the extractions" />
              Include the entities and events already extracted (every extractor, each tagged)
            </label>
          )}
          {error ? <p className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm" role="alert">{errorText(error)}</p> : null}
        </div>
        <SheetFooter>
          {started ? (
            <Button type="button" variant="outline" className="h-12" onClick={() => onOpenChange(false)}>Close</Button>
          ) : (
            <Button
              type="button"
              className="h-12"
              disabled={pending || count === 0 || (kind === "extract" && (checked.size === 0 || !extractors.data))}
              onClick={() => void run().catch(() => undefined)}
            >
              {pending ? "Starting" : kind === "extract" ? `Extract with ${checked.size} extractor${checked.size === 1 ? "" : "s"}` : `Send ${noun}`}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

interface SelectionBarProps {
  threadIds: readonly string[];
  onClear: () => void;
  /** mobile sits above the tab bar; desktop sits at the foot of its panel. */
  placement?: "mobile" | "desktop";
}

/** "N selected" with the two actions, shown only while something is checked. */
export function ConversationSelectionBar({ threadIds, onClear, placement = "desktop" }: SelectionBarProps) {
  const [sheet, setSheet] = useState<ActionKind | null>(null);
  if (threadIds.length === 0 && sheet === null) return null;
  return (
    <>
      {threadIds.length > 0 ? (
        <div
          role="region"
          aria-label="Selected conversations"
          className={cn(
            "z-40 flex items-center gap-2 border-t border-border bg-card px-3 py-2 shadow-lg",
            placement === "mobile"
              ? "fixed inset-x-0 bottom-[calc(4rem+env(safe-area-inset-bottom))]"
              : "sticky bottom-0",
          )}
        >
          <span className="min-w-0 flex-1 truncate text-sm font-semibold">{threadIds.length} selected</span>
          <Button type="button" variant="outline" className="h-11" onClick={onClear}>Clear</Button>
          <Button type="button" variant="outline" className="h-11" onClick={() => setSheet("extract")}>Extract</Button>
          <Button type="button" className="h-11" onClick={() => setSheet("send")}>Send to Surreal</Button>
        </div>
      ) : null}
      {sheet ? (
        <ConversationActionSheet
          kind={sheet}
          threadIds={threadIds}
          open
          onOpenChange={(open) => { if (!open) setSheet(null); }}
          onStarted={onClear}
          side={placement === "mobile" ? "bottom" : "right"}
        />
      ) : null}
    </>
  );
}

/** Extractions, Extract and Send to Surreal for the one conversation being read. Read-only view; the actions start workflows. */
export function ConversationToolbar({ threadId, placement = "mobile" }: { threadId: string; placement?: "mobile" | "desktop" }) {
  const [sheet, setSheet] = useState<ActionKind | "view" | null>(null);
  const side = placement === "mobile" ? "bottom" : "right";
  return (
    <div className="flex flex-wrap gap-2 px-3 pt-3" role="toolbar" aria-label="Conversation actions">
      <Button type="button" variant="outline" className="h-11 flex-1" onClick={() => setSheet("view")}>Extractions</Button>
      <Button type="button" variant="outline" className="h-11 flex-1" onClick={() => setSheet("extract")}>Extract</Button>
      <Button type="button" variant="outline" className="h-11 flex-1" onClick={() => setSheet("send")}>Send to Surreal</Button>
      {sheet === "view" ? (
        <Sheet open onOpenChange={(open) => { if (!open) setSheet(null); }}>
          <SheetContent side={side} className={cn("overflow-y-auto", side === "bottom" ? "max-h-[90dvh]" : "w-full sm:max-w-xl")}>
            <SheetHeader>
              <SheetTitle>Extractions</SheetTitle>
              <SheetDescription>What each extractor found in this conversation. Read-only.</SheetDescription>
            </SheetHeader>
            <ExtractionsView threadId={threadId} onExtract={() => setSheet("extract")} />
          </SheetContent>
        </Sheet>
      ) : sheet ? (
        <ConversationActionSheet kind={sheet} threadIds={[threadId]} open onOpenChange={(open) => { if (!open) setSheet(null); }} side={side} />
      ) : null}
    </div>
  );
}
