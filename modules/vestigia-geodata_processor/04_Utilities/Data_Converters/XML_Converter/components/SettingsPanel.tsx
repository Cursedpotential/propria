
import React, { useState } from 'react';
import { Base64Option, ConversionConfig, DeduplicationStrategy, SplitMethod, TargetDb } from '../types';
import { SupabaseService } from '../services/supabaseService';
import { GeminiService } from '../services/geminiService';

interface Props {
  config: ConversionConfig;
  setConfig: React.Dispatch<React.SetStateAction<ConversionConfig>>;
  disabled: boolean;
}

export const SettingsPanel: React.FC<Props> = ({ config, setConfig, disabled }) => {
  const [testStatus, setTestStatus] = useState<{ success?: boolean; message?: string } | null>(null);
  const [deployStatus, setDeployStatus] = useState<{ success?: boolean; message?: string } | null>(null);
  const [isTesting, setIsTesting] = useState(false);
  const [isDeploying, setIsDeploying] = useState(false);
  const [showSecrets, setShowSecrets] = useState(false);
  const [activeTab, setActiveTab] = useState<'general' | 'supabase' | 'metadata'>('general');

  // AI SQL Helper
  const [aiSuggestion, setAiSuggestion] = useState<string | null>(null);

  const testConnection = async () => {
    if (!config.supabaseUrl || !config.supabaseKey) {
        setTestStatus({ success: false, message: 'URL and Key are required.' });
        return;
    }
    setIsTesting(true);
    setTestStatus(null);
    try {
        const s = new SupabaseService(config.supabaseUrl, config.supabaseKey, config.tableName);
        const result = await s.testConnection();
        setTestStatus(result);
    } catch (e) {
        setTestStatus({ success: false, message: 'Unknown error occurred.' });
    } finally {
        setIsTesting(false);
    }
  };

  const deploySchema = async () => {
      if (!config.supabaseAccessToken || !config.supabaseUrl) {
          setDeployStatus({ success: false, message: 'Personal Access Token and URL required for auto-deployment.' });
          return;
      }
      
      setIsDeploying(true);
      setDeployStatus(null);
      try {
          const s = new SupabaseService(config.supabaseUrl, config.supabaseKey, config.tableName, config.entityTable, config.attachmentTable, config.callsTable);
          const result = await s.deploySchema(config.supabaseAccessToken, config.columns);
          setDeployStatus(result);
          if (result.success) testConnection();
      } catch (e) {
          setDeployStatus({ success: false, message: 'Deployment failed.' });
      } finally {
          setIsDeploying(false);
      }
  };

  const suggestSql = async () => {
      // Mocked AI suggestion for SQL improvements
      setAiSuggestion("Analyzing schema... Suggesting: Add partition by date for 'messages' table to improve query performance on large datasets.");
  };

  const exportConfig = () => {
      const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `etl_config_${Date.now()}.json`;
      a.click();
      URL.revokeObjectURL(url);
  };

  const importConfig = async () => {
      try {
          const [fileHandle] = await window.showOpenFilePicker({
              types: [{ description: 'JSON Config', accept: { 'application/json': ['.json'] } }],
              multiple: false
          });
          const file = await fileHandle.getFile();
          const text = await file.text();
          const json = JSON.parse(text);
          // Validate minimally
          if (json.columns && Array.isArray(json.columns)) {
              setConfig({ ...config, ...json });
              alert("Configuration loaded successfully.");
          } else {
              alert("Invalid configuration file.");
          }
      } catch (e) {
          console.error(e); // User cancelled or error
      }
  };

  return (
    <div className="bg-slate-900 rounded-xl border border-slate-800 shadow-2xl overflow-hidden flex flex-col w-full h-full max-h-[calc(100vh-10rem)]">
      
      {/* Drawer Header */}
      <div className="bg-slate-950 px-6 py-4 border-b border-slate-800 flex justify-between items-center">
         <div className="flex items-center gap-2">
            <div className="bg-slate-800 p-1.5 rounded-md">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.532 1.532 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.532 1.532 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clipRule="evenodd" />
                </svg>
            </div>
            <h3 className="font-semibold text-white">Project Settings</h3>
         </div>
         <div className="flex gap-2">
             <button 
                onClick={importConfig}
                className="px-3 py-1 bg-slate-800 border border-slate-700 hover:bg-slate-700 text-xs text-zinc-300 rounded transition-colors"
                title="Load Schema & Settings"
             >
                Import JSON
             </button>
             <button 
                onClick={exportConfig}
                className="px-3 py-1 bg-slate-800 border border-slate-700 hover:bg-slate-700 text-xs text-zinc-300 rounded transition-colors"
                title="Save Schema & Settings"
             >
                Export JSON
             </button>
         </div>
         <div className="flex bg-slate-900 rounded-lg p-1 border border-slate-800 ml-4">
             <button 
                onClick={() => setActiveTab('general')}
                className={`px-3 py-1 rounded-md text-xs font-medium transition-all ${activeTab === 'general' ? 'bg-slate-800 text-white shadow-sm' : 'text-slate-500 hover:text-slate-300'}`}
             >
                General
             </button>
             <button 
                onClick={() => setActiveTab('supabase')}
                className={`px-3 py-1 rounded-md text-xs font-medium transition-all ${activeTab === 'supabase' ? 'bg-indigo-900/30 text-indigo-300 shadow-sm border border-indigo-500/20' : 'text-slate-500 hover:text-slate-300'}`}
             >
                Integrations
             </button>
             <button 
                onClick={() => setActiveTab('metadata')}
                className={`px-3 py-1 rounded-md text-xs font-medium transition-all ${activeTab === 'metadata' ? 'bg-teal-900/30 text-teal-300 shadow-sm border border-teal-500/20' : 'text-slate-500 hover:text-slate-300'}`}
             >
                Metadata
             </button>
         </div>
      </div>

      <div className="p-6 space-y-6 overflow-y-auto custom-scrollbar">
        
        {/* GENERAL TAB */}
        {activeTab === 'general' && (
            <div className="space-y-6 animate-fade-in">
                
                {/* DUAL OUTPUT CONFIG */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="bg-slate-950 rounded-lg p-4 border border-slate-800">
                         <div className="flex justify-between items-start mb-3">
                            <h4 className="text-xs font-bold text-slate-300 uppercase">Local Export</h4>
                            <div className="relative inline-block w-8 align-middle select-none">
                                <input type="checkbox" checked={config.exportLocalFile} onChange={(e) => setConfig({...config, exportLocalFile: e.target.checked})} className="absolute block w-4 h-4 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-4"/>
                                <label className={`block overflow-hidden h-4 rounded-full cursor-pointer ${config.exportLocalFile ? 'bg-emerald-500' : 'bg-slate-700'}`}></label>
                            </div>
                         </div>
                         <div className="space-y-3 opacity-90">
                            {/* SQL DIALECT */}
                            <div className="space-y-1">
                                <label className="text-xs text-slate-500 block">CSV/SQL Format</label>
                                <select
                                    disabled={!config.exportLocalFile}
                                    value={config.localExportFormat}
                                    onChange={(e) => setConfig({ ...config, localExportFormat: e.target.value as TargetDb })}
                                    className="w-full bg-slate-900 border border-slate-700 rounded-md py-1.5 px-3 text-white text-xs"
                                >
                                    <option value={TargetDb.POSTGRES}>Postgres Compatible</option>
                                    <option value={TargetDb.SQLITE}>SQLite Compatible</option>
                                </select>
                            </div>
                            
                            {/* SQLITE DB EXPORT TOGGLE */}
                            <label className={`flex items-center justify-between cursor-pointer group p-2 rounded border ${config.exportSqliteFile ? 'bg-indigo-900/20 border-indigo-500/30' : 'bg-slate-900 border-slate-800'}`}>
                                <div className="flex flex-col">
                                    <span className="text-sm text-slate-300 group-hover:text-white transition-colors font-medium">Export .sqlite File</span>
                                    <span className="text-[10px] text-slate-500">Standalone DB file (in-browser)</span>
                                </div>
                                <div className="relative inline-block w-8 mr-1 align-middle select-none transition duration-200 ease-in">
                                    <input type="checkbox" checked={config.exportSqliteFile} onChange={(e) => setConfig({...config, exportSqliteFile: e.target.checked})} className="toggle-checkbox absolute block w-4 h-4 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-4"/>
                                    <label className={`toggle-label block overflow-hidden h-4 rounded-full cursor-pointer ${config.exportSqliteFile ? 'bg-indigo-500' : 'bg-slate-700'}`}></label>
                                </div>
                            </label>

                             {/* NEO4J EXPORT TOGGLE */}
                            <label className={`flex items-center justify-between cursor-pointer group p-2 rounded border ${config.exportNeo4j ? 'bg-orange-900/20 border-orange-500/30' : 'bg-slate-900 border-slate-800'}`}>
                                <div className="flex flex-col">
                                    <span className="text-sm text-slate-300 group-hover:text-white transition-colors font-medium">Export Neo4j Graph</span>
                                    <span className="text-[10px] text-slate-500">Generates nodes.csv & relationships.csv</span>
                                </div>
                                <div className="relative inline-block w-8 mr-1 align-middle select-none transition duration-200 ease-in">
                                    <input type="checkbox" checked={config.exportNeo4j} onChange={(e) => setConfig({...config, exportNeo4j: e.target.checked})} className="toggle-checkbox absolute block w-4 h-4 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-4"/>
                                    <label className={`toggle-label block overflow-hidden h-4 rounded-full cursor-pointer ${config.exportNeo4j ? 'bg-orange-500' : 'bg-slate-700'}`}></label>
                                </div>
                            </label>

                         </div>
                    </div>

                    <div className="bg-slate-950 rounded-lg p-4 border border-slate-800">
                         <div className="flex justify-between items-start mb-3">
                            <h4 className="text-xs font-bold text-indigo-300 uppercase">Cloud Stream</h4>
                             <div className="relative inline-block w-8 align-middle select-none">
                                <input type="checkbox" checked={config.streamToSupabase} onChange={(e) => setConfig({...config, streamToSupabase: e.target.checked})} className="absolute block w-4 h-4 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-4"/>
                                <label className={`block overflow-hidden h-4 rounded-full cursor-pointer ${config.streamToSupabase ? 'bg-indigo-500' : 'bg-slate-700'}`}></label>
                            </div>
                         </div>
                         <p className="text-[10px] text-slate-500 leading-snug">
                             Process and push directly to your Supabase project in real-time batches. Requires connection keys.
                         </p>
                    </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-400 mb-1">Attachments Handling</label>
                        <select
                            disabled={disabled}
                            value={config.base64Option}
                            onChange={(e) => setConfig({ ...config, base64Option: e.target.value as Base64Option })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-md py-2 px-3 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent text-sm"
                            >
                            <option value={Base64Option.SKIP}>Skip (Minimize Size)</option>
                            <option value={Base64Option.INLINE}>Inline (Base64 in DB/CSV)</option>
                            <option value={Base64Option.SEPARATE_FILE}>Separate CSV (Parts)</option>
                            <option value={Base64Option.EXPORT_IMAGES}>Export to Folder (Local Only)</option>
                            <option value={Base64Option.UPLOAD_TO_STORAGE}>Upload to Supabase Storage</option>
                        </select>
                    </div>
                    <div>
                         <label className="block text-sm font-medium text-slate-400 mb-1">Deduplication Logic</label>
                         <select
                            disabled={disabled}
                            value={config.deduplicationStrategy || DeduplicationStrategy.STRICT_HASH}
                            onChange={(e) => setConfig({ ...config, deduplicationStrategy: e.target.value as DeduplicationStrategy })}
                            className="w-full bg-slate-950 border border-slate-800 rounded-md py-2 px-3 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent text-sm"
                         >
                            <option value={DeduplicationStrategy.STRICT_HASH}>Strict (Hash of Date+Body+Sender)</option>
                            <option value={DeduplicationStrategy.FUZZY_BODY}>Fuzzy (Body Text Only)</option>
                            <option value={DeduplicationStrategy.ID_ONLY}>ID Based (Fastest)</option>
                         </select>
                    </div>
                </div>

                <div className="bg-slate-950 rounded-lg p-4 border border-slate-800">
                    <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Processing Options</label>
                    <div className="space-y-3">
                        <label className="flex items-center justify-between cursor-pointer group">
                            <span className="text-sm text-slate-300 group-hover:text-white transition-colors">Integrity Hashing (SHA-256)</span>
                            <div className="relative inline-block w-10 mr-2 align-middle select-none transition duration-200 ease-in">
                                <input type="checkbox" checked={config.calculateHash} onChange={(e) => setConfig({...config, calculateHash: e.target.checked})} className="toggle-checkbox absolute block w-5 h-5 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-5"/>
                                <label className={`toggle-label block overflow-hidden h-5 rounded-full cursor-pointer ${config.calculateHash ? 'bg-indigo-600' : 'bg-slate-700'}`}></label>
                            </div>
                        </label>
                         <label className="flex items-center justify-between cursor-pointer group">
                            <span className="text-sm text-slate-300 group-hover:text-white transition-colors">Generate UUID v7 (Time-Sortable)</span>
                            <div className="relative inline-block w-10 mr-2 align-middle select-none transition duration-200 ease-in">
                                <input type="checkbox" checked={config.generateUuid} onChange={(e) => setConfig({...config, generateUuid: e.target.checked})} className="toggle-checkbox absolute block w-5 h-5 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-5"/>
                                <label className={`toggle-label block overflow-hidden h-5 rounded-full cursor-pointer ${config.generateUuid ? 'bg-indigo-600' : 'bg-slate-700'}`}></label>
                            </div>
                        </label>
                        <label className="flex items-center justify-between cursor-pointer group">
                            <span className="text-sm text-slate-300 group-hover:text-white transition-colors">Human Readable Time (EST)</span>
                            <div className="relative inline-block w-10 mr-2 align-middle select-none transition duration-200 ease-in">
                                <input type="checkbox" checked={config.humanReadableTime} onChange={(e) => setConfig({...config, humanReadableTime: e.target.checked})} className="toggle-checkbox absolute block w-5 h-5 rounded-full bg-white border-4 appearance-none cursor-pointer checked:right-0 right-5"/>
                                <label className={`toggle-label block overflow-hidden h-5 rounded-full cursor-pointer ${config.humanReadableTime ? 'bg-indigo-600' : 'bg-slate-700'}`}></label>
                            </div>
                        </label>
                    </div>
                </div>
            </div>
        )}

        {/* SUPABASE & INTEGRATIONS TAB */}
        {activeTab === 'supabase' && (
            <div className="space-y-6 animate-fade-in">
                 
                 {/* Google Drive Auth */}
                 <div className="bg-slate-950 border border-slate-800 rounded-lg overflow-hidden">
                    <div className="bg-slate-900/50 px-4 py-2 border-b border-slate-800 flex justify-between items-center">
                         <span className="text-xs font-bold text-blue-400 uppercase">Google Drive</span>
                    </div>
                    <div className="p-4 space-y-2">
                        <label className="text-xs text-slate-500">Client ID (OAuth 2.0)</label>
                        <input
                            type="text"
                            value={config.googleDriveClientId || ''}
                            onChange={(e) => setConfig({...config, googleDriveClientId: e.target.value})}
                            placeholder="123...apps.googleusercontent.com"
                            className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-blue-500 outline-none transition-colors font-mono"
                        />
                        <p className="text-[10px] text-zinc-500">Required for accessing private files in Google Drive. Must be authorized origin.</p>
                    </div>
                 </div>

                 {/* Supabase Connection */}
                 <div className="bg-slate-950 border border-slate-800 rounded-lg overflow-hidden">
                    <div className="bg-slate-900/50 px-4 py-2 border-b border-slate-800 flex justify-between items-center">
                         <span className="text-xs font-bold text-indigo-400 uppercase">Supabase Connection</span>
                         <button 
                            onClick={() => setShowSecrets(!showSecrets)} 
                            className="text-xs text-slate-500 hover:text-white transition-colors"
                         >
                            {showSecrets ? 'Hide Secrets' : 'Show Secrets'}
                         </button>
                    </div>
                    <div className="p-4 space-y-4">
                        <div className="space-y-1">
                            <label className="text-xs text-slate-500">Project URL</label>
                            <input
                                type="text"
                                value={config.supabaseUrl || ''}
                                onChange={(e) => setConfig({...config, supabaseUrl: e.target.value})}
                                placeholder="https://xyz.supabase.co"
                                className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-indigo-500 outline-none transition-colors font-mono"
                            />
                        </div>
                        <div className="space-y-1">
                            <label className="text-xs text-slate-500">API Key (Anon/Service)</label>
                             <div className="relative">
                                <input
                                    type={showSecrets ? "text" : "password"}
                                    value={config.supabaseKey || ''}
                                    onChange={(e) => setConfig({...config, supabaseKey: e.target.value})}
                                    placeholder="sbp_..."
                                    className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-indigo-500 outline-none transition-colors font-mono"
                                />
                                <div className="absolute right-3 top-2.5 w-2 h-2 rounded-full bg-emerald-500 shadow-lg shadow-emerald-500/50"></div>
                             </div>
                        </div>
                         <div className="space-y-2 pt-2">
                             <div className="flex gap-2">
                                 <input
                                     type="text"
                                     placeholder="Messages Table"
                                     value={config.tableName || 'messages'}
                                     onChange={(e) => setConfig({...config, tableName: e.target.value})}
                                     className="flex-1 bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white font-mono"
                                 />
                                 <button
                                     onClick={testConnection}
                                     disabled={isTesting}
                                     className="px-4 bg-slate-800 hover:bg-slate-700 text-white text-xs font-medium uppercase tracking-wide rounded transition-colors disabled:opacity-50"
                                 >
                                     {isTesting ? 'Ping...' : 'Test'}
                                 </button>
                             </div>
                             <div className="grid grid-cols-3 gap-2">
                                 <input
                                     type="text"
                                     placeholder="Entities"
                                     value={config.entityTable || 'entities'}
                                     onChange={(e) => setConfig({...config, entityTable: e.target.value})}
                                     className="bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white font-mono"
                                 />
                                 <input
                                     type="text"
                                     placeholder="Attachments"
                                     value={config.attachmentTable || 'attachments'}
                                     onChange={(e) => setConfig({...config, attachmentTable: e.target.value})}
                                     className="bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white font-mono"
                                 />
                                 <input
                                     type="text"
                                     placeholder="Calls"
                                     value={config.callsTable || 'calls'}
                                     onChange={(e) => setConfig({...config, callsTable: e.target.value})}
                                     className="bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white font-mono"
                                     disabled={!config.separateCallsTable}
                                 />
                             </div>
                             <div className="flex items-center gap-2 pt-1">
                                 <input
                                     type="checkbox"
                                     id="separateCallsTable"
                                     checked={config.separateCallsTable || false}
                                     onChange={(e) => setConfig({...config, separateCallsTable: e.target.checked})}
                                     className="w-4 h-4 rounded bg-slate-900 border-slate-700 text-indigo-600 focus:ring-indigo-500 focus:ring-offset-0"
                                 />
                                 <label htmlFor="separateCallsTable" className="text-xs text-slate-400">
                                     Export call logs to separate table (instead of mixing with messages)
                                 </label>
                             </div>
                         </div>
                    </div>
                     {testStatus && (
                        <div className={`px-4 py-2 text-xs border-t ${testStatus.success ? 'bg-emerald-900/10 border-emerald-900/30 text-emerald-400' : 'bg-red-900/10 border-red-900/30 text-red-400'}`}>
                            {testStatus.success ? '● System Operational' : `● ${testStatus.message}`}
                        </div>
                    )}
                 </div>

                 {/* Storage Bucket Config */}
                 {config.base64Option === Base64Option.UPLOAD_TO_STORAGE && (
                     <div className="bg-slate-950 border border-slate-800 rounded-lg p-4 animate-fade-in">
                         <div className="flex items-center gap-2 mb-3">
                            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-orange-400" viewBox="0 0 20 20" fill="currentColor">
                                <path fillRule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clipRule="evenodd" />
                            </svg>
                            <span className="text-xs font-bold text-slate-300 uppercase">Storage Bucket</span>
                         </div>
                         <input
                             type="text"
                             value={config.storageBucket || 'attachments'}
                             onChange={(e) => setConfig({...config, storageBucket: e.target.value})}
                             placeholder="Bucket Name (e.g. attachments)"
                             className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white font-mono mb-2"
                         />
                     </div>
                 )}

                 {/* Platform Automation (Secrets Manager style) */}
                 <div className="bg-gradient-to-br from-indigo-950/30 to-slate-950 border border-indigo-500/20 rounded-lg p-4">
                     <div className="flex justify-between items-center mb-4">
                        <div className="flex items-center gap-2">
                             <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
                                <path fillRule="evenodd" d="M2 5a2 2 0 012-2h12a2 2 0 012 2v2a2 2 0 01-2 2H4a2 2 0 01-2-2V5zm14 1a1 1 0 11-2 0 1 1 0 012 0zM2 13a2 2 0 012-2h12a2 2 0 012 2v2a2 2 0 01-2 2H4a2 2 0 01-2-2v-2zm14 1a1 1 0 11-2 0 1 1 0 012 0z" clipRule="evenodd" />
                            </svg>
                            <span className="text-xs font-bold text-indigo-100 uppercase">Schema Automation</span>
                        </div>
                     </div>

                     <div className="space-y-3">
                         <div className="relative group">
                            <label className="text-[10px] text-indigo-300 uppercase font-bold mb-1 block">
                                Personal Access Token (for Schema Deployment)
                            </label>
                            <input
                                type={showSecrets ? "text" : "password"}
                                placeholder="sbp_..."
                                value={config.supabaseAccessToken || ''}
                                onChange={(e) => setConfig({...config, supabaseAccessToken: e.target.value})}
                                className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-indigo-500 outline-none font-mono group-hover:border-slate-600 transition-colors"
                            />
                            <p className="text-[10px] text-slate-500 mt-1">
                                Requires a <strong>Personal Access Token</strong> from your Supabase Account Settings (Security), NOT the Project API Key.
                            </p>
                         </div>
                         
                         <div className="flex gap-2">
                            <button
                                onClick={suggestSql}
                                className="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded border border-slate-700 transition-colors flex items-center gap-1"
                            >
                                <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
                                </svg>
                                AI SQL Check
                            </button>
                             <button 
                                onClick={deploySchema}
                                disabled={isDeploying || !config.supabaseAccessToken}
                                className="flex-1 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-bold uppercase rounded shadow-lg shadow-indigo-900/20 transition-all disabled:opacity-50 disabled:shadow-none"
                            >
                                {isDeploying ? 'Deploying...' : 'Deploy Schema'}
                            </button>
                         </div>
                         
                         {aiSuggestion && (
                             <div className="mt-2 p-2 bg-indigo-900/20 border border-indigo-500/20 rounded text-xs text-indigo-200 italic">
                                 ✨ {aiSuggestion}
                             </div>
                         )}

                         {deployStatus && (
                            <div className={`text-xs mt-2 ${deployStatus.success ? 'text-emerald-400' : 'text-red-400'}`}>
                                {deployStatus.message}
                            </div>
                        )}
                     </div>
                 </div>
            </div>
        )}

        {/* METADATA TAB */}
        {activeTab === 'metadata' && (
            <div className="space-y-6 animate-fade-in">
                <div className="bg-slate-950 border border-slate-800 rounded-lg p-4">
                    <h4 className="text-xs font-bold text-teal-400 uppercase tracking-wider mb-4">Project Metadata</h4>
                    <div className="space-y-4">
                        <div>
                            <label className="text-xs text-slate-500 block mb-1">Case ID / Reference</label>
                            <input 
                                type="text"
                                value={config.projectMetadata?.caseId || ''}
                                onChange={(e) => setConfig({...config, projectMetadata: {...config.projectMetadata, caseId: e.target.value}})}
                                className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-teal-500 outline-none"
                                placeholder="e.g. CASE-2024-001"
                            />
                        </div>
                        <div>
                            <label className="text-xs text-slate-500 block mb-1">Investigator / Examiner</label>
                            <input 
                                type="text"
                                value={config.projectMetadata?.investigator || ''}
                                onChange={(e) => setConfig({...config, projectMetadata: {...config.projectMetadata, investigator: e.target.value}})}
                                className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-teal-500 outline-none"
                                placeholder="Name or Badge #"
                            />
                        </div>
                        <div>
                            <label className="text-xs text-slate-500 block mb-1">Project Notes</label>
                            <textarea 
                                value={config.projectMetadata?.description || ''}
                                onChange={(e) => setConfig({...config, projectMetadata: {...config.projectMetadata, description: e.target.value}})}
                                className="w-full bg-slate-900 border border-slate-700 rounded px-3 py-2 text-sm text-white focus:border-teal-500 outline-none h-24 resize-none"
                                placeholder="Description of the data source, chain of custody notes, etc."
                            />
                        </div>
                    </div>
                </div>
            </div>
        )}

      </div>
    </div>
  );
};
