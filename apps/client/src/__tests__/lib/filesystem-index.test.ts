import { describe, expect, it, vi } from 'vitest';
import { FilesystemIndex } from '@xplorer/sdk';
import {
  addSearchHitsToSelection,
  filesystemParentPath,
  parseFilesystemSearch,
  searchFilesystemIndex,
} from '@/lib/filesystem-index';

vi.mock('@xplorer/sdk', () => ({ FilesystemIndex: { search: vi.fn() } }));
const hit = {
  object_id: 'obj1',
  source_id: 'r2-raw',
  source_path: 'V:\\raw\\Export\\message.json',
  document_id: 'd1',
  chunk_id: 'c1',
  filename: 'message.json',
  text: 'indexed text',
  score: 0.83,
};
const response = {
  query: 'export',
  backend: 'weaviate',
  collection: 'IntakeTest',
  target_vector: 'text',
  coverage: 'unknown',
  hits: [hit],
};

describe('filesystem index adapter', () => {
  it('preserves real source identity, score and unknown coverage', () => {
    expect(parseFilesystemSearch(response)).toEqual(response);
  });
  it('does not fabricate results from broken backend responses', () => {
    expect(() => parseFilesystemSearch({ hits: [] })).toThrow('invalid response');
    expect(() => parseFilesystemSearch({ ...response, hits: [{ ...hit, score: NaN }] })).toThrow(
      'invalid result',
    );
  });
  it('calls the configured native bridge only for an explicit valid search', async () => {
    vi.mocked(FilesystemIndex.search).mockResolvedValueOnce(response);
    await expect(searchFilesystemIndex(' export ', 'keyword')).resolves.toEqual(response);
    expect(FilesystemIndex.search).toHaveBeenCalledWith('export', 'keyword', 20);
    vi.mocked(FilesystemIndex.search).mockClear();
    await expect(searchFilesystemIndex('', 'hybrid')).rejects.toThrow('1–4096');
    expect(FilesystemIndex.search).not.toHaveBeenCalled();
  });
  it('surfaces structured native unavailability errors', async () => {
    vi.mocked(FilesystemIndex.search).mockRejectedValueOnce({
      code: 'not_configured',
      message: 'Configure INTAKE_FILESYSTEM_API_URL',
      status: null,
    });
    await expect(searchFilesystemIndex('export', 'hybrid')).rejects.toThrow(
      'Configure INTAKE_FILESYSTEM_API_URL',
    );
  });
  it('navigates filesystem parents without guessing remote mappings', () => {
    expect(filesystemParentPath(hit.source_path)).toBe('V:\\raw\\Export');
    expect(filesystemParentPath('E:\\file.txt')).toBe('E:\\');
    expect(filesystemParentPath('s3://raw/file.txt')).toBeNull();
    expect(filesystemParentPath('relative/file.txt')).toBeNull();
  });
  it('adds deduplicated paths to chat selection without reading files', () => {
    const listener = vi.fn();
    window.addEventListener('intake-select-search-paths', listener);
    addSearchHitsToSelection([hit, hit]);
    expect(listener.mock.calls[0][0].detail).toEqual([hit.source_path]);
    window.removeEventListener('intake-select-search-paths', listener);
  });
});
