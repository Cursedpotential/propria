"use client";
import { useWorkspace } from '@/lib/WorkspaceContext';
import { useTheme } from '@/lib/useTheme';
import { TabKey } from '@/types';
import { Activity, BarChart3, FileSpreadsheet, Download, Settings, Moon, Sun, Search } from 'lucide-react';

const tabs: { key: TabKey; label: string; icon: any }[] = [
  { key: 'explore', label: 'Explore', icon: Search },
  { key: 'analytics', label: 'Analytics', icon: BarChart3 },
  { key: 'tables', label: 'Tables', icon: FileSpreadsheet },
  { key: 'export', label: 'Export', icon: Download },
  { key: 'config', label: 'Config', icon: Settings },
];

export function AppTopBar() {
  const { tab, setTab, mode, setMode } = useWorkspace();
  const { theme, toggle } = useTheme();
  return (
    <header className="flex items-center justify-between h-14 px-4 border-b border-border bg-surface shrink-0">
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5 text-signal" />
          <span className="font-semibold tracking-tight">TraceIQ</span>
        </div>
        <span className="text-xs px-2 py-0.5 rounded-full bg-surface-2 text-muted border border-border">Subject K</span>
      </div>

      <nav className="hidden md:flex items-center gap-1">
        {tabs.map(t => {
          const Icon = t.icon;
          const active = tab === t.key;
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg transition-colors ${active ? 'bg-signal-soft text-signal' : 'text-muted hover:text-ink hover:bg-surface-2'}`}
            >
              <Icon className="w-4 h-4" />
              {t.label}
            </button>
          );
        })}
      </nav>

      <div className="flex items-center gap-3">
        <div className="flex items-center bg-surface-2 rounded-lg p-0.5 border border-border">
          <button
            onClick={() => setMode('manual')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition ${mode === 'manual' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'}`}
          >Manual</button>
          <button
            onClick={() => setMode('agent')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition ${mode === 'agent' ? 'bg-signal text-white shadow-sm' : 'text-muted hover:text-ink'}`}
          >Agent</button>
        </div>
        <button
          onClick={toggle}
          className="p-2 rounded-lg border border-border bg-surface text-muted hover:text-ink"
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
        </button>
      </div>
    </header>
  );
}
