// Selection-linked, progressively disclosed inspection over real stored records.
// Byline: ChatGPT, 2026-10-17.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { FileMetadataScreen } from "@/components/metadata/file-metadata-screen";
import { ContextReviewPanel } from "@/components/review/context-review";
import { ReadAttribution } from "@/components/read/read-attribution";
import { ErrorBox } from "@/components/mobile/mobile-ui";
import { Button } from "@/components/ui/button";
import { importedApi } from "@/lib/imported-client";
import { useOperatingMode } from "@/lib/fixed-case-context";
import type { ImportedUnit, UnitMessage } from "@/lib/unit-workspace-types";

/** Keep complete stored values available without covering the conversation.
 * Inputs: selected unit/message and record ID. Output: compact card and expandable
 * metadata/PG/review editor. Effects: reads; existing explicit correction actions.
 */
export function UnitInspector({ unit, message, recordId }: {
  unit: ImportedUnit; message: UnitMessage | null; recordId: string | null;
}) {
  const mode = useOperatingMode();
  const [expanded, setExpanded] = useState(false);
  const [metadataOpen, setMetadataOpen] = useState(false);
  const record = useQuery({
    queryKey: ["unit-record", unit.source_id, recordId],
    queryFn: ({ signal }) => importedApi.unitRecord(unit.source_id, recordId!, signal),
    enabled: Boolean(recordId), retry: false,
  });
  const handle = record.data?.preview_handle ?? message?.attribution?.preview_handle;
  const source = message ? unit.members.find((member) => member.source.id === message.source.id)?.source
    : record.data ? unit.members.find((member) => member.versions.some((version) => version.export_key === record.data!.export_key))?.source
      : !recordId && unit.members.length === 1 ? unit.members[0].source : undefined;
  const recordLabel = message?.source.file_name ?? (record.data ? record.data.source_key.split("/").at(-1) : undefined);
  const owners = [...new Set(unit.members.map((member) => member.source.owner_name || member.source.owner).filter(Boolean))];
  const ownerLabel = recordId ? source?.owner_name || source?.owner || "Not resolved for this record"
    : owners.length === 1 ? owners[0] : owners.length > 1 ? "Multiple source-owner labels" : "Not recorded";
  return <aside className="min-w-0 space-y-3" aria-label="Selected item inspector">
    <section className="rounded-lg border bg-card p-4">
      <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{recordId ? "Selected record" : "Selected unit"}</p>
      <h2 className="mt-2 break-words text-sm font-semibold">{message?.sender.label || (recordId ? "Stored message" : unit.root.split("/").at(-1))}</h2>
      <p className="mt-1 break-all text-xs text-muted-foreground">{recordLabel || `${unit.members.length} registered source parts`}</p>
      <dl className="mt-3 space-y-2 text-xs">
        <div><dt className="text-muted-foreground">Source perspective</dt><dd>{message?.projection_kind?.replaceAll("_", " ") || "Select a message to inspect"}</dd></div>
        <div><dt className="text-muted-foreground">Recorded source-owner label</dt><dd>{ownerLabel}</dd></div>
        <div><dt className="text-muted-foreground">PostgreSQL</dt><dd>{record.data ? `${record.data.current_generation ? "Current" : "Historical"} record read back` : `${unit.message_count.toLocaleString()} messages in selected generations`}</dd></div>
      </dl>
      <Button className="mt-4 w-full" variant="outline" aria-expanded={expanded} aria-controls="unit-expanded-detail" onClick={() => setExpanded((open) => !open)}>
        {expanded ? "Collapse details" : "Details & corrections"}
      </Button>
    </section>
    {expanded ? <section id="unit-expanded-detail" className="rounded-lg border bg-card p-3" aria-label="Complete selected item details">
      {record.isError ? <ErrorBox error={record.error} onRetry={() => void record.refetch()} /> : null}
      {recordId && record.isPending ? <p className="p-2 text-xs" role="status">Reading the actual PostgreSQL record...</p> : null}
      {record.data ? <>
        <dl className="space-y-2 break-all text-xs">
          <div><dt className="font-semibold">Record / generation</dt><dd>{record.data.id}<br />{record.data.generation_id}</dd></div>
          <div><dt className="font-semibold">Original member / ordinal</dt><dd>{record.data.source_key}<br />#{record.data.record_ordinal}</dd></div>
          <div><dt className="font-semibold">Source version</dt><dd>{record.data.source_version_id}</dd></div>
          <div><dt className="font-semibold">Readback time</dt><dd>{record.data.read_at}</dd></div>
        </dl>
        <details className="mt-3 text-xs"><summary className="cursor-pointer font-medium">All normalized fields from PostgreSQL</summary>
          <pre className="mt-2 max-h-96 overflow-auto whitespace-pre-wrap break-all rounded bg-muted p-2 text-[11px]">{JSON.stringify(record.data.normalized_payload, null, 2)}</pre>
        </details>
        <p className="mt-2 text-[11px] text-muted-foreground">Stored values are not an independent correctness check. Original-versus-normalized validation remains a separate check.</p>
      </> : <p className="text-xs text-muted-foreground">Select a message to inspect its exact PG record and available source operation.</p>}
      {message && mode === "LIVE" ? <ReadAttribution message={message} /> : null}
      {handle && recordId ? <>
        <ContextReviewPanel key={`${mode}:${recordId}`} previewHandle={handle} mode={mode} messageId={recordId} />
        <Button variant="outline" size="sm" className="mt-2 w-full" onClick={() => setMetadataOpen(true)}>File metadata & corrections</Button>
        <FileMetadataScreen open={metadataOpen} onOpenChange={setMetadataOpen} previewHandle={handle} mode={mode} fileLabel={recordLabel} />
      </> : null}
      <details className="mt-3 text-xs"><summary className="cursor-pointer font-medium">Unit membership and generation policy</summary>
        <p className="mt-2 break-all">{unit.root}</p><p className="mt-2">{unit.generation_policy}</p><p className="mt-2">{unit.coverage}</p>
        <p className="mt-2 text-muted-foreground">The existing identifier editor changes a registry alias; message-local contact reassignment is not implemented by this panel.</p>
      </details>
    </section> : null}
  </aside>;
}
