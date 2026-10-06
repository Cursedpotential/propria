// Byline: Codex · GPT-5.6 · 2026-09-12 (hydrate deep-linked preview mode and handle atomically)
// Byline: Claude Code · Opus 5.5 · 2026-09-25 (Actions panel wiring: re-run + gate answers; one mode indicator)
// Byline amendment: Claude Code · Opus 5.5 · 2026-09-26 (repair re-entry opens in Review; unbound-run count flag)
"use client";

import { ChevronLeft, CircleDot, Loader2, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useRouter } from "@tanstack/react-router";

import { ProfferOperatorPreview } from "@/components/sbv/proffer-operator-preview";
import { ReviewResourceList } from "@/components/sbv/review-resource-list";
import { ContextFlowRail } from "@/components/intake/context-flow-rail";
import { Button } from "@/components/ui/button";
import { useReviewRerun } from "@/hooks/use-review-rerun";
import {
  createProfferPreviewEventSource,
  createProfferPotentialPromotionFlag,
  decideProffer,
  decideProfferHandler,
  decideProfferRepair,
  getProfferOperatorSnapshot,
  getProfferPreviewContent,
  getProfferPreview,
  getProfferPreviewMessages,
  listProfferProposalResources,
  listProfferPotentialPromotionFlags,
} from "@/lib/api-client";
import { PROFFER_CONTEXT_CHECKPOINTS, profferContextFlowComplete } from "@/lib/proffer-context-checkpoints";
import { useFixedCase } from "@/lib/fixed-case-context";
import { parseOperatingMode } from "@/lib/operating-mode";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { previewSelectionHref } from "@/lib/workflow-links";
import type {
  ProfferPreviewEvent,
  ProfferPreviewMessage,
  ProfferPreviewParticipant,
  ProfferPreviewResponse,
  ProfferOperatorSnapshot,
  ProfferParserCandidate,
  ProfferContentResponse,
  ProfferPotentialPromotionFlag,
  ProfferPotentialPromotionScope,
  ProfferProposalResource,
} from "@/lib/shared/types";

// Shown as one small line inside the Decision block while Approve is locked.
const APPROVAL_LOCK_REASON = "Approval remains locked until this exact attempt has normalized records, source locators, and every required completed receipt.";

function initialHandle(mode: "DEV" | "LIVE") {
  if (typeof window === "undefined") return "";
  const query = new URLSearchParams(window.location.search);
  if (parseOperatingMode(query.get("mode")) !== mode) return "";
  return (query.get("resource") ?? query.get("preview_handle") ?? query.get("attempt"))?.trim() ?? "";
}

export function ProfferPreviewClient() {
  const { mode } = useFixedCase();
  return <ModeScopedPreviewClient key={mode} mode={mode} />;
}

function ModeScopedPreviewClient({ mode }: { mode: "DEV" | "LIVE" }) {
  const router = useRouter();
  const routeParams = useBrowserSearchParams();
  const [initialUrlHandle] = useState(() => initialHandle(mode));
  const [previewHandle, setPreviewHandle] = useState(initialUrlHandle);
  const [resources, setResources] = useState<ProfferProposalResource[]>([]);
  // Runs whose Test/Live mode the server could not prove: never listed, only counted.
  const [unboundCount, setUnboundCount] = useState(0);
  const [resourcesLoading, setResourcesLoading] = useState(true);
  const [resourcesError, setResourcesError] = useState<string | null>(null);
  const [preview, setPreview] = useState<ProfferPreviewResponse | null>(null);
  const [operatorSnapshot, setOperatorSnapshot] = useState<ProfferOperatorSnapshot | null>(null);
  const [messages, setMessages] = useState<ProfferPreviewMessage[]>([]);
  const [participants, setParticipants] = useState<ProfferPreviewParticipant[]>([]);
  const [content, setContent] = useState<ProfferContentResponse | null>(null);
  const [contentError, setContentError] = useState<string | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const [potentialFlags, setPotentialFlags] = useState<ProfferPotentialPromotionFlag[]>([]);
  const [flagError, setFlagError] = useState<string | null>(null);
  const [flagPendingTarget, setFlagPendingTarget] = useState<string | null>(null);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [events, setEvents] = useState<ProfferPreviewEvent[]>([]);
  const [snapshotError, setSnapshotError] = useState<string | null>(null);
  const [messageError, setMessageError] = useState<string | null>(null);
  const [eventError, setEventError] = useState<string | null>(null);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [decisionPending, setDecisionPending] = useState(false);
  const [rejectionReason, setRejectionReason] = useState("");
  const generationRef = useRef(initialUrlHandle ? 1 : 0);
  const activeHandleRef = useRef(initialUrlHandle);
  const snapshotControllerRef = useRef<AbortController | null>(null);
  const messageControllersRef = useRef(new Map<string, AbortController>());
  const contentControllerRef = useRef<AbortController | null>(null);
  const flagControllerRef = useRef<AbortController | null>(null);
  const requestedCursorsRef = useRef(new Set<string>());
  const resourcesControllerRef = useRef<AbortController | null>(null);
  // Whether the selected operation has finished; read by the event stream's error handler.
  const operationTerminalRef = useRef(false);

  const activateHandle = useCallback((handle: string) => {
    generationRef.current += 1;
    activeHandleRef.current = handle;
    snapshotControllerRef.current?.abort();
    messageControllersRef.current.forEach((controller) => controller.abort());
    contentControllerRef.current?.abort();
    flagControllerRef.current?.abort();
    messageControllersRef.current.clear();
    requestedCursorsRef.current.clear();
    setPreview(null);
    setOperatorSnapshot(null);
    setMessages([]);
    setParticipants([]);
    setContent(null);
    setContentError(null);
    setContentLoading(false);
    setPotentialFlags([]);
    setFlagError(null);
    setFlagPendingTarget(null);
    setNextCursor(null);
    setEvents([]);
    setSnapshotError(null);
    setMessageError(null);
    setEventError(null);
    setMessagesLoading(false);
    setDecisionPending(false);
    setRejectionReason("");
    setPreviewHandle(handle);
  }, []);

  const selectResource = useCallback((handle: string) => {
    activateHandle(handle);
    void router.navigate({ href: previewSelectionHref(window.location.search, handle, mode), replace: true });
  }, [activateHandle, mode, router]);

  // Byline: Codex · GPT-6 · 2026-10-06. External links update the attempt without
  // remounting pending repair answers when this component initiates navigation.
  useEffect(() => {
    if (parseOperatingMode(routeParams.get("mode")) !== mode) return;
    const handle = (routeParams.get("resource") ?? routeParams.get("preview_handle") ?? routeParams.get("attempt"))?.trim() ?? "";
    if (!handle || handle === activeHandleRef.current) return;
    const timer = window.setTimeout(() => activateHandle(handle), 0);
    return () => window.clearTimeout(timer);
  }, [activateHandle, mode, routeParams]);

  const loadResources = useCallback(async () => {
    resourcesControllerRef.current?.abort();
    const controller = new AbortController();
    resourcesControllerRef.current = controller;
    setResourcesLoading(true);
    try {
      const response = await listProfferProposalResources(mode, { limit: 50 }, controller.signal);
      if (controller.signal.aborted) return;
      setResources(response.items);
      setUnboundCount(response.unbound_count ?? 0);
      setResourcesError(null);
      if (!activeHandleRef.current && response.items.length > 0) {
        // Land on a run the operator can actually read, not the newest — which is
        // often a failed attempt (opened on an error + the Overview system table)
        // or a 9-stage derive-only run with no preview messages (also fell to
        // Overview). Prefer non-failed runs with the MOST completed stages: the
        // full 24-26-stage runs are the ones carrying normalized messages, so the
        // page lands on the conversation. Owner 2026-09-24: "still fucked".
        const reviewable = response.items
          .filter((item) => item.lifecycle !== "failed" && item.lifecycle !== "unavailable")
          .sort((a, b) => b.completed_stage_count - a.completed_stage_count);
        selectResource((reviewable[0] ?? response.items[0]).preview_handle);
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        setResourcesError(error instanceof Error ? error.message : "Review resources are unavailable");
      }
    } finally {
      if (resourcesControllerRef.current === controller) {
        resourcesControllerRef.current = null;
        setResourcesLoading(false);
      }
    }
  }, [mode, selectResource]);

  useEffect(() => {
    queueMicrotask(() => void loadResources());
    return () => resourcesControllerRef.current?.abort();
  }, [loadResources]);

  const loadSnapshot = useCallback(async () => {
    const handle = previewHandle;
    if (!handle) return;
    const generation = generationRef.current;
    snapshotControllerRef.current?.abort();
    const controller = new AbortController();
    snapshotControllerRef.current = controller;
    try {
      const [result, operator] = await Promise.all([
        getProfferPreview(handle, mode, controller.signal),
        getProfferOperatorSnapshot(handle, mode, controller.signal),
      ]);
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      if (result.preview_handle !== handle) throw new Error("Preview snapshot correlation failed");
      setPreview(result);
      setOperatorSnapshot(operator);
      operationTerminalRef.current = Boolean(operator.terminal);
      setSnapshotError(null);
    } catch (error) {
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setSnapshotError(error instanceof Error ? error.message : "Preview snapshot is unavailable");
    } finally {
      if (snapshotControllerRef.current === controller) snapshotControllerRef.current = null;
    }
  }, [previewHandle, mode]);

  const loadMessages = useCallback(async (cursor?: string) => {
    const handle = previewHandle;
    if (!handle) return;
    const generation = generationRef.current;
    const cursorKey = cursor ?? "__first__";
    if (requestedCursorsRef.current.has(cursorKey)) return;
    requestedCursorsRef.current.add(cursorKey);
    const controller = new AbortController();
    messageControllersRef.current.set(cursorKey, controller);
    setMessagesLoading(true);
    try {
      const page = await getProfferPreviewMessages(handle, mode, cursor, 100, controller.signal);
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      if (page.preview_handle !== handle) throw new Error("Preview message correlation failed");
      setParticipants((current) => {
        const merged = new Map((cursor ? current : []).map((item) => [item.participant_id, item]));
        page.participants.forEach((item) => merged.set(item.participant_id, item));
        return [...merged.values()];
      });
      setMessages((current) => {
        const merged = new Map((cursor ? current : []).map((item) => [item.message_id, item]));
        page.messages.forEach((item) => merged.set(item.message_id, item));
        return [...merged.values()].sort((left, right) => left.ordinal - right.ordinal);
      });
      setNextCursor(page.next_cursor ?? null);
      setMessageError(null);
    } catch (error) {
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setMessageError(error instanceof Error ? error.message : "Preview messages are unavailable");
    } finally {
      if (messageControllersRef.current.get(cursorKey) === controller) {
        requestedCursorsRef.current.delete(cursorKey);
        messageControllersRef.current.delete(cursorKey);
      }
      if (generation === generationRef.current && activeHandleRef.current === handle) {
        setMessagesLoading(messageControllersRef.current.size > 0);
      }
    }
  }, [previewHandle, mode]);

  const loadContent = useCallback(async (recordCursor?: string, chunkCursor?: string) => {
    const handle = previewHandle;
    if (!handle) return;
    const generation = generationRef.current;
    contentControllerRef.current?.abort();
    const controller = new AbortController();
    contentControllerRef.current = controller;
    setContentLoading(true);
    try {
      const page = await getProfferPreviewContent(handle, mode, recordCursor, chunkCursor, 100, controller.signal);
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setContent((current) => {
        const append = Boolean(recordCursor || chunkCursor);
        const records = new Map((append && current ? current.records : []).map((item) => [item.record_id, item]));
        page.records.forEach((item) => records.set(item.record_id, item));
        const chunks = new Map((append && current ? current.chunks : []).map((item) => [item.chunk_ref, item]));
        page.chunks.forEach((item) => chunks.set(item.chunk_ref, item));
        return {
          ...page,
          records: [...records.values()].sort((left, right) => left.ordinal - right.ordinal),
          chunks: [...chunks.values()].sort((left, right) => left.index - right.index),
        };
      });
      setContentError(null);
    } catch (error) {
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setContentError(error instanceof Error ? error.message : "Package, record, and chunk preview is unavailable");
    } finally {
      if (contentControllerRef.current === controller) contentControllerRef.current = null;
      if (generation === generationRef.current && activeHandleRef.current === handle) setContentLoading(false);
    }
  }, [previewHandle, mode]);

  const loadPotentialFlags = useCallback(async () => {
    const handle = previewHandle;
    if (!handle) return;
    const generation = generationRef.current;
    flagControllerRef.current?.abort();
    const controller = new AbortController();
    flagControllerRef.current = controller;
    try {
      const result = await listProfferPotentialPromotionFlags(handle, mode, controller.signal);
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setPotentialFlags(result.flags);
      setFlagError(null);
    } catch (error) {
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setFlagError(error instanceof Error ? error.message : "Potential-promotion flags are unavailable");
    } finally {
      if (flagControllerRef.current === controller) flagControllerRef.current = null;
    }
  }, [previewHandle, mode]);

  useEffect(() => {
    if (!previewHandle) return;
    const generation = generationRef.current;
    const initialLoad = window.setTimeout(() => {
      void loadSnapshot();
    }, 0);
    operationTerminalRef.current = false;
    let replayedEvents = 0;
    const source = createProfferPreviewEventSource(previewHandle, mode);
    const onEvent = (raw: MessageEvent<string>) => {
      // Count what THIS stream delivered before any generation gate: the gate can be
      // stale for the first URL-driven load, and a delivered event is proof the stream
      // works whatever the page decides to do with it.
      replayedEvents += 1;
      try {
        if (generation !== generationRef.current || activeHandleRef.current !== previewHandle) return;
        const event = JSON.parse(raw.data) as ProfferPreviewEvent;
        if (event.preview_handle !== previewHandle) throw new Error("Preview event correlation failed");
        if (event.matter_mode !== mode) throw new Error("Preview event crossed the active DEV/LIVE boundary");
        setEvents((current) => [...current.filter((item) => item.event_id !== event.event_id), event]
          .sort((left, right) => left.event_id - right.event_id)
          .slice(-100));
        setEventError(null);
        void loadSnapshot();
      } catch (error) {
        setEventError(error instanceof Error ? error.message : "Malformed preview event");
        source.close();
      }
    };
    source.addEventListener("proffer.preview", onEvent as EventListener);
    source.onerror = () => {
      // The server replays the durable events and then ends the response. EventSource reports
      // every ended response as an error and reconnects with Last-Event-ID, so an end after a
      // replay is the normal case, not an outage (measured live 2026-09-21: HTTP 200, events
      // delivered, response closed in 0.18 s). A finished operation has nothing further to
      // stream, so its source is closed rather than left reconnecting every few seconds.
      if (replayedEvents > 0) {
        if (operationTerminalRef.current) source.close();
        return;
      }
      if (generation !== generationRef.current || activeHandleRef.current !== previewHandle) return;
      setEventError("The Proffer preview event stream is unavailable");
    };
    const messageControllers = messageControllersRef.current;
    const requestedCursors = requestedCursorsRef.current;
    return () => {
      window.clearTimeout(initialLoad);
      source.close();
      snapshotControllerRef.current?.abort();
      contentControllerRef.current?.abort();
      flagControllerRef.current?.abort();
      messageControllers.forEach((controller) => controller.abort());
      messageControllers.clear();
      requestedCursors.clear();
    };
  }, [loadSnapshot, mode, previewHandle]);

  const contextFlowComplete = profferContextFlowComplete(preview?.receipts, preview?.checkpoints);

  useEffect(() => {
    if (!previewHandle || !contextFlowComplete) return;
    const timer = window.setTimeout(() => {
      void loadMessages();
      void loadContent();
      void loadPotentialFlags();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [contextFlowComplete, loadContent, loadMessages, loadPotentialFlags, previewHandle]);

  async function flagPotentialPromotion(
    scope: ProfferPotentialPromotionScope,
    targetId: string,
    attemptId: string,
    reason: string,
  ) {
    const handle = previewHandle;
    const generation = generationRef.current;
    if (!handle || !reason.trim()) return;
    const pendingKey = `${scope}:${targetId}`;
    setFlagPendingTarget(pendingKey);
    try {
      const created = await createProfferPotentialPromotionFlag(handle, mode, {
        scope,
        target_id: targetId,
        attempt_id: attemptId,
        reason: reason.trim(),
      });
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      setPotentialFlags((current) => [
        ...current.filter((flag) => flag.flag_id !== created.flag_id),
        created,
      ]);
      setFlagError(null);
      toast.success("Potential-promotion classification recorded");
    } catch (error) {
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      const detail = error instanceof Error ? error.message : "Potential-promotion flag failed";
      setFlagError(detail);
      toast.error(detail);
    } finally {
      if (generation === generationRef.current && activeHandleRef.current === handle) {
        setFlagPendingTarget(null);
      }
    }
  }

  async function decide(approved: boolean, explicitReason = rejectionReason.trim()) {
    const handle = previewHandle;
    const generation = generationRef.current;
    if (!handle || !decisionEligible) {
      toast.error("Load the correlated messages, provenance, and completed receipts before deciding");
      return;
    }
    if (!approved && !explicitReason) {
      toast.error("A rejection requires a reason");
      return;
    }
    setDecisionPending(true);
    try {
      const result = await decideProffer(handle, mode, { approved, reason: approved ? "" : explicitReason });
      if (generation !== generationRef.current || activeHandleRef.current !== handle) return;
      if (result.preview_handle !== handle) throw new Error("Decision response correlation failed");
      toast.success(approved ? "Review approved" : "Review rejected");
      await loadSnapshot();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Decision failed");
    } finally {
      if (generation === generationRef.current && activeHandleRef.current === handle) {
        setDecisionPending(false);
      }
    }
  }

  async function retainOriginal() {
    if (!previewHandle || preview?.phase !== "awaiting_repair_decision") return;
    setDecisionPending(true);
    try {
      await decideProfferRepair(previewHandle, mode, { approved: true, apply_repair: false });
      toast.success("Original-source override recorded");
      await loadSnapshot();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "The repair decision failed");
    } finally {
      setDecisionPending(false);
    }
  }

  async function selectHandler(candidate: ProfferParserCandidate) {
    if (!previewHandle || preview?.phase !== "awaiting_handler_selection" || !preview.handler_recommendation_ref) return;
    setDecisionPending(true);
    try {
      await decideProfferHandler(previewHandle, mode, {
        recommendation_ref: preview.handler_recommendation_ref,
        handler_id: candidate.handler_id,
        handler_version: candidate.handler_version,
        execution_path: candidate.execution_path,
        compatibility_ref: candidate.compatibility_ref,
      });
      toast.success("Handler selection recorded");
      await loadSnapshot();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "The handler selection failed");
    } finally {
      setDecisionPending(false);
    }
  }

  const awaitingDecision = preview?.phase === "awaiting_decision";
  const requiredReceiptTypes = useMemo(
    () => PROFFER_CONTEXT_CHECKPOINTS.map(({ type }) => type),
    [],
  );
  const provenanceLoaded = Boolean(content && content.records.length > 0 && content.records.every((record) =>
    Boolean(record.source_locator_ref),
  ));
  const receiptsComplete = requiredReceiptTypes.every((receiptType) =>
    preview?.receipts?.some((receipt) => receipt.receipt_type === receiptType && receipt.status === "completed"),
  );
  const decisionEligible = Boolean(
    awaitingDecision &&
    preview?.preview_handle === previewHandle &&
    content &&
    !snapshotError &&
    !contentError &&
    provenanceLoaded &&
    receiptsComplete,
  );

  // A new run of the same source is selected as soon as it starts, so its progress and any
  // stop it reaches are on screen at once.
  const openStartedRun = useCallback((handle: string) => {
    selectResource(handle);
    void loadResources();
  }, [loadResources, selectResource]);
  const { rerun, starting: rerunStarting, pendingForRun } = useReviewRerun({
    mode,
    previewHandle,
    snapshot: operatorSnapshot,
    preview,
    loadSnapshot,
    onStarted: openStartedRun,
  });

  return (
    // One-viewport Review (owner ruling 2026-09-20 23:48): a one-line header, the sources list as
    // a left rail, and the selected source's strip, checkpoints and views beside it.
    <div className="mx-auto w-full max-w-[1680px] space-y-2 p-2 md:p-3">
      <header className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-3">
          <Button asChild variant="outline" size="sm"><AppLink data-testid="back-to-proffer-intake" href="/intake"><ChevronLeft className="size-4" /> Back to intake</AppLink></Button>
          <h1 className="truncate text-base font-semibold tracking-tight" title="Select a source proposal, inspect every available context artifact, and control the exact processing attempt.">Context Review workspace</h1>
          <p className="sr-only">Unified operator surface · bounded client</p>
          <p className="sr-only">
            Select a source proposal, inspect every available context artifact, and control the exact processing attempt.
          </p>
        </div>
        {/* No Test / Live switch and no mode chip here: the top bar holds the only switch and the
            run header carries one small flag (owner 2026-09-25: Test was shown four times). */}
      </header>

      <div className="grid gap-2 lg:grid-cols-[19rem_minmax(0,1fr)] lg:items-start">
      <section className="platform-panel overflow-hidden lg:sticky lg:top-0" aria-labelledby="review-resources-heading">
        <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
          <div>
            <h2 id="review-resources-heading" className="text-sm font-semibold">Sources and proposals</h2>
          </div>
          <div className="flex items-center gap-2">
            <Button asChild variant="outline" size="sm"><AppLink href={`/intake?mode=${mode}`}>Start intake</AppLink></Button>
            <Button variant="ghost" size="sm" onClick={() => void loadResources()} disabled={resourcesLoading}>
              {resourcesLoading ? <Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> : <RefreshCw className="size-4" />} Refresh list
            </Button>
          </div>
        </header>
        {resourcesError && <div className="border-b border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert"><strong>Resource list unavailable.</strong> {resourcesError}</div>}
        <ReviewResourceList resources={resources} unboundCount={unboundCount} loading={resourcesLoading} selectedHandle={previewHandle} onSelect={selectResource} />
      </section>

      <div className="min-w-0 space-y-2">
      {previewHandle && <ContextFlowRail
        started
        phase={preview?.phase}
        receipts={preview?.receipts}
        checkpoints={preview?.checkpoints}
        events={events}
      />}

      {!previewHandle ? (
        <div className="platform-panel rounded-md px-5 py-8 text-center">
          <CircleDot className="mx-auto size-9 text-muted-foreground" />
          <p className="mt-3 text-sm font-medium">Choose a source proposal or start intake.</p>
        </div>
      ) : snapshotError ? (
        <div className="platform-panel border-destructive/50 p-5 text-sm text-destructive" role="alert">{snapshotError}</div>
      ) : !preview || !operatorSnapshot ? (
        <section className="platform-panel flex items-center justify-center p-6 text-sm text-muted-foreground" aria-label="Review loading"><Loader2 className="mr-2 size-4 animate-spin motion-reduce:animate-none" /> Loading the {mode} review workspace…</section>
      ) : (
        <>
          {eventError && <p className="truncate px-1 text-[11px] text-muted-foreground" role="status" title={eventError}>Live updates paused: {eventError}</p>}
          <ProfferOperatorPreview
            key={`${mode}:${previewHandle}`}
            snapshot={operatorSnapshot}
            preview={preview}
            messages={messages}
            participants={participants}
            events={events}
            content={content}
            contentLoading={contentLoading}
            contentError={contentError}
            potentialFlags={potentialFlags}
            flagError={flagError}
            flagPendingTarget={flagPendingTarget}
            messagesLoading={messagesLoading}
            messageError={messageError}
            hasMore={Boolean(nextCursor)}
            onLoadMore={() => void loadMessages(nextCursor ?? undefined)}
            onLoadMoreContent={(recordCursor, chunkCursor) => void loadContent(recordCursor, chunkCursor)}
            onFlagPotentialPromotion={(scope, targetId, attemptId, reason) => void flagPotentialPromotion(scope, targetId, attemptId, reason)}
            onRefresh={() => void loadSnapshot()}
            onApprove={() => decisionEligible && void decide(true)}
            onReject={(reason) => decisionEligible && void decide(false, reason)}
            onRetainOriginal={() => void retainOriginal()}
            onSelectHandler={(candidate) => void selectHandler(candidate)}
            onRerun={(request) => void rerun(request)}
            rerunPending={rerunStarting}
            pendingAnswers={pendingForRun}
            onOpenRun={openStartedRun}
            decisionLockReason={APPROVAL_LOCK_REASON}
            actionPending={decisionPending}
            decisionReady={decisionEligible}
          />
        </>
      )}
      </div>
      </div>
    </div>
  );
}
