import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import IntakeFilesystemSearchPanel from '@/components/explorer/IntakeFilesystemSearchPanel';
import { searchFilesystemIndex, addSearchHitsToSelection } from '@/lib/filesystem-index';
import { TauriAPI } from '@/lib/tauri-api';

vi.mock('@/lib/filesystem-index', () => ({
  searchFilesystemIndex: vi.fn(),
  addSearchHitsToSelection: vi.fn(),
  filesystemParentPath: () => 'V:\\raw\\Export',
}));

vi.mock('@/lib/tauri-api', () => ({
  TauriAPI: { grepSearch: vi.fn() },
}));

const hit = {
  object_id: 'obj',
  source_id: 'raw-bucket',
  source_path: 'V:\\raw\\Export\\messages.json',
  document_id: 'doc',
  chunk_id: 'chunk',
  filename: 'messages.json',
  text: 'Snippet from the actual index',
  score: 0.82,
};

describe('Intake filesystem search sidebar', () => {
  beforeEach(() => vi.clearAllMocks());

  it('searches only on submission, then navigates and adds checked results to selection', async () => {
    vi.mocked(searchFilesystemIndex).mockResolvedValueOnce({
      query: 'export',
      backend: 'weaviate',
      collection: 'IntakeTest',
      target_vector: 'text',
      coverage: 'unknown',
      hits: [hit],
    });
    const navigate = vi.fn();
    render(<IntakeFilesystemSearchPanel navigateToPath={navigate} />);
    expect(searchFilesystemIndex).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'export' } });
    expect(searchFilesystemIndex).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'search' }));
    expect(await screen.findByText('messages.json')).toBeTruthy();
    expect(searchFilesystemIndex).toHaveBeenCalledWith('export', 'hybrid');
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'addSelection' }));
    expect(addSearchHitsToSelection).toHaveBeenCalledWith([hit]);
    fireEvent.click(screen.getByRole('button', { name: 'openLocation' }));
    expect(navigate).toHaveBeenCalledWith('V:\\raw\\Export');
  });

  it('shows unavailable rather than invented empty results when disconnected', async () => {
    vi.mocked(searchFilesystemIndex).mockRejectedValueOnce(new Error('not configured'));
    render(<IntakeFilesystemSearchPanel navigateToPath={vi.fn()} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'export' } });
    fireEvent.click(screen.getByRole('button', { name: 'search' }));
    await waitFor(() => expect(screen.getByRole('alert').textContent).toContain('not configured'));
    expect(screen.queryByText('noResults')).toBeNull();
  });

  it('runs the local rg method under the active pane root and shows provenance per row', async () => {
    vi.mocked(TauriAPI.grepSearch).mockResolvedValueOnce([
      {
        file: 'V:\\raw\\Export\\notes.md',
        line: 12,
        content: 'missing export line',
        filename: 'notes.md',
      },
    ]);
    render(
      <IntakeFilesystemSearchPanel navigateToPath={vi.fn()} activePaneRoot={'V:\\raw\\Export'} />,
    );
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'export' } });
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'rg' } });
    fireEvent.click(screen.getByRole('button', { name: 'search' }));
    expect(await screen.findByText('notes.md')).toBeTruthy();
    expect(TauriAPI.grepSearch).toHaveBeenCalledWith('export', 'V:\\raw\\Export', 100);
    expect(screen.getByText(/rg · V:\\raw\\Export · rg-local/)).toBeTruthy();
  });

  it('disables rg and flags it when no active pane root is known', () => {
    render(<IntakeFilesystemSearchPanel navigateToPath={vi.fn()} />);
    const rgOption = screen.getByRole('option', { name: 'rg' }) as HTMLOptionElement;
    expect(rgOption.disabled).toBe(true);
  });
});
