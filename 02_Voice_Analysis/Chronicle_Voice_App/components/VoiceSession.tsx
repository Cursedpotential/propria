
import React, { useState, useEffect, useRef, useCallback } from 'react';
import { GoogleGenAI, Modality, LiveServerMessage, Type, FunctionDeclaration } from '@google/genai';
import { decode, decodeAudioData, encode } from '../utils/audio';
import { TimelineEvent, Entity, ContextFact, ContextPhase, AISettings } from '../types';

interface VoiceSessionProps {
  onEventRecorded: (event: Partial<TimelineEvent>) => void;
  onEntityRecorded: (entity: Partial<Entity>) => void;
  onContextFactRecorded?: (fact: Partial<ContextFact>) => void;
  isActive: boolean;
  onSessionStateChange: (active: boolean) => void;
  caseContext?: string;
  contextPhases?: ContextPhase[]; // Injected Context Phases
  aiSettings?: AISettings;
}

const QUOTES = [
  "You own everything that happened to you. Tell your stories.",
  "The truth does not mind being questioned. A lie does not like being challenged.",
  "Courage is resistance to fear, mastery of fear, not absence of fear.",
  "Your history is a resource, not a destination.",
  "Healing is not linear.",
  "You are the author of your own timeline.",
  "Speak your truth, even if your voice shakes.",
  "It is not your job to be digestible.",
  "What is mentionable is manageable.",
  "Trauma interrupts the story. You are here to resume it."
];

const recordEventTool: FunctionDeclaration = {
  name: 'recordTimelineEvent',
  parameters: {
    type: Type.OBJECT,
    description: 'Records a life event, memory, or incident.',
    properties: {
      date: { type: Type.STRING, description: 'Date, year, or period (e.g., "Childhood", "Summer 1995").' },
      category: { 
        type: Type.STRING, 
        enum: ['Childhood Memory', 'Career/Professional', 'Strength/Resilience', 'Historical Trauma', 'Current Case Incident', 'Legal Milestone', 'Personal Growth', 'Medical/Psychiatric', 'Financial Abuse', 'Relationship Phase'],
      },
      description: { type: Type.STRING, description: 'Factual description of what happened.' },
      mood: { type: Type.STRING },
      vulnerability: { type: Type.STRING },
      rawQuotes: { type: Type.ARRAY, items: { type: Type.STRING } },
      location: { type: Type.STRING, description: 'Specific location (e.g., "Kitchen", "Disneyland", "Courtroom"). MANDATORY if known.' },
      childPresence: { type: Type.STRING },
      witnesses: { type: Type.ARRAY, items: { type: Type.STRING }, description: 'Names of people present. MANDATORY if known.' },
      emotionalContext: { type: Type.STRING },
      reactionContext: { type: Type.STRING },
      historicalTraumaLink: { type: Type.STRING, description: "If this event triggers a past trauma, note the link." },
      manipulationPattern: { type: Type.STRING },
      weaponization: { type: Type.STRING, description: "How the antagonist used the user's past/insecurity against them in this moment." },
      courtExplanation: { type: Type.STRING },
      evidenceStatus: { type: Type.STRING, enum: ['Have', 'Need'] },
      evidenceLinks: { type: Type.ARRAY, items: { type: Type.STRING }, description: 'Specific items of proof mentioned (texts, photos).' },
      isSignificant: { type: Type.BOOLEAN, description: 'Mark true if the user emphasizes this as a major turning point.' },
      involvedEntities: { type: Type.ARRAY, items: { type: Type.STRING } }
    },
    required: ['date', 'description', 'category']
  }
};

const recordEntityTool: FunctionDeclaration = {
  name: 'recordEntity',
  parameters: {
    type: Type.OBJECT,
    description: 'Records profiles for key people.',
    properties: {
      name: { type: Type.STRING },
      type: { type: Type.STRING },
      role: { type: Type.STRING },
      relationship: { type: Type.STRING },
      impactOnCase: { type: Type.STRING },
      frequency: { type: Type.STRING },
      notes: { type: Type.STRING }
    },
    required: ['name', 'type', 'relationship']
  }
};

const recordContextFactTool: FunctionDeclaration = {
    name: 'recordContextFact',
    parameters: {
        type: Type.OBJECT,
        description: 'Records a static fact about personal history.',
        properties: {
            category: { type: Type.STRING },
            content: { type: Type.STRING },
            source: { type: Type.STRING },
            relevance: { type: Type.STRING }
        },
        required: ['category', 'content']
    }
};

const VoiceSession: React.FC<VoiceSessionProps> = ({ 
  onEventRecorded, 
  onEntityRecorded, 
  onContextFactRecorded,
  isActive, 
  onSessionStateChange, 
  caseContext,
  contextPhases,
  aiSettings
}) => {
  const [isConnecting, setIsConnecting] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [volume, setVolume] = useState(0);
  const [isFlareActive, setIsFlareActive] = useState(false);
  const [currentQuote, setCurrentQuote] = useState(QUOTES[0]);
  
  const audioContextRef = useRef<AudioContext | null>(null);
  const outAudioContextRef = useRef<AudioContext | null>(null);
  const nextStartTimeRef = useRef<number>(0);
  const sourcesRef = useRef<Set<AudioBufferSourceNode>>(new Set());
  const sessionRef = useRef<any>(null);
  const processorRef = useRef<ScriptProcessorNode | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  useEffect(() => {
    // Cycle quote randomly on mount
    const randomQuote = QUOTES[Math.floor(Math.random() * QUOTES.length)];
    setCurrentQuote(randomQuote);
    
    // Cycle quote periodically
    const interval = setInterval(() => {
       const nextQuote = QUOTES[Math.floor(Math.random() * QUOTES.length)];
       setCurrentQuote(nextQuote);
    }, 60000); // Change every minute
    
    return () => clearInterval(interval);
  }, []);

  const stopSession = useCallback(() => {
    if (sessionRef.current) {
      sessionRef.current.close();
      sessionRef.current = null;
    }
    if (processorRef.current) {
      processorRef.current.disconnect();
      processorRef.current = null;
    }
    if (streamRef.current) {
        streamRef.current.getTracks().forEach(track => track.stop());
        streamRef.current = null;
    }
    sourcesRef.current.forEach(s => s.stop());
    sourcesRef.current.clear();
    setVolume(0);
    onSessionStateChange(false);
  }, [onSessionStateChange]);

  const triggerFlare = () => {
    setIsFlareActive(true);
    setTimeout(() => setIsFlareActive(false), 2000);
  };

  const startSession = async () => {
    setIsConnecting(true);
    try {
      const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
      
      if (!audioContextRef.current) {
        audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)({ sampleRate: 16000 });
      }
      if (!outAudioContextRef.current) {
        outAudioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)({ sampleRate: 24000 });
      }

      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      
      const liveModel = aiSettings?.liveModel || 'gemini-2.5-flash-native-audio-preview-09-2025';

      // Construct Master Context String from Phases
      const phaseContext = contextPhases 
        ? contextPhases.map(p => `- ${p.name} (${p.startDate} to ${p.endDate}): ${p.emotionalBaseline} [${p.category}]`).join('\n')
        : "No life phases defined yet.";

      const sessionPromise = ai.live.connect({
        model: liveModel,
        callbacks: {
            onopen: () => {
                setIsConnecting(false);
                onSessionStateChange(true);
                
                if (!audioContextRef.current) return;
                const source = audioContextRef.current.createMediaStreamSource(stream);
                const processor = audioContextRef.current.createScriptProcessor(4096, 1, 1);
                processorRef.current = processor;
                
                processor.onaudioprocess = (e) => {
                    const inputData = e.inputBuffer.getChannelData(0);
                    
                    let sum = 0;
                    for (let i = 0; i < inputData.length; i++) {
                        sum += inputData[i] * inputData[i];
                    }
                    const rms = Math.sqrt(sum / inputData.length);
                    setVolume(prev => prev * 0.8 + (rms * 10) * 0.2);

                    const pcmData = new Int16Array(inputData.length);
                    for (let i = 0; i < inputData.length; i++) {
                        pcmData[i] = Math.max(-1, Math.min(1, inputData[i])) * 32768;
                    }
                    const base64 = encode(new Uint8Array(pcmData.buffer));
                    sessionPromise.then(session => session.sendRealtimeInput({
                        media: {
                            mimeType: 'audio/pcm;rate=16000',
                            data: base64
                        }
                    }));
                };
                source.connect(processor);
                processor.connect(audioContextRef.current.destination);
            },
            onmessage: async (msg: LiveServerMessage) => {
                const audioData = msg.serverContent?.modelTurn?.parts?.[0]?.inlineData?.data;
                if (audioData && outAudioContextRef.current) {
                    setIsSpeaking(true);
                    const ctx = outAudioContextRef.current;
                    nextStartTimeRef.current = Math.max(nextStartTimeRef.current, ctx.currentTime);
                    const buffer = await decodeAudioData(decode(audioData), ctx, 24000, 1);
                    const source = ctx.createBufferSource();
                    source.buffer = buffer;
                    source.connect(ctx.destination);
                    source.onended = () => {
                        sourcesRef.current.delete(source);
                        if (sourcesRef.current.size === 0) setIsSpeaking(false);
                    };
                    source.start(nextStartTimeRef.current);
                    nextStartTimeRef.current += buffer.duration;
                    sourcesRef.current.add(source);
                }

                if (msg.toolCall) {
                    const responses = [];
                    for (const fc of msg.toolCall.functionCalls) {
                        const args = { ...fc.args as any, isSignificant: isFlareActive ? true : (fc.args as any).isSignificant };
                        
                        if (fc.name === 'recordTimelineEvent') {
                            onEventRecorded(args);
                            responses.push({ id: fc.id, name: fc.name, response: { result: "Event recorded." } });
                        } else if (fc.name === 'recordEntity') {
                            onEntityRecorded(args);
                            responses.push({ id: fc.id, name: fc.name, response: { result: "Entity recorded." } });
                        } else if (fc.name === 'recordContextFact' && onContextFactRecorded) {
                            onContextFactRecorded(args);
                            responses.push({ id: fc.id, name: fc.name, response: { result: "Context fact recorded." } });
                        } else {
                            responses.push({ id: fc.id, name: fc.name, response: { result: "Tool not found." } });
                        }
                    }
                    sessionPromise.then(session => session.sendToolResponse({ functionResponses: responses }));
                }
            },
            onclose: () => { stopSession(); },
            onerror: (err) => { console.error("Session error", err); stopSession(); }
        },
        config: {
          responseModalities: [Modality.AUDIO],
          speechConfig: { voiceConfig: { prebuiltVoiceConfig: { voiceName: 'Kore' } } },
          tools: [{ functionDeclarations: [recordEventTool, recordEntityTool, recordContextFactTool] }],
          systemInstruction: `You are Chronicle, a Forensic Therapist and Biographer for a user reclaiming their life history.

CORE MISSION:
The user is reconstructing their "Whole Life" context. You are the Anchor.

MASTER CONTEXT (LIFE ERAS):
Use these defined Eras to orient the user. If they mention a date, map it to the correct Era immediately.
${phaseContext}

YOUR ROLES:
1. **The Archaeologist:** Dig up the user's "Old Self." Ask about childhood dreams, professional successes, and moments of strength *before* the relationship. Record these as 'Childhood Memory', 'Career/Professional', or 'Strength/Resilience'.
2. **The Pattern Spotter:** When the user mentions a current conflict, actively scan for the root cause. Did the ex target a specific childhood insecurity? Did they undermine a specific professional strength?
   - *Example:* If user says "She made me feel stupid at the work party," check if the user had childhood academic insecurity. If so, record that link in "weaponization".
3. **The Anchor:** Embrace non-linearity. The user has ADD. They will jump from 1995 to 2024. Follow them. Do not force chronological order.

DATA PRIORITY:
- **LOCATION:** You MUST extract the specific location (e.g., "Kitchen," "Disneyland," "The Courtroom") for every event.
- **WITNESSES:** You MUST extract who else was there. This is critical for corroboration.

CONTEXT:
${caseContext || "No narrative summary provided."}
`,
        },
      });
      sessionRef.current = await sessionPromise;
    } catch (err) {
        console.error("Failed to connect", err);
        setIsConnecting(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center space-y-8 w-full relative">
      
      {/* Background Ambience / Volume Glow */}
      <div className={`absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full blur-3xl transition-all duration-100 ease-out pointer-events-none
        ${isActive ? 'opacity-50' : 'opacity-0'}
      `}
      style={{
        width: `${180 + volume * 350}px`,
        height: `${180 + volume * 350}px`,
        background: isFlareActive 
            ? 'radial-gradient(circle, rgba(234, 179, 8, 0.4) 0%, rgba(234, 179, 8, 0) 70%)' // Amber flare
            : 'radial-gradient(circle, rgba(99, 102, 241, 0.3) 0%, rgba(99, 102, 241, 0) 70%)' // Indigo normal
      }}
      ></div>

      <div className={`relative flex items-center justify-center w-40 h-40 rounded-full transition-all duration-500 border border-white/10 backdrop-blur-md z-10 ${isActive ? 'bg-slate-700/50 shadow-inner' : 'bg-slate-700/50 shadow-xl'}`}>
        {isActive && (
          <div className="absolute inset-0 rounded-full border border-indigo-400/30 animate-[spin_4s_linear_infinite]"></div>
        )}
        
        <button
          onClick={isActive ? stopSession : startSession}
          disabled={isConnecting}
          className={`w-28 h-28 rounded-full flex items-center justify-center shadow-2xl transition-all hover:scale-105 active:scale-95 z-20 relative group ${
            isActive 
              ? 'bg-gradient-to-b from-indigo-500 to-indigo-700 text-white shadow-indigo-500/40' 
              : 'bg-gradient-to-b from-indigo-600 to-indigo-700 text-white shadow-lg shadow-indigo-500/30'
          }`}
        >
           {isConnecting ? (
             <svg className="animate-spin h-10 w-10 text-white/80" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
           ) : isActive ? (
             <svg className="w-12 h-12 drop-shadow-md" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" /></svg>
           ) : (
             <svg className="w-12 h-12 drop-shadow-md" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" /></svg>
           )}
        </button>
      </div>

      {isActive && (
          <button 
            onClick={triggerFlare}
            className={`absolute -right-16 top-1/2 -translate-y-1/2 flex flex-col items-center gap-2 group transition-all duration-300 ${isFlareActive ? 'scale-110' : 'scale-100'}`}
          >
             <div className={`p-3 rounded-full border transition-all shadow-lg ${isFlareActive ? 'bg-amber-500 border-amber-300 text-white shadow-amber-500/50' : 'bg-slate-700 border-slate-600 text-amber-500 hover:bg-amber-950/30 hover:border-amber-500/50'}`}>
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" /></svg>
             </div>
             <span className="text-[9px] font-black uppercase tracking-widest text-amber-500/80 group-hover:text-amber-400">Flare</span>
          </button>
      )}

      <div className="text-center space-y-2 relative z-10">
        <h2 className="text-xl font-black text-white tracking-tight">
          {isActive ? (isSpeaking ? "Reflecting..." : "Listening...") : "Start Recall Session"}
        </h2>
        <p className="text-xs text-slate-300 max-w-xs mx-auto font-medium">
          {isActive 
            ? "I am listening for the whole story—childhood to now. Finding the links." 
            : "I am your forensic diary. Reclaim your timeline."}
        </p>
      </div>

      {/* Daily Anchor / Inspiration */}
      {!isActive && (
          <div className="pt-6 border-t border-slate-600/30 w-full max-w-sm">
             <p className="text-[10px] text-indigo-300 uppercase font-black tracking-widest mb-2 text-center">Daily Anchor</p>
             <p className="text-xs text-slate-300 text-center italic leading-relaxed animate-in fade-in duration-500">"{currentQuote}"</p>
          </div>
      )}
    </div>
  );
};

export default VoiceSession;