import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { usePaneSelection } from '@/hooks/use-pane-selection';
import type { EditorGroup } from '@/types/split-view';
import type { FileEntry } from '@/lib/tauri-api';

const group = (id: string, path: string): EditorGroup => ({
  id,
  currentPath: path,
  activeTabId: `${id}-tab`,
  tabs: [],
  pathHistory: [path],
  historyIndex: 0,
});
const initialGroups = { left: group('left', 'E:\\left'), right: group('right', 'V:\\raw') };
const file = (path: string): FileEntry => ({
  path,
  name: path,
  is_dir: false,
  size: 10,
  modified: 0,
  file_type: 'file',
});
const setup = () =>
  renderHook(
    ({ groups, active }) => {
      const panes = usePaneSelection(groups);
      return { panes, active: panes[active] };
    },
    { initialProps: { groups: initialGroups as Record<string, EditorGroup>, active: 'left' } },
  );

describe('usePaneSelection', () => {
  it('preserves the left group while focus switches right and back', () => {
    const { result, rerender } = setup();
    act(() => result.current.active.setSelectedFiles(new Set(['A', 'B'])));
    rerender({ groups: initialGroups, active: 'right' });
    expect([...result.current.active.selectedFiles]).toEqual([]);
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A', 'B']);
    rerender({ groups: initialGroups, active: 'left' });
    expect([...result.current.active.selectedFiles]).toEqual(['A', 'B']);
  });

  it('keeps distinct groups and active preview independently', () => {
    const { result, rerender } = setup();
    const a = file('A');
    const c = file('C');
    act(() => {
      result.current.panes.left.setSelectedFiles(new Set(['A', 'B']));
      result.current.panes.left.setSelectedFile(a);
      result.current.panes.right.setSelectedFiles(new Set(['C']));
      result.current.panes.right.setSelectedFile(c);
    });
    rerender({ groups: initialGroups, active: 'right' });
    expect([...result.current.active.selectedFiles]).toEqual(['C']);
    expect(result.current.active.selectedFile).toEqual(c);
    expect(result.current.panes.left.selectedFile).toEqual(a);
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A', 'B']);
  });

  it('binds an inactive pane first-click handler to its own pane', () => {
    const { result, rerender } = setup();
    act(() => result.current.panes.left.setSelectedFiles(new Set(['A'])));
    const rightClickSetter = result.current.panes.right.setSelectedFiles;
    rerender({ groups: initialGroups, active: 'right' });
    act(() => rightClickSetter(new Set(['C'])));
    expect([...result.current.panes.right.selectedFiles]).toEqual(['C']);
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A']);
  });

  it('clears only the navigated pane and rejects delayed old-path events', () => {
    const { result, rerender } = setup();
    act(() => {
      result.current.panes.left.setSelectedFiles(new Set(['A', 'B']));
      result.current.panes.right.setSelectedFiles(new Set(['C']));
      result.current.panes.right.setSelectedFile(file('C'));
    });
    const oldSetter = result.current.panes.right.setSelectedFiles;
    const groups = {
      ...initialGroups,
      right: { ...initialGroups.right, currentPath: 'Y:\\sorted' },
    };
    rerender({ groups, active: 'right' });
    act(() => oldSetter(new Set(['stale'])));
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A', 'B']);
    expect([...result.current.active.selectedFiles]).toEqual([]);
    expect(result.current.active.selectedFile).toBeNull();
  });

  it('does not restore retired path selection on return navigation', () => {
    const { result, rerender } = setup();
    act(() => result.current.panes.right.setSelectedFiles(new Set(['C'])));
    rerender({
      groups: { ...initialGroups, right: { ...initialGroups.right, currentPath: 'Y:\\sorted' } },
      active: 'right',
    });
    rerender({ groups: initialGroups, active: 'right' });
    expect([...result.current.active.selectedFiles]).toEqual([]);
  });

  it('retires a closed pane without disturbing the remaining pane', () => {
    const { result, rerender } = setup();
    act(() => {
      result.current.panes.left.setSelectedFiles(new Set(['A']));
      result.current.panes.right.setSelectedFiles(new Set(['C']));
    });
    const oldSetter = result.current.panes.right.setSelectedFiles;
    rerender({ groups: { left: initialGroups.left }, active: 'left' });
    act(() => oldSetter(new Set(['stale'])));
    expect(result.current.panes.right).toBeUndefined();
    expect([...result.current.active.selectedFiles]).toEqual(['A']);
    rerender({ groups: initialGroups, active: 'left' });
    expect([...result.current.panes.right.selectedFiles]).toEqual([]);
  });

  it('preserves selection through layout-only updates and split additions', () => {
    const { result, rerender } = setup();
    act(() => result.current.panes.left.setSelectedFiles(new Set(['A'])));
    rerender({ groups: { ...initialGroups, extra: group('extra', 'E:\\left') }, active: 'extra' });
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A']);
    expect([...result.current.active.selectedFiles]).toEqual([]);
    rerender({ groups: { ...initialGroups }, active: 'left' });
    expect([...result.current.active.selectedFiles]).toEqual(['A']);
  });

  it('treats switching tab as navigation for that pane only', () => {
    const { result, rerender } = setup();
    act(() => {
      result.current.panes.left.setSelectedFiles(new Set(['A']));
      result.current.panes.right.setSelectedFiles(new Set(['C']));
    });
    rerender({
      groups: { ...initialGroups, right: { ...initialGroups.right, activeTabId: 'another-tab' } },
      active: 'right',
    });
    expect([...result.current.panes.right.selectedFiles]).toEqual([]);
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A']);
  });

  it('supports updater functions without leaking mutations or changing setter identity', () => {
    const { result } = setup();
    const input = new Set(['A']);
    const setter = result.current.panes.left.setSelectedFiles;
    act(() => setter(input));
    input.add('external');
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A']);
    act(() => setter((previous) => new Set([...previous, 'B'])));
    expect([...result.current.panes.left.selectedFiles]).toEqual(['A', 'B']);
    expect(result.current.panes.left.setSelectedFiles).toBe(setter);
  });
});
