
import React, { useMemo } from 'react';
import { StagedVideo } from '../types';
import { transformToDbSchema } from '../utils/dataTransformer';

interface CaseDataViewerProps {
  stagedVideos: StagedVideo[];
  speakerMap: Record<string, string>;
}

export const CaseDataViewer: React.FC<CaseDataViewerProps> = ({ stagedVideos, speakerMap }) => {
  const caseData = useMemo(() => {
    const verifiedVideos = stagedVideos.filter(v => v.status === 'Verified');
    if (verifiedVideos.length === 0) {
      return null;
    }
    // We pass null for existingData to just see what this session will produce
    return transformToDbSchema(verifiedVideos, speakerMap, null);
  }, [stagedVideos, speakerMap]);

  return (
    <div className="h-full flex flex-col">
      <h2 className="text-2xl font-bold text-slate-100 mb-4">Case Data Viewer</h2>
      <p className="text-slate-300 mb-6">
        This is a read-only preview of the structured data generated from your <span className="font-bold text-cyan-400">Verified</span> videos. This is the data that will be merged or exported.
      </p>
      <div className="flex-grow bg-slate-900 rounded-md p-4 overflow-auto border border-slate-700">
        {caseData ? (
          <pre className="text-xs text-slate-200 whitespace-pre-wrap">
            {JSON.stringify(caseData, null, 2)}
          </pre>
        ) : (
          <div className="flex items-center justify-center h-full text-slate-400">
            <p>No verified videos to display. Go to the 'Review & Verify' tab to approve analyzed videos.</p>
          </div>
        )}
      </div>
    </div>
  );
};
