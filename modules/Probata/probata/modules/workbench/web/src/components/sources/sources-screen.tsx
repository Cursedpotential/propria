// Byline: Claude Code · Opus 5 · 2026-09-22
// Byline: Codex · GPT-6 · 2026-10-06 (durable selection, upload, partial retries and Activity links).
// Sources — the front door that replaces the Intake page.
//
// Ratified 2026-09-22 09:09 (docs/pending-review/2026-09-21-intake-review-module-rethink.md,
// option A): one viewport, three regions — folder tree, file rows, metadata
// panel — over the B2 roots, with search built in from step one and ONE
// Process button. Owner, 2026-09-22 10:48: "I don't have time to neatly sort
// before ingest, so it needs to happen, or be able to happen, at the same
// time" — so nothing here gates Process on sorting, on marking a unit, or on
// moving a file to a canonical home. Those are independent actions, in any
// order, on any file wherever it sits.
//
// Every read is a TanStack Query against a governed BFF route; the surface
// holds no derived copy of evidence and promotes nothing.
"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { AlertTriangle, Loader2, Play, Upload, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { SourceMetadataPanel, type SourceSelection } from "@/components/sources/source-metadata-panel";
import { mergeSourcePages, sourceContinuation } from "@/components/sources/source-pages";
import { SourceRowsGrid, type SourceGridRow } from "@/components/sources/source-rows-grid";
import { SourceSearch } from "@/components/sources/source-search";
import { ContextSearch } from "@/components/read/context-search";
import { SourceTree } from "@/components/sources/source-tree";
import { detectedFormat, runsBySource, sourceState } from "@/components/sources/source-state";
import { Button } from "@/components/ui/button";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { processSelection, type SourceSubmission } from "@/components/sources/process-selection";
import {
  ApiError,
  getCatalogProvenance,
  getDecodedExists,
  getProfferBatch,
  getUnitsUnderPrefix,
  previewProfferSource,
  listProfferProposalResources,
  listProfferSources,
  listSourceUnitMarks,
  lookupCatalogUnits,
  proposeSourceUnit,
  recordSourceUnitMark,
  startProffer,
  startProfferBatch,
  uploadProfferSource,
} from "@/lib/api-client";
import { getDecodedManifest } from "@/lib/decoded-source-client";
import { useFixedCase } from "@/lib/fixed-case-context";
import type { CatalogUnitLookup, ProfferUploadResponse, SourceUnitKind, SourceUnitMark } from "@/lib/shared/types";

type CatalogUnit = CatalogUnitLookup["units"][number];

function errorText(error: unknown) {
  return error instanceof ApiError || error instanceof Error ? error.message : "The request failed";
}

function newBatchId() {
  return `batch${crypto.randomUUID().replaceAll("-", "")}`;
}

export function SourcesScreen() {
  const { mode } = useFixedCase();
  return <SourcesScreenMode key={mode} />;
}

function SourcesScreenMode() {
  const { matter, primaryCourtCase, mode } = useFixedCase();
  const searchParams = useBrowserSearchParams();
  const router = useRouter();

  const rootId = searchParams.get("root") ?? "";
  const prefix = searchParams.get("prefix") ?? "";
  const linkedFile = searchParams.get("file") ?? "";
  const [appliedFilter, setAppliedFilter] = useState(linkedFile);
  const [selectedRef, setSelectedRef] = useState<string | null>(null);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [checkedRefs, setCheckedRefs] = useState<ReadonlySet<string>>(() => new Set());
  const [handlerOverride, setHandlerOverride] = useState("");

  const [query, setQuery] = useState(linkedFile);
  const [showContentSearch, setShowContentSearch] = useState(false);
  const [searchSummary, setSearchSummary] = useState<string | null>(null);
  useEffect(() => {
    setAppliedFilter(linkedFile); setQuery(linkedFile); setShowContentSearch(false);
    setSearchSummary(linkedFile ? `Recorded file location: ${linkedFile}` : null);
  }, [linkedFile, rootId]);

  const [unitMarkKind, setUnitMarkKind] = useState<SourceUnitKind | "">("");
  const [markPending, setMarkPending] = useState(false);
  const [markError, setMarkError] = useState<string | null>(null);
  const [localMarks, setLocalMarks] = useState<SourceUnitMark[]>([]);

  const [batchIdent, setBatchIdent] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const [processMessage, setProcessMessage] = useState<string | null>(null);
  const [processError, setProcessError] = useState<string | null>(null);
  // Keep transport retries bound to the same requests, including partial selections.
  const submissions = useRef(new Map<string, SourceSubmission>());
  const batchRequests = useRef(new Map<string, string>());
  const processBusy = useRef(false);
  const [activityHref, setActivityHref] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const [localSource, setLocalSource] = useState<{ file: File; requestId: string } | null>(null);
  const localUploads = useRef(new Map<string, ProfferUploadResponse>());
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);

  // --- reads ---------------------------------------------------------------
  const listingQuery = useInfiniteQuery({
    queryKey: ["sources", "listing", mode, rootId, prefix, appliedFilter],
    initialPageParam: undefined as string | undefined,
    queryFn: ({ signal, pageParam }) =>
      listProfferSources({
        mode,
        rootId: rootId || undefined,
        prefix: appliedFilter ? "" : prefix,
        filter: appliedFilter || undefined,
        continuationToken: pageParam,
        pageSize: 200,
        signal,
      }),
    getNextPageParam: (_lastPage, pages, _lastPageParam, pageParams) =>
      sourceContinuation(pages, pageParams).token,
  });
  const pages = listingQuery.data?.pages;
  const listing = pages?.[0] ?? null;
  const { objects, prefixes } = useMemo(() => mergeSourcePages(pages ?? []), [pages]);
  const continuation = sourceContinuation(pages ?? [], listingQuery.data?.pageParams ?? []);

  const runsQuery = useQuery({
    queryKey: ["sources", "runs", mode],
    queryFn: ({ signal }) => listProfferProposalResources(mode, { limit: 100 }, signal),
  });
  const runsIndex = useMemo(() => runsBySource(runsQuery.data?.items ?? []), [runsQuery.data]);

  const marksQuery = useQuery({
    queryKey: ["sources", "unit-marks"],
    queryFn: ({ signal }) => listSourceUnitMarks(signal),
    retry: false,
  });
  const marks = useMemo(
    () => [
      ...localMarks,
      ...(marksQuery.data?.items ?? []).filter(
        (item) => !localMarks.some((local) => local.unit_root === item.unit_root),
      ),
    ],
    [localMarks, marksQuery.data],
  );

  const sourceRefs = useMemo(() => objects.map((object) => object.source_ref), [objects]);
  const decodedQuery = useQuery({
    queryKey: ["sources", "decoded", sourceRefs],
    queryFn: ({ signal }) => getDecodedExists(sourceRefs, signal),
    enabled: sourceRefs.length > 0,
    retry: false,
  });
  const facts = useMemo(
    () => ({
      runs: runsIndex,
      decoded: new Set((decodedQuery.data?.items ?? []).filter((item) => item.decoded).map((item) => item.source_ref)),
    }),
    [decodedQuery.data, runsIndex],
  );

  const folderKeys = useMemo(() => prefixes.map((entry) => entry.prefix.replace(/\/$/, "")), [prefixes]);
  const objectKeys = useMemo(() => objects.map((object) => object.key), [objects]);
  const unitsQuery = useQuery({
    queryKey: ["sources", "units", folderKeys, objectKeys],
    queryFn: ({ signal }) => lookupCatalogUnits(folderKeys, objectKeys, signal),
    enabled: folderKeys.length > 0 || objectKeys.length > 0,
    retry: false,
  });

  const selectedObject = useMemo(
    () => objects.find((object) => object.source_ref === selectedRef) ?? null,
    [objects, selectedRef],
  );
  const activeRootId = rootId || listing?.active_root_id || "";

  const inspectionQuery = useQuery({
    queryKey: ["sources", "inspection", mode, activeRootId, selectedObject?.source_ref],
    queryFn: ({ signal }) => previewProfferSource(selectedObject!, mode, activeRootId, signal),
    enabled: Boolean(selectedObject && activeRootId),
    retry: false,
  });

  const manifestQuery = useQuery({
    queryKey: ["sources", "manifest", selectedObject?.source_ref],
    queryFn: ({ signal }) => getDecodedManifest(selectedObject!.source_ref, signal),
    enabled: Boolean(selectedObject),
    retry: false,
  });

  const provenanceQuery = useQuery({
    queryKey: ["sources", "provenance", selectedObject?.key],
    queryFn: ({ signal }) => getCatalogProvenance(selectedObject!.key, signal),
    enabled: Boolean(selectedObject),
    retry: false,
  });

  const folderNames = useMemo(() => objects.map((object) => object.name), [objects]);
  const proposalQuery = useQuery({
    queryKey: ["sources", "unit-proposal", selectedFolder, folderNames],
    queryFn: ({ signal }) => proposeSourceUnit(selectedFolder!, folderNames, signal),
    enabled: Boolean(selectedFolder),
    retry: false,
  });

  // A vault folder never equals a catalog `unit_root` (those are ORIGINAL
  // source paths — verified live 2026-09-22), so a folder's unit is derived
  // from the catalog membership of the files under it, one folder at a time.
  const folderUnitsQuery = useQuery({
    queryKey: ["sources", "folder-units", selectedFolder],
    queryFn: ({ signal }) => getUnitsUnderPrefix(selectedFolder!, signal),
    enabled: Boolean(selectedFolder),
    retry: false,
  });

  const batchQuery = useQuery({
    queryKey: ["sources", "batch", batchIdent, mode],
    queryFn: ({ signal }) => getProfferBatch(batchIdent!, mode, signal),
    enabled: Boolean(batchIdent),
    refetchInterval: (query) => (query.state.data?.terminal ? false : 5000),
    retry: false,
  });

  // --- derived -------------------------------------------------------------
  const unitRoots = useMemo(() => {
    const map = new Map<string, string>();
    for (const unit of unitsQuery.data?.units ?? []) {
      map.set(unit.unit_root.replace(/\/$/, ""), unit.unit_type);
      if (unit.export_root) map.set(unit.export_root.replace(/\/$/, ""), unit.unit_type);
    }
    for (const mark of marks) map.set(mark.unit_root.replace(/\/$/, ""), mark.unit_type);
    return map;
  }, [marks, unitsQuery.data]);

  const memberUnits = useMemo(() => {
    const map = new Map<string, string>();
    for (const member of unitsQuery.data?.members ?? []) map.set(member.key, member.unit_type);
    return map;
  }, [unitsQuery.data]);

  const rows = useMemo<SourceGridRow[]>(
    () =>
      objects.map((object) => ({
        sourceRef: object.source_ref,
        name: object.name,
        byteLength: object.byte_length,
        lastModified: object.last_modified ?? null,
        state: sourceState(object.source_ref, facts),
        unitMark: memberUnits.get(object.key)?.replaceAll("_", " ") ?? "",
      })),
    [facts, memberUnits, objects],
  );
  const selectedIndex = rows.findIndex((row) => row.sourceRef === selectedRef);

  const proposal = proposalQuery.data ?? null;
  const markKind: SourceUnitKind = unitMarkKind || proposal?.looks_like || "other";

  const selection: SourceSelection = selectedFolder
    ? {
        kind: "folder",
        prefix: selectedFolder,
        unit:
          findUnit(unitsQuery.data?.units ?? [], selectedFolder) ??
          folderUnitsQuery.data?.single_unit ??
          null,
        unitsUnder: folderUnitsQuery.data?.units.length ?? 0,
        mark: marks.find((mark) => mark.unit_root === selectedFolder.replace(/\/$/, "")) ?? null,
      }
    : selectedObject
      ? {
          kind: "file",
          object: selectedObject,
          state: sourceState(selectedObject.source_ref, facts),
          memberOfUnit: memberUnits.get(selectedObject.key) ?? null,
        }
      : null;

  const selectedFiles = useMemo(
    () => objects.filter((object) => checkedRefs.has(object.source_ref)),
    [checkedRefs, objects],
  );
  const processTarget = localSource ? localSource.file.name : selectedFiles.length
    ? `${selectedFiles.length} file(s)`
    : selectedFolder
      ? `folder ${selectedFolder}`
      : selectedObject
        ? selectedObject.name
        : null;

  // --- actions -------------------------------------------------------------
  const runSearch = useCallback(() => {
    const text = query.trim();
    if (!text) return;
    setAppliedFilter(text);
    setSelectedRef(null);
    setSelectedFolder(null);
    setSearchSummary(`Names and paths across this location: “${text}”.`);
  }, [query]);

  const clearSearch = useCallback(() => {
    setQuery("");
    setAppliedFilter("");
    setSearchSummary(null);
  }, []);

  /** Browse an exact source location with a bookmarkable URL and fresh selection.
   * Inputs: allowlisted root ID and relative prefix. Output: navigation; no evidence writes.
   * Use for folder/root clicks so Back and shared links return to the same location.
   */
  function browse(nextRoot: string, nextPrefix: string) {
    clearSearch();
    setLocalSource(null); setSelectedFolder(null); setSelectedRef(null); setCheckedRefs(new Set());
    void router.navigate({ href: `/sources?${new URLSearchParams({ root: nextRoot, prefix: nextPrefix, mode })}` });
  }

  /** Submit selected sources and expose the returned durable Activity link.
   * Input: selected files/folder and admitted case. Output: receipts/status in UI.
   * Effects: starts processing through the existing API; never copies or moves originals.
   * Byline: Codex · GPT-6 · 2026-10-06.
   */
  async function process() {
    if (!matter || !primaryCourtCase || processBusy.current) return;
    processBusy.current = true;
    setProcessing(true);
    setProcessError(null);
    setProcessMessage(null);
    try {
      if (localSource) {
        // Acquisition returns a verified canonical receipt; re-trying start must not upload twice.
        const sealed = localUploads.current.get(localSource.requestId) ?? await uploadProfferSource(localSource.file, mode, setUploadProgress);
        localUploads.current.set(localSource.requestId, sealed);
        const key = localSource.requestId;
        let entry = submissions.current.get(key);
        if (!entry) {
          entry = { request: {
            request_id: key, source_ref: sealed.acquisition_ref,
            declared_format: detectedFormat(localSource.file.name),
            parser_options_ref: "pending-handler-selection/v1",
            matter_id: matter.id, court_case_id: primaryCourtCase.id, matter_mode: mode,
          } };
          submissions.current.set(key, entry);
        }
        const result = await processSelection([entry], startProffer, (accepted) => {
          setActivityHref(`/activity?${new URLSearchParams({ preview_handle: accepted.response!.preview_handle, mode })}`);
        });
        setProcessMessage(result.accepted ? `${localSource.file.name} submitted.` : "File received; processing has not started.");
        setProcessError(result.error);
        return;
      }
      const files = selectedFiles.length ? selectedFiles : selectedObject ? [selectedObject] : [];
      if (files.length) {
        const entries = files.map((object) => {
          const key = JSON.stringify([matter.id, primaryCourtCase.id, mode, object.source_ref, handlerOverride]);
          let entry = submissions.current.get(key);
          if (entry) return entry;
          entry = { request: {
            request_id: `proffer-${matter.id}-${crypto.randomUUID()}`,
            source_ref: object.source_ref,
            declared_format: handlerOverride || detectedFormat(object.name),
            parser_options_ref: "pending-handler-selection/v1",
            matter_id: matter.id,
            court_case_id: primaryCourtCase.id,
            matter_mode: mode,
          } };
          submissions.current.set(key, entry);
          return entry;
        });
        const result = await processSelection(entries, startProffer, (entry) => {
          const handle = entry.response!.preview_handle;
          setActivityHref(`/activity?${new URLSearchParams({ preview_handle: handle, mode })}`);
        });
        setProcessMessage(`${result.accepted} of ${files.length} files submitted.`);
        if (result.error) setProcessError(`${result.error} Retry keeps the accepted files and resends only the unfinished requests.`);
        void runsQuery.refetch();
      } else if (selectedFolder && listing) {
        const root = listing.available_roots.find((entry) => entry.root_id === activeRootId);
        if (!root) throw new Error("This folder has no confirmed source location.");
        const folderRef = `${root.root_ref.replace(/\/$/, "")}/${selectedFolder.replace(/^\//, "")}`;
        const key = JSON.stringify([matter.id, primaryCourtCase.id, mode, folderRef, handlerOverride]);
        const batchId = batchRequests.current.get(key) ?? newBatchId();
        batchRequests.current.set(key, batchId);
        const started = await startProfferBatch({
          batch_id: batchId,
          matter_id: matter.id,
          court_case_id: primaryCourtCase.id,
          folder_ref: folderRef,
          declared_format: handlerOverride || detectedFormat(selectedFolder),
          parser_options_ref: "pending-handler-selection/v1",
          matter_mode: mode,
        });
        setBatchIdent(started.batch_id);
        setActivityHref(`/activity?${new URLSearchParams({ batch: started.batch_id, mode })}`);
        setProcessMessage(`Batch started for ${selectedFolder}.`);
      }
    } catch (error) {
      setProcessError(errorText(error));
    } finally {
      processBusy.current = false;
      setProcessing(false);
      setUploadProgress(null);
    }
  }

  async function markUnit(confirm: boolean) {
    if (!selectedFolder) return;
    setMarkPending(true);
    setMarkError(null);
    try {
      const mark = await recordSourceUnitMark({ unit_root: selectedFolder, unit_type: markKind, confirm });
      setLocalMarks((current) => [mark, ...current.filter((item) => item.unit_root !== mark.unit_root)]);
    } catch (error) {
      setMarkError(errorText(error));
    } finally {
      setMarkPending(false);
    }
  }

  const listingError = listingQuery.isError && !listingQuery.data ? errorText(listingQuery.error) : null;
  const nextPageError = listingQuery.isFetchNextPageError ? errorText(listingQuery.error) : null;
  const loadNextPage = () => {
    if (continuation.token && !listingQuery.isFetchingNextPage) void listingQuery.fetchNextPage();
  };

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="sources-screen">
      <SourceSearch
        query={query}
        searching={listingQuery.isFetching}
        onQueryChange={setQuery}
        onSubmit={runSearch}
        onClear={clearSearch}
        resultSummary={searchSummary}
      />

      <div className="flex flex-wrap items-center gap-2 border-b bg-card px-3 py-2">
        <Button size="sm" variant={showContentSearch ? "default" : "outline"}
          aria-expanded={showContentSearch} onClick={() => setShowContentSearch((value) => !value)}>
          {showContentSearch ? "Back to files" : "Search content and relationships"}
        </Button>
        <input ref={fileInput} type="file" className="sr-only" aria-label="Choose a file to add" disabled={processing}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) {
              setLocalSource({ file, requestId: `proffer-${matter?.id ?? "upload"}-${crypto.randomUUID()}` });
              setProcessMessage(null); setProcessError(null); setActivityHref(null);
            }
            event.target.value = "";
          }} />
        <Button size="sm" variant="outline" disabled={processing} onClick={() => fileInput.current?.click()}><Upload className="h-3.5 w-3.5" /> Add file</Button>
        <Button
          size="sm"
          disabled={processing || !processTarget || !matter || !primaryCourtCase}
          onClick={() => void process()}
        >
          {processing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />} Process
        </Button>
        <span className="text-[11px] text-muted-foreground">
          {uploadProgress !== null ? `Uploading ${uploadProgress}%` : processTarget ? `Ready: ${processTarget}` : "Select files or a folder."}
        </span>
        {localSource && <Button size="icon" variant="ghost" aria-label="Clear local file selection" disabled={processing} onClick={() => setLocalSource(null)}><X className="h-3.5 w-3.5" /></Button>}
        {processMessage && <span className="text-[11px] text-[#17794b] dark:text-[#72d9a1]">{processMessage}</span>}
        {activityHref && <Button asChild variant="outline" size="sm"><AppLink href={activityHref}>View activity</AppLink></Button>}
        {processError && (
          <span className="flex items-center gap-1 text-[11px] text-[#8f302a] dark:text-[#ffb5ae]" role="alert">
            <AlertTriangle className="h-3 w-3" /> {processError}
          </span>
        )}
      </div>

      <PanelGroup direction="horizontal" className="min-h-0 flex-1" autoSaveId="sources-layout">
        <Panel defaultSize={22} minSize={14}>
          <SourceTree
            roots={listing?.available_roots ?? []}
            activeRootId={activeRootId}
            prefix={prefix}
            prefixes={prefixes}
            loading={listingQuery.isPending}
            unitRoots={unitRoots}
            selectedFolder={selectedFolder}
            onRootChange={(nextRoot) => {
              browse(nextRoot, "");
            }}
            onPrefixChange={(nextPrefix) => {
              browse(activeRootId, nextPrefix);
            }}
            onSelectFolder={(nextFolder) => {
              setLocalSource(null);
              setCheckedRefs(new Set());
              setSelectedFolder(nextFolder);
              setSelectedRef(null);
              setUnitMarkKind("");
            }}
          />
        </Panel>
        <PanelResizeHandle className="w-1 bg-border hover:bg-primary" />
        <Panel defaultSize={52} minSize={26}>
          <div className="flex h-full min-h-0 flex-col">
            {listingError && (
              <p className="flex items-center gap-1 border-b px-3 py-2 text-[11px] text-[#8f302a] dark:text-[#ffb5ae]" role="alert">
                <AlertTriangle className="h-3 w-3" /> {listingError}
              </p>
            )}
            {showContentSearch ? (
              <div className="min-h-0 flex-1 overflow-auto p-3"><ContextSearch /></div>
            ) : rows.length ? (
              <>
                <div className="min-h-0 flex-1">
                  <SourceRowsGrid
                    rows={rows}
                    selectedIndex={selectedIndex}
                    checkedRefs={checkedRefs}
                    onSelectIndex={(index) => {
                      setLocalSource(null);
                      setSelectedRef(rows[index]?.sourceRef ?? null);
                      setSelectedFolder(null);
                      setHandlerOverride("");
                    }}
                    onToggleChecked={(refs) => { setLocalSource(null); setCheckedRefs(new Set(refs)); }}
                    onReachEnd={() => { if (!nextPageError) loadNextPage(); }}
                  />
                </div>
                <div className="flex items-center justify-between gap-2 border-t px-3 py-1 text-[11px]" role="status">
                  <span>
                    {nextPageError || continuation.issue ||
                      (listingQuery.isFetchingNextPage ? "Loading more files…" :
                        continuation.complete ? `All ${rows.length} files in this location loaded.` :
                          `${rows.length} files loaded; more available.`)}
                  </span>
                  {(continuation.token || continuation.issue) && (
                    <Button size="sm" variant="outline" disabled={listingQuery.isFetchingNextPage}
                      onClick={() => { if (continuation.issue) void listingQuery.refetch(); else loadNextPage(); }}>
                      {continuation.issue ? "Refresh" : nextPageError ? "Retry" : "Load more"}
                    </Button>
                  )}
                </div>
              </>
            ) : (
              <div className="space-y-2 px-3 py-4 text-xs text-muted-foreground">
                <p>{listingQuery.isPending ? "Loading files" : "No files on this page."}</p>
                {listingError && <Button size="sm" variant="outline" onClick={() => void listingQuery.refetch()}>Retry</Button>}
                {continuation.token && (
                  <Button size="sm" variant="outline" disabled={listingQuery.isFetchingNextPage}
                    onClick={loadNextPage}>
                    {listingQuery.isFetchingNextPage ? "Loading more files…" : nextPageError ? "Retry" : "Load more"}
                  </Button>
                )}
                {(nextPageError || continuation.issue) && <p role="alert">{nextPageError || continuation.issue}</p>}
                {continuation.issue && <Button size="sm" variant="outline" onClick={() => void listingQuery.refetch()}>Refresh</Button>}
              </div>
            )}
          </div>
        </Panel>
        <PanelResizeHandle className="w-1 bg-border hover:bg-primary" />
        <Panel defaultSize={26} minSize={18}>
          <SourceMetadataPanel
            selection={selection}
            inspection={inspectionQuery.data ?? null}
            inspectionLoading={inspectionQuery.isFetching}
            inspectionError={inspectionQuery.error ? errorText(inspectionQuery.error) : null}
            manifest={manifestQuery.data ?? null}
            runs={selectedObject ? runsIndex.get(selectedObject.source_ref) ?? [] : []}
            provenance={provenanceQuery.data ?? null}
            provenanceMissing={Boolean(provenanceQuery.error) || provenanceQuery.data?.items.length === 0}
            handlerOverride={handlerOverride}
            onHandlerOverrideChange={setHandlerOverride}
            unitProposal={proposal}
            unitMarkKind={markKind}
            onUnitMarkKindChange={setUnitMarkKind}
            onMarkUnit={(confirm) => void markUnit(confirm)}
            markPending={markPending}
            markError={markError}
            markStorage={marksQuery.data?.storage ?? "the Workbench data volume"}
            batch={batchQuery.data ?? null}
            canonicalHomeOffered
          />
        </Panel>
      </PanelGroup>
    </div>
  );
}

function findUnit(units: readonly CatalogUnit[], folder: string): CatalogUnit | null {
  const trimmed = folder.replace(/\/$/, "");
  return (
    units.find(
      (unit) =>
        unit.unit_root.replace(/\/$/, "") === trimmed || (unit.export_root ?? "").replace(/\/$/, "") === trimmed,
    ) ?? null
  );
}
