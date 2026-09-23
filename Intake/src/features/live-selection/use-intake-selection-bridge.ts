/**
 * Byline: Claude Code · Sonnet 5 · 2026-09-14
 *
 * Receives the live selection manifest broadcast by the Xplorer engine iframe
 * (see xplorer-copilot-buildkit/xplorer-copilot apps/client/src/hooks/use-intake-embed-bridge.ts)
 * over window.postMessage. Metadata only -- names, paths, sizes and type; no file
 * bytes cross this bridge. This app never reads the selected files itself.
 */
import { useEffect, useMemo, useState } from "react";

const BRIDGE_SOURCE = "xplorer-intake-bridge";

export interface BridgeSelectionEntry {
  name: string;
  path: string;
  file_type: string;
  size?: number;
  metadataOnly?: boolean;
}

export interface BridgeSelectionState {
  /** True once any message has been received from an embedded Xplorer engine. */
  connected: boolean;
  currentPath: string;
  count: number;
  selection: BridgeSelectionEntry[];
}

const initialState: BridgeSelectionState = {
  connected: false,
  currentPath: "",
  count: 0,
  selection: [],
};

export interface SelectionBridgeTrust {
  trustedOrigin: string;
  expectedSource: MessageEventSource;
}

type SelectionBridgeMessage =
  | { type: "ready" }
  | {
      type: "selection";
      currentPath: string;
      count: number;
      selection: BridgeSelectionEntry[];
    };

function isExactHttpOrigin(value: string): boolean {
  try {
    const parsed = new URL(value);
    return (
      (parsed.protocol === "http:" || parsed.protocol === "https:") &&
      parsed.origin === value
    );
  } catch {
    return false;
  }
}

/**
 * Builds the explicit browser bridge trust contract. A top-level window has no
 * embedded sender and therefore cannot enable this bridge accidentally.
 */
export function createSelectionBridgeTrust(
  trustedOrigin: string | undefined,
  expectedSource: MessageEventSource | null | undefined,
): SelectionBridgeTrust | null {
  const origin = trustedOrigin?.trim();
  if (!origin || !isExactHttpOrigin(origin) || !expectedSource) return null;
  return { trustedOrigin: origin, expectedSource };
}

const isSelectionMessage = (
  data: unknown,
): data is {
  source: string;
  type: "selection";
  currentPath: string;
  count: number;
  selection: BridgeSelectionEntry[];
} =>
  typeof data === "object" &&
  data !== null &&
  (data as { source?: unknown }).source === BRIDGE_SOURCE &&
  (data as { type?: unknown }).type === "selection" &&
  typeof (data as { currentPath?: unknown }).currentPath === "string" &&
  Number.isInteger((data as { count?: unknown }).count) &&
  (data as { count: number }).count >= 0 &&
  Array.isArray((data as { selection?: unknown }).selection) &&
  (data as { selection: unknown[] }).selection.every(isSelectionEntry);

function isSelectionEntry(value: unknown): value is BridgeSelectionEntry {
  if (typeof value !== "object" || value === null) return false;
  const entry = value as Record<string, unknown>;
  return (
    typeof entry.name === "string" &&
    typeof entry.path === "string" &&
    typeof entry.file_type === "string" &&
    (entry.size === undefined || Number.isFinite(entry.size)) &&
    (entry.metadataOnly === undefined || typeof entry.metadataOnly === "boolean")
  );
}

const isReadyMessage = (
  data: unknown,
): data is { source: string; type: "ready" } =>
  typeof data === "object" &&
  data !== null &&
  (data as { source?: unknown }).source === BRIDGE_SOURCE &&
  (data as { type?: unknown }).type === "ready";

/** Returns a parsed bridge message only after the sender's origin and identity match. */
export function getTrustedSelectionBridgeMessage(
  event: Pick<MessageEvent, "origin" | "source" | "data">,
  trust: SelectionBridgeTrust | null,
): SelectionBridgeMessage | null {
  if (
    !trust ||
    event.origin !== trust.trustedOrigin ||
    event.source !== trust.expectedSource
  ) {
    return null;
  }

  if (isReadyMessage(event.data)) return { type: "ready" };
  if (isSelectionMessage(event.data)) {
    return {
      type: "selection",
      currentPath: event.data.currentPath,
      count: event.data.count,
      selection: event.data.selection,
    };
  }
  return null;
}

/** Applies a trusted parsed message; an untrusted event is rejected before this point. */
export function applySelectionBridgeMessage(
  previous: BridgeSelectionState,
  message: SelectionBridgeMessage,
): BridgeSelectionState {
  if (message.type === "ready") return { ...previous, connected: true };
  return {
    connected: true,
    currentPath: message.currentPath,
    count: message.count,
    selection: message.selection,
  };
}

/**
 * Untrusted transport: configure VITE_INTAKE_SELECTION_BRIDGE_ORIGIN with the
 * exact http(s) origin of the embedding Xplorer shell. The source must also be
 * this window's direct parent; absent configuration disables the bridge.
 */
export function useIntakeSelectionBridge(): BridgeSelectionState {
  const [state, setState] = useState<BridgeSelectionState>(initialState);
  const trust = useMemo(() => {
    const parent = window.parent === window ? null : window.parent;
    return createSelectionBridgeTrust(
      import.meta.env.VITE_INTAKE_SELECTION_BRIDGE_ORIGIN,
      parent,
    );
  }, []);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const message = getTrustedSelectionBridgeMessage(event, trust);
      if (!message) return;
      setState((previous) => applySelectionBridgeMessage(previous, message));
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [trust]);

  return state;
}
