
import React from 'react';
import { TranscriptEntry } from '../types';
import { EditableSpeaker } from './EditableSpeaker';

interface TranscriptDisplayProps {
  transcript: TranscriptEntry[];
  speakerMap?: Record<string, string>;
  onSpeakerNameUpdate?: (originalName: string, newName:string) => void;
  onTimestampClick?: (timeInSeconds: number) => void;
}

const parseTimestamp = (timestamp: string): number => {
    return parseFloat(timestamp.replace('s', ''));
}

export const TranscriptDisplay: React.FC<TranscriptDisplayProps> = ({ transcript, speakerMap = {}, onSpeakerNameUpdate, onTimestampClick }) => {
  return (
    <div className="space-y-4 text-sm">
      {transcript.map((entry, index) => (
        <div key={index} className="flex items-start gap-3">
          <div className="w-28 flex-shrink-0">
            <EditableSpeaker 
              originalSpeaker={entry.speaker}
              speakerMap={speakerMap}
              onUpdate={onSpeakerNameUpdate}
            />
            <button 
              onClick={() => onTimestampClick && onTimestampClick(parseTimestamp(entry.timestamp))}
              className="text-xs text-cyan-400 hover:text-cyan-300 hover:underline mt-1 text-left disabled:text-slate-500 disabled:no-underline"
              title={`Jump to ${entry.timestamp}`}
              disabled={!onTimestampClick}
            >
              {entry.timestamp}
            </button>
          </div>
          <p className="text-slate-300 leading-relaxed">{entry.text}</p>
        </div>
      ))}
    </div>
  );
};
