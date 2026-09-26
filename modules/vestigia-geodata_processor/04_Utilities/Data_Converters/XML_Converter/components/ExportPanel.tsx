import React from 'react';
import { Base64Option, ConversionConfig, SplitMethod, TargetDb } from '../types';

interface Props {
  config: ConversionConfig;
  setConfig: React.Dispatch<React.SetStateAction<ConversionConfig>>;
  onStart: () => void;
  isProcessing: boolean;
}

export const ExportPanel: React.FC<Props> = ({ config, setConfig, onStart, isProcessing }) => {
  return (
    <div className="bg-slate-900 rounded-xl border border-slate-800 shadow-2xl h-full flex flex-col">
       <div className="p-6 border-b border-slate-800">
           <h2 className="text-xl font-bold text-white mb-1">Export Configuration</h2>
           <p className="text-zinc-400 text-sm">Manage file naming, splitting, and formats.</p>
       </div>

       <div className="flex-1 overflow-y-auto p-6 space-y-8 custom-scrollbar">
           
           {/* 1. Naming Convention */}
           <div className="space-y-3">
               <h3 className="text-sm font-bold text-blue-400 uppercase tracking-wider">Naming Strategy</h3>
               <div className="bg-slate-800 p-4 rounded-lg border border-slate-700 space-y-3">
                   <div>
                       <label className="text-xs text-zinc-400 block mb-1">Filename Pattern</label>
                       <input 
                            type="text" 
                            value={config.filenamePattern} 
                            onChange={(e) => setConfig({...config, filenamePattern: e.target.value})}
                            placeholder="{Source}_{Date}_{Seq}"
                            className="w-full bg-slate-900 border border-slate-600 rounded p-2 text-white font-mono text-sm"
                       />
                       <p className="text-[10px] text-zinc-500 mt-2">
                           Available Variables: <span className="text-blue-300">{"{Source}"}</span> (Label), <span className="text-blue-300">{"{Date}"}</span> (Current Split Date), <span className="text-blue-300">{"{Seq}"}</span> (1, 2, 3...)
                       </p>
                   </div>
                   <div className="p-2 bg-black/20 rounded border border-white/5 text-xs text-zinc-400 font-mono">
                       Preview: {config.sourceLabel || 'Export'}_{config.dateSplitGranularity === 'YEAR' ? '2023' : '2023-10'}_001.csv
                   </div>
               </div>
           </div>

           {/* 2. Format Selection */}
           <div className="space-y-3">
               <h3 className="text-sm font-bold text-violet-400 uppercase tracking-wider">Output Format</h3>
               <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                   <label className={`p-4 rounded-lg border cursor-pointer transition-all ${config.exportLocalFile ? 'bg-violet-900/20 border-violet-500' : 'bg-slate-800 border-slate-700 hover:border-zinc-500'}`}>
                       <div className="flex justify-between">
                           <span className="font-bold text-white">Local Files (CSV/SQL)</span>
                           <input type="checkbox" checked={config.exportLocalFile} onChange={(e) => setConfig({...config, exportLocalFile: e.target.checked})} className="accent-violet-500" />
                       </div>
                       <p className="text-xs text-zinc-400 mt-2">Writes split CSVs and SQL dump directly to a local folder.</p>
                   </label>
                   
                   <label className={`p-4 rounded-lg border cursor-pointer transition-all ${config.exportNeo4j ? 'bg-orange-900/20 border-orange-500' : 'bg-slate-800 border-slate-700 hover:border-zinc-500'}`}>
                       <div className="flex justify-between">
                           <span className="font-bold text-white">Neo4j Graph (CSV)</span>
                           <input type="checkbox" checked={config.exportNeo4j} onChange={(e) => setConfig({...config, exportNeo4j: e.target.checked})} className="accent-orange-500" />
                       </div>
                       <p className="text-xs text-zinc-400 mt-2">Generates nodes.csv and relationships.csv for admin import.</p>
                   </label>
               </div>
           </div>

           {/* 3. Chunking / Splitting */}
           <div className="space-y-3">
               <div className="flex items-center gap-2">
                   <h3 className="text-sm font-bold text-teal-400 uppercase tracking-wider">File Splitting</h3>
                   <span className="text-[10px] bg-teal-900/30 text-teal-200 px-2 py-0.5 rounded border border-teal-500/30">Memory Safe</span>
               </div>
               
               <div className="bg-slate-800 p-4 rounded-lg border border-slate-700">
                   <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mb-4">
                        <button 
                            onClick={() => setConfig({...config, splitMethod: SplitMethod.NONE})}
                            className={`py-2 text-xs font-bold rounded border ${config.splitMethod === SplitMethod.NONE ? 'bg-slate-600 border-white/20 text-white' : 'bg-slate-900 border-slate-700 text-zinc-500'}`}
                        >
                            No Split
                        </button>
                        <button 
                             onClick={() => setConfig({...config, splitMethod: SplitMethod.FILE_SIZE_MB})}
                             className={`py-2 text-xs font-bold rounded border ${config.splitMethod === SplitMethod.FILE_SIZE_MB ? 'bg-teal-600 border-teal-400 text-white' : 'bg-slate-900 border-slate-700 text-zinc-500'}`}
                        >
                            By Size (MB)
                        </button>
                        <button 
                             onClick={() => setConfig({...config, splitMethod: SplitMethod.ROW_COUNT})}
                             className={`py-2 text-xs font-bold rounded border ${config.splitMethod === SplitMethod.ROW_COUNT ? 'bg-teal-600 border-teal-400 text-white' : 'bg-slate-900 border-slate-700 text-zinc-500'}`}
                        >
                            By Rows
                        </button>
                         <button 
                             onClick={() => setConfig({...config, splitMethod: SplitMethod.DATE})}
                             className={`py-2 text-xs font-bold rounded border ${config.splitMethod === SplitMethod.DATE ? 'bg-teal-600 border-teal-400 text-white' : 'bg-slate-900 border-slate-700 text-zinc-500'}`}
                        >
                            By Date
                        </button>
                   </div>
                   
                   {config.splitMethod === SplitMethod.FILE_SIZE_MB && (
                       <div className="space-y-2 animate-fade-in">
                           <label className="text-xs text-zinc-400">Max File Size (MB)</label>
                           <input 
                                type="number" 
                                value={config.splitThreshold} 
                                onChange={(e) => setConfig({...config, splitThreshold: Number(e.target.value)})}
                                className="w-full bg-slate-900 border border-slate-600 rounded p-2 text-white font-mono"
                           />
                       </div>
                   )}

                    {config.splitMethod === SplitMethod.ROW_COUNT && (
                       <div className="space-y-2 animate-fade-in">
                           <label className="text-xs text-zinc-400">Max Rows Per File</label>
                           <input 
                                type="number" 
                                value={config.splitThreshold} 
                                onChange={(e) => setConfig({...config, splitThreshold: Number(e.target.value)})}
                                className="w-full bg-slate-900 border border-slate-600 rounded p-2 text-white font-mono"
                           />
                       </div>
                   )}

                   {config.splitMethod === SplitMethod.DATE && (
                        <div className="space-y-2 animate-fade-in">
                            <label className="text-xs text-zinc-400">Split Granularity</label>
                            <select 
                                value={config.dateSplitGranularity}
                                onChange={(e) => setConfig({...config, dateSplitGranularity: e.target.value as 'MONTH' | 'YEAR'})}
                                className="w-full bg-slate-900 border border-slate-600 rounded p-2 text-white font-mono text-sm"
                            >
                                <option value="MONTH">Monthly (YYYY-MM)</option>
                                <option value="YEAR">Yearly (YYYY)</option>
                            </select>
                            <p className="text-[10px] text-zinc-500">A new file will be created for each period.</p>
                        </div>
                   )}
               </div>
           </div>
           
           {/* 4. Analysis Prep */}
           <div className="space-y-3">
               <h3 className="text-sm font-bold text-indigo-400 uppercase tracking-wider">Analysis Preparation</h3>
               <div className="p-4 bg-indigo-900/10 border border-indigo-500/30 rounded-lg">
                   <div className="flex items-center gap-2 mb-2">
                       <input type="checkbox" checked={true} readOnly className="accent-indigo-500 opacity-50 cursor-not-allowed" />
                       <span className="text-sm text-indigo-200 font-medium">Include Abuse & Sentiment Columns</span>
                   </div>
                   <p className="text-xs text-zinc-400">
                       Schema automatically includes <code>abuse_score</code>, <code>sentiment_label</code>, and <code>manipulation_tags</code> columns (initialized as NULL) to support post-processing by AI agents.
                   </p>
               </div>
           </div>

       </div>

       <div className="p-6 border-t border-slate-800 bg-slate-950">
           <button 
               onClick={onStart}
               disabled={isProcessing}
               className="w-full py-4 bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-500 hover:to-indigo-500 text-white font-bold text-lg rounded-xl shadow-lg shadow-violet-900/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex justify-center items-center gap-2"
           >
               {isProcessing ? 'Processing...' : 'Start Export Job'}
           </button>
       </div>
    </div>
  );
};
