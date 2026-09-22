// Byline: Claude Code · Opus 5 · 2026-09-22
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

import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Loader2, Play } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";

import { SourceMetadataPanel, type SourceSelection } from "@/components/sources/source-metadata-panel";
import { SourceRowsGrid, type SourceGridRow } from "@/components/sources/source-rows-grid";
import { SourceSearch, modeAvailable, type SearchMode } from "@/components/sources/source-search";
import { SourceTree } from "@/components/sources/source-tree";
import { detectedFormat, runsBySource, sourceState } from "@/components/sources/source-state";
import { Button } from "@/components/ui/button";
import {
  ApiError,
  getCatalogProvenance,
  getDecodedExists,
  getDiscoveryCapabilities,
  getProfferBatch,
  getUnitsUnderPrefix,
  inspectProfferSource,
  listProfferProposalResources,
  listProfferSources,
  listSourceUnitMarks,
  lookupCatalogUnits,
  proposeSourceUnit,
  recordSourceUnitMark,
  searchDiscovery,
  startProffer,
  startProfferBatch,
} from "@/lib/api-client";
import { getDecodedManifest } from "@/lib/decoded-source-client";
import type { DiscoveryItem } from "@/lib/discovery-types";
import { useFixedCase } from "@/lib/fixed-case-context";
import type { CatalogUnitLookup, SourceUnitKind, SourceUnitMark } from "@/lib/shared/types";

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

  const [rootId, setRootId] = useState("");
  const [prefix, setPrefix] = useState("");
  const [appliedFilter, setAppliedFilter] = useState("");
  const [selectedRef, setSelectedRef] = useState<string | null>(null);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [checkedRefs, setCheckedRefs] = useState<ReadonlySet<string>>(() => new Set());
  const [handlerOverride, setHandlerOverride] = useState("");

  const [query, setQuery] = useState("");
  const [searchMode, setSearchMode] = useState<SearchMode>("names");
  const [indexQuery, setIndexQuery] = useState<{ text: string; mode: SearchMode } | null>(null);
  const [searchSummary, setSearchSummary] = useState<string | null>(null);

  const [unitMarkKind, setUnitMarkKind] = useState<SourceUnitKind | "">("");
  const [markPending, setMarkPending] = useState(false);
  const [markError, setMarkError] = useState<string | null>(null);
  const [localMarks, setLocalMarks] = useState<SourceUnitMark[]>([]);

  const [batchIdent, setBatchIdent] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const [processMessage, setProcessMessage] = useState<string | null>(null);
  const [processError, setProcessError] = useState<string | null>(null);

  // --- reads ---------------------------------------------------------------
  const listingQuery = useQuery({
    queryKey: ["sources", "listing", mode, rootId, prefix, appliedFilter],
    queryFn: ({ signal }) =>
      listProfferSources({
        mode,
        rootId: rootId || undefined,
        prefix: appliedFilter ? "" : prefix,
        filter: appliedFilter || undefined,
        pageSize: 200,
        signal,
      }),
  });
  const listing = listingQuery.data ?? null;
  const objects = useMemo(() => listing?.objects ?? [], [listing]);
  const prefixes = useMemo(() => listing?.prefixes ?? [], [listing]);

  const runsQuery = useQuery({
    queryKey: ["sources", "runs", mode],
    queryFn: ({ signal }) => listProfferProposalResources(mode, { limit: 100 }, signal),
  });
  const runsIndex = useMemo(() => runsBySource(runsQuery.data?.items ?? []), [runsQuery.data]);

  const capabilitiesQuery = useQuery({
    queryKey: ["sources", "discovery-capabilities"],
    queryFn: ({ signal }) => getDiscoveryCapabilities(signal),
    retry: false,
  });

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

  const indexSearchQuery = useQuery({
    queryKey: ["sources", "index-search", indexQuery?.text, indexQuery?.mode],
    queryFn: ({ signal }) =>
      searchDiscovery(indexQuery!.text, "", indexQuery!.mode === "contents" ? "contents" : "hybrid", undefined, signal),
    enabled: Boolean(indexQuery),
    retry: false,
  });

  const selectedObject = useMemo(
    () => objects.find((object) => object.source_ref === selectedRef) ?? null,
    [objects, selectedRef],
  );
  const activeRootId = rootId || listing?.active_root_id || "";

  const inspectionQuery = useQuery({
    queryKey: ["sources", "inspection", mode, activeRootId, selectedObject?.source_ref],
    queryFn: ({ signal }) => inspectProfferSource(selectedObject!, mode, activeRootId, signal),
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
  const processTarget = selectedFiles.length
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
    if (searchMode === "names") {
      setIndexQuery(null);
      setAppliedFilter(text);
      setSelectedRef(null);
      setSelectedFolder(null);
      setSearchSummary(`Names and paths across this location: “${text}”.`);
      return;
    }
    if (!modeAvailable(searchMode, capabilitiesQuery.data ?? null)) {
      setIndexQuery(null);
      setSearchSummary(null);
      return;
    }
    setAppliedFilter("");
    setIndexQuery({ text, mode: searchMode });
    setSearchSummary(`Index search: “${text}”.`);
  }, [capabilitiesQuery.data, query, searchMode]);

  const clearSearch = useCallback(() => {
    setQuery("");
    setAppliedFilter("");
    setIndexQuery(null);
    setSearchSummary(null);
  }, []);

  async function process() {
    if (!matter || !primaryCourtCase) return;
    setProcessing(true);
    setProcessError(null);
    setProcessMessage(null);
    try {
      const files = selectedFiles.length ? selectedFiles : selectedObject ? [selectedObject] : [];
      if (files.length) {
        for (const object of files) {
          await startProffer({
            request_id: `proffer-${matter.id}-${crypto.randomUUID()}`,
            source_ref: object.source_ref,
            declared_format: handlerOverride || detectedFormat(object.name),
            parser_options_ref: "pending-handler-selection/v1",
            matter_id: matter.id,
            court_case_id: primaryCourtCase.id,
            matter_mode: mode,
          });
        }
        setProcessMessage(`Started ${files.length} run(s).`);
      } else if (selectedFolder && listing) {
        const root = listing.available_roots.find((entry) => entry.root_id === activeRootId);
        if (!root) throw new Error("This folder has no confirmed source location.");
        const started = await startProfferBatch({
          batch_id: newBatchId(),
          matter_id: matter.id,
          court_case_id: primaryCourtCase.id,
          folder_ref: `${root.root_ref.replace(/\/$/, "")}/${selectedFolder.replace(/^\//, "")}`,
          declared_format: handlerOverride || detectedFormat(selectedFolder),
          parser_options_ref: "pending-handler-selection/v1",
          matter_mode: mode,
        });
        setBatchIdent(started.batch_id);
        setProcessMessage(`Batch started for ${selectedFolder}.`);
      }
    } catch (error) {
      setProcessError(errorText(error));
    } finally {
      setProcessing(false);
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

  const listingError = listingQuery.error ? errorText(listingQuery.error) : null;
  const indexResults = indexQuery ? indexSearchQuery.data?.items ?? [] : null;

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="sources-screen">
      <SourceSearch
        query={query}
        mode={searchMode}
        capabilities={capabilitiesQuery.data ?? null}
        searching={listingQuery.isFetching || indexSearchQuery.isFetching}
        onQueryChange={setQuery}
        onModeChange={setSearchMode}
        onSubmit={runSearch}
        onClear={clearSearch}
        resultSummary={searchSummary}
      />

      <div className="flex flex-wrap items-center gap-2 border-b bg-card px-3 py-2">
        <Button
          size="sm"
          disabled={processing || !processTarget || !matter || !primaryCourtCase}
          onClick={() => void process()}
        >
          {processing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />} Process
        </Button>
        <span className="text-[11px] text-muted-foreground">
          {processTarget ? `Ready: ${processTarget}` : "Select files or a folder."}
        </span>
        {processMessage && <span className="text-[11px] text-[#17794b] dark:text-[#72d9a1]">{processMessage}</span>}
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
              setRootId(nextRoot);
              setPrefix("");
              setAppliedFilter("");
              setSelectedFolder(null);
              setSelectedRef(null);
              setCheckedRefs(new Set());
            }}
            onPrefixChange={(nextPrefix) => {
              setPrefix(nextPrefix);
              setAppliedFilter("");
              setSelectedFolder(null);
              setSelectedRef(null);
              setCheckedRefs(new Set());
            }}
            onSelectFolder={(nextFolder) => {
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
            {indexResults ? (
              <IndexResults
                results={indexResults}
                pending={indexSearchQuery.isPending}
                onOpen={(item) => {
                  setPrefix(item.parent ? `${item.parent}/` : "");
                  setIndexQuery(null);
                }}
              />
            ) : rows.length ? (
              <SourceRowsGrid
                rows={rows}
                selectedIndex={selectedIndex}
                checkedRefs={checkedRefs}
                onSelectIndex={(index) => {
                  setSelectedRef(rows[index]?.sourceRef ?? null);
                  setSelectedFolder(null);
                  setHandlerOverride("");
                }}
                onToggleChecked={(refs) => setCheckedRefs(new Set(refs))}
                onReachEnd={() => undefined}
              />
            ) : (
              <p className="px-3 py-4 text-xs text-muted-foreground">
                {listingQuery.isPending ? "Loading files" : "No files here."}
              </p>
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

/** Index hits (contents / meaning) stay read-only until they carry a resolvable B2 key. */
function IndexResults({
  results,
  pending,
  onOpen,
}: {
  results: readonly DiscoveryItem[];
  pending: boolean;
  onOpen: (item: DiscoveryItem) => void;
}) {
  if (pending) {
    return (
      <p className="flex items-center gap-2 px-3 py-4 text-xs text-muted-foreground" role="status">
        <Loader2 className="h-3.5 w-3.5 animate-spin" /> Searching the index
      </p>
    );
  }
  if (!results.length) {
    return <p className="px-3 py-4 text-xs text-muted-foreground">No indexed results.</p>;
  }
  return (
    <ul className="min-h-0 flex-1 divide-y overflow-auto text-xs">
      {results.map((item) => (
        <li key={item.id} className="px-3 py-2">
          <button type="button" className="text-left font-medium hover:text-primary" onClick={() => onOpen(item)}>
            {item.name}
          </button>
          <span className="mt-0.5 block break-all font-mono text-[10px] text-muted-foreground">{item.rel}</span>
          {item.text && <span className="mt-1 block text-[11px] text-muted-foreground">{item.text.slice(0, 200)}</span>}
        </li>
      ))}
    </ul>
  );
}
