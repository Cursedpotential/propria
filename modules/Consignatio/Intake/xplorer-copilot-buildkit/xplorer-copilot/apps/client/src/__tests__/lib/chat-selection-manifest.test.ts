import { describe, expect, it, vi } from 'vitest';
import { buildSelectionManifest } from '@/components/panels/chat-context-helpers';
import { buildFileContext } from '@/components/panels/use-chat-send';
import { TauriAPI } from '@/lib/tauri-api';

// Keep this unit test independent of extension registries and chat UI startup.
vi.mock('@/components/panels/chat-file-actions', () => ({
  basename: (p: string) => p.split(/[/\\]/).pop() ?? p,
}));
vi.mock('@/components/panels/chat-slash-commands', () => ({
  matchSlashCommand: vi.fn(),
  SLASH_COMMANDS: [],
  LANG_EXTENSIONS: {},
}));
vi.mock('@/components/panels/chat-slash-commands-extra', () => ({
  matchExtraSlashCommand: vi.fn(),
}));
vi.mock('@/components/panels/chat-special-commands', () => ({
  handleSpecialSlashCommand: vi.fn(),
}));
vi.mock('@/components/panels/chat-action-templates', () => ({
  handleTemplateSlashCommand: vi.fn(),
}));
vi.mock('@/components/panels/chat-extension-awareness', () => ({
  buildMarketplaceSuggestionText: vi.fn(),
}));

vi.mock('@/lib/tauri-api', () => ({
  TauriAPI: {
    readTextFile: vi.fn(),
    readBinaryFile: vi.fn(),
    extractDocumentText: vi.fn(),
    readDirectory: vi.fn(),
  },
}));

describe('selection metadata context', () => {
  it('keeps directories, zero sizes, exact paths and unknown cross-tab metadata', () => {
    const result = buildSelectionManifest([
      { name: 'Takeout', path: 'V:\\raw\\Takeout', is_dir: true },
      { name: 'empty.txt', path: 'E:\\empty.txt', is_dir: false, size: 0 },
      { name: 'Other.tab', path: 'Y:\\raw\\Other.tab', is_dir: false, metadata_known: false },
    ]);
    expect(result).toEqual([
      {
        name: 'Takeout',
        path: 'V:\\raw\\Takeout',
        file_type: 'directory',
        size: undefined,
        metadataOnly: true,
      },
      { name: 'empty.txt', path: 'E:\\empty.txt', file_type: 'file', size: 0, metadataOnly: true },
      {
        name: 'Other.tab',
        path: 'Y:\\raw\\Other.tab',
        file_type: 'unknown',
        size: undefined,
        metadataOnly: true,
      },
    ]);
  });

  it('does not read, extract, list, compare or hydrate selected content', async () => {
    const result = await buildFileContext(
      [
        { name: 'one.jpg', path: 'V:\\one.jpg', is_dir: false, size: 10_000_000 },
        { name: 'two.pdf', path: 'Y:\\two.pdf', is_dir: false },
      ],
      'compare these files',
    );
    expect(result.fileContexts).toHaveLength(2);
    expect(result.compareContext).toBeNull();
    expect(result.imageContexts).toEqual([]);
    expect(TauriAPI.readTextFile).not.toHaveBeenCalled();
    expect(TauriAPI.readBinaryFile).not.toHaveBeenCalled();
    expect(TauriAPI.extractDocumentText).not.toHaveBeenCalled();
    expect(TauriAPI.readDirectory).not.toHaveBeenCalled();
  });

  it('keeps the entire group, not just the first five files', async () => {
    const files = Array.from({ length: 700 }, (_, i) => ({
      name: `${i}`,
      path: `V:\\${i}`,
      is_dir: i % 2 === 0,
    }));
    const result = await buildFileContext(files, 'these belong together');
    expect(result.fileContexts).toHaveLength(700);
    expect(result.fileContexts[699].path).toBe('V:\\699');
  });

  it('does not convert invalid sizes into known zero values', () => {
    for (const size of [-1, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1]) {
      expect(
        buildSelectionManifest([{ name: 'x', path: 'x', is_dir: false, size }])[0].size,
      ).toBeUndefined();
    }
  });
});
