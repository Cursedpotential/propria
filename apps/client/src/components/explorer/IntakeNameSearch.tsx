// Byline: Claude Code · Opus 5 · 2026-09-22
// "When I click the magnifying glass I should be able to search for a folder name … the whole
// level, every fucking thing" (owner 2026-09-22 18:54 EDT). Name search over every folder and file
// the catalog recorded and every object in B2, whatever folder happens to be open.
//
// The scope is a dropdown, the way file search has worked for twenty-five years (owner 18:55 EDT),
// and the one last used is remembered. Each row shows both places the file has lived — "now" and
// "was" — because the exports and the dedupe renamed and moved things (owner 18:58 EDT).
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
import {
  TauriAPI,
  NAME_SEARCH_SCOPES,
  SCOPES_NEEDING_FOLDER,
  type NameSearchHit,
  type NameSearchKinds,
  type NameSearchResult,
  type NameSearchScope,
  type NameSearchSort,
} from '@/lib/tauri-api';
import { STORAGE_KEYS } from '@/lib/storage-keys';
import { openSearchHit, setSelectedChatEvent } from '@/lib/chat-event-selection';

interface Prefs {
  scope: NameSearchScope;
  kinds: NameSearchKinds;
  sort: NameSearchSort;
}

const DEFAULT_PREFS: Prefs = { scope: 'everything', kinds: 'both', sort: 'name' };

const SCOPE_IDS = NAME_SEARCH_SCOPES.map((s) => s.id) as readonly string[];

export const readPrefs = (raw: string | null): Prefs => {
  if (!raw) return DEFAULT_PREFS;
  try {
    const parsed = JSON.parse(raw) as Partial<Prefs>;
    return {
      scope: SCOPE_IDS.includes(parsed.scope ?? '')
        ? (parsed.scope as NameSearchScope)
        : 'everything',
      kinds: ['both', 'folders', 'files'].includes(parsed.kinds ?? '')
        ? (parsed.kinds as NameSearchKinds)
        : 'both',
      sort: parsed.sort === 'path' ? 'path' : 'name',
    };
  } catch {
    return DEFAULT_PREFS;
  }
};

const loadPrefs = (): Prefs => {
  try {
    return readPrefs(window.localStorage.getItem(STORAGE_KEYS.NAME_SEARCH_PREFS));
  } catch {
    return DEFAULT_PREFS;
  }
};

const savePrefs = (prefs: Prefs): void => {
  try {
    window.localStorage.setItem(STORAGE_KEYS.NAME_SEARCH_PREFS, JSON.stringify(prefs));
  } catch {
    // A browser with site data blocked still searches; it just forgets the scope.
  }
};

const bytes = (n: number | null): string => {
  if (n === null || n <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = n;
  let i = 0;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i += 1;
  }
  return `${value < 10 && i > 0 ? value.toFixed(1) : Math.round(value)} ${units[i]}`;
};

const FolderIcon = () => (
  <svg
    width="13"
    height="13"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    style={{ flexShrink: 0 }}
  >
    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
  </svg>
);

const FileIcon = () => (
  <svg
    width="13"
    height="13"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
    style={{ flexShrink: 0 }}
  >
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <polyline points="14 2 14 8 20 8" />
  </svg>
);

interface Props {
  /** Where a clicked hit opens. */
  navigateToPath: (path: string) => void;
  /** The active pane's folder, for the folder-relative scopes. */
  activePaneRoot?: string;
}

export interface IntakeNameSearchHandle {
  focus: () => void;
}

const IntakeNameSearch = forwardRef<IntakeNameSearchHandle, Props>(function IntakeNameSearch(
  { navigateToPath, activePaneRoot },
  ref,
) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const requestId = useRef(0);
  const [query, setQuery] = useState('');
  const [prefs, setPrefs] = useState<Prefs>(loadPrefs);
  const [result, setResult] = useState<NameSearchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useImperativeHandle(ref, () => ({ focus: () => inputRef.current?.focus() }));

  useEffect(() => {
    savePrefs(prefs);
  }, [prefs]);

  const needsFolder = SCOPES_NEEDING_FOLDER.includes(prefs.scope);
  const folderMissing = needsFolder && !activePaneRoot;

  const run = useCallback(async () => {
    const q = query.trim();
    if (q.length < 2) return;
    const id = ++requestId.current;
    setLoading(true);
    setError(null);
    try {
      const response = await TauriAPI.searchNames({
        query: q,
        scope: prefs.scope,
        kinds: prefs.kinds,
        sort: prefs.sort,
        ...(needsFolder && activePaneRoot ? { path: activePaneRoot } : {}),
      });
      if (id === requestId.current) setResult(response);
    } catch (cause) {
      if (id === requestId.current) {
        setResult(null);
        setError(cause instanceof Error ? cause.message : String(cause));
      }
    } finally {
      if (id === requestId.current) setLoading(false);
    }
  }, [query, prefs, needsFolder, activePaneRoot]);

  const openHit = useCallback(
    (hit: NameSearchHit) => {
      setSelectedChatEvent(null);
      if (hit.kind === 'folder') {
        navigateToPath(hit.navigate_to);
        return;
      }
      // A file: open its folder and select it once that listing has loaded.
      if (hit.select) {
        openSearchHit(hit.select);
      } else {
        navigateToPath(hit.navigate_to);
      }
    },
    [navigateToPath],
  );

  const summary = useMemo(() => {
    if (!result) return null;
    return t('intakeNameSearch.summary', {
      folders: result.total_folders.toLocaleString('en-US'),
      files: result.total_files.toLocaleString('en-US'),
      ms: result.elapsed_ms.toLocaleString('en-US'),
    });
  }, [result, t]);

  return (
    <section className="flex flex-col gap-2" aria-label={t('intakeNameSearch.title')}>
      <h2 className="text-sm font-semibold">{t('intakeNameSearch.title')}</h2>
      <p className="text-xp-text-muted text-[11px]">{t('intakeNameSearch.hint')}</p>
      <form
        className="flex flex-col gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          void run();
        }}
      >
        <input
          ref={inputRef}
          value={query}
          maxLength={200}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            // Enter submits here; the explorer's global key handling must not swallow it.
            if (event.key === 'Enter') {
              event.preventDefault();
              event.stopPropagation();
              void run();
            }
          }}
          aria-label={t('intakeNameSearch.query')}
          placeholder={t('intakeNameSearch.placeholder')}
          data-testid="name-search-query"
          className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
        />
        <label className="text-xp-text-muted flex flex-col text-[11px]">
          {t('intakeNameSearch.scope')}
          <select
            value={prefs.scope}
            aria-label={t('intakeNameSearch.scope')}
            data-testid="name-search-scope"
            onChange={(event) =>
              setPrefs((p) => ({ ...p, scope: event.target.value as NameSearchScope }))
            }
            className="border-xp-border bg-xp-surface text-xp-text rounded border p-1 text-xs"
          >
            {NAME_SEARCH_SCOPES.map((scope) => (
              <option key={scope.id} value={scope.id}>
                {t(scope.labelKey)}
              </option>
            ))}
          </select>
        </label>
        <div className="grid grid-cols-2 gap-2">
          <label className="text-xp-text-muted flex flex-col text-[11px]">
            {t('intakeNameSearch.kinds')}
            <select
              value={prefs.kinds}
              aria-label={t('intakeNameSearch.kinds')}
              data-testid="name-search-kinds"
              onChange={(event) =>
                setPrefs((p) => ({ ...p, kinds: event.target.value as NameSearchKinds }))
              }
              className="border-xp-border bg-xp-surface text-xp-text rounded border p-1 text-xs"
            >
              <option value="both">{t('intakeNameSearch.kindsBoth')}</option>
              <option value="folders">{t('intakeNameSearch.kindsFolders')}</option>
              <option value="files">{t('intakeNameSearch.kindsFiles')}</option>
            </select>
          </label>
          <label className="text-xp-text-muted flex flex-col text-[11px]">
            {t('intakeNameSearch.sort')}
            <select
              value={prefs.sort}
              aria-label={t('intakeNameSearch.sort')}
              data-testid="name-search-sort"
              onChange={(event) =>
                setPrefs((p) => ({ ...p, sort: event.target.value as NameSearchSort }))
              }
              className="border-xp-border bg-xp-surface text-xp-text rounded border p-1 text-xs"
            >
              <option value="name">{t('intakeNameSearch.sortName')}</option>
              <option value="path">{t('intakeNameSearch.sortPath')}</option>
            </select>
          </label>
        </div>
        {needsFolder && (
          <p className="text-xp-text-muted break-all text-[10px]" data-testid="name-search-folder">
            {folderMissing ? t('intakeNameSearch.noFolder') : activePaneRoot}
          </p>
        )}
        <button
          type="submit"
          disabled={loading || query.trim().length < 2}
          className="border-xp-border rounded border p-2 text-sm disabled:opacity-50"
        >
          {loading ? t('intakeNameSearch.running') : t('intakeNameSearch.run')}
        </button>
      </form>

      {error && (
        <p role="alert" className="text-xs">
          {t('intakeNameSearch.failed')}: {error}
        </p>
      )}

      {result && (
        <>
          <p className="text-xp-text-muted text-[11px]" data-testid="name-search-summary">
            {summary}
            {result.capped ? ` ${t('intakeNameSearch.capped')}` : ''}
            {result.hits.length < result.total_folders + result.total_files
              ? ` · ${t('intakeNameSearch.showingFirst', { shown: result.hits.length })}`
              : ''}
          </p>
          {result.notes.map((note) => (
            <p key={note} className="text-xp-text-muted text-[11px]">
              {note}
            </p>
          ))}
          {result.hits.length === 0 && (
            <p className="text-xs" data-testid="name-search-empty">
              {t('intakeNameSearch.noResults')}
            </p>
          )}
          <ul className="space-y-1" data-testid="name-search-hits">
            {result.hits.map((hit) => (
              <li key={`${hit.kind}:${hit.path}`}>
                <button
                  type="button"
                  onClick={() => openHit(hit)}
                  title={hit.path}
                  className="hover:bg-xp-surface-light border-xp-border w-full rounded border p-1.5 text-left text-xs"
                >
                  <span className="flex items-center gap-1.5">
                    {hit.kind === 'folder' ? <FolderIcon /> : <FileIcon />}
                    <span className="min-w-0 flex-1 break-all font-medium">{hit.name}</span>
                    {hit.size !== null && hit.size > 0 && (
                      <span className="text-xp-text-muted text-[10px]">{bytes(hit.size)}</span>
                    )}
                  </span>
                  <span className="text-xp-text-muted mt-0.5 block break-all text-[10px]">
                    {hit.path}
                  </span>
                  {hit.now && hit.was && hit.now !== hit.was && (
                    <span className="text-xp-text-muted mt-0.5 block break-all text-[10px]">
                      {t('intakeNameSearch.now')}: {hit.now} · {t('intakeNameSearch.was')}:{' '}
                      {hit.was}
                    </span>
                  )}
                  {hit.member_of && (
                    <span className="text-xp-text-muted mt-0.5 block text-[10px]">
                      {t('intakeNameSearch.inside', { archive: hit.member_of })}
                    </span>
                  )}
                  {hit.flag && !hit.member_of && (
                    <span className="text-xp-text-muted mt-0.5 block break-words text-[10px]">
                      {hit.flag}
                    </span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
});

export default IntakeNameSearch;
