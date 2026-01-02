
import React, { useState, useEffect } from 'react';
import { ConversionConfig, GraphConfig, GraphNodeConfig, GraphEdgeConfig } from '../types';

interface Props {
    config: ConversionConfig;
    setConfig: React.Dispatch<React.SetStateAction<ConversionConfig>>;
    availableColumns: string[];
}

export const GraphBuilder: React.FC<Props> = ({ config, setConfig, availableColumns }) => {
    const [activeTab, setActiveTab] = useState<'nodes' | 'edges'>('nodes');

    // Safe Initialization via useEffect to prevent render loop errors
    useEffect(() => {
        if (!config.graphConfig) {
            const defaultConfig: GraphConfig = {
                nodes: [
                    { id: '1', label: 'Person', sourceColumn: 'address', properties: ['contact_name', 'abuse_score'] },
                    { id: '2', label: 'Message', sourceColumn: 'uuid_v7', properties: ['date_iso', 'body', 'type', 'sentiment_label', 'manipulation_tags'] }
                ],
                edges: [
                    { id: 'e1', type: 'SENT', sourceNodeId: '1', targetNodeId: '2', properties: [], condition: "type == '2'" },
                    { id: 'e2', type: 'RECEIVED', sourceNodeId: '1', targetNodeId: '2', properties: [], condition: "type == '1'" }
                ]
            };
            setConfig(prev => ({ ...prev, graphConfig: defaultConfig }));
        }
    }, [config.graphConfig, setConfig]);

    if (!config.graphConfig) {
        return <div className="p-10 text-center text-zinc-500 animate-pulse">Initializing Graph Schema...</div>;
    }

    const { nodes, edges } = config.graphConfig;

    const updateGraphConfig = (newConfig: GraphConfig) => {
        setConfig({ ...config, graphConfig: newConfig });
    };

    const importGraphSchema = async () => {
        try {
            const [fileHandle] = await window.showOpenFilePicker({
                types: [{ description: 'Graph Schema', accept: { 'application/json': ['.json'] } }],
                multiple: false
            });
            const file = await fileHandle.getFile();
            const text = await file.text();
            const json = JSON.parse(text);
            if (json.nodes && json.edges) {
                updateGraphConfig(json);
                alert("Graph Schema Imported");
            }
        } catch (e) { }
    };

    const exportGraphSchema = () => {
        const blob = new Blob([JSON.stringify(config.graphConfig, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `neo4j_schema_${Date.now()}.json`;
        a.click();
        URL.revokeObjectURL(url);
    };

    // -- Node Handlers --
    const addNode = () => {
        const newNode: GraphNodeConfig = {
            id: Date.now().toString(),
            label: 'NewNode',
            sourceColumn: availableColumns[0] || 'id',
            properties: []
        };
        updateGraphConfig({ ...config.graphConfig!, nodes: [...nodes, newNode] });
    };

    const deleteNode = (id: string) => {
        updateGraphConfig({ ...config.graphConfig!, nodes: nodes.filter(n => n.id !== id) });
    };

    const updateNode = (id: string, field: keyof GraphNodeConfig, value: any) => {
        updateGraphConfig({
            ...config.graphConfig!,
            nodes: nodes.map(n => n.id === id ? { ...n, [field]: value } : n)
        });
    };

    const toggleNodeProp = (id: string, prop: string) => {
        const node = nodes.find(n => n.id === id);
        if (!node) return;
        const newProps = node.properties.includes(prop) 
            ? node.properties.filter(p => p !== prop)
            : [...node.properties, prop];
        updateNode(id, 'properties', newProps);
    };

    // -- Edge Handlers --
    const addEdge = (presetType: string = 'INTERACTED_WITH') => {
         const newEdge: GraphEdgeConfig = {
             id: Date.now().toString(),
             type: presetType,
             sourceNodeId: nodes[0]?.id || '1',
             targetNodeId: nodes[1]?.id || '2',
             properties: [],
             condition: "true"
         };
         updateGraphConfig({ ...config.graphConfig!, edges: [...edges, newEdge] });
    };

    const deleteEdge = (id: string) => {
         updateGraphConfig({ ...config.graphConfig!, edges: edges.filter(e => e.id !== id) });
    };

    const updateEdge = (id: string, field: keyof GraphEdgeConfig, value: any) => {
        updateGraphConfig({
            ...config.graphConfig!,
            edges: edges.map(e => e.id === id ? { ...e, [field]: value } : e)
        });
    };

    return (
        <div className="h-full flex flex-col gap-6">
            <div className="flex justify-between items-center bg-slate-900 p-4 rounded-lg border border-slate-800">
                <div>
                    <h3 className="text-lg font-bold text-white">Graph Schema Builder</h3>
                    <p className="text-sm text-zinc-400">Define Neo4j structure.</p>
                </div>
                <div className="flex gap-2">
                     <button onClick={importGraphSchema} className="px-3 py-1 bg-slate-800 text-xs text-zinc-300 rounded hover:bg-slate-700">Import Schema</button>
                     <button onClick={exportGraphSchema} className="px-3 py-1 bg-slate-800 text-xs text-zinc-300 rounded hover:bg-slate-700">Export Schema</button>
                     <div className="h-6 w-px bg-slate-700 mx-2"></div>
                     <button 
                        onClick={() => setActiveTab('nodes')}
                        className={`px-4 py-2 rounded text-sm font-medium transition-colors ${activeTab === 'nodes' ? 'bg-orange-600 text-white shadow-lg' : 'bg-slate-800 text-zinc-400 hover:bg-slate-700'}`}
                     >
                        Nodes
                     </button>
                     <button 
                        onClick={() => setActiveTab('edges')}
                        className={`px-4 py-2 rounded text-sm font-medium transition-colors ${activeTab === 'edges' ? 'bg-orange-600 text-white shadow-lg' : 'bg-slate-800 text-zinc-400 hover:bg-slate-700'}`}
                     >
                        Relationships
                     </button>
                </div>
            </div>

            <div className="flex-1 overflow-y-auto bg-black/20 rounded-xl border border-white/5 p-6 custom-scrollbar">
                
                {/* NODES EDITOR */}
                {activeTab === 'nodes' && (
                    <div className="space-y-6 animate-fade-in">
                        {nodes.map(node => (
                            <div key={node.id} className="bg-slate-900 border border-slate-700 rounded-lg p-4 shadow-lg">
                                <div className="flex justify-between mb-4">
                                    <h4 className="font-mono text-orange-400 font-bold flex items-center gap-2 text-lg">
                                        (:{node.label})
                                    </h4>
                                    <button onClick={() => deleteNode(node.id)} className="text-zinc-600 hover:text-red-400 transition-colors">
                                        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" /></svg>
                                    </button>
                                </div>
                                
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div>
                                        <label className="text-xs text-zinc-500 uppercase block mb-1">Label</label>
                                        <input 
                                            type="text" 
                                            value={node.label}
                                            onChange={(e) => updateNode(node.id, 'label', e.target.value)}
                                            className="w-full bg-black/40 border border-slate-700 rounded px-2 py-1 text-sm text-white focus:border-orange-500 outline-none"
                                        />
                                    </div>
                                    <div>
                                        <label className="text-xs text-zinc-500 uppercase block mb-1">Primary Key (Source Column)</label>
                                        <select 
                                            value={node.sourceColumn}
                                            onChange={(e) => updateNode(node.id, 'sourceColumn', e.target.value)}
                                            className="w-full bg-black/40 border border-slate-700 rounded px-2 py-1 text-sm text-white focus:border-orange-500 outline-none"
                                        >
                                            {availableColumns.map(c => <option key={c} value={c}>{c}</option>)}
                                        </select>
                                    </div>
                                </div>

                                <div className="mt-4">
                                    <label className="text-xs text-zinc-500 uppercase block mb-2">Properties to Include</label>
                                    <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto pr-2">
                                        {availableColumns.map(col => (
                                            <button
                                                key={col}
                                                onClick={() => toggleNodeProp(node.id, col)}
                                                className={`px-2 py-1 rounded text-xs border transition-colors ${node.properties.includes(col) ? 'bg-orange-900/40 border-orange-500 text-orange-200' : 'bg-slate-800 border-slate-700 text-zinc-500 hover:text-zinc-300'}`}
                                            >
                                                {col}
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ))}
                        <button onClick={addNode} className="w-full py-3 border border-dashed border-slate-700 rounded-lg text-zinc-500 hover:text-white hover:border-slate-500 transition-colors">
                            + Add Node Definition
                        </button>
                    </div>
                )}

                {/* EDGES EDITOR */}
                {activeTab === 'edges' && (
                    <div className="space-y-6 animate-fade-in">
                        {edges.map(edge => (
                            <div key={edge.id} className="bg-slate-900 border border-slate-700 rounded-lg p-4 shadow-lg flex flex-col gap-4">
                                <div className="flex justify-between items-center">
                                     <div className="flex items-center gap-2">
                                         <span className="text-zinc-400 text-sm">(:{nodes.find(n => n.id === edge.sourceNodeId)?.label || 'Src'})</span>
                                         <span className="text-slate-600">→</span>
                                         <input 
                                            type="text" 
                                            value={edge.type}
                                            onChange={(e) => updateEdge(edge.id, 'type', e.target.value)}
                                            className="bg-black/40 border border-slate-700 rounded px-2 py-1 text-xs text-orange-400 font-mono font-bold w-32 text-center"
                                         />
                                         <span className="text-slate-600">→</span>
                                         <span className="text-zinc-400 text-sm">(:{nodes.find(n => n.id === edge.targetNodeId)?.label || 'Tgt'})</span>
                                     </div>
                                     <button onClick={() => deleteEdge(edge.id)} className="text-zinc-600 hover:text-red-400">×</button>
                                </div>

                                <div className="bg-black/20 p-2 rounded border border-white/5">
                                    <label className="text-[10px] text-zinc-500 uppercase block mb-1">Logic Condition (JS)</label>
                                    <input 
                                        type="text"
                                        value={edge.condition || ''}
                                        onChange={(e) => updateEdge(edge.id, 'condition', e.target.value)}
                                        className="w-full bg-transparent text-xs font-mono text-green-400 outline-none placeholder:text-zinc-700"
                                        placeholder="e.g. type == '1'"
                                    />
                                    <p className="text-[10px] text-zinc-600 mt-1">
                                        Use row fields variables (e.g. <code>type</code>, <code>status</code>). Return true to create edge.
                                    </p>
                                </div>
                            </div>
                        ))}
                        
                        <div className="grid grid-cols-2 gap-3">
                             <button onClick={() => addEdge('INTERACTED_WITH')} className="py-2 bg-slate-800 rounded border border-slate-700 text-xs text-zinc-400 hover:text-white hover:bg-slate-700 transition-colors">
                                + Standard Relationship
                             </button>
                             <div className="relative group">
                                <button className="w-full py-2 bg-indigo-900/30 rounded border border-indigo-500/30 text-xs text-indigo-300 hover:text-white hover:bg-indigo-900/50 transition-colors">
                                    + Complex Relationship
                                </button>
                                <div className="absolute bottom-full left-0 w-full mb-1 bg-slate-900 border border-slate-700 rounded shadow-xl hidden group-hover:block z-20">
                                    <button onClick={() => addEdge('TRIANGULATION')} className="w-full text-left px-3 py-2 text-xs text-zinc-400 hover:bg-slate-800 hover:text-white">Triangulation</button>
                                    <button onClick={() => addEdge('MANIPULATIVE')} className="w-full text-left px-3 py-2 text-xs text-zinc-400 hover:bg-slate-800 hover:text-white">Manipulative Behavior</button>
                                    <button onClick={() => addEdge('GASLIGHTING')} className="w-full text-left px-3 py-2 text-xs text-zinc-400 hover:bg-slate-800 hover:text-white">Gaslighting</button>
                                </div>
                             </div>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};
