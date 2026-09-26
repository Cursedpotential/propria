
import Dexie, { Table } from 'dexie';
import { EvidenceSource, EvidenceChunk, AnalysisRecord } from '../types.ts';

export class ForensicDB extends Dexie {
  evidenceSources!: Table<EvidenceSource, string>;
  evidenceChunks!: Table<EvidenceChunk, string>;
  analysisRecords!: Table<AnalysisRecord, string>;

  constructor() {
    super('ForensicDataRefineryDB');
    this.version(1).stores({
      evidenceSources: 'id, &file_hash', // id is PK, file_hash is unique index
      evidenceChunks: 'id, source_id, chunk_hash', // id is PK
      analysisRecords: 'id, chunk_id, verification_status, *tags', // id is PK
    });
  }
}

export const db = new ForensicDB();
