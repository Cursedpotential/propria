
import JSZip from 'jszip';
import { db } from './db.ts';
import { generateUUIDv7, sha256 } from './utils.ts';
import { IngestionResult, EvidenceSource, EvidenceChunk, AnalysisRecord } from '../types.ts';

const CHUNK_SIZE_LINES = 50;

async function processAndStoreFile(filename: string, content: string, contextKeywords: Record<string, string[]>): Promise<IngestionResult> {
  const fileHash = await sha256(content);

  const existingSource = await db.evidenceSources.where('file_hash').equals(fileHash).first();
  if (existingSource) {
    return { success: false, message: `File "${filename}" already ingested.` };
  }

  const sourceId = generateUUIDv7();
  const source: EvidenceSource = {
    id: sourceId,
    file_hash: fileHash,
    filename,
    imported_at: new Date(),
  };

  const chunkRegex = /###\s+Chunk\s+\d+\s+of\s+\d+/g;
  const hasChunkHeaders = chunkRegex.test(content);
  let chunksText: string[];

  if (hasChunkHeaders) {
    chunksText = content.split(chunkRegex).map(s => s.trim()).filter(Boolean);
  } else {
    const lines = content.split('\n');
    chunksText = [];
    for (let i = 0; i < lines.length; i += CHUNK_SIZE_LINES) {
      chunksText.push(lines.slice(i, i + CHUNK_SIZE_LINES).join('\n'));
    }
  }

  if (chunksText.length === 0) {
    return { success: false, message: `File "${filename}" contains no processable content.` };
  }

  const evidenceChunks: EvidenceChunk[] = [];
  const analysisRecords: AnalysisRecord[] = [];
  const chunkCount = chunksText.length;

  for (let i = 0; i < chunkCount; i++) {
    const raw_text = chunksText[i];
    const chunk_hash = await sha256(raw_text);
    const chunkId = generateUUIDv7();

    evidenceChunks.push({
      id: chunkId,
      source_id: sourceId,
      raw_text,
      chunk_hash,
      sequence_index: i + 1,
    });

    // Sanitize text: [Label](url) -> Label
    const clean_text = raw_text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '$1');

    // Pre-fill tags from context dictionary
    const tags: string[] = [];
    if (contextKeywords) {
        for (const [tag, keywords] of Object.entries(contextKeywords)) {
            if (keywords.some(keyword => new RegExp(`\\b${keyword}\\b`, 'i').test(clean_text))) {
                tags.push(tag);
            }
        }
    }

    analysisRecords.push({
      id: generateUUIDv7(),
      chunk_id: chunkId,
      short_id: chunkId.slice(-8),
      clean_text,
      entities: [],
      timeline_date: null,
      tags: tags,
      verification_status: 'Pending',
    });
  }

  await db.transaction('rw', db.evidenceSources, db.evidenceChunks, db.analysisRecords, async () => {
    await db.evidenceSources.add(source);
    await db.evidenceChunks.bulkAdd(evidenceChunks);
    await db.analysisRecords.bulkAdd(analysisRecords);
  });

  return { success: true, message: `Successfully ingested "${filename}" as ${chunkCount} chunks.`, sourceId, chunkCount };
}

export async function ingestFile(file: File, contextKeywords: Record<string, string[]>): Promise<IngestionResult> {
  if (file.type === 'application/zip' || file.name.endsWith('.zip')) {
    const zip = await JSZip.loadAsync(file);
    let totalChunks = 0;
    const results: string[] = [];

    for (const filename in zip.files) {
      if (!zip.files[filename].dir && (filename.endsWith('.md') || filename.endsWith('.txt'))) {
        const content = await zip.files[filename].async('string');
        const result = await processAndStoreFile(filename, content, contextKeywords);
        if (result.success && result.chunkCount) {
            totalChunks += result.chunkCount;
        }
        results.push(result.message);
      }
    }
    return { success: true, message: `ZIP processed. ${totalChunks} total chunks ingested. Details: ${results.join('; ')}` };
  } else if (file.type === 'text/markdown' || file.type === 'text/plain' || file.name.endsWith('.md') || file.name.endsWith('.txt')) {
    const content = await file.text();
    return processAndStoreFile(file.name, content, contextKeywords);
  } else {
    return { success: false, message: 'Unsupported file type. Please upload Markdown, TXT, or ZIP files.' };
  }
}
