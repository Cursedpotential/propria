
import React, { useState, useEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { AnalysisRecord, EvidenceChunk, VerificationStatus } from '../types.ts';
import { db } from '../lib/db.ts';
import { X, CheckCircle, AlertTriangle, Clock, Save } from 'lucide-react';

interface SplitInspectorProps {
  record: AnalysisRecord;
  onClose: () => void;
  addNotification: (message: string, type?: 'success' | 'error' | 'info') => void;
}

export function SplitInspector({ record, onClose, addNotification }: SplitInspectorProps) {
  const chunk = useLiveQuery(() => db.evidenceChunks.get(record.chunk_id), [record.chunk_id]);
  const [cleanText, setCleanText] = useState(record.clean_text);
  const [isDirty, setIsDirty] = useState(false);

  useEffect(() => {
    setCleanText(record.clean_text);
    setIsDirty(false);
  }, [record]);

  const handleTextChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setCleanText(e.target.value);
    setIsDirty(true);
  };

  const handleSave = async () => {
    try {
      await db.analysisRecords.update(record.id, { clean_text: cleanText });
      setIsDirty(false);
      addNotification('Clean text saved.', 'success');
    } catch (error) {
      addNotification('Failed to save text.', 'error');
    }
  };

  const handleStatusChange = async (status: VerificationStatus) => {
    try {
      await db.analysisRecords.update(record.id, { verification_status: status });
      addNotification(`Status updated to ${status}.`, 'success');
    } catch (error) {
      addNotification('Failed to update status.', 'error');
    }
  };

  return (
    <div className="bg-gray-800/70 backdrop-blur-sm border border-gray-700 rounded-lg shadow-2xl p-4 space-y-4 animate-fade-in">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-100">Inspector: <span className="font-mono text-cyan-400">{record.short_id}</span></h3>
        <div className="flex items-center space-x-2">
          {isDirty && (
            <button onClick={handleSave} className="flex items-center text-sm px-3 py-1 rounded-md bg-green-600 text-white hover:bg-green-700">
              <Save className="h-4 w-4 mr-1" /> Save
            </button>
          )}
          <button onClick={onClose} className="p-1 rounded-md text-gray-400 hover:text-white hover:bg-gray-700">
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Left Panel: Raw Text */}
        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">Raw Evidence (Immutable)</label>
          <textarea
            readOnly
            value={chunk?.raw_text || 'Loading...'}
            className="w-full h-48 p-2 font-mono text-sm bg-gray-900 border border-gray-600 rounded-md text-gray-400 resize-none"
            aria-label="Raw evidence text"
          />
        </div>
        {/* Right Panel: Clean Text */}
        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">Clean Text (Editable)</label>
          <textarea
            value={cleanText}
            onChange={handleTextChange}
            className="w-full h-48 p-2 font-mono text-sm bg-gray-800 border border-gray-600 rounded-md text-gray-200 focus:ring-cyan-500 focus:border-cyan-500 resize-none"
            aria-label="Clean, editable text"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* AI Entities */}
        <div>
          <h4 className="text-sm font-medium text-gray-400 mb-1">AI Extracted Entities</h4>
          <div className="bg-gray-900 p-3 rounded-md border border-gray-700 h-24 overflow-y-auto">
            {record.entities && Array.isArray(record.entities) && record.entities.length > 0 ? (
              <ul className="space-y-1">
                {record.entities.map((entity, index) => (
                  <li key={index} className="text-sm">
                    <span className="font-semibold text-cyan-300">{entity.name}</span>
                    <span className="text-gray-400"> ({entity.type})</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-gray-500">No entities extracted. Use the 'Analyze' button in the table.</p>
            )}
          </div>
        </div>
        {/* Verification Status */}
        <div>
          <h4 className="text-sm font-medium text-gray-400 mb-1">Verification</h4>
          <div className="flex space-x-2">
            <button onClick={() => handleStatusChange('Verified')} className={`flex-1 flex items-center justify-center p-2 text-sm rounded-md border ${record.verification_status === 'Verified' ? 'bg-green-500/20 border-green-500 text-green-300' : 'bg-gray-700/50 border-gray-600 text-gray-300 hover:bg-green-500/10'}`}>
              <CheckCircle className="h-5 w-5 mr-2" /> Verified
            </button>
            <button onClick={() => handleStatusChange('Flagged')} className={`flex-1 flex items-center justify-center p-2 text-sm rounded-md border ${record.verification_status === 'Flagged' ? 'bg-yellow-500/20 border-yellow-500 text-yellow-300' : 'bg-gray-700/50 border-gray-600 text-gray-300 hover:bg-yellow-500/10'}`}>
              <AlertTriangle className="h-5 w-5 mr-2" /> Flagged
            </button>
            <button onClick={() => handleStatusChange('Pending')} className={`flex-1 flex items-center justify-center p-2 text-sm rounded-md border ${record.verification_status === 'Pending' ? 'bg-gray-500/20 border-gray-500 text-gray-300' : 'bg-gray-700/50 border-gray-600 text-gray-300 hover:bg-gray-500/10'}`}>
              <Clock className="h-5 w-5 mr-2" /> Pending
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
