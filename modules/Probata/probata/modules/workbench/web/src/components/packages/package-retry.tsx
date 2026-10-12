// Byline: Codex · GPT-6 · 2026-10-12.
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { packageCapability, retryContextPackage, type PackageCapability } from "@/lib/context-package-client";

/** Retry extraction only through an available backend capability and configured model choices.
 * Inputs: package handle, actual capability inventory and surrounding dirty state.
 * Output: working retry form/receipt or nothing when no retry writer is advertised.
 * Effects: explicit POST and package-only invalidation. Pick for configured model retry;
 * source originals and previous extraction attempts remain owned by the backend workflow.
 */
export function PackageRetry({ handle, capabilities, disabled, onDirtyChange, onPendingChange }: { handle: string; capabilities?: Array<string | PackageCapability>; disabled: boolean; onDirtyChange: (value: boolean) => void; onPendingChange: (value: boolean) => void }) {
  const client = useQueryClient();
  const [model, setModel] = useState("");
  const [reason, setReason] = useState("");
  const keys = useRef(new Map<string, string>());
  const mutation = useMutation({ mutationFn: ({ capability, fields, key }: { capability: PackageCapability; fields: Record<string, unknown>; key: string }) => retryContextPackage(capability, fields, key), retry: false,
    onSuccess: async () => { await client.invalidateQueries({ queryKey: ["context-package", handle] }); },
  });
  const capability = packageCapability(capabilities, "retry_extraction") ?? packageCapability(capabilities, "retry");
  if (!capability?.path || capability.method?.toUpperCase() !== "POST") return null;
  const models = Array.isArray(capability.models) ? capability.models.flatMap((value): Array<{ id: string; label: string }> => {
    if (typeof value === "string") return [{ id: value, label: value }];
    if (value && typeof value === "object" && "id" in value && typeof value.id === "string") return [{ id: value.id, label: "label" in value && typeof value.label === "string" ? value.label : value.id }];
    return [];
  }) : [];
  const defaults = capability.body && typeof capability.body === "object" && !Array.isArray(capability.body) ? capability.body as Record<string, unknown> : {};
  if (!models.length && typeof defaults.model !== "string") return null;
  const selected = model || (typeof defaults.model === "string" ? defaults.model : "");

  /** Dispatch one actual retry and retain the action key after a network failure.
   * Input: current server-provided model choice and owner reason. Output: workflow receipt.
   * Effects: explicit mutation. Pick instead of starting a parser workflow as a model retry.
   */
  const retry = async () => {
    const fields = { model: selected, reason: reason.trim() };
    const signature = JSON.stringify(fields);
    let key = keys.current.get(signature);
    if (!key) { key = crypto.randomUUID(); keys.current.set(signature, key); }
    onPendingChange(true);
    try { await mutation.mutateAsync({ capability, fields, key }); setReason(""); setModel(""); onDirtyChange(false); }
    catch { /* The mutation's actual error stays visible and drafts remain intact. */ }
    finally { onPendingChange(false); }
  };
  return <details className="rounded border bg-card p-3 text-xs"><summary className="cursor-pointer font-medium">Retry model extraction</summary>
    <div className="mt-3 flex flex-wrap items-end gap-2">
      {models.length > 0 ? <label>Configured model<select aria-label="Configured extraction model" className="mt-1 block rounded border bg-background p-2" value={selected} disabled={disabled || mutation.isPending} onChange={(event) => { setModel(event.target.value); onDirtyChange(Boolean(event.target.value || reason)); }}>
        <option value="">Choose a configured model</option>{models.map((entry) => <option key={entry.id} value={entry.id}>{entry.label}</option>)}
      </select></label> : <p>Configured model: {selected}</p>}
      <label className="min-w-48 flex-1">Reason<Input aria-label="Extraction retry reason" className="mt-1" value={reason} disabled={disabled || mutation.isPending} onChange={(event) => { setReason(event.target.value); onDirtyChange(Boolean(event.target.value || model)); }} /></label>
      <Button size="sm" disabled={disabled || mutation.isPending || !selected || !reason.trim()} onClick={() => void retry()}>{mutation.isPending ? "Starting retry…" : "Retry extraction"}</Button>
      {(reason || model) && <Button size="sm" variant="ghost" disabled={mutation.isPending} onClick={() => { setReason(""); setModel(""); onDirtyChange(false); }}>Discard retry draft</Button>}
    </div>
    {mutation.error && <p role="alert" className="mt-2 text-destructive">{mutation.error.message}</p>}
    {mutation.data && <details className="mt-2"><summary className="cursor-pointer">Extraction retry receipt</summary><pre className="mt-2 whitespace-pre-wrap break-all">{JSON.stringify(mutation.data, null, 2)}</pre></details>}
  </details>;
}
