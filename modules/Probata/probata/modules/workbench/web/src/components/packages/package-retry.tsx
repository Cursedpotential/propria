// Byline: Codex · GPT-6 · 2026-10-12.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { getPackageRetryStatus, packageCapability, retryContextPackage, type PackageCapability } from "@/lib/context-package-client";

/** Retry extraction only through an available backend capability and configured model choices.
 * Inputs: package handle, actual capability inventory and surrounding dirty state.
 * Output: working retry form/receipt or nothing when no retry writer is advertised.
 * Effects: explicit POST and package-only invalidation. Pick for configured model retry;
 * source originals and previous extraction attempts remain owned by the backend workflow.
 */
export function PackageRetry({ handle, capabilities, disabled, onDirtyChange, onPendingChange }: { handle: string; capabilities?: Array<string | PackageCapability>; disabled: boolean; onDirtyChange: (value: boolean) => void; onPendingChange: (value: boolean) => void }) {
  const client = useQueryClient();
  const [model, setModel] = useState("");
  const keys = useRef(new Map<string, string>());
  const mutation = useMutation({ mutationFn: ({ capability, fields, key }: { capability: PackageCapability; fields: Record<string, unknown>; key: string }) => retryContextPackage(capability, fields, key), retry: false,
    onSuccess: async () => { await client.invalidateQueries({ queryKey: ["context-package", handle] }); },
  });
  const statusPath = typeof mutation.data?.status_path === "string" ? mutation.data.status_path : "";
  const attempt = useQuery({ queryKey: ["context-package-attempt", handle, statusPath], enabled: Boolean(statusPath),
    queryFn: ({ signal }) => getPackageRetryStatus(statusPath, signal), retry: false,
    refetchInterval: (query) => ["complete", "completed", "failed", "partial", "partial_enrichment", "needs_parser", "cancelled", "canceled"].includes(String(query.state.data?.status ?? "")) ? false : query.state.error ? false : 3000,
  });
  const businessStatus = attempt.data?.status;
  useEffect(() => {
    if (!disabled && statusPath && ["complete", "completed", "failed", "partial", "partial_enrichment"].includes(String(businessStatus))) void client.invalidateQueries({ queryKey: ["context-package", handle] });
  }, [businessStatus, statusPath, client, handle, disabled]);
  const capability = packageCapability(capabilities, "retry_extraction") ?? packageCapability(capabilities, "retry");
  const models = Array.isArray(capability?.models) ? capability.models.flatMap((value): Array<{ id: string; label: string }> => {
    if (typeof value === "string") return [{ id: value, label: value }];
    if (value && typeof value === "object" && "id" in value && typeof value.id === "string") return [{ id: value.id, label: "label" in value && typeof value.label === "string" ? value.label : value.id }];
    return [];
  }) : [];
  const defaults = capability?.body && typeof capability.body === "object" && !Array.isArray(capability.body) ? capability.body as Record<string, unknown> : {};
  const canRetry = Boolean(capability?.path && capability.method?.toUpperCase() === "POST" && (models.length || typeof defaults.model_id === "string"));
  if (!canRetry && !mutation.data) return null;
  const selected = model || (typeof defaults.model_id === "string" ? defaults.model_id : "");

  /** Dispatch one actual retry and retain the action key after a network failure.
   * Input: current server-provided model choice and source version. Output: workflow receipt.
   * Effects: explicit mutation. Pick instead of starting a parser workflow as a model retry.
   */
  const retry = async () => {
    if (!capability || !canRetry) return;
    const fields = { model_id: selected };
    const signature = JSON.stringify({ ...defaults, ...fields });
    let key = keys.current.get(signature);
    if (!key) { key = crypto.randomUUID(); keys.current.set(signature, key); }
    onPendingChange(true);
    try { await mutation.mutateAsync({ capability, fields, key }); keys.current.delete(signature); setModel(""); onDirtyChange(false); }
    catch { /* The mutation's actual error stays visible and drafts remain intact. */ }
    finally { onPendingChange(false); }
  };
  return <details className="rounded border bg-card p-3 text-xs"><summary className="cursor-pointer font-medium">Retry model extraction</summary>
    {canRetry && <div className="mt-3 flex flex-wrap items-end gap-2">
      {models.length > 0 ? <label>Configured model<select aria-label="Configured extraction model" className="mt-1 block rounded border bg-background p-2" value={selected} disabled={disabled || mutation.isPending} onChange={(event) => { setModel(event.target.value); onDirtyChange(Boolean(event.target.value)); }}>
        <option value="">Choose a configured model</option>{models.map((entry) => <option key={entry.id} value={entry.id}>{entry.label}</option>)}
      </select></label> : <p>Configured model: {selected}</p>}
      <Button size="sm" disabled={disabled || mutation.isPending || !selected} onClick={() => void retry()}>{mutation.isPending ? "Starting retry…" : "Retry extraction"}</Button>
      {model && <Button size="sm" variant="ghost" disabled={mutation.isPending} onClick={() => { setModel(""); onDirtyChange(false); }}>Discard retry draft</Button>}
    </div>}
    {mutation.error && <p role="alert" className="mt-2 text-destructive">{mutation.error.message}</p>}
    {statusPath && <section className="mt-3 space-y-2" aria-label="Extraction retry business progress">
      <p role="status" className={attempt.data?.status === "failed" || String(attempt.data?.status).startsWith("partial") ? "font-semibold text-destructive" : "font-semibold"}>Extraction business status: {typeof attempt.data?.status === "string" ? attempt.data.status : "awaiting status read"}</p>
      {typeof attempt.data?.reason === "string" && attempt.data.reason && <p className="whitespace-pre-wrap">{attempt.data.reason}</p>}
      {attempt.error && <p role="alert" className="text-destructive">{attempt.error.message}</p>}
      <Button size="sm" variant="outline" disabled={attempt.isFetching} onClick={() => void attempt.refetch()}>Refresh attempt status</Button>
      {attempt.data && <details><summary className="cursor-pointer">Complete attempt progress and source references</summary><pre className="mt-2 whitespace-pre-wrap break-all">{JSON.stringify(attempt.data, null, 2)}</pre></details>}
    </section>}
    {mutation.data && <details className="mt-2"><summary className="cursor-pointer">Extraction retry receipt</summary><pre className="mt-2 whitespace-pre-wrap break-all">{JSON.stringify(mutation.data, null, 2)}</pre></details>}
  </details>;
}
