"use client";
import { useWorkspace } from '@/lib/WorkspaceContext';
import { getAdapter } from '@/mock/adapter';
import { useState, FormEvent } from 'react';
import { Send, Bot, Receipt, User, Sparkles } from 'lucide-react';

interface Msg {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  receipts?: string[];
  loading?: boolean;
}

export function ChatPane() {
  const { mode, filters, selectedEvent, setSelectedEvent } = useWorkspace();
  const [msgs, setMsgs] = useState<Msg[]>([
    { id: 'intro', role: 'assistant', content: "Ask me about the timeline. I'll cite rows as receipts." },
  ]);
  const [input, setInput] = useState('');

  const send = async (e?: FormEvent) => {
    e?.preventDefault();
    if (!input.trim()) return;
    const q = input.trim();
    setInput('');
    const userMsg: Msg = { id: crypto.randomUUID(), role: 'user', content: q };
    const pending: Msg = { id: crypto.randomUUID(), role: 'assistant', content: '', loading: true };
    setMsgs(prev => [...prev, userMsg, pending]);
    const res = await getAdapter().askAgent(q, filters);
    setMsgs(prev => prev.map(m => m.id === pending.id ? { ...m, content: res.answer, receipts: res.receipts, loading: false } : m));
  };

  return (
    <div className="flex flex-col h-full border-t border-border bg-surface">
      <div className="flex items-center justify-between px-4 py-2 border-b border-border">
        <div className="flex items-center gap-2 text-xs font-medium text-muted"><Bot className="w-4 h-4" /> Assistant</div>
        <span className={`text-[10px] px-2 py-0.5 rounded-full border ${mode === 'agent' ? 'bg-signal text-white border-signal' : 'bg-surface-2 text-muted border-border'}`}>{mode === 'agent' ? 'Agent mode' : 'Manual mode'}</span>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {msgs.map(m => (
          <div key={m.id} className={`flex gap-2 ${m.role === 'user' ? 'flex-row-reverse' : ''}`}>
            <div className={`w-6 h-6 rounded-full flex items-center justify-center shrink-0 ${m.role === 'user' ? 'bg-surface-2 text-muted' : 'bg-signal-soft text-signal'}`}>
              {m.role === 'user' ? <User className="w-3.5 h-3.5" /> : <Sparkles className="w-3.5 h-3.5" />}
            </div>
            <div className={`max-w-[85%] px-3 py-2 rounded-xl text-xs border ${m.role === 'user' ? 'bg-surface-2 border-border text-ink rounded-br-none' : 'bg-surface text-ink border-border rounded-bl-none'}`}>
              {m.loading ? <span className="animate-pulse">Thinking…</span> : m.content}
              {m.receipts && m.receipts.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {m.receipts.map(rid => (
                    <button
                      key={rid}
                      onClick={() => getAdapter().getEvent(rid).then(e => e && setSelectedEvent(e))}
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-signal-soft border border-signal/20 text-signal text-[10px] hover:bg-signal hover:text-white transition"
                    ><Receipt className="w-3 h-3" />{rid.slice(0, 8)}…</button>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      <form onSubmit={send} className="p-3 border-t border-border flex gap-2">
        <input
          value={input}
          onChange={e => setInput(e.target.value)}
          placeholder={mode === 'agent' ? 'Ask the agent…' : 'Chat disabled in manual mode'}
          disabled={mode !== 'agent'}
          className="flex-1 px-3 py-2 text-xs bg-surface-2 border border-border rounded-lg focus:outline-none focus:border-signal disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={mode !== 'agent' || !input.trim()}
          className="px-3 py-2 rounded-lg bg-signal text-white disabled:opacity-50"
        ><Send className="w-4 h-4" /></button>
      </form>
    </div>
  );
}
