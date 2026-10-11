import { useQuery } from "@tanstack/react-query";

import { CancelRunSection } from "@/components/sbv/cancel-run-section";
import { getProfferOperatorSnapshot } from "@/lib/api-client";
import type { MatterMode } from "@/lib/shared/types";

const PREVIEW_HANDLE_PATTERN = /^[A-Za-z0-9_-]{32,128}$/;

// Byline: Codex · GPT-6 · 2026-10-06
/** Load the exact mode-bound operator snapshot before exposing cancellation.
 * Inputs: preview handle and Dev/Live mode.
 * Output: existing CancelRunSection only when the snapshot matches both inputs.
 * Side effects: reads the operator snapshot; CancelRunSection owns any explicit cancel action.
 * Use for Activity cancellation; never infer cancel eligibility from a list row alone.
 */
export function OperationCancelControl({ previewHandle, mode }: { previewHandle: string; mode: MatterMode }) {
  const validHandle = PREVIEW_HANDLE_PATTERN.test(previewHandle);
  const snapshot = useQuery({
    queryKey: ["activity", "operator-snapshot", mode, previewHandle],
    queryFn: ({ signal }) => getProfferOperatorSnapshot(previewHandle, mode, signal),
    enabled: validHandle,
    retry: false,
    refetchInterval: (query) => query.state.data?.terminal ? false : 5_000,
  });

  if (!validHandle) return <p className="text-xs text-muted-foreground">This attempt reference cannot be reopened for controls.</p>;
  if (snapshot.isPending) return <p className="text-xs text-muted-foreground" role="status">Checking this attempt in {mode} mode…</p>;
  if (snapshot.error) return <p className="text-xs text-destructive" role="alert">Run controls are unavailable: {snapshot.error.message}</p>;
  const exactSnapshot = snapshot.data;
  if (!exactSnapshot || exactSnapshot.preview_handle !== previewHandle || exactSnapshot.matter_mode !== mode) {
    return <p className="text-xs text-destructive" role="alert">Run controls are unavailable because the returned attempt or mode did not match.</p>;
  }

  return <CancelRunSection snapshot={exactSnapshot} />;
}
