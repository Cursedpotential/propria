import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { TauriAPI } from '@/lib/tauri-api';
import { isTauri } from '@/lib/transport';
import { getIntakeEngineInfo } from '@/lib/tauri-api/intake-engine';
import {
  addSearchHitsToSelection,
  filesystemParentPath,
  searchFilesystemIndex,
  type FilesystemSearchResponse,
} from '@/lib/filesystem-index';
import { searchCocoIndex, runLakeQuery, type LakeQueryResult } from '@/lib/intake-backend';
import ResultsGrid from '@/components/panels/ResultsGrid';
// Byline: Claude Code · Opus 5 · 2026-09-27 -- mount name search on the native path.
// Docstore note:intake_native_invoke_reachability_20260924 recorded that LeftSidebar routes
// Tauri users here, and that this panel never rendered IntakeNameSearch, so the magnifying
// glass could search file CONTENT but never answer "where is that folder?". The engine
// command intake_search_names has been registered in src-tauri/main.rs the whole time.
import IntakeNameSearch, { type IntakeNameSearchHandle } from './IntakeNameSearch';

// Byline: Claude Code · Sonnet · 2026-09-14
// Byline: Claude Code · Sonnet 5 · 2026-09-14 -- added cocoindex + duckdb methods
type SearchMode = 'rg' | 'keyword' | 'hybrid' | 'cocoindex' | 'duckdb';

interface Props {
  navigateToPath: (path: string) => void;
  /** Active pane's current directory; the local rg-style method runs under this root. */
  activePaneRoot?: string;
}
export interface IntakeSearchHandle {
  focus: () => void;
}

/** Local content matches are normalized into the same shape as index hits so
 * selection/open-location/provenance rendering stay a single code path. */
const rgMatchesToResponse = (
  query: string,
  root: string,
  matches: { file: string; line: number; content: string; filename: string }[],
): FilesystemSearchResponse => ({
  query,
  backend: 'rg',
  collection: root,
  target_vector: null,
  coverage: 'unknown',
  hits: matches.map((match) => ({
    object_id: `rg:${match.file}:${match.line}`,
    source_id: 'rg-local',
    source_path: match.file,
    document_id: match.file,
    chunk_id: `line-${match.line}`,
    filename: match.filename,
    text: match.content,
    score: 0,
  })),
});

/** CocoIndex/Parquet-lake hits normalized into the same display shape; the
 * relative path is not an absolute filesystem path, so "Open location" stays
 * disabled for these rows (filesystemParentPath rejects non-absolute input). */
const cocoIndexToResponse = (
  query: string,
  snapshot: string | null,
  hits: {
    score: number;
    document_id: string;
    chunk_id: string;
    relative_path: string;
    filename: string;
    text: string;
  }[],
): FilesystemSearchResponse => ({
  query,
  backend: 'cocoindex',
  collection: snapshot ?? 'active snapshot',
  target_vector: null,
  coverage: 'unknown',
  hits: hits.map((hit) => ({
    object_id: `cocoindex:${hit.document_id}:${hit.chunk_id}`,
    source_id: 'cocoindex-parquet-lake',
    source_path: hit.relative_path,
    document_id: hit.document_id,
    chunk_id: hit.chunk_id,
    filename: hit.filename,
    text: hit.text,
    score: hit.score,
  })),
});

const IntakeFilesystemSearchPanel = forwardRef<IntakeSearchHandle, Props>(
  ({ navigateToPath, activePaneRoot }, ref) => {
    const { t } = useTranslation();
    const inputRef = useRef<HTMLInputElement>(null);
    const nameSearchRef = useRef<IntakeNameSearchHandle>(null);
    const requestId = useRef(0);
    const [query, setQuery] = useState('');
    const [mode, setMode] = useState<SearchMode>('hybrid');
    const [result, setResult] = useState<FilesystemSearchResponse | null>(null);
    const [lakeResult, setLakeResult] = useState<LakeQueryResult | null>(null);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const rgAvailable = Boolean(activePaneRoot);
    // Hosted Intake (browser + engine): ask the engine whether an index service is connected
    // instead of sending searches that can only fail. The CocoIndex/DuckDB lake modes talk to a
    // loopback backend that a hosted page cannot reach unless a URL was built in.
    // (Claude Code · Opus 5 · 2026-09-18)
    const hostedWeb = !isTauri();
    const [fsIndexConnected, setFsIndexConnected] = useState<boolean | null>(
      hostedWeb ? null : true,
    );
    useEffect(() => {
      if (!hostedWeb) return;
      let live = true;
      getIntakeEngineInfo()
        .then((info) => live && setFsIndexConnected(Boolean(info.filesystem_search)))
        .catch(() => live && setFsIndexConnected(false));
      return () => {
        live = false;
      };
    }, [hostedWeb]);
    const lakeReachable = !hostedWeb || Boolean(import.meta.env.VITE_INTAKE_BACKEND_API_URL);
    const indexNotConnected =
      ((mode === 'keyword' || mode === 'hybrid') && fsIndexConnected === false) ||
      ((mode === 'cocoindex' || mode === 'duckdb') && !lakeReachable);
    // Focus lands on name search, the same as the hosted panel: the magnifying glass is a
    // question about where something IS, not about what is inside it.
    useImperativeHandle(ref, () => ({ focus: () => nameSearchRef.current?.focus() }));
    useEffect(
      () => () => {
        requestId.current++;
      },
      [],
    );

    const search = async () => {
      if (indexNotConnected) return;
      const id = ++requestId.current;
      setLoading(true);
      setError(null);
      setResult(null);
      setLakeResult(null);
      setSelected(new Set());
      try {
        if (mode === 'rg') {
          if (!activePaneRoot) throw new Error(t('intakeSearch.rgNeedsFolder'));
          const matches = await TauriAPI.grepSearch(query.trim(), activePaneRoot, 100);
          if (id === requestId.current) {
            setResult(rgMatchesToResponse(query.trim(), activePaneRoot, matches));
          }
        } else if (mode === 'keyword' || mode === 'hybrid') {
          const response = await searchFilesystemIndex(query, mode);
          if (id === requestId.current) setResult(response);
        } else if (mode === 'cocoindex') {
          const response = await searchCocoIndex(query, { limit: 20 });
          if (id === requestId.current) {
            setResult(cocoIndexToResponse(response.query, response.snapshot, response.hits));
          }
        } else if (mode === 'duckdb') {
          const response = await runLakeQuery(query, 100);
          if (id === requestId.current) setLakeResult(response);
        }
      } catch (cause) {
        if (id === requestId.current) {
          setError(cause instanceof Error ? cause.message : String(cause));
        }
      } finally {
        if (id === requestId.current) setLoading(false);
      }
    };

    return (
      <section
        className="text-xp-text flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3"
        aria-label={t('intakeSearch.title')}
      >
        {/* Names first: clicking the magnifying glass is asking "where is that folder?". */}
        <IntakeNameSearch
          ref={nameSearchRef}
          navigateToPath={navigateToPath}
          activePaneRoot={activePaneRoot}
        />
        <h2 className="border-xp-border border-t pt-3 text-sm font-semibold">
          {t('intakeSearch.title')}
        </h2>
        <p className="text-xp-text-muted text-xs">{t('intakeSearch.description')}</p>
        <form
          className="flex flex-col gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void search();
          }}
        >
          {mode === 'duckdb' ? (
            <textarea
              value={query}
              maxLength={8000}
              rows={4}
              onChange={(event) => setQuery(event.target.value)}
              aria-label={t('intakeSearch.duckdbSql')}
              placeholder="SELECT relative_path, title FROM documents LIMIT 20"
              className="border-xp-border bg-xp-surface rounded border p-2 font-mono text-xs"
            />
          ) : (
            <input
              ref={inputRef}
              value={query}
              maxLength={4096}
              onChange={(event) => setQuery(event.target.value)}
              aria-label={t('intakeSearch.query')}
              placeholder={t('intakeSearch.query')}
              className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
            />
          )}
          <select
            value={mode}
            aria-label={t('intakeSearch.mode')}
            onChange={(event) => setMode(event.target.value as SearchMode)}
            className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
          >
            <option value="hybrid">{t('intakeSearch.hybrid')}</option>
            <option value="keyword">{t('intakeSearch.keyword')}</option>
            <option value="cocoindex">{t('intakeSearch.cocoindex')}</option>
            <option value="duckdb">{t('intakeSearch.duckdb')}</option>
            <option value="rg" disabled={!rgAvailable}>
              {t('intakeSearch.rg')}
            </option>
          </select>
          {indexNotConnected && (
            <p role="status" className="text-xp-text-muted text-xs">
              {t('intakeSearch.notConnected')}
            </p>
          )}
          {mode === 'rg' && !rgAvailable && (
            <p className="text-xp-text-muted text-xs">{t('intakeSearch.rgNeedsFolder')}</p>
          )}
          <button
            type="submit"
            disabled={loading || !query.trim() || indexNotConnected}
            className="border-xp-border rounded border p-2 text-sm disabled:opacity-50"
          >
            {loading ? t('intakeSearch.searching') : t('intakeSearch.search')}
          </button>
        </form>
        <p className="text-xp-text-muted text-xs" role="status">
          {t('intakeSearch.coverage')}: {result?.coverage ?? t('intakeSearch.unknown')}
        </p>
        {error && (
          <p role="alert" className="text-xs">
            {t('intakeSearch.unavailable')}: {error}
          </p>
        )}
        {result && (
          <>
            <p className="break-all text-xs">
              {result.backend} · {result.collection} ·{' '}
              {result.target_vector ?? t('intakeSearch.keyword')}
            </p>
            <button
              type="button"
              disabled={selected.size === 0}
              className="border-xp-border rounded border p-2 text-xs disabled:opacity-50"
              onClick={() =>
                addSearchHitsToSelection(result.hits.filter((hit) => selected.has(hit.object_id)))
              }
            >
              {t('intakeSearch.addSelection', { count: selected.size })}
            </button>
            {result.hits.length === 0 && <p className="text-xs">{t('intakeSearch.noResults')}</p>}
            <ul className="space-y-3">
              {result.hits.map((hit) => {
                const parent = filesystemParentPath(hit.source_path);
                return (
                  <li key={hit.object_id} className="border-xp-border rounded border p-2 text-xs">
                    <label className="flex items-start gap-2">
                      <input
                        type="checkbox"
                        checked={selected.has(hit.object_id)}
                        onChange={(event) => {
                          setSelected((previous) => {
                            const next = new Set(previous);
                            if (event.target.checked) next.add(hit.object_id);
                            else next.delete(hit.object_id);
                            return next;
                          });
                        }}
                      />
                      <span className="break-all font-medium">{hit.filename}</span>
                    </label>
                    <p className="mt-1 break-all">{hit.source_path}</p>
                    <p
                      className="text-xp-text-muted mt-1 break-all"
                      title={t('intakeSearch.provenance')}
                    >
                      {result.backend} · {result.collection} · {hit.source_id} ·{' '}
                      {t('intakeSearch.score')}: {hit.score.toPrecision(4)}
                    </p>
                    <p className="mt-2 whitespace-pre-wrap break-words">
                      {hit.text.slice(0, 800)}
                      {hit.text.length > 800 ? '…' : ''}
                    </p>
                    <button
                      type="button"
                      disabled={!parent}
                      title={!parent ? t('intakeSearch.unmapped') : parent}
                      className="border-xp-border mt-2 rounded border p-1 disabled:opacity-50"
                      onClick={() => {
                        if (parent) navigateToPath(parent);
                      }}
                    >
                      {t('intakeSearch.openLocation')}
                    </button>
                  </li>
                );
              })}
            </ul>
          </>
        )}
        {lakeResult && (
          <>
            <p className="break-all text-xs">
              duckdb · {lakeResult.views_available.join(', ') || 'no views yet'} ·{' '}
              {lakeResult.row_count} rows{lakeResult.truncated ? ' (truncated)' : ''}
            </p>
            <ResultsGrid
              columns={lakeResult.columns.map((name) => ({ key: name, title: name }))}
              rows={lakeResult.rows.map((row) =>
                Object.fromEntries(
                  lakeResult.columns.map((name, index) => [
                    name,
                    row[index] as string | number | boolean | null,
                  ]),
                ),
              )}
              storageKey="intake-duckdb-grid-engine"
              emptyMessage={t('intakeSearch.noResults')}
            />
          </>
        )}
      </section>
    );
  },
);
IntakeFilesystemSearchPanel.displayName = 'IntakeFilesystemSearchPanel';
export default IntakeFilesystemSearchPanel;
