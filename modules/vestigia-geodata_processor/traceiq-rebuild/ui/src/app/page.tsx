"use client";
import { WorkspaceProvider, useWorkspace } from '@/lib/WorkspaceContext';
import { AppTopBar } from '@/components/shell/AppTopBar';
import { QueryPanel } from '@/components/shell/QueryPanel';
import { MapView } from '@/components/map/MapView';
import { ResultsTable } from '@/components/table/ResultsTable';
import { ChatPane } from '@/components/chat/ChatPane';

function ExplorePage() {
  // WorkspaceProvider owns all fetching (bounds + window). This page used to fetch as
  // well, which double-loaded the corpus on every open.
  const { tab } = useWorkspace();

  if (tab !== 'explore') {
    return (
      <div className="flex flex-col h-full">
        <AppTopBar />
        <div className="flex-1 flex items-center justify-center text-muted">
          <div className="text-center">
            <h2 className="text-lg font-medium text-ink mb-1">{tab.charAt(0).toUpperCase() + tab.slice(1)}</h2>
            <p className="text-sm">Placeholder page for {tab} module.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <AppTopBar />
      <div className="flex-1 flex overflow-hidden">
        <div className="flex h-full">
          <div className="flex flex-col h-full">
            <div className="flex-1 overflow-hidden">
              <QueryPanel />
            </div>
            <div className="h-[260px] shrink-0">
              <ChatPane />
            </div>
          </div>
        </div>
        <MapView />
        <ResultsTable />
      </div>
    </div>
  );
}

export default function Home() {
  return (
    <WorkspaceProvider>
      <ExplorePage />
    </WorkspaceProvider>
  );
}
