
import React from 'react';
import { LoadingPhase } from '../types';

interface LoaderProps {
  phase: LoadingPhase;
}

const phaseMessages: Record<LoadingPhase, string> = {
  idle: 'Waiting to start...',
  initializing: 'Initializing analysis engine...',
  extractingFrames: 'Extracting keyframes from video...',
  extractingAudio: 'Extracting audio from video...',
  analyzing: 'AI is performing basic analysis...',
  advancedAnalysis: 'AI is performing advanced forensic analysis...',
  compilingReport: 'Compiling report...',
  done: 'Analysis complete!',
};

export const Loader: React.FC<LoaderProps> = ({ phase }) => {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center text-slate-400">
      <div className="w-16 h-16 border-4 border-cyan-400 border-t-transparent rounded-full animate-spin mb-6"></div>
      <h3 className="text-lg font-semibold text-slate-300">Analysis in Progress</h3>
      <p className="mt-2 transition-all duration-300">{phaseMessages[phase]}</p>
    </div>
  );
};
