import React, { useState, useMemo } from 'react';
import { ContextPhase } from '../types';
import { Plus, Trash2, Calendar, Activity, ChevronDown, ChevronUp } from 'lucide-react';

interface ContextGridProps {
  phases: ContextPhase[];
  onUpdate: (phases: ContextPhase[]) => void;
}

const ContextGrid: React.FC<ContextGridProps> = ({ phases, onUpdate }) => {
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  
  // New Phase State
  const [newName, setNewName] = useState('');
  const [newStart, setNewStart] = useState('');
  const [newEnd, setNewEnd] = useState('');
  const [newCategory, setNewCategory] = useState<ContextPhase['category']>('Life Era');
  const [newBaseline, setNewBaseline] = useState('');
  const [newDesc, setNewDesc] = useState('');

  const sortedPhases = useMemo(() => {
    return [...phases].sort((a, b) => {
      const dateA = new Date(a.startDate || '1900-01-01').getTime();
      const dateB = new Date(b.startDate || '1900-01-01').getTime();
      return sortOrder === 'asc' ? dateA - dateB : dateB - dateA;
    });
  }, [phases, sortOrder]);

  const addPhase = () => {
    if (!newName || !newStart) return;
    const newPhase: ContextPhase = {
      id: crypto.randomUUID(),
      name: newName,
      startDate: newStart,
      endDate: newEnd || 'Present',
      category: newCategory,
      emotionalBaseline: newBaseline,
      description: newDesc,
      notes: ''
    };
    onUpdate([...phases, newPhase]);
    // Reset form
    setNewName('');
    setNewStart('');
    setNewEnd('');
    setNewBaseline('');
    setNewDesc('');
  };

  const deletePhase = (id: string) => {
    if (window.confirm("Remove this Life Phase? This may affect timeline grouping.")) {
      onUpdate(phases.filter(p => p.id !== id));
    }
  };

  const updatePhase = (id: string, field: keyof ContextPhase, value: string) => {
    onUpdate(phases.map(p => p.id === id ? { ...p, [field]: value } : p));
  };

  const getCategoryColor = (cat: ContextPhase['category']) => {
    switch(cat) {
      case 'Life Era': return 'text-emerald-400 bg-emerald-950/30 border-emerald-900';
      case 'Relationship': return 'text-rose-400 bg-rose-950/30 border-rose-900';
      case 'Professional': return 'text-sky-400 bg-sky-950/30 border-sky-900';
      case 'Legal': return 'text-amber-400 bg-amber-950/30 border-amber-900';
      default: return 'text-slate-400 bg-slate-800 border-slate-700';
    }
  };

  return (
    <div className="flex flex-col h-full bg-slate-950 p-4 lg:p-8">
      
      <div className="max-w-6xl mx-auto w-full space-y-6">
        
        {/* Header */}
        <div className="flex justify-between items-end border-b border-slate-800 pb-4">
          <div>
            <h2 className="text-2xl font-black text-white uppercase tracking-tighter flex items-center gap-2">
              <Calendar className="text-indigo-500" />
              Master Context Grid
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              Define the "Eras" of your life. This table is the Source of Truth for all AI analysis.
            </p>
          </div>
          <div className="flex gap-2">
            <button onClick={() => setSortOrder('asc')} className={`p-2 rounded ${sortOrder === 'asc' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-500'}`}><ChevronUp size={16} /></button>
            <button onClick={() => setSortOrder('desc')} className={`p-2 rounded ${sortOrder === 'desc' ? 'bg-indigo-600 text-white' : 'bg-slate-800 text-slate-500'}`}><ChevronDown size={16} /></button>
          </div>
        </div>

        {/* Input Row */}
        <div className="bg-slate-900 p-4 rounded-xl border border-slate-700 shadow-xl grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
          <div className="md:col-span-3">
            <label className="text-[9px] font-black uppercase text-slate-500 block mb-1">Phase Name</label>
            <input type="text" placeholder="e.g. The College Years" value={newName} onChange={e => setNewName(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white outline-none focus:border-indigo-500" />
          </div>
          <div className="md:col-span-2">
             <label className="text-[9px] font-black uppercase text-slate-500 block mb-1">Category</label>
             <select value={newCategory} onChange={e => setNewCategory(e.target.value as any)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white outline-none focus:border-indigo-500">
               <option value="Life Era">Life Era</option>
               <option value="Relationship">Relationship</option>
               <option value="Professional">Career</option>
               <option value="Medical">Medical</option>
               <option value="Legal">Legal</option>
               <option value="Geographic">Geographic</option>
             </select>
          </div>
          <div className="md:col-span-2">
             <label className="text-[9px] font-black uppercase text-slate-500 block mb-1">Start Date</label>
             <input type="text" placeholder="YYYY or YYYY-MM" value={newStart} onChange={e => setNewStart(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white outline-none focus:border-indigo-500" />
          </div>
          <div className="md:col-span-2">
             <label className="text-[9px] font-black uppercase text-slate-500 block mb-1">End Date</label>
             <input type="text" placeholder="YYYY or Present" value={newEnd} onChange={e => setNewEnd(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white outline-none focus:border-indigo-500" />
          </div>
          <div className="md:col-span-2">
             <label className="text-[9px] font-black uppercase text-slate-500 block mb-1">Emotional Baseline</label>
             <input type="text" placeholder="e.g. Stable, Manic" value={newBaseline} onChange={e => setNewBaseline(e.target.value)} className="w-full bg-slate-950 border border-slate-800 rounded p-2 text-xs text-white outline-none focus:border-indigo-500" />
          </div>
          <div className="md:col-span-1">
            <button onClick={addPhase} className="w-full bg-emerald-600 hover:bg-emerald-500 text-white p-2 rounded flex justify-center items-center shadow-lg shadow-emerald-500/20">
              <Plus size={16} />
            </button>
          </div>
        </div>

        {/* The Grid Table */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-2xl">
          <div className="grid grid-cols-12 bg-slate-950 p-3 border-b border-slate-800 text-[10px] font-black uppercase text-slate-500 tracking-widest">
            <div className="col-span-3">Phase Name</div>
            <div className="col-span-2">Type</div>
            <div className="col-span-2">Start</div>
            <div className="col-span-2">End</div>
            <div className="col-span-2">Baseline</div>
            <div className="col-span-1 text-center">Action</div>
          </div>
          
          <div className="divide-y divide-slate-800">
            {sortedPhases.length === 0 ? (
              <div className="p-8 text-center text-slate-600 text-xs italic">
                No phases defined. Start by adding your "Early Childhood" or a major "Relationship Phase".
              </div>
            ) : (
              sortedPhases.map(phase => (
                <div key={phase.id} className="grid grid-cols-12 p-1 group hover:bg-slate-800/50 transition-colors items-center">
                  
                  {/* Name */}
                  <div className="col-span-3 p-1">
                    <input 
                      type="text" 
                      value={phase.name} 
                      onChange={e => updatePhase(phase.id, 'name', e.target.value)}
                      className="w-full bg-transparent text-xs font-bold text-white outline-none border-b border-transparent focus:border-indigo-500" 
                    />
                  </div>

                  {/* Type */}
                  <div className="col-span-2 p-1">
                    <span className={`text-[9px] font-black uppercase px-2 py-0.5 rounded border ${getCategoryColor(phase.category)}`}>
                      {phase.category}
                    </span>
                  </div>

                  {/* Start */}
                  <div className="col-span-2 p-1">
                     <input 
                      type="text" 
                      value={phase.startDate} 
                      onChange={e => updatePhase(phase.id, 'startDate', e.target.value)}
                      className="w-full bg-transparent text-xs text-slate-300 outline-none border-b border-transparent focus:border-indigo-500 font-mono" 
                    />
                  </div>

                  {/* End */}
                  <div className="col-span-2 p-1">
                     <input 
                      type="text" 
                      value={phase.endDate} 
                      onChange={e => updatePhase(phase.id, 'endDate', e.target.value)}
                      className="w-full bg-transparent text-xs text-slate-300 outline-none border-b border-transparent focus:border-indigo-500 font-mono" 
                    />
                  </div>

                  {/* Baseline */}
                  <div className="col-span-2 p-1 flex items-center gap-2">
                    <Activity size={12} className="text-slate-600" />
                    <input 
                      type="text" 
                      value={phase.emotionalBaseline} 
                      onChange={e => updatePhase(phase.id, 'emotionalBaseline', e.target.value)}
                      className="w-full bg-transparent text-xs text-slate-400 outline-none border-b border-transparent focus:border-indigo-500 italic" 
                    />
                  </div>

                  {/* Action */}
                  <div className="col-span-1 p-1 flex justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                    <button onClick={() => deletePhase(phase.id)} className="text-slate-600 hover:text-red-400">
                      <Trash2 size={14} />
                    </button>
                  </div>

                </div>
              ))
            )}
          </div>
        </div>
        
      </div>
    </div>
  );
};

export default ContextGrid;