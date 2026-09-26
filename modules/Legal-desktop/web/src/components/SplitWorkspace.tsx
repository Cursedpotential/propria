"use client";

import type { ReactNode } from "react";
import { navLabel, surfaceForPath } from "@/lib/surfaces";

// Byline: Grok · grok-4.6 · 2026-08-18
// Right pane is a same-origin iframe so App Router can show two modules at once.

export const SPLIT_PIN_KEY = "lw-split-pin";
export const SPLIT_MSG_SOURCE = "lw-split";

export function isSafePinPath(raw: string | null | undefined): raw is string {
  if (!raw) return false;
  return raw.startsWith("/") && !raw.startsWith("//") && !raw.includes("://");
}

export function embedSrc(path: string): string {
  return `${path}${path.includes("?") ? "&" : "?"}embed=1`;
}

export function SplitWorkspace({
  pinnedPath,
  onClose,
  children,
}: {
  pinnedPath: string | null;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!pinnedPath || !isSafePinPath(pinnedPath)) {
    return <div className="module-body">{children}</div>;
  }

  const pinned = surfaceForPath(pinnedPath.split("?")[0] ?? pinnedPath);

  return (
    <div className="workspace-split">
      <div className="workspace-split-pane">
        <div className="module-body">{children}</div>
      </div>
      <div className="workspace-split-pane workspace-split-pane-pinned">
        <button
          type="button"
          className="split-close-btn"
          onClick={onClose}
          title="Close split view"
        >
          × split
        </button>
        <iframe
          className="workspace-split-frame"
          src={embedSrc(pinnedPath)}
          title={`Pinned ${navLabel(pinned)}`}
        />
      </div>
    </div>
  );
}
