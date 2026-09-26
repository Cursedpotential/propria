
import React, { useState, useCallback } from 'react';
import { FileUp, Settings, BrainCircuit } from 'lucide-react';
import { ReviewDeck } from './components/ReviewDeck.tsx';
import { SettingsModal } from './components/SettingsModal.tsx';
import { ingestFile } from './lib/ingestor.ts';
import { useSettings } from './hooks/useSettings.ts';
import { AnalysisRecord } from './types.ts';
import { db } from './lib/db.ts';

type Notification = {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
};

export default function App() {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<AnalysisRecord | null>(null);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const { settings, setSetting } = useSettings();
  const [isIngesting, setIsIngesting] = useState(false);

  const addNotification = useCallback((message: string, type: 'success' | 'error' | 'info' = 'info') => {
    const id = Date.now();
    setNotifications(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setNotifications(prev => prev.filter(n => n.id !== id));
    }, 5000);
  }, []);

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setIsIngesting(true);
      addNotification(`Ingesting ${file.name}...`, 'info');
      try {
        const result = await ingestFile(file, settings.contextKeywords);
        addNotification(result.message, result.success ? 'success' : 'error');
      } catch (error) {
        console.error('Ingestion failed:', error);
        addNotification(error instanceof Error ? error.message : 'An unknown error occurred during ingestion.', 'error');
      } finally {
        setIsIngesting(false);
        // Reset file input to allow re-uploading the same file
        event.target.value = '';
      }
    }
  };
  
  const handleClearDatabase = async () => {
    if (window.confirm('Are you sure you want to delete all ingested data? This action cannot be undone.')) {
      try {
        await Promise.all([
            db.analysisRecords.clear(),
            db.evidenceChunks.clear(),
            db.evidenceSources.clear(),
        ]);
        setSelectedRecord(null);
        addNotification('Database cleared successfully.', 'success');
      } catch (error) {
        console.error('Failed to clear database:', error);
        addNotification('Failed to clear database.', 'error');
      }
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 text-gray-300 flex flex-col">
      <header className="bg-gray-800/50 backdrop-blur-sm border-b border-gray-700 shadow-lg sticky top-0 z-20">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-3">
              <BrainCircuit className="h-8 w-8 text-cyan-400" />
              <h1 className="text-xl font-bold text-gray-100">Forensic Data Refinery</h1>
            </div>
            <div className="flex items-center space-x-2">
              <label htmlFor="file-upload" className={`flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-cyan-600 hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 focus:ring-offset-gray-900 cursor-pointer ${isIngesting ? 'opacity-50 cursor-not-allowed' : ''}`}>
                <FileUp className="h-5 w-5 mr-2" />
                {isIngesting ? 'Ingesting...' : 'Ingest Data'}
              </label>
              <input id="file-upload" name="file-upload" type="file" className="sr-only" onChange={handleFileChange} accept=".md,.txt,.zip" disabled={isIngesting} />
              <button onClick={() => setIsSettingsOpen(true)} className="p-2 rounded-md text-gray-400 hover:text-white hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-white">
                <Settings className="h-6 w-6" />
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="flex-grow container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <ReviewDeck selectedRecord={selectedRecord} setSelectedRecord={setSelectedRecord} addNotification={addNotification} />
      </main>

      <SettingsModal 
        isOpen={isSettingsOpen} 
        onClose={() => setIsSettingsOpen(false)} 
        settings={settings}
        setSetting={setSetting}
        addNotification={addNotification}
        onClearDatabase={handleClearDatabase}
      />

      {/* Notifications */}
      <div className="fixed bottom-4 right-4 w-80 space-y-3 z-50">
        {notifications.map(n => (
          <div key={n.id} className={`px-4 py-3 rounded-md shadow-lg text-sm font-medium flex items-center ${n.type === 'success' ? 'bg-green-600/90 text-white' : n.type === 'error' ? 'bg-red-600/90 text-white' : 'bg-blue-600/90 text-white'}`}>
            {n.message}
          </div>
        ))}
      </div>
    </div>
  );
}
