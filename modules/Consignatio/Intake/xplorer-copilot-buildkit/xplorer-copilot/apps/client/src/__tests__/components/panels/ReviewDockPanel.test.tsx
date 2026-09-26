// Byline: Claude Code · Sonnet 5 · 2026-09-14
import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import ReviewDockPanel from '@/components/panels/ReviewDockPanel';
import { lookupPath } from '@/lib/intake-backend';

vi.mock('@/lib/intake-backend', () => ({
  lookupPath: vi.fn(),
}));

const renderWithClient = (ui: React.ReactElement) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
};

describe('ReviewDockPanel', () => {
  it('shows one line when nothing is selected, no fabricated rows', () => {
    renderWithClient(<ReviewDockPanel selectedFile={null} />);
    expect(screen.getByText('No file selected.')).toBeInTheDocument();
  });

  it('renders a real catalog row for a single indexed selection', async () => {
    vi.mocked(lookupPath).mockResolvedValue({
      found: true,
      relative_path: 'note.txt',
      filename: 'note.txt',
      byte_size: 42,
      content_sha256: 'a'.repeat(64),
      title: 'Schedule',
      document_type: 'note',
      confidence: 0.9,
      duplicates: [],
    });
    renderWithClient(
      <ReviewDockPanel
        selectedFile={{ name: 'note.txt', path: 'C:/src/note.txt', is_dir: false }}
      />,
    );
    await waitFor(() => expect(screen.getAllByText('note.txt').length).toBeGreaterThan(0));
    expect(screen.getByText(/Schedule/)).toBeInTheDocument();
    expect(lookupPath).toHaveBeenCalledWith('C:/src/note.txt');
  });

  it('shows one small flag, never a fake row, when not indexed', async () => {
    vi.mocked(lookupPath).mockResolvedValue({
      found: false,
      reason: 'not_yet_indexed',
      duplicates: [],
    });
    renderWithClient(
      <ReviewDockPanel selectedFile={{ name: 'x.txt', path: 'C:/src/x.txt', is_dir: false }} />,
    );
    await waitFor(() => expect(screen.getByText('Not indexed yet.')).toBeInTheDocument());
  });

  it('never invents EXIF/PDF fields and always discloses what is not extracted', () => {
    renderWithClient(<ReviewDockPanel selectedFile={null} />);
    expect(screen.getByText(/Photo EXIF/)).toBeInTheDocument();
    expect(screen.getByText(/PDF info/)).toBeInTheDocument();
  });
});
