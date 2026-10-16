// Byline: Codex · GPT-6 · 2026-10-11.
import { DiscoveryReview } from "@/components/review/discovery-review";
import { ErrorBox, Loading } from "@/components/mobile/mobile-ui";
import { useFixedCase } from "@/lib/fixed-case-context";

/** Open optional discovery with the shell's admitted actual case and operating policy.
 * Inputs: case context. Output: scoped review workspace. Effects: child requests only;
 * choose for run-local discovery definitions and feedback, never source processing.
 */
export default function DiscoveryPage() {
  const { matter, primaryCourtCase, mode, loading, error } = useFixedCase();
  if (loading) return <Loading label="Loading case" />;
  if (error || !matter || !primaryCourtCase) return <ErrorBox error={error || "The admitted case is unavailable"} />;
  const scope = { mode, matter_id: matter.id, court_case_id: primaryCourtCase.id };
  return <DiscoveryReview key={`${mode}:${matter.id}:${primaryCourtCase.id}`} scope={scope} />;
}
