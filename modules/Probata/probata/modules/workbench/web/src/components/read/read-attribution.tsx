import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { IdentifierDialog, type IdentifierDialogTarget } from "@/components/case/identity-dialogs";
import { Button } from "@/components/ui/button";
import { getCaseIdentity } from "@/lib/case-identity-client";
import type { ImportedMessage } from "@/lib/imported-client";
import { AppLink } from "@/lib/router-compat";

/** Inspect source-relative attribution and open the existing global identity editor.
 * Inputs: exact displayed message. Output: literal/resolved identities and scope.
 * Effects: registry GET only until the owner opens and saves the audited editor.
 * Pick at a questionable message; no one-message speaker override is supported.
 */
export function ReadAttribution({ message }: { message: ImportedMessage }) {
  const [open, setOpen] = useState(false);
  const [edit, setEdit] = useState<IdentifierDialogTarget | null>(null);
  const registry = useQuery({ queryKey: ["case-identity", "LIVE"], queryFn: () => getCaseIdentity("LIVE"), enabled: open, retry: false });
  const attribution = message.attribution;
  const raw = attribution?.raw_sender;
  const digits = raw?.replace(/\D/g, "").slice(-10);
  const matches = registry.data?.people.flatMap((person) => person.identifiers.filter((alias) => alias.status !== "retired" && raw && (alias.normalized === raw || (digits && alias.kind === "phone" && alias.normalized.replace(/\D/g, "").slice(-10) === digits))).map((identifier) => ({ person, identifier }))) ?? [];
  return <details className="mt-1 px-2 text-xs text-muted-foreground" onToggle={(event) => setOpen(event.currentTarget.open)}>
    <summary className="cursor-pointer">Inspect attribution</summary>
    <dl className="mt-2 space-y-1 break-words">
      <dt className="font-semibold">Raw normalized speaker</dt><dd>{raw ?? "Unknown: no explicit sender role"}</dd>
      <dt className="font-semibold">Resolved display identity</dt><dd>{message.sender.label} · {attribution?.status ?? "Unknown"}{message.sender.entity_id ? ` · ${message.sender.entity_id}` : ""}</dd>
      <dt className="font-semibold">Raw recipients</dt><dd>{attribution?.raw_recipients.join(", ") || "Not supplied"}</dd>
      <dt className="font-semibold">Source account/device perspective</dt><dd>{attribution?.source_owner ?? "Unknown"} · direction {attribution?.direction ?? "unknown"}</dd>
      <dt className="font-semibold">Source owner display basis</dt><dd>{attribution?.source_owner_basis ?? "Unknown: no ownership receipt supplied"}</dd>
      <dt className="font-semibold">Acquisition relationship</dt><dd>{message.party?.replaceAll("_", " ") ?? "Unclassified"}; independent of speaker and legal role</dd>
      <dt className="font-semibold">Acquisition basis</dt><dd>{attribution?.acquisition_basis ?? "Unknown"}</dd>
      <dt className="font-semibold">Human / AI authorship</dt><dd>{attribution?.authorship_kind ?? "unknown"}; this message projection does not supply a verified authorship kind</dd>
      <dt className="font-semibold">Attribution basis</dt><dd>{attribution?.basis ?? "Not supplied"}</dd>
      <dt className="font-semibold">Source version / normalized generation / ordinal</dt><dd>{attribution?.source_version_id ?? "Unknown"} / {attribution?.generation_id ?? "Unknown"} / {attribution?.record_ordinal ?? "Unknown"}</dd>
    </dl>
    <p className="mt-2">Identifier correction edits one global registry alias. Its period is descriptive; Imported does not enforce it as a date filter. A single-message speaker override is unavailable.</p>
    {registry.isError ? <p role="alert">Registry mapping unavailable.</p> : null}
    {matches.map(({ person, identifier }) => <div key={identifier.id} className="mt-2 rounded border p-2">
      <p>{person.display_name} · identifier {identifier.status} · basis: {identifier.basis || "Not supplied"} · period: {identifier.period || "Not supplied"} (descriptive; no Imported time filter)</p>
      <Button variant="outline" size="sm" className="mt-2" onClick={() => setEdit({ mode: "edit", person, people: registry.data!.people, identifier })}>Inspect scope and correct {identifier.raw_value}</Button>
    </div>)}
    {matches.length > 1 ? <p role="status">Multiple current aliases match this raw identifier. Inspect each mapping; this display does not establish a unique historical person.</p> : null}
    {raw === "self" ? <p className="mt-2">Self is the source perspective marker. Correct its source context rather than assigning it a global phone alias.</p> : null}
    {attribution?.preview_handle ? <AppLink className="mt-2 block underline underline-offset-4" href={`/review?resource=${encodeURIComponent(attribution.preview_handle)}&mode=LIVE`}>Open source operation for acquisition-context revision</AppLink> : <p className="mt-2">No source operation link is retained for context editing.</p>}
    <p className="mt-2">A source-context save appends a revision for that operation and observation. It does not rewrite historical normalized messages. Reprocessing is a separate new-version action.</p>
    {edit ? <IdentifierDialog target={edit} onClose={() => setEdit(null)} /> : null}
  </details>;
}
