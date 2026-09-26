
import React, { useState, useRef, useEffect } from 'react';
import { AnalysisRecord, VerificationStatus } from '../types.ts';
import { db } from '../lib/db.ts';
import { CheckCircle, AlertTriangle, Clock, Tag, Bot } from 'lucide-react';
import { useSettings } from '../hooks/useSettings.ts';
import { ForensicAI } from '../lib/aiClient.ts';

interface ReviewTableProps {
  records: AnalysisRecord[];
  selectedRecordId: string | null;
  onRecordSelect: (record: AnalysisRecord) => void;
  addNotification: (message: string, type?: 'success' | 'error' | 'info') => void;
}

type EditingCell = {
  recordId: string;
  field: keyof AnalysisRecord;
} | null;

const StatusIcon = ({ status }: { status: VerificationStatus }) => {
  switch (status) {
    case 'Verified': return <CheckCircle className="h-5 w-5 text-green-400" />;
    case 'Flagged': return <AlertTriangle className="h-5 w-5 text-yellow-400" />;
    case 'Pending':
    default: return <Clock className="h-5 w-5 text-gray-500" />;
  }
};

export function ReviewTable({ records, selectedRecordId, onRecordSelect, addNotification }: ReviewTableProps) {
  const [editingCell, setEditingCell] = useState<EditingCell>(null);
  const [editValue, setEditValue] = useState<string>('');
  const inputRef = useRef<HTMLInputElement>(null);
  const { settings } = useSettings();
  const [analyzingId, setAnalyzingId] = useState<string | null>(null);

  useEffect(() => {
    if (editingCell && inputRef.current) {
      inputRef.current.focus();
    }
  }, [editingCell]);

  const handleCellClick = (record: AnalysisRecord, field: keyof AnalysisRecord) => {
    if (field === 'clean_text' || field === 'tags') {
      setEditingCell({ recordId: record.id, field });
      setEditValue(Array.isArray(record[field]) ? (record[field] as string[]).join(', ') : String(record[field]));
    }
  };

  const handleSave = async () => {
    if (!editingCell) return;

    const { recordId, field } = editingCell;
    let valueToSave: string | string[] = editValue;
    if (field === 'tags') {
        valueToSave = editValue.split(',').map(t => t.trim()).filter(Boolean);
    }

    try {
      await db.analysisRecords.update(recordId, { [field]: valueToSave });
      addNotification('Update saved.', 'success');
    } catch (error) {
      console.error('Failed to save update:', error);
      addNotification('Failed to save update.', 'error');
    }
    setEditingCell(null);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSave();
    } else if (e.key === 'Escape') {
      setEditingCell(null);
    }
  };

  const handleAnalyze = async (record: AnalysisRecord) => {
    if (analyzingId) return;
    setAnalyzingId(record.id);
    addNotification(`Analyzing chunk ${record.short_id}...`, 'info');
    try {
      const aiClient = new ForensicAI(settings);
      const result = await aiClient.analyzeChunk(record.clean_text);
      
      const timelineDate = result.timeline_date ? new Date(result.timeline_date) : null;
      
      await db.analysisRecords.update(record.id, {
        entities: result.entities || [],
        timeline_date: timelineDate,
      });
      addNotification(`Analysis complete for ${record.short_id}.`, 'success');
    } catch (error) {
      console.error('AI analysis failed:', error);
      addNotification(error instanceof Error ? error.message : 'AI analysis failed.', 'error');
    } finally {
      setAnalyzingId(null);
    }
  };

  return (
    <div className="bg-gray-800/50 rounded-lg shadow-md overflow-auto h-full border border-gray-700">
      <table className="min-w-full divide-y divide-gray-700">
        <thead className="bg-gray-800 sticky top-0 z-10">
          <tr>
            <th scope="col" className="w-16 px-3 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Status</th>
            <th scope="col" className="w-24 px-3 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Short ID</th>
            <th scope="col" className="px-3 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Clean Text (Click to edit)</th>
            <th scope="col" className="w-48 px-3 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Tags</th>
            <th scope="col" className="w-24 px-3 py-3 text-center text-xs font-medium text-gray-400 uppercase tracking-wider">Analyze</th>
          </tr>
        </thead>
        <tbody className="bg-gray-900 divide-y divide-gray-700">
          {records.map((record) => (
            <tr
              key={record.id}
              onClick={() => onRecordSelect(record)}
              className={`cursor-pointer ${selectedRecordId === record.id ? 'bg-cyan-900/30' : 'hover:bg-gray-800/60'}`}
            >
              <td className="px-3 py-3 whitespace-nowrap text-sm text-gray-300"><StatusIcon status={record.verification_status} /></td>
              <td className="px-3 py-3 whitespace-nowrap text-sm font-mono text-gray-400">{record.short_id}</td>
              <td className="px-3 py-3 text-sm text-gray-300 max-w-md truncate" onClick={() => handleCellClick(record, 'clean_text')}>
                {editingCell?.recordId === record.id && editingCell?.field === 'clean_text' ? (
                  <input
                    ref={inputRef}
                    type="text"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={handleSave}
                    onKeyDown={handleKeyDown}
                    className="bg-gray-900 border border-cyan-500 rounded-md w-full px-2 py-1"
                  />
                ) : (
                  record.clean_text
                )}
              </td>
              <td className="px-3 py-3 text-sm text-gray-400" onClick={() => handleCellClick(record, 'tags')}>
                {editingCell?.recordId === record.id && editingCell?.field === 'tags' ? (
                  <input
                    ref={inputRef}
                    type="text"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={handleSave}
                    onKeyDown={handleKeyDown}
                    className="bg-gray-900 border border-cyan-500 rounded-md w-full px-2 py-1"
                  />
                ) : (
                  <div className="flex flex-wrap gap-1">
                    {record.tags.map(tag => <span key={tag} className="px-2 py-0.5 text-xs rounded-full bg-gray-700 text-gray-300 flex items-center gap-1"><Tag className="w-3 h-3"/>{tag}</span>)}
                  </div>
                )}
              </td>
              <td className="px-3 py-3 whitespace-nowrap text-center">
                <button 
                  onClick={(e) => { e.stopPropagation(); handleAnalyze(record); }}
                  disabled={analyzingId === record.id}
                  className="p-2 rounded-md text-cyan-400 hover:bg-cyan-400/10 disabled:text-gray-500 disabled:cursor-wait"
                  aria-label="Analyze with AI"
                >
                  <Bot className={`h-5 w-5 ${analyzingId === record.id ? 'animate-spin' : ''}`} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
