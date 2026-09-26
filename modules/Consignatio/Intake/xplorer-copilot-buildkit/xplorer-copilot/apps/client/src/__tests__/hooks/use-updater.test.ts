import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { check } from '@tauri-apps/plugin-updater';
import { relaunch } from '@tauri-apps/plugin-process';
import useUpdater from '@/hooks/use-updater';

vi.mock('@tauri-apps/plugin-updater', () => ({ check: vi.fn() }));
vi.mock('@tauri-apps/plugin-process', () => ({ relaunch: vi.fn() }));

describe('useUpdater fork isolation', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(check).mockReset();
    vi.mocked(check).mockResolvedValue(null);
    vi.mocked(relaunch).mockReset();
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.unstubAllEnvs();
  });

  it('does not schedule or perform upstream checks in Intake', async () => {
    vi.stubEnv('VITE_INTAKE_MODE', '1');
    const { result } = renderHook(() => useUpdater());
    expect(vi.getTimerCount()).toBe(0);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(8 * 60 * 60 * 1000);
      expect(await result.current.checkForUpdate()).toBeNull();
    });
    expect(check).not.toHaveBeenCalled();
    expect(result.current.status.available).toBe(false);
  });

  it('blocks direct install requests in Intake without checking or relaunching', async () => {
    vi.stubEnv('VITE_INTAKE_MODE', '1');
    const { result } = renderHook(() => useUpdater());
    await act(async () => result.current.installUpdate());
    expect(check).not.toHaveBeenCalled();
    expect(relaunch).not.toHaveBeenCalled();
    expect(result.current.status.downloading).toBe(false);
  });

  it('preserves legacy startup and periodic checks and cleans up timers', async () => {
    vi.stubEnv('VITE_INTAKE_MODE', '0');
    const { unmount } = renderHook(() => useUpdater());
    expect(vi.getTimerCount()).toBe(2);
    await act(async () => vi.advanceTimersByTimeAsync(5000));
    expect(check).toHaveBeenCalledTimes(1);
    await act(async () => vi.advanceTimersByTimeAsync(4 * 60 * 60 * 1000));
    expect(check).toHaveBeenCalledTimes(2);
    unmount();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('preserves legacy manual checks and installation', async () => {
    vi.stubEnv('VITE_INTAKE_MODE', '0');
    const downloadAndInstall = vi.fn().mockResolvedValue(undefined);
    vi.mocked(check).mockResolvedValue({
      version: '1.2.3',
      downloadAndInstall,
    } as unknown as NonNullable<Awaited<ReturnType<typeof check>>>);
    const { result } = renderHook(() => useUpdater());
    await act(async () => {
      await result.current.checkForUpdate();
    });
    expect(result.current.status.available).toBe(true);
    await act(async () => result.current.installUpdate());
    expect(downloadAndInstall).toHaveBeenCalledOnce();
    expect(relaunch).toHaveBeenCalledOnce();
  });
});
