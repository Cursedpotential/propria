
import React, { useState, useCallback, useEffect, useRef } from 'react';
import VoiceSession from './components/VoiceSession';
import Timeline from './components/Timeline';
import KnowledgeGraph from './components/KnowledgeGraph';
import Reports from './components/Reports';
import ContextGrid from './components/ContextGrid'; // New Import
import { TimelineEvent, Entity, AppState, ContextFact, ContextPhase, EntityRelationship, GlobalNote } from './types';
import { supabase } from './utils/supabase';
import { GoogleGenAI } from "@google/genai";
import { LayoutDashboard, Mic, Activity, Network, FileText, Database, Settings, LogOut, Cloud, Wifi, WifiOff, Server, Calendar, RefreshCw } from 'lucide-react';

const STORAGE_KEY = 'chronicle_case_data_v16'; // Incremented Version

// Initial Entities based on user context
const PRELOADED_ENTITIES: Entity[] = [
  { id: 'e2', name: 'Rita', type: 'person', relationship: 'Mother', impactOnCase: 'high', notes: 'Primary source of historical trauma.', frequency: 'High', origin: 'Chronicle' },
  { id: 'e5', name: 'Rick', type: 'person', relationship: 'Stepfather', impactOnCase: 'high', notes: 'Secondary trauma source.', frequency: 'Low', origin: 'Chronicle' },
  { id: 'e1', name: 'Jessica (Jessie)', type: 'person', relationship: 'Ex-Wife', impactOnCase: 'medium', notes: 'Opposing party.', frequency: 'High', origin: 'Chronicle' },
  { id: 'e3', name: 'Suheil', type: 'person', relationship: 'Father', impactOnCase: 'low', notes: '', frequency: 'Low', origin: 'Chronicle' },
  { id: 'e4', name: 'Traci', type: 'person', relationship: 'Stepmother', impactOnCase: 'low', notes: '', frequency: 'Low', origin: 'Chronicle' },
  { id: 'e6', name: 'Older Children', type: 'person', relationship: 'Children', impactOnCase: 'medium', notes: '', frequency: 'Medium', origin: 'Chronicle' },
];

const PRELOADED_RELATIONSHIPS: EntityRelationship[] = [
  { id: 'r1', sourceId: 'e2', targetId: 'e5', label: 'Married', type: 'romantic', origin: 'Chronicle' },
  { id: 'r2', sourceId: 'e2', targetId: 'e3', label: 'Divorced', type: 'conflict', origin: 'Chronicle' },
];

const App: React.FC = () => {
  const [state, setState] = useState<AppState>({
    events: [],
    entities: PRELOADED_ENTITIES,
    entityRelationships: PRELOADED_RELATIONSHIPS,
    contextFacts: [],
    contextPhases: [], // Renamed from relationshipPhases
    globalNotes: [],
    isSessionActive: false,
    caseContext: '',
    currentTab: 'recall',
    transcripts: [],
    aiSettings: {
      focus: 'Fact Extraction',
      strictness: 'Strict/Literal',
      liveModel: 'gemini-2.5-flash-native-audio-preview-09-2025',
      analysisModel: 'gemini-3-pro-preview'
    },
    connectionSettings: {
      weaviateUrl: '',
      weaviateApiKey: '',
      neo4jUrl: '',
      neo4jUser: '',
      neo4jPassword: ''
    }
  });
  
  const [sessionAnalysis, setSessionAnalysis] = useState<string | null>(null);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  
  // Cloud & System State
  const [user, setUser] = useState<any>(null);
  const [authEmail, setAuthEmail] = useState('');
  const [authPassword, setAuthPassword] = useState('');
  const [authMode, setAuthMode] = useState<'login' | 'signup'>('login');
  const [cloudStatus, setCloudStatus] = useState<'disconnected' | 'synced' | 'saving' | 'error' | 'updating'>('disconnected');
  const [cloudMessage, setCloudMessage] = useState('');
  const [connectionTestStatus, setConnectionTestStatus] = useState<'idle' | 'testing' | 'success' | 'fail'>('idle');
  const saveTimeoutRef = useRef<any>(null);
  const isRemoteUpdateRef = useRef(false);

  // New Note State
  const [newNoteTitle, setNewNoteTitle] = useState('');
  const [newNoteContent, setNewNoteContent] = useState('');
  const [newNoteCategory, setNewNoteCategory] = useState<GlobalNote['category']>('Observation');

  // Initialize Auth
  useEffect(() => {
    if (!supabase) return;
    
    supabase.auth.getSession().then(({ data: { session } }) => {
      setUser(session?.user ?? null);
      if (session?.user) {
        setCloudStatus('synced');
        loadFromCloud(session.user.id);
      }
    });

    const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
      setUser(session?.user ?? null);
      if (session?.user) {
        setCloudStatus('synced');
        loadFromCloud(session.user.id);
      } else {
        setCloudStatus('disconnected');
      }
    });

    return () => subscription.unsubscribe();
  }, []);

  // --- REALTIME SUBSCRIPTION FOR EXTERNAL APPS (The Trinity Link) ---
  useEffect(() => {
    if (!user || !supabase) return;

    // Listen for updates to the database that happen OUTSIDE this window (e.g. App 2 or App 3)
    const channel = supabase
        .channel('chronicle_db_changes')
        .on(
            'postgres_changes',
            {
                event: 'UPDATE',
                schema: 'public',
                table: 'chronicle_cases',
                filter: `user_id=eq.${user.id}`
            },
            (payload) => {
                // Determine if this update is from US (local save) or THEM (external app)
                // Since we don't have a sophisticated client ID in the DB yet, we assume
                // any incoming UPDATE event while we are NOT actively saving *might* be external.
                // For robustness, we will fetch the new data and merge/update state.
                
                // Note: We use a ref flag to prevent the incoming update from triggering an immediate auto-save loop
                isRemoteUpdateRef.current = true;
                
                console.log("Remote update detected from Ecosystem:", payload);
                setCloudStatus('updating');
                
                const remoteData = payload.new.case_data;
                if (remoteData) {
                    setState(prev => ({
                        ...prev,
                        ...remoteData,
                        // Preserve local UI state that isn't saved
                        isSessionActive: prev.isSessionActive,
                        currentTab: prev.currentTab
                    }));
                    setTimeout(() => setCloudStatus('synced'), 1000);
                }
                
                setTimeout(() => { isRemoteUpdateRef.current = false; }, 2000);
            }
        )
        .subscribe();

    return () => { supabase.removeChannel(channel); }
  }, [user]);

  // Initial Local Load
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        
        let safeTab = parsed.currentTab;
        if (safeTab === 'investigation') safeTab = 'recall';

        // Migration Logic: Move old relationshipPhases to contextPhases if exists
        const phases = parsed.contextPhases || parsed.relationshipPhases || [];
        
        // Ensure categories exist on migrated phases
        const robustPhases = phases.map((p: any) => ({
             ...p,
             category: p.category || 'Relationship',
             emotionalBaseline: p.emotionalBaseline || 'Unknown'
        }));

        setState(prev => ({
          ...prev,
          ...parsed,
          entities: parsed.entities && parsed.entities.length > 0 ? parsed.entities : PRELOADED_ENTITIES,
          entityRelationships: parsed.entityRelationships || PRELOADED_RELATIONSHIPS,
          globalNotes: parsed.globalNotes || [],
          contextPhases: robustPhases,
          isSessionActive: false,
          currentTab: safeTab || 'recall',
          aiSettings: { ...prev.aiSettings, ...(parsed.aiSettings || {}) },
          connectionSettings: { ...prev.connectionSettings, ...(parsed.connectionSettings || {}) }
        }));
      } catch (e) {
        console.error("Failed to load saved case data", e);
      }
    }
  }, []);

  // Save Effect
  useEffect(() => {
    // If we just received a remote update, DO NOT save immediately, or we might create a loop or race condition.
    if (isRemoteUpdateRef.current) return;

    const payload = { 
      events: state.events, 
      entities: state.entities, 
      entityRelationships: state.entityRelationships,
      contextFacts: state.contextFacts,
      contextPhases: state.contextPhases,
      globalNotes: state.globalNotes,
      caseContext: state.caseContext,
      aiSettings: state.aiSettings,
      connectionSettings: state.connectionSettings,
      currentTab: state.currentTab
    };
    
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));

    if (user && supabase) {
      setCloudStatus('saving');
      if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
      
      saveTimeoutRef.current = setTimeout(async () => {
        try {
          const { error } = await supabase
            .from('chronicle_cases')
            .upsert({ 
              user_id: user.id, 
              case_data: payload,
              updated_at: new Date().toISOString()
            });
          
          if (error) throw error;
          setCloudStatus('synced');
        } catch (err: any) {
          console.error("Cloud save failed:", err);
          setCloudStatus('error');
          setCloudMessage(err.message || "Save failed");
        }
      }, 2000);
    }
  }, [state, user]);

  const loadFromCloud = async (userId: string) => {
    if (!supabase) return;
    try {
      const { data, error } = await supabase
        .from('chronicle_cases')
        .select('case_data')
        .eq('user_id', userId)
        .single();
      
      if (error && error.code !== 'PGRST116') throw error;
      if (data?.case_data) {
         let safeTab = data.case_data.currentTab;
         if (safeTab === 'investigation') safeTab = 'recall';
         setState(prev => ({ ...prev, ...data.case_data, currentTab: safeTab || 'recall', isSessionActive: false }));
      }
    } catch (err) {
      console.error("Failed to load from cloud:", err);
    }
  };

  const testConnection = async () => {
    setConnectionTestStatus('testing');
    if (!supabase) { setConnectionTestStatus('fail'); return; }
    try {
        const { error } = await supabase.from('chronicle_cases').select('count', { count: 'exact', head: true });
        if (error) throw error;
        setConnectionTestStatus('success');
    } catch (e) { setConnectionTestStatus('fail'); }
  };

  const handleAuth = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!supabase) return;
    setCloudMessage('');
    try {
      if (authMode === 'signup') {
        const { error } = await supabase.auth.signUp({ email: authEmail, password: authPassword });
        if (error) throw error;
        setCloudMessage("Check email!");
      } else {
        const { error } = await supabase.auth.signInWithPassword({ email: authEmail, password: authPassword });
        if (error) throw error;
      }
    } catch (err: any) {
      setCloudMessage(err.message);
    }
  };

  const handleLogout = async () => {
    if (!supabase) return;
    await supabase.auth.signOut();
    setUser(null);
  };

  // Handlers for Voice Session
  const handleEventRecorded = useCallback((eventData: Partial<TimelineEvent>) => {
    const newEvent: TimelineEvent = {
      id: crypto.randomUUID(),
      // PROVENANCE STAMP: STRICTLY CHRONICLE
      origin: 'Chronicle',
      verificationStatus: 'Verified',
      
      date: eventData.date || 'Unknown',
      category: eventData.category || 'Current Case Incident',
      description: eventData.description || '',
      location: eventData.location,
      childPresence: eventData.childPresence || 'Unknown',
      witnesses: eventData.witnesses || [],
      emotionalContext: eventData.emotionalContext,
      reactionContext: eventData.reactionContext,
      vulnerability: eventData.vulnerability,
      mood: eventData.mood,
      rawQuotes: eventData.rawQuotes,
      historicalTraumaLink: eventData.historicalTraumaLink,
      manipulationPattern: eventData.manipulationPattern,
      weaponization: eventData.weaponization,
      courtExplanation: eventData.courtExplanation || 'Framed as response to context.',
      evidenceStatus: (eventData.evidenceStatus as any) || 'Need',
      evidenceStrength: (eventData.evidenceStrength as any) || 'Moderate',
      evidenceTags: eventData.evidenceTags || [],
      evidenceLinks: eventData.evidenceLinks || [],
      isSignificant: eventData.isSignificant || false,
      involvedEntities: eventData.involvedEntities || [],
      createdAt: Date.now()
    };
    setState(prev => ({ ...prev, events: [...prev.events, newEvent] }));
  }, []);

  const handleEntityRecorded = useCallback((entityData: Partial<Entity>) => {
    const newEntity: Entity = {
      id: crypto.randomUUID(),
      origin: 'Chronicle', // Provenance
      name: entityData.name || 'Unknown',
      type: entityData.type || 'other',
      role: entityData.role,
      relationship: entityData.relationship,
      impactOnCase: (entityData.impactOnCase as any) || 'medium',
      frequency: entityData.frequency,
      notes: entityData.notes || '',
    };
    setState(prev => {
      const exists = prev.entities.findIndex(e => e.name.toLowerCase() === newEntity.name.toLowerCase());
      if (exists !== -1) return prev;
      return { ...prev, entities: [...prev.entities, newEntity] };
    });
  }, []);

  const handleContextFactRecorded = useCallback((factData: Partial<ContextFact>) => {
    const newFact: ContextFact = {
      id: crypto.randomUUID(),
      origin: 'Chronicle', // Provenance
      category: factData.category || 'Personal History',
      content: factData.content || '',
      source: factData.source || 'User Interview',
      relevance: factData.relevance || 'Context'
    };
    setState(prev => ({ ...prev, contextFacts: [...prev.contextFacts, newFact] }));
  }, []);

  // Update Context Phase
  const handleContextPhasesUpdate = useCallback((newPhases: ContextPhase[]) => {
      setState(prev => ({ ...prev, contextPhases: newPhases }));
  }, []);

  const handleUpdateEvent = useCallback((updatedEvent: TimelineEvent) => {
    setState(prev => ({
      ...prev,
      events: prev.events.map(e => e.id === updatedEvent.id ? updatedEvent : e)
    }));
  }, []);

  const deleteEvent = useCallback((id: string) => {
    if (window.confirm("Remove this entry?")) {
      setState(prev => ({ ...prev, events: prev.events.filter(e => e.id !== id) }));
    }
  }, []);

  // Central Notes Handlers
  const addGlobalNote = () => {
      if (!newNoteTitle || !newNoteContent) return;
      const note: GlobalNote = {
          id: crypto.randomUUID(),
          title: newNoteTitle,
          content: newNoteContent,
          category: newNoteCategory,
          createdAt: Date.now(),
          lastUpdated: Date.now()
      };
      setState(prev => ({ ...prev, globalNotes: [note, ...prev.globalNotes] }));
      setNewNoteTitle('');
      setNewNoteContent('');
      setNewNoteCategory('Observation');
  };

  const deleteNote = (id: string) => {
      if(window.confirm("Delete note?")) {
          setState(prev => ({ ...prev, globalNotes: prev.globalNotes.filter(n => n.id !== id) }));
      }
  };

  // Graph Updates
  const updateGraphData = (newEntities: Entity[], newRelationships: EntityRelationship[]) => {
      setState(prev => ({ ...prev, entities: newEntities, entityRelationships: newRelationships }));
  };

  const tabs = [
    { id: 'recall', label: 'Recall', icon: Mic },
    { id: 'context', label: 'Context', icon: Calendar }, // New Tab
    { id: 'timeline', label: 'Timeline', icon: Activity },
    { id: 'graph', label: 'Network', icon: Network },
    { id: 'reports', label: 'Reports', icon: FileText },
    { id: 'vault', label: 'Connections', icon: Server },
  ];

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col font-sans selection:bg-indigo-500 selection:text-white text-slate-200">
      
      {/* Settings Modal */}
      {isSettingsOpen && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
           <div className="bg-slate-900 rounded-3xl border border-slate-800 shadow-2xl max-w-sm w-full p-6">
              <div className="flex justify-between items-center mb-6">
                 <h3 className="text-lg font-black uppercase text-indigo-400">System Config</h3>
                 <button onClick={() => setIsSettingsOpen(false)} className="text-slate-500 hover:text-white"><LogOut size={20} /></button>
              </div>
              <div className="space-y-4">
                 <div>
                    <label className="text-[10px] font-black uppercase text-slate-500">Live Model</label>
                    <select 
                        value={state.aiSettings.liveModel} 
                        onChange={(e) => setState(p => ({...p, aiSettings: {...p.aiSettings, liveModel: e.target.value}}))}
                        className="w-full bg-slate-800 border border-slate-700 rounded-lg p-2 text-xs mt-1"
                    >
                        <option value="gemini-2.5-flash-native-audio-preview-09-2025">Gemini 2.5 Flash Audio</option>
                    </select>
                 </div>
                 <div>
                    <label className="text-[10px] font-black uppercase text-slate-500">Analysis Model</label>
                    <select 
                        value={state.aiSettings.analysisModel} 
                        onChange={(e) => setState(p => ({...p, aiSettings: {...p.aiSettings, analysisModel: e.target.value}}))}
                        className="w-full bg-slate-800 border border-slate-700 rounded-lg p-2 text-xs mt-1"
                    >
                        <option value="gemini-3-pro-preview">Gemini 3 Pro</option>
                        <option value="gemini-3-flash-preview">Gemini 3 Flash</option>
                    </select>
                 </div>
                 <button onClick={() => setIsSettingsOpen(false)} className="w-full py-2 bg-indigo-600 text-white text-xs font-bold rounded-lg mt-4">Close</button>
              </div>
           </div>
        </div>
      )}

      {/* Header */}
      <header className="bg-slate-950/90 border-b border-indigo-500/10 sticky top-0 z-50 backdrop-blur-md no-print">
        <div className="max-w-screen-2xl mx-auto px-4 lg:px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-indigo-600 p-1.5 rounded-lg shadow-lg shadow-indigo-500/20">
              <LayoutDashboard size={18} className="text-white" />
            </div>
            <div>
              <h1 className="text-sm font-black tracking-tighter uppercase text-white hidden sm:block">CHRONICLE</h1>
              <div className="flex items-center gap-2">
                {cloudStatus === 'synced' ? <span className="flex items-center gap-1 text-[9px] text-blue-400 font-bold"><Cloud size={10} /> SYNCED</span> : 
                 cloudStatus === 'disconnected' ? <span className="flex items-center gap-1 text-[9px] text-emerald-400 font-bold"><WifiOff size={10} /> LOCAL</span> : 
                 cloudStatus === 'updating' ? <span className="flex items-center gap-1 text-[9px] text-purple-400 font-bold animate-pulse"><RefreshCw size={10} className="animate-spin" /> MERGING...</span> :
                 <span className="text-[9px] text-amber-400 font-bold">SAVING...</span>}
              </div>
            </div>
          </div>

          <nav className="hidden md:flex items-center bg-slate-900 p-1 rounded-xl border border-slate-800">
            {tabs.map(tab => {
              const Icon = tab.icon;
              return (
                <button 
                  key={tab.id}
                  onClick={() => setState(p => ({...p, currentTab: tab.id as any}))}
                  className={`px-3 py-1.5 text-[10px] font-black uppercase tracking-widest rounded-lg flex items-center gap-2 transition-all ${state.currentTab === tab.id ? 'bg-indigo-600 text-white shadow-md' : 'text-slate-500 hover:text-white hover:bg-slate-800'}`}
                >
                  <Icon size={12} />
                  {tab.label}
                </button>
              );
            })}
          </nav>

          <button onClick={() => setIsSettingsOpen(true)} className="p-2 text-slate-400 hover:text-indigo-400 transition-colors">
            <Settings size={20} />
          </button>
        </div>
        
        {/* Mobile Tab Bar */}
        <div className="md:hidden flex overflow-x-auto border-t border-slate-800 bg-slate-950 pb-safe">
            {tabs.map(tab => {
                const Icon = tab.icon;
                return (
                    <button key={tab.id} onClick={() => setState(p => ({...p, currentTab: tab.id as any}))} className={`flex-1 flex flex-col items-center justify-center py-3 border-b-2 ${state.currentTab === tab.id ? 'border-indigo-500 text-indigo-400' : 'border-transparent text-slate-500'}`}>
                        <Icon size={16} className="mb-1" />
                        <span className="text-[9px] font-bold uppercase">{tab.label}</span>
                    </button>
                )
            })}
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col overflow-hidden relative z-10">
        
        {state.currentTab === 'recall' && (
          <div className="flex-1 p-4 lg:p-10 overflow-y-auto">
             <div className="max-w-6xl mx-auto space-y-8">
               
               {/* Top: Voice & Analysis */}
               <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                   <div className="lg:col-span-2 bg-slate-900/50 rounded-[2.5rem] p-8 border border-indigo-500/10 shadow-2xl flex flex-col items-center relative backdrop-blur-md min-h-[400px] justify-center">
                     <VoiceSession 
                       onEventRecorded={handleEventRecorded}
                       onEntityRecorded={handleEntityRecorded}
                       onContextFactRecorded={handleContextFactRecorded}
                       isActive={state.isSessionActive}
                       onSessionStateChange={(active) => setState(prev => ({...prev, isSessionActive: active}))}
                       caseContext={state.caseContext}
                       contextPhases={state.contextPhases} // PASS CONTEXT PHASES TO AI
                       aiSettings={state.aiSettings}
                     />
                     {state.isSessionActive && <div className="mt-8 text-center text-xs text-indigo-300 animate-pulse font-medium">Listening for entities, relationships, and events...</div>}
                   </div>

                   {/* Master Narrative Context */}
                   <div className="flex flex-col gap-4 h-full">
                        <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 flex-1 flex flex-col">
                            <div className="flex items-center gap-2 text-indigo-400 mb-4">
                                <Network size={18} />
                                <h3 className="text-xs font-black uppercase tracking-widest">Master Context</h3>
                            </div>
                            <textarea 
                                value={state.caseContext}
                                onChange={(e) => setState(p => ({...p, caseContext: e.target.value}))}
                                className="w-full flex-1 bg-slate-950 border border-slate-800 rounded-xl p-4 text-xs leading-relaxed text-slate-300 focus:border-indigo-500 outline-none resize-none"
                                placeholder="Describe the high-level context here. This informs the AI's understanding of all sessions."
                            />
                        </div>
                   </div>
               </div>
               
               {/* Middle: Session Debrief (Dynamic) */}
               {sessionAnalysis && (
                  <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 relative">
                      <button onClick={() => setSessionAnalysis(null)} className="absolute top-4 right-4 text-slate-500 hover:text-white">✕</button>
                      <h3 className="text-xs font-black text-indigo-400 uppercase tracking-widest mb-4">Session Debrief</h3>
                      <div className="prose prose-sm prose-invert max-w-none text-xs text-slate-300 whitespace-pre-wrap">{sessionAnalysis}</div>
                  </div>
               )}

               {/* Bottom: Central Case Notes */}
               <div className="pt-8 border-t border-slate-800">
                    <div className="flex items-center justify-between mb-6">
                        <div className="flex items-center gap-2 text-amber-400">
                            <FileText size={16} />
                            <h3 className="text-xs font-black uppercase tracking-widest">Self-Discovery & Notes</h3>
                        </div>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        {/* New Note Input */}
                        <div className="bg-slate-900 border border-slate-700 rounded-xl p-4 space-y-3 h-fit">
                            <input 
                                type="text" 
                                placeholder="Note Title (e.g., 'Lost Strength: Public Speaking')" 
                                value={newNoteTitle} 
                                onChange={e => setNewNoteTitle(e.target.value)}
                                className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs font-bold text-white outline-none focus:border-indigo-500"
                            />
                            <select 
                                value={newNoteCategory} 
                                onChange={(e) => setNewNoteCategory(e.target.value as any)}
                                className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-slate-300 outline-none focus:border-indigo-500"
                            >
                                <option value="Observation">Observation</option>
                                <option value="Reflection">Reflection</option>
                                <option value="Memory">Memory</option>
                                <option value="Strength">Strength</option>
                                <option value="Vulnerability">Vulnerability</option>
                                <option value="Strategy">Strategy</option>
                            </select>
                            <textarea 
                                placeholder="Write your observation, theory, or strategy..." 
                                value={newNoteContent}
                                onChange={e => setNewNoteContent(e.target.value)}
                                className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-slate-300 h-32 resize-none outline-none focus:border-indigo-500"
                            />
                            <button onClick={addGlobalNote} className="w-full bg-indigo-600 hover:bg-indigo-500 text-white py-2 rounded text-xs font-bold uppercase">Add Note</button>
                        </div>

                        {/* Note List */}
                        <div className="lg:col-span-2 grid grid-cols-1 md:grid-cols-2 gap-4">
                            {state.globalNotes.length === 0 && <p className="text-slate-600 text-xs italic p-4">No notes yet.</p>}
                            {state.globalNotes.map(note => (
                                <div key={note.id} className="bg-slate-900 border border-slate-800 rounded-xl p-4 hover:border-slate-600 transition-colors group relative">
                                    <button onClick={() => deleteNote(note.id)} className="absolute top-2 right-2 text-slate-600 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity">✕</button>
                                    <h4 className="text-xs font-bold text-white mb-2">{note.title}</h4>
                                    <p className="text-xs text-slate-400 whitespace-pre-wrap leading-relaxed">{note.content}</p>
                                    <div className="mt-3 pt-3 border-t border-slate-800 flex justify-between items-center">
                                        <span className={`text-[9px] font-bold uppercase px-2 py-0.5 rounded ${
                                            note.category === 'Strength' ? 'bg-amber-950 text-amber-400' :
                                            note.category === 'Vulnerability' ? 'bg-rose-950 text-rose-400' :
                                            'bg-slate-800 text-slate-400'
                                        }`}>{note.category}</span>
                                        <span className="text-[9px] text-slate-600">{new Date(note.createdAt).toLocaleDateString()}</span>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
             </div>
          </div>
        )}
        
        {state.currentTab === 'context' && (
           <ContextGrid phases={state.contextPhases} onUpdate={handleContextPhasesUpdate} />
        )}

        {state.currentTab === 'timeline' && (
          <Timeline events={state.events} onDelete={deleteEvent} onUpdate={handleUpdateEvent} analysisModel={state.aiSettings.analysisModel} />
        )}

        {state.currentTab === 'graph' && (
           <KnowledgeGraph data={state} onUpdate={updateGraphData} />
        )}

        {state.currentTab === 'reports' && (
           <Reports events={state.events} entities={state.entities} notes={state.globalNotes} aiSettings={state.aiSettings} />
        )}

        {state.currentTab === 'vault' && (
          <div className="flex-1 p-4 lg:p-10 overflow-y-auto">
             <div className="max-w-4xl mx-auto space-y-10">
                
                <div className="flex items-center gap-3 mb-8 border-b border-slate-800 pb-4">
                    <Server className="text-indigo-400" size={24} />
                    <h2 className="text-2xl font-black text-white uppercase tracking-tighter">Connection Console</h2>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    
                    {/* Supabase Connection */}
                    <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 space-y-4">
                         <div className="flex items-center gap-2 text-emerald-400 mb-2">
                            <Database size={16} />
                            <h3 className="text-xs font-black uppercase tracking-widest">Data Sync (Supabase)</h3>
                        </div>
                         <div className="flex justify-between items-center pb-4 border-b border-slate-800">
                            <span className="text-xs font-bold text-slate-400">Status</span>
                            <div className="flex items-center gap-2">
                                <span className={`text-[10px] font-black uppercase ${connectionTestStatus === 'success' ? 'text-emerald-400' : 'text-slate-500'}`}>{connectionTestStatus.toUpperCase() || 'IDLE'}</span>
                                <button onClick={testConnection} className="bg-slate-800 hover:bg-slate-700 text-slate-300 px-2 py-1 rounded text-[10px] font-bold">TEST</button>
                            </div>
                         </div>
                         <div className="space-y-3">
                            {!user ? (
                                <form onSubmit={handleAuth} className="space-y-2">
                                    <input type="email" placeholder="Email" value={authEmail} onChange={e => setAuthEmail(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" />
                                    <input type="password" placeholder="Password" value={authPassword} onChange={e => setAuthPassword(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" />
                                    <button type="submit" className="w-full bg-indigo-600 hover:bg-indigo-500 text-white py-2 rounded text-xs font-bold uppercase">Connect Cloud</button>
                                    <p className="text-[10px] text-slate-500 text-center">{cloudMessage}</p>
                                </form>
                            ) : (
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-indigo-300">{user.email}</span>
                                    <button onClick={handleLogout} className="text-red-400 text-[10px] font-bold uppercase hover:underline">Disconnect</button>
                                </div>
                            )}
                         </div>
                    </div>

                    {/* Weaviate Connection */}
                    <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 space-y-4">
                         <div className="flex items-center gap-2 text-indigo-400 mb-2">
                            <Network size={16} />
                            <h3 className="text-xs font-black uppercase tracking-widest">Semantic Memory (Weaviate)</h3>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <label className="text-[9px] font-bold text-slate-500 uppercase block mb-1">Cluster URL</label>
                                <input 
                                    type="text" 
                                    placeholder="https://your-cluster.weaviate.cloud" 
                                    value={state.connectionSettings.weaviateUrl}
                                    onChange={(e) => setState(p => ({...p, connectionSettings: {...p.connectionSettings, weaviateUrl: e.target.value}}))}
                                    className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" 
                                />
                            </div>
                            <div>
                                <label className="text-[9px] font-bold text-slate-500 uppercase block mb-1">API Key</label>
                                <input 
                                    type="password" 
                                    placeholder="sk-..." 
                                    value={state.connectionSettings.weaviateApiKey}
                                    onChange={(e) => setState(p => ({...p, connectionSettings: {...p.connectionSettings, weaviateApiKey: e.target.value}}))}
                                    className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" 
                                />
                            </div>
                            <div className="pt-2">
                                <p className="text-[10px] text-slate-500 leading-tight">Enables semantic search of your timeline events (e.g. "Find times I felt scared").</p>
                            </div>
                        </div>
                    </div>

                    {/* Neo4j Connection */}
                    <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 space-y-4">
                         <div className="flex items-center gap-2 text-blue-400 mb-2">
                            <Network size={16} />
                            <h3 className="text-xs font-black uppercase tracking-widest">Knowledge Graph (Neo4j)</h3>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <label className="text-[9px] font-bold text-slate-500 uppercase block mb-1">Bolt URL</label>
                                <input 
                                    type="text" 
                                    placeholder="bolt://localhost:7687" 
                                    value={state.connectionSettings.neo4jUrl}
                                    onChange={(e) => setState(p => ({...p, connectionSettings: {...p.connectionSettings, neo4jUrl: e.target.value}}))}
                                    className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" 
                                />
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                                <div>
                                    <label className="text-[9px] font-bold text-slate-500 uppercase block mb-1">User</label>
                                    <input 
                                        type="text" 
                                        placeholder="neo4j" 
                                        value={state.connectionSettings.neo4jUser}
                                        onChange={(e) => setState(p => ({...p, connectionSettings: {...p.connectionSettings, neo4jUser: e.target.value}}))}
                                        className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" 
                                    />
                                </div>
                                <div>
                                    <label className="text-[9px] font-bold text-slate-500 uppercase block mb-1">Password</label>
                                    <input 
                                        type="password" 
                                        placeholder="..." 
                                        value={state.connectionSettings.neo4jPassword}
                                        onChange={(e) => setState(p => ({...p, connectionSettings: {...p.connectionSettings, neo4jPassword: e.target.value}}))}
                                        className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white" 
                                    />
                                </div>
                            </div>
                            <div className="pt-2">
                                <p className="text-[10px] text-slate-500 leading-tight">Powers the entity relationship graph with industrial strength querying.</p>
                            </div>
                        </div>
                    </div>

                </div>
             </div>
          </div>
        )}
      </main>
    </div>
  );
};

export default App;