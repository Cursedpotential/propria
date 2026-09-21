// Byline: Claude Code · Opus 5 · 2026-09-20 (TanStack Query host for cursor-paged operator reads)
import { QueryClient } from "@tanstack/react-query";

/**
 * One browser-scoped query cache. Read-only projections only: nothing here may
 * promote, approve, or rewrite evidence. PostgreSQL remains canonical.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
});
