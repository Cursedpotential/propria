// Byline: Codex · GPT-5 · 2026-09-12 (URL-hydrated, single canonical case shell context)
"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

import { ApiError, getMatter, listMatters } from "@/lib/api-client";
import { parseOperatingMode, selectAdmittedCourtCase } from "@/lib/operating-mode";
import type { CourtCase, MatterDetail, MatterMode } from "@/lib/shared/types";

type FixedCaseContextValue = {
  matter: MatterDetail | null;
  primaryCourtCase: CourtCase | null;
  loading: boolean;
  error: string | null;
  mode: MatterMode;
  setMode: (mode: MatterMode) => void;
};

const FixedCaseContext = createContext<FixedCaseContextValue | null>(null);

function errorText(error: unknown) {
  return error instanceof ApiError
    ? error.message
    : error instanceof Error
      ? error.message
      : "The fixed case could not be loaded";
}

function initialMatterMode(): { mode: MatterMode; error: string | null } {
  try {
    const requestedMode = typeof window === "undefined" ? null : new URLSearchParams(window.location.search).get("mode");
    return { mode: parseOperatingMode(requestedMode), error: null };
  } catch (error) {
    return { mode: "LIVE", error: errorText(error) };
  }
}

export function FixedCaseProvider({ children }: { children: React.ReactNode }) {
  const [matter, setMatter] = useState<MatterDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [initialSelection] = useState(initialMatterMode);
  const [invalidMode, setInvalidMode] = useState(initialSelection.error);
  const [mode, setModeState] = useState<MatterMode>(initialSelection.mode);

  const setMode = useCallback((nextMode: MatterMode) => {
    if (nextMode === mode && !invalidMode) return;
    setInvalidMode(null);
    setMatter(null);
    setError(null);
    setLoading(true);
    setModeState(nextMode);
  }, [mode, invalidMode]);

  useEffect(() => {
    if (invalidMode) return;
    let cancelled = false;

    listMatters(50, 0, mode)
      .then((response) => {
        if (response.total === 0) {
          throw new Error(
            "The Platform has no case. Restore the single canonical case before intake continues.",
          );
        }
        if (response.total !== 1 || response.data.length !== 1) {
          throw new Error(
            `The Platform returned ${response.total} Matters. This is split or duplicated case data, not a choice for the operator.`,
          );
        }
        if (response.data[0].matter_mode !== mode) {
          throw new Error(`The Platform did not return a matter explicitly scoped to ${mode}. Mode switching is blocked to prevent DEV/LIVE bleed.`);
        }
        return getMatter(response.data[0].id, mode);
      })
      .then((fixedMatter) => {
        if (fixedMatter.matter_mode !== mode) {
          throw new Error(`The selected matter did not confirm ${mode} mode. Mode switching is blocked to prevent DEV/LIVE bleed.`);
        }
        if (!cancelled) {
          setMatter(fixedMatter);
          setError(null);
        }
      })
      .catch((requestError) => {
        if (!cancelled) setError(errorText(requestError));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [mode, invalidMode]);

  const value = useMemo<FixedCaseContextValue>(
    () => ({
      matter,
      primaryCourtCase: selectAdmittedCourtCase(matter),
      loading,
      error,
      mode,
      setMode,
    }),
    [matter, loading, error, mode, setMode],
  );

  return (
    <FixedCaseContext.Provider value={value}>
      {invalidMode ? <div role="alert">{invalidMode}</div> : children}
    </FixedCaseContext.Provider>
  );
}

export function useFixedCase() {
  const context = useContext(FixedCaseContext);
  if (!context) throw new Error("useFixedCase must be used within FixedCaseProvider");
  return context;
}

/** Read desktop policy or validate the mobile URL policy; omission is Live.
 * Input: optional case context. Output: policy. Effects: none.
 * Use for shared desktop/mobile identity actions, not required case data reads.
 */
export function useOperatingMode(): MatterMode {
  const context = useContext(FixedCaseContext);
  if (context) return context.mode;
  return parseOperatingMode(typeof window === "undefined" ? null : new URLSearchParams(window.location.search).get("mode"));
}
