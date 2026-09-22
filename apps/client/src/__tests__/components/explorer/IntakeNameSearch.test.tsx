// Byline: Claude Code · Opus 5 · 2026-09-22
// The magnifying glass finds a folder name anywhere (owner 2026-09-22 18:54 EDT), with the scope
// dropdown remembered between searches (18:55 EDT) and both paths on every row (18:58 EDT).
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import IntakeNameSearch, { readPrefs } from '@/components/explorer/IntakeNameSearch';
import type { NameSearchHit, NameSearchResult } from '@/lib/tauri-api';

const searchNames = vi.fn();
const openSearchHit = vi.fn();

// jsdom is started without web storage here, which is also why the component guards every access.
const store = new Map<string, string>();
const localStorageStub = {
  getItem: (k: string) => store.get(k) ?? null,
  setItem: (k: string, v: string) => void store.set(k, v),
  removeItem: (k: string) => void store.delete(k),
  clear: () => store.clear(),
  key: (i: number) => Array.from(store.keys())[i] ?? null,
  get length() {
    return store.size;
  },
};
Object.defineProperty(window, 'localStorage', { value: localStorageStub, configurable: true });

vi.mock('@/lib/tauri-api', () => ({
  TauriAPI: { searchNames: (...args: unknown[]) => searchNames(...args) },
  NAME_SEARCH_SCOPES: [
    { id: 'everything', labelKey: 'intakeNameSearch.scopeEverything' },
    { id: 'this_folder', labelKey: 'intakeNameSearch.scopeThisFolder' },
    { id: 'this_folder_and_subfolders', labelKey: 'intakeNameSearch.scopeThisFolderAndSubfolders' },
    { id: 'subfolders_only', labelKey: 'intakeNameSearch.scopeSubfoldersOnly' },
    { id: 'one_level_up', labelKey: 'intakeNameSearch.scopeOneLevelUp' },
    { id: 'catalog_only', labelKey: 'intakeNameSearch.scopeCatalogOnly' },
    { id: 'b2_only', labelKey: 'intakeNameSearch.scopeB2Only' },
  ],
  SCOPES_NEEDING_FOLDER: [
    'this_folder',
    'this_folder_and_subfolders',
    'subfolders_only',
    'one_level_up',
  ],
}));

// The shared mock drops interpolation values (the key's last segment has no placeholders), so this
// one keeps them: the summary line has to carry the real counts and the real time.
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      const base = key.split('.').pop()!;
      return opts ? `${base}:${Object.values(opts).join('|')}` : base;
    },
    i18n: { language: 'en', changeLanguage: vi.fn() },
  }),
  Trans: ({ children }: { children?: React.ReactNode }) => children,
  initReactI18next: { type: '3rdParty', init: vi.fn() },
}));

vi.mock('@/lib/chat-event-selection', () => ({
  openSearchHit: (...args: unknown[]) => openSearchHit(...args),
  setSelectedChatEvent: vi.fn(),
}));

const folderHit: NameSearchHit = {
  kind: 'folder',
  name: 'Takeout',
  path: '/srv/openlist/b2/salem-data/consignatio/vault/v1/Takeout',
  navigate_to: '/srv/openlist/b2/salem-data/consignatio/vault/v1/Takeout',
  select: null,
  now: null,
  was: null,
  source: 'b2',
  matched: 'name',
  member_of: null,
  size: null,
  modified: null,
  flag: null,
};

const fileHit: NameSearchHit = {
  kind: 'file',
  name: 'sms-20190104.xml',
  path: '/srv/openlist/b2/salem-data/consignatio/vault/v1/sms/sms-20190104.xml',
  navigate_to: '/srv/openlist/b2/salem-data/consignatio/vault/v1/sms',
  select: '/srv/openlist/b2/salem-data/consignatio/vault/v1/sms/sms-20190104.xml',
  now: '/srv/openlist/b2/salem-data/consignatio/vault/v1/sms/sms-20190104.xml',
  was: 'catalog://gdrive/salemnet/Court & Legal Project/sms-20190104.xml',
  source: 'catalog',
  matched: 'name',
  member_of: null,
  size: 4096,
  modified: null,
  flag: null,
};

const result = (hits: NameSearchHit[]): NameSearchResult => ({
  query: 'takeout',
  scope: 'everything',
  kinds: 'both',
  sort: 'name',
  folder: null,
  hits,
  shown_folders: hits.filter((h) => h.kind === 'folder').length,
  shown_files: hits.filter((h) => h.kind === 'file').length,
  total_folders: hits.filter((h) => h.kind === 'folder').length,
  total_files: hits.filter((h) => h.kind === 'file').length,
  capped: false,
  scan_cap: 4000,
  elapsed_ms: 812,
  legs: [],
  notes: [],
});

const props = {
  navigateToPath: vi.fn(),
  activePaneRoot: '/srv/openlist/b2/salem-data/consignatio/vault/v1',
};

const type = (value: string) => {
  fireEvent.change(screen.getByTestId('name-search-query'), { target: { value } });
};

describe('IntakeNameSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    searchNames.mockResolvedValue(result([folderHit, fileHit]));
  });

  it('offers every scope the owner asked for, defaulting to Everything', () => {
    render(<IntakeNameSearch {...props} />);
    const select = screen.getByTestId('name-search-scope') as HTMLSelectElement;
    expect(Array.from(select.options).map((o) => o.value)).toEqual([
      'everything',
      'this_folder',
      'this_folder_and_subfolders',
      'subfolders_only',
      'one_level_up',
      'catalog_only',
      'b2_only',
    ]);
    expect(select.value).toBe('everything');
  });

  it('searches everything without sending the open folder', async () => {
    render(<IntakeNameSearch {...props} />);
    type('Takeout');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    await waitFor(() => expect(searchNames).toHaveBeenCalledTimes(1));
    expect(searchNames).toHaveBeenCalledWith({
      query: 'Takeout',
      scope: 'everything',
      kinds: 'both',
      sort: 'name',
    });
  });

  it('sends the open folder only for a folder-relative scope, and remembers the choice', async () => {
    const { unmount } = render(<IntakeNameSearch {...props} />);
    fireEvent.change(screen.getByTestId('name-search-scope'), {
      target: { value: 'subfolders_only' },
    });
    type('Katrina');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    await waitFor(() => expect(searchNames).toHaveBeenCalledTimes(1));
    expect(searchNames).toHaveBeenCalledWith({
      query: 'Katrina',
      scope: 'subfolders_only',
      kinds: 'both',
      sort: 'name',
      path: props.activePaneRoot,
    });

    unmount();
    render(<IntakeNameSearch {...props} />);
    expect((screen.getByTestId('name-search-scope') as HTMLSelectElement).value).toBe(
      'subfolders_only',
    );
  });

  it('will not search on one character', () => {
    render(<IntakeNameSearch {...props} />);
    type('K');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    expect(searchNames).not.toHaveBeenCalled();
  });

  it('counts folders and files and shows how long it took', async () => {
    render(<IntakeNameSearch {...props} />);
    type('Takeout');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    const summary = await screen.findByTestId('name-search-summary');
    // t('…summary', { folders, files, ms }) — the counts and the time, in that order.
    expect(summary).toHaveTextContent('summary:1|1|812');
  });

  it('shows where a file is now and where it used to be', async () => {
    render(<IntakeNameSearch {...props} />);
    type('sms');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    await screen.findByTestId('name-search-hits');
    expect(
      screen.getByText(/catalog:\/\/gdrive\/salemnet\/Court & Legal Project/),
    ).toBeInTheDocument();
  });

  it('opens a folder hit in the active pane', async () => {
    render(<IntakeNameSearch {...props} />);
    type('Takeout');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    await screen.findByTestId('name-search-hits');
    fireEvent.click(screen.getByTitle(folderHit.path));
    expect(props.navigateToPath).toHaveBeenCalledWith(folderHit.navigate_to);
    expect(openSearchHit).not.toHaveBeenCalled();
  });

  it('navigates to a file hit and selects the file', async () => {
    render(<IntakeNameSearch {...props} />);
    type('sms');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    await screen.findByTestId('name-search-hits');
    fireEvent.click(screen.getByTitle(fileHit.path));
    expect(openSearchHit).toHaveBeenCalledWith(fileHit.select);
  });

  it('says so when nothing has that name', async () => {
    searchNames.mockResolvedValue(result([]));
    render(<IntakeNameSearch {...props} />);
    type('zzzz');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    expect(await screen.findByTestId('name-search-empty')).toBeInTheDocument();
  });

  it('reports a failed search instead of showing stale hits', async () => {
    searchNames.mockRejectedValue(new Error('catalog connection refused'));
    render(<IntakeNameSearch {...props} />);
    type('Takeout');
    fireEvent.submit(screen.getByTestId('name-search-query').closest('form')!);
    expect(await screen.findByRole('alert')).toHaveTextContent('catalog connection refused');
  });

  describe('remembered preferences', () => {
    it('falls back to Everything for anything it cannot trust', () => {
      expect(readPrefs(null)).toEqual({ scope: 'everything', kinds: 'both', sort: 'name' });
      expect(readPrefs('not json')).toEqual({ scope: 'everything', kinds: 'both', sort: 'name' });
      expect(readPrefs('{"scope":"made up","kinds":"nonsense","sort":"nope"}')).toEqual({
        scope: 'everything',
        kinds: 'both',
        sort: 'name',
      });
    });

    it('keeps a scope, kind and sort it recognises', () => {
      expect(readPrefs('{"scope":"b2_only","kinds":"folders","sort":"path"}')).toEqual({
        scope: 'b2_only',
        kinds: 'folders',
        sort: 'path',
      });
    });
  });
});
