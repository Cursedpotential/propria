import React, { useState, useEffect, useRef } from 'react';
import { Base64Option, ConversionConfig, ParsedMessage, AnalysisResult, FileSystemDirectoryHandle, SplitMethod, TargetDb, StreamSource, DeduplicationStrategy } from './types';
import { XmlStreamProcessor } from './services/xmlStreamService';
import { AIService } from './services/aiService';
import { SettingsPanel } from './components/SettingsPanel';
import { PreviewTable } from './components/PreviewTable';
import { AnalysisPanel } from './components/AnalysisPanel';
import { ColumnManager } from './components/ColumnManager';
import { FileDropZone } from './components/FileDropZone';
import { DataBrowser } from './components/DataBrowser';
import { QueryBuilder } from './components/QueryBuilder';
import { GraphBuilder } from './components/GraphBuilder';
import { ExportPanel } from './components/ExportPanel';

const App: React.FC = () => {
  const [activeView, setActiveView] = useState<'convert' | 'data' | 'query' | 'graph' | 'export' | 'settings'>('convert');
  const [source, setSource] = useState<StreamSource | null>(null);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  
  // Configuration State
  const [config, setConfig] = useState<ConversionConfig>({
    sourceLabel: '',
    base64Option: Base64Option.SKIP,
    itemTag: '', 
    exportLocalFile: true, 
    exportSqliteFile: false,
    exportNeo4j: false, 
    localExportFormat: TargetDb.POSTGRES,
    streamToSupabase: false,
    
    // Auth & Services (Loaded from .env)
    supabaseUrl: import.meta.env.VITE_SUPABASE_URL || '',
    supabaseKey: import.meta.env.VITE_SUPABASE_SERVICE_ROLE_KEY || '',
    supabaseAccessToken: '', // Still requires manual PAT for schema deployment if needed
    googleDriveClientId: import.meta.env.VITE_GOOGLE_DRIVE_CLIENT_ID || '',

    tableName: 'messages',
    entityTable: 'entities',
    attachmentTable: 'attachments',
    callsTable: 'calls',
    separateCallsTable: true,
    storageBucket: 'attachments',
    
    calculateHash: true,
    deduplicationStrategy: DeduplicationStrategy.STRICT_HASH, // Default robust dedup
    generateUuid: true,
    humanReadableTime: true,
    
    // Splitting Defaults
    splitMethod: SplitMethod.FILE_SIZE_MB, 
    splitThreshold: 500, 
    filenamePattern: '{Source}_{Date}_{Seq}', // Default pattern
    dateSplitGranularity: 'MONTH',
    columns: [],
    
    projectMetadata: {
      caseId: '',
      investigator: '',
      description: ''
    }
  });

  // Data State
  const [discoveredKeys, setDiscoveredKeys] = useState<string[]>([]);
  const [previewData, setPreviewData] = useState<ParsedMessage[]>([]);
  const [dataConfidence, setDataConfidence] = useState<number>(100);
  
  // Analysis State
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<AnalysisResult | null>(null);
  
  // Process State
  const [step, setStep] = useState<'IDLE' | 'SCANNING' | 'CONFIG' | 'CONVERTING' | 'DONE'>('IDLE');
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState<string>('Idle');
  const [error, setError] = useState<string | null>(null);

  const aiService = useRef(new AIService());

  useEffect(() => {
    if (source) {
      scanFile();
    }
  }, [source]); 

  const scanFile = async () => {
    if (!source) return;
    setStep('SCANNING');
    setStatus('Scanning structure & validating...');
    setError(null);
    setPreviewData([]);
    setDiscoveredKeys([]);
    setDataConfidence(100);
    
    try {
      const processor = new XmlStreamProcessor(source, config);
      const { preview, allKeys, detectedTag, confidence } = await processor.scanForColumns(50);
      
      const keysArray = Array.from(allKeys).sort();
      
      setPreviewData(preview);
      setDiscoveredKeys(keysArray);
      setDataConfidence(confidence);
      
      const defaultCols = keysArray.length > 0 ? keysArray : Object.keys(preview[0] || {});
      
      setConfig(prev => ({
          ...prev, 
          itemTag: detectedTag,
          columns: defaultCols 
      }));
      
      setStep('CONFIG');
      setStatus('Scan complete.');
    } catch (err) {
      console.error(err);
      setError('Failed to scan file. Check format.');
      setStep('IDLE');
    }
  };

  const runAnalysis = async () => {
    if (previewData.length === 0) return;
    setIsAnalyzing(true);
    try {
      const result = await aiService.current.analyzeContext(previewData);
      setAnalysisResult(result);
    } catch (e: any) {
      setError(`AI Analysis failed: ${e.message}`);
    } finally {
      setIsAnalyzing(false);
    }
  };

  const startConversion = async () => {
    if (!source) return;
    if (!config.sourceLabel) {
        setError("Source Label is required for deduplication.");
        return;
    }

    if (!config.exportLocalFile && !config.streamToSupabase) {
        setError("Please select at least one output destination (Local File or Cloud Stream).");
        return;
    }

    if (config.streamToSupabase && (!config.supabaseUrl || !config.supabaseKey)) {
        setError("Supabase credentials required for Cloud Stream.");
        return;
    }

    setError(null);
    setStep('CONVERTING');
    setStatus('Selecting output...');

    let dirHandle: FileSystemDirectoryHandle | null = null;

    try {
      if (config.exportLocalFile) {
          if ('showDirectoryPicker' in window) {
             dirHandle = await window.showDirectoryPicker();
          } else {
             throw new Error("Directory Access API not supported.");
          }
      }

      setStatus('Initializing stream...');
      const processor = new XmlStreamProcessor(source, config);
      
      const startTime = Date.now();

      await processor.convert(dirHandle, (count) => {
        setProgress(count);
        const elapsed = (Date.now() - startTime) / 1000;
        const rate = Math.round(count / elapsed);
        setStatus(`Processing... ${count.toLocaleString()} (${rate}/s)`);
      });

      setStatus(`Completed! ${progress} records.`);
      setStep('DONE');

    } catch (err: any) {
      console.error(err);
      setError(err.message);
      setStep('CONFIG');
      setStatus('Failed');
    }
  };

  const closeMobileMenu = () => setIsMobileMenuOpen(false);

  return (
    <div className="flex h-screen overflow-hidden text-zinc-200 font-sans selection:bg-violet-500/30 selection:text-white relative bg-zinc-950">
      {/* Mobile Overlay */}
      {isMobileMenuOpen && (
          <div className="fixed inset-0 bg-black/60 z-30 md:hidden backdrop-blur-sm" onClick={closeMobileMenu}></div>
      )}

      {/* Sidebar Navigation - Force visible on md+ screens */}
      <aside 
        className={`
            fixed md:static inset-y-0 left-0 z-40 w-64 glass-panel border-r border-white/5 flex flex-col 
            transform transition-transform duration-300 ease-in-out
            ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
            bg-zinc-950/90 backdrop-blur-xl
        `}
      >
          <div className="p-6 flex items-center justify-between">
             <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-gradient-to-br from-violet-600 to-indigo-600 rounded-lg flex items-center justify-center shadow-lg shadow-violet-500/20">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-white" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clipRule="evenodd" />
                    </svg>
                </div>
                <div>
                    <span className="font-bold text-lg text-white tracking-tight">ETL Studio</span>
                    <span className="text-[10px] block text-violet-400 font-mono -mt-0.5">PREVIEW BUILD</span>
                </div>
             </div>
             <button onClick={closeMobileMenu} className="md:hidden text-zinc-400 hover:text-white">
                 <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                 </svg>
             </button>
          </div>

          <nav className="flex-1 px-4 space-y-2 mt-4">
              <button 
                onClick={() => { setActiveView('convert'); closeMobileMenu(); }}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'convert' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                   <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.384-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" /></svg>
                   Pipeline
              </button>
              <button 
                onClick={() => { setActiveView('data'); closeMobileMenu(); }}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'data' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" /></svg>
                  Data Browser
              </button>
              <button 
                onClick={() => { setActiveView('query'); closeMobileMenu(); }}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'query' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /></svg>
                  Query Builder
              </button>
              <button 
                onClick={() => { setActiveView('graph'); closeMobileMenu(); }}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'graph' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>
                  Graph Builder
              </button>
              <button 
                onClick={() => { setActiveView('export'); closeMobileMenu(); }}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'export' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                  Export & Chunking
              </button>
              <button 
                 onClick={() => { setActiveView('settings'); closeMobileMenu(); }}
                 className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${activeView === 'settings' ? 'bg-gradient-to-r from-violet-600/20 to-transparent border-l-2 border-violet-500 text-white' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}
              >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                  Settings
              </button>
          </nav>
      </aside>

      {/* MAIN CONTENT */}
      <main className="flex-1 overflow-y-auto relative z-10 w-full flex flex-col">
        <div className="p-4 md:p-8 xl:p-12 max-w-[1800px] mx-auto space-y-6 w-full flex-1 flex flex-col">
            
            <div className="flex justify-between items-end pb-6 border-b border-white/10 shrink-0">
                <div className="flex items-center gap-4">
                   <button 
                       onClick={() => setIsMobileMenuOpen(true)}
                       className="md:hidden p-2 -ml-2 text-zinc-400 hover:text-white rounded-lg hover:bg-white/5 transition-colors"
                   >
                       <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                           <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                       </svg>
                   </button>
                   <h2 className="text-2xl md:text-3xl font-bold text-white tracking-tight capitalize">
                       {activeView.replace('convert', 'Ingestion Pipeline').replace('-', ' ')}
                   </h2>
                </div>
                {step === 'CONVERTING' && activeView === 'convert' && (
                    <div className="text-right hidden sm:block">
                        <div className="text-3xl font-mono text-violet-400 font-bold">{progress.toLocaleString()}</div>
                        <div className="text-xs text-zinc-500 uppercase tracking-widest font-bold">Records Processed</div>
                    </div>
                )}
            </div>

            {error && (
                <div className="bg-red-500/10 border border-red-500/50 text-red-200 p-4 rounded-xl flex items-center gap-3 shadow-lg shadow-red-900/20 backdrop-blur-md">
                    <span className="font-medium text-sm">{error}</span>
                </div>
            )}

            {/* VIEWS */}
            {activeView === 'convert' && (
                <div className="grid grid-cols-1 md:grid-cols-12 gap-8">
                    {/* Left Col: Upload & Status */}
                    <div className="md:col-span-5 lg:col-span-4 space-y-6">
                        <FileDropZone 
                            currentFile={source?.type === 'FILE' ? source.file : null}
                            currentSourceName={source?.type === 'DRIVE' ? source.name : undefined}
                            onFileSelect={(s) => { setSource(s); setAnalysisResult(null); setProgress(0); }} 
                            sourceLabel={config.sourceLabel}
                            onSourceLabelChange={(l) => setConfig(prev => ({...prev, sourceLabel: l}))}
                            googleDriveClientId={config.googleDriveClientId}
                        />
                        
                        {/* Status Card */}
                        <div className="glass-panel border border-white/5 rounded-xl p-5">
                            <h4 className="text-xs font-bold text-zinc-500 uppercase mb-3 tracking-wider">Status</h4>
                            <div className="flex justify-between items-center text-sm mb-3">
                                <span className="text-zinc-200 font-medium">{status}</span>
                                {step === 'CONVERTING' && <span className="w-2 h-2 rounded-full bg-violet-500 animate-pulse shadow-[0_0_8px_rgba(139,92,246,0.6)]"></span>}
                            </div>
                            {(step === 'SCANNING' || step === 'CONVERTING') && (
                                <div className="h-1.5 w-full bg-zinc-800 rounded-full overflow-hidden">
                                    <div className="h-full bg-indigo-500 animate-pulse w-full bg-gradient-to-r from-violet-500 via-fuchsia-500 to-violet-500"></div>
                                </div>
                            )}
                            
                            {/* Validation Score */}
                            {step !== 'IDLE' && (
                                <div className="mt-4 pt-4 border-t border-white/5">
                                    <div className="flex justify-between text-xs mb-1">
                                        <span className="text-zinc-500">Data Confidence</span>
                                        <span className={`font-bold ${dataConfidence > 90 ? 'text-emerald-400' : dataConfidence > 70 ? 'text-yellow-400' : 'text-red-400'}`}>
                                            {Math.round(dataConfidence)}%
                                        </span>
                                    </div>
                                    <div className="h-1 w-full bg-zinc-800 rounded-full overflow-hidden">
                                        <div style={{ width: `${dataConfidence}%` }} className={`h-full ${dataConfidence > 90 ? 'bg-emerald-500' : dataConfidence > 70 ? 'bg-yellow-500' : 'bg-red-500'}`}></div>
                                    </div>
                                </div>
                            )}
                        </div>

                        {/* Quick Configuration - Restored for better UX */}
                        <div className="glass-panel border border-white/5 rounded-xl p-5 space-y-4">
                            <h4 className="text-xs font-bold text-zinc-500 uppercase tracking-wider mb-2">Quick Configuration</h4>
                            
                            <div className="space-y-3">
                                <label className="flex items-center justify-between cursor-pointer group">
                                    <span className="text-sm text-zinc-300">Export Local File</span>
                                    <input 
                                        type="checkbox" 
                                        checked={config.exportLocalFile}
                                        onChange={(e) => setConfig({...config, exportLocalFile: e.target.checked})}
                                        className="accent-violet-500 w-4 h-4"
                                    />
                                </label>
                                {config.exportLocalFile && (
                                    <select 
                                        value={config.localExportFormat}
                                        onChange={(e) => setConfig({...config, localExportFormat: e.target.value as TargetDb})}
                                        className="w-full bg-black/20 border border-white/10 rounded text-xs p-1.5 text-zinc-300"
                                    >
                                        <option value={TargetDb.POSTGRES}>Postgres SQL + CSV</option>
                                        <option value={TargetDb.SQLITE}>SQLite Compatible</option>
                                    </select>
                                )}

                                <label className="flex items-center justify-between cursor-pointer group">
                                    <span className="text-sm text-zinc-300">Stream to Supabase</span>
                                    <input 
                                        type="checkbox" 
                                        checked={config.streamToSupabase}
                                        onChange={(e) => setConfig({...config, streamToSupabase: e.target.checked})}
                                        className="accent-indigo-500 w-4 h-4"
                                    />
                                </label>

                                <div>
                                    <label className="text-xs text-zinc-500 block mb-1">Attachments</label>
                                    <select 
                                        value={config.base64Option}
                                        onChange={(e) => setConfig({...config, base64Option: e.target.value as Base64Option})}
                                        className="w-full bg-black/20 border border-white/10 rounded text-xs p-1.5 text-zinc-300"
                                    >
                                        <option value={Base64Option.SKIP}>Skip (Fastest)</option>
                                        <option value={Base64Option.INLINE}>Inline (Base64)</option>
                                        <option value={Base64Option.SEPARATE_FILE}>Separate File</option>
                                        <option value={Base64Option.EXPORT_IMAGES}>Export Images</option>
                                        <option value={Base64Option.UPLOAD_TO_STORAGE}>Upload Cloud</option>
                                    </select>
                                </div>
                            </div>

                            <div className="pt-2 flex gap-2">
                                <button 
                                    onClick={() => setActiveView('export')}
                                    className="flex-1 py-2 bg-white/5 hover:bg-white/10 rounded text-xs text-zinc-300 transition-colors border border-white/5"
                                >
                                    More Export Settings
                                </button>
                            </div>
                        </div>

                        {/* Action Button */}
                        <button 
                            onClick={startConversion}
                            disabled={step === 'CONVERTING' || !source}
                            className={`
                                w-full py-3 rounded-xl font-bold text-sm uppercase tracking-wide transition-all shadow-lg
                                ${step === 'CONVERTING' ? 'bg-zinc-800 text-zinc-500 cursor-not-allowed' : 'bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-500 hover:to-indigo-500 text-white shadow-violet-900/20'}
                            `}
                        >
                            {step === 'CONVERTING' ? 'Processing...' : 'Run Pipeline'}
                        </button>
                    </div>

                    {/* Right Col: Preview & Analysis */}
                    <div className="md:col-span-7 lg:col-span-8 space-y-6">
                         {step !== 'IDLE' && (
                             <>
                                <AnalysisPanel 
                                    analysis={analysisResult} 
                                    loading={isAnalyzing} 
                                    onAnalyze={runAnalysis} 
                                    hasData={previewData.length > 0} 
                                />
                                <div className="glass-panel border border-white/5 rounded-xl overflow-hidden shadow-2xl">
                                    <div className="p-4 bg-white/5 border-b border-white/5 flex justify-between items-center backdrop-blur-md">
                                        <h3 className="text-sm font-bold text-white">Source Preview</h3>
                                        <span className="text-xs text-zinc-400 bg-black/30 px-2 py-1 rounded border border-white/5 font-mono">
                                            {previewData.length} records sampled
                                        </span>
                                    </div>
                                    <PreviewTable data={previewData} />
                                </div>
                                <ColumnManager 
                                    allKeys={discoveredKeys} 
                                    selectedKeys={config.columns} 
                                    onChange={(cols) => setConfig({...config, columns: cols})} 
                                />
                             </>
                         )}
                    </div>
                </div>
            )}

            {activeView === 'data' && (
                <div className="h-[calc(100vh-14rem)] glass-panel rounded-xl overflow-hidden border border-white/5 shadow-2xl">
                    <DataBrowser config={config} />
                </div>
            )}
            
            {activeView === 'query' && (
                <div className="h-[calc(100vh-14rem)]">
                    <QueryBuilder config={config} />
                </div>
            )}

            {activeView === 'graph' && (
                <div className="h-[calc(100vh-14rem)]">
                    <GraphBuilder config={config} setConfig={setConfig} availableColumns={discoveredKeys} />
                </div>
            )}
            
            {activeView === 'export' && (
                <div className="h-[calc(100vh-14rem)]">
                    <ExportPanel 
                        config={config} 
                        setConfig={setConfig} 
                        onStart={startConversion} 
                        isProcessing={step === 'CONVERTING'} 
                    />
                </div>
            )}

            {activeView === 'settings' && (
                <div className="max-w-4xl glass-panel rounded-xl border border-white/5 shadow-2xl">
                     <SettingsPanel config={config} setConfig={setConfig} disabled={step === 'CONVERTING'} />
                </div>
            )}

        </div>
      </main>
    </div>
  );
};

export default App;