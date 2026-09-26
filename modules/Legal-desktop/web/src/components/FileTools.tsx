// Byline: Claude Code · Fable 5.1 · 2026-09-21
"use client";

import { useMemo, useState } from "react";
import { legalApiBase } from "@/lib/api/client";

type TimeCandidate = { value: string; source: string; field: string; confidence: string; timezone_known: boolean };
type OriginalTime = {
  resolved: boolean;
  value: string | null;
  source: string | null;
  confidence: string | null;
  timezone_known: boolean;
  conflict: boolean;
  note: string;
  candidates: TimeCandidate[];
};
type MetadataReport = {
  engine: string;
  source_name: string;
  content_hash: string;
  size_bytes: number;
  file_type: string | null;
  summary: Record<string, string | number>;
  original_time: OriginalTime;
  has_gps: boolean;
  metadata: Record<string, unknown>;
};
type OcrResult = {
  engine: string;
  source_name: string;
  text: string;
  word_count: number;
  mean_confidence: number;
  lines: Array<{ text: string; confidence: number }>;
};
type FileOutput = { source_name: string; output_name: string; content_hash: string; removed_fields?: string[]; remaining_authored_fields?: string[] };
type Row = { file: File; meta?: MetadataReport; ocr?: OcrResult; output?: FileOutput; error?: string };

const IMAGE = /\.(png|jpe?g|webp|tiff?|bmp|gif)$/i;
const OFFICE = /\.(docx?|odt|rtf|txt|xlsx?|ods|pptx?|odp)$/i;
const PDF = /\.pdf$/i;
const SOURCE_LABEL: Record<string, string> = {
  exif_capture_time: "camera/phone capture time (EXIF)",
  takeout_photo_taken_time: "Google Takeout sidecar",
  xmp_date_created: "embedded creation date (XMP)",
  png_creation_time: "embedded creation time (PNG)",
  container_creation_date: "embedded creation date",
  embedded_create_date: "embedded create date",
  exif_digitized_time: "EXIF digitized time",
  filename_timestamp: "device-generated file name",
  filename_epoch: "timestamp in the file name",
  filename_date: "date in the file name (day only)",
};

async function post<T>(path: string, form: FormData): Promise<T> {
  const response = await fetch(`${legalApiBase()}${path}`, { method: "POST", body: form });
  if (!response.ok) {
    const detail = await response.json().then((body) => body.detail as string).catch(() => "");
    throw new Error(detail || `request failed (${response.status})`);
  }
  return (await response.json()) as T;
}

function sortKey(row: Row): string {
  return row.meta?.original_time.value?.slice(0, 19) ?? "9999";
}

export function FileTools() {
  const [picked, setPicked] = useState<File[]>([]);
  const [rows, setRows] = useState<Row[]>([]);
  const [layout, setLayout] = useState("auto");
  const [busy, setBusy] = useState<string | null>(null);
  const [ordered, setOrdered] = useState(false);

  // A Takeout sidecar is "<image name>.json" (or ".supplemental-metadata.json"); pair it, don't list it.
  const { files, sidecars } = useMemo(() => {
    const byName = new Map(picked.map((file) => [file.name, file]));
    const paired = new Map<string, File>();
    for (const file of picked) {
      if (!file.name.toLowerCase().endsWith(".json")) continue;
      const target = file.name.replace(/(\.supplemental-metadata)?\.json$/i, "");
      if (byName.has(target)) paired.set(target, file);
    }
    const used = new Set([...paired.values()].map((file) => file.name));
    return { files: picked.filter((file) => !used.has(file.name)), sidecars: paired };
  }, [picked]);

  const images = files.filter((file) => IMAGE.test(file.name));
  const office = files.filter((file) => OFFICE.test(file.name));
  const pdfs = files.filter((file) => PDF.test(file.name));

  async function run(label: string, targets: File[], work: (file: File) => Promise<Partial<Row>>, sort = false) {
    setBusy(label);
    setOrdered(sort);
    const next: Row[] = [];
    for (const file of targets) {
      try {
        next.push({ file, ...(await work(file)) });
      } catch (exc) {
        next.push({ file, error: exc instanceof Error ? exc.message : "failed" });
      }
      setRows(sort ? [...next].sort((a, b) => sortKey(a).localeCompare(sortKey(b))) : [...next]);
    }
    setBusy(null);
  }

  async function metadata(file: File): Promise<Partial<Row>> {
    const form = new FormData();
    form.append("file", file);
    const sidecar = sidecars.get(file.name);
    if (sidecar) form.append("takeout_sidecar_json", await sidecar.text());
    return { meta: await post<MetadataReport>("/v1/documents:metadata", form) };
  }

  async function ocr(file: File): Promise<Partial<Row>> {
    const form = new FormData();
    form.append("file", file);
    form.append("layout", layout);
    return { ocr: await post<OcrResult>("/v1/documents:ocr", form) };
  }

  async function output(path: string, file: File): Promise<Partial<Row>> {
    const form = new FormData();
    form.append("file", file);
    return { output: await post<FileOutput>(path, form) };
  }

  const button = (label: string, count: number, onClick: () => void) => (
    <button type="button" disabled={busy !== null || count === 0} onClick={onClick}>
      {busy === label ? "Working…" : `${label}${count ? ` (${count})` : ""}`}
    </button>
  );

  return (
    <section style={{ display: "grid", gap: 14, margin: "16px 0 32px" }}>
      <label>
        Choose files (photos, screenshots, video, audio, office documents, PDFs)
        <input
          type="file"
          multiple
          onChange={(event) => {
            setPicked(Array.from(event.target.files ?? []));
            setRows([]);
          }}
          style={{ display: "block", marginTop: 6 }}
        />
      </label>
      {picked.length ? (
        <p style={{ color: "var(--text-muted)", margin: 0 }}>
          {files.length} file{files.length === 1 ? "" : "s"} · {images.length} image{images.length === 1 ? "" : "s"}
          {sidecars.size ? ` · ${sidecars.size} Takeout sidecar${sidecars.size === 1 ? "" : "s"} paired` : ""}
        </p>
      ) : null}
      <div style={{ display: "flex", flexWrap: "wrap", gap: 8, alignItems: "center" }}>
        {button("Read metadata", files.length, () => void run("Read metadata", files, metadata))}
        {button("Put images in time order", images.length, () => void run("Put images in time order", images, metadata, true))}
        {button("Read text (OCR)", images.length, () => void run("Read text (OCR)", images, ocr))}
        <label style={{ color: "var(--text-muted)" }}>
          layout{" "}
          <select value={layout} onChange={(event) => setLayout(event.target.value)}>
            <option value="auto">auto</option>
            <option value="block">one column (message thread)</option>
            <option value="sparse">scattered text</option>
          </select>
        </label>
        {button("Convert to PDF", office.length, () => void run("Convert to PDF", office, (file) => output("/v1/documents:convert", file)))}
        {button("Remove PDF metadata", pdfs.length, () => void run("Remove PDF metadata", pdfs, (file) => output("/v1/documents:scrub-metadata", file)))}
      </div>

      {rows.map((row, index) => (
        <article key={`${row.file.name}-${index}`} style={{ borderTop: "1px solid var(--border)", padding: "14px 0" }}>
          <strong>
            {ordered ? `${index + 1}. ` : ""}
            {row.file.name}
          </strong>
          {row.error ? <p style={{ color: "var(--status-warn)" }}>{row.error}</p> : null}

          {row.meta ? (
            <>
              <p style={{ margin: "6px 0" }}>
                {row.meta.original_time.resolved ? (
                  <>
                    Original time: <strong>{row.meta.original_time.value}</strong> — from{" "}
                    {SOURCE_LABEL[row.meta.original_time.source ?? ""] ?? row.meta.original_time.source} (
                    {row.meta.original_time.confidence} confidence
                    {row.meta.original_time.timezone_known ? "" : ", device-local time"})
                    {row.meta.original_time.conflict ? (
                      <span style={{ color: "var(--status-warn)" }}> · sources disagree</span>
                    ) : null}
                  </>
                ) : (
                  <span style={{ color: "var(--status-warn)" }}>No original time in the file or its name.</span>
                )}
              </p>
              <p style={{ color: "var(--text-muted)", margin: "6px 0" }}>
                {[
                  [row.meta.summary.device_make, row.meta.summary.device_model].filter(Boolean).join(" "),
                  row.meta.summary.software ? `software: ${row.meta.summary.software}` : "",
                  row.meta.has_gps ? `GPS ${row.meta.summary.gps_latitude}, ${row.meta.summary.gps_longitude}` : "",
                  row.meta.summary.width ? `${row.meta.summary.width}×${row.meta.summary.height}` : "",
                  row.meta.file_type ?? "",
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </p>
              <p style={{ color: "var(--text-muted)", margin: "6px 0", wordBreak: "break-all" }}>
                sha256 {row.meta.content_hash}
              </p>
              <details>
                <summary>
                  All {Object.keys(row.meta.metadata).length} fields · {row.meta.original_time.candidates.length} time
                  candidate{row.meta.original_time.candidates.length === 1 ? "" : "s"} · {row.meta.engine}
                </summary>
                {row.meta.original_time.candidates.length ? (
                  <ul>
                    {row.meta.original_time.candidates.map((candidate, at) => (
                      <li key={`${candidate.field}-${at}`}>
                        {candidate.value} — {SOURCE_LABEL[candidate.source] ?? candidate.source} ({candidate.field})
                      </li>
                    ))}
                  </ul>
                ) : null}
                <pre style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>
                  {JSON.stringify(row.meta.metadata, null, 2)}
                </pre>
              </details>
            </>
          ) : null}

          {row.ocr ? (
            <>
              <p style={{ color: "var(--text-muted)", margin: "6px 0" }}>
                {row.ocr.word_count} words · mean confidence {row.ocr.mean_confidence} · {row.ocr.engine} · OCR text
              </p>
              <pre style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{row.ocr.text || "(no text found)"}</pre>
              <button type="button" onClick={() => void navigator.clipboard.writeText(row.ocr?.text ?? "")}>
                Copy text
              </button>
            </>
          ) : null}

          {row.output ? (
            <>
              <p style={{ margin: "6px 0" }}>
                <a href={`${legalApiBase()}/v1/documents/renders/${encodeURIComponent(row.output.output_name)}`}>
                  Download {row.output.output_name}
                </a>
              </p>
              {row.output.removed_fields ? (
                <p style={{ color: "var(--text-muted)", margin: "6px 0" }}>
                  Removed: {row.output.removed_fields.join(", ") || "nothing"} · still present:{" "}
                  {row.output.remaining_authored_fields?.join(", ") || "nothing"}
                </p>
              ) : null}
              <p style={{ color: "var(--text-muted)", margin: "6px 0", wordBreak: "break-all" }}>
                sha256 {row.output.content_hash}
              </p>
            </>
          ) : null}
        </article>
      ))}
    </section>
  );
}
