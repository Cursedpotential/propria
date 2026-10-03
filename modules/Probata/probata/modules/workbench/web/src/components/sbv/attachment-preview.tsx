// Ported from modules/forks/sbv/frontend/src/components/LazyMedia.jsx
// MIT, Copyright (c) 2025 lowcarbdev
//
// Owner correction 2026-09-20 (Control surfaces decisions, relayed mid-task): media
// has TWO modes, both meant to render the way SBV renders them.
//   (a) BEFORE processing: the media is still base64 inside the backup XML parts.
//       SBV already decodes and displays that. THIS CLIENT CANNOT WIRE THAT MODE —
//       `ProfferPreviewAttachment` (src/lib/shared/types.ts) carries only metadata
//       (filename, media_type, byte_length, sha256, source_locator_ref), never
//       inline bytes, and no endpoint in api-client.ts returns pre-ingest base64
//       for an attachment. SBV's own decode path is `LazyMedia.jsx`'s `loadMedia()`
//       fetching SBV's own `/api/media?id=<message_id>` as a blob/text — that
//       assumes SBV's backend already extracted the bytes server-side. Wiring (a)
//       needs either an engine endpoint that inlines base64 for attachments on a
//       pre-ingest source version, or a dedicated pre-ingest media route; neither
//       exists yet. Left unwired; see the fallback path below and the task report.
//   (b) AFTER ingest: media comes from `<key>.derived/media/<sha256><ext>` in B2,
//       served by `GET /api/proffer/previews/{handle}/media/{sha256}` (contract in
//       api-client.ts `getProfferPreviewMediaUrl`). THIS is wired: image/video/audio
//       attempt the real element with that URL, matching SBV's own
//       <img>/<video>/<audio src=...> usage; a load failure (commonly: this source
//       version predates ingest, so the derived file does not exist yet) falls back
//       to the metadata-only card with a small "preview unavailable" flag, per the
//       "no fake URLs, no banners" rule — never blank, never guessed, never a
//       second network attempt at an invented path.
// Byline: Claude Code · Opus 5 · 2026-09-20
// Byline: Claude Code · Sonnet 5.5 · 2026-10-03 (a PDF the media route serves gets an "Open" card, not "preview unavailable")
"use client";

import { AudioLines, Contact2, File, FileText, Image as ImageIcon, TriangleAlert, Video } from "lucide-react";
import { useState } from "react";

import { VCardPreview } from "@/components/sbv/vcard-preview";
import { Button } from "@/components/ui/button";
import { getProfferPreviewMediaUrl } from "@/lib/api-client";
import type { MatterMode, ProfferPreviewAttachment } from "@/lib/shared/types";

function byteLabel(value: number | null | undefined) {
  if (value === null || value === undefined) return "size unavailable";
  if (value < 1024) return `${value} bytes`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

export type AttachmentKind = "image" | "video" | "audio" | "vcard" | "pdf" | "other";

export function attachmentKind(mediaType: string | null | undefined): AttachmentKind {
  if (!mediaType) return "other";
  if (mediaType.startsWith("image/")) return "image";
  if (mediaType.startsWith("video/")) return "video";
  if (mediaType.startsWith("audio/")) return "audio";
  if (mediaType === "application/pdf") return "pdf";
  if (mediaType === "text/x-vcard" || mediaType === "text/vcard" || mediaType === "text/directory") return "vcard";
  return "other";
}

const KIND_ICON: Record<AttachmentKind, typeof ImageIcon> = {
  image: ImageIcon,
  video: Video,
  audio: AudioLines,
  vcard: Contact2,
  pdf: FileText,
  other: File,
};

const KIND_LABEL: Record<AttachmentKind, string> = {
  image: "Image",
  video: "Video",
  audio: "Audio",
  vcard: "Contact",
  pdf: "PDF document",
  other: "Attachment",
};

function MetadataCard({
  attachment,
  kind,
  variant,
  onOpen,
}: {
  attachment: ProfferPreviewAttachment;
  kind: AttachmentKind;
  variant: "compact" | "tile";
  onOpen?: () => void;
}) {
  const Icon = KIND_ICON[kind];
  const isTile = variant === "tile";
  return (
    <button
      type="button"
      onClick={onOpen}
      disabled={!onOpen}
      className={
        isTile
          ? "flex aspect-square w-full flex-col items-center justify-center gap-1.5 rounded-md border bg-muted/40 p-2 text-center disabled:cursor-default"
          : "flex w-full min-w-0 items-center gap-2 rounded-md border bg-muted/30 px-2 py-1.5 text-left disabled:cursor-default"
      }
      data-testid="attachment-preview-metadata"
      data-kind={kind}
    >
      <Icon className={isTile ? "size-6 text-muted-foreground" : "size-4 shrink-0 text-muted-foreground"} aria-hidden="true" />
      <span className={isTile ? "min-w-0" : "min-w-0 flex-1"}>
        <span className="block truncate text-xs font-medium">{attachment.filename ?? KIND_LABEL[kind]}</span>
        <span className="block truncate text-[10px] text-muted-foreground">
          {KIND_LABEL[kind]} · {byteLabel(attachment.byte_length)}
        </span>
      </span>
      <span
        className="inline-flex shrink-0 items-center gap-1 rounded-full bg-muted px-1.5 py-0.5 text-[9px] font-medium text-muted-foreground"
        title="The derived media file for this attachment is not available (commonly: this source version predates ingest). Only metadata is shown."
        data-testid="attachment-preview-unwired-flag"
      >
        <TriangleAlert className="size-2.5" aria-hidden="true" /> preview unavailable
      </span>
    </button>
  );
}

/**
 * A PDF the media route serves: a card with the file name, size and an "Open" link to the
 * retained bytes (new tab). It replaces the "preview unavailable" metadata card, which was
 * wrong for a PDF the media route does serve. No inline viewer is loaded until the reader
 * opens the file.
 */
function PdfLinkCard({ attachment, url, variant }: { attachment: ProfferPreviewAttachment; url: string; variant: "compact" | "tile" }) {
  const isTile = variant === "tile";
  return (
    <div
      className={
        isTile
          ? "flex aspect-square w-full flex-col items-center justify-center gap-1.5 rounded-md border bg-muted/40 p-2 text-center"
          : "flex w-full min-w-0 items-center gap-2 rounded-md border bg-muted/30 px-2 py-1.5 text-left"
      }
      data-testid="attachment-preview-pdf"
      data-kind="pdf"
    >
      <FileText className={isTile ? "size-6 text-muted-foreground" : "size-4 shrink-0 text-muted-foreground"} aria-hidden="true" />
      <span className={isTile ? "min-w-0" : "min-w-0 flex-1"}>
        <span className="block truncate text-xs font-medium text-foreground">{attachment.filename ?? KIND_LABEL.pdf}</span>
        <span className="block truncate text-[10px] text-muted-foreground">
          {KIND_LABEL.pdf} · {byteLabel(attachment.byte_length)}
        </span>
      </span>
      <Button asChild size="sm" variant="outline" className="h-6 shrink-0 px-2 text-[11px]">
        <a href={url} target="_blank" rel="noopener noreferrer" data-testid="attachment-preview-pdf-open">
          Open
        </a>
      </Button>
    </div>
  );
}

interface AttachmentPreviewProps {
  attachment: ProfferPreviewAttachment;
  previewHandle: string;
  mode: MatterMode;
  /** Compact renders the SBV chat-bubble inline size; "tile" renders the media-grid square. */
  variant?: "compact" | "tile";
  onOpen?: () => void;
}

/** Real post-ingest media (image/video/audio) with a metadata-card fallback. Never fakes a URL. */
export function AttachmentPreview({ attachment, previewHandle, mode, variant = "compact", onOpen }: AttachmentPreviewProps) {
  const kind = attachmentKind(attachment.media_type);
  const [broken, setBroken] = useState(false);
  const url = attachment.sha256 ? getProfferPreviewMediaUrl(previewHandle, mode, attachment.sha256) : null;
  const isTile = variant === "tile";

  // The backup names this part and carries no bytes for it: one small flag on the item,
  // in the slot the photo would have taken.
  if (attachment.payload_missing) {
    return (
      <div
        className={isTile
          ? "grid aspect-square w-full place-content-center rounded-md border border-dashed border-destructive/60 p-2 text-center"
          : "rounded-md border border-dashed border-destructive/60 px-3 py-2"}
        data-testid="attachment-payload-missing"
        title={`${attachment.filename ?? "Attachment"} is named in this backup, but the backup holds no bytes for it.`}
      >
        <span className="block truncate text-xs font-medium">{attachment.filename ?? KIND_LABEL[kind]}</span>
        <span className="block text-[11px] text-destructive">Missing from this backup</span>
      </div>
    );
  }

  if (kind === "vcard" && !isTile) {
    return <VCardPreview attachment={attachment} previewHandle={previewHandle} mode={mode} />;
  }

  if (kind === "pdf" && url) {
    return <PdfLinkCard attachment={attachment} url={url} variant={variant} />;
  }

  if (!url || broken || (kind !== "image" && kind !== "video" && kind !== "audio")) {
    return <MetadataCard attachment={attachment} kind={kind} variant={variant} onOpen={onOpen} />;
  }

  const frame = isTile
    ? "aspect-square w-full overflow-hidden rounded-md border bg-muted/20"
    : "max-w-full overflow-hidden rounded-md border";

  return (
    <button
      type="button"
      onClick={onOpen}
      disabled={!onOpen}
      className={`${frame} block disabled:cursor-default`}
      data-testid="attachment-preview-media"
      data-kind={kind}
    >
      {kind === "image" && (
        <img
          src={url}
          alt={attachment.filename ?? "Attachment"}
          loading="lazy"
          className={isTile ? "size-full object-cover" : "max-h-64 w-auto object-contain"}
          onError={() => setBroken(true)}
        />
      )}
      {kind === "video" && (
        <video
          src={url}
          controls={!isTile}
          muted={isTile}
          playsInline
          preload="metadata"
          className={isTile ? "size-full object-cover" : "max-h-64 w-full"}
          onError={() => setBroken(true)}
        />
      )}
      {kind === "audio" && !isTile && (
        <audio src={url} controls className="w-full" onError={() => setBroken(true)} />
      )}
      {kind === "audio" && isTile && (
        <div className="flex size-full items-center justify-center">
          <AudioLines className="size-6 text-muted-foreground" aria-hidden="true" />
        </div>
      )}
    </button>
  );
}
