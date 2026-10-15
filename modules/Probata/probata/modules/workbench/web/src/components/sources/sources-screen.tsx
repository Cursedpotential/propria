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
import { HumanProcessStatus } from "@/components/sources/human-process-status";
import { humanContextAttribution, isSMSContextSource, type HumanContextProgress, type HumanContextSourceRequest } from "@/lib/human-context";
import { mergeSourcePages, sourceContinuation } from "@/components/sources/source-pages";
import { pinnedSourceRef, restoreSourceNavigation, saveSourceNavigation } from "@/components/sources/source-navigation";
import { SourceRowsGrid, type SourceGridRow } from "@/components/sources/source-rows-grid";
import { SourceSearch } from "@/components/sources/source-search";
import { ContextSearch } from "@/components/read/context-search";
import { SourceTree } from "@/components/sources/source-tree";
import { detectedFormat, runsBySource, sourceRunHistory, sourceState } from "@/components/sources/source-state";
import { Button } from "@/components/ui/button";
import { AppLink, useBrowserSearchParams } from "@/lib/router-compat";
import { processSelection, type SourceSubmission } from "@/components/sources/process-selection";
import { confirmedCaseOwner, processingIdentity, requiresMessagePerspective, sourcePerspectiveScope, type SourcePerspectiveChoice } from "@/components/sources/source-perspective";
import { getCaseIdentity } from "@/lib/case-identity-client";
import {
  ApiError,
  getCatalogProvenance,
  getDecodedExists,
  getProfferBatch,
  getProfferSourceVersion,
  getAIContextSourceStatus,
  getUnitsUnderPrefix,
  previewProfferSource,
  listProfferProposalResources,
  listProfferSources,
  listSourceUnitMarks,
  lookupCatalogUnits,
  proposeSourceUnit,
  recordSourceUnitMark,
  startProffer,
  startHumanContextSource,
  startAIContextSource,
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
  const searchParams = useBrowserSearchParams();
  const rootId = searchParams.get("root") ?? "";
  const linkedFile = searchParams.get("file") ?? "";
  return <SourcesScreenMode key={`${mode}:${rootId}:${searchParams.get("prefix") ?? ""}:${linkedFile}`} />;
}

function SourcesScreenMode() {
  const { matter, primaryCourtCase, mode } = useFixedCase();
  const searchParams = useBrowserSearchParams();
  const router = useRouter();

  const rootId = searchParams.get("root") ?? "";
  const prefix = searchParams.get("prefix") ?? "";
  const linkedFile = searchParams.get("file") ?? "";
  const navigationKey = JSON.stringify([mode, rootId, prefix, linkedFile]);
  const [restored] = useState(() => restoreSourceNavigation(navigationKey));
  const [appliedFilter, setAppliedFilter] = useState(restored?.appliedFilter ?? linkedFile);
  const [selectedRef, setSelectedRef] = useState<string | null>(restored ? restored.selectedRef : linkedFile || null);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(restored ? restored.selectedFolder : searchParams.get("select_folder") === "1" ? prefix : null);
  const [checkedRefs, setCheckedRefs] = useState<ReadonlySet<string>>(() => new Set(restored?.checkedRefs ?? []));
  const [scrollRow, setScrollRow] = useState(restored?.scrollRow ?? 0);
  const [handlerOverride, setHandlerOverride] = useState("");
  const [perspectiveChoice, setPerspectiveChoice] = useState<SourcePerspectiveChoice | null>(null);
  const [aiProvider, setAIProvider] = useState<"" | "chatgpt" | "claude" | "gemini_markdown">(restored?.aiProvider ?? "");
  const [aiExportSelection, setAIExportSelection] = useState<string | null>(restored?.aiExportSelection ?? null);
  const [aiPending, setAIPending] = useState(false);
  const [aiError, setAIError] = useState<string | null>(null);
  const [aiWorkflow, setAIWorkflow] = useState<{ workflow_id: string; run_id: string; source_ref: string } | null>(null);
  const aiRequests = useRef(new Map<string, string>());

  const [query, setQuery] = useState(restored?.query ?? linkedFile);
  const [showContentSearch, setShowContentSearch] = useState(false);
  const [searchSummary, setSearchSummary] = useState<string | null>(linkedFile ? `Recorded file location: ${linkedFile}` : null);

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
  const humanSubmissions = useRef(new Map<string, { key: string; response?: HumanContextProgress }>());
  const [humanWorkflows, setHumanWorkflows] = useState<Array<{ workflowID: string; name: string }>>([]);
  const batchRequests = useRef(new Map<string, string>());
  const processBusy = useRef(false);
  const [activityHref, setActivityHref] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const [localSource, setLocalSource] = useState<{ file: File; requestId: string } | null>(null);
  const localUploads = useRef(new Map<string, ProfferUploadResponse>());
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);

  // --- reads ---------------------------------------------------------------
  const identityQuery = useQuery({
    queryKey: ["sources", "case-identity", mode],
    queryFn: () => getCaseIdentity(mode),
    retry: false,
  });
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
    () => restored?.selectedRef === selectedRef && restored.selectedObject
      ? restored.selectedObject : objects.find((object) => object.source_ref === selectedRef
        || (selectedRef === linkedFile && linkedFile && (object.key === linkedFile || object.name === linkedFile))) ?? null,
    [objects, restored, selectedRef, linkedFile],
  );
  const activeRootId = rootId || listing?.active_root_id || "";



  const aiStatusQuery = useQuery({
    queryKey: ["sources", "ai-context", aiWorkflow?.workflow_id],
    queryFn: () => getAIContextSourceStatus(aiWorkflow!.workflow_id),
    enabled: Boolean(aiWorkflow),
    retry: false,
    refetchInterval: (query) => query.state.data && !["complete", "partial", "needs_parser", "failed"].includes(String(query.state.data.status)) ? 3000 : false,
  });

  const inspectionQuery = useQuery({
    queryKey: ["sources", "inspection", mode, activeRootId, selectedObject?.source_ref, selectedObject?.etag, restored?.selectedRef === selectedRef ? restored.providerVersion : null],
    queryFn: ({ signal }) => previewProfferSource(selectedObject!, mode, activeRootId, signal, restored?.selectedRef === selectedRef ? restored.providerVersion : null),
    enabled: Boolean(selectedObject && activeRootId),
    retry: false,
  });

  const sourceVersionQuery = useQuery({
    queryKey: ["sources", "provider-version", activeRootId, selectedObject?.source_ref, selectedObject?.etag, inspectionQuery.data?.provider_version_id],
    queryFn: ({ signal }) => getProfferSourceVersion(selectedObject!, activeRootId, signal, inspectionQuery.data?.provider_version_id),
    enabled: Boolean(aiProvider && selectedObject && aiExportSelection === selectedObject.source_ref && activeRootId && inspectionQuery.data),
    retry: false,
  });

  useEffect(() => {
    saveSourceNavigation(navigationKey, {
      selectedRef: selectedObject?.source_ref ?? selectedRef, selectedFolder, checkedRefs: [...checkedRefs], selectedObject,
      providerVersion: inspectionQuery.data?.provider_version_id ?? (restored?.selectedRef === selectedRef ? restored.providerVersion : null),
      query, appliedFilter, scrollRow,
      aiExportSelection, aiProvider,
    });
  }, [navigationKey, selectedRef, selectedFolder, checkedRefs, selectedObject, inspectionQuery.data, restored, query, appliedFilter, scrollRow, aiExportSelection, aiProvider]);

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
        state: sourceState(object.source_ref, facts, inspectionQuery.data?.source_ref === object.source_ref ? inspectionQuery.data.provider_version_id : undefined),
        unitMark: memberUnits.get(object.key)?.replaceAll("_", " ") ?? "",
      })),
    [facts, memberUnits, objects, inspectionQuery.data],
  );
  const selectedIndex = rows.findIndex((row) => row.sourceRef === (selectedObject?.source_ref ?? selectedRef));

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
          state: sourceState(selectedObject.source_ref, facts, inspectionQuery.data?.provider_version_id),
          memberOfUnit: memberUnits.get(selectedObject.key) ?? null,
        }
      : null;

  const selectedFiles = useMemo(
    () => objects.filter((object) => checkedRefs.has(object.source_ref)),
    [checkedRefs, objects],
  );
  const processTarget = showContentSearch ? null : localSource ? localSource.file.name : selectedFiles.length
    ? `${selectedFiles.length} file(s)`
    : selectedFolder
      ? `folder ${selectedFolder} · recursive scope, at most 2,000 files per batch, derived outputs excluded; exact count unknown until inventory`
      : selectedObject
        ? selectedObject.name
        : null;
  const processFiles = selectedFiles.length ? selectedFiles : selectedObject ? [selectedObject] : [];
  const unsupportedMedia = [localSource?.file.name, ...processFiles.map((object) => object.name)].filter((name): name is string => Boolean(name))
    .some((name) => /\.(mp4|m4a|mp3|wav|ogg|webm|mov|avi|flac|aac)$/i.test(name) && detectedFormat(name) === "unknown_binary");
  const hasAIProcessTarget = !localSource && processFiles.some((object) => object.source_ref === aiExportSelection);
  const aiProcessBlocked = hasAIProcessTarget && (processFiles.length !== 1 || selectedObject?.source_ref !== aiExportSelection
    || !aiProvider || !sourceVersionQuery.data || sourceVersionQuery.isFetching || aiPending);
  const hasNativeProcessTarget = hasAIProcessTarget || (localSource
    ? isSMSContextSource(localSource.file.name, handlerOverride || detectedFormat(localSource.file.name))
    : processFiles.some((object) => isSMSContextSource(object.name, handlerOverride || detectedFormat(object.name))));
  const needsPerspective = localSource
    ? requiresMessagePerspective(localSource.file.name, handlerOverride || detectedFormat(localSource.file.name))
    : processFiles.some((object) => requiresMessagePerspective(object.name, handlerOverride || detectedFormat(object.name)));
  const canChooseAIProvider = Boolean(selectedObject && ["json", "zip", "md", "markdown"].includes(selectedObject.name.split(".").pop()?.toLowerCase() ?? ""));
  const perspectiveScope = sourcePerspectiveScope(mode, localSource
    ? [[localSource.requestId, null, localSource.file.size]]
    : processFiles.map((object) => [object.source_ref, object.etag, object.byte_length]));
  const registryPeople = identityQuery.data?.people ?? [];
  const registryOwner = confirmedCaseOwner(registryPeople);

  // --- actions -------------------------------------------------------------
  const runSearch = useCallback(() => {
    const text = query.trim();
    if (!text) return;
    setAppliedFilter(text);
    setPerspectiveChoice(null);
    setSelectedRef(null);
    setSelectedFolder(null);
    setCheckedRefs(new Set()); setLocalSource(null); setScrollRow(0);
    setSearchSummary(`Names and paths across the entire selected location: “${text}”. Browsed folder: ${prefix || "root"}.`);
  }, [query, prefix]);

  const clearSearch = useCallback(() => {
    setQuery("");
    setAppliedFilter("");
    setSearchSummary(null);
    setSelectedRef(null); setSelectedFolder(null); setCheckedRefs(new Set()); setScrollRow(0);
  }, []);

  /** Browse an exact source location with a bookmarkable URL and fresh selection.
   * Inputs: allowlisted root ID and relative prefix. Output: navigation; no evidence writes.
   * Use for folder/root clicks so Back and shared links return to the same location.
   */
  function browse(nextRoot: string, nextPrefix: string) {
    clearSearch();
    setPerspectiveChoice(null);
    setLocalSource(null); setSelectedFolder(null); setSelectedRef(null); setCheckedRefs(new Set());
    void router.navigate({ href: `/sources?${new URLSearchParams({ root: nextRoot, prefix: nextPrefix, mode })}` });
  }

  /** Submit a human source once while preserving accepted execution IDs across transport retries.
   * Inputs: native source/version fields, selection key and display filename. Output: tracked real workflow.
   * Effects: context POST and tracking state. Choose for SMS Process before legacy handler selection.
   */
  async function submitHumanSource(request: HumanContextSourceRequest, selectionKey: string, name: string) {
    let entry = humanSubmissions.current.get(selectionKey);
    if (!entry) {
      entry = { key: crypto.randomUUID() };
      humanSubmissions.current.set(selectionKey, entry);
    }
    if (!entry.response) {
      const response = await startHumanContextSource(request, entry.key);
      if (!response.workflow_id || !response.run_id) throw new Error("The context starter returned no processing execution IDs.");
      entry.response = response;
    }
    const workflowID = entry.response.workflow_id;
    setHumanWorkflows((current) => current.some((item) => item.workflowID === workflowID) ? current : [...current, { workflowID, name }]);
  }

  /** Submit selected sources and expose returned durable processing links.
   * Input: selected files/folder, explicit source kind/provider or export attribution and optional case for native context.
   * Output: per-file receipts/status; legacy sources still require the admitted case.
   * Effects: explicitly selected AI exports use native context to Weaviate; other sources use their existing API. No original moves.
   * Byline: Codex · GPT-6 · 2026-10-06.
   */
  async function process() {
    if (processBusy.current || showContentSearch || !processTarget || unsupportedMedia || aiProcessBlocked) return;
    if (hasAIProcessTarget) {
      await addAIChatToContext();
      return;
    }
    processBusy.current = true;
    setProcessing(true);
    setProcessError(null);
    setProcessMessage(null);
    try {
      const humanCase = {
        ...(matter?.id ? { matter_id: matter.id } : {}),
        ...(primaryCourtCase?.id ? { court_case_id: primaryCourtCase.id } : {}),
      };
      if (localSource) {
        const native = isSMSContextSource(localSource.file.name, handlerOverride || detectedFormat(localSource.file.name));
        if (!native && (!matter || !primaryCourtCase)) throw new Error(`${localSource.file.name}: legacy processing needs a fixed case.`);
        const humanAttribution = native ? humanContextAttribution(registryPeople, perspectiveScope, perspectiveChoice) : undefined;
        const identity = native ? {} : processingIdentity(registryPeople, perspectiveScope, perspectiveChoice, requiresMessagePerspective(localSource.file.name, detectedFormat(localSource.file.name)));
        // Acquisition returns a verified canonical receipt; re-trying start must not upload twice.
        const sealed = localUploads.current.get(localSource.requestId) ?? await uploadProfferSource(localSource.file, mode, setUploadProgress);
        localUploads.current.set(localSource.requestId, sealed);
        if (native) {
          const key = JSON.stringify([localSource.requestId, humanCase, mode, humanAttribution]);
          await submitHumanSource({
            contract_version: "context-v1", source_kind: "sms", source_ref: sealed.acquisition_ref,
            declared_format: "smsbackuprestore_xml", ...humanCase,
            byte_length: sealed.byte_length, ...humanAttribution,
          }, key, localSource.file.name);
          setProcessMessage(`${localSource.file.name} submitted for context processing.`);
          return;
        }
        if (!matter || !primaryCourtCase) throw new Error(`${localSource.file.name}: legacy processing needs a fixed case.`);
        const key = JSON.stringify([localSource.requestId, matter.id, primaryCourtCase.id, mode, identity]);
        let entry = submissions.current.get(key);
        if (!entry) {
          entry = { request: {
            request_id: `proffer-${matter.id}-${crypto.randomUUID()}`, source_ref: sealed.acquisition_ref,
            ...identity,
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
        let accepted = 0;
        const submissionErrors: string[] = [];
        const outcomes: string[] = [];
        for (const object of files) {
          const format = handlerOverride || detectedFormat(object.name);
          try {
            if (isSMSContextSource(object.name, format)) {
              const humanAttribution = humanContextAttribution(registryPeople, perspectiveScope, perspectiveChoice);
              const key = JSON.stringify([humanCase, mode, object.source_ref, object.etag, object.byte_length, handlerOverride, humanAttribution]);
              if (humanSubmissions.current.get(key)?.response) { accepted += 1; outcomes.push(`${object.name}: already submitted`); continue; }
              const version = await getProfferSourceVersion(object, activeRootId, undefined,
                object.source_ref === selectedObject?.source_ref ? inspectionQuery.data?.provider_version_id ?? restored?.providerVersion : undefined);
              await submitHumanSource({
                contract_version: "context-v1", source_kind: "sms", source_ref: version.source_ref,
                provider_version_id: version.provider_version_id, declared_format: "smsbackuprestore_xml",
                ...humanCase,
                byte_length: version.byte_length, etag: version.etag, ...humanAttribution,
              }, key, object.name);
              accepted += 1;
              outcomes.push(`${object.name}: submitted for context processing`);
              continue;
            }
            if (!matter || !primaryCourtCase) throw new Error("Legacy processing needs a fixed case.");
            const identity = processingIdentity(registryPeople, perspectiveScope, perspectiveChoice, requiresMessagePerspective(object.name, format));
            const inspected = inspectionQuery.data?.source_ref === object.source_ref ? inspectionQuery.data
              : await previewProfferSource(object, mode, activeRootId, undefined, object.source_ref === selectedObject?.source_ref ? restored?.providerVersion : undefined);
            const sourceRef = pinnedSourceRef(object.source_ref, inspected.provider_version_id);
            const key = JSON.stringify([matter.id, primaryCourtCase.id, mode, sourceRef, object.etag, object.byte_length, handlerOverride, identity]);
            let entry = submissions.current.get(key);
            if (!entry) {
              entry = { request: {
                request_id: `proffer-${matter.id}-${crypto.randomUUID()}`,
                source_ref: sourceRef,
                ...identity,
                declared_format: format,
                parser_options_ref: "pending-handler-selection/v1",
                matter_id: matter.id,
                court_case_id: primaryCourtCase.id,
                matter_mode: mode,
              } };
              submissions.current.set(key, entry);
            }
            const result = await processSelection([entry], startProffer, (completed) => {
              setActivityHref(`/activity?${new URLSearchParams({ preview_handle: completed.response!.preview_handle, mode })}`);
            });
            accepted += result.accepted;
            if (result.error) submissionErrors.push(`${object.name}: ${result.error}`);
            else outcomes.push(`${object.name}: submitted`);
          } catch (error) { submissionErrors.push(`${object.name}: ${errorText(error)}`); }
        }
        setProcessMessage(`${accepted} of ${files.length} files submitted. ${outcomes.join(" · ")}`);
        if (submissionErrors.length) setProcessError(`${submissionErrors.join(" · ")} Retry keeps the accepted files and resends only the unfinished requests.`);
        void runsQuery.refetch();
      } else if (selectedFolder && listing) {
        if (!matter || !primaryCourtCase) throw new Error("Legacy folder processing needs a fixed case.");
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

  /** Register a version-pinned AI export in Context through the Go engine.
   * Inputs: explicit provider, selected B2 object and metadata HEAD version.
   * Output: Temporal execution IDs and status. Effect: one idempotent Go start;
   * choose this instead of the case-bound Process path for AI-chat exports.
   */
  async function addAIChatToContext() {
    const version = sourceVersionQuery.data;
    if (!selectedObject || !version || !aiProvider || aiPending || aiExportSelection !== selectedObject.source_ref || version.source_ref !== selectedObject.source_ref) return;
    const identity = JSON.stringify([version.source_ref, version.provider_version_id, aiProvider]);
    const requestID = aiRequests.current.get(identity) ?? crypto.randomUUID();
    aiRequests.current.set(identity, requestID);
    setAIPending(true);
    setAIError(null);
    try {
      const started = await startAIContextSource({
        source_ref: version.source_ref,
        provider_version_id: version.provider_version_id,
        source_kind: "ai_chat",
        declared_format: aiProvider,
      }, requestID);
      setAIWorkflow({ ...started, source_ref: version.source_ref });
    } catch (error) {
      setAIError(errorText(error));
    } finally {
      setAIPending(false);
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
          aria-expanded={showContentSearch} onClick={() => {
            setSelectedRef(null); setSelectedFolder(null); setCheckedRefs(new Set()); setLocalSource(null); setPerspectiveChoice(null);
            setShowContentSearch((value) => !value);
          }}>
          {showContentSearch ? "Back to files" : "Search content and relationships"}
        </Button>
        <input ref={fileInput} type="file" className="sr-only" aria-label="Choose a file to add" disabled={processing}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) {
              setPerspectiveChoice(null);
              setLocalSource({ file, requestId: `proffer-${matter?.id ?? "upload"}-${crypto.randomUUID()}` });
              setProcessMessage(null); setProcessError(null); setActivityHref(null);
            }
            event.target.value = "";
          }} />
        <Button size="sm" variant="outline" disabled={processing} onClick={() => fileInput.current?.click()}><Upload className="h-3.5 w-3.5" /> Add file</Button>
        {needsPerspective && !showContentSearch && (
          <label className="flex items-center gap-2 text-[11px]">
            Whose phone/export?
            <select aria-label="Whose phone/export?" className="rounded border bg-background px-2 py-1"
              disabled={processing || identityQuery.isPending}
              value={perspectiveChoice?.scope === perspectiveScope ? perspectiveChoice.personId : ""}
              onChange={(event) => setPerspectiveChoice(event.target.value ? { scope: perspectiveScope, personId: event.target.value } : null)}>
              <option value="">Choose for selected file(s)</option>
              {registryOwner && <option value={registryOwner.id}>My phone/export</option>}
              {registryPeople.filter((person) => person.id !== registryOwner?.id && person.verification_state === "confirmed").map((person) => (
                <option key={person.id} value={person.id}>{person.display_name}’s phone/export</option>
              ))}
            </select>
            {identityQuery.error && <span role="alert">{errorText(identityQuery.error)}</span>}
          </label>
        )}
        <Button
          size="sm"
          disabled={processing || !processTarget || unsupportedMedia || aiProcessBlocked || (!hasNativeProcessTarget && (!matter || !primaryCourtCase))}
          onClick={() => void process()}
        >
          {processing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />} Process
        </Button>
        <span className="text-[11px] text-muted-foreground">
          {hasAIProcessTarget ? aiProcessBlocked ? "AI chat target: select one export and choose its provider; exact source version must be available before native context processing." : `Ready: ${processTarget} · native AI context route to Weaviate` : unsupportedMedia ? "Original media can be inspected; no registered audio/video extractor is available in this Process route." : uploadProgress !== null ? `Uploading ${uploadProgress}%` : processTarget ? `Ready: ${processTarget}` : "Select files or a folder."}
        </span>
        {processFiles.length > 0 && !showContentSearch && <details className="max-w-full text-[11px]"><summary className="cursor-pointer">Exact processing target ({processFiles.length} file{processFiles.length === 1 ? "" : "s"}; no folder recursion)</summary><ul className="max-h-40 overflow-auto">{processFiles.map((object) => <li className="break-all" key={object.source_ref}>{object.name} · {object.source_ref} · {object.byte_length} bytes · ETag {object.etag ?? "not supplied"}</li>)}</ul></details>}
        {localSource && <Button size="icon" variant="ghost" aria-label="Clear local file selection" disabled={processing} onClick={() => { setLocalSource(null); setPerspectiveChoice(null); }}><X className="h-3.5 w-3.5" /></Button>}
        {processMessage && <span className="text-[11px] text-[#17794b] dark:text-[#72d9a1]">{processMessage}</span>}
        {humanWorkflows.map((workflow) => <HumanProcessStatus key={workflow.workflowID} workflowID={workflow.workflowID} name={workflow.name} mode={mode} />)}
        {activityHref && <Button asChild variant="outline" size="sm"><AppLink href={activityHref}>View activity</AppLink></Button>}
        {processError && (
          <div className="text-[11px] text-[#8f302a] dark:text-[#ffb5ae]" role="alert">
            <p><AlertTriangle className="inline h-3 w-3" /> Processing request failed. Review the selected source and prerequisite details before submitting again.</p>
            <details><summary className="cursor-pointer">Processing failure details</summary><pre className="max-h-40 max-w-full overflow-auto whitespace-pre-wrap break-all">{processError}</pre></details>
          </div>
        )}
        {canChooseAIProvider && selectedObject && !selectedFolder && !localSource && selectedFiles.length === 0 && !showContentSearch && (
          <details className="text-[11px]"><summary className="cursor-pointer">Additional source handlers</summary>
          <label className="flex items-center gap-2 py-1"><input type="checkbox" checked={aiExportSelection === selectedObject.source_ref} onChange={(event) => { setAIExportSelection(event.target.checked ? selectedObject.source_ref : null); setAIProvider(""); }} />This selected file is an AI chat export</label>
          {aiExportSelection === selectedObject.source_ref && (
          <div className="flex flex-wrap items-center gap-2 border-l pl-2 text-[11px]">
            <label htmlFor="ai-chat-provider">AI chat export</label>
            <select id="ai-chat-provider" aria-label="AI chat provider" className="rounded border bg-background px-2 py-1"
              value={aiProvider} onChange={(event) => { setAIProvider(event.target.value as "" | "chatgpt" | "claude" | "gemini_markdown"); setAIError(null); setAIWorkflow(null); }}>
              <option value="">Choose provider</option>
              <option value="chatgpt">ChatGPT</option>
              <option value="claude">Claude</option>
              <option value="gemini_markdown">Gemini Markdown</option>
            </select>
            {aiProvider && (
              <Button size="sm" variant="outline" disabled={aiPending || !sourceVersionQuery.data || sourceVersionQuery.isFetching}
                onClick={() => void addAIChatToContext()}>
                {aiPending ? "Starting…" : "Add AI chat to context"}
              </Button>
            )}
            {aiProvider && sourceVersionQuery.isFetching && <span>Checking B2 version…</span>}
            {aiProvider && sourceVersionQuery.error && <span role="alert">{errorText(sourceVersionQuery.error)}</span>}
            {aiError && <span role="alert">{aiError}</span>}
            {aiWorkflow?.source_ref === selectedObject.source_ref && (
              <span role="status">Workflow {aiWorkflow.workflow_id}: {String(aiStatusQuery.data?.status ?? "submitted")}
                {aiStatusQuery.data?.reason ? ` · ${String(aiStatusQuery.data.reason)}` : ""}
                {aiStatusQuery.error ? ` (${errorText(aiStatusQuery.error)})` : ""}
                <AppLink className="ml-3 underline underline-offset-4" href={`/read?context_workflow=${encodeURIComponent(aiWorkflow.workflow_id)}`}>Review extracted context</AppLink>
              </span>
            )}
          </div>
          )}
          </details>
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
              if (nextFolder !== prefix) {
                void router.navigate({ href: `/sources?${new URLSearchParams({ root: activeRootId, prefix: nextFolder, select_folder: "1", mode })}` });
              } else {
                clearSearch(); setPerspectiveChoice(null); setLocalSource(null); setCheckedRefs(new Set()); setSelectedFolder(nextFolder); setSelectedRef(null); setUnitMarkKind("");
              }
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
                      setPerspectiveChoice(null);
                      setLocalSource(null);
                      setSelectedRef(rows[index]?.sourceRef ?? null);
                      setSelectedFolder(null);
                      setHandlerOverride("");
                    }}
                    onToggleChecked={(refs) => { setPerspectiveChoice(null); setLocalSource(null); setCheckedRefs(new Set(refs)); }}
                    onReachEnd={() => { if (!appliedFilter && !nextPageError) loadNextPage(); }}
                    initialScrollRow={restored?.scrollRow ?? 0}
                    onScrollRowChange={setScrollRow}
                  />
                </div>
                <div className="flex items-center justify-between gap-2 border-t px-3 py-1 text-[11px]" role="status">
                  <span>
                    {nextPageError || continuation.issue ||
                      (listingQuery.isFetchingNextPage ? "Loading more files…" : appliedFilter ?
                        `${rows.length} matches found${pages?.at(-1)?.search_complete ? "; search complete" : " so far; continue search to scan more"}. ${pages?.reduce((sum, page) => sum + page.scanned_count, 0) ?? 0} objects scanned.${pages?.at(-1)?.scan_limit_reached ? " Scan limit reached for this request." : ""}` :
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
                <p role="status">{listingQuery.isPending ? "Loading files" : appliedFilter ?
                  `No matches found${pages?.at(-1)?.search_complete ? "; search complete" : " so far"}. Searching names and paths across the entire selected location, not only the browsed folder.` : "No files on this page."}</p>
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
            onRetryInspection={() => void inspectionQuery.refetch()}
            manifest={manifestQuery.data ?? null}
            runs={selectedObject ? sourceRunHistory(selectedObject.source_ref, facts, inspectionQuery.data?.provider_version_id).exact : []}
            unpinnedRuns={selectedObject ? sourceRunHistory(selectedObject.source_ref, facts, inspectionQuery.data?.provider_version_id).unpinned : []}
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
