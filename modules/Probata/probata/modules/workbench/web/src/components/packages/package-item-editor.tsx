// Byline: Codex · GPT-6 · 2026-10-12.
import { useRef, useState } from "react";
import { MetadataFieldTable, type CorrectionSubmit } from "@/components/metadata/metadata-field-table";
import { PackageItemContent } from "@/components/packages/package-content";
import { PackageRelations } from "@/components/packages/package-relations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { packageCapability, type ContextPackageItem, type PackageReviewInput } from "@/lib/context-package-client";
import { searchCommittedEntities, type RegistryEntity } from "@/lib/entity-extraction-client";

export interface PackageItemEditorProps {
  item: ContextPackageItem;
  onSave: (input: PackageReviewInput, key: string) => Promise<Record<string, unknown>>;
  onDirtyChange: (dirty: boolean) => void;
  onSelect: (id: string) => void;
}

/** Inspect/correct a complete item using the established metadata correction table.
 * Inputs: exact item revision, durable save function and dirty/selection callbacks.
 * Output: all fields, source locators, real edges, human actions and saved receipt.
 * Effects: clicked revision-guarded writes and explicit registry lookup. Pick for any
 * package section; failed saves keep drafts, source originals are never overwritten.
 */
export function PackageItemEditor({ item, onSave, onDirtyChange, onSelect }: PackageItemEditorProps) {
  const [pendingField, setPendingField] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<Record<string, unknown> | null>(null);
  const [reason, setReason] = useState("");
  const [patch, setPatch] = useState("");
  const [targetQuery, setTargetQuery] = useState("");
  const [targets, setTargets] = useState<RegistryEntity[]>([]);
  const [target, setTarget] = useState("");
  const [searching, setSearching] = useState(false);
  const [fieldDirty, setFieldDirty] = useState(false);
  const keys = useRef(new Map<string, string>());
  const edit = packageCapability(item.capabilities, "edit") ?? packageCapability(item.capabilities, "metadata");
  const supplement = packageCapability(item.capabilities, "supplement") ?? edit;
  const reassociate = packageCapability(item.capabilities, "reassociate");
  const actions = ["accept", "reject", "commit"].filter((operation) => packageCapability(item.capabilities, operation));
  const busy = pendingField !== null;

  /** Retain one idempotency key for an unchanged failed submission and show the durable receipt.
   * Inputs: operation, fields and reason. Output: saved boolean. Effects: explicit parent mutation.
   * Pick for every editor action; failures leave all input fields intact for correction/retry.
   */
  const submit = async (operation: string, fields: Record<string, unknown> | undefined, note: string, extras: Record<string, unknown> = {}, pending = operation) => {
    const descriptor = packageCapability(item.capabilities, operation);
    if (!descriptor) return false;
    const defaults = descriptor.body && typeof descriptor.body === "object" && !Array.isArray(descriptor.body) ? descriptor.body : {};
    const input: PackageReviewInput = { ...defaults, operation, item_id: item.id, revision: item.revision, source_refs: item.source_refs, reason: note, ...(fields ? { fields } : {}), ...extras };
    const signature = JSON.stringify(input);
    let key = keys.current.get(signature);
    if (!key) { key = crypto.randomUUID(); keys.current.set(signature, key); }
    setError(null); setPendingField(pending);
    try {
      const result = await onSave(input, key);
      setReceipt(result); setPatch(""); setReason(""); setTarget(""); onDirtyChange(false);
      return true;
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : String(failure));
      return false;
    } finally { setPendingField(null); }
  };

  /** Translate a donor metadata correction into a typed package field patch.
   * Input: field correction with observed value. Output: durable-save result.
   * Effects: clicked write. Pick for scalar/native JSON fields; invalid JSON remains unsaved.
   */
  const correct = async (input: CorrectionSubmit) => {
    if (!edit) return false;
    try {
      const value = input.observed !== null && typeof input.observed === "object" && typeof input.correctedValue === "string" ? JSON.parse(input.correctedValue) as unknown : input.correctedValue;
      return await submit(edit.operation, { [input.fieldKey]: value }, input.reason, {}, input.fieldKey);
    } catch (failure) { setError(failure instanceof Error ? failure.message : String(failure)); return false; }
  };

  /** Validate a complete supplemental JSON object before appending it through the advertised writer.
   * Inputs: current patch/reason drafts. Output: saved result in state. Effects: explicit mutation.
   * Pick for additional fields, nested metadata, provenance basis and reviewed edge corrections.
   */
  const savePatch = async () => {
    if (!supplement) return;
    try {
      const value: unknown = JSON.parse(patch);
      if (!value || Array.isArray(value) || typeof value !== "object") throw new Error("Enter a JSON object of field names and values.");
      await submit(supplement.operation, value as Record<string, unknown>, reason.trim());
    } catch (failure) { setError(failure instanceof Error ? failure.message : String(failure)); }
  };

  /** Search the existing committed entity registry for an explicit reassociation choice.
   * Input: human query. Output: actual registry entities. Effects: read-only HTTP lookup.
   * Pick for reassociation; account/device identities are displayed separately from people.
   */
  const findTargets = async () => {
    setSearching(true); setError(null);
    try { setTargets((await searchCommittedEntities(targetQuery.trim())).entities); }
    catch (failure) { setError(failure instanceof Error ? failure.message : String(failure)); }
    finally { setSearching(false); }
  };

  return <aside className="space-y-5 p-4" aria-label="Selected item editor" data-testid="package-item-editor">
    <PackageItemContent item={item} />
    <section aria-label="All item fields"><h3 className="platform-kicker mb-2">All fields · {Object.keys(item.fields).length}</h3>
      <MetadataFieldTable rows={Object.entries(item.fields).map(([fieldKey, value]) => ({ fieldKey, label: fieldKey, value }))} corrections={new Map()} correctable={Boolean(edit) && !busy && !reason && !patch && !target && !fieldDirty}
        pendingField={pendingField} onSubmit={correct} onDirtyChange={(value) => { setFieldDirty(value); onDirtyChange(value || Boolean(reason || patch || target)); }} />
    </section>
    {item.placement && <section><h3 className="platform-kicker mb-2">Recorded placement</h3><pre className="whitespace-pre-wrap break-all text-xs">{JSON.stringify(item.placement, null, 2)}</pre></section>}
    {supplement && <details className="rounded border p-3"><summary className="cursor-pointer text-sm font-medium">Supplement or edit fields</summary>
      <p className="my-2 text-xs text-muted-foreground">Append field corrections as a JSON object. Recorded source values remain preserved.</p>
      <Textarea aria-label="Supplemental fields JSON" className="min-h-28 font-mono text-xs" value={patch} disabled={busy || fieldDirty} onChange={(event) => { setPatch(event.target.value); onDirtyChange(Boolean(event.target.value || reason || target)); }} />
    </details>}
    {reassociate && <section className="space-y-2 rounded border p-3" aria-label="Reassociate item"><h3 className="text-sm font-medium">Reassociate</h3>
      <div className="flex gap-2"><Input aria-label="Find existing person, account or device" value={targetQuery} onChange={(event) => setTargetQuery(event.target.value)} />
        <Button variant="outline" size="sm" disabled={searching || !targetQuery.trim() || busy} onClick={() => void findTargets()}>Find existing</Button></div>
      {targets.length > 0 && <select className="w-full rounded border bg-background p-2 text-xs" aria-label="Association target" value={target} disabled={busy || fieldDirty} onChange={(event) => { setTarget(event.target.value); onDirtyChange(Boolean(event.target.value || reason || patch)); }}>
        <option value="">Choose a recorded identity</option>{targets.map((entry) => <option key={entry.id} value={entry.id}>{entry.display_name} · {entry.registry_type} · {entry.id}</option>)}</select>}
      {!searching && targetQuery && !targets.length && <p className="text-xs text-muted-foreground">No identities loaded. Use Find existing to query the registry.</p>}
    </section>}
    {(actions.length > 0 || supplement || reassociate) && <section className="sticky bottom-0 space-y-2 border-t bg-card py-3" aria-label="Human review actions">
      <label className="block text-xs font-medium">Reason for this change or decision<Input className="mt-1" aria-label="Review reason" value={reason} maxLength={4000} disabled={busy || fieldDirty} onChange={(event) => { setReason(event.target.value); onDirtyChange(Boolean(event.target.value || patch || target)); }} /></label>
      <div className="flex flex-wrap gap-2">
        {supplement && patch.trim() && <Button size="sm" disabled={busy || !reason.trim()} onClick={() => void savePatch()}>Save supplemental fields</Button>}
        {reassociate && target && <Button size="sm" disabled={busy || !reason.trim() || Boolean(patch.trim())} onClick={() => void submit("reassociate", undefined, reason.trim(), { target_ref: target })}>Save association</Button>}
        {actions.map((operation) => <Button key={operation} size="sm" variant={operation === "reject" ? "outline" : "default"} disabled={busy || !reason.trim() || Boolean(patch.trim()) || Boolean(target)} onClick={() => void submit(operation, undefined, reason.trim())}>{operation === "accept" ? "Accept" : operation === "reject" ? "Reject" : "Commit reviewed finding"}</Button>)}
        {(reason || patch || target) && <Button size="sm" variant="ghost" disabled={busy} onClick={() => { setReason(""); setPatch(""); setTarget(""); onDirtyChange(false); }}>Discard action draft</Button>}
      </div>
      {busy && <p role="status" className="text-xs">Saving {pendingField}…</p>}
      {error && <p role="alert" className="whitespace-pre-wrap text-sm text-destructive">{error}</p>}
      {receipt && <details className="text-xs"><summary className="cursor-pointer">Saved review receipt</summary><pre className="mt-1 whitespace-pre-wrap break-all">{JSON.stringify(receipt, null, 2)}</pre></details>}
    </section>}
    {!actions.length && !edit && !reassociate && <p className="text-xs text-muted-foreground">The backend advertises no review writes for this item.</p>}
    {error && !actions.length && !supplement && !reassociate && <p role="alert" className="text-destructive">{error}</p>}
    <PackageRelations items={[item]} selected={item} onSelect={onSelect} />
    <details className="rounded border p-3 text-xs"><summary className="cursor-pointer">Complete item record and review history</summary><pre className="mt-2 whitespace-pre-wrap break-all">{JSON.stringify(item, null, 2)}</pre></details>
  </aside>;
}
