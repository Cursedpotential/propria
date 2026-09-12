import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FilesystemIndex } from '@xplorer/sdk';
import { TauriAPI } from '@/lib/tauri-api';
import { executeFileAction, parseFileActions } from '@/components/panels/chat-file-actions';
import { formatFilesystemSearchForAgent } from '@/lib/filesystem-index';

vi.mock('@xplorer/sdk', () => ({ FilesystemIndex: { search: vi.fn() } }));
vi.mock('@/lib/tauri-api', () => ({
  TauriAPI: { findFiles: vi.fn(), readDirectory: vi.fn(), readTextFile: vi.fn() },
}));
vi.mock('@/lib/utils', () => ({ formatFileSize: vi.fn() }));

const hit = {
  object_id: 'object-1',
  source_id: 'raw-bucket',
  source_path: 'V:\\raw\\takeout.zip',
  document_id: 'doc-1',
  chunk_id: 'chunk-1',
  filename: 'takeout.zip',
  text: 'export companion',
  score: 0.7,
};
const result = {
  query: 'missing export',
  backend: 'weaviate',
  collection: 'Intake',
  target_vector: 'text',
  coverage: 'unknown',
  hits: [hit],
};

describe('assistant combined-index search action', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubEnv('VITE_INTAKE_MODE', '1');
  });
  afterEach(() => vi.unstubAllEnvs());

  it('parses a search action and returns actual backend evidence to the existing agent loop', async () => {
    const parsed = parseFileActions(
      '```json\n{"action":"search_files","path":"","query":"missing export","mode":"keyword"}\n```',
    );
    expect(parsed.actions).toHaveLength(1);
    vi.mocked(FilesystemIndex.search).mockResolvedValueOnce(result);
    const output = JSON.parse((await executeFileAction(parsed.actions[0]))!);
    expect(FilesystemIndex.search).toHaveBeenCalledWith('missing export', 'keyword', 20);
    expect(output.hits[0].source_id).toBe('raw-bucket');
    expect(output.hits[0].source_path).toBe(hit.source_path);
    expect(output.coverage).toBe('unknown');
    expect(TauriAPI.findFiles).not.toHaveBeenCalled();
    expect(TauriAPI.readDirectory).not.toHaveBeenCalled();
    expect(TauriAPI.readTextFile).not.toHaveBeenCalled();
  });

  it('propagates failure rather than silently scanning the corpus', async () => {
    vi.mocked(FilesystemIndex.search).mockRejectedValueOnce({
      code: 'not_configured',
      message: 'Index service unavailable',
    });
    await expect(
      executeFileAction({ action: 'search_files', path: 'V:\\', query: 'missing export' }),
    ).rejects.toThrow('Index service unavailable');
    expect(TauriAPI.findFiles).not.toHaveBeenCalled();
  });

  it('rejects invalid modes and non-string paths', () => {
    expect(
      parseFileActions(
        '```json\n{"action":"search_files","path":"","query":"export","mode":"recursive"}\n```',
      ).actions,
    ).toHaveLength(0);
    expect(
      parseFileActions('```json\n{"action":"search_files","path":42,"query":"export"}\n```')
        .actions,
    ).toHaveLength(0);
  });

  it('bounds snippets and count without changing source identity', () => {
    const output = JSON.parse(
      formatFilesystemSearchForAgent({
        ...result,
        hits: Array.from({ length: 25 }, () => ({ ...hit, text: 'x'.repeat(1000) })),
      }),
    );
    expect(output.returned_hits).toBe(25);
    expect(output.shown_hits).toBe(20);
    expect(output.hits).toHaveLength(20);
    expect(output.hits[0].text).toHaveLength(800);
    expect(output.hits[0].text_truncated).toBe(true);
    expect(output.hits[0].object_id).toBe(hit.object_id);
  });

  it('leaves legacy Xplorer search routing alone outside Intake mode', async () => {
    vi.stubEnv('VITE_INTAKE_MODE', '0');
    vi.mocked(TauriAPI.findFiles).mockResolvedValueOnce(['E:\\a.txt']);
    const output = await executeFileAction({
      action: 'search_files',
      path: 'E:\\',
      query: '*.txt',
    });
    expect(output).toContain('E:\\a.txt');
    expect(FilesystemIndex.search).not.toHaveBeenCalled();
  });
});
