import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { createClient } from '@supabase/supabase-js';
import { ConversionConfig } from '../types';
import { InfiniteList } from './InfiniteList';

interface Props {
  config: ConversionConfig;
}

export const DataBrowser: React.FC<Props> = ({ config }) => {
  const [activeTable, setActiveTable] = useState<string>(config.tableName || 'messages');
  
  // Memoize client to prevent recreation on every render unless config changes
  const client = useMemo(() => {
    if (!config.supabaseUrl || !config.supabaseKey) return null;
    return createClient(config.supabaseUrl, config.supabaseKey);
  }, [config.supabaseUrl, config.supabaseKey]);

  // Define render item based on active table to display relevant columns
  const renderRow = (row: any, index: number) => {
      const keys = Object.keys(row).slice(0, 6); // Show first 6 columns
      return (
        <div key={row.id || index} className="flex border-b border-white/5 hover:bg-white/5 transition-colors text-xs font-mono text-zinc-300">
            {keys.map(k => {
                 const val = row[k];
                 const isDate = k.includes('date') || k.includes('seen');
                 const isId = k.includes('id') || k.includes('uuid');
                 return (
                     <div key={k} className={`flex-1 p-3 truncate ${isDate ? 'text-violet-300' : isId ? 'text-emerald-400/80' : ''}`} title={String(val)}>
                         {typeof val === 'object' ? JSON.stringify(val) : String(val)}
                     </div>
                 )
            })}
        </div>
      );
  };

  const orderByDate = useCallback((query: any) => {
      const col = activeTable === (config.tableName || 'messages') ? 'date_iso' : 'message_date';
      return query.order(col, { ascending: false });
  }, [activeTable, config.tableName]);

  return (
    <div className="bg-transparent flex flex-col h-full">
        {/* Toolbar */}
        <div className="p-4 border-b border-white/5 flex flex-col md:flex-row justify-between items-start md:items-center bg-black/20 backdrop-blur-md gap-4 z-10">
            <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4 w-full md:w-auto">
                 <div className="flex bg-black/40 rounded-lg p-1 border border-white/10 w-full sm:w-auto">
                    <button 
                        onClick={() => setActiveTable(config.tableName || 'messages')}
                        className={`flex-1 sm:flex-none px-4 py-1.5 rounded-md text-xs font-bold transition-all ${activeTable === (config.tableName || 'messages') ? 'bg-violet-600 text-white shadow-lg' : 'text-zinc-400 hover:text-white'}`}
                    >
                        Messages
                    </button>
                    <button 
                        onClick={() => setActiveTable(config.attachmentTable || 'attachments')}
                        className={`flex-1 sm:flex-none px-4 py-1.5 rounded-md text-xs font-bold transition-all ${activeTable === (config.attachmentTable || 'attachments') ? 'bg-violet-600 text-white shadow-lg' : 'text-zinc-400 hover:text-white'}`}
                    >
                        Attachments
                    </button>
                 </div>
                 <span className="text-zinc-500 text-xs font-mono hidden lg:inline-block">
                     public.{activeTable}
                 </span>
            </div>
            
            {!client && (
                 <div className="text-red-400 text-xs flex items-center gap-2">
                     <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
                     Disconnected (Check Settings)
                 </div>
            )}
        </div>

        {/* Infinite List Container */}
        <div className="flex-1 overflow-hidden relative bg-black/10">
            {client ? (
                <>
                    {/* Header Row Simulation (Static) */}
                    <div className="flex border-b border-white/10 bg-zinc-900/90 text-xs font-bold text-zinc-500 uppercase tracking-wider sticky top-0 z-10 shadow-sm">
                        {/* We don't know columns ahead of time easily without a fetch, so we just use a generic header or wait for first render. 
                            For this simulation, we'll let the content define width. 
                        */}
                        <div className="p-3 w-full text-center italic opacity-50">
                            Dynamic Column View - {activeTable}
                        </div>
                    </div>
                    
                    <InfiniteList 
                        key={activeTable} // Force remount on table switch
                        client={client}
                        tableName={activeTable}
                        pageSize={50}
                        renderItem={renderRow}
                        trailingQuery={orderByDate}
                        renderNoResults={() => (
                            <div className="flex flex-col items-center justify-center h-full text-zinc-500 space-y-4 pt-20">
                                <div className="bg-zinc-800/50 p-6 rounded-full">
                                    <svg xmlns="http://www.w3.org/2000/svg" className="h-10 w-10 text-zinc-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                                    </svg>
                                </div>
                                <p className="text-sm font-medium">Table is empty or loading...</p>
                                <p className="text-xs max-w-xs text-center opacity-70">
                                    Run the ingestion pipeline to populate <code>{activeTable}</code> with data.
                                </p>
                            </div>
                        )}
                    />
                </>
            ) : (
                <div className="absolute inset-0 flex flex-col items-center justify-center text-zinc-500 space-y-4">
                     <p>Please configure Supabase credentials in Settings.</p>
                </div>
            )}
        </div>
    </div>
  );
};