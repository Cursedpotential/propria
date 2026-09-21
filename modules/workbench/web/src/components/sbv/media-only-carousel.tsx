// Ported from modules/forks/sbv/frontend/src/components/MediaCarousel.jsx (+ .css)
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: the full-screen overlay, counter, close button, and prev/next nav-button
// layout are ported. Media itself follows the same real-endpoint-with-fallback rule
// as attachment-preview.tsx: it renders SBV's own `<img>`/`<video src=.../media?id=...}>`
// pattern against this app's post-ingest `getProfferPreviewMediaUrl`, and falls back
// to the attachment's metadata (filename, media type, size, sha256) when that file
// is not available (commonly: this source version predates ingest) — never a
// fabricated URL. Touch-swipe handlers and the react-datepicker/header-visibility
// DOM hacks (querySelector on '.react-datepicker-popper', 'header') were dropped as
// inapplicable to this app's structure.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { useEffect, useState } from "react";

import { attachmentKind } from "@/components/sbv/attachment-preview";
import { getProfferPreviewMediaUrl } from "@/lib/api-client";
import type { MatterMode, ProfferPreviewAttachment } from "@/lib/shared/types";

function byteLabel(value: number | null | undefined) {
  if (value === null || value === undefined) return "size unavailable";
  if (value < 1024) return `${value} bytes`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

interface MediaOnlyCarouselProps {
  items: ProfferPreviewAttachment[];
  index: number;
  previewHandle: string;
  mode: MatterMode;
  onIndexChange: (index: number) => void;
  onClose: () => void;
}

export function MediaOnlyCarousel({ items, index, previewHandle, mode, onIndexChange, onClose }: MediaOnlyCarouselProps) {
  const current = items[index];
  // The failed item is tracked by index rather than reset from an effect, so
  // moving to the next slide clears the error during render instead of
  // triggering a cascading re-render.
  const [brokenIndex, setBrokenIndex] = useState<number | null>(null);
  const broken = brokenIndex === index;
  const setBroken = () => setBrokenIndex(index);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
      else if (event.key === "ArrowLeft" && index > 0) onIndexChange(index - 1);
      else if (event.key === "ArrowRight" && index < items.length - 1) onIndexChange(index + 1);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [index, items.length, onClose, onIndexChange]);

  useEffect(() => {
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "";
    };
  }, []);

  if (!current) return null;
  const kind = attachmentKind(current.media_type);
  const url = current.sha256 ? getProfferPreviewMediaUrl(previewHandle, mode, current.sha256) : null;
  const showRealMedia = Boolean(url) && !broken && (kind === "image" || kind === "video");

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-6"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      data-testid="media-only-carousel"
    >
      <button
        type="button"
        className="absolute right-4 top-4 rounded-full bg-white/10 p-2 text-white hover:bg-white/20"
        onClick={onClose}
        aria-label="Close"
      >
        <X className="size-5" />
      </button>

      <div className="absolute left-1/2 top-4 -translate-x-1/2 rounded-full bg-white/10 px-3 py-1 text-xs text-white">
        {index + 1} / {items.length}
      </div>

      {index > 0 && (
        <button
          type="button"
          className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white hover:bg-white/20"
          onClick={(event) => {
            event.stopPropagation();
            onIndexChange(index - 1);
          }}
          aria-label="Previous"
        >
          <ChevronLeft className="size-6" />
        </button>
      )}
      {index < items.length - 1 && (
        <button
          type="button"
          className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-2 text-white hover:bg-white/20"
          onClick={(event) => {
            event.stopPropagation();
            onIndexChange(index + 1);
          }}
          aria-label="Next"
        >
          <ChevronRight className="size-6" />
        </button>
      )}

      {showRealMedia ? (
        <div className="flex max-h-[90vh] max-w-[90vw] flex-col items-center gap-2" onClick={(event) => event.stopPropagation()}>
          {kind === "image" ? (
            <img
              src={url ?? undefined}
              alt={current.filename ?? "Attachment"}
              className="max-h-[85vh] max-w-full rounded-md object-contain"
              onError={() => setBroken()}
            />
          ) : (
            <video
              src={url ?? undefined}
              controls
              autoPlay
              playsInline
              className="max-h-[85vh] max-w-full rounded-md"
              onError={() => setBroken()}
            />
          )}
          <p className="text-xs text-white/70">{current.filename ?? "Attachment"}</p>
        </div>
      ) : (
        <div
          className="max-h-[85vh] w-full max-w-lg rounded-md border bg-card p-6 text-card-foreground shadow-lg"
          onClick={(event) => event.stopPropagation()}
        >
          <p className="text-xs uppercase tracking-wide text-muted-foreground">{kind}</p>
          <h3 className="mt-1 break-all text-sm font-semibold">{current.filename ?? "Unnamed attachment"}</h3>
          <dl className="mt-4 space-y-2 text-xs">
            <div>
              <dt className="text-muted-foreground">Media type</dt>
              <dd className="font-mono">{current.media_type ?? "not reported"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Size</dt>
              <dd className="font-mono">{byteLabel(current.byte_length)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">SHA-256</dt>
              <dd className="break-all font-mono">{current.sha256 ?? "not reported"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Source locator</dt>
              <dd className="break-all font-mono">{current.source_locator_ref}</dd>
            </div>
          </dl>
          <p className="mt-4 rounded-md bg-muted px-2 py-1.5 text-[11px] text-muted-foreground">
            Preview unavailable — the derived media file for this attachment was not found
            (commonly: this source version predates ingest).
          </p>
        </div>
      )}
    </div>
  );
}
