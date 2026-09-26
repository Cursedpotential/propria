import { useEffect, useState } from 'react';
import { getXplorerState, type SelectionEntry, type XplorerState } from './chat-context-helpers';

/** A hidden retained chat does not poll; reopening immediately refreshes its selection snapshot. */
export const useChatContext = (active: boolean) => {
  const [currentPath, setCurrentPath] = useState('');
  const [selectedFiles, setSelectedFiles] = useState<SelectionEntry[]>([]);
  const [editorSelection, setEditorSelection] = useState<XplorerState['editorSelection']>(null);
  useEffect(() => {
    if (!active) return;
    const sync = () => {
      const state = getXplorerState();
      if (!state) return;
      setCurrentPath(state.currentPath ?? '');
      setSelectedFiles(state.selectedFiles ?? []);
      setEditorSelection(state.editorSelection ?? null);
    };
    sync();
    window.addEventListener('xplorer-state-change', sync);
    const interval = setInterval(sync, 1000);
    return () => {
      window.removeEventListener('xplorer-state-change', sync);
      clearInterval(interval);
    };
  }, [active]);
  return { currentPath, selectedFiles, editorSelection };
};
