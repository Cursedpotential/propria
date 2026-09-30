import React, {
  useState,
  useRef,
  useMemo,
  useImperativeHandle,
  forwardRef,
  useSyncExternalStore,
} from 'react';
import { GripHorizontal } from 'lucide-react';
import { FileEntry } from '@/lib/tauri-api';
// Byline: Claude Code · Sonnet · 2026-09-14 (activePaneRoot passthrough for the rg search method)
import IntakeFilesystemSearchPanel from './IntakeFilesystemSearchPanel';
// Byline: Claude Code · Opus 5 · 2026-09-18 (hosted Intake: index-first chats search + capped live folder search)
import IntakeChatSearchPanel from './IntakeChatSearchPanel';
import { isTauri } from '@/lib/transport';
import SearchResultsPanel, {
  type SearchResultsPanelHandle,
} from '@/components/explorer/SearchResultsPanel';
import { extensionHost } from '@/lib/extension-host';
import { type FileCollection } from '@/lib/collections';
import { useSidebarResize } from '@/hooks/use-sidebar-resize';
import SidebarTabBar from '@/components/explorer/sidebar/SidebarTabBar';
import SidebarQuickAccess from '@/components/explorer/sidebar/SidebarQuickAccess';
import SidebarRecent from '@/components/explorer/sidebar/SidebarRecent';
import SidebarBookmarks from '@/components/explorer/sidebar/SidebarBookmarks';
import SidebarCollections from '@/components/explorer/sidebar/SidebarCollections';
import SidebarDrives from '@/components/explorer/sidebar/SidebarDrives';
import SidebarFileTree from '@/components/explorer/sidebar/SidebarFileTree';

export interface LeftSidebarHandle {
  focusSearch: () => void;
}

interface LeftSidebarProps {
  currentPath: string;
  navigateToPath: (path: string) => void;
  handleFileClick: (file: FileEntry) => void;
  handleFileRightClick?: (file: FileEntry, event: React.MouseEvent) => void;
  handleFileOpen?: (file: FileEntry) => void;
  getFileIcon: (file: FileEntry) => React.ReactNode;
  width?: number;
  searchPanelOpen?: boolean;
  onToggleSearchPanel?: () => void;
  onCreateCollection?: () => void;
  onEditCollection?: (collection: FileCollection) => void;
  // Active collection filter (collection applied as filter on current directory)
  activeCollectionFilter?: FileCollection | null;
  onToggleCollectionFilter?: (collection: FileCollection) => void;
  'data-tour'?: string;
}

const LeftSidebar = forwardRef<LeftSidebarHandle, LeftSidebarProps>(function LeftSidebar(
  {
    currentPath,
    navigateToPath,
    handleFileClick,
    handleFileRightClick,
    handleFileOpen,
    getFileIcon,
    width,
    searchPanelOpen = false,
    onToggleSearchPanel,
    onCreateCollection,
    onEditCollection,
    activeCollectionFilter,
    onToggleCollectionFilter,
    'data-tour': dataTour,
  },
  ref,
) {
  const searchPanelRef = useRef<SearchResultsPanelHandle>(null);

  useImperativeHandle(ref, () => ({
    focusSearch: () => {
      if (!searchPanelOpen && onToggleSearchPanel) {
        onToggleSearchPanel();
        setTimeout(() => searchPanelRef.current?.focus(), 100);
      } else {
        searchPanelRef.current?.focus();
      }
    },
  }));

  // ─── Extension sidebar tabs ────────────────────────────────────────────
  const [activeExtensionTab, setActiveExtensionTab] = useState<string | null>(null);

  const extRefreshKey = useSyncExternalStore(
    extensionHost.subscribe,
    extensionHost.getSnapshotVersion,
  );

  const extensionSidebarTabs = useMemo(() => {
    try {
      return extensionHost.getSidebarTabs();
    } catch {
      return [];
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [extRefreshKey]);

  let activeTabId: string;
  if (activeExtensionTab) {
    activeTabId = activeExtensionTab;
  } else if (searchPanelOpen) {
    activeTabId = '__search__';
  } else {
    activeTabId = '__explorer__';
  }

  const handleTabClick = (tabId: string) => {
    if (tabId === '__explorer__') {
      setActiveExtensionTab(null);
      if (searchPanelOpen && onToggleSearchPanel) onToggleSearchPanel();
    } else if (tabId === '__search__') {
      setActiveExtensionTab(null);
      if (!searchPanelOpen && onToggleSearchPanel) onToggleSearchPanel();
    } else {
      if (searchPanelOpen && onToggleSearchPanel) onToggleSearchPanel();
      setActiveExtensionTab(tabId);
    }
  };

  // Search rides ALONGSIDE the explorer. An extension tab still owns the whole sidebar,
  // because that is a genuinely separate surface rather than a panel sharing live state.
  const showSearch = searchPanelOpen && !activeExtensionTab;

  // ─── Section collapsed state + resize ─────────────────────────────────
  const { sectionCollapsed, sectionHeights, toggleSection, onResizeStart } = useSidebarResize();

  return (
    <nav
      data-tour={dataTour}
      role="navigation"
      aria-label="File explorer sidebar"
      className="bg-xp-surface border-xp-border flex flex-shrink-0 flex-col border-r"
      style={{ width: width ?? 256, minHeight: 0, overflow: 'hidden' }}
    >
      {/* Sidebar tab bar */}
      {onToggleSearchPanel && (
        <SidebarTabBar
          activeTabId={activeTabId}
          onTabClick={handleTabClick}
          extensionTabs={extensionSidebarTabs}
        />
      )}

      {/* Search is a SECTION, not a mode.
          Opening it used to swap out the entire sidebar body, so the file tree disappeared the
          moment the magnifying glass was clicked -- owner 2026-09-26: "There's no file tree, like
          there's regression in the other pages." The design contract rev 3 (2026-09-24) asks for
          stable collapsible categories rather than mode tabs, so search now sits above the
          explorer, is resizable by the same handle every other section uses, and both stay on
          screen together.
          Byline: Claude Code - Opus 5 - 2026-09-27 */}
      {showSearch && (
        <div
          className="border-xp-border flex min-h-0 flex-col border-b"
          role="region"
          aria-label="Search"
          data-sidebar-section="search"
          style={{ height: sectionHeights.search ?? 340 }}
        >
          {import.meta.env.VITE_INTAKE_MODE === '1' && !isTauri() && (
            <IntakeChatSearchPanel
              ref={searchPanelRef}
              navigateToPath={navigateToPath}
              activePaneRoot={currentPath}
            />
          )}
          {import.meta.env.VITE_INTAKE_MODE === '1' && isTauri() && (
            <IntakeFilesystemSearchPanel
              ref={searchPanelRef}
              navigateToPath={navigateToPath}
              activePaneRoot={currentPath}
            />
          )}
          {import.meta.env.VITE_INTAKE_MODE !== '1' && (
            <SearchResultsPanel
              ref={searchPanelRef}
              basePath={currentPath}
              navigateToPath={navigateToPath}
              onFileSelect={handleFileClick}
              onFileOpen={handleFileOpen}
            />
          )}
        </div>
      )}
      {showSearch && (
        <div
          className="hover:bg-xp-blue/30 group flex h-2 flex-shrink-0 cursor-row-resize items-center justify-center transition-colors"
          onMouseDown={(e) => onResizeStart('search', e)}
        >
          <GripHorizontal className="text-xp-text-muted/20 group-hover:text-xp-text-muted/60 h-3 w-4 transition-colors" />
        </div>
      )}

      {/* Extension sidebar tab content */}
      {activeExtensionTab &&
        (() => {
          const renderer = extensionHost.getSidebarTabRenderer(activeExtensionTab);
          if (!renderer) {
            return (
              <div className="text-xp-text-muted flex flex-1 items-center justify-center p-4 text-xs">
                Extension tab not available
              </div>
            );
          }
          return renderer({ currentPath, isActive: true });
        })()}

      {/* Explorer content -- always on screen unless an extension tab has taken the sidebar.
          The file tree in particular must never be swapped out by opening search. */}
      {!activeExtensionTab && (
        <>
          <SidebarQuickAccess
            currentPath={currentPath}
            navigateToPath={navigateToPath}
            collapsed={sectionCollapsed.quickAccess}
            onToggleCollapsed={() => toggleSection('quickAccess')}
            sectionHeight={sectionHeights.quickAccess}
            onResizeStart={onResizeStart}
          />

          <SidebarRecent
            navigateToPath={navigateToPath}
            collapsed={sectionCollapsed.recent}
            onToggleCollapsed={() => toggleSection('recent')}
            sectionHeight={sectionHeights.recent}
            onResizeStart={onResizeStart}
          />

          <SidebarBookmarks
            currentPath={currentPath}
            navigateToPath={navigateToPath}
            handleFileRightClick={handleFileRightClick}
            collapsed={sectionCollapsed.favorites}
            onToggleCollapsed={() => toggleSection('favorites')}
            sectionHeight={sectionHeights.favorites}
            onResizeStart={onResizeStart}
          />

          <SidebarCollections
            currentPath={currentPath}
            navigateToPath={navigateToPath}
            activeCollectionFilter={activeCollectionFilter}
            onToggleCollectionFilter={onToggleCollectionFilter}
            onCreateCollection={onCreateCollection}
            onEditCollection={onEditCollection}
            collapsed={sectionCollapsed.collections}
            onToggleCollapsed={() => toggleSection('collections')}
            sectionHeight={sectionHeights.collections}
            onResizeStart={onResizeStart}
          />

          <SidebarDrives
            navigateToPath={navigateToPath}
            collapsed={sectionCollapsed.drives}
            onToggleCollapsed={() => toggleSection('drives')}
            sectionHeight={sectionHeights.drives}
            onResizeStart={onResizeStart}
          />

          <SidebarFileTree
            currentPath={currentPath}
            navigateToPath={navigateToPath}
            handleFileClick={handleFileClick}
            handleFileRightClick={handleFileRightClick}
            getFileIcon={getFileIcon}
            collapsed={sectionCollapsed.fileTree}
            onToggleCollapsed={() => toggleSection('fileTree')}
          />
        </>
      )}
    </nav>
  );
});

export default LeftSidebar;
