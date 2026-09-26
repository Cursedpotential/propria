// Byline: Claude Code · Opus 5 · 2026-09-18
// "Search this folder live": an explicit, capped text scan of the open storage folder, read from B2
// through the mount (engine `intake_live_folder_search`: <= 2,000 files, <= 200 MB, files > 20 MB
// skipped, never catalog:// or a bucket/storage root). Progress arrives as SSE events.
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { listenToEvent } from '@/lib/transport';
import {
  TauriAPI,
  LIVE_SEARCH_PROGRESS_EVENT,
  type LiveFolderProgress,
  type LiveFolderSearchResult,
} from '@/lib/tauri-api';
import { openSearchHit, setSelectedChatEvent } from '@/lib/chat-event-selection';

const mb = (bytes: number): string => `${(bytes / 1_048_576).toFixed(1)} MB`;

/** Mirrors the engine rule: a real storage folder at least three levels below /srv/openlist. */
export const liveSearchAllowed = (folder?: string): boolean => {
  if (!folder || folder.startsWith('catalog://') || !folder.startsWith('/srv/openlist/')) {
    return false;
  }
  return (
    folder
      .replace(/^\/srv\/openlist\//, '')
      .split('/')
      .filter(Boolean).length >= 3
  );
};

const IntakeLiveFolderSearch = ({ folder }: { folder?: string }) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState<LiveFolderProgress | null>(null);
  const [result, setResult] = useState<LiveFolderSearchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const searchId = useRef<string | null>(null);
  const stopListening = useRef<(() => void) | null>(null);
  const allowed = liveSearchAllowed(folder);

  useEffect(
    () => () => {
      stopListening.current?.();
    },
    [],
  );

  const run = async () => {
    if (!folder || !allowed || !query.trim()) return;
    const id = `live-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    searchId.current = id;
    setRunning(true);
    setError(null);
    setResult(null);
    setProgress(null);
    stopListening.current?.();
    stopListening.current = await listenToEvent<LiveFolderProgress>(
      LIVE_SEARCH_PROGRESS_EVENT,
      (p) => {
        if (p.searchId === searchId.current) setProgress(p);
      },
    );
    try {
      const r = await TauriAPI.liveFolderSearch(folder, query.trim(), id);
      if (searchId.current === id) setResult(r);
    } catch (cause) {
      if (searchId.current === id) setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      if (searchId.current === id) {
        setRunning(false);
        stopListening.current?.();
        stopListening.current = null;
      }
    }
  };

  return (
    <div className="border-xp-border mt-2 flex flex-col gap-2 border-t pt-3">
      <h3 className="text-xs font-semibold">{t('intakeChatSearch.liveTitle')}</h3>
      <p className="text-xp-text-muted text-[11px]">{t('intakeChatSearch.liveDescription')}</p>
      {!allowed && (
        <p className="text-xp-text-muted text-[11px]" data-testid="live-search-disabled">
          {t('intakeChatSearch.liveNeedsFolder')}
        </p>
      )}
      <form
        className="flex gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          void run();
        }}
      >
        <input
          value={query}
          maxLength={200}
          disabled={!allowed}
          onChange={(event) => setQuery(event.target.value)}
          aria-label={t('intakeChatSearch.liveQuery')}
          placeholder={t('intakeChatSearch.liveQuery')}
          className="border-xp-border bg-xp-surface min-w-0 flex-1 rounded border p-1 text-xs disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={!allowed || running || !query.trim()}
          className="border-xp-border rounded border px-2 text-xs disabled:opacity-50"
        >
          {t('intakeChatSearch.liveRun')}
        </button>
      </form>
      {allowed && folder && <p className="text-xp-text-muted break-all text-[10px]">{folder}</p>}
      {running && (
        <p role="status" className="text-[11px]" data-testid="live-search-progress">
          {t('intakeChatSearch.liveReading')}: {progress?.files_read ?? 0} / 2,000{' '}
          {t('intakeChatSearch.files')} · {mb(progress?.bytes_read ?? 0)} / 200 MB
        </p>
      )}
      {error && (
        <p role="alert" className="text-[11px]">
          {error}
        </p>
      )}
      {result && (
        <div className="space-y-1 text-[11px]" data-testid="live-search-result">
          <p>
            {result.matches.length} {t('intakeChatSearch.liveMatches')} · {result.files_read}{' '}
            {t('intakeChatSearch.files')} · {mb(result.bytes_read)} · {result.skipped_large}{' '}
            {t('intakeChatSearch.liveSkippedLarge')}
          </p>
          {result.stopped && (
            <p className="font-medium" data-testid="live-search-stopped">
              {t('intakeChatSearch.liveStopped')}: {result.stopped}
            </p>
          )}
          <ul className="space-y-1">
            {result.matches.map((m) => (
              <li key={`${m.file}:${m.line}`}>
                <button
                  type="button"
                  className="hover:bg-xp-surface-light border-xp-border w-full rounded border p-1 text-left"
                  onClick={() => {
                    setSelectedChatEvent(null);
                    openSearchHit(m.file);
                  }}
                >
                  <span className="block break-all font-medium">
                    {m.filename}:{m.line}
                  </span>
                  <span className="text-xp-text-muted block break-words">{m.content}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default IntakeLiveFolderSearch;
