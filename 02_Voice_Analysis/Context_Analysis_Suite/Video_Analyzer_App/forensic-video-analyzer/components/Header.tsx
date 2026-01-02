
import React from 'react';
import { BrainIcon } from './icons/BrainIcon';

export const Header: React.FC = () => {
  return (
    <header className="bg-slate-800/80 backdrop-blur-sm sticky top-0 z-20 border-b border-slate-600/80">
      <div className="container mx-auto px-4 py-4 flex items-center gap-4">
        <BrainIcon className="w-10 h-10 text-cyan-400" />
        <div>
          <h1 className="text-2xl font-bold text-slate-100 tracking-tight">
            Forensic Video Analyzer
          </h1>
          <p className="text-sm text-slate-400">
            AI-Powered Visual Intelligence
          </p>
        </div>
      </div>
    </header>
  );
};
