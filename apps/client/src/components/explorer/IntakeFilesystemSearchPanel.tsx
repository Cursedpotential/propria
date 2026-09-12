import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  addSearchHitsToSelection,
  filesystemParentPath,
  searchFilesystemIndex,
  type FilesystemSearchResponse,
} from '@/lib/filesystem-index';

interface Props {
  navigateToPath: (path: string) => void;
}
export interface IntakeSearchHandle {
  focus: () => void;
}

const IntakeFilesystemSearchPanel = forwardRef<IntakeSearchHandle, Props>(
  ({ navigateToPath }, ref) => {
    const { t } = useTranslation();
    const inputRef = useRef<HTMLInputElement>(null);
    const requestId = useRef(0);
    const [query, setQuery] = useState('');
    const [mode, setMode] = useState<'keyword' | 'hybrid'>('hybrid');
    const [result, setResult] = useState<FilesystemSearchResponse | null>(null);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    useImperativeHandle(ref, () => ({ focus: () => inputRef.current?.focus() }));
    useEffect(
      () => () => {
        requestId.current++;
      },
      [],
    );

    const search = async () => {
      const id = ++requestId.current;
      setLoading(true);
      setError(null);
      setResult(null);
      setSelected(new Set());
      try {
        const response = await searchFilesystemIndex(query, mode);
        if (id === requestId.current) setResult(response);
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
        <h2 className="text-sm font-semibold">{t('intakeSearch.title')}</h2>
        <p className="text-xp-text-muted text-xs">{t('intakeSearch.description')}</p>
        <form
          className="flex flex-col gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void search();
          }}
        >
          <input
            ref={inputRef}
            value={query}
            maxLength={4096}
            onChange={(event) => setQuery(event.target.value)}
            aria-label={t('intakeSearch.query')}
            placeholder={t('intakeSearch.query')}
            className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
          />
          <select
            value={mode}
            aria-label={t('intakeSearch.mode')}
            onChange={(event) => setMode(event.target.value as 'hybrid' | 'keyword')}
            className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
          >
            <option value="hybrid">{t('intakeSearch.hybrid')}</option>
            <option value="keyword">{t('intakeSearch.keyword')}</option>
          </select>
          <button
            type="submit"
            disabled={loading || !query.trim()}
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
                    <p className="text-xp-text-muted mt-1 break-all">
                      {hit.source_id} · {t('intakeSearch.score')}: {hit.score.toPrecision(4)}
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
      </section>
    );
  },
);
IntakeFilesystemSearchPanel.displayName = 'IntakeFilesystemSearchPanel';
export default IntakeFilesystemSearchPanel;
