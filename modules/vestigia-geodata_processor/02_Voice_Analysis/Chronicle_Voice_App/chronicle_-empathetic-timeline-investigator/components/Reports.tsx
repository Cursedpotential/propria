
import React, { useState } from 'react';
import { TimelineEvent, Entity, GlobalNote, AISettings } from '../types';
import { Filter, FileText, Printer, Sparkles, AlertTriangle, Scale, Calendar, UserCheck } from 'lucide-react';
import { GoogleGenAI } from "@google/genai";

interface ReportsProps {
  events: TimelineEvent[];
  entities: Entity[];
  notes: GlobalNote[];
  aiSettings: AISettings;
}

type DocumentType = 'Affidavit' | 'Pattern Report' | 'Child Impact' | 'Character Brief';

const Reports: React.FC<ReportsProps> = ({ events, entities, notes, aiSettings }) => {
  const [activeDoc, setActiveDoc] = useState<DocumentType>('Affidavit');
  const [generatedContent, setGeneratedContent] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);

  const generateDocument = async (type: DocumentType) => {
    setIsGenerating(true);
    const ai = new GoogleGenAI({ apiKey: process.env.API_KEY });
    const model = aiSettings.analysisModel || 'gemini-3-pro-preview';
    
    let prompt = "";
    
    // Context Construction
    const relevantEvents = events.map(e => ({
        date: e.date,
        desc: e.description,
        cat: e.category,
        pattern: e.manipulationPattern,
        child: e.childPresence
    }));

    if (type === 'Affidavit') {
        prompt = `Act as a Forensic Legal Aide. Draft a "Statement of Facts" for a family court affidavit based strictly on the timeline below.
        
        RULES:
        1. Tone: Clinical, Objective, Fact-Based (No emotional embellishment).
        2. Format: Chronological list with numbered paragraphs.
        3. Highlight: Incidents of conflict, safety concerns, and agreements broken.
        4. Do NOT invent information.
        
        DATA:
        ${JSON.stringify(relevantEvents)}`;
    } else if (type === 'Pattern Report') {
        prompt = `Act as a Forensic Psychologist. Create a "Behavioral Pattern Analysis" report.
        
        RULES:
        1. Do NOT list chronologically. Group by BEHAVIOR (e.g., "Alienation", "Gaslighting", "Financial Abuse").
        2. Cite specific dates as evidence for each pattern.
        3. Conclusion: Summary of the psychological impact on the subject.
        
        DATA:
        ${JSON.stringify(relevantEvents)}`;
    } else if (type === 'Child Impact') {
        prompt = `Act as a Child Advocate. Create a "Child Impact Assessment".
        
        RULES:
        1. Filter ONLY for events where 'child' is NOT 'Not Present'.
        2. Analyze exposure to conflict.
        3. Assess stability of routine.
        
        DATA:
        ${JSON.stringify(relevantEvents.filter(e => e.child !== 'Not Present'))}`;
    }

    try {
        const response = await ai.models.generateContent({
            model: model,
            contents: prompt
        });
        setGeneratedContent(response.text || "Failed to generate report.");
    } catch (e) {
        setGeneratedContent("Error generating report. Please check API Key or try again.");
    } finally {
        setIsGenerating(false);
    }
  };

  return (
    <div className="flex h-full bg-slate-950 text-slate-200">
      
      {/* Left Panel: Document Templates */}
      <div className="w-64 bg-slate-900 border-r border-slate-800 p-4 flex flex-col gap-4 no-print">
         <h2 className="text-xs font-black uppercase text-slate-500 tracking-widest mb-2 flex items-center gap-2">
            <FileText size={14} /> Document Generator
         </h2>
         
         <button 
            onClick={() => { setActiveDoc('Affidavit'); setGeneratedContent(''); }}
            className={`text-left p-3 rounded-xl border transition-all ${activeDoc === 'Affidavit' ? 'bg-indigo-600 border-indigo-500 text-white shadow-lg' : 'bg-slate-800 border-slate-700 text-slate-400 hover:bg-slate-700'}`}
         >
            <div className="flex items-center gap-2 mb-1">
                <Scale size={14} />
                <span className="text-xs font-bold uppercase">Chronological Affidavit</span>
            </div>
            <p className="text-[9px] opacity-70">Strict facts for legal filing. No emotion.</p>
         </button>

         <button 
            onClick={() => { setActiveDoc('Pattern Report'); setGeneratedContent(''); }}
            className={`text-left p-3 rounded-xl border transition-all ${activeDoc === 'Pattern Report' ? 'bg-indigo-600 border-indigo-500 text-white shadow-lg' : 'bg-slate-800 border-slate-700 text-slate-400 hover:bg-slate-700'}`}
         >
            <div className="flex items-center gap-2 mb-1">
                <AlertTriangle size={14} />
                <span className="text-xs font-bold uppercase">Pattern Analysis</span>
            </div>
            <p className="text-[9px] opacity-70">Groups events by behavioral tactic.</p>
         </button>

         <button 
            onClick={() => { setActiveDoc('Child Impact'); setGeneratedContent(''); }}
            className={`text-left p-3 rounded-xl border transition-all ${activeDoc === 'Child Impact' ? 'bg-indigo-600 border-indigo-500 text-white shadow-lg' : 'bg-slate-800 border-slate-700 text-slate-400 hover:bg-slate-700'}`}
         >
            <div className="flex items-center gap-2 mb-1">
                <UserCheck size={14} />
                <span className="text-xs font-bold uppercase">Child Impact</span>
            </div>
            <p className="text-[9px] opacity-70">Focus on child exposure & welfare.</p>
         </button>
      </div>

      {/* Right Panel: Content */}
      <div className="flex-1 flex flex-col bg-slate-50 text-slate-900 relative">
          
          {/* Toolbar */}
          <div className="h-14 border-b border-slate-200 bg-white flex items-center justify-between px-6 no-print">
              <h3 className="text-sm font-black uppercase tracking-wide text-slate-800">{activeDoc} Draft</h3>
              <div className="flex gap-2">
                  <button 
                    onClick={() => generateDocument(activeDoc)}
                    disabled={isGenerating}
                    className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white text-xs font-bold uppercase rounded-lg hover:bg-indigo-500 disabled:opacity-50"
                  >
                      {isGenerating ? <Sparkles size={14} className="animate-spin" /> : <Sparkles size={14} />}
                      {generatedContent ? 'Regenerate' : 'Generate Draft'}
                  </button>
                  <button onClick={() => window.print()} className="p-2 text-slate-500 hover:text-indigo-600">
                      <Printer size={18} />
                  </button>
              </div>
          </div>

          {/* Document Preview */}
          <div className="flex-1 overflow-y-auto p-8 lg:p-12 print-container">
              {generatedContent ? (
                  <div className="max-w-3xl mx-auto bg-white shadow-sm border border-slate-200 p-12 min-h-[800px] print:shadow-none print:border-none print:p-0">
                      <div className="mb-8 border-b-2 border-slate-900 pb-4">
                          <h1 className="text-2xl font-black uppercase text-slate-900">{activeDoc.toUpperCase()}</h1>
                          <p className="text-sm text-slate-500">Generated: {new Date().toLocaleDateString()}</p>
                      </div>
                      <div className="prose prose-sm max-w-none prose-slate">
                           <div className="whitespace-pre-wrap leading-relaxed font-serif text-slate-800">
                               {generatedContent}
                           </div>
                      </div>
                  </div>
              ) : (
                  <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-4">
                      <FileText size={48} className="opacity-20" />
                      <p className="text-sm font-medium">Select a template and click Generate to draft a forensic report.</p>
                  </div>
              )}
          </div>
      </div>

    </div>
  );
};

export default Reports;
