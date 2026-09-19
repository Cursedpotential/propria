// Byline: Claude Code · Opus 5 · 2026-09-18
// The chats-index hit the user last opened. The Metadata panel shows it while the selected file is
// that hit's source file (or nothing is selected); selecting any other file hides it.
import { useSyncExternalStore } from 'react';

export interface SelectedChatEvent {
  dedupKey: string;
  sourcePath: string | null;
}

let current: SelectedChatEvent | null = null;
const listeners = new Set<() => void>();

export const setSelectedChatEvent = (value: SelectedChatEvent | null): void => {
  current = value;
  listeners.forEach((l) => l());
};

const subscribe = (listener: () => void): (() => void) => {
  listeners.add(listener);
  return () => listeners.delete(listener);
};

export const useSelectedChatEvent = (): SelectedChatEvent | null =>
  useSyncExternalStore(subscribe, () => current);

/** Window event the explorer handles: open the Metadata panel, navigate to the file and select it. */
export const OPEN_SEARCH_HIT_EVENT = 'intake-open-search-hit';

export const openSearchHit = (path: string): void => {
  window.dispatchEvent(new CustomEvent(OPEN_SEARCH_HIT_EVENT, { detail: { path } }));
};
