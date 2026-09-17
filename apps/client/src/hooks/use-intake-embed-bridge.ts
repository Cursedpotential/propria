/**
 * Byline: Claude Code · Sonnet 5 · 2026-09-14
 *
 * Intake co-workspace bridge: when this Xplorer build is embedded as an iframe
 * inside the Consignatio Intake shell (progress-board /intake/ page), broadcast
 * the live selection manifest to the parent frame via postMessage so a docked
 * review/metadata panel can read the same selection Xplorer's own state exposes
 * to its chat panel (see chat-context-helpers.ts: getXplorerState/buildSelectionManifest).
 *
 * No-op outside an iframe (window.parent === window) and does not read file
 * bytes -- metadata only, matching the existing chat-selection-manifest contract.
 */
import { useEffect } from 'react';
import {
  buildSelectionManifest,
  getXplorerState,
  type XplorerState,
} from '@/components/panels/chat-context-helpers';

export const INTAKE_BRIDGE_MESSAGE_SOURCE = 'xplorer-intake-bridge';

export interface IntakeBridgeSelectionMessage {
  source: typeof INTAKE_BRIDGE_MESSAGE_SOURCE;
  type: 'selection';
  currentPath: string;
  count: number;
  selection: ReturnType<typeof buildSelectionManifest>;
}

export interface IntakeBridgeReadyMessage {
  source: typeof INTAKE_BRIDGE_MESSAGE_SOURCE;
  type: 'ready';
}

const isEmbedded = (): boolean => {
  try {
    return typeof window !== 'undefined' && window.parent !== window;
  } catch {
    // Cross-origin parent access can throw in some sandboxed contexts; treat as embedded-unknown.
    return false;
  }
};

const postToParent = (message: IntakeBridgeSelectionMessage | IntakeBridgeReadyMessage): void => {
  // Embedded same-origin under the progress-board host (/intake/xplorer/ inside /intake/):
  // restrict the target to this document's own origin rather than '*'.
  window.parent.postMessage(message, window.location.origin);
};

const readSelection = (): IntakeBridgeSelectionMessage => {
  const state: XplorerState | undefined = getXplorerState();
  const selection = buildSelectionManifest(state?.selectedFiles ?? []);
  return {
    source: INTAKE_BRIDGE_MESSAGE_SOURCE,
    type: 'selection',
    currentPath: state?.currentPath ?? '',
    count: selection.length,
    selection,
  };
};

/** Mount once at the top of the Explorer page. Emits on every xplorer-state-change. */
export const useIntakeEmbedBridge = (): void => {
  useEffect(() => {
    if (!isEmbedded()) return;
    postToParent({ source: INTAKE_BRIDGE_MESSAGE_SOURCE, type: 'ready' });
    const sync = () => postToParent(readSelection());
    sync();
    window.addEventListener('xplorer-state-change', sync);
    return () => window.removeEventListener('xplorer-state-change', sync);
  }, []);
};
