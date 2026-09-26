// Byline: Claude Code · Fable 5.1 · 2026-09-21
import { FileTools } from "@/components/FileTools";

export default function FileToolsPage() {
  return (
    <>
      <p className="section-eyebrow">Contracts</p>
      <p className="muted">File tools</p>
      <h1 style={{ fontFamily: "Georgia, serif", fontWeight: 500 }}>
        Metadata, original timestamps, screenshot text, PDF conversion.
      </h1>
      <p>
        Select one file or a series to inspect metadata, recover timestamps, extract screenshot text, or convert PDFs. To order screenshots of one conversation, select them with any Google Takeout <code>.json</code> sidecars and use <em>Put images in time order</em>.
      </p>
      <p style={{ color: "var(--text-muted)" }}>
        Resolved times retain their named source. Conversions and scrubbed copies are derived work product.
      </p>
      <FileTools />
    </>
  );
}
