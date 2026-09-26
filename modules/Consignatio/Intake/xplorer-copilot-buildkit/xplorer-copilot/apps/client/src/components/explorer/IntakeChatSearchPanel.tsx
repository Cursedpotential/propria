// Byline: Claude Code · Opus 5 · 2026-09-18
// Hosted Content Search, index-first (owner 2026-09-18 19:56 EDT "Search yea"): answers come from
// the chats index (Postgres full-text + Weaviate hybrid, merged by the engine), never from reading
// B2. Reading files is the separate, capped "Search this folder live" action below it.
import { useQuery } from '@tanstack/react-query';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  TauriAPI,
  type ChatPerson,
  type ChatSearchHit,
  type ChatSearchResponse,
} from '@/lib/tauri-api';
import { openSearchHit, setSelectedChatEvent } from '@/lib/chat-event-selection';
import IntakeLiveFolderSearch from './IntakeLiveFolderSearch';
// Name search across the whole catalog and all of B2 (Claude Code · Opus 5 · 2026-09-22).
import IntakeNameSearch, { type IntakeNameSearchHandle } from './IntakeNameSearch';
import type { IntakeSearchHandle } from './IntakeFilesystemSearchPanel';

interface Props {
  navigateToPath: (path: string) => void;
  /** Active pane's current directory (the only folder "Search this folder live" may read). */
  activePaneRoot?: string;
}

const fmt = (n: number | null | undefined): string =>
  typeof n === 'number' ? n.toLocaleString('en-US') : '…';

/** Snippets mark matched words with U+27E6 / U+27E7 (engine ts_headline). */
const Snippet = ({ text }: { text: string }) => (
  <>
    {text.split(/(⟦[^⟧]*⟧)/).map((part, i) =>
      part.startsWith('⟦') ? (
        <mark key={i} className="bg-xp-accent/20 text-xp-text rounded px-0.5">
          {part.slice(1, -1)}
        </mark>
      ) : (
        <span key={i}>{part}</span>
      ),
    )}
  </>
);

const TagChips = ({ hit }: { hit: ChatSearchHit }) => (
  <span className="flex flex-wrap gap-1">
    {hit.tags.map((tag) => (
      <span
        key={tag.tag}
        title={`${tag.tag}: ${tag.confidence}${tag.ref_type ? ` (${tag.ref_type})` : ''}`}
        className={`rounded border px-1 text-[10px] ${
          tag.confidence === 'strong' ? 'border-xp-accent' : 'border-xp-border text-xp-text-muted'
        }`}
      >
        {tag.tag}
        {tag.confidence !== 'strong' ? ` · ${tag.confidence}` : ''}
      </span>
    ))}
  </span>
);

const IntakeChatSearchPanel = forwardRef<IntakeSearchHandle, Props>(
  ({ navigateToPath, activePaneRoot }, ref) => {
    const { t } = useTranslation();
    const inputRef = useRef<HTMLInputElement>(null);
    const nameSearchRef = useRef<IntakeNameSearchHandle>(null);
    const requestId = useRef(0);
    const [query, setQuery] = useState('');
    const [person, setPerson] = useState<ChatPerson>('all');
    const [from, setFrom] = useState('');
    const [to, setTo] = useState('');
    const [sourceFormat, setSourceFormat] = useState('all');
    const [myWordsOnly, setMyWordsOnly] = useState(false);
    const [topic, setTopic] = useState('all');
    const [result, setResult] = useState<ChatSearchResponse | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [openedKey, setOpenedKey] = useState<string | null>(null);
    // Ctrl+Shift+F lands on the name box: the magnifying glass is first of all "where is that
    // folder?" (owner 2026-09-22). The chats query is still one tab away.
    useImperativeHandle(ref, () => ({ focus: () => nameSearchRef.current?.focus() }));

    const info = useQuery({
      queryKey: ['intake-chat-index-info'],
      queryFn: () => TauriAPI.getChatIndexInfo(),
      staleTime: 300_000,
      retry: false,
    });

    const search = async () => {
      const q = query.trim();
      if (!q) return;
      const id = ++requestId.current;
      setLoading(true);
      setError(null);
      try {
        const response = await TauriAPI.searchChatIndex({
          query: q,
          person,
          from: from || undefined,
          to: to || undefined,
          sourceFormat: sourceFormat === 'all' ? undefined : sourceFormat,
          speaker: myWordsOnly ? 'owner' : undefined,
          topic: topic === 'all' ? undefined : topic,
          limit: 50,
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
    };

    const openHit = (hit: ChatSearchHit) => {
      setOpenedKey(hit.dedup_key);
      setSelectedChatEvent({ dedupKey: hit.dedup_key, sourcePath: hit.source_path });
      if (hit.source_path) openSearchHit(hit.source_path);
    };

    const scope = info.data;
    return (
      <section
        className="text-xp-text flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3"
        aria-label={t('intakeChatSearch.title')}
      >
        {/* Names first: clicking the magnifying glass is asking "where is that folder?". */}
        <IntakeNameSearch
          ref={nameSearchRef}
          navigateToPath={navigateToPath}
          activePaneRoot={activePaneRoot}
        />
        <h2 className="border-xp-border border-t pt-3 text-sm font-semibold">
          {t('intakeChatSearch.title')}
        </h2>
        <p role="status" className="text-xs" data-testid="chat-index-scope">
          {info.isError
            ? `${t('intakeChatSearch.indexUnavailable')}: ${String((info.error as Error)?.message ?? '')}`
            : `${t('intakeChatSearch.searchingAll')} (${fmt(scope?.events)})`}
        </p>
        {scope && (
          <p className="text-xp-text-muted text-[11px]">
            {scope.first} → {scope.last} · {t('intakeChatSearch.notFiles')}
          </p>
        )}
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
            maxLength={500}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              // Enter runs the search; the explorer's global key handling must not swallow it.
              if (event.key === 'Enter') {
                event.preventDefault();
                event.stopPropagation();
                void search();
              }
            }}
            aria-label={t('intakeChatSearch.query')}
            placeholder={t('intakeChatSearch.query')}
            className="border-xp-border bg-xp-surface rounded border p-2 text-sm"
          />
          <div className="grid grid-cols-2 gap-2">
            <select
              value={person}
              aria-label={t('intakeChatSearch.person')}
              onChange={(event) => setPerson(event.target.value as ChatPerson)}
              className="border-xp-border bg-xp-surface rounded border p-1 text-xs"
            >
              <option value="all">{t('intakeChatSearch.personAll')}</option>
              <option value="katrina">Katrina</option>
              <option value="daughter">{t('intakeChatSearch.personDaughter')}</option>
            </select>
            <select
              value={sourceFormat}
              aria-label={t('intakeChatSearch.sourceFormat')}
              onChange={(event) => setSourceFormat(event.target.value)}
              className="border-xp-border bg-xp-surface rounded border p-1 text-xs"
            >
              <option value="all">{t('intakeChatSearch.allFormats')}</option>
              {(scope?.formats ?? [])
                .filter((f) => f.format)
                .map((f) => (
                  <option key={f.format} value={f.format ?? ''}>
                    {f.format} ({fmt(f.events)})
                  </option>
                ))}
            </select>
            <label className="text-xp-text-muted flex flex-col text-[11px]">
              {t('intakeChatSearch.from')}
              <input
                type="date"
                value={from}
                onChange={(event) => setFrom(event.target.value)}
                aria-label={t('intakeChatSearch.from')}
                className="border-xp-border bg-xp-surface text-xp-text rounded border p-1 text-xs"
              />
            </label>
            <label className="text-xp-text-muted flex flex-col text-[11px]">
              {t('intakeChatSearch.to')}
              <input
                type="date"
                value={to}
                onChange={(event) => setTo(event.target.value)}
                aria-label={t('intakeChatSearch.to')}
                className="border-xp-border bg-xp-surface text-xp-text rounded border p-1 text-xs"
              />
            </label>
          </div>
          {(scope?.speakers?.length ?? 0) > 0 && (
            <label className="flex items-center gap-2 text-[11px]">
              <input
                type="checkbox"
                checked={myWordsOnly}
                onChange={(event) => setMyWordsOnly(event.target.checked)}
                aria-label={t('intakeChatSearch.myWordsOnly')}
              />
              {t('intakeChatSearch.myWordsOnly')}
            </label>
          )}
          {(scope?.topics?.length ?? 0) > 0 && (
            <select
              value={topic}
              aria-label={t('intakeChatSearch.topic')}
              onChange={(event) => setTopic(event.target.value)}
              className="border-xp-border bg-xp-surface rounded border p-1 text-xs"
            >
              <option value="all">{t('intakeChatSearch.allTopics')}</option>
              {(scope?.topics ?? []).map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </select>
          )}
          <button
            type="submit"
            disabled={loading || !query.trim() || info.isError}
            className="border-xp-border rounded border p-2 text-sm disabled:opacity-50"
          >
            {loading ? t('intakeChatSearch.running') : t('intakeChatSearch.search')}
          </button>
        </form>
        {error && (
          <p role="alert" className="text-xs">
            {t('intakeChatSearch.failed')}: {error}
          </p>
        )}
        {result && (
          <>
            <p className="text-xp-text-muted text-[11px]" data-testid="chat-search-summary">
              {result.hits.length} {t('intakeChatSearch.shown')}
              {typeof result.keyword_matches === 'number'
                ? ` · ${fmt(result.keyword_matches)} ${t('intakeChatSearch.keywordMatches')}`
                : ''}{' '}
              · {result.timings_ms.total} ms
            </p>
            {result.notes.map((note) => (
              <p key={note} className="text-xp-text-muted text-[11px]">
                {note}
              </p>
            ))}
            {result.hits.length === 0 && (
              <p className="text-xs">{t('intakeChatSearch.noResults')}</p>
            )}
            <ul className="space-y-2" data-testid="chat-search-hits">
              {result.hits.map((hit) => (
                <li key={hit.dedup_key}>
                  <button
                    type="button"
                    onClick={() => openHit(hit)}
                    title={hit.source_path ?? t('intakeChatSearch.noSource')}
                    className={`hover:bg-xp-surface-light w-full rounded border p-2 text-left text-xs ${
                      openedKey === hit.dedup_key ? 'border-xp-accent' : 'border-xp-border'
                    }`}
                  >
                    <span className="flex flex-wrap items-center justify-between gap-1">
                      <span className="font-medium">{hit.date ?? '—'}</span>
                      <span className="text-xp-text-muted text-[10px]">{hit.source_format}</span>
                    </span>
                    <span className="mt-0.5 block break-words">
                      {hit.sender ?? '?'} → {hit.recipients ?? hit.participants ?? '?'}
                    </span>
                    <span className="text-xp-text-muted mt-1 block whitespace-pre-wrap break-words">
                      <Snippet text={hit.snippet} />
                    </span>
                    <span className="mt-1 flex items-center justify-between gap-1">
                      <TagChips hit={hit} />
                      <span className="text-xp-text-muted text-[10px]">
                        {hit.matched_by.join(' + ')}
                      </span>
                    </span>
                  </button>
                  {hit.catalog_path && (
                    <button
                      type="button"
                      className="text-xp-text-muted mt-0.5 text-[10px] underline"
                      onClick={() => {
                        const parent = hit.catalog_path!.replace(/\/[^/]*$/, '');
                        navigateToPath(parent);
                      }}
                    >
                      {t('intakeChatSearch.openCatalog')}
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </>
        )}
        <IntakeLiveFolderSearch folder={activePaneRoot} />
      </section>
    );
  },
);
IntakeChatSearchPanel.displayName = 'IntakeChatSearchPanel';
export default IntakeChatSearchPanel;
