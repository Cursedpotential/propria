// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// `queryOptions()` factories (TanStack Query best practice `ts-query-options-loader`
// / `load-ensure-query-data`) — route loaders call `queryClient.ensureQueryData(...)`
// with these, and components call `useSuspenseQuery(...)`/`useQuery(...)` with the
// SAME options object, so cache keys always match between loader and component.

import { queryOptions } from "@tanstack/react-query";
import { authApi, storeApi } from "./api-client";
import type { TimelineMode } from "@/types/store";

export const summaryQuery = () =>
  queryOptions({
    queryKey: ["store", "summary"],
    queryFn: storeApi.summary,
    staleTime: 15_000,
  });

export const authStatusQuery = () =>
  queryOptions({
    queryKey: ["auth", "status"],
    queryFn: authApi.status,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

export const docketQuery = () =>
  queryOptions({
    queryKey: ["store", "docket"],
    queryFn: storeApi.docket,
    staleTime: 15_000,
  });

export const timelineQuery = (mode: TimelineMode, knownBy?: string) =>
  queryOptions({
    queryKey: ["store", "timeline", mode, knownBy ?? null],
    queryFn: () => storeApi.timeline({ mode, knownBy }),
    staleTime: 15_000,
  });

export const memosQuery = () =>
  queryOptions({
    queryKey: ["store", "memos"],
    queryFn: storeApi.memos,
    staleTime: 15_000,
  });

export const evidenceQuery = () =>
  queryOptions({
    queryKey: ["store", "evidence"],
    queryFn: storeApi.evidence,
    staleTime: 15_000,
  });

export const evalsQuery = () =>
  queryOptions({
    queryKey: ["store", "evals"],
    queryFn: storeApi.evals,
    staleTime: 15_000,
  });

export const referenceQuery = (match?: string) =>
  queryOptions({
    queryKey: ["store", "reference", match ?? null],
    queryFn: () => storeApi.reference(match),
    staleTime: 15_000,
  });

export const factorMapQuery = () =>
  queryOptions({
    queryKey: ["store", "factor-map"],
    queryFn: storeApi.factorMap,
    staleTime: 30_000,
  });

export const searchQuery = (q: string) =>
  queryOptions({
    queryKey: ["store", "search", q],
    queryFn: () => storeApi.search({ q }),
    enabled: q.trim().length > 0,
    staleTime: 5_000,
  });
