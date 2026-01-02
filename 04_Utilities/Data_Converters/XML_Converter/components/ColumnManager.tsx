import React from 'react';

interface Props {
  allKeys: string[];
  selectedKeys: string[];
  onChange: (keys: string[]) => void;
}

export const ColumnManager: React.FC<Props> = ({ allKeys, selectedKeys, onChange }) => {
  
  const toggleKey = (key: string) => {
    if (selectedKeys.includes(key)) {
      onChange(selectedKeys.filter(k => k !== key));
    } else {
      onChange([...selectedKeys, key]);
    }
  };

  const moveKey = (index: number, direction: 'up' | 'down') => {
    const newKeys = [...selectedKeys];
    if (direction === 'up' && index > 0) {
      [newKeys[index], newKeys[index - 1]] = [newKeys[index - 1], newKeys[index]];
    } else if (direction === 'down' && index < newKeys.length - 1) {
      [newKeys[index], newKeys[index + 1]] = [newKeys[index + 1], newKeys[index]];
    }
    onChange(newKeys);
  };

  return (
    <div className="bg-slate-800 rounded-lg border border-slate-700 shadow-lg p-6">
      <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-indigo-400" viewBox="0 0 20 20" fill="currentColor">
          <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
        </svg>
        Column Builder
      </h3>
      <p className="text-sm text-slate-400 mb-4">Select and reorder columns for CSV and SQL export.</p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Selected Columns (Ordered) */}
        <div>
          <h4 className="text-sm font-bold text-slate-300 uppercase mb-2">Export Order</h4>
          <div className="bg-slate-900 rounded-md border border-slate-700 p-2 max-h-96 overflow-y-auto space-y-1">
            {selectedKeys.map((key, idx) => (
              <div key={key} className="flex items-center justify-between bg-slate-800 p-2 rounded border border-slate-700">
                <span className="text-sm font-mono text-indigo-300">{idx + 1}. {key}</span>
                <div className="flex items-center gap-1">
                  <button 
                    onClick={() => moveKey(idx, 'up')}
                    disabled={idx === 0}
                    className="p-1 hover:bg-slate-700 rounded text-slate-400 disabled:opacity-30"
                  >
                    ↑
                  </button>
                  <button 
                    onClick={() => moveKey(idx, 'down')}
                    disabled={idx === selectedKeys.length - 1}
                    className="p-1 hover:bg-slate-700 rounded text-slate-400 disabled:opacity-30"
                  >
                    ↓
                  </button>
                  <button 
                    onClick={() => toggleKey(key)}
                    className="p-1 hover:bg-red-900/50 rounded text-red-400 ml-2"
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
            {selectedKeys.length === 0 && <div className="text-slate-500 text-sm italic p-2">No columns selected</div>}
          </div>
        </div>

        {/* Available Columns */}
        <div>
          <h4 className="text-sm font-bold text-slate-300 uppercase mb-2">Available Fields</h4>
          <div className="bg-slate-900 rounded-md border border-slate-700 p-2 max-h-96 overflow-y-auto">
             <div className="flex flex-wrap gap-2">
               {allKeys.filter(k => !selectedKeys.includes(k)).map(key => (
                 <button
                   key={key}
                   onClick={() => toggleKey(key)}
                   className="px-3 py-1 bg-slate-800 hover:bg-slate-700 border border-slate-600 rounded text-sm text-slate-300 transition-colors"
                 >
                   + {key}
                 </button>
               ))}
               {allKeys.filter(k => !selectedKeys.includes(k)).length === 0 && (
                   <div className="text-slate-500 text-sm italic p-2">All available fields selected</div>
               )}
             </div>
          </div>
        </div>
      </div>
    </div>
  );
};