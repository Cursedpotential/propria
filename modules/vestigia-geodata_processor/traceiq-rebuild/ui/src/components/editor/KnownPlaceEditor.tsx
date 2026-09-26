"use client";
import { useState, useCallback } from 'react';
import { TraceEvent, KnownPlace } from '@/types';
import { Flag, Eye, AlertTriangle, Hash, Save, Loader2, CheckCircle } from 'lucide-react';

const SUGGESTED_TAGS = ['home', 'work', 'relative', 'personal_interest', 'bar', 'medical', 'commercial', 'sensitive', 'watch'];

interface Props {
  event: TraceEvent;
  initial?: KnownPlace | null;
  onSaved?: (place: KnownPlace) => void;
}

export function KnownPlaceEditor({ event, initial, onSaved }: Props) {
  const [label, setLabel] = useState(initial?.label ?? event.place_id ?? '');
  const [tags, setTags] = useState<string[]>(initial?.tags ?? event.tags ?? []);
  const [tagInput, setTagInput] = useState('');
  const [isConcerning, setIsConcerning] = useState(initial?.is_concerning ?? false);
  const [isWatch, setIsWatch] = useState(initial?.is_watch ?? false);
  const [severity, setSeverity] = useState<number | ''>(initial?.severity ?? '');
  const [note, setNote] = useState(initial?.note ?? '');
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<'idle' | 'ok' | 'err'>('idle');

  const pushTag = useCallback((t: string) => {
    const clean = t.trim().toLowerCase();
    if (!clean || tags.includes(clean)) return;
    setTags([...tags, clean]);
  }, [tags]);

  const removeTag = (t: string) => setTags(tags.filter(x => x !== t));

  const save = async () => {
    if (!label.trim()) return;
    setSaving(true);
    setStatus('idle');
    try {
      const body: Record<string, unknown> = {
        label: label.trim(),
        tags,
        is_concerning: isConcerning,
        is_watch: isWatch,
        note: note.trim() || null,
      };
      if (severity !== '' && severity !== null) body.severity = severity;
      if (event.place_id) {
        body.place_id = event.place_id;
      } else {
        body.lat_r4 = event.lat_r4;
        body.lng_r4 = event.lng_r4;
      }
      const res = await fetch('/api/known-place', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok || data.error) throw new Error(data.error || 'save failed');
      setStatus('ok');
      onSaved?.(data as KnownPlace);
    } catch (e) {
      setStatus('err');
      // eslint-disable-next-line no-console
      console.error('KnownPlaceEditor save failed', e);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-3 text-xs">
      <div>
        <label className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-muted mb-1">
          <Flag className="w-3 h-3" /> Label
        </label>
        <input
          value={label}
          onChange={e => setLabel(e.target.value)}
          placeholder="Label this place"
          className="w-full px-2 py-1.5 bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal text-sm"
        />
      </div>

      <div>
        <label className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-muted mb-1">
          <Hash className="w-3 h-3" /> Tags
        </label>
        <div className="flex flex-wrap gap-1.5 mb-2">
          {tags.map(tag => (
            <span key={tag} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-surface-2 border border-border text-[10px]">
              {tag}
              <button onClick={() => removeTag(tag)} className="text-faint hover:text-critical">×</button>
            </span>
          ))}
        </div>
        <div className="flex flex-wrap gap-1.5 mb-2">
          {SUGGESTED_TAGS.map(tag => (
            <button
              key={tag}
              onClick={() => pushTag(tag)}
              disabled={tags.includes(tag)}
              className={`px-2 py-0.5 rounded-full border text-[10px] transition ${tags.includes(tag) ? 'bg-signal-soft border-signal text-signal' : 'bg-surface border-border text-muted hover:text-ink'}`}
            >
              + {tag}
            </button>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            value={tagInput}
            onChange={e => setTagInput(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); pushTag(tagInput); setTagInput(''); }}}
            placeholder="Add custom tag"
            className="flex-1 px-2 py-1 bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal"
          />
          <button
            onClick={() => { pushTag(tagInput); setTagInput(''); }}
            disabled={!tagInput.trim()}
            className="px-2 py-1 rounded-md border border-border bg-surface-2 text-muted hover:text-ink disabled:opacity-40"
          >Add</button>
        </div>
      </div>

      <div className="flex gap-2">
        <button
          onClick={() => setIsWatch(!isWatch)}
          className={`flex-1 flex items-center justify-center gap-1.5 px-2 py-1.5 rounded-md border transition ${isWatch ? 'bg-signal-soft border-signal text-signal' : 'bg-surface-2 border-border text-muted hover:text-ink'}`}
        >
          <Eye className="w-3.5 h-3.5" /> Watch
        </button>
        <button
          onClick={() => setIsConcerning(!isConcerning)}
          className={`flex-1 flex items-center justify-center gap-1.5 px-2 py-1.5 rounded-md border transition ${isConcerning ? 'bg-concerning/10 border-concerning text-concerning' : 'bg-surface-2 border-border text-muted hover:text-ink'}`}
        >
          <AlertTriangle className="w-3.5 h-3.5" /> Concerning
        </button>
      </div>

      <div>
        <label className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wide text-muted mb-1">
          Severity (1–5)
        </label>
        <input
          type="number"
          min={1}
          max={5}
          value={severity}
          onChange={e => {
            const v = e.target.value === '' ? '' : Math.max(1, Math.min(5, Number(e.target.value)));
            setSeverity(v);
          }}
          placeholder="optional"
          className="w-full px-2 py-1 bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal"
        />
      </div>

      <div>
        <label className="text-[10px] font-medium uppercase tracking-wide text-muted mb-1 block">Note</label>
        <textarea
          value={note}
          onChange={e => setNote(e.target.value)}
          rows={2}
          placeholder="Audit-friendly note"
          className="w-full px-2 py-1 bg-surface-2 border border-border rounded-md focus:outline-none focus:border-signal resize-none"
        />
      </div>

      <div className="flex items-center gap-2 pt-1">
        <button
          onClick={save}
          disabled={saving || !label.trim()}
          className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg bg-signal text-white text-xs font-medium hover:opacity-90 disabled:opacity-50"
        >
          {saving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : status === 'ok' ? <CheckCircle className="w-3.5 h-3.5" /> : <Save className="w-3.5 h-3.5" />}
          {saving ? 'Saving…' : status === 'ok' ? 'Saved' : 'Save to ref.known_place'}
        </button>
        {status === 'err' && <span className="text-critical text-[10px]">Save failed</span>}
      </div>

      <div className="text-[10px] text-faint">
        Key: {event.place_id ? `place_id=${event.place_id}` : `lat_r4=${event.lat_r4}, lng_r4=${event.lng_r4}`}
      </div>
    </div>
  );
}
