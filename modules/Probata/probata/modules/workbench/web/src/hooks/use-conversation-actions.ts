// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Queries and actions behind Extract, Send to Surreal and the Extractions view, for desktop and /m alike.
"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";

import { conversationApi, newIdempotencyKey } from "@/lib/conversation-actions-client";

/** The registry changes with a deploy, not with a click. */
export function useExtractors() {
  return useQuery({ queryKey: ["extractors"], queryFn: ({ signal }) => conversationApi.extractors(signal), staleTime: 5 * 60_000 });
}

/** A set of checked conversation ids. */
export function useSelection() {
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const toggle = useCallback((id: string) => {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);
  const clear = useCallback(() => setSelected(new Set()), []);
  const setAll = useCallback((ids: readonly string[]) => setSelected(new Set(ids)), []);
  return { selected, toggle, clear, setAll };
}

/** Start extraction. One Idempotency-Key per click; the mutation keeps it for an automatic retry of that click. */
export function useStartExtraction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ threadIds, extractors }: { threadIds: string[]; extractors: string[] }) =>
      conversationApi.startExtraction(threadIds, extractors, newIdempotencyKey("extract")),
    onSuccess: (_started, { threadIds }) => {
      for (const id of threadIds) void queryClient.invalidateQueries({ queryKey: ["thread-extractions", id] });
    },
  });
}

/** Start the send to surreal-case. */
export function useStartSend() {
  return useMutation({
    mutationFn: ({ threadIds, includeExtractions }: { threadIds: string[]; includeExtractions: boolean }) =>
      conversationApi.startSend(threadIds, includeExtractions, newIdempotencyKey("send")),
  });
}

/** Polls a workflow until it has an outcome other than running. Refreshes the extraction views as steps finish. */
export function useActionStatus(workflowId: string | null, threadIds: readonly string[] = []) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: ["conversation-workflow", workflowId] as const,
    enabled: Boolean(workflowId),
    queryFn: async ({ signal }) => {
      const status = await conversationApi.status(workflowId as string, signal);
      for (const id of threadIds) void queryClient.invalidateQueries({ queryKey: ["thread-extractions", id] });
      return status;
    },
    refetchInterval: (query) => (query.state.data && query.state.data.outcome !== "running" ? false : 2500),
  });
}

/** What the extractors found in one conversation. Keeps refreshing while any extractor is still running. */
export function useThreadExtractions(threadId: string, enabled = true) {
  return useQuery({
    queryKey: ["thread-extractions", threadId] as const,
    enabled,
    queryFn: ({ signal }) => conversationApi.extractions(threadId, signal),
    refetchInterval: (query) => (query.state.data?.extractors.some((group) => group.status === "running") ? 4000 : false),
  });
}
