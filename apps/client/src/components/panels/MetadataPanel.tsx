/**
 * Byline: Claude Code · Opus 5 · 2026-09-17
 *
 * Native Metadata panel (owner decision 2026-09-17, item 6; the docked-iframe review panel was
 * rejected 2026-09-14). Everything comes from the hosted Intake engine over the normal command
 * transport, credentials stay server-side:
 *   (a) properties  — the donor's get_detailed_file_properties
 *   (b) catalog     — every recorded occurrence of the file (PG casebible.raw_duck)
 *   (c) extracted   — EXIF, audio tags, ffprobe streams, document text
 */
import { useQuery } from '@tanstack/react-query';
import React, { useState } from 'react';
import { TauriAPI, type IntakeCatalogLookup, type IntakeFileMetadata } from '@/lib/tauri-api';

export interface MetadataPanelFile {
  name: string;
  path: string;
  is_dir: boolean;
}

interface MetadataPanelProps {
  selectedFile: MetadataPanelFile | null;
  selectedFiles?: MetadataPanelFile[];
}

const formatBytes = (bytes: unknown): string => {
  if (typeof bytes !== 'number' || !Number.isFinite(bytes)) return '—';
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(1)} ${units[unit]}`;
};

const asRecord = (v: unknown): Record<string, unknown> | null =>
  v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null;

const Section = ({
  title,
  children,
  defaultOpen = true,
}: {
  title: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
}) => {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <section className="border-xp-border border-b">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="hover:bg-xp-surface-light flex w-full items-center justify-between px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide"
      >
        <span>{title}</span>
        <span className="text-xp-text-muted">{open ? '−' : '+'}</span>
      </button>
      {open && <div className="px-3 pb-3 text-xs">{children}</div>}
    </section>
  );
};

const Rows = ({ rows }: { rows: Array<[string, React.ReactNode]> }) => (
  <dl className="grid grid-cols-[minmax(6rem,auto)_1fr] gap-x-3 gap-y-1">
    {rows.map(([k, v]) => (
      <React.Fragment key={k}>
        <dt className="text-xp-text-muted">{k}</dt>
        <dd className="break-all">{v ?? '—'}</dd>
      </React.Fragment>
    ))}
  </dl>
);

const ErrorLine = ({ value }: { value: unknown }) => {
  const rec = asRecord(value);
  if (!rec || typeof rec.error !== 'string') return null;
  return (
    <p className="text-xp-text-muted" title={rec.error}>
      <span
        aria-hidden="true"
        className="mr-1 inline-block h-1.5 w-1.5 rounded-full bg-amber-500"
      />
      {rec.error}
    </p>
  );
};

const Properties = ({ meta }: { meta: IntakeFileMetadata }) => {
  const p = asRecord(meta.properties);
  const entry = asRecord(meta.catalog_entry);
  const totals = asRecord(meta.catalog_totals);
  if (entry) {
    return (
      <Rows
        rows={[
          ['Name', String(entry.name ?? '')],
          ['Path', String(entry.path ?? '')],
          ['Files inside', String(totals?.files ?? '—')],
          ['Folders inside', String(totals?.dirs ?? '—')],
          ['Total size', formatBytes(totals?.bytes)],
        ]}
      />
    );
  }
  if (!p) return <p className="text-xp-text-muted">No properties returned.</p>;
  if (typeof p.error === 'string') return <ErrorLine value={p} />;
  return (
    <Rows
      rows={[
        ['Name', String(p.name ?? '')],
        ['Type', String(p.file_type ?? '')],
        ['Size', `${String(p.size_formatted ?? formatBytes(p.size))}`],
        ['Modified', String(p.modified_formatted ?? '')],
        ['Created', String(p.created_formatted ?? '')],
        ['MIME', String(p.mime_type ?? '—')],
        ['On B2', String(meta.resolved_path ?? p.path ?? '')],
      ]}
    />
  );
};

const Catalog = ({ lookup, error }: { lookup?: IntakeCatalogLookup; error?: string }) => {
  if (error) return <ErrorLine value={{ error }} />;
  if (!lookup) return <p className="text-xp-text-muted">Looking up the catalog…</p>;
  if (lookup.note) return <p className="text-xp-text-muted">{lookup.note}</p>;
  return (
    <div className="space-y-2">
      <Rows
        rows={[
          ['B2 key', lookup.b2_key ?? '—'],
          ['SHA-1', lookup.vault_object?.sha1 ?? '—'],
          ['Vault size', formatBytes(lookup.vault_object?.size ?? undefined)],
          ['Occurrences', String(lookup.count)],
        ]}
      />
      {lookup.count === 0 && (
        <p className="text-xp-text-muted">
          <span
            aria-hidden="true"
            className="mr-1 inline-block h-1.5 w-1.5 rounded-full bg-amber-500"
          />
          No recorded occurrence for this object
        </p>
      )}
      <ul className="space-y-2">
        {lookup.occurrences.map((o, i) => (
          <li key={`${o.source}-${o.path}-${i}`} className="border-xp-border rounded border p-2">
            <div className="font-medium">
              {o.source}
              {o.scope ? ` · ${o.scope}` : ''}
            </div>
            <div className="break-all">{o.path}</div>
            <div className="text-xp-text-muted mt-1 flex flex-wrap gap-x-3">
              <span>{formatBytes(o.size ?? undefined)}</span>
              {o.modtime && <span>{new Date(o.modtime).toLocaleString()}</span>}
              {o.disposition && <span>{o.disposition}</span>}
              {o.md5 && <span title={o.md5}>md5 {o.md5.slice(0, 10)}…</span>}
              {o.native_hash_kind && o.native_hash && (
                <span title={o.native_hash}>
                  {o.native_hash_kind} {o.native_hash.slice(0, 10)}…
                </span>
              )}
            </div>
          </li>
        ))}
      </ul>
      {lookup.engine_ops && lookup.engine_ops.length > 0 && (
        <div>
          <div className="text-xp-text-muted mb-1">Changes made in Intake</div>
          <ul className="list-disc pl-4">
            {lookup.engine_ops.map((op, i) => (
              <li key={i} className="break-all">
                {String(op.op)}: {String(op.vault_key_from ?? op.from ?? '')} →{' '}
                {String(op.vault_key_to ?? op.to ?? 'removed')}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

const Extracted = ({ meta }: { meta: IntakeFileMetadata }) => {
  const exif = asRecord(meta.exif);
  const image = asRecord(meta.image);
  const audio = asRecord(meta.audio);
  const media = asRecord(meta.media);
  const doc = asRecord(meta.document_text);
  const nothing = !exif && !image && !audio && !media && !doc;
  if (nothing) return <p className="text-xp-text-muted">No extractor applies to this file type.</p>;
  return (
    <div className="space-y-3">
      {image && !image.error && (
        <Rows
          rows={[
            ['Dimensions', `${String(image.width)} × ${String(image.height)}`],
            ['Format', String(image.format ?? '')],
            ['Color', String(image.color_type ?? '')],
          ]}
        />
      )}
      {exif &&
        (exif.error ? (
          <ErrorLine value={exif} />
        ) : (
          <div>
            <div className="text-xp-text-muted mb-1">EXIF ({String(exif.count)} fields)</div>
            <Rows
              rows={((exif.fields as Array<Record<string, string>>) ?? []).map((f) => [
                f.tag,
                f.value,
              ])}
            />
          </div>
        ))}
      {audio &&
        (audio.error ? (
          <ErrorLine value={audio} />
        ) : (
          <div>
            <div className="text-xp-text-muted mb-1">Audio</div>
            <Rows
              rows={[
                ['Duration', `${((Number(audio.duration_ms) || 0) / 1000).toFixed(1)} s`],
                ['Bitrate', `${String(audio.audio_bitrate_kbps ?? '—')} kbps`],
                ['Sample rate', String(audio.sample_rate ?? '—')],
                ['Channels', String(audio.channels ?? '—')],
                ...((audio.tags as Array<Record<string, string>>) ?? []).map(
                  (t) => [t.key, t.value] as [string, string],
                ),
              ]}
            />
          </div>
        ))}
      {media &&
        (media.error ? (
          <ErrorLine value={media} />
        ) : (
          <div>
            <div className="text-xp-text-muted mb-1">Container / streams (ffprobe)</div>
            <Rows
              rows={[
                ['Format', String(asRecord(media.format)?.format_long_name ?? '—')],
                ['Duration', `${Number(asRecord(media.format)?.duration ?? 0).toFixed(1)} s`],
                ...((media.streams as Array<Record<string, unknown>>) ?? []).map(
                  (s, i) =>
                    [
                      `Stream ${i}`,
                      [
                        s.codec_type,
                        s.codec_name,
                        s.width && s.height ? `${String(s.width)}×${String(s.height)}` : null,
                        s.sample_rate ? `${String(s.sample_rate)} Hz` : null,
                      ]
                        .filter(Boolean)
                        .join(' · '),
                    ] as [string, string],
                ),
              ]}
            />
          </div>
        ))}
      {doc &&
        (doc.error ? (
          <ErrorLine value={doc} />
        ) : (
          <div>
            <div className="text-xp-text-muted mb-1">
              Document text ({String(doc.chars)} characters
              {doc.truncated ? ', first 20,000 shown' : ''})
            </div>
            <pre className="bg-xp-surface-light max-h-72 overflow-auto whitespace-pre-wrap rounded p-2 text-[11px]">
              {String(doc.text ?? '')}
            </pre>
          </div>
        ))}
    </div>
  );
};

const MetadataPanel = ({ selectedFile, selectedFiles }: MetadataPanelProps) => {
  const target = selectedFile ?? selectedFiles?.[0] ?? null;
  const path = target?.path ?? '';
  const meta = useQuery({
    queryKey: ['intake-file-metadata', path],
    queryFn: () => TauriAPI.getIntakeFileMetadata(path),
    enabled: Boolean(path),
    staleTime: 60_000,
    retry: false,
  });
  const lookup = useQuery({
    queryKey: ['intake-catalog-lookup', path],
    queryFn: () => TauriAPI.getIntakeCatalogLookup(path),
    enabled: Boolean(path) && !target?.is_dir,
    staleTime: 60_000,
    retry: false,
  });

  if (!target) {
    return <p className="text-xp-text-muted p-3 text-xs">Select a file to see its metadata.</p>;
  }
  const extra = (selectedFiles?.length ?? 0) > 1 ? selectedFiles!.length - 1 : 0;
  return (
    <div className="h-full overflow-auto" data-testid="metadata-panel">
      <div className="border-xp-border border-b px-3 py-2">
        <div className="truncate text-sm font-medium" title={target.path}>
          {target.name}
        </div>
        {extra > 0 && (
          <div className="text-xp-text-muted text-xs">
            and {extra} more selected (showing the first)
          </div>
        )}
      </div>
      <Section title="Properties">
        {meta.isLoading ? (
          <p className="text-xp-text-muted">Reading…</p>
        ) : meta.error ? (
          <ErrorLine value={{ error: String((meta.error as Error).message ?? meta.error) }} />
        ) : (
          meta.data && <Properties meta={meta.data} />
        )}
      </Section>
      {!target.is_dir && (
        <Section title="Catalog history">
          <Catalog
            lookup={lookup.data}
            error={
              lookup.error ? String((lookup.error as Error).message ?? lookup.error) : undefined
            }
          />
        </Section>
      )}
      {!target.is_dir && (
        <Section title="Extracted content">
          {meta.isLoading ? (
            <p className="text-xp-text-muted">Extracting…</p>
          ) : (
            meta.data && <Extracted meta={meta.data} />
          )}
        </Section>
      )}
    </div>
  );
};

export default MetadataPanel;
