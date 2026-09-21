// Ported from modules/forks/sbv/frontend/src/components/MediaGrid.jsx (+ .css)
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: SBV fetched `${API_BASE}/media-items` and rendered real thumbnails
// (<img>/<video>) in a CSS grid, with click-to-open-carousel and video-transcode
// retry handling. This BFF has no media-serving endpoint, so the grid is built
// directly from the already-loaded message rows' `attachments` arrays (filtered to
// image/video kinds, mirroring SBV's "photos only" toggle) and each tile renders
// AttachmentPreview (metadata only) instead of a thumbnail. The failed/transcode
// video bookkeeping has no equivalent here since no video is ever actually played.
// Byline: Claude Code · Opus 5 · 2026-09-20
"use client";

import { useMemo, useState } from "react";

import { AttachmentPreview, attachmentKind } from "@/components/sbv/attachment-preview";
import { MediaOnlyCarousel } from "@/components/sbv/media-only-carousel";
import type { PreviewMessageRow } from "@/hooks/use-preview-messages";
import type { MatterMode, ProfferPreviewAttachment } from "@/lib/shared/types";

interface MediaOnlyGridProps {
  rows: PreviewMessageRow[];
  previewHandle: string;
  mode: MatterMode;
}

export function MediaOnlyGrid({ rows, previewHandle, mode }: MediaOnlyGridProps) {
  const mediaItems = useMemo<ProfferPreviewAttachment[]>(() => {
    const items: ProfferPreviewAttachment[] = [];
    for (const row of rows) {
      for (const attachment of row.message.attachments) {
        const kind = attachmentKind(attachment.media_type);
        if (kind === "image" || kind === "video") items.push(attachment);
      }
    }
    return items;
  }, [rows]);

  const [openIndex, setOpenIndex] = useState<number | null>(null);

  if (mediaItems.length === 0) {
    return (
      <div className="grid min-h-40 place-content-center text-center text-sm text-muted-foreground" data-testid="media-only-grid-empty">
        No photos or videos found in this conversation.
      </div>
    );
  }

  return (
    <>
      <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5" data-testid="media-only-grid">
        {mediaItems.map((item, index) => (
          <AttachmentPreview
            key={item.attachment_id}
            attachment={item}
            previewHandle={previewHandle}
            mode={mode}
            variant="tile"
            onOpen={() => setOpenIndex(index)}
          />
        ))}
      </div>
      {openIndex !== null && (
        <MediaOnlyCarousel
          items={mediaItems}
          index={openIndex}
          previewHandle={previewHandle}
          mode={mode}
          onIndexChange={setOpenIndex}
          onClose={() => setOpenIndex(null)}
        />
      )}
    </>
  );
}
