// Byline: Claude Code · Opus 5 · 2026-09-22
// RIGHT region of Sources: the metadata panel. Always open, follows selection.
//
// Owner, 2026-09-22 09:00: "i need the metadata in view and some way to signify
// it's a unit of some kind". Everything shown here is a server fact with its
// origin named — observed object, catalog record, decode manifest, run record —
// and anything unknown says so on the item itself, never as a caveat paragraph.
"use client";

import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";

import { DecodedSourceViewer } from "@/components/sbv/decoded-source-viewer";
import {
  HANDLER_CHOICES,
  SOURCE_STATE_CLASS,
  SOURCE_STATE_LABEL,
  formatBytes,
  type SourceState,
} from "@/components/sources/source-state";
import { Button } from "@/components/ui/button";
import type { DecodedManifest } from "@/lib/decoded-source-client";
import type {
  CatalogProvenance,
  CatalogUnitLookup,
  ProfferBatchStatus,
  ProfferProposalResource,
  ProfferSourceInspection,
  ProfferSourceObject,
  SourceUnitKind,
  SourceUnitMark,
  SourceUnitProposal,
} from "@/lib/shared/types";
import { cn } from "@/lib/utils";

type CatalogUnit = CatalogUnitLookup["units"][number];

export interface FolderSelection {
  kind: "folder";
  prefix: string;
  unit: CatalogUnit | null;
  mark: SourceUnitMark | null;
  /** How many recorded units the files under this folder belong to. */
  unitsUnder: number;
}

export interface FileSelection {
  kind: "file";
  object: ProfferSourceObject;
  state: SourceState;
  memberOfUnit: string | null;
}

export type SourceSelection = FolderSelection | FileSelection | null;

function Field({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="border-b px-3 py-2 last:border-b-0">
      <dt className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className={cn("mt-0.5 break-all text-xs", mono && "font-mono text-[10px]")}>{value}</dd>
    </div>
  );
}

function Flag({ children }: { children: React.ReactNode }) {
  return (
    <span className="border border-[#c58214] bg-[#fff4dd] px-1 text-[9px] font-semibold uppercase text-[#684b18] dark:bg-[#43351f] dark:text-[#ffe0a6]">
      {children}
    </span>
  );
}

export function SourceMetadataPanel({
  selection,
  inspection,
  inspectionLoading,
  inspectionError,
  manifest,
  runs,
  provenance,
  provenanceMissing,
  handlerOverride,
  onHandlerOverrideChange,
  unitProposal,
  unitMarkKind,
  onUnitMarkKindChange,
  onMarkUnit,
  markPending,
  markError,
  markStorage,
  batch,
  canonicalHomeOffered,
}: {
  selection: SourceSelection;
  inspection: ProfferSourceInspection | null;
  inspectionLoading: boolean;
  inspectionError: string | null;
  manifest: DecodedManifest | null;
  runs: readonly ProfferProposalResource[];
  provenance: CatalogProvenance | null;
  provenanceMissing: boolean;
  handlerOverride: string;
  onHandlerOverrideChange: (value: string) => void;
  unitProposal: SourceUnitProposal | null;
  unitMarkKind: SourceUnitKind;
  onUnitMarkKindChange: (value: SourceUnitKind) => void;
  onMarkUnit: (confirm: boolean) => void;
  markPending: boolean;
  markError: string | null;
  markStorage: string;
  batch: ProfferBatchStatus | null;
  canonicalHomeOffered: boolean;
}) {
  return (
    <aside className="flex h-full min-h-0 flex-col border-l bg-card" aria-label="Selected source details">
      {!selection ? (
        <p className="px-3 py-4 text-xs text-muted-foreground">Select a file or a folder to see its details here.</p>
      ) : selection.kind === "folder" ? (
        <FolderDetail
          selection={selection}
          unitProposal={unitProposal}
          unitMarkKind={unitMarkKind}
          onUnitMarkKindChange={onUnitMarkKindChange}
          onMarkUnit={onMarkUnit}
          markPending={markPending}
          markError={markError}
          markStorage={markStorage}
          batch={batch}
          canonicalHomeOffered={canonicalHomeOffered}
        />
      ) : (
        <FileDetail
          selection={selection}
          inspection={inspection}
          inspectionLoading={inspectionLoading}
          inspectionError={inspectionError}
          manifest={manifest}
          runs={runs}
          provenance={provenance}
          provenanceMissing={provenanceMissing}
          handlerOverride={handlerOverride}
          onHandlerOverrideChange={onHandlerOverrideChange}
        />
      )}
    </aside>
  );
}

function FolderDetail({
  selection,
  unitProposal,
  unitMarkKind,
  onUnitMarkKindChange,
  onMarkUnit,
  markPending,
  markError,
  markStorage,
  batch,
  canonicalHomeOffered,
}: {
  selection: FolderSelection;
  unitProposal: SourceUnitProposal | null;
  unitMarkKind: SourceUnitKind;
  onUnitMarkKindChange: (value: SourceUnitKind) => void;
  onMarkUnit: (confirm: boolean) => void;
  markPending: boolean;
  markError: string | null;
  markStorage: string;
  batch: ProfferBatchStatus | null;
  canonicalHomeOffered: boolean;
}) {
  const unit = selection.unit;
  const takeout = unitProposal?.takeout_sets ?? [];
  const needsConfirm = Boolean(unitProposal?.requires_confirmation) || unitMarkKind.startsWith("takeout");

  return (
    <div className="min-h-0 flex-1 overflow-auto">
      <header className="border-b px-3 py-2">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Folder</p>
        <strong className="block break-all text-sm">{selection.prefix}</strong>
        <div className="mt-1 flex flex-wrap items-center gap-1">
          {unit && (
            <span className="border border-[#4051b9] bg-[#e9ecfb] px-1 text-[9px] font-semibold uppercase text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]">
              catalog unit · {unit.unit_type.replaceAll("_", " ")}
            </span>
          )}
          {selection.mark && (
            <span className="border border-[#2f9d67] bg-[#e2f3e9] px-1 text-[9px] font-semibold uppercase text-[#17794b] dark:bg-[#203d31] dark:text-[#72d9a1]">
              hand-marked · {selection.mark.unit_type.replaceAll("_", " ")}
            </span>
          )}
          {selection.unitsUnder > 1 && <Flag>files from {selection.unitsUnder} units</Flag>}
          {!unit && !selection.mark && selection.unitsUnder === 0 && <Flag>not a unit</Flag>}
        </div>
      </header>

      {unit && (
        <dl className="border-b">
          <Field label="Unit kind" value={unit.unit_type.replaceAll("_", " ")} />
          <Field label="Members" value={unit.member_count.toLocaleString()} />
          <Field label="Total size" value={formatBytes(unit.total_bytes)} />
          <Field label="Members without sha1" value={unit.members_without_sha1.toLocaleString()} />
          <Field label="Parent unit" value={unit.parent_unit_id ? String(unit.parent_unit_id) : "none"} />
          <Field label="Recorded root" value={unit.unit_root} mono />
        </dl>
      )}

      <section className="border-b px-3 py-3">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Mark as unit</p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <select
            className="h-8 border bg-background px-2 text-xs"
            value={unitMarkKind}
            aria-label="Unit kind to mark"
            onChange={(event) => onUnitMarkKindChange(event.target.value as SourceUnitKind)}
          >
            {["takeout", "takeout_zip", "facebook", "snapchat", "cube_acr", "git_repo", "obsidian_vault", "other"].map((kind) => (
              <option key={kind} value={kind}>{kind.replaceAll("_", " ")}</option>
            ))}
          </select>
          <Button size="sm" variant="outline" disabled={markPending} onClick={() => onMarkUnit(needsConfirm)}>
            {markPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
            {needsConfirm ? "Confirm and mark" : "Mark as unit"}
          </Button>
        </div>
        {takeout.map((set) => (
          <p key={`${set.stamp}-${set.job}`} className="mt-2 text-[11px] text-muted-foreground">
            Takeout {set.stamp}-{set.job}: parts {set.parts_present.length}/{set.highest_part} present
            {set.parts_missing.length ? (
              <> · <span className="text-[#8f302a] dark:text-[#ffb5ae]">missing {set.parts_missing.join(", ")}</span></>
            ) : (
              <> · <CheckCircle2 className="inline h-3 w-3" /> complete run of parts</>
            )}
          </p>
        ))}
        {markError && <p className="mt-2 text-[11px] text-[#8f302a] dark:text-[#ffb5ae]" role="alert">{markError}</p>}
        <p className="mt-2 text-[10px] leading-4 text-muted-foreground">
          Hand-marked units are stored by the Workbench ({markStorage}). The catalog&apos;s own unit tables stay read-only.
        </p>
        {canonicalHomeOffered && (
          <p className="mt-1 text-[10px] leading-4 text-muted-foreground">
            Moving a marked unit to a canonical home is an option, not a step: Process runs on this folder wherever it sits.
          </p>
        )}
      </section>

      {batch && <BatchProgress batch={batch} />}
    </div>
  );
}

function BatchProgress({ batch }: { batch: ProfferBatchStatus }) {
  const { counts } = batch;
  return (
    <section className="border-b px-3 py-3" aria-label="Batch progress">
      <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Batch {batch.terminal ? "finished" : "running"}</p>
      <p className="mt-1 text-xs">
        {counts.done} done · {counts.running} running · {counts.queued} queued · {counts.waiting_on_gate} waiting · {counts.failed} failed · {counts.skipped} skipped
        {" "}of {counts.total}
      </p>
      {batch.listing_truncated && <p className="mt-1 text-[11px] text-muted-foreground">This folder holds more members than one batch imports.</p>}
      <p className="mt-1 break-all font-mono text-[10px] text-muted-foreground">{batch.batch_id}</p>
    </section>
  );
}

function FileDetail({
  selection,
  inspection,
  inspectionLoading,
  inspectionError,
  manifest,
  runs,
  provenance,
  provenanceMissing,
  handlerOverride,
  onHandlerOverrideChange,
}: {
  selection: FileSelection;
  inspection: ProfferSourceInspection | null;
  inspectionLoading: boolean;
  inspectionError: string | null;
  manifest: DecodedManifest | null;
  runs: readonly ProfferProposalResource[];
  provenance: CatalogProvenance | null;
  provenanceMissing: boolean;
  handlerOverride: string;
  onHandlerOverrideChange: (value: string) => void;
}) {
  const object = selection.object;
  const detected = inspection?.parser_preflight.declared_format ?? "";
  const catalogRow = provenance?.items[0] ?? null;

  return (
    <div className="min-h-0 flex-1 overflow-auto">
      <header className="border-b px-3 py-2">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">File</p>
        <strong className="block break-all text-sm">{object.name}</strong>
        <div className="mt-1 flex flex-wrap items-center gap-1">
          <span className={cn("border px-1 text-[9px] font-semibold uppercase", SOURCE_STATE_CLASS[selection.state])}>
            {SOURCE_STATE_LABEL[selection.state]}
          </span>
          {selection.memberOfUnit && (
            <span className="border border-[#4051b9] bg-[#e9ecfb] px-1 text-[9px] font-semibold uppercase text-[#2b3785] dark:bg-[#313a66] dark:text-[#c9d0fb]">
              part of {selection.memberOfUnit.replaceAll("_", " ")}
            </span>
          )}
        </div>
      </header>

      <dl className="border-b">
        <Field label="Size" value={formatBytes(object.byte_length)} />
        <Field label="Modified" value={object.last_modified ? new Date(object.last_modified).toLocaleString() : "not reported"} />
        <Field
          label="Hash (sha256)"
          value={inspectionLoading ? "reading and hashing" : inspection?.sha256 || "not hashed"}
          mono
        />
        <Field label="Detected format" value={inspectionLoading ? "inspecting" : detected || "not inspected"} />
        <Field label="Object key" value={object.key} mono />
        <Field label="Source reference" value={object.source_ref} mono />
      </dl>

      {inspectionError && (
        <p className="flex items-start gap-1 border-b px-3 py-2 text-[11px] text-[#8f302a] dark:text-[#ffb5ae]" role="alert">
          <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" /> {inspectionError}
        </p>
      )}

      <section className="border-b px-3 py-3">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Handler</p>
        <p className="mt-1 text-xs">
          Recommended: <strong>{HANDLER_CHOICES.find((choice) => choice.formats.includes(detected))?.label ?? "detected on the engine"}</strong>
        </p>
        <label className="mt-2 grid gap-1 text-[10px] uppercase tracking-wide text-muted-foreground">
          Override before Process
          <select
            className="h-8 border bg-background px-2 text-xs normal-case text-foreground"
            value={handlerOverride}
            onChange={(event) => onHandlerOverrideChange(event.target.value)}
          >
            <option value="">Use the detected format</option>
            {HANDLER_CHOICES.map((choice) => (
              <option key={choice.id} value={choice.formats[0]}>{choice.label}</option>
            ))}
          </select>
        </label>
      </section>

      <section className="border-b px-3 py-3">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Decode</p>
        {manifest ? (
          <p className="mt-1 text-xs">
            {manifest.records.toLocaleString()} records · {manifest.threads.length.toLocaleString()} threads ·{" "}
            {manifest.media_objects.toLocaleString()} media · {manifest.rejected.toLocaleString()} rejected
            {manifest.decoder ? ` · ${manifest.decoder}` : ""}
          </p>
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">Not decoded yet.</p>
        )}
      </section>

      <section className="border-b px-3 py-3">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Runs that touched it</p>
        {runs.length ? (
          <ul className="mt-1 space-y-1 text-[11px]">
            {runs.slice(0, 6).map((run) => (
              <li key={run.preview_handle} className="flex items-center justify-between gap-2">
                <span className="truncate font-mono text-[10px]">{run.preview_handle}</span>
                <span className="shrink-0">{run.lifecycle.replaceAll("_", " ")}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">No run has touched this source.</p>
        )}
      </section>

      <section className="border-b px-3 py-3">
        <p className="flex items-center gap-1 text-[10px] uppercase tracking-wide text-muted-foreground">
          Catalog provenance {provenanceMissing && <Flag>no catalog record</Flag>}
        </p>
        {catalogRow ? (
          <p className="mt-1 text-[11px]">
            {catalogRow.source ?? "source unknown"} · {catalogRow.scope ?? "scope unknown"}
            <span className="mt-0.5 block break-all font-mono text-[10px] text-muted-foreground">{catalogRow.rel}</span>
            <span className="mt-0.5 block">{provenance?.occurrences.toLocaleString()} occurrence(s) recorded</span>
          </p>
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">Not resolved in the pre-ingest catalog.</p>
        )}
      </section>

      <section className="min-h-[220px] px-3 py-3" aria-label="Preview">
        <p className="text-[10px] uppercase tracking-wide text-muted-foreground">Preview</p>
        <div className="mt-2">
          {manifest ? (
            <DecodedSourceViewer sourceRef={object.source_ref} />
          ) : inspectionLoading ? (
            <p className="flex items-center gap-2 text-xs text-muted-foreground"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Reading</p>
          ) : inspection?.preview_kind === "image" && inspection.preview_url ? (
            <img className="max-h-[320px] max-w-full border object-contain" src={inspection.preview_url} alt={`View of ${inspection.name}`} />
          ) : inspection?.preview_kind === "pdf" && inspection.preview_url ? (
            <iframe className="h-[320px] w-full border bg-white" src={inspection.preview_url} title={`View of ${inspection.name}`} />
          ) : inspection?.preview_text ? (
            <pre className="max-h-[320px] overflow-auto border bg-background p-2 font-mono text-[10px] leading-4">{inspection.preview_text.slice(0, 4000)}</pre>
          ) : (
            <p className="text-xs text-muted-foreground">No inline preview for this format.</p>
          )}
        </div>
      </section>
    </div>
  );
}
