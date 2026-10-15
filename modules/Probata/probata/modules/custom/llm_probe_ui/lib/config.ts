// Use the bundled same-origin API by default. An explicit public URL preserves
// the old separate UI deployment during cutover. Provider credentials remain
// in FastAPI; this public routing value never contains a provider key.
export const LLM_PROBE_URL =
  process.env.NEXT_PUBLIC_LLM_PROBE_URL || "/api/probe";
