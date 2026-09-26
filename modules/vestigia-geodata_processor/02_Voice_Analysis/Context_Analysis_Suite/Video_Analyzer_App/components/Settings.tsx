
import React from 'react';
import { AppSettings } from '../types';
import { InfoIcon } from './icons/InfoIcon';

interface SettingsProps {
  settings: AppSettings;
  setSettings: React.Dispatch<React.SetStateAction<AppSettings>>;
}

const googleModelOptions = [
    'gemini-2.5-flash',
    // Add other compatible Google models here as they become available
];

export const Settings: React.FC<SettingsProps> = ({ settings, setSettings }) => {

  const handleApiKeyChange = (provider: keyof AppSettings['apiKeys'], value: string) => {
    setSettings(prev => ({
      ...prev,
      apiKeys: {
        ...prev.apiKeys,
        [provider]: value,
      }
    }));
  };

  const handleModelChange = (task: keyof AppSettings['modelPreferences'], value: string) => {
    setSettings(prev => ({
        ...prev,
        modelPreferences: {
            ...prev.modelPreferences,
            [task]: value,
        }
    }));
  };

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold text-slate-100 mb-6">Settings</h2>
      <div className="space-y-8">
        
        {/* API Key Management */}
        <div className="space-y-4 bg-slate-700/50 p-6 rounded-lg border border-slate-600">
            <h3 className="text-lg font-semibold text-cyan-300 border-b border-slate-600 pb-2">API Keys</h3>
            <div>
                <label htmlFor="googleKey" className="block text-sm font-medium text-slate-200">Google API Key</label>
                <input type="password" id="googleKey" value={settings.apiKeys.google} onChange={e => handleApiKeyChange('google', e.target.value)} placeholder="Enter your Google API Key" className="mt-1 block w-full bg-slate-800 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
                <p className="mt-1 text-xs text-slate-400">Used for both Vertex AI and AI Studio models.</p>
            </div>
            <div>
                <label htmlFor="openAiKey" className="block text-sm font-medium text-slate-200">OpenAI API Key (Placeholder)</label>
                <input type="password" id="openAiKey" value={settings.apiKeys.openai} onChange={e => handleApiKeyChange('openai', e.target.value)} placeholder="Enter your OpenAI API Key" className="mt-1 block w-full bg-slate-800 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
                <p className="mt-1 text-xs text-slate-400">This is a placeholder to demonstrate architecture. A backend (e.g., on Vercel) is required to use this key securely.</p>
            </div>
        </div>

        {/* Model Preferences */}
        <div className="space-y-4 bg-slate-700/50 p-6 rounded-lg border border-slate-600">
            <h3 className="text-lg font-semibold text-cyan-300 border-b border-slate-600 pb-2">Model Preferences</h3>
            <div>
                <label htmlFor="basicModel" className="flex items-center gap-2 text-sm font-medium text-slate-200">
                    Basic Analysis Model
                    <div className="group relative">
                        <InfoIcon className="w-4 h-4 text-slate-400" />
                        <div className="absolute bottom-full mb-2 hidden w-64 rounded-md bg-slate-900 p-2 text-xs text-slate-200 shadow-lg group-hover:block border border-slate-700 z-10">
                            Model used for the initial, "naive" analysis in the Ingestion Workbench. A faster, cheaper model is recommended.
                        </div>
                    </div>
                </label>
                <select id="basicModel" value={settings.modelPreferences.basic} onChange={e => handleModelChange('basic', e.target.value)} className="mt-1 block w-full bg-slate-800 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm">
                    {googleModelOptions.map(model => <option key={model} value={model}>{model}</option>)}
                </select>
            </div>
            <div>
                <label htmlFor="advancedModel" className="flex items-center gap-2 text-sm font-medium text-slate-200">
                    Advanced Analysis Model
                     <div className="group relative">
                        <InfoIcon className="w-4 h-4 text-slate-400" />
                        <div className="absolute bottom-full mb-2 hidden w-64 rounded-md bg-slate-900 p-2 text-xs text-slate-200 shadow-lg group-hover:block border border-slate-700 z-10">
                            Model used for the deep-dive forensic analysis. A more powerful, "smarter" model is recommended.
                        </div>
                    </div>
                </label>
                <select id="advancedModel" value={settings.modelPreferences.advanced} onChange={e => handleModelChange('advanced', e.target.value)} className="mt-1 block w-full bg-slate-800 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm">
                    {googleModelOptions.map(model => <option key={model} value={model}>{model}</option>)}
                    {/* When you add a backend, you could add other models here */}
                    {/* <option value="openai/gpt-4o">OpenAI: GPT-4o (via Backend)</option> */}
                </select>
            </div>
        </div>

      </div>
    </div>
  );
};
