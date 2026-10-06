import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, ArrowUpRight, ChevronDown, Loader2, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useFixedCase } from "@/lib/fixed-case-context";
import { AppLink, useAppNavigate, useBrowserSearchParams } from "@/lib/router-compat";
import { ApiError, getProfferOperation, getProfferRunSourceContext, listProfferOperations } from "@/lib/api-client";
import type { ProfferOperationDetail, ProfferOperationLifecycle, ProfferOperationSummary } from "@/lib/shared/types";

import { OperationCancelControl } from "./operation-cancel-control";
import { attemptReviewHref, mergeOperationRows, nextOperationAction, sourceFilename } from "./proffer-operations-state";

const PAGE_SIZE = 50;
const POLL_MS = 5_000;
const HANDLE_PATTERN = /^[A-Za-z0-9_-]{32,128}$/;
const OPERATION_STATUSES: readonly ProfferOperationLifecycle[] = [
  "running",
  "awaiting_repair_decision",
  "awaiting_preview_decision",
  "completed",
  "failed",
  "cancelled",
  "unavailable",
];

function requestErrorText(error: unknown) {
  return error instanceof ApiError || error instanceof Error ? error.message : "The operation request failed.";
}

function statusLabel(status: ProfferOperationLifecycle) {
  if (status === "running") return "In progress";
  if (status === "awaiting_preview_decision") return "Needs review";
  if (status === "awaiting_repair_decision") return "Needs a repair decision";
  if (status === "failed") return "Stopped";
  if (status === "completed") return "Completed";
  if (status === "cancelled") return "Canceled";
  return "Status unavailable";
}

function statusTone(status: ProfferOperationLifecycle) {
  if (status === "completed") return "border-emerald-700/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300";
  if (status === "failed") return "border-destructive/40 bg-destructive/10 text-destructive";
  if (status.startsWith("awaiting_")) return "border-amber-700/30 bg-amber-500/10 text-amber-900 dark:text-amber-200";
  if (status === "cancelled" || status === "unavailable") return "border-border bg-muted text-muted-foreground";
  return "border-primary/30 bg-primary/10 text-primary";
}

function workLabel(operation: ProfferOperationSummary) {
  if (operation.wait === "repair_decision") return "A repair decision is waiting";
  if (operation.wait === "preview_decision") return "Your review is needed";
  if (operation.lifecycle === "failed") return operation.reason || "The import stopped before completion.";
  if (operation.lifecycle === "unavailable") return operation.reason || "Current status is unavailable.";
  if (operation.lifecycle === "cancelled") return operation.reason || "This import was canceled.";
  if (operation.terminal) return "This import has finished.";
  return "The source is being imported.";
}

function dateLabel(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function actionText(operation: ProfferOperationSummary) {
  return nextOperationAction(operation.lifecycle);
}

// Byline: Codex · GPT-6 · 2026-10-06
/** Show durable Proffer operations with URL-backed filters and cursor pagination.
 * Inputs: Activity search parameters and the fixed-case mode.
 * Output: operation rows, selected attempt details, and safe existing operator controls.
 * Side effects: reads/polls the operation API and replaces Activity URL parameters.
 * Use as the shared Activity ledger; it does not read legacy Workbench run records.
 */
export function ActivityOperationsLedger() {
  const searchParams = useBrowserSearchParams();
  const rawStatus = searchParams.get("status") ?? searchParams.get("operation_status");
  const statusFilter: ProfferOperationLifecycle | "all" = OPERATION_STATUSES.includes(rawStatus as ProfferOperationLifecycle)
    ? rawStatus as ProfferOperationLifecycle
    : "all";
  return <ActivityOperationsLedgerForStatus key={statusFilter} statusFilter={statusFilter} />;
}

function ActivityOperationsLedgerForStatus({ statusFilter }: { statusFilter: ProfferOperationLifecycle | "all" }) {
  const { mode } = useFixedCase();
  const searchParams = useBrowserSearchParams();
  const navigate = useAppNavigate();
  const sourceFilter = searchParams.get("q") ?? searchParams.get("operation_source") ?? "";
  const selectedHandle = searchParams.get("preview_handle") ?? searchParams.get("operation") ?? "";

  const [operations, setOperations] = useState<ProfferOperationSummary[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const listBusyRef = useRef(false);
  const listAbortRef = useRef<AbortController | null>(null);
  const loadMoreAbortRef = useRef<AbortController | null>(null);
  const pageCountRef = useRef(1);

  const replaceSearch = useCallback((changes: Record<string, string | null>) => {
    const next = new URLSearchParams(searchParams);
    if ("status" in changes) next.delete("operation_status");
    if ("q" in changes) next.delete("operation_source");
    for (const [key, value] of Object.entries(changes)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    const query = next.toString();
    void navigate.replace(`/activity${query ? `?${query}` : ""}`);
  }, [navigate, searchParams]);

  const refresh = useCallback(async (quiet = false, reset = false) => {
    if (listBusyRef.current) return;
    listBusyRef.current = true;
    const controller = new AbortController();
    listAbortRef.current = controller;
    if (!quiet) setRefreshing(true);
    if (reset) setLoading(true);
    try {
      const response = await listProfferOperations({
        status: statusFilter === "all" ? undefined : statusFilter,
        limit: PAGE_SIZE,
      }, controller.signal);
      if (controller.signal.aborted) return;
      setOperations((current) => reset ? response.items : mergeOperationRows(current, response.items));
      if (reset || pageCountRef.current <= 1) setNextCursor(response.next_cursor ?? null);
      setListError(null);
    } catch (error) {
      if (!controller.signal.aborted) setListError(requestErrorText(error));
    } finally {
      if (listAbortRef.current === controller) {
        listAbortRef.current = null;
        listBusyRef.current = false;
        if (reset) setLoading(false);
        if (!quiet) setRefreshing(false);
      }
    }
  }, [setListError, setLoading, setOperations, setNextCursor, setRefreshing, statusFilter]);

  useEffect(() => {
    queueMicrotask(() => void refresh(false, true));
    const timer = window.setInterval(() => void refresh(true), POLL_MS);
    return () => {
      window.clearInterval(timer);
      listAbortRef.current?.abort();
      loadMoreAbortRef.current?.abort();
      listAbortRef.current = null;
      loadMoreAbortRef.current = null;
      listBusyRef.current = false;
    };
  }, [refresh]);

  const filtered = useMemo(() => {
    const needle = sourceFilter.trim().toLocaleLowerCase();
    return operations.filter((operation) => {
      const statusMatches = statusFilter === "all" || operation.lifecycle === statusFilter;
      const sourceMatches = !needle || [sourceFilename(operation.source_ref), operation.source_ref]
        .some((value) => value.toLocaleLowerCase().includes(needle));
      return statusMatches && sourceMatches;
    });
  }, [operations, sourceFilter, statusFilter]);

  const detail = useQuery({
    queryKey: ["activity", "operation-detail", mode, selectedHandle],
    queryFn: ({ signal }) => getProfferOperation(selectedHandle, signal),
    enabled: HANDLE_PATTERN.test(selectedHandle),
    retry: false,
    refetchInterval: (query) => query.state.data?.terminal ? false : POLL_MS,
  });
  const sourceContext = useQuery({
    queryKey: ["activity", "source-context", mode, selectedHandle],
    queryFn: ({ signal }) => getProfferRunSourceContext(selectedHandle, mode, signal),
    enabled: HANDLE_PATTERN.test(selectedHandle),
    retry: false,
  });

  const validSelectedHandle = HANDLE_PATTERN.test(selectedHandle);

  async function loadMore() {
    if (!nextCursor || listBusyRef.current) return;
    listBusyRef.current = true;
    const controller = new AbortController();
    loadMoreAbortRef.current = controller;
    setLoadingMore(true);
    try {
      const response = await listProfferOperations({
        status: statusFilter === "all" ? undefined : statusFilter,
        cursor: nextCursor,
        limit: PAGE_SIZE,
      }, controller.signal);
      if (controller.signal.aborted) return;
      // The server page is older than the already loaded rows; pass it as the
      // lower-priority page so current/latest rows remain at the top.
      setOperations((current) => mergeOperationRows(response.items, current));
      setNextCursor(response.next_cursor ?? null);
      pageCountRef.current += 1;
      setListError(null);
    } catch (error) {
      if (!controller.signal.aborted) setListError(requestErrorText(error));
    } finally {
      if (loadMoreAbortRef.current === controller) {
        loadMoreAbortRef.current = null;
        listBusyRef.current = false;
        setLoadingMore(false);
      }
    }
  }

  return (
    <section className="overflow-hidden border-y bg-card" aria-labelledby="activity-ledger-heading">
      <header className="flex flex-col gap-4 border-b px-4 py-5 sm:px-5 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-xl">
          <h2 id="activity-ledger-heading" className="text-xl font-semibold tracking-tight">Imports</h2>
          <p className="mt-1 text-sm text-muted-foreground">Each entry follows one source from intake through its recorded result.</p>
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <label className="grid gap-1 text-xs font-medium text-muted-foreground">
            Find a source
            <input value={sourceFilter} onChange={(event) => replaceSearch({ q: event.target.value || null, operation_source: null })} placeholder="Filename or folder" className="h-9 w-56 border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary" />
          </label>
          <label className="grid gap-1 text-xs font-medium text-muted-foreground">
            Status
            <select value={statusFilter} onChange={(event) => replaceSearch({ status: event.target.value === "all" ? null : event.target.value, operation_status: null })} className="h-9 border bg-background px-2 text-sm text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary">
              <option value="all">All statuses</option>
              {OPERATION_STATUSES.map((status) => <option key={status} value={status}>{statusLabel(status)}</option>)}
            </select>
          </label>
          <Button type="button" size="sm" variant="outline" className="h-9" disabled={refreshing} onClick={() => void refresh()}>
            <RefreshCw className={`size-3.5 ${refreshing ? "animate-spin motion-reduce:animate-none" : ""}`} /> Refresh
          </Button>
        </div>
      </header>

      {selectedHandle && (
        <OperationDetailPanel
          handle={selectedHandle}
          mode={mode}
          detail={detail.data ?? null}
          loading={validSelectedHandle && detail.isPending}
          error={detail.error ? requestErrorText(detail.error) : null}
          sourceName={sourceContext.data?.registration?.original_filename || sourceFilename(detail.data?.source_ref ?? "")}
          sourceContextLoading={validSelectedHandle && sourceContext.isPending}
          onClose={() => replaceSearch({ preview_handle: null, operation: null })}
        />
      )}

      {listError && (
        <div className="flex items-start gap-2 border-b border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive" role="alert">
          <AlertCircle className="mt-0.5 size-4 shrink-0" />
          <span className="min-w-0 flex-1">The activity list could not be refreshed: {listError}</span>
          <button type="button" className="text-xs font-semibold underline" onClick={() => void refresh()}>Try again</button>
        </div>
      )}

      {loading ? (
        <div className="flex items-center gap-2 px-4 py-12 text-sm text-muted-foreground" role="status"><Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> Loading imports…</div>
      ) : filtered.length === 0 ? (
        <div className="px-5 py-12">
          <h3 className="text-base font-semibold">{operations.length ? "No imports match these filters" : "No imports yet"}</h3>
          <p className="mt-1 max-w-lg text-sm text-muted-foreground">{operations.length ? "Clear the source or status filter to see the loaded activity." : "Imports started from Sources will appear here with status that survives refresh and navigation."}</p>
        </div>
      ) : (
        <ul aria-label="Proffer imports" className="divide-y">
          {filtered.map((operation) => {
            const selected = selectedHandle === operation.preview_handle;
            const title = sourceFilename(operation.source_ref);
            return (
              <li key={operation.preview_handle} className={selected ? "bg-muted/40" : "bg-card"}>
                <article className="grid gap-3 px-4 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:px-5">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="max-w-full truncate text-sm font-semibold" title={operation.source_ref}>{title}</h3>
                      <span className={`inline-flex min-h-6 items-center border px-2 text-[11px] font-medium ${statusTone(operation.lifecycle)}`}>{statusLabel(operation.lifecycle)}</span>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">{workLabel(operation)}</p>
                    <p className="mt-2 text-xs text-muted-foreground">Started {dateLabel(operation.created_at)}</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-3 sm:justify-end">
                    <AppLink href={attemptReviewHref(operation.preview_handle, mode)} className="inline-flex min-h-9 items-center gap-1.5 px-2 text-xs font-semibold text-primary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">
                      {actionText(operation)} <ArrowUpRight className="size-3.5" aria-hidden="true" />
                    </AppLink>
                    <Button type="button" size="sm" variant="outline" aria-expanded={selected} onClick={() => replaceSearch({ preview_handle: selected ? null : operation.preview_handle, operation: null })}>
                      {selected ? "Hide details" : "Details"}<ChevronDown className={`size-3.5 transition-transform ${selected ? "rotate-180" : ""}`} />
                    </Button>
                  </div>
                </article>
              </li>
            );
          })}
        </ul>
      )}

      <footer className="flex flex-wrap items-center justify-between gap-3 border-t bg-muted/20 px-4 py-3 text-xs text-muted-foreground sm:px-5">
        <span>{filtered.length} of {operations.length} loaded imports shown</span>
        {nextCursor ? <Button type="button" size="sm" variant="outline" disabled={loadingMore} onClick={() => void loadMore()}>{loadingMore && <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" />} Load older imports</Button> : <span>All available imports are loaded</span>}
      </footer>
    </section>
  );
}

function OperationDetailPanel({
  handle,
  mode,
  detail,
  loading,
  error,
  sourceName,
  sourceContextLoading,
  onClose,
}: {
  handle: string;
  mode: "DEV" | "LIVE";
  detail: ProfferOperationDetail | null;
  loading: boolean;
  error: string | null;
  sourceName: string;
  sourceContextLoading: boolean;
  onClose: () => void;
}) {
  return (
    <section className="border-b bg-background" aria-label="Import details" aria-live="polite">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5">
        <div className="min-w-0">
          <h3 className="truncate text-sm font-semibold">{sourceName || (sourceContextLoading ? "Resolving source name…" : "Import details")}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">The detailed record is for the selected attempt only.</p>
        </div>
        <button type="button" onClick={onClose} className="min-h-9 px-3 text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">Close</button>
      </header>

      {!HANDLE_PATTERN.test(handle) ? (
        <p className="px-4 py-4 text-sm text-destructive" role="alert">This import link is invalid.</p>
      ) : loading ? (
        <p className="flex items-center gap-2 px-4 py-4 text-sm text-muted-foreground" role="status"><Loader2 className="size-4 animate-spin motion-reduce:animate-none" /> Loading import details…</p>
      ) : error ? (
        <p className="px-4 py-4 text-sm text-destructive" role="alert">{error}</p>
      ) : detail ? (
        <div className="grid gap-4 p-4 sm:grid-cols-[minmax(0,1fr)_minmax(14rem,0.7fr)] sm:p-5">
          <div className="space-y-3">
            <p className="text-sm">{workLabel(detail)}</p>
            {detail.reason && <p className="border-l-2 border-destructive/50 pl-3 text-sm text-muted-foreground">{detail.reason}</p>}
            <details className="border px-3 py-2 text-xs text-muted-foreground">
              <summary className="cursor-pointer font-medium text-foreground">Technical details</summary>
              <dl className="mt-3 grid gap-2">
                <div><dt className="font-medium">Source reference</dt><dd className="mt-0.5 break-all font-mono text-[10px]">{detail.source_ref}</dd></div>
                <div><dt className="font-medium">Attempt reference</dt><dd className="mt-0.5 break-all font-mono text-[10px]">{detail.preview_handle}</dd></div>
                <div><dt className="font-medium">Request reference</dt><dd className="mt-0.5 break-all font-mono text-[10px]">{detail.request_id}</dd></div>
                {detail.source_version_ref && <div><dt className="font-medium">Source version</dt><dd className="mt-0.5 break-all font-mono text-[10px]">{detail.source_version_ref}</dd></div>}
                {detail.current_stage && <div><dt className="font-medium">Current stage</dt><dd className="mt-0.5">{detail.current_stage.replaceAll("_", " ")}</dd></div>}
                <div><dt className="font-medium">Recorded stages</dt><dd className="mt-1 space-y-2">{detail.stages.length ? detail.stages.map((stage, index) => <div key={`${stage.stage}-${index}`} className="border-t pt-2"><span className="font-medium text-foreground">{stage.stage.replaceAll("_", " ")}</span><span> — {stage.status.replaceAll("_", " ")}</span>{stage.reason && <p className="mt-1">{stage.reason}</p>}</div>) : "No stage rows are available."}</dd></div>
              </dl>
            </details>
          </div>
          <aside className="space-y-3 border-t pt-3 sm:border-l sm:border-t-0 sm:pl-4 sm:pt-0">
            <AppLink href={attemptReviewHref(detail.preview_handle, mode)} className="inline-flex min-h-9 items-center gap-1.5 text-xs font-semibold text-primary hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">
              {detail.lifecycle === "failed" ? "Open Review to start a new import" : "Open this import in Review"}<ArrowUpRight className="size-3.5" aria-hidden="true" />
            </AppLink>
            <OperationCancelControl previewHandle={detail.preview_handle} mode={mode} />
          </aside>
        </div>
      ) : null}
    </section>
  );
}
