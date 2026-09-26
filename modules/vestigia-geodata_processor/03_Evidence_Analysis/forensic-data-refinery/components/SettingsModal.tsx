
import React, { useRef } from 'react';
import { AppSettings } from '../types.ts';
import { X, Upload, Trash2 } from 'lucide-react';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  settings: AppSettings;
  setSetting: <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => void;
  addNotification: (message: string, type?: 'success' | 'error' | 'info') => void;
  onClearDatabase: () => void;
}

export function SettingsModal({ isOpen, onClose, settings, setSetting, addNotification, onClearDatabase }: SettingsModalProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen) return null;

  const handleContextFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file && file.type === 'application/json') {
      try {
        const text = await file.text();
        const json = JSON.parse(text);
        // Basic validation
        if (typeof json === 'object' && !Array.isArray(json) && json !== null) {
          setSetting('contextKeywords', json);
          addNotification('Context dictionary loaded.', 'success');
        } else {
          throw new Error('Invalid format. Expected a JSON object.');
        }
      } catch (error) {
        console.error('Failed to load context file:', error);
        addNotification(error instanceof Error ? error.message : 'Failed to parse context file.', 'error');
      }
    } else if (file) {
      addNotification('Invalid file type. Please upload a JSON file.', 'error');
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 z-40 flex justify-center items-center" aria-modal="true" role="dialog">
      <div className="bg-gray-800 rounded-lg shadow-xl w-full max-w-lg border border-gray-700 m-4">
        <div className="flex justify-between items-center p-4 border-b border-gray-700">
          <h2 className="text-lg font-semibold text-white">Settings</h2>
          <button onClick={onClose} className="p-1 rounded-md text-gray-400 hover:text-white hover:bg-gray-700">
            <X className="h-6 w-6" />
          </button>
        </div>
        <div className="p-6 space-y-6">
          {/* AI Infrastructure Settings */}
          <div className="space-y-4">
            <h3 className="text-md font-medium text-cyan-400 border-b border-cyan-400/20 pb-2">AI Infrastructure</h3>
            <div>
              <label htmlFor="api-key" className="block text-sm font-medium text-gray-300">Gemini API Key</label>
              <input
                type="password"
                id="api-key"
                value={settings.geminiApiKey}
                onChange={(e) => setSetting('geminiApiKey', e.target.value)}
                className="mt-1 block w-full bg-gray-900 border border-gray-600 rounded-md shadow-sm py-2 px-3 text-gray-200 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                placeholder="Enter your API key (session only, not stored)"
              />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-300">Use Cloudflare Proxy</span>
              <label htmlFor="use-proxy-toggle" className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" id="use-proxy-toggle" className="sr-only peer" checked={settings.useProxy} onChange={(e) => setSetting('useProxy', e.target.checked)} />
                <div className="w-11 h-6 bg-gray-600 rounded-full peer peer-focus:ring-4 peer-focus:ring-cyan-800 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600"></div>
              </label>
            </div>
            <div>
              <label htmlFor="proxy-url" className="block text-sm font-medium text-gray-300">Proxy URL</label>
              <input
                type="url"
                id="proxy-url"
                value={settings.proxyUrl}
                onChange={(e) => setSetting('proxyUrl', e.target.value)}
                className="mt-1 block w-full bg-gray-900 border border-gray-600 rounded-md shadow-sm py-2 px-3 text-gray-200 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                placeholder="https://worker.example.com"
              />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-300">Debug Mode</span>
              <label htmlFor="debug-mode-toggle" className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" id="debug-mode-toggle" className="sr-only peer" checked={settings.debugMode} onChange={(e) => setSetting('debugMode', e.target.checked)} />
                <div className="w-11 h-6 bg-gray-600 rounded-full peer peer-focus:ring-4 peer-focus:ring-cyan-800 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600"></div>
              </label>
            </div>
          </div>

          {/* Context & Data Settings */}
          <div className="space-y-4">
            <h3 className="text-md font-medium text-cyan-400 border-b border-cyan-400/20 pb-2">Context & Data</h3>
            <div>
              <label className="block text-sm font-medium text-gray-300">Context Dictionary</label>
              <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-600 border-dashed rounded-md">
                <div className="space-y-1 text-center">
                  <Upload className="mx-auto h-10 w-10 text-gray-500" />
                  <div className="flex text-sm text-gray-400">
                    <label htmlFor="context-file-upload" className="relative cursor-pointer bg-gray-800 rounded-md font-medium text-cyan-400 hover:text-cyan-300 focus-within:outline-none">
                      <span>Upload a file</span>
                      <input id="context-file-upload" ref={fileInputRef} onChange={handleContextFileChange} type="file" className="sr-only" accept=".json" />
                    </label>
                    <p className="pl-1">or drag and drop</p>
                  </div>
                  <p className="text-xs text-gray-500">dictionary.json up to 1MB</p>
                </div>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-red-400">Danger Zone</label>
              <button onClick={onClearDatabase} className="mt-1 w-full flex items-center justify-center px-4 py-2 border border-red-500/50 text-sm font-medium rounded-md shadow-sm text-red-300 bg-red-900/20 hover:bg-red-900/40 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 focus:ring-offset-gray-800">
                <Trash2 className="h-5 w-5 mr-2" />
                Clear All Local Data
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
