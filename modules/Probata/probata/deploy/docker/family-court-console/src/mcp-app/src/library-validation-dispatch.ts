// Byline: Codex, GPT-6, 2026-10-04. Shared durable proposal to tracked validation bridge.
import { readFileSync } from "node:fs";
import type { StoreOk } from "./store.js";

/** Queue validation of an already retained shared proposal without sending personal bodies to Temporal.
 * Inputs: shared proposal id and mounted starter endpoint/token configuration. Outputs: explicit queued or retryable state.
 * Effects: authenticated idempotent workflow start and proposal dispatch metadata; publication remains blocked on a trusted receipt.
 * Choose after proposal creation or for a failed-queue retry; a missing service cannot discard the saved edit.
 */
export async function queueLibraryValidation(store: StoreOk, proposalId: string): Promise<Record<string, unknown>> {
  if (!/^library_proposal:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/.test(proposalId))
    throw new Error("Invalid library proposal identity");
  const state = (await store.db.query("SELECT status, citations FROM ONLY type::record('library_proposal', $key);", { key: proposalId.split(":")[1] })).at(-1) as { status?: string; citations?: unknown[] } | undefined;
  if (!state) throw new Error("Saved library proposal missing");
  if (state.status === "published") return { state: "already_published", retryable: false };
  if (!state.citations?.length) return { state: "citation_required", reason: "Add claim citations to the retained imported draft before validation.", retryable: false };
  const endpoint = process.env.TOOLKIT_VALIDATION_START_URL;
  const tokenFile = process.env.TOOLKIT_VALIDATION_TOKEN_FILE;
  let outcome: Record<string, unknown>;
  let failure = "VALIDATION_CONFIGURATION";
  try {
    if (!endpoint || !tokenFile) throw new Error("Validation service is not configured");
    const url = new URL(endpoint);
    if (url.username || url.password || url.search || url.hash || url.pathname !== "/toolkit/library/validate"
      || !["https:", "http:"].includes(url.protocol)) throw new Error("Invalid validation service endpoint");
    if (url.protocol === "http:" && !/^100\.(?:6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])\.(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$/.test(url.hostname))
      throw new Error("Plain HTTP validation must use the private tailnet");
    failure = "VALIDATION_CREDENTIAL";
    const token = readFileSync(tokenFile, "utf8").trim();
    if (token.length < 32 || token.length > 4098 || /[\r\n]/.test(token)) throw new Error("Invalid mounted validation credential");
    failure = "VALIDATION_NETWORK";
    const response = await fetch(url, {
      method: "POST", redirect: "error", signal: AbortSignal.timeout(15000),
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "Idempotency-Key": proposalId },
      body: JSON.stringify({ proposal_id: proposalId }),
    });
    if (!response.ok) { failure = `VALIDATION_HTTP_${response.status}`; throw new Error("Validation starter refused the request"); }
    failure = "VALIDATION_RESPONSE";
    const result = await response.json() as Record<string, unknown>;
    if (typeof result.workflow_id !== "string" || typeof result.run_id !== "string")
      throw new Error("Validation starter returned an invalid workflow envelope");
    outcome = { state: "queued", workflow_id: result.workflow_id, run_id: result.run_id, retryable: false };
  } catch {
    // Remote bodies and credential-related exception text never enter public responses or logs.
    outcome = { state: "queue_failed", code: failure, reason: `Saved edit requires validation; service dispatch failed (${failure}). Retry validation.`, retryable: true };
  }
  await store.db.query("UPDATE type::record('library_proposal', $key) SET dispatch = $outcome, dispatch_at = time::now();",
    { key: proposalId.split(":")[1], outcome });
  return outcome;
}
