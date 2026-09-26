
import React, { useState } from 'react';
import { StagedVideo } from '../types';
import { generateCypherScript } from '../utils/dataTransformer';

interface GraphExportProps {
  stagedVideos: StagedVideo[];
  speakerMap: Record<string, string>;
}

export const GraphExport: React.FC<GraphExportProps> = ({ stagedVideos, speakerMap }) => {
  const [statusMessage, setStatusMessage] = useState('');

  const handleExport = () => {
    const verifiedVideos = stagedVideos.filter(v => v.status === 'Verified');
    if (verifiedVideos.length === 0) {
      setStatusMessage('No verified videos to export.');
      return;
    }

    try {
      const script = generateCypherScript(verifiedVideos, speakerMap);
      const blob = new Blob([script], { type: 'application/cypher' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `graph_export_${new Date().toISOString()}.cypher`;
      a.click();
      URL.revokeObjectURL(url);
      setStatusMessage(`Successfully generated Cypher script for ${verifiedVideos.length} verified videos.`);
    } catch (e: any) {
      setStatusMessage(`Error generating script: ${e.message}`);
    }
  };

  const verifiedCount = stagedVideos.filter(v => v.status === 'Verified').length;

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold text-slate-100 mb-6">Neo4j Graph Export</h2>
      <p className="text-slate-300 mb-6">
        Generate a Cypher script to import your <span className="font-bold text-cyan-400">Verified</span> analysis data into a graph database like Neo4j. This script will create nodes for entities and events, and establish relationships between them.
      </p>
      <div className="mt-8">
        <button onClick={handleExport} disabled={verifiedCount === 0} className="w-full bg-teal-600 hover:bg-teal-500 text-white font-bold py-3 px-4 rounded-lg disabled:bg-slate-600 disabled:cursor-not-allowed flex items-center justify-center gap-2">
          Generate & Download Cypher Script
        </button>
      </div>
      {statusMessage && (
        <div className="mt-4 p-3 rounded-lg text-sm bg-slate-700/50 text-slate-200">
          {statusMessage}
        </div>
      )}
       <div className="mt-6 text-xs text-slate-400 bg-slate-700/50 p-3 rounded-md border border-slate-600">
            <p><span className="font-bold text-slate-300">How to use:</span> Open the downloaded <code className="text-xs">.cypher</code> file in your Neo4j Browser or other Cypher-compatible client and run the entire script.</p>
        </div>
    </div>
  );
};
