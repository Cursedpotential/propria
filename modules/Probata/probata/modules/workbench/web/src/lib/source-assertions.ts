// Byline: Claude Code · Opus 5.5 · 2026-09-25
// Operator source-context assertions shared by Intake (first entry) and Review
// (read, correct, re-run). Pure helpers only; the form is source-assertions-fields.tsx.
import type {
  ProfferHumanSourceAssertions,
  ProfferObservedSource,
  ProfferSourceRegistration,
} from "@/lib/shared/types";

export const EMPTY_ASSERTIONS: ProfferHumanSourceAssertions = {
  source_class: "unknown",
  source_principal: "",
  other_party: "",
  acquired_at: null,
  acquisition_method: "",
  acquisition_authority: "",
  source_device: "",
  device_custodian: "",
  occurred_start: "",
  occurred_end: "",
  date_certainty: "",
  context: "",
  notes: "",
};

const SHA256_HEX = /^[0-9a-f]{64}$/;

function pad(value: number) {
  return String(value).padStart(2, "0");
}

/** RFC3339 -> the `datetime-local` value the form edits (local wall time, minutes). */
function localInputFromIso(value: string | null | undefined): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** A stored revision as editable form state. */
export function assertionsForForm(stored: ProfferHumanSourceAssertions | null | undefined): ProfferHumanSourceAssertions {
  if (!stored) return { ...EMPTY_ASSERTIONS };
  return { ...EMPTY_ASSERTIONS, ...stored, acquired_at: localInputFromIso(stored.acquired_at) };
}

/**
 * Form state as the API body. An untouched "when acquired" keeps its exact stored
 * value (the form edits minutes, the record may carry seconds), so re-saving never
 * silently rewrites a timestamp the operator did not change.
 */
export function assertionsForSubmit(
  form: ProfferHumanSourceAssertions,
  stored?: ProfferHumanSourceAssertions | null,
): ProfferHumanSourceAssertions {
  let acquiredAt: string | null = null;
  if (form.acquired_at) {
    acquiredAt = stored?.acquired_at && localInputFromIso(stored.acquired_at) === form.acquired_at
      ? stored.acquired_at
      : new Date(form.acquired_at).toISOString();
  }
  return { ...form, acquired_at: acquiredAt };
}

const SOURCE_CLASS_LABEL: Record<ProfferHumanSourceAssertions["source_class"], string> = {
  unknown: "Unknown relationship",
  first_party: "First party / mine",
  acquired_third_party: "Acquired third party",
};

/** The filled-in assertions as short label/value rows for a read-only summary. */
export function assertionSummary(assertions: ProfferHumanSourceAssertions | null | undefined): Array<[string, string]> {
  if (!assertions) return [];
  const rows: Array<[string, string]> = [["Relationship", SOURCE_CLASS_LABEL[assertions.source_class] ?? assertions.source_class]];
  const text: Array<[string, string | null | undefined]> = [
    ["Other party", assertions.other_party],
    ["Came from", assertions.source_principal],
    ["How acquired", assertions.acquisition_method?.replaceAll("_", " ")],
    ["When acquired", assertions.acquired_at ? new Date(assertions.acquired_at).toLocaleString() : ""],
    ["Authority", assertions.acquisition_authority?.replaceAll("_", " ")],
    ["Dates", [assertions.occurred_start, assertions.occurred_end].filter(Boolean).join(" to ")],
    ["Date certainty", assertions.date_certainty],
    ["Device", assertions.source_device],
    ["Device held by", assertions.device_custodian],
    ["Context", assertions.context],
    ["Notes", assertions.notes],
  ];
  for (const [label, value] of text) if (value && value.trim()) rows.push([label, value.trim()]);
  return rows;
}

/** The object key (or upload digest) a source reference names, as the engine checks it. */
function sourceKey(sourceRef: string): string | null {
  const match = /^([a-z0-9]+):\/\/([^/]+)\/?(.*)$/i.exec(sourceRef.trim());
  if (!match) return null;
  const [, scheme, host, path] = match;
  if (scheme.toLowerCase() === "upload") return host.toLowerCase();
  try {
    return decodeURIComponent(path) || null;
  } catch {
    return path || null;
  }
}

/**
 * The observation a first context revision is bound to, built from what the engine
 * recorded when it retained this run's original (digest and length). Null when the
 * run never retained its original: there is then nothing truthful to bind to.
 */
export function observationFromRegistration(
  sourceRef: string,
  registration: ProfferSourceRegistration | null | undefined,
): ProfferObservedSource | null {
  const sha = registration?.original_sha256?.toLowerCase();
  const length = registration?.original_bytes;
  if (!sha || !SHA256_HEX.test(sha) || length === null || length === undefined || length < 0) return null;
  const key = sourceKey(sourceRef);
  if (!key) return null;
  const name = registration?.original_filename?.trim() || key.split("/").filter(Boolean).at(-1) || key;
  return {
    key,
    name,
    byte_length: length,
    etag: `sha256:${sha}`,
    preview_sha256: sha,
    verification_state: "preview_only",
  };
}
