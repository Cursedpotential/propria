// Byline: Claude Code · Opus 5.5 · 2026-09-25
// The filename-extension -> declared_format rule intake uses when it starts a run.
// Moved out of unified-intake.tsx unchanged so the Review page's Re-run declares a
// source exactly the way intake would when the run's own registration is unknown.

const DECLARED_FORMAT_BY_EXTENSION: Record<string, string> = {
  xml: "xml",
  json: "message_export_json",
  md: "markdown",
  txt: "delimited_text",
  csv: "delimited_text",
  pdf: "pdf",
  png: "image",
  jpg: "image",
  jpeg: "image",
  gif: "image",
  webp: "image",
  avif: "image",
  tif: "image",
  tiff: "image",
  bmp: "image",
  docx: "docx",
  html: "html",
  htm: "html",
  zip: "archive",
  tar: "archive",
  tgz: "archive",
  gz: "archive",
  "7z": "archive",
  rar: "archive",
};

export function declaredFormat(source: { name: string }) {
  const extension = source.name.split(".").pop()?.toLowerCase();
  return DECLARED_FORMAT_BY_EXTENSION[extension ?? ""] ?? "unknown_binary";
}
