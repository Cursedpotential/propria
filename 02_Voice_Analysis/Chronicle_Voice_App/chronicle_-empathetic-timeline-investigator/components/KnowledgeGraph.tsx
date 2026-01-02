import React, { useEffect, useRef, useState, useMemo } from 'react';
import { 
    select, 
    zoom as d3Zoom, 
    forceSimulation, 
    forceLink, 
    forceManyBody, 
    forceCenter, 
    forceCollide, 
    drag as d3Drag 
} from 'd3';
import { AppState, Entity, EntityRelationship, ContextFact, ContextPhase } from '../types';
import { Network, Plus, Trash2, X, Database, Share2, Image as ImageIcon, Calendar, BrainCircuit, Sparkles, FileKey, BookOpen, Layers, Activity, AlertCircle } from 'lucide-react';
import { GoogleGenAI } from "@google/genai";

interface KnowledgeGraphProps {
  data: AppState;
  onUpdate: (entities: Entity[], relationships: EntityRelationship[]) => void;
  onUpdateContext?: (facts: ContextFact[], phases: ContextPhase[]) => void;
}

const KnowledgeGraph: React.FC<KnowledgeGraphProps> = ({ data, onUpdate, onUpdateContext }) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  
  // UI State
  const [selectedNode, setSelectedNode] = useState<Entity | null>(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const [showContextPanel, setShowContextPanel] = useState(false);
  const [showAiMenu, setShowAiMenu] = useState(false);
  const [aiAnalysisResult, setAiAnalysisResult] = useState<{title: string, content: string} | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  
  // Linking State
  const [isLinking, setIsLinking] = useState(false);
  const [linkSource, setLinkSource] = useState<string | null>(null);

  // Forms
  const [newEntityName, setNewEntityName] = useState('');
  const [newEntityType, setNewEntityType] = useState<Entity['type']>('person');
  
  // Relationship Form
  const [newRelLabel, setNewRelLabel] = useState('');
  const [newRelType, setNewRelType] = useState<EntityRelationship['type']>('professional');
  const [newRelStart, setNewRelStart] = useState('');
  const [newRelEnd, setNewRelEnd] = useState('');

  // Prepare Graph Data
  const graphData = useMemo(() => {
    const nodes = data.entities.map(e => ({
      ...e,
      id: e.id,
      radius: e.type === 'person' ? (e.impactOnCase === 'high' ? 35 : 20) : 15, // Larger for high impact
      color: e.type === 'person' ? (e.impactOnCase === 'high' ? '#ef4444' : '#6366f1') : 
             e.type === 'location' ? '#10b981' : 
             e.type === 'workplace' ? '#f59e0b' : 
             e.type === 'school' ? '#8b5cf6' : '#64748b'
    }));

    const links = data.entityRelationships.map(r => ({
      ...r,
      source: r.sourceId,
      target: r.targetId
    }));

    return { nodes, links };
  }, [data.entities, data.entityRelationships]);

  // --- D3 RENDER LOGIC ---
  useEffect(() => {
    if (!svgRef.current || !wrapperRef.current) return;

    const width = wrapperRef.current.clientWidth;
    const height = wrapperRef.current.clientHeight;

    const svg = select(svgRef.current);
    svg.selectAll("*").remove(); 

    // Define arrow markers
    const defs = svg.append("defs");
    defs.selectAll("marker")
      .data(["end"])
      .enter().append("marker")
      .attr("id", "arrow")
      .attr("viewBox", "0 -5 10 10")
      .attr("refX", 25)
      .attr("refY", 0)
      .attr("markerWidth", 6)
      .attr("markerHeight", 6)
      .attr("orient", "auto")
      .append("path")
      .attr("d", "M0,-5L10,0L0,5")
      .attr("fill", "#64748b");
    
    // Define Glow Filter for High Impact Nodes
    const filter = defs.append("filter")
        .attr("id", "glow");
    filter.append("feGaussianBlur")
        .attr("stdDeviation", "2.5")
        .attr("result", "coloredBlur");
    const feMerge = filter.append("feMerge");
    feMerge.append("feMergeNode").attr("in", "coloredBlur");
    feMerge.append("feMergeNode").attr("in", "SourceGraphic");

    const g = svg.append("g");
    const zoomBehavior = d3Zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on("zoom", (event) => g.attr("transform", event.transform));
    
    svg.call(zoomBehavior);

    const simulation = forceSimulation(graphData.nodes as any)
      .force("link", forceLink(graphData.links).id((d: any) => d.id).distance(150))
      .force("charge", forceManyBody().strength(-400))
      .force("center", forceCenter(width / 2, height / 2))
      .force("collide", forceCollide().radius((d: any) => d.radius + 15));

    const link = g.append("g").attr("class", "links")
      .selectAll("line")
      .data(graphData.links)
      .enter().append("g");

    link.append("line")
      .attr("stroke", "#475569")
      .attr("stroke-width", 1.5)
      .attr("marker-end", "url(#arrow)");

    // Link Labels
    link.append("text")
      .text((d: any) => d.label)
      .attr("fill", "#94a3b8")
      .attr("font-size", "9px")
      .attr("text-anchor", "middle")
      .attr("dy", -5)
      .attr("font-weight", "bold")
      .style("pointer-events", "none")
      .style("text-transform", "uppercase");

    // Link Dates (if present)
    link.append("text")
      .text((d: any) => d.startDate ? `${d.startDate}${d.endDate ? '-' + d.endDate : '+'}` : '')
      .attr("fill", "#64748b")
      .attr("font-size", "7px")
      .attr("text-anchor", "middle")
      .attr("dy", 8)
      .style("pointer-events", "none");

    const node = g.append("g").attr("class", "nodes")
      .selectAll("g")
      .data(graphData.nodes)
      .enter().append("g")
      .call(d3Drag<any, any>()
        .on("start", (e, d) => { if (!e.active) simulation.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
        .on("drag", (e, d) => { d.fx = e.x; d.fy = e.y; })
        .on("end", (e, d) => { if (!e.active) simulation.alphaTarget(0); d.fx = null; d.fy = null; }));

    // Node Circle
    node.append("circle")
      .attr("r", (d: any) => d.radius)
      .attr("fill", (d: any) => d.color)
      .attr("stroke", (d: any) => d.impactOnCase === 'high' ? '#fca5a5' : '#1e293b') // Red border for high impact
      .attr("stroke-width", (d: any) => d.impactOnCase === 'high' ? 4 : 2)
      .attr("cursor", "pointer")
      .style("filter", (d: any) => d.impactOnCase === 'high' ? "url(#glow)" : "") // Glow for high impact
      .on("click", (event, d) => {
          if (isLinking && linkSource) {
              completeRelationship(linkSource, d.id);
          } else {
              setSelectedNode(d as Entity);
              setIsSidebarOpen(true);
          }
          event.stopPropagation();
      });

    // Label Background
    node.append("rect")
      .attr("rx", 4).attr("ry", 4).attr("fill", "#0f172a").attr("opacity", 0.7)
      .attr("width", (d: any) => d.name.length * 7 + 10)
      .attr("height", 14)
      .attr("x", (d: any) => -(d.name.length * 7 + 10) / 2)
      .attr("y", (d: any) => d.radius + 8);

    // Node Label
    node.append("text")
      .text((d: any) => d.name)
      .attr("fill", "white").attr("font-size", "10px").attr("font-weight", "bold")
      .attr("text-anchor", "middle").attr("y", (d: any) => d.radius + 18)
      .style("pointer-events", "none");

    // Node Icon/Type Indicator
    node.append("text")
        .attr("text-anchor", "middle").attr("dominant-baseline", "central")
        .attr("fill", "white").attr("font-size", (d: any) => d.radius)
        .style("pointer-events", "none")
        .text((d: any) => {
            if(d.impactOnCase === 'high') return '★'; // Star for high impact
            if(d.type === 'person') return '';
            if(d.type === 'workplace') return '🏢';
            if(d.type === 'location') return '📍';
            if(d.type === 'school') return '🎓';
            return '•';
        });

    simulation.on("tick", () => {
      link.select("line")
        .attr("x1", (d: any) => d.source.x).attr("y1", (d: any) => d.source.y)
        .attr("x2", (d: any) => d.target.x).attr("y2", (d: any) => d.target.y);
      link.selectAll("text")
        .attr("x", (d: any) => (d.source.x + d.target.x) / 2)
        .attr("y", (d: any, i, nodes) => {
            const base = (d.source.y + d.target.y) / 2;
            const element = select(nodes[i]);
            const dy = parseFloat(element.attr("dy"));
            return base + (isNaN(dy) ? 0 : dy); 
        });
        
      node.attr("transform", (d: any) => `translate(${d.x},${d.y})`);
    });

    return () => { simulation.stop(); };
  }, [graphData, isLinking, linkSource]);

  // --- ACTIONS ---
  const addEntity = () => {
      if (!newEntityName) return;
      const newEntity: Entity = {
          id: crypto.randomUUID(),
          name: newEntityName,
          type: newEntityType,
          impactOnCase: 'medium',
          notes: '',
          relationship: ''
      };
      onUpdate([...data.entities, newEntity], data.entityRelationships);
      setNewEntityName('');
  };

  const updateEntity = (id: string, updates: Partial<Entity>) => {
      const updatedEntities = data.entities.map(e => e.id === id ? { ...e, ...updates } : e);
      onUpdate(updatedEntities, data.entityRelationships);
      if (selectedNode && selectedNode.id === id) {
          setSelectedNode({ ...selectedNode, ...updates });
      }
  };

  const startLinking = () => {
      if (selectedNode) {
          setLinkSource(selectedNode.id);
          setIsLinking(true);
          setIsSidebarOpen(false);
          setNewRelLabel('');
          setNewRelStart('');
          setNewRelEnd('');
      }
  };

  const completeRelationship = (source: string, target: string) => {
      if (source === target) return;
      const newRel: EntityRelationship = {
          id: crypto.randomUUID(),
          sourceId: source,
          targetId: target,
          label: newRelLabel || 'Connected',
          type: newRelType,
          startDate: newRelStart || undefined,
          endDate: newRelEnd || undefined
      };
      onUpdate(data.entities, [...data.entityRelationships, newRel]);
      setIsLinking(false);
      setLinkSource(null);
  };

  const deleteSelectedEntity = () => {
      if (!selectedNode) return;
      if (window.confirm(`Delete ${selectedNode.name}?`)) {
          const newEntities = data.entities.filter(e => e.id !== selectedNode.id);
          const newRels = data.entityRelationships.filter(r => r.sourceId !== selectedNode.id && r.targetId !== selectedNode.id);
          onUpdate(newEntities, newRels);
          setSelectedNode(null);
          setIsSidebarOpen(false);
      }
  };

  const deleteRelationship = (relId: string) => {
      onUpdate(data.entities, data.entityRelationships.filter(r => r.id !== relId));
  };

  // --- GEMINI INTELLIGENCE ---
  const handleGraphAnalysis = async (mode: 'Isolation' | 'Influence' | 'Timeline' | 'General') => {
      setIsAnalyzing(true);
      try {
          const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
          const model = data.aiSettings.analysisModel || 'gemini-3-pro-preview';
          
          let prompt = "";
          const graphContext = JSON.stringify({ nodes: data.entities, edges: data.entityRelationships, facts: data.contextFacts });

          if (mode === 'Isolation') {
              prompt = `Analyze this social graph for ISOLATION tactics. Who is cut off? Who are the gatekeepers? 
              Identify nodes with few connections or where connections pass through a single "controller" node.
              Data: ${graphContext}`;
          } else if (mode === 'Influence') {
              prompt = `Analyze this social graph for POWER DYNAMICS. Who has the most connections? Who bridges different groups (e.g. Family vs Work)?
              Data: ${graphContext}`;
          } else if (mode === 'Timeline') {
              prompt = `Analyze the DATES in the relationships vs the Entity roles. Are there inconsistencies? Do the phases (Love Bombing, etc) match the relationship dates?
              Data: ${graphContext}`;
          } else {
              prompt = `Perform a general forensic analysis of this network. What stands out? Missing links?
              Data: ${graphContext}`;
          }

          const response = await ai.models.generateContent({ model, contents: prompt });
          setAiAnalysisResult({
              title: `${mode} Analysis`,
              content: response.text || "No analysis returned."
          });
          setShowAiMenu(false);
      } catch (e) {
          alert("AI Analysis failed.");
      } finally {
          setIsAnalyzing(false);
      }
  };

  // --- EXPORT LOGIC ---
  const exportCypher = () => {
      let cypher = `// Nodes\n`;
      data.entities.forEach(e => {
          cypher += `MERGE (e:Entity {id: '${e.id}'}) SET e.name = "${e.name}", e.type = "${e.type}", e.impact = "${e.impactOnCase}";\n`;
      });
      cypher += `\n// Relationships\n`;
      data.entityRelationships.forEach(r => {
          let props = `type: '${r.type}'`;
          if (r.startDate) props += `, startDate: '${r.startDate}'`;
          if (r.endDate) props += `, endDate: '${r.endDate}'`;
          cypher += `MATCH (a:Entity {id: '${r.sourceId}'}), (b:Entity {id: '${r.targetId}'}) MERGE (a)-[:${r.label.replace(/\s+/g, '_').toUpperCase()} {${props}}]->(b);\n`;
      });
      const blob = new Blob([cypher], { type: 'text/plain' });
      const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = `Chronicle-Graph-Neo4j.cql`; a.click();
      setShowExportMenu(false);
  };

  const exportJSONLD = () => {
      const jsonLd = {
          "@context": "https://schema.org",
          "@graph": data.entities.map(e => ({
              "@id": `chronicle:${e.id}`,
              "@type": e.type === 'person' ? 'Person' : 'Thing',
              "name": e.name,
              "description": e.notes,
              "impact": e.impactOnCase,
              "relations": data.entityRelationships.filter(r => r.sourceId === e.id).map(r => ({
                  "@type": "Relationship", "label": r.label, "target": `chronicle:${r.targetId}`
              }))
          }))
      };
      const blob = new Blob([JSON.stringify(jsonLd, null, 2)], { type: 'application/ld+json' });
      const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = `Chronicle-Graph.jsonld`; a.click();
      setShowExportMenu(false);
  };

  const exportImage = () => {
      if (!svgRef.current) return;
      const source = new XMLSerializer().serializeToString(svgRef.current);
      const blob = new Blob([source], { type: 'image/svg+xml;charset=utf-8' });
      const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = `Chronicle-Graph-Chart.svg`; a.click();
      setShowExportMenu(false);
  };

  return (
    <div className="flex h-full relative" ref={wrapperRef}>
      
      {/* AI Analysis Modal */}
      {aiAnalysisResult && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm">
              <div className="bg-slate-900 rounded-3xl shadow-2xl max-w-2xl w-full max-h-[80vh] flex flex-col overflow-hidden border border-slate-800">
                  <div className="p-6 border-b border-slate-800 flex justify-between items-center bg-indigo-900/50 text-white">
                      <div className="flex items-center gap-3">
                          <BrainCircuit className="w-5 h-5 text-indigo-300" />
                          <h3 className="text-sm font-black uppercase tracking-widest">{aiAnalysisResult.title}</h3>
                      </div>
                      <button onClick={() => setAiAnalysisResult(null)} className="hover:bg-indigo-500/50 p-1 rounded-full"><X size={18} /></button>
                  </div>
                  <div className="p-8 overflow-y-auto bg-slate-900 prose prose-sm prose-invert max-w-none">
                      <div className="whitespace-pre-wrap text-slate-300 text-xs leading-relaxed font-medium">{aiAnalysisResult.content}</div>
                  </div>
              </div>
          </div>
      )}

      {/* Main Canvas */}
      <div className={`flex-1 bg-slate-950 relative overflow-hidden ${isLinking ? 'cursor-crosshair' : ''}`}>
        
        {isLinking && <div className="absolute top-4 left-1/2 -translate-x-1/2 z-20 bg-indigo-600 text-white px-4 py-2 rounded-full shadow-lg animate-pulse font-bold text-xs">Select target node...</div>}
        
        {/* Top LEFT Controls */}
        <div className="absolute top-4 left-4 z-10 flex flex-col gap-2">
            <div className="bg-slate-900/80 p-3 rounded-xl border border-slate-700 backdrop-blur-sm shadow-xl">
                <h3 className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2 flex items-center gap-2">
                    <Network size={12} /> Graph Builder
                </h3>
                <div className="flex gap-2">
                    <input className="bg-slate-950 border border-slate-700 rounded px-2 py-1 text-xs text-white w-24 outline-none" placeholder="Name" value={newEntityName} onChange={e => setNewEntityName(e.target.value)} />
                    <select className="bg-slate-950 border border-slate-700 rounded px-2 py-1 text-xs text-white outline-none" value={newEntityType} onChange={e => setNewEntityType(e.target.value as any)}>
                        <option value="person">Person</option>
                        <option value="workplace">Work</option>
                        <option value="location">Place</option>
                        <option value="school">School</option>
                    </select>
                    <button onClick={addEntity} className="bg-indigo-600 hover:bg-indigo-500 text-white p-1 rounded"><Plus size={14} /></button>
                </div>
            </div>
            
            <button onClick={() => setShowContextPanel(!showContextPanel)} className="bg-slate-900/80 p-3 rounded-xl border border-slate-700 text-slate-300 hover:text-white flex items-center gap-2 text-xs font-bold w-fit">
                <BookOpen size={14} /> Context & Phases
            </button>
        </div>

        {/* Top RIGHT Controls (Exports & AI) */}
        <div className="absolute top-4 right-4 z-10 flex gap-2">
            <div className="relative">
                <button onClick={() => setShowAiMenu(!showAiMenu)} className="bg-indigo-600 p-2 rounded-lg text-white hover:bg-indigo-500 shadow-lg flex items-center gap-2">
                    <BrainCircuit size={16} />
                    <span className="text-[10px] font-black uppercase hidden md:inline">Gemini Consult</span>
                </button>
                {showAiMenu && (
                     <div className="absolute right-0 top-12 w-64 bg-slate-900 border border-slate-700 rounded-xl shadow-2xl overflow-hidden backdrop-blur-md">
                        <div className="p-3 border-b border-slate-800 bg-slate-950/50"><h4 className="text-[10px] font-black uppercase text-indigo-400 tracking-widest">Select Mode</h4></div>
                        <button onClick={() => handleGraphAnalysis('Isolation')} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 border-b border-slate-800">Check for Isolation Tactics</button>
                        <button onClick={() => handleGraphAnalysis('Influence')} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 border-b border-slate-800">Map Influence & Power</button>
                        <button onClick={() => handleGraphAnalysis('Timeline')} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 border-b border-slate-800">Timeline Consistency Check</button>
                        <button onClick={() => handleGraphAnalysis('General')} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800">General Network Analysis</button>
                     </div>
                )}
            </div>

            <div className="relative">
                <button onClick={() => setShowExportMenu(!showExportMenu)} className="bg-slate-900/80 p-2 rounded-lg border border-slate-700 text-slate-300 hover:text-white shadow-lg backdrop-blur-sm flex items-center gap-2">
                    <Share2 size={16} />
                </button>
                {showExportMenu && (
                    <div className="absolute right-0 top-12 w-56 bg-slate-900 border border-slate-700 rounded-xl shadow-2xl overflow-hidden backdrop-blur-md">
                         <div className="p-3 border-b border-slate-800 bg-slate-950/50"><h4 className="text-[10px] font-black uppercase text-indigo-400 tracking-widest">Graph Exports</h4></div>
                        <button onClick={exportImage} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 flex items-center gap-3 border-b border-slate-800"><ImageIcon size={14} /> Visual Chart (SVG)</button>
                        <button onClick={exportCypher} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 flex items-center gap-3 border-b border-slate-800"><Database size={14} /> Neo4j (Cypher)</button>
                        <button onClick={exportJSONLD} className="w-full text-left px-4 py-3 text-xs text-slate-300 hover:bg-slate-800 flex items-center gap-3"><Database size={14} /> JSON-LD</button>
                    </div>
                )}
            </div>
        </div>

        <svg ref={svgRef} className="w-full h-full block"></svg>
      </div>

      {/* Context Panel (Slide over) */}
      {showContextPanel && (
          <div className="absolute left-0 top-0 bottom-0 w-80 bg-slate-900 border-r border-slate-800 p-6 z-20 overflow-y-auto shadow-2xl">
              <div className="flex justify-between items-center mb-6">
                  <h3 className="text-sm font-black text-white uppercase tracking-widest flex items-center gap-2"><BookOpen size={16} /> Context Layers</h3>
                  <button onClick={() => setShowContextPanel(false)}><X size={16} className="text-slate-500 hover:text-white" /></button>
              </div>
              
              <div className="space-y-6">
                  <div>
                      <h4 className="text-[10px] font-black text-slate-500 uppercase mb-2">Context Facts</h4>
                      {data.contextFacts.length === 0 && <p className="text-xs text-slate-600 italic">No facts recorded.</p>}
                      <div className="space-y-2">
                          {data.contextFacts.map(f => (
                              <div key={f.id} className="bg-slate-950 p-2 rounded border border-slate-800 text-xs text-slate-400">
                                  <span className="text-indigo-400 font-bold block mb-1">{f.category}</span>
                                  {f.content}
                              </div>
                          ))}
                      </div>
                  </div>

                  <div>
                      <h4 className="text-[10px] font-black text-slate-500 uppercase mb-2">Life Phases</h4>
                      {data.contextPhases.length === 0 && <p className="text-xs text-slate-600 italic">No phases recorded.</p>}
                      <div className="space-y-2">
                          {data.contextPhases.map(p => (
                              <div key={p.id} className="bg-slate-950 p-2 rounded border border-slate-800 text-xs text-slate-400">
                                  <div className="flex justify-between">
                                    <span className="text-amber-400 font-bold">{p.name}</span>
                                    <span className="text-[9px] text-slate-600">{p.startDate} - {p.endDate}</span>
                                  </div>
                                  <p className="mt-1 text-[9px] text-slate-500">{p.category}</p>
                              </div>
                          ))}
                      </div>
                  </div>
              </div>
          </div>
      )}

      {/* Node Detail Sidebar */}
      {isSidebarOpen && selectedNode && (
        <div className="w-80 bg-slate-900 border-l border-slate-800 p-6 overflow-y-auto absolute right-0 top-0 bottom-0 shadow-2xl z-30">
            <div className="flex justify-between items-start mb-6">
                <div>
                    <h2 className="text-lg font-black text-white">{selectedNode.name}</h2>
                    <span className="text-xs font-bold text-indigo-400 uppercase bg-indigo-950/30 px-2 py-1 rounded mt-1 inline-block">{selectedNode.type}</span>
                </div>
                <button onClick={() => setIsSidebarOpen(false)} className="text-slate-500 hover:text-white"><X size={18} /></button>
            </div>
            
            <div className="space-y-6">
                 {/* Basic Info */}
                 <div className="bg-slate-950 p-4 rounded-xl border border-slate-800 mb-6">
                    <label className="text-[10px] font-black text-slate-500 uppercase flex items-center gap-2 mb-3">
                        <Activity size={14} /> Significance Level
                    </label>
                    <div className="grid grid-cols-3 gap-2">
                        {(['low', 'medium', 'high'] as const).map((level) => (
                            <button
                                key={level}
                                onClick={() => updateEntity(selectedNode.id, { impactOnCase: level })}
                                className={`
                                    py-2 rounded-lg text-[10px] font-black uppercase transition-all
                                    ${selectedNode.impactOnCase === level
                                        ? (level === 'high' ? 'bg-red-600 text-white shadow-lg shadow-red-500/20 ring-1 ring-red-400' 
                                        : level === 'medium' ? 'bg-amber-600 text-white shadow-lg shadow-amber-500/20 ring-1 ring-amber-400' 
                                        : 'bg-emerald-600 text-white shadow-lg shadow-emerald-500/20 ring-1 ring-emerald-400')
                                        : 'bg-slate-900 text-slate-500 hover:bg-slate-800 border border-slate-800'
                                    }
                                `}
                            >
                                {level}
                            </button>
                        ))}
                    </div>
                    <p className="text-[9px] text-slate-500 mt-2 text-center italic">
                        {selectedNode.impactOnCase === 'high' ? 'Critical entity. Driver of conflict or support.' : 
                         selectedNode.impactOnCase === 'medium' ? 'Active participant in the timeline.' : 
                         'Background or contextual entity.'}
                    </p>
                 </div>

                 <div>
                    <label className="text-[10px] font-black text-slate-500 uppercase">Notes / Keys</label>
                    <textarea 
                        className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-slate-300 mt-1 min-h-[80px]"
                        value={selectedNode.notes}
                        onChange={(e) => updateEntity(selectedNode.id, { notes: e.target.value })}
                    />
                 </div>

                 {/* Document Attachments */}
                 <div className="bg-slate-950 p-3 rounded-lg border border-slate-800">
                     <label className="text-[10px] font-black text-indigo-400 uppercase flex items-center gap-2 mb-2">
                         <FileKey size={12} /> Evidence Keys / Docs
                     </label>
                     <div className="space-y-1">
                         <input type="text" placeholder="Add Document ID (e.g. DOC-001)" className="w-full bg-slate-900 border border-slate-700 rounded p-1.5 text-xs text-white mb-2" />
                         <p className="text-[9px] text-slate-500">Attach keys to link this entity to specific evidence files in the Vault.</p>
                     </div>
                 </div>

                 {/* Relationships List */}
                 <div>
                    <label className="text-[10px] font-black text-slate-500 uppercase">Connections</label>
                    <div className="space-y-2 mt-2">
                        {data.entityRelationships.filter(r => r.sourceId === selectedNode.id || r.targetId === selectedNode.id).map(r => {
                            const isSource = r.sourceId === selectedNode.id;
                            const other = data.entities.find(e => e.id === (isSource ? r.targetId : r.sourceId));
                            return (
                                <div key={r.id} className="bg-slate-950 p-2 rounded border border-slate-800 text-xs group">
                                    <div className="flex justify-between items-center">
                                        <div className="flex items-center gap-2">
                                            <span className="text-indigo-400 font-bold">{isSource ? '→' : '←'} {r.label}</span>
                                            <span className="text-slate-300 font-semibold">{other?.name}</span>
                                        </div>
                                        <button onClick={() => deleteRelationship(r.id)} className="text-slate-600 hover:text-red-400 opacity-0 group-hover:opacity-100"><Trash2 size={12} /></button>
                                    </div>
                                    {(r.startDate || r.endDate) && (
                                        <div className="text-[9px] text-slate-600 mt-1 pl-5">
                                            {r.startDate || '?'} to {r.endDate || 'Present'}
                                        </div>
                                    )}
                                </div>
                            )
                        })}
                        <button onClick={startLinking} className="w-full mt-2 bg-indigo-900/50 hover:bg-indigo-900 text-indigo-300 border border-indigo-500/30 py-2 rounded text-xs font-bold uppercase flex items-center justify-center gap-2">
                            <Network size={14} /> Add Connection
                        </button>
                    </div>
                </div>
                
                <div className="pt-4 border-t border-slate-800">
                    <button onClick={deleteSelectedEntity} className="flex items-center gap-2 text-red-400 hover:text-red-300 text-xs font-bold uppercase">
                        <Trash2 size={14} /> Delete Entity
                    </button>
                </div>
            </div>
        </div>
      )}
    </div>
  );
};

export default KnowledgeGraph;