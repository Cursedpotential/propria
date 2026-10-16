import { useState } from "react";
import type { ProfferSourcePreview } from "@/lib/shared/types";
import { Button } from "@/components/ui/button";

/** Display original media with observable browser loading, decoding and retry states.
 * Inputs: inspected source descriptor. Output: native media reader and recovery.
 * Effects: read-only requests to the same pinned URL; choose for image/audio/video.
 */
export function SourceMediaPreview({ inspection }: { inspection: ProfferSourcePreview }) {
  const [attempt, setAttempt] = useState(0);
  const [status, setStatus] = useState<"loading" | "ready" | "failed">("loading");
  const [reason, setReason] = useState("");
  const ready = () => setStatus("ready");
  const failed = (detail: string) => { setReason(detail); setStatus("failed"); };
  // Retry bypasses the browser's failed-resource cache without changing any source identity pin.
  const mediaUrl = inspection.preview_url && attempt > 0
    ? `${inspection.preview_url}${inspection.preview_url.includes("?") ? "&" : "?"}_preview_retry=${attempt}`
    : inspection.preview_url;
  const mediaProps = {
    src: mediaUrl ?? undefined,
    onError: () => failed("The browser could not load or decode this original. The stored bytes, delivery, or media format need inspection."),
  };
  return <div className="space-y-2" aria-busy={status === "loading"}>
    {status === "loading" && <p role="status" className="text-xs">Loading original media… If loading does not finish, retry or download this version for inspection.</p>}
    {status === "failed" && <p role="alert" className="text-xs text-destructive">{reason} Retrying reads the same version; it does not repair or replace the original.</p>}
    <Button variant="outline" size="sm" onClick={() => { setStatus("loading"); setReason(""); setAttempt((value) => value + 1); }}>Retry media</Button>
    {inspection.preview_kind === "image" ? <img key={attempt} {...mediaProps} onLoad={ready} className="max-h-[70vh] max-w-full border object-contain" alt={`Original ${inspection.name}`} />
      : inspection.preview_kind === "audio" ? <audio key={attempt} {...mediaProps} onLoadedMetadata={ready} onAbort={() => failed("Loading the original audio was interrupted.")} className="w-full" controls preload="metadata" aria-label={`Listen to ${inspection.name}`} />
        : <video key={attempt} {...mediaProps} onLoadedMetadata={ready} onAbort={() => failed("Loading the original video was interrupted.")} className="max-h-[70vh] w-full border" controls preload="metadata" aria-label={`Watch ${inspection.name}`} />}
    <p className="break-all text-[11px] text-muted-foreground">{inspection.content_type} · {inspection.provider_version_id ? `Provider version ${inspection.provider_version_id}` : "Provider did not supply a version; reader verifies ETag and length."}</p>
  </div>;
}
