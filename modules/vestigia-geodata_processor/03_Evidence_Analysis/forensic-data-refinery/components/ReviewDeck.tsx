
import React from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { db } from '../lib/db.ts';
import { AnalysisRecord } from '../types.ts';
import { ReviewTable } from './ReviewTable.tsx';
import { SplitInspector } from './SplitInspector.tsx';
import { Inbox } from 'lucide-react';

interface ReviewDeckProps {
  selectedRecord: AnalysisRecord | null;
  setSelectedRecord: (record: AnalysisRecord | null) => void;
  addNotification: (message: string, type?: 'success' | 'error' | 'info') => void;
}

export function ReviewDeck({ selectedRecord, setSelectedRecord, addNotification }: ReviewDeckProps) {
  const records = useLiveQuery(() => db.analysisRecords.orderBy('id').toArray(), []);

  if (records === undefined) {
    return <div className="flex justify-center items-center h-full"><p>Loading records...</p></div>;
  }

  if (records.length === 0) {
    return (
      <div className="text-center text-gray-500 mt-20">
        <Inbox className="mx-auto h-12 w-12" />
        <h3 className="mt-2 text-sm font-medium text-gray-300">No Data Ingested</h3>
        <p className="mt-1 text-sm">Use the "Ingest Data" button to upload a file.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full space-y-4">
      <div className="flex-grow min-h-0">
        <ReviewTable
          records={records}
          selectedRecordId={selectedRecord?.id || null}
          onRecordSelect={setSelectedRecord}
          addNotification={addNotification}
        />
      </div>
      {selectedRecord && (
        <div className="flex-shrink-0">
          <SplitInspector
            record={selectedRecord}
            onClose={() => setSelectedRecord(null)}
            addNotification={addNotification}
          />
        </div>
      )}
    </div>
  );
}
