import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useChatContext } from '@/components/panels/use-chat-context';
import type { XplorerState } from '@/components/panels/chat-context-helpers';

vi.mock('@/lib/tauri-api', () => ({ TauriAPI: {} }));
vi.mock('@/components/panels/chat-file-actions', () => ({ basename: (path: string) => path }));

const stateWindow = window as unknown as { __xplorer_state__?: XplorerState };
afterEach(() => {
  vi.useRealTimers();
  delete stateWindow.__xplorer_state__;
});

describe('retained chat context visibility', () => {
  it('pauses hidden polling and refreshes selection immediately when reopened', () => {
    vi.useFakeTimers();
    const initial = { name: 'first', path: 'V:\\first', is_dir: true };
    const later = { name: 'second', path: 'Y:\\second', is_dir: false, size: 0 };
    stateWindow.__xplorer_state__ = { currentPath: 'V:\\', selectedFiles: [initial] };
    const { result, rerender } = renderHook(({ active }) => useChatContext(active), {
      initialProps: { active: true },
    });
    expect(result.current.selectedFiles).toEqual([initial]);
    expect(vi.getTimerCount()).toBe(1);
    rerender({ active: false });
    expect(vi.getTimerCount()).toBe(0);
    stateWindow.__xplorer_state__ = { currentPath: 'Y:\\', selectedFiles: [later] };
    act(() => window.dispatchEvent(new Event('xplorer-state-change')));
    expect(result.current.selectedFiles).toEqual([initial]);
    rerender({ active: true });
    expect(result.current.currentPath).toBe('Y:\\');
    expect(result.current.selectedFiles).toEqual([later]);
  });
});
