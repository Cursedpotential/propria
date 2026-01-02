
import React, { useState, useEffect } from 'react';

interface EditableSpeakerProps {
  originalSpeaker: string;
  speakerMap: Record<string, string>;
  onUpdate?: (originalName: string, newName: string) => void;
}

export const EditableSpeaker: React.FC<EditableSpeakerProps> = ({ originalSpeaker, speakerMap, onUpdate }) => {
  const displayName = speakerMap[originalSpeaker] || originalSpeaker;
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState(displayName);

  useEffect(() => {
    setValue(displayName);
  }, [displayName]);

  const handleSave = () => {
    if (onUpdate && value.trim() && value.trim() !== displayName) {
      onUpdate(originalSpeaker, value.trim());
    }
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSave();
    } else if (e.key === 'Escape') {
      setValue(displayName);
      setIsEditing(false);
    }
  };

  if (isEditing) {
    return (
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onBlur={handleSave}
        onKeyDown={handleKeyDown}
        autoFocus
        className="bg-slate-700 text-cyan-300 font-semibold p-1 rounded-md w-full outline-none focus:ring-2 focus:ring-cyan-500"
      />
    );
  }

  return (
    <div 
      onClick={() => onUpdate && setIsEditing(true)} 
      className={`font-semibold p-1 rounded-md transition-colors ${onUpdate ? 'cursor-pointer hover:bg-slate-700/50' : ''} ${speakerMap[originalSpeaker] ? 'text-cyan-300' : 'text-slate-400'}`}
      title={onUpdate ? `Click to edit name for ${originalSpeaker}` : displayName}
    >
      {displayName}
    </div>
  );
};
