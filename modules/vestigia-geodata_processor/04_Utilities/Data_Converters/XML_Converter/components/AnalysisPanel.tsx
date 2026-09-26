import React from 'react';
import { AnalysisResult } from '../types';

interface Props {
  analysis: AnalysisResult | null;
  loading: boolean;
  onAnalyze: () => void;
  hasData: boolean;
}

export const AnalysisPanel: React.FC<Props> = ({ analysis, loading, onAnalyze, hasData }) => {
  return (
    <div className="bg-gradient-to-br from-slate-900 to-slate-800 p-6 rounded-lg border border-indigo-500/30 shadow-lg relative overflow-hidden">
      <div className="absolute top-0 right-0 w-32 h-32 bg-indigo-500/10 rounded-full blur-3xl -translate-y-16 translate-x-16"></div>
      
      <div className="flex justify-between items-start mb-6 relative z-10">
        <div>
          <h3 className="text-lg font-semibold text-white flex items-center gap-2">
             <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
            Deep Research Analysis
          </h3>
          <p className="text-slate-400 text-sm mt-1">Powered by Gemini 3 Flash</p>
        </div>
        {!analysis && (
          <button
            onClick={onAnalyze}
            disabled={loading || !hasData}
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-md text-sm font-medium transition-all shadow-md shadow-indigo-900/20"
          >
            {loading ? 'Analyzing...' : 'Analyze Context'}
          </button>
        )}
      </div>

      {loading && (
        <div className="animate-pulse space-y-4">
          <div className="h-4 bg-slate-700 rounded w-3/4"></div>
          <div className="h-4 bg-slate-700 rounded w-1/2"></div>
          <div className="grid grid-cols-2 gap-4">
            <div className="h-20 bg-slate-700 rounded"></div>
            <div className="h-20 bg-slate-700 rounded"></div>
          </div>
        </div>
      )}

      {analysis && (
        <div className="space-y-6 relative z-10 animate-fade-in">
          <div className="bg-slate-900/50 p-4 rounded border border-slate-700">
            <h4 className="text-xs font-bold text-indigo-400 uppercase tracking-wider mb-2">Summary</h4>
            <p className="text-slate-300 text-sm leading-relaxed">{analysis.summary}</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-slate-900/50 p-4 rounded border border-slate-700">
              <h4 className="text-xs font-bold text-teal-400 uppercase tracking-wider mb-2">Participants</h4>
              <ul className="text-sm text-slate-300 list-disc list-inside">
                {analysis.participants.map((p, i) => <li key={i}>{p}</li>)}
              </ul>
            </div>
            
            <div className="bg-slate-900/50 p-4 rounded border border-slate-700">
              <h4 className="text-xs font-bold text-amber-400 uppercase tracking-wider mb-2">Details</h4>
              <div className="text-sm text-slate-300 space-y-2">
                <p><span className="text-slate-500">Dates:</span> {analysis.dateRange}</p>
                <div>
                   <span className="text-slate-500">Topics:</span>
                   <div className="flex flex-wrap gap-1 mt-1">
                     {analysis.topics.map((t, i) => (
                       <span key={i} className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">{t}</span>
                     ))}
                   </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};