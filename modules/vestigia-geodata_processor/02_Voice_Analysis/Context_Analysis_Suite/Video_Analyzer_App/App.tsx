
import React, { useState } from 'react';
import { Header } from './components/Header';
import { VideoProcessor } from './components/VideoProcessor';
import { DataSync } from './components/DataSync';
import { GraphExport } from './components/GraphExport';
import { CaseDataViewer } from './components/CaseDataViewer';
import { ReviewAndVerify } from './components/ReviewAndVerify';
import { Settings } from './components/Settings';
import { StagedVideo, AppSettings } from './types';
import { IngestionIcon } from './components/icons/IngestionIcon';
import { DatabaseIcon } from './components/icons/DatabaseIcon';
import { GraphIcon } from './components/icons/GraphIcon';
import { CaseDataIcon } from './components/icons/CaseDataIcon';
import { ReviewIcon } from './components/icons/ReviewIcon';
import { SettingsIcon } from './components/icons/SettingsIcon';
import { useLocalStorage } from './hooks/useLocalStorage';

type Tab = 'workbench' | 'review' | 'viewer' | 'sync' | 'graph' | 'settings';

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('workbench');
  const [stagedVideos, setStagedVideos] = useState<StagedVideo[]>([]);
  const [speakerMap, setSpeakerMap] = useState<Record<string, string>>({});
  const [settings, setSettings] = useLocalStorage<AppSettings>('appSettings', {
    apiKeys: {
      google: '',
      openai: '',
    },
    modelPreferences: {
      basic: 'gemini-2.5-flash',
      advanced: 'gemini-2.5-flash',
    }
  });

  const TabButton: React.FC<{ tabName: Tab; label: string; icon: React.ReactNode }> = ({ tabName, label, icon }) => (
    <button
      onClick={() => setActiveTab(tabName)}
      className={`flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-t-md border-b-2 transition-colors ${
        activeTab === tabName
          ? 'border-cyan-400 text-white'
          : 'border-transparent text-slate-400 hover:border-slate-400 hover:text-slate-100'
      }`}
    >
      {icon}
      {label}
    </button>
  );

  return (
    <div className="min-h-screen bg-slate-900 font-sans flex flex-col">
      <Header />
      <main className="container mx-auto px-4 py-8 flex-grow flex flex-col">
        <div className="mb-4 border-b border-slate-700">
          <div className="flex items-center gap-2 flex-wrap">
            <TabButton tabName="workbench" label="Ingestion Workbench" icon={<IngestionIcon className="w-5 h-5" />} />
            <TabButton tabName="review" label="Review & Verify" icon={<ReviewIcon className="w-5 h-5" />} />
            <TabButton tabName="viewer" label="Case Data Viewer" icon={<CaseDataIcon className="w-5 h-5" />} />
            <TabButton tabName="sync" label="Supabase Sync" icon={<DatabaseIcon className="w-5 h-5" />} />
            <TabButton tabName="graph" label="Neo4j Export" icon={<GraphIcon className="w-5 h-5" />} />
            <TabButton tabName="settings" label="Settings" icon={<SettingsIcon className="w-5 h-5" />} />
          </div>
        </div>

        <div className="flex-grow h-full bg-slate-800 rounded-xl shadow-2xl p-4 sm:p-6 lg:p-8 border border-slate-700">
          {activeTab === 'workbench' && (
            <VideoProcessor 
              stagedVideos={stagedVideos}
              setStagedVideos={setStagedVideos}
              settings={settings}
            />
          )}
          {activeTab === 'review' && (
            <ReviewAndVerify
              stagedVideos={stagedVideos}
              setStagedVideos={setStagedVideos}
              speakerMap={speakerMap}
              setSpeakerMap={setSpeakerMap}
              settings={settings}
            />
          )}
          {activeTab === 'viewer' && <CaseDataViewer stagedVideos={stagedVideos} speakerMap={speakerMap} />}
          {activeTab === 'sync' && <DataSync stagedVideos={stagedVideos} speakerMap={speakerMap} />}
          {activeTab === 'graph' && <GraphExport stagedVideos={stagedVideos} speakerMap={speakerMap} />}
          {activeTab === 'settings' && <Settings settings={settings} setSettings={setSettings} />}
        </div>
      </main>
      <footer className="text-center py-4 text-slate-500 text-sm">
        <p>Powered by React, Tailwind CSS, and Google Gemini</p>
      </footer>
    </div>
  );
}

export default App;
