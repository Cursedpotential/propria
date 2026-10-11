import { ActivityOperationsLedger } from "@/components/activity/activity-operations-ledger";

// Byline: Codex · GPT-6 · 2026-10-06
/** Keep the legacy Intake embedding pointed at the shared Activity ledger.
 * Inputs: none.
 * Output: the durable Proffer operation list.
 * Side effects: reads operation data through ActivityOperationsLedger.
 * Use the Activity route for the full page; use this wrapper only for legacy Intake composition.
 */
export function ProfferOperationsTable() {
  return <ActivityOperationsLedger />;
}
