// Byline: Claude Code · Opus 5.5 · 2026-09-26
// The Review metadata screen: click a file (the run's source, or an attachment)
// and see ALL of its metadata — what the engine recorded for it (every class,
// native values verbatim, extractor provenance), its sidecars with where they
// were found, custody hashes, retained members — and correct any value.
//
// Owner 2026-09-25 19:13 / 19:15. Recorded values are immutable: a correction
// is an attributed, append-only overlay revision shown beside the value.
// Anything unavailable is one small flag on the item, never a banner.
"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FileSearch, Loader2 } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";

import { MetadataFieldTable, type CorrectionSubmit, type FieldRow } from "@/components/metadata/metadata-field-table";
import {
  METADATA_CLASS_LABEL,
  SIDECAR_KIND_LABEL,
  correctionsByField,
  flattenFields,
  formatBytes,
  formatValue,
  formatWhen,
} from "@/components/metadata/metadata-format";
import { SmallFlag } from "@/components/metadata/small-flag";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  getMetadataScreen,
  postMetadataCorrection,
  type MetadataScreen,
  type Sidecar,
} from "@/lib/review-overlays-client";
import type { MatterMode } from "@/lib/shared/types";

const EMBEDDED_CLASSES = new Set(["embedded", "container", "media_tool"]);

function lastSegment(ref: string) {
  return ref.split("/").filter(Boolean).at(-1) ?? ref;
}

function Section({ label, title, aside, children }: { label: string; title: string; aside?: ReactNode; children: ReactNode }) {
  return (
    <section aria-label={label} className="space-y-2">
      <header className="flex flex-wrap items-center gap-2 border-b pb-1">
        <h3 className="text-sm font-semibold">{title}</h3>
        {aside}
      </header>
      {children}
    </section>
  );
}

function Fact({ label, value, mono = false }: { label: string; value: ReactNode; mono?: boolean }) {
  return (
    <div className="bg-card px-3 py-2">
      <dt className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className={mono ? "mt-0.5 break-all font-mono text-[10px]" : "mt-0.5 break-words text-xs"}>{value}</dd>
    </div>
  );
}

function conflictDetails(screen: MetadataScreen): Map<string, string> {
  const byKey = new Map<string, string>();
  const nameByKey = new Map(screen.sidecars.map((sidecar) => [sidecar.key, sidecar.name]));
  for (const conflict of screen.sidecar_conflicts) {
    const sidecarField = `sidecar:${nameByKey.get(conflict.sidecar_key) ?? conflict.sidecar_key}:${conflict.sidecar_path}`;
    byKey.set(sidecarField, `${conflict.topic.replace("_", " ")}: the file says ${formatValue(conflict.file_value)} (${conflict.file_field}); ${conflict.detail}`);
    byKey.set(conflict.file_field, `${conflict.topic.replace("_", " ")}: ${conflict.sidecar_key} says ${formatValue(conflict.sidecar_value)}; ${conflict.detail}`);
  }
  return byKey;
}

function sidecarRows(sidecar: Sidecar): FieldRow[] {
  return sidecar.fields.map((field) => ({ fieldKey: `sidecar:${sidecar.name}:${field.path}`, label: field.path, value: field.value }));
}

export function FileMetadataScreen({
  open,
  onOpenChange,
  previewHandle,
  mode,
  subjectSha256,
  fileLabel,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  previewHandle: string;
  mode: MatterMode;
  /** Omit for the run's source file; an attachment's or member's SHA-256 otherwise. */
  subjectSha256?: string | null;
  fileLabel?: string;
}) {
  const queryClient = useQueryClient();
  const queryKey = useMemo(() => ["review-metadata", mode, previewHandle, subjectSha256 ?? "source"] as const, [mode, previewHandle, subjectSha256]);
  const query = useQuery({
    queryKey,
    queryFn: () => getMetadataScreen(previewHandle, mode, subjectSha256),
    enabled: open,
    retry: false,
  });
  const [pendingField, setPendingField] = useState<string | null>(null);
  const [writeError, setWriteError] = useState<string | null>(null);
  const screen = query.data;
  const corrections = useMemo(() => correctionsByField(screen?.corrections ?? []), [screen]);
  const conflicts = useMemo(() => (screen ? conflictDetails(screen) : new Map<string, string>()), [screen]);
  const correctable = Boolean(screen?.corrections_available && screen.subject_sha256);

  const submit = async (input: CorrectionSubmit) => {
    if (!screen?.subject_sha256) return false;
    setPendingField(input.fieldKey);
    setWriteError(null);
    try {
      await postMetadataCorrection(previewHandle, mode, {
        subject_sha256: screen.subject_sha256,
        field_key: input.fieldKey,
        supersedes_ref: input.supersedesRef,
        action: input.action,
        source_value: input.observed,
        corrected_value: input.correctedValue,
        change_reason: input.reason,
      });
      await queryClient.invalidateQueries({ queryKey });
      return true;
    } catch (error) {
      setWriteError(error instanceof Error ? error.message : "The correction was not saved");
      return false;
    } finally {
      setPendingField(null);
    }
  };

  const source = screen?.source ?? null;
  const isSource = screen?.subject_kind === "source";
  const embeddedRows = screen?.metadata.filter((row) => EMBEDDED_CLASSES.has(row.metadata_class)) ?? [];
  const title = fileLabel ?? (isSource ? source?.original_filename ?? lastSegment(screen?.source_ref ?? "") : screen?.attachment?.filename) ?? "File metadata";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="flex h-[92vh] max-h-[92vh] w-[min(1400px,96vw)] max-w-[min(1400px,96vw)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[min(1400px,96vw)]"
        data-testid="file-metadata-screen"
      >
        <DialogHeader className="border-b px-5 py-3 text-left">
          <DialogTitle className="flex flex-wrap items-center gap-2 text-base">
            <FileSearch className="size-4" aria-hidden="true" />
            <span className="min-w-0 break-all">{title}</span>
            {screen && <SmallFlag tone="info">{screen.subject_kind === "source" ? "source file" : screen.subject_kind}</SmallFlag>}
            {query.isError && <SmallFlag title={query.error instanceof Error ? query.error.message : undefined}>metadata unavailable</SmallFlag>}
            {writeError && <SmallFlag title={writeError}>correction not saved</SmallFlag>}
            {screen && !screen.corrections_available && <SmallFlag title="The correction overlay table is not installed on this platform yet">corrections unavailable</SmallFlag>}
          </DialogTitle>
          <DialogDescription className="text-xs">
            Every value the platform recorded for this file, with where it came from. Corrections are added beside a value; the recorded value never changes.
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-5 py-4">
          {query.isPending && open && (
            <p className="flex items-center gap-2 text-sm text-muted-foreground" role="status"><Loader2 className="size-4 animate-spin" /> Reading metadata</p>
          )}
          {screen && (
            <>
              <Section label="This file" title="This file">
                <dl className="grid gap-px border bg-border sm:grid-cols-2 xl:grid-cols-3">
                  <Fact label="SHA-256" value={screen.subject_sha256 || "not recorded"} mono />
                  <Fact label="Size" value={formatBytes(isSource ? source?.original?.byte_length : screen.attachment?.byte_length)} />
                  <Fact label={isSource ? "Declared format" : "Media type"} value={(isSource ? source?.declared_format : screen.attachment?.media_type) || "not recorded"} />
                  {isSource && <Fact label="Retained copy" value={source?.original ? `${source.original.storage_class} · sealed ${formatWhen(source.original.immutable_at)}` : "not retained yet"} />}
                  {isSource && <Fact label="Acquired" value={formatWhen(source?.acquired_at)} />}
                  {isSource && <Fact label="Provenance class" value={source?.provenance_class || "not recorded"} />}
                  <Fact label="Source reference" value={screen.source_ref} mono />
                  {source && <Fact label="Source version" value={source.source_version_ref} mono />}
                  {source?.source_context_ref && <Fact label="Your context revision" value={source.source_context_ref} mono />}
                  {!isSource && screen.attachment && <Fact label="Carried by messages" value={`${screen.attachment.message_ids.length}${screen.attachment.message_ids.length >= 50 ? "+" : ""}`} />}
                  {!isSource && screen.attachment && <Fact label="Attachment locator" value={screen.attachment.source_locator_ref} mono />}
                </dl>
              </Section>

              <Section
                label="Recorded metadata"
                title="Recorded metadata"
                aside={
                  <>
                    {isSource && embeddedRows.length === 0 && (
                      <SmallFlag title="The embedded-metadata stage recorded no values for this run (EXIF, document properties, media tags). No reader is wired to it yet.">embedded metadata not read</SmallFlag>
                    )}
                    {!isSource && <SmallFlag title="Embedded metadata is read for a run's source file; attachments have no reader yet.">attachment metadata not read</SmallFlag>}
                    {screen.metadata_truncated && <SmallFlag>more rows than shown</SmallFlag>}
                  </>
                }
              >
                {isSource && screen.metadata.map((row) => {
                  const { fields, truncated } = flattenFields(row.fields);
                  return (
                    <div key={row.metadata_ref} className="space-y-1" data-metadata-class={row.metadata_class}>
                      <p className="flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                        <strong className="text-foreground">{METADATA_CLASS_LABEL[row.metadata_class] ?? row.metadata_class}</strong>
                        <span className="font-mono">{row.extractor_id}{row.extractor_version ? `@${row.extractor_version}` : ""}</span>
                        <span>recorded {formatWhen(row.generated_at)}</span>
                        {row.fields === null && <SmallFlag>{`larger than the screen reads (${formatBytes(row.fields_bytes)})`}</SmallFlag>}
                        {truncated && <SmallFlag>fields truncated</SmallFlag>}
                      </p>
                      <MetadataFieldTable
                        rows={fields.map((field) => ({ fieldKey: `${row.metadata_class}:${field.path}`, label: field.path, value: field.value }))}
                        corrections={corrections}
                        conflicts={conflicts}
                        correctable={correctable}
                        pendingField={pendingField}
                        onSubmit={submit}
                      />
                    </div>
                  );
                })}
                {isSource && screen.record_metadata_count > 0 && (
                  <p className="text-[11px] text-muted-foreground">{screen.record_metadata_count.toLocaleString()} per-record native metadata rows belong to individual messages and records.</p>
                )}
              </Section>

              {isSource && (
                <Section
                  label="Sidecars"
                  title="Sidecars and companion files"
                  aside={
                    <>
                      {screen.sidecar_lookup.beside_object === "unavailable" && <SmallFlag>listing beside the file unavailable</SmallFlag>}
                      {screen.sidecar_lookup.catalog_folder === "unavailable" && <SmallFlag>catalog lookup unavailable</SmallFlag>}
                      {screen.sidecar_lookup.catalog_folder === "not_configured" && <SmallFlag title="The pre-ingest catalog connection is not configured for the Workbench">catalog not connected</SmallFlag>}
                      {screen.sidecar_conflicts.length > 0 && <SmallFlag>{`${screen.sidecar_conflicts.length} disagreement${screen.sidecar_conflicts.length === 1 ? "" : "s"} with the file`}</SmallFlag>}
                    </>
                  }
                >
                  {screen.sidecars.length === 0 && <p className="text-xs text-muted-foreground">No sidecar was found for this file.</p>}
                  {screen.sidecars.map((sidecar) => (
                    <div key={`${sidecar.found_via}:${sidecar.key}`} className="space-y-1" data-sidecar-kind={sidecar.kind}>
                      <p className="flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                        <strong className="break-all text-foreground">{sidecar.name}</strong>
                        <span>{SIDECAR_KIND_LABEL[sidecar.kind] ?? sidecar.kind}</span>
                        <span>{sidecar.found_via === "beside_object" ? "beside the file" : "in the catalog's original folder"}</span>
                        <span>{formatBytes(sidecar.byte_length)}</span>
                        {sidecar.unread_reason && <SmallFlag title={sidecar.unread_reason}>not read</SmallFlag>}
                        {sidecar.fields_truncated && <SmallFlag>fields truncated</SmallFlag>}
                      </p>
                      {sidecar.fields.length > 0 && (
                        <MetadataFieldTable rows={sidecarRows(sidecar)} corrections={corrections} conflicts={conflicts} correctable={correctable} pendingField={pendingField} onSubmit={submit} />
                      )}
                      {sidecar.text !== null && (
                        <pre className="max-h-72 overflow-auto whitespace-pre-wrap border bg-muted/30 p-2 font-mono text-[10px] leading-4">
                          {sidecar.text}{sidecar.text_truncated ? "\n…" : ""}
                        </pre>
                      )}
                    </div>
                  ))}
                </Section>
              )}

              <Section label="Custody hashes" title="Custody hashes" aside={screen.hashes.length === 0 ? <SmallFlag>no hash receipt recorded</SmallFlag> : undefined}>
                {screen.hashes.length > 0 && (
                  <ol className="divide-y border text-[11px]">
                    {screen.hashes.map((hash) => (
                      <li key={`${hash.hash_kind}:${hash.digest}`} className="grid gap-1 px-3 py-1.5 md:grid-cols-[14rem_minmax(0,1fr)_12rem]">
                        <span>{hash.hash_kind.replaceAll("_", " ")}</span>
                        <span className="break-all font-mono text-[10px]">{hash.digest}</span>
                        <span className="text-muted-foreground">{hash.computed_by} · {formatWhen(hash.computed_at)}</span>
                      </li>
                    ))}
                  </ol>
                )}
              </Section>

              {isSource && screen.members.length > 0 && (
                <details className="border">
                  <summary className="cursor-pointer px-3 py-2 text-xs font-semibold">
                    Retained members ({screen.members.length}{screen.members_truncated ? "+" : ""}) · {screen.attachment_count.toLocaleString()} projected attachments
                  </summary>
                  <ol className="divide-y border-t text-[11px]">
                    {screen.members.map((member) => (
                      <li key={member.object_ref} className="grid gap-1 px-3 py-1.5 md:grid-cols-[10rem_minmax(0,1fr)_10rem]">
                        <span>{member.role.replaceAll("_", " ")}</span>
                        <span className="break-all font-mono text-[10px]">{member.sha256}</span>
                        <span className="text-muted-foreground">{formatBytes(member.byte_length)} · {member.storage_class}</span>
                      </li>
                    ))}
                  </ol>
                </details>
              )}
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
