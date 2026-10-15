// Byline: Claude Code · Opus 5 · 2026-09-20 (provenance for the selected message; follows selection)
// Byline: Claude Code · Opus 5 · 2026-09-20 (borrows SBV's media preview into the dense/default Table
// mode's detail panel per owner ruling — "borrow" SBV's media/carousel/vCard behavior, not the
// bubble layout, so it now renders next to every attachment listed here instead of raw fields only)
// Byline: Claude Code · Opus 5.5 · 2026-09-26 (each attachment opens the full metadata screen)
"use client";

import { FileSearch, FileText, ShieldCheck } from "lucide-react";
import { useState } from "react";

import { FileMetadataScreen } from "@/components/metadata/file-metadata-screen";
import { AttachmentPreview } from "@/components/sbv/attachment-preview";
import { Button } from "@/components/ui/button";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPackageProjection } from "@/lib/shared/types";

interface MessageSourcePanelProps {
  row: PreviewMessageRow | null;
  packageProjection: ProfferPackageProjection | null;
  previewHandle: string;
  mode: MatterMode;
}

function byteLabel(value: number | null | undefined) {
  if (value === null || value === undefined) return "size unavailable";
  if (value < 1024) return `${value} bytes`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-all font-mono text-[11px]">{value}</dd>
    </div>
  );
}

/** Display native part metadata while preserving every original attribute and payload range.
 * Input: DuckDB native_parts value. Output: metadata and explicit materialization state.
 * Effects: none; no attachment request. Choose for source ranges before decoded media is available.
 */
function NativeParts({ value }: { value: unknown }) {
  const parts = Array.isArray(value) ? value : [];
  return <section>
    <h4 className="platform-rule-title text-xs">Native parts ({parts.length})</h4>
    {parts.length === 0 ? <p className="mt-2 text-xs text-muted-foreground">No native parts are recorded.</p> :
      <ul className="mt-2 space-y-2">{parts.map((part: unknown, index) => {
        const metadata = part && typeof part === "object" ? part as Record<string, unknown> : {};
        const ranges = Array.isArray(metadata.payload_ranges) ? metadata.payload_ranges : [];
        return <li key={index} className="rounded border p-2 text-xs">
          <p className="break-all font-medium">{typeof metadata.filename === "string" && metadata.filename ? metadata.filename : `Part ${index + 1}`}</p>
          <p>{typeof metadata.content_type === "string" ? metadata.content_type : "Content type unavailable"}</p>
          <p className="mt-1 text-muted-foreground">{ranges.length ? "Original payload ranges recorded; decoded media is not materialized." : "No payload range is recorded for this part."}</p>
          <details className="mt-2"><summary className="cursor-pointer">Original attributes and byte ranges</summary>
            <pre className="mt-1 whitespace-pre-wrap break-all font-mono text-[11px]">{JSON.stringify(part, null, 2)}</pre>
          </details>
        </li>;
      })}</ul>}
  </section>;
}

export function MessageSourcePanel({ row, packageProjection, previewHandle, mode }: MessageSourcePanelProps) {
  const [metadataFor, setMetadataFor] = useState<{ sha256: string; label: string } | null>(null);
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="message-source-panel">
      <header className="border-b px-4 py-3">
        <p className="platform-kicker">Provenance</p>
        <h3 className="mt-1 flex items-center gap-2 text-sm font-semibold">
          <ShieldCheck className="size-4" aria-hidden="true" /> {row?.message.native_record ? "Original source record" : "Source and custody"}
        </h3>
      </header>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-3">
        <section>
          <h4 className="platform-rule-title text-xs">Selected message</h4>
          <dl className="mt-2 space-y-2">
            {row ? (
              <Field label="Source locator" value={row.message.source_locator_ref} />
            ) : (
              <p className="text-xs text-muted-foreground">No row is selected.</p>
            )}
          </dl>
        </section>

        {row && !row.message.native_record && (
          <section>
            <h4 className="platform-rule-title text-xs">
              Attachments ({row.attachmentCount})
            </h4>
            {row.attachmentCount === 0 ? (
              <p className="mt-2 text-xs text-muted-foreground">This message carries no retained attachment.</p>
            ) : (
              <ul className="mt-2 space-y-3">
                {row.message.attachments.map((attachment) => (
                  <li key={attachment.attachment_id} className="border p-2">
                    <div className="flex items-start gap-1.5">
                      <FileText className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                      <span className="min-w-0 flex-1 break-all text-xs font-medium">{attachment.filename ?? "Unnamed attachment"}</span>
                      {attachment.sha256 && (
                        <Button
                          type="button"
                          size="sm"
                          variant="ghost"
                          className="h-6 shrink-0 px-1.5 text-[11px]"
                          title="Open every metadata value recorded for this file"
                          onClick={() => attachment.sha256 && setMetadataFor({ sha256: attachment.sha256, label: attachment.filename ?? "Attachment" })}
                        >
                          <FileSearch className="size-3.5" /> All metadata
                        </Button>
                      )}
                    </div>
                    <div className="mt-2">
                      <AttachmentPreview attachment={attachment} previewHandle={previewHandle} mode={mode} />
                    </div>
                    <dl className="mt-2 space-y-1.5">
                      <Field label="Media type" value={attachment.media_type ?? "not reported"} />
                      <Field label="Size" value={byteLabel(attachment.byte_length)} />
                      <Field label="SHA-256" value={attachment.sha256 ?? "not reported"} />
                      <Field label="Source locator" value={attachment.source_locator_ref} />
                    </dl>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}

        {row?.message.native_record && <NativeParts value={row.message.native_record.native.native_parts} />}
        {row?.message.native_record ? <section>
          <h4 className="platform-rule-title text-xs">Native fields and byte locators</h4>
          <p className="mt-2 text-xs">Record status: {row.message.native_record.status}</p>
          <details className="mt-2 text-xs"><summary className="cursor-pointer">All native fields, child elements and locators</summary>
            <pre className="mt-2 whitespace-pre-wrap break-all font-mono text-[11px]">{JSON.stringify(row.message.native_record.native, null, 2)}</pre>
          </details>
        </section> : <section>
          <h4 className="platform-rule-title text-xs">Retained package</h4>
          {packageProjection ? (
            <dl className="mt-2 space-y-2">
              <Field label="Original object" value={packageProjection.original_ref ?? "not retained"} />
              <Field label="Original SHA-256" value={packageProjection.original_sha256 ?? "not reported"} />
              <Field label="Original bytes" value={byteLabel(packageProjection.original_bytes)} />
              <Field
                label="Declared format / status"
                value={`${packageProjection.declared_format} · ${packageProjection.status}`}
              />
              <Field label="Source version" value={packageProjection.source_version_ref} />
              <Field label="Storage class" value={packageProjection.storage_class ?? "not reported"} />
            </dl>
          ) : (
            <p className="mt-2 text-xs text-muted-foreground">
              The package projection has not been returned for this attempt.
            </p>
          )}
        </section>}
      </div>

      <footer className="border-t px-4 py-2 text-[10px] text-muted-foreground">
        {row?.message.native_record ? "Native source bundle · exact source locators" : "Read-only projection · PostgreSQL remains canonical"}
      </footer>
      <FileMetadataScreen
        open={metadataFor !== null}
        onOpenChange={(open) => !open && setMetadataFor(null)}
        previewHandle={previewHandle}
        mode={mode}
        subjectSha256={metadataFor?.sha256}
        fileLabel={metadataFor?.label}
      />
    </div>
  );
}
