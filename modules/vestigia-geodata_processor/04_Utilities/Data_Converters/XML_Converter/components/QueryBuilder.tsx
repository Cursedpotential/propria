import React, { useState, useEffect } from 'react';
import { ConversionConfig, SavedQuery } from '../types';

interface Props {
    config: ConversionConfig;
}

// Mock Table Data for Visualization
const TABLE_SCHEMAS: Record<string, string[]> = {
    'messages': ['uuid_v7', 'date_iso', 'body', 'address', 'type', 'source_id', 'record_hash'],
    'entities': ['id', 'phone_number', 'primary_name', 'last_seen'],
    'attachments': ['id', 'message_id', 'filename', 'content_type', 'storage_path']
};

export const QueryBuilder: React.FC<Props> = ({ config }) => {
    const [selectedTables, setSelectedTables] = useState<string[]>(['messages']);
    const [joins, setJoins] = useState<{from: string, to: string, on: string}[]>([]);
    const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
    const [currentQueryName, setCurrentQueryName] = useState('');
    
    useEffect(() => {
        const loaded = localStorage.getItem('etl_saved_queries');
        if (loaded) {
            setSavedQueries(JSON.parse(loaded));
        }
    }, []);

    const saveQuery = () => {
        if (!currentQueryName) return;
        const newQuery: SavedQuery = {
            id: Date.now().toString(),
            name: currentQueryName,
            tables: selectedTables,
            joins: joins,
            createdAt: Date.now()
        };
        const updated = [...savedQueries, newQuery];
        setSavedQueries(updated);
        localStorage.setItem('etl_saved_queries', JSON.stringify(updated));
        setCurrentQueryName('');
    };

    const loadQuery = (q: SavedQuery) => {
        setSelectedTables(q.tables);
        setJoins(q.joins);
    };

    const deleteQuery = (id: string) => {
        const updated = savedQueries.filter(q => q.id !== id);
        setSavedQueries(updated);
        localStorage.setItem('etl_saved_queries', JSON.stringify(updated));
    };

    // Simple state to simulate "dragging" or adding tables
    const addTable = (table: string) => {
        if (!selectedTables.includes(table)) {
            setSelectedTables([...selectedTables, table]);
            // Auto join suggestion
            if (table === 'entities' && selectedTables.includes('messages')) {
                setJoins([...joins, { from: 'messages.address', to: 'entities.phone_number', on: 'Left Join' }]);
            }
            if (table === 'attachments' && selectedTables.includes('messages')) {
                setJoins([...joins, { from: 'messages.uuid_v7', to: 'attachments.message_id', on: 'Inner Join' }]);
            }
        }
    };

    const removeTable = (table: string) => {
        setSelectedTables(selectedTables.filter(t => t !== table));
        setJoins(joins.filter(j => !j.from.startsWith(table) && !j.to.startsWith(table)));
    };

    const generateSQL = () => {
        if (selectedTables.length === 0) return '-- Select tables to generate query';
        
        let query = `SELECT\n  ${selectedTables[0]}.*\nFROM ${selectedTables[0]}`;
        
        joins.forEach(j => {
            const type = j.on === 'Left Join' ? 'LEFT JOIN' : 'INNER JOIN';
            const targetTable = j.to.split('.')[0];
            query += `\n${type} ${targetTable} ON ${j.from} = ${j.to}`;
        });
        
        query += `\nLIMIT 100;`;
        return query;
    };

    return (
        <div className="h-full flex flex-col md:flex-row gap-6">
            {/* Sidebar / Sources */}
            <div className="w-full md:w-64 flex-shrink-0 space-y-4">
                <div className="bg-slate-900 rounded-xl border border-slate-800 p-4">
                    <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">Available Sources</h3>
                    <div className="space-y-2">
                        {Object.keys(TABLE_SCHEMAS).map(table => (
                            <button 
                                key={table}
                                onClick={() => addTable(table)}
                                disabled={selectedTables.includes(table)}
                                className="w-full text-left px-3 py-2 rounded bg-slate-800 hover:bg-slate-700 border border-slate-700 text-sm text-slate-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex justify-between items-center group"
                            >
                                <span>{table}</span>
                                <span className="opacity-0 group-hover:opacity-100 text-indigo-400 text-xs">+</span>
                            </button>
                        ))}
                         <button className="w-full text-left px-3 py-2 rounded border border-dashed border-slate-700 text-sm text-slate-500 hover:text-slate-400 hover:border-slate-600 transition-colors">
                            + Add External Source...
                        </button>
                    </div>
                </div>
                
                 <div className="bg-slate-900 rounded-xl border border-slate-800 p-4">
                    <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-3">Saved Queries</h3>
                    <div className="space-y-2">
                        {savedQueries.length === 0 ? (
                             <div className="text-sm text-slate-500 italic">No saved queries.</div>
                        ) : (
                            savedQueries.map(q => (
                                <div key={q.id} className="flex items-center justify-between group">
                                    <button onClick={() => loadQuery(q)} className="text-sm text-slate-300 hover:text-indigo-400 text-left truncate flex-1">
                                        {q.name}
                                    </button>
                                    <button onClick={() => deleteQuery(q.id)} className="text-slate-600 hover:text-red-400 opacity-0 group-hover:opacity-100 px-1">
                                        ×
                                    </button>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>

            {/* Canvas / Builder Area */}
            <div className="flex-1 flex flex-col gap-6">
                <div className="bg-slate-900/50 rounded-xl border border-slate-700/50 p-6 flex-1 relative overflow-hidden">
                     <div className="absolute inset-0 opacity-10 pointer-events-none" 
                          style={{ backgroundImage: 'radial-gradient(#4f46e5 1px, transparent 1px)', backgroundSize: '20px 20px' }}>
                     </div>
                     
                     <div className="relative z-10 flex flex-wrap gap-8 items-start">
                         {selectedTables.map((table, idx) => (
                             <div key={table} className="flex items-center">
                                 {/* Table Card */}
                                 <div className="w-48 bg-slate-800 rounded-lg border border-slate-600 shadow-xl overflow-hidden">
                                     <div className="bg-slate-900 px-3 py-2 border-b border-slate-700 flex justify-between items-center">
                                         <span className="text-xs font-bold text-indigo-300">{table}</span>
                                         <button onClick={() => removeTable(table)} className="text-slate-500 hover:text-red-400">×</button>
                                     </div>
                                     <div className="p-2 space-y-1">
                                         {TABLE_SCHEMAS[table].slice(0, 5).map(col => (
                                             <div key={col} className="text-[10px] text-slate-400 px-1 py-0.5 rounded hover:bg-slate-700 cursor-pointer">{col}</div>
                                         ))}
                                         <div className="text-[10px] text-slate-600 px-1 italic">...more</div>
                                     </div>
                                 </div>

                                 {/* Connection Line Visual (Mock) */}
                                 {idx < selectedTables.length - 1 && (
                                     <div className="w-12 h-0.5 bg-indigo-500/50 mx-2 relative">
                                         <div className="absolute -top-3 left-1/2 -translate-x-1/2 text-[9px] bg-slate-900 text-indigo-300 px-1 rounded border border-indigo-500/30">
                                             JOIN
                                         </div>
                                     </div>
                                 )}
                             </div>
                         ))}
                     </div>
                </div>

                {/* SQL Preview & Save */}
                <div className="bg-slate-950 rounded-xl border border-slate-800 p-0 overflow-hidden shadow-2xl">
                    <div className="bg-slate-900 px-4 py-2 border-b border-slate-800 flex justify-between items-center">
                        <span className="text-xs font-bold text-emerald-500 font-mono">GENERATED SQL</span>
                        <div className="flex gap-2">
                             <div className="flex bg-black/20 rounded border border-slate-700">
                                <input 
                                    type="text" 
                                    placeholder="Query Name" 
                                    value={currentQueryName}
                                    onChange={(e) => setCurrentQueryName(e.target.value)}
                                    className="bg-transparent text-xs px-2 py-1 outline-none text-white w-32"
                                />
                                <button 
                                    onClick={saveQuery}
                                    disabled={!currentQueryName}
                                    className="text-xs bg-slate-700 hover:bg-slate-600 text-white px-2 py-1 border-l border-slate-600 disabled:opacity-50"
                                >
                                    Save
                                </button>
                             </div>
                             <button className="text-xs bg-indigo-600 hover:bg-indigo-500 text-white px-3 py-1 rounded transition-colors">Run Query</button>
                        </div>
                    </div>
                    <pre className="p-4 text-xs font-mono text-slate-300 overflow-x-auto bg-[#0d1117]">
                        {generateSQL()}
                    </pre>
                </div>
            </div>
        </div>
    );
};
