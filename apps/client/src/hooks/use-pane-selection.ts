import { useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import type { EditorGroup } from '@/types/split-view';
import type { FileEntry } from '@/lib/tauri-api';

export interface PaneSelection {
  selectedFiles: Set<string>;
  setSelectedFiles: Dispatch<SetStateAction<Set<string>>>;
  selectedFile: FileEntry | null;
  setSelectedFile: Dispatch<SetStateAction<FileEntry | null>>;
}

interface StoredSelection {
  scope: string;
  paths: Set<string>;
  preview: FileEntry | null;
}

const scopeOf = (group: EditorGroup): string =>
  JSON.stringify([group.activeTabId, group.currentPath]);

/** One selection per pane; focus changes never copy or clear another pane. */
export const usePaneSelection = (
  groups: Record<string, EditorGroup>,
): Record<string, PaneSelection> => {
  const [stored, setStored] = useState<Record<string, StoredSelection>>({});
  const groupsRef = useRef(groups);
  groupsRef.current = groups;

  // Closed groups and actual navigation retire only their own selection. The
  // render projection also checks scope, so stale selections never flash first.
  useEffect(() => {
    setStored((previous) => {
      const remaining = Object.entries(previous).filter(
        ([id, value]) => groups[id] && scopeOf(groups[id]) === value.scope,
      );
      return remaining.length === Object.keys(previous).length
        ? previous
        : Object.fromEntries(remaining);
    });
  }, [groups]);

  const bindings = useMemo(() => {
    const selections: Record<
      string,
      Pick<PaneSelection, 'setSelectedFiles' | 'setSelectedFile'> & {
        scope: string;
        emptyPaths: Set<string>;
      }
    > = {};
    for (const [id, group] of Object.entries(groups)) {
      const scope = scopeOf(group);
      const update = (change: (previous: StoredSelection) => StoredSelection) => {
        setStored((previous) => {
          const currentGroup = groupsRef.current[id];
          // Delayed events from a closed/navigated pane cannot re-add its old state.
          if (!currentGroup || scopeOf(currentGroup) !== scope) return previous;
          const current =
            previous[id]?.scope === scope
              ? previous[id]
              : { scope, paths: new Set<string>(), preview: null };
          return { ...previous, [id]: change(current) };
        });
      };
      selections[id] = {
        scope,
        emptyPaths: new Set<string>(),
        setSelectedFiles: (value) =>
          update((current) => ({
            ...current,
            paths: new Set(typeof value === 'function' ? value(new Set(current.paths)) : value),
          })),
        setSelectedFile: (value) =>
          update((current) => ({
            ...current,
            preview: typeof value === 'function' ? value(current.preview) : value,
          })),
      };
    }
    return selections;
  }, [groups]);

  return useMemo(
    () =>
      Object.fromEntries(
        Object.entries(bindings).map(([id, binding]) => {
          const existing = stored[id]?.scope === binding.scope ? stored[id] : undefined;
          return [
            id,
            {
              selectedFiles: existing?.paths ?? binding.emptyPaths,
              selectedFile: existing?.preview ?? null,
              setSelectedFiles: binding.setSelectedFiles,
              setSelectedFile: binding.setSelectedFile,
            },
          ];
        }),
      ),
    [bindings, stored],
  );
};
