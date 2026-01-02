
import React, { useState, useMemo } from 'react';
import { TimelineEvent, EventCategory } from '../types';
import { GoogleGenAI, Type } from "@google/genai";
import { MapPin, Users, Star, Globe, Search, AlertCircle, Fingerprint, Activity, User, Video, MessageSquare } from 'lucide-react';

interface TimelineProps {
  events: TimelineEvent[];
  onDelete: (id: string) => void;
  onUpdate: (event: TimelineEvent) => void;
  analysisModel?: string; // Optional passed model from App state
}

const Timeline: React.FC<TimelineProps> = ({ events, onDelete, onUpdate, analysisModel }) => {
  const [filter, setFilter] = useState<EventCategory | 'All'>('All');
  const [zoomLevel, setZoomLevel] = useState<'compact' | 'detailed'>('detailed');
  const [yearFilter, setYearFilter] = useState<string | 'All'>('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [tagFilter, setTagFilter] = useState<string | 'All'>('All');
  
  const [editingId, setEditingId] = useState<string | null>(null);
  const [refinePrompt, setRefinePrompt] = useState('');
  
  // AI Analysis State
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [useDeepResearch, setUseDeepResearch] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<{title: string, content: string, sources?: any[]} | null>(null);
  
  // Tag & Link input state
  const [newTag, setNewTag] = useState('');
  const [newLink, setNewLink] = useState('');

  // Use the passed analysis model or default to Pro
  const activeModel = analysisModel || 'gemini-3-pro-preview';

  const sortedEvents = useMemo(() => {
    return [...events].sort((a, b) => {
      const dateA = new Date(a.date).getTime() || 0;
      const dateB = new Date(b.date).getTime() || 0;
      return dateA - dateB;
    });
  }, [events]);

  const years = useMemo(() => {
    const y = new Set<string>();
    events.forEach(e => {
      const year = new Date(e.date).getFullYear();
      if (!isNaN(year)) y.add(year.toString());
    });
    return Array.from(y).sort();
  }, [events]);

  const availableTags = useMemo(() => {
    const tags = new Set<string>();
    events.forEach(e => e.evidenceTags.forEach(t => tags.add(t.name)));
    return Array.from(tags).sort();
  }, [events]);

  const filteredEvents = useMemo(() => {
    return sortedEvents.filter(e => {
      const catMatch = filter === 'All' || e.category === filter;
      
      const year = new Date(e.date).getFullYear().toString();
      const yearMatch = yearFilter === 'All' || year === yearFilter;
      
      const tagMatch = tagFilter === 'All' || e.evidenceTags.some(t => t.name === tagFilter);
      
      const searchLower = searchQuery.toLowerCase();
      const searchMatch = !searchQuery || 
        e.description.toLowerCase().includes(searchLower) ||
        (e.location && e.location.toLowerCase().includes(searchLower)) ||
        e.courtExplanation?.toLowerCase().includes(searchLower) ||
        e.manipulationPattern?.toLowerCase().includes(searchLower) ||
        e.involvedEntities.some(ent => ent.toLowerCase().includes(searchLower));

      return catMatch && yearMatch && tagMatch && searchMatch;
    });
  }, [sortedEvents, filter, yearFilter, tagFilter, searchQuery]);

  // --- MACRO ANALYSIS (Full Timeline) ---
  const handleMacroAnalysis = async () => {
    if (filteredEvents.length === 0) return;
    setIsAnalyzing(true);
    try {
      const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
      
      const tools = useDeepResearch ? [{ googleSearch: {} }] : [];
      const researchContext = useDeepResearch ? "Perform deep research using Google Search to verify dates, historical context, weather conditions, or public news events mentioned." : "";

      const response = await ai.models.generateContent({
        model: activeModel,
        contents: `Act as a Forensic Biographer and Strategist. Analyze this timeline of events.
        
        ${researchContext}

        DATASET:
        ${JSON.stringify(filteredEvents.map(e => ({ 
            date: e.date, 
            desc: e.description, 
            loc: e.location, 
            wit: e.witnesses, 
            pattern: e.manipulationPattern,
            forensics: e.forensicAnalysis // Include forensic data in context
        })))}

        STRICT RULES FOR LARGE DOCUMENT HANDLING:
        1. **NO LOSS OF NUANCE:** Do NOT summarize away the emotional or factual details. We need specific patterns.
        2. **VERBATIM PRESERVATION:** If specific quotes or forensic details (deception scores) are present, cite them exactly.
        3. **COMPREHENSIVE READ:** Read every single event provided before forming conclusions.
        4. **PATTERN RECOGNITION:** Connect the dots between early "Childhood Memories" and current "Triggered Reactions".
        
        PROVIDE A REPORT:
        1. **The Biographical Arc**: What is the life story told here?
        2. **Credibility Assessment**: How consistent are the locations, witness accounts, and forensic data?
        3. **Pattern Recognition**: Identify cyclic behaviors.
        4. **Strategic Summary**: How does this history support the user?
        
        Format as clear Markdown.`,
        config: { 
            tools,
            thinkingConfig: { thinkingBudget: 2048 } // Enable Deep Thinking for Nuance
        }
      });

      // Extract grounding metadata if available
      const sources = response.candidates?.[0]?.groundingMetadata?.groundingChunks || [];

      setAnalysisResult({
        title: `Macro-Analysis ${useDeepResearch ? '(Deep Research)' : ''}`,
        content: response.text || "Analysis failed.",
        sources
      });
    } catch (err) {
      alert("Macro analysis failed.");
    } finally {
      setIsAnalyzing(false);
    }
  };

  // --- MICRO ANALYSIS (Single Event) ---
  const handleMicroAnalysis = async (event: TimelineEvent) => {
    setIsAnalyzing(true);
    try {
        const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
        
        const tools = useDeepResearch ? [{ googleSearch: {} }] : [];
        const researchContext = useDeepResearch ? "Use Google Search to verify the location, date, historical weather, or any public context surrounding this event." : "";

        const response = await ai.models.generateContent({
            model: activeModel,
            contents: `Analyze this SINGLE EVENT for a historical/forensic file.
            
            ${researchContext}

            EVENT: ${JSON.stringify(event)}
            
            TASKS:
            1. **Narrative Consistency**: Does this event fit the broader pattern?
            2. **Forensic Validity**: Analyze any attached forensic data (deception scores, sentiment) for congruency with the description.
            3. **Optics**: How will this look to an objective third party?
            4. **Verification**: Are the location and witnesses specific enough?
            
            Keep it concise but detailed.`,
            config: { 
                tools,
                thinkingConfig: { thinkingBudget: 1024 } // Enable thinking for specific event analysis
            }
        });
        
        const result = response.text || "Analysis failed.";
        const sources = response.candidates?.[0]?.groundingMetadata?.groundingChunks || [];

        // Save to event object so it persists
        onUpdate({ ...event, aiAnalysis: result });
        
        setAnalysisResult({
            title: `Micro-Analysis: ${event.date}`,
            content: result,
            sources
        });
    } catch (err) {
        alert("Micro analysis failed.");
    } finally {
        setIsAnalyzing(false);
    }
  };

  const handleRefine = async (event: TimelineEvent) => {
    if (!refinePrompt.trim()) return;
    setIsAnalyzing(true);
    try {
      const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
      const response = await ai.models.generateContent({
        model: activeModel,
        contents: `Review this forensic entry: ${JSON.stringify(event)}. Feedback: "${refinePrompt}". Update derived fields (courtExplanation, pattern, context) for accuracy. Maintain all existing nuance.`,
        config: {
          responseMimeType: "application/json",
          responseSchema: {
            type: Type.OBJECT,
            properties: {
              description: { type: Type.STRING },
              courtExplanation: { type: Type.STRING },
              manipulationPattern: { type: Type.STRING },
              reactionContext: { type: Type.STRING },
              historicalTraumaLink: { type: Type.STRING },
              evidenceStrength: { type: Type.STRING, enum: ['Weak', 'Moderate', 'Strong', 'Conclusive'] },
            },
            required: ['description', 'courtExplanation']
          }
        }
      });
      const updatedFields = JSON.parse(response.text);
      onUpdate({ ...event, ...updatedFields });
      setEditingId(null);
      setRefinePrompt('');
    } catch (err) {
      alert("Analysis failed.");
    } finally { setIsAnalyzing(false); }
  };

  const toggleSignificance = (event: TimelineEvent, e: React.MouseEvent) => {
      e.stopPropagation();
      onUpdate({ ...event, isSignificant: !event.isSignificant });
  };

  const getEventColor = (category: EventCategory) => {
      switch (category) {
          case 'Childhood Memory': return 'bg-sky-400 border-sky-900 text-sky-400';
          case 'Career/Professional': return 'bg-slate-400 border-slate-900 text-slate-400';
          case 'Strength/Resilience': return 'bg-amber-400 border-amber-900 text-amber-400';
          case 'Historical Trauma': return 'bg-rose-500 border-rose-900 text-rose-400';
          case 'Current Case Incident': return 'bg-indigo-500 border-indigo-900 text-indigo-400';
          default: return 'bg-emerald-500 border-emerald-900 text-emerald-400';
      }
  };

  const getEventLabelStyle = (category: EventCategory) => {
      switch (category) {
          case 'Childhood Memory': return 'bg-sky-950 text-sky-400 border border-sky-900';
          case 'Career/Professional': return 'bg-slate-800 text-slate-300 border border-slate-600';
          case 'Strength/Resilience': return 'bg-amber-950 text-amber-400 border border-amber-900';
          case 'Historical Trauma': return 'bg-rose-950 text-rose-400 border border-rose-900';
          case 'Current Case Incident': return 'bg-indigo-950 text-indigo-400 border border-indigo-900';
          default: return 'bg-emerald-950 text-emerald-400 border border-emerald-900';
      }
  };
  
  const getOriginIcon = (origin: string) => {
      switch (origin) {
          case 'Chronicle': return <User size={10} className="text-blue-400" />;
          case 'Sentinel': return <Video size={10} className="text-red-400" />;
          case 'Anamnesis': return <MessageSquare size={10} className="text-purple-400" />;
          default: return <Activity size={10} className="text-slate-400" />;
      }
  };

  // --- RENDER ---

  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-slate-600 h-full">
        <AlertCircle size={48} className="mb-4 opacity-20" />
        <p className="text-center font-bold text-[10px] uppercase tracking-widest max-w-[200px]">Timeline Empty</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full bg-slate-950 relative">
      
      {/* Analysis Modal */}
      {analysisResult && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm">
              <div className="bg-slate-900 rounded-3xl shadow-2xl max-w-2xl w-full max-h-[80vh] flex flex-col overflow-hidden border border-slate-800">
                  <div className="p-6 border-b border-slate-800 flex justify-between items-center bg-indigo-900/50 text-white">
                      <div className="flex items-center gap-3">
                          <Globe className="w-5 h-5 text-indigo-300" />
                          <h3 className="text-sm font-black uppercase tracking-widest">{analysisResult.title}</h3>
                      </div>
                      <button onClick={() => setAnalysisResult(null)} className="hover:bg-indigo-500/50 p-1 rounded-full transition-colors">
                          <div className="w-5 h-5 flex items-center justify-center">✕</div>
                      </button>
                  </div>
                  <div className="p-8 overflow-y-auto bg-slate-900">
                      <div className="prose prose-sm prose-invert max-w-none mb-6">
                          <div className="whitespace-pre-wrap text-slate-300 text-xs leading-relaxed font-medium">
                            {analysisResult.content}
                          </div>
                      </div>
                      
                      {/* Sources Section */}
                      {analysisResult.sources && analysisResult.sources.length > 0 && (
                          <div className="mt-6 pt-6 border-t border-slate-800">
                              <h4 className="text-[10px] font-black text-slate-500 uppercase tracking-widest mb-3">Verified Sources</h4>
                              <div className="space-y-2">
                                  {analysisResult.sources.map((source, idx) => (
                                      <div key={idx} className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-xs">
                                          {source.web?.uri ? (
                                              <a href={source.web.uri} target="_blank" rel="noopener noreferrer" className="text-indigo-400 hover:underline font-medium block truncate">
                                                  {source.web.title || source.web.uri}
                                              </a>
                                          ) : (
                                              <span className="text-slate-400">{source.text || "Source info unavailable"}</span>
                                          )}
                                      </div>
                                  ))}
                              </div>
                          </div>
                      )}
                  </div>
                  <div className="p-4 bg-slate-950 border-t border-slate-800 flex justify-end">
                      <button onClick={() => setAnalysisResult(null)} className="px-6 py-2 bg-indigo-600 text-white text-[10px] font-black uppercase rounded-xl hover:bg-indigo-500 transition-colors">
                          Close Report
                      </button>
                  </div>
              </div>
          </div>
      )}

      {/* Workbench Header */}
      <div className="bg-slate-900 border-b border-slate-800 p-4 sticky top-0 z-20 space-y-4 shadow-sm">
        <div className="flex items-center justify-between max-w-4xl mx-auto w-full">
          <h3 className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Timeline Workbench</h3>
          <div className="flex items-center gap-2">
             
             {/* Deep Research Toggle */}
             <button 
                onClick={() => setUseDeepResearch(!useDeepResearch)}
                className={`flex items-center gap-2 px-3 py-1.5 border rounded-lg text-[9px] font-black uppercase tracking-widest transition-all ${
                    useDeepResearch 
                    ? 'bg-emerald-900/30 border-emerald-500 text-emerald-400' 
                    : 'bg-slate-800 border-slate-700 text-slate-500 hover:text-slate-300'
                }`}
                title="Enable Google Search Grounding for Verification"
             >
                <Globe size={12} />
                {useDeepResearch ? 'Deep Research ON' : 'Deep Research OFF'}
             </button>

             <button 
                onClick={handleMacroAnalysis}
                disabled={isAnalyzing}
                className="flex items-center gap-2 px-3 py-1.5 bg-indigo-600 text-white text-[9px] font-black uppercase tracking-widest rounded-lg hover:bg-indigo-500 disabled:opacity-50 transition-all shadow-lg shadow-indigo-500/20"
             >
                {isAnalyzing ? (
                     <div className="animate-spin h-3 w-3 border-2 border-white/30 border-t-white rounded-full"></div>
                ) : (
                     <Search size={12} />
                )}
                Analyze
             </button>
             
             <div className="flex bg-slate-800 p-0.5 rounded-lg border border-slate-700 ml-2">
                <button onClick={() => setZoomLevel('compact')} className={`px-3 py-1 text-[9px] font-black rounded-md transition-all ${zoomLevel === 'compact' ? 'bg-slate-700 shadow-sm text-indigo-400' : 'text-slate-500'}`}>COMPACT</button>
                <button onClick={() => setZoomLevel('detailed')} className={`px-3 py-1 text-[9px] font-black rounded-md transition-all ${zoomLevel === 'detailed' ? 'bg-slate-700 shadow-sm text-indigo-400' : 'text-slate-500'}`}>DETAILED</button>
             </div>
          </div>
        </div>
        
        {/* Filters & Search */}
        <div className="flex flex-col md:flex-row gap-2 max-w-4xl mx-auto w-full">
          {/* Text Search */}
          <div className="relative flex-1">
             <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-3 w-3 text-slate-500" />
             </div>
             <input 
                type="text" 
                placeholder="Search description, location, witnesses..." 
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-3 py-2 text-[10px] font-semibold bg-slate-800 border border-slate-700 text-slate-200 rounded-lg outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500/20 transition-all placeholder:text-slate-600"
             />
          </div>

          <div className="flex gap-2 overflow-x-auto pb-1 md:pb-0">
            <select value={filter} onChange={(e) => setFilter(e.target.value as any)} className="flex-1 min-w-[120px] text-[10px] font-black uppercase bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 outline-none text-slate-300">
                <option value="All">All Categories</option>
                <option value="Childhood Memory">Childhood</option>
                <option value="Career/Professional">Career</option>
                <option value="Strength/Resilience">Resilience</option>
                <option value="Historical Trauma">Historical Trauma</option>
                <option value="Current Case Incident">Current Incident</option>
            </select>
            
            <select value={yearFilter} onChange={(e) => setYearFilter(e.target.value)} className="w-20 text-[10px] font-black uppercase bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 outline-none text-slate-300">
                <option value="All">Year</option>
                {years.map(y => <option key={y} value={y}>{y}</option>)}
            </select>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 lg:p-8">
        <div className="max-w-4xl mx-auto relative">
          <div className="absolute left-6 lg:left-8 top-0 bottom-0 w-px bg-slate-800"></div>

          <div className="space-y-6">
            {filteredEvents.map((event) => {
              const isEditing = editingId === event.id;
              const colorClass = getEventColor(event.category);
              const labelClass = getEventLabelStyle(event.category);
              const hasForensics = !!event.forensicAnalysis;
              const originIcon = getOriginIcon(event.origin || 'Chronicle');

              return (
                <div key={event.id} className="relative pl-8 lg:pl-12 group">
                  <div className={`absolute left-[21px] lg:left-[29px] top-1 w-2 h-2 rounded-full border-2 border-slate-950 shadow-sm z-10 ${colorClass.split(' ')[0]}`}></div>

                  <div className={`bg-slate-900 rounded-2xl border transition-all ${event.isSignificant ? 'border-amber-500/50 shadow-amber-900/20' : 'border-slate-800 shadow-sm'} ${zoomLevel === 'compact' ? 'py-3 px-5' : 'p-6'} ${isEditing ? 'ring-2 ring-indigo-500 ring-offset-2 ring-offset-slate-950' : 'hover:shadow-md hover:border-slate-700'}`}>
                    
                    {/* Header Row */}
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-3">
                        {isEditing ? (
                          <input type="text" value={event.date} onChange={(e) => onUpdate({ ...event, date: e.target.value })} className="text-xs font-black text-indigo-400 bg-indigo-950 border border-indigo-900 rounded px-2 py-1 outline-none w-32" />
                        ) : (
                          <span className="text-[10px] font-black text-slate-500 uppercase tabular-nums">{event.date}</span>
                        )}
                        <span className={`text-[9px] font-black px-2 py-0.5 rounded uppercase tracking-wide ${labelClass}`}>
                          {event.category}
                        </span>
                        {/* Origin Badge */}
                        <span className="flex items-center gap-1 text-[8px] font-black uppercase text-slate-500 bg-slate-950 px-2 py-0.5 rounded border border-slate-800">
                             {originIcon} {event.origin || 'Unknown'}
                        </span>

                        {/* Forensic Badge */}
                        {hasForensics && (
                            <span className="text-[9px] font-black px-2 py-0.5 rounded uppercase tracking-wide bg-red-950 text-red-400 border border-red-900 flex items-center gap-1">
                                <Fingerprint size={10} /> Forensic Data
                            </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {/* Significance Star */}
                        <button 
                            onClick={(e) => toggleSignificance(event, e)} 
                            className={`p-1 transition-colors ${event.isSignificant ? 'text-amber-400' : 'text-slate-700 hover:text-amber-400'}`}
                            title="Toggle Significance (Flag)"
                        >
                            <Star size={16} fill={event.isSignificant ? "currentColor" : "none"} />
                        </button>
                        
                        {!isEditing && (
                           <button onClick={() => handleMicroAnalysis(event)} className="p-1.5 text-indigo-400 hover:text-indigo-300 hover:bg-indigo-950 rounded-lg transition-colors flex items-center gap-1" title="Deep Micro-Analysis">
                              <Search size={14} />
                              <span className="text-[8px] font-bold uppercase hidden md:inline">Verify</span>
                           </button>
                        )}
                        <button onClick={() => onDelete(event.id)} className="p-1 text-slate-600 hover:text-red-400 transition-colors" title="Delete Event">
                           <div className="w-4 h-4 flex items-center justify-center">✕</div>
                        </button>
                        <button onClick={() => setEditingId(isEditing ? null : event.id)} className={`p-1 transition-colors ${isEditing ? 'text-indigo-400' : 'text-slate-600 hover:text-indigo-400'}`} title="Edit Event">
                           <div className="w-4 h-4 border-2 border-current rounded-sm"></div>
                        </button>
                      </div>
                    </div>

                    {/* Content Body */}
                    <div className="space-y-4">
                      {isEditing ? (
                        <div className="space-y-4">
                           <textarea 
                              value={event.description} 
                              onChange={(e) => onUpdate({ ...event, description: e.target.value })} 
                              className="w-full bg-slate-950 border border-indigo-900/50 rounded-xl p-3 text-xs text-slate-300 focus:border-indigo-500 outline-none h-24 resize-none leading-relaxed" 
                              placeholder="Event description..."
                           />
                           
                           {/* Quick Refine with AI */}
                           <div className="flex gap-2">
                              <input 
                                type="text" 
                                value={refinePrompt} 
                                onChange={(e) => setRefinePrompt(e.target.value)}
                                placeholder="Refine with AI (e.g., 'Make it more objective', 'Emphasize the trigger')"
                                className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-[10px] text-slate-300 outline-none focus:border-indigo-500"
                              />
                              <button onClick={() => handleRefine(event)} disabled={!refinePrompt || isAnalyzing} className="px-3 py-1 bg-indigo-600 text-white text-[9px] font-bold rounded-lg uppercase hover:bg-indigo-500 disabled:opacity-50">
                                {isAnalyzing ? '...' : 'Refine'}
                              </button>
                           </div>

                           <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                              <div>
                                 <label className="text-[9px] font-black text-slate-500 uppercase block mb-1">Location</label>
                                 <input type="text" value={event.location || ''} onChange={(e) => onUpdate({ ...event, location: e.target.value })} className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-300 outline-none focus:border-indigo-500" placeholder="Where?" />
                              </div>
                              <div>
                                 <label className="text-[9px] font-black text-slate-500 uppercase block mb-1">Witnesses (Comma separated)</label>
                                 <input 
                                    type="text" 
                                    value={event.witnesses?.join(', ') || ''} 
                                    onChange={(e) => onUpdate({ ...event, witnesses: e.target.value.split(',').map(s => s.trim()) })} 
                                    className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-300 outline-none focus:border-indigo-500" 
                                    placeholder="Who saw it?"
                                />
                              </div>
                           </div>
                           
                           <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                               <div>
                                    <label className="text-[9px] font-black text-slate-500 uppercase block mb-1">Weaponization</label>
                                    <input type="text" placeholder="How was this used against you?" value={event.weaponization || ''} onChange={(e) => onUpdate({ ...event, weaponization: e.target.value })} className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-300 outline-none focus:border-indigo-500" />
                               </div>
                               <div>
                                 <label className="text-[9px] font-black text-slate-500 uppercase block mb-1">Evidence Strength</label>
                                 <select value={event.evidenceStrength} onChange={(e) => onUpdate({ ...event, evidenceStrength: e.target.value as any })} className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2 text-xs text-slate-300 outline-none focus:border-indigo-500">
                                    <option value="Weak">Weak</option>
                                    <option value="Moderate">Moderate</option>
                                    <option value="Strong">Strong</option>
                                    <option value="Conclusive">Conclusive</option>
                                 </select>
                              </div>
                           </div>
                        </div>
                      ) : (
                        <>
                           {zoomLevel === 'detailed' && (
                              <p className="text-sm font-medium text-slate-300 leading-relaxed whitespace-pre-wrap">{event.description}</p>
                           )}
                           
                           {/* Location & Witnesses Row */}
                           {(event.location || (event.witnesses && event.witnesses.length > 0)) && (
                               <div className="flex flex-wrap items-center gap-4 mt-2 text-xs text-slate-400 bg-slate-950/50 p-2 rounded-lg border border-slate-800/50">
                                   {event.location && (
                                       <div className="flex items-center gap-1.5 text-emerald-400/80">
                                           <MapPin size={12} />
                                           <span className="font-semibold">{event.location}</span>
                                       </div>
                                   )}
                                   {event.witnesses && event.witnesses.length > 0 && (
                                       <div className="flex items-center gap-1.5 text-blue-400/80">
                                           <Users size={12} />
                                           <span className="font-semibold">{event.witnesses.join(', ')}</span>
                                       </div>
                                   )}
                               </div>
                           )}

                           {/* Forensic Data Display (Deception/Sentiment) */}
                           {hasForensics && event.forensicAnalysis && (
                               <div className="grid grid-cols-2 gap-2 mt-2">
                                   <div className="bg-red-950/20 border border-red-900/50 p-2 rounded-lg">
                                       <h5 className="text-[9px] font-black text-red-500 uppercase tracking-widest flex items-center gap-1">
                                            <Activity size={10} /> Deception Analysis
                                       </h5>
                                       <div className="flex items-center gap-2 mt-1">
                                            <div className="flex-1 h-1.5 bg-slate-800 rounded-full overflow-hidden">
                                                <div className="h-full bg-red-500" style={{ width: `${event.forensicAnalysis.deceptionScore || 0}%` }}></div>
                                            </div>
                                            <span className="text-[9px] font-bold text-red-400">{event.forensicAnalysis.deceptionScore || 0}%</span>
                                       </div>
                                   </div>
                                   <div className="bg-indigo-950/20 border border-indigo-900/50 p-2 rounded-lg">
                                       <h5 className="text-[9px] font-black text-indigo-500 uppercase tracking-widest flex items-center gap-1">
                                            <Fingerprint size={10} /> Sentiment
                                       </h5>
                                       <span className="text-xs text-indigo-300 font-bold capitalize mt-1 block">
                                            {event.forensicAnalysis.sentiment || 'Unknown'}
                                       </span>
                                   </div>
                                   {event.forensicAnalysis.transcriptExcerpt && (
                                       <div className="col-span-2 bg-slate-950 p-2 rounded border border-slate-800">
                                            <p className="text-[10px] text-slate-400 italic">"{event.forensicAnalysis.transcriptExcerpt}"</p>
                                       </div>
                                   )}
                               </div>
                           )}

                           {/* Context Chips */}
                           {(event.manipulationPattern || event.vulnerability || event.weaponization) && (
                              <div className="flex flex-wrap gap-2 mt-2">
                                 {event.manipulationPattern && <span className="text-[9px] font-bold text-amber-500 bg-amber-950/30 px-2 py-1 rounded border border-amber-500/20 uppercase tracking-wide">Pattern: {event.manipulationPattern}</span>}
                                 {event.vulnerability && <span className="text-[9px] font-bold text-rose-400 bg-rose-950/30 px-2 py-1 rounded border border-rose-500/20 uppercase tracking-wide">Vulnerability: {event.vulnerability}</span>}
                                 {event.weaponization && <span className="text-[9px] font-bold text-red-400 bg-red-950/30 px-2 py-1 rounded border border-red-500/20 uppercase tracking-wide">Weaponized: {event.weaponization}</span>}
                                 {event.evidenceStrength === 'Conclusive' && <span className="text-[9px] font-bold text-emerald-400 bg-emerald-950/30 px-2 py-1 rounded border border-emerald-500/20 uppercase tracking-wide">Conclusive Proof</span>}
                              </div>
                           )}

                           {/* AI Analysis Snippet */}
                           {event.aiAnalysis && zoomLevel === 'detailed' && (
                               <div className="mt-4 p-3 bg-indigo-950/20 border-l-2 border-indigo-500 rounded-r-lg">
                                   <p className="text-[10px] text-indigo-300 italic leading-relaxed line-clamp-3">"{event.aiAnalysis.substring(0, 150)}..."</p>
                               </div>
                           )}
                        </>
                      )}
                    </div>

                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Timeline;