import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useChatActions } from '@/components/panels/use-chat-actions';
import { executeFileAction, type PendingFileAction } from '@/components/panels/chat-file-actions';

vi.mock('@/components/panels/chat-file-actions', () => ({
  executeFileAction: vi.fn(), executeRunCommand: vi.fn(), captureFileForUndo: vi.fn(), undoFileAction: vi.fn(),
  isAlwaysAllowed: () => false, isReadOnlyAction: () => true, isAlwaysAskAction: () => false, setAlwaysAllowed: vi.fn(),
}));
vi.mock('@/components/panels/chat-audit-log', () => ({ logAction: vi.fn() }));
vi.mock('@/components/panels/chat-security-rules', () => ({ checkAndLogSecurity: () => ({ allowed: true }) }));

const action = (id: number): PendingFileAction => ({ id: String(id), status: 'pending', action: { action: 'search_files', path: '', query: `query ${id}` } });

describe('bounded assistant index actions', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubEnv('VITE_INTAKE_MODE', '1'); });
  afterEach(() => vi.unstubAllEnvs());

  it('feeds actual results to the existing follow-up loop and does not execute excess searches', async () => {
    vi.mocked(executeFileAction).mockResolvedValue('actual indexed result');
    const { result } = renderHook(() => useChatActions({ current: [] }, vi.fn(), vi.fn()));
    let output: Awaited<ReturnType<typeof result.current.autoExecuteActions>> | undefined;
    await act(async () => { output = await result.current.autoExecuteActions(0, [action(1), action(2), action(3), action(4)]); });
    expect(executeFileAction).toHaveBeenCalledTimes(3);
    expect(output?.readOnlyResults.slice(0, 3)).toEqual(['actual indexed result', 'actual indexed result', 'actual indexed result']);
    expect(output?.readOnlyResults[3]).toContain('not executed');
    expect(output?.hasRemainingPending).toBe(false);
  });

  it('returns failure context rather than simulated search success', async () => {
    vi.mocked(executeFileAction).mockRejectedValueOnce(new Error('Index disconnected'));
    const { result } = renderHook(() => useChatActions({ current: [] }, vi.fn(), vi.fn()));
    let output: Awaited<ReturnType<typeof result.current.autoExecuteActions>> | undefined;
    await act(async () => { output = await result.current.autoExecuteActions(0, [action(1)]); });
    expect(output?.readOnlyResults).toEqual(['Error executing search_files: Index disconnected']);
  });
});
