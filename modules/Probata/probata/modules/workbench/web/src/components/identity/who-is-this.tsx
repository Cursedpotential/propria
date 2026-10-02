// Byline: Claude Code · Sonnet · 2026-10-02
// "Who is this?" for a number nobody has named. Used by the mobile /m views and the desktop thread and
// calls views. Owner 2026-10-02 14:32: every unknown number is a placeholder person in the registry;
// naming it renames the placeholder (everything already linked follows), and "same as someone I know"
// merges it into an existing person (its identifiers and rows move; nothing is deleted).
// Every save goes through the governed case-identity API and lands in registry.identity_change; this
// component never writes a registry table itself.
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { UserRoundSearch, X } from "lucide-react";
import { useState } from "react";

import { addPlaceholders, editPerson, mergePerson, newIdempotencyKey } from "@/lib/case-identity-client";
import { importedApi } from "@/lib/imported-client";
import { cn } from "@/lib/utils";

const ROLES: [string, string][] = [
  ["third_party", "Someone else"],
  ["witness", "Witness"],
  ["attorney", "Attorney"],
  ["partner", "Partner"],
  ["neutral", "Neutral"],
  ["child", "Child"],
];

export function prettyNumber(number: string) {
  return number.length === 10 ? `(${number.slice(0, 3)}) ${number.slice(3, 6)}-${number.slice(6)}` : number;
}

interface WhoIsThisProps {
  /** 10-digit number; null for a person known only by an email. */
  number: string | null;
  /** What to show in the header when there is no number. */
  label?: string;
  /** The name a contact export gave, when the person is already named but not confirmed. */
  currentName?: string | null;
  /** The placeholder person already carrying it, when there is one. */
  entityId?: string | null;
  /** Names a contact export gave this number; one tap puts it in the name field. */
  candidates?: string[];
  /** Where the owner saw it, for the change log ("a call", "a text from ..."). */
  context?: string;
  className?: string;
}

export function WhoIsThis({ number, label, currentName, entityId, candidates, context = "the imported records", className }: WhoIsThisProps) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        onClick={(event) => { event.preventDefault(); event.stopPropagation(); setOpen(true); }}
        className={cn(
          "inline-flex min-h-9 items-center gap-1.5 rounded-full border border-amber-500/60 bg-amber-100 px-3 text-xs font-semibold text-amber-950 active:opacity-80 dark:bg-amber-950 dark:text-amber-100",
          className,
        )}
      >
        <UserRoundSearch className="size-4" aria-hidden="true" /> {currentName ? "Confirm" : "Who is this?"}
      </button>
      {open ? <WhoIsThisSheet number={number} label={label} currentName={currentName ?? null} entityId={entityId ?? null} candidates={candidates ?? []} context={context} onClose={() => setOpen(false)} /> : null}
    </>
  );
}

function WhoIsThisSheet({ number, label, currentName, entityId, candidates, context, onClose }: { number: string | null; label?: string; currentName: string | null; entityId: string | null; candidates: string[]; context: string; onClose: () => void }) {
  const client = useQueryClient();
  const [mode, setMode] = useState<"name" | "same">("name");
  const [name, setName] = useState(currentName ?? "");
  const [role, setRole] = useState("third_party");
  const [note, setNote] = useState("");
  const [target, setTarget] = useState("");
  const [pending, setPending] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const people = useQuery({ queryKey: ["identity-people"], queryFn: ({ signal }) => importedApi.identity(signal), staleTime: 30_000 });

  const ready = mode === "name" ? name.trim().length > 0 : target.length > 0;

  async function save() {
    if (!ready || pending) return;
    setPending(true);
    setFailure(null);
    try {
      let personId = entityId;
      if (!personId) {
        if (!number) throw new Error("This person has no number to start from.");
        // Nobody carries the number yet: it becomes a placeholder first, then is named or merged.
        const made = await addPlaceholders(
          { numbers: [number as string], change_reason: `number seen in ${context}, identified from the Workbench` },
          newIdempotencyKey("placeholder"),
        );
        personId = made.detail?.entity_ids?.[number as string] ?? null;
        if (!personId) {
          const status = await importedApi.numberStatus([number as string]);
          personId = Object.values(status.items)[0]?.entity_id ?? null;
        }
        if (!personId) throw new Error("The placeholder for this number could not be found. Try again.");
      }
      if (mode === "name") {
        const fields: Record<string, string | null> = {
          display_name: name.trim(),
          role_in_case: role,
          connection_to: "third_party",
          verification_state: "confirmed",
          requires_human_review: "false",
          review_status: "approved",
        };
        if (note.trim()) fields.relationship_type = note.trim().slice(0, 200);
        await editPerson(personId, { fields, change_reason: `named by the owner (seen in ${context})` }, newIdempotencyKey("name"));
        setDone(`Saved. ${number ? prettyNumber(number) : (label ?? "This person")} is ${name.trim()}.`);
      } else {
        await mergePerson(
          personId,
          { into_id: target, change_reason: `owner: ${number ? prettyNumber(number) : (label ?? "this person")} is this person${note.trim() ? ` (${note.trim()})` : ""} (seen in ${context})` },
          newIdempotencyKey("merge"),
        );
        const chosen = people.data?.people.find((person) => person.entity_id === target);
        setDone(`Saved. ${number ? prettyNumber(number) : (label ?? "This person")} is now ${chosen?.name ?? "that person"}.`);
      }
      await client.invalidateQueries();
    } catch (error) {
      setFailure(error instanceof Error ? error.message : "Could not save");
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center" role="dialog" aria-modal="true" aria-label={`Who is ${number ? prettyNumber(number) : (label ?? "this")}?`}>
      <div className="max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-2xl bg-card p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] text-card-foreground shadow-xl sm:rounded-2xl">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">Who is this?</h2>
            <p className="text-2xl font-bold tabular-nums">{number ? prettyNumber(number) : (label ?? "Unknown")}</p>
            {currentName ? <p className="text-sm text-muted-foreground">A contact export says: {currentName}. Confirm it, or change it.</p> : null}
          </div>
          <button type="button" onClick={onClose} aria-label="Close" className="flex size-12 items-center justify-center rounded-full active:bg-muted">
            <X className="size-6" />
          </button>
        </div>

        {done ? (
          <div className="mt-4 space-y-3">
            <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">{done}</p>
            <p className="text-xs text-muted-foreground">Recorded in the case identity log. Every call and message for this number now shows the name.</p>
            <button type="button" onClick={onClose} className="h-12 w-full rounded-lg bg-primary font-semibold text-primary-foreground active:opacity-80">Done</button>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid grid-cols-2 gap-2" role="tablist">
              {([["name", "Name them"], ["same", "Same as someone I know"]] as const).map(([value, label]) => (
                <button
                  key={value}
                  type="button"
                  role="tab"
                  aria-selected={mode === value}
                  onClick={() => setMode(value)}
                  className={cn("min-h-12 rounded-lg border px-2 text-sm font-semibold", mode === value ? "border-primary bg-primary text-primary-foreground" : "border-border bg-background")}
                >
                  {label}
                </button>
              ))}
            </div>

            {mode === "name" ? (
              <div className="space-y-3">
                {candidates.length > 0 ? (
                  <div>
                    <p className="text-sm font-semibold">Your contacts say</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {candidates.map((candidate) => (
                        <button key={candidate} type="button" onClick={() => setName(candidate)}
                          className="min-h-11 rounded-full border border-border bg-background px-4 text-sm font-medium active:bg-muted">{candidate}</button>
                      ))}
                    </div>
                  </div>
                ) : null}
                <label className="block text-sm font-semibold" htmlFor="who-name">Name</label>
                <input id="who-name" value={name} onChange={(event) => setName(event.target.value)} autoComplete="off" placeholder="First and last name"
                  className="h-12 w-full rounded-lg border border-input bg-background px-3 text-base" />
                <label className="block text-sm font-semibold" htmlFor="who-role">Who are they in the case?</label>
                <select id="who-role" value={role} onChange={(event) => setRole(event.target.value)} className="h-12 w-full rounded-lg border border-input bg-background px-3 text-base">
                  {ROLES.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                </select>
              </div>
            ) : (
              <div className="space-y-2">
                <p className="text-sm font-semibold">Pick the person</p>
                {people.isPending ? <p className="text-sm text-muted-foreground">Loading people</p> : (
                  <ul className="space-y-2">
                    {(people.data?.people ?? []).map((person) => (
                      <li key={person.entity_id}>
                        <label className={cn("flex min-h-12 cursor-pointer items-center gap-3 rounded-lg border px-3", target === person.entity_id ? "border-primary bg-accent" : "border-border")}>
                          <input type="radio" name="who-target" value={person.entity_id} checked={target === person.entity_id} onChange={() => setTarget(person.entity_id)} className="size-5" />
                          <span className="text-[15px] font-medium">{person.name}</span>
                        </label>
                      </li>
                    ))}
                  </ul>
                )}
                <p className="text-xs text-muted-foreground">Their number joins that person, and every call and message for it moves to them.</p>
              </div>
            )}

            <div className="space-y-2">
              <label className="block text-sm font-semibold" htmlFor="who-note">Relationship or note (optional)</label>
              <textarea id="who-note" value={note} onChange={(event) => setNote(event.target.value)} rows={2} placeholder="For example: Katrina's sister, babysitter, coworker"
                className="w-full rounded-lg border border-input bg-background p-3 text-base" />
            </div>

            {failure ? <p className="text-sm font-semibold text-destructive" role="alert">{failure}</p> : null}
            <div className="grid grid-cols-2 gap-2">
              <button type="button" onClick={onClose} className="h-12 rounded-lg border border-border font-semibold active:bg-muted">Cancel</button>
              <button type="button" onClick={() => void save()} disabled={!ready || pending}
                className="h-12 rounded-lg bg-primary font-semibold text-primary-foreground active:opacity-80 disabled:opacity-50">
                {pending ? "Saving" : "Save"}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
