// Byline: Claude Code · Opus 5.5 · 2026-09-25 (entity + event extraction queries and owner actions)
"use client";

import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  commitEntities,
  correctEntity,
  correctEvent,
  getEntityProposals,
  getWorkflowProgress,
  markEventFromRecord,
  newIdempotencyKey,
  startEntityExtraction,
  validateEntityCommit,
  type EntityCorrection,
  type EventCorrection,
} from "@/lib/entity-extraction-client";
import type { MatterMode } from "@/lib/shared/types";

const proposalsKey = (mode: MatterMode, previewHandle: string) => ["entities", "proposals", mode, previewHandle] as const;
/** Every owner write to a run's proposals, from any component, shares this key. */
const writesKey = (mode: MatterMode, previewHandle: string) => ["entities", "write", mode, previewHandle] as const;

/** True while any write to this run's proposals (and its refetch) is in flight. */
export function useEntityWritesPending(previewHandle: string, mode: MatterMode) {
  return useIsMutating({ mutationKey: writesKey(mode, previewHandle) }) > 0;
}

export function useEntityProposals(previewHandle: string, mode: MatterMode) {
  return useQuery({
    queryKey: proposalsKey(mode, previewHandle),
    enabled: Boolean(previewHandle),
    queryFn: ({ signal }) => getEntityProposals(previewHandle, mode, signal),
  });
}

/** Polls a workflow while it runs; stops once it has an outcome. */
export function useWorkflowProgress(kind: "extraction" | "commit", workflowId: string | null, previewHandle: string, mode: MatterMode) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: ["entities", kind, mode, previewHandle, workflowId] as const,
    enabled: Boolean(workflowId && previewHandle),
    queryFn: async ({ signal }) => {
      const progress = await getWorkflowProgress(kind, workflowId as string, previewHandle, mode, signal);
      // Every finished step may have staged or promoted rows: refresh them.
      void queryClient.invalidateQueries({ queryKey: proposalsKey(mode, previewHandle) });
      return progress;
    },
    refetchInterval: (query) => (query.state.data && query.state.data.outcome !== "running" ? false : 2500),
  });
}

/** Owner actions. Each mutation mints its Idempotency-Key once, when the
 * owner acts; a TanStack retry reuses the same variables and so the same key. */
export function useEntityActions(previewHandle: string, mode: MatterMode) {
  const queryClient = useQueryClient();
  // Returning the refetch keeps each write pending until the new proposals arrive.
  const refresh = () => queryClient.invalidateQueries({ queryKey: proposalsKey(mode, previewHandle) });
  const mutationKey = writesKey(mode, previewHandle);
  const extract = useMutation({
    mutationKey,
    mutationFn: ({ useModel, key }: { useModel: boolean; key: string }) => startEntityExtraction(previewHandle, mode, useModel, key),
    onSuccess: refresh,
  });
  const entity = useMutation({
    mutationKey,
    mutationFn: ({ correction, key }: { correction: EntityCorrection; key: string }) => correctEntity(previewHandle, mode, correction, key),
    onSuccess: refresh,
  });
  const event = useMutation({
    mutationKey,
    mutationFn: ({ correction, key }: { correction: EventCorrection; key: string }) => correctEvent(previewHandle, mode, correction, key),
    onSuccess: refresh,
  });
  const mark = useMutation({
    mutationKey,
    mutationFn: ({ recordId, title, key }: { recordId: string; title?: string; key: string }) =>
      markEventFromRecord(previewHandle, mode, recordId, title ? { title } : {}, key),
    onSuccess: refresh,
  });
  const validate = useMutation({ mutationFn: () => validateEntityCommit(previewHandle, mode) });
  const commit = useMutation({
    mutationKey,
    mutationFn: ({ digest, key }: { digest: string; key: string }) => commitEntities(previewHandle, mode, digest, key),
    onSuccess: refresh,
  });
  return {
    extract, entity, event, mark, validate, commit,
    correctEntity: (correction: EntityCorrection) => entity.mutateAsync({ correction, key: newIdempotencyKey(`entity-${correction.op}`) }),
    correctEvent: (correction: EventCorrection) => event.mutateAsync({ correction, key: newIdempotencyKey(`event-${correction.op}`) }),
    markEvent: (recordId: string, title?: string) => mark.mutateAsync({ recordId, title, key: newIdempotencyKey("mark") }),
  };
}
