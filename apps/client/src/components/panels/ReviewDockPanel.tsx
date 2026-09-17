/**
 * Byline: Claude Code · Sonnet 5 · 2026-09-14
 *
 * Native metadata panel for the selected file(s) -- replaces the earlier
 * iframe-of-a-separate-app version. Owner directive 2026-09-14 22:03-22:08 EDT:
 * "IT HAS AN IFRAME IN AN IFRAME IN AN IFRAME" is rejected; this panel is a
 * first-class component in Xplorer's own React tree, reads Xplorer's own
 * selection props directly (no postMessage, no second app), and calls the
 * `casebible-corpus serve` API through the one backend adapter
 * (`@/lib/intake-backend`) -- the renderer never holds a database credential.
 *
 * Owner directive 2026-09-14 22:16 EDT: show exactly the fields the backend
 * really extracts today (the Parquet `documents` row) -- no invented EXIF/PDF/
 * Office/media fields. A field CLASS the backend does not extract yet (photo
 * EXIF/XMP, HEIC internals, PDF info/XMP, Office properties, media tags) gets
 * one small "not extracted" flag, never a fake value or a banner. No "Extract
 * metadata" action is wired here because no such backend endpoint exists yet;
 * when one does, it is a one-line addition, not a redesign.
 */
import { useQueries } from '@tanstack/react-query';
import { lookupPath, type LookupResult } from '@/lib/intake-backend';
import ResultsGrid, { type GridRow } from './ResultsGrid';

export interface MetadataPanelFile {
  name: string;
  path: string;
  is_dir: boolean;
}

interface ReviewDockPanelProps {
  selectedFile: MetadataPanelFile | null;
  selectedFiles?: MetadataPanelFile[];
}

const NOT_EXTRACTED_CATEGORIES = [
  'Photo EXIF / XMP',
  'HEIC internals',
  'PDF info / XMP',
  'Office document properties',
  'Media (audio/video) tags',
];

const formatBytes = (bytes?: number): string => {
  if (bytes === undefined || bytes === null) return '—';
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes / 1024;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(1)} ${units[unitIndex]}`;
};

const DetailCard = ({ path, result }: { path: string; result: LookupResult | undefined }) => {
  if (!result) {
    return (
      <p className="text-xp-text-muted text-xs" role="status">
        Looking up catalog row…
      </p>
    );
  }
  if (!result.found) {
    return (
      <div className="border-xp-border rounded border p-2 text-xs">
        <p className="break-all font-medium">{path}</p>
        <p className="text-xp-text-muted mt-1 flex items-center gap-1">
          <span aria-hidden="true">⚠</span>
          {result.reason === 'path_outside_configured_source'
            ? 'Outside the backend-configured source; no catalog row possible.'
            : result.reason === 'no_active_index_snapshot'
              ? 'No index has completed for this source yet.'
              : 'Not indexed yet.'}
        </p>
      </div>
    );
  }
  const tagLists: [string, string[] | undefined][] = [
    ['People', result.people],
    ['Organizations', result.organizations],
    ['Locations', result.locations],
    ['Topics', result.topics],
    ['Keywords', result.keywords],
  ];
  return (
    <div className="border-xp-border flex flex-col gap-2 rounded border p-3 text-xs">
      <p className="break-all text-sm font-semibold">{result.filename}</p>
      <p className="text-xp-text-muted break-all">{result.relative_path}</p>
      <dl className="grid grid-cols-2 gap-x-3 gap-y-1">
        <dt className="text-xp-text-muted">Size</dt>
        <dd>{formatBytes(result.byte_size)}</dd>
        <dt className="text-xp-text-muted">Content SHA-256</dt>
        <dd className="break-all">{result.content_sha256?.slice(0, 16)}…</dd>
        <dt className="text-xp-text-muted">Source created</dt>
        <dd>{result.source_created_at ?? '—'}</dd>
        <dt className="text-xp-text-muted">Source modified</dt>
        <dd>{result.source_modified_at ?? '—'}</dd>
        <dt className="text-xp-text-muted">Indexed at</dt>
        <dd>{result.indexed_at ?? '—'}</dd>
        <dt className="text-xp-text-muted">Document type</dt>
        <dd>{result.document_type ?? 'unknown'}</dd>
        <dt className="text-xp-text-muted">Document date</dt>
        <dd>
          {result.document_date ?? 'unknown'}
          {result.date_basis ? ` (${result.date_basis})` : ''}
        </dd>
        <dt className="text-xp-text-muted">Review state</dt>
        <dd>{result.review_state ?? 'unknown'}</dd>
        <dt className="text-xp-text-muted">Index status</dt>
        <dd>{result.index_status ?? 'unknown'}</dd>
        <dt className="text-xp-text-muted">Extraction method</dt>
        <dd>{result.extraction_method ?? 'unknown'}</dd>
        <dt className="text-xp-text-muted">Chunks / chars</dt>
        <dd>
          {result.chunk_count ?? 0} / {result.text_char_count ?? 0}
        </dd>
        <dt className="text-xp-text-muted">Summary coverage</dt>
        <dd>
          {result.summary_coverage ?? 'unknown'}
          {typeof result.summary_coverage_ratio === 'number'
            ? ` (${(result.summary_coverage_ratio * 100).toFixed(0)}%)`
            : ''}
        </dd>
        <dt className="text-xp-text-muted">Confidence</dt>
        <dd>{result.confidence !== undefined ? result.confidence.toFixed(2) : '—'}</dd>
      </dl>
      {result.title && (
        <p>
          <span className="text-xp-text-muted">Title: </span>
          {result.title}
        </p>
      )}
      {result.short_summary && (
        <p>
          <span className="text-xp-text-muted">Summary: </span>
          {result.short_summary}
        </p>
      )}
      {tagLists
        .filter(([, values]) => values && values.length > 0)
        .map(([label, values]) => (
          <p key={label}>
            <span className="text-xp-text-muted">{label}: </span>
            {values!.join(', ')}
          </p>
        ))}
      <div>
        <span className="text-xp-text-muted">Duplicates in this corpus: </span>
        {result.duplicates.length === 0 ? (
          '0'
        ) : (
          <span>
            {result.duplicates.length} —{' '}
            {result.duplicates
              .slice(0, 5)
              .map((duplicate) => duplicate.relative_path)
              .join('; ')}
            {result.duplicates.length > 5 ? '…' : ''}
          </span>
        )}
      </div>
    </div>
  );
};

const NotExtractedFlags = () => (
  <div className="border-xp-border rounded border p-2 text-xs">
    <p className="text-xp-text-muted mb-1">Not extracted by this backend yet:</p>
    <ul className="flex flex-wrap gap-x-3 gap-y-1">
      {NOT_EXTRACTED_CATEGORIES.map((category) => (
        <li key={category} className="flex items-center gap-1">
          <span aria-hidden="true">⚠</span>
          {category}
        </li>
      ))}
    </ul>
  </div>
);

const ReviewDockPanel = ({ selectedFile, selectedFiles }: ReviewDockPanelProps) => {
  const files = (
    selectedFiles && selectedFiles.length > 0 ? selectedFiles : selectedFile ? [selectedFile] : []
  ).filter((file) => !file.is_dir);

  const queries = useQueries({
    queries: files.map((file) => ({
      queryKey: ['intake-lookup', file.path],
      queryFn: () => lookupPath(file.path),
      staleTime: 30_000,
    })),
  });

  if (files.length === 0) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-3 p-3">
        <p className="text-xp-text-muted text-xs" role="status">
          No file selected.
        </p>
        <NotExtractedFlags />
      </div>
    );
  }

  if (files.length === 1) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
        <h2 className="text-sm font-semibold">Metadata</h2>
        <DetailCard path={files[0].path} result={queries[0]?.data} />
        <NotExtractedFlags />
      </div>
    );
  }

  const rows: GridRow[] = files.map((file, index) => {
    const result = queries[index]?.data;
    return {
      filename: file.name,
      path: file.path,
      found: result ? (result.found ? 'yes' : 'no') : 'loading…',
      title: result?.title ?? '',
      document_type: result?.document_type ?? '',
      byte_size: result?.byte_size ?? null,
      confidence: result?.confidence ?? null,
      review_state: result?.review_state ?? '',
      duplicates: result ? result.duplicates.length : null,
    };
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
      <h2 className="text-sm font-semibold">Metadata ({files.length} files)</h2>
      <ResultsGrid
        columns={[
          { key: 'filename', title: 'File', width: 180 },
          { key: 'found', title: 'Indexed', width: 80 },
          { key: 'title', title: 'Title', width: 160 },
          { key: 'document_type', title: 'Type', width: 110 },
          { key: 'byte_size', title: 'Bytes', width: 90 },
          { key: 'confidence', title: 'Confidence', width: 90 },
          { key: 'review_state', title: 'Review state', width: 110 },
          { key: 'duplicates', title: 'Duplicates', width: 90 },
        ]}
        rows={rows}
        storageKey="intake-metadata-grid-engine"
        emptyMessage="No files selected."
      />
      <NotExtractedFlags />
    </div>
  );
};

export default ReviewDockPanel;
