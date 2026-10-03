// Byline: Claude Code · Sonnet · 2026-10-02
// "Who is this?" for a number nobody has confirmed, on the Workbench's own shadcn Sheet, Button, Input,
// Textarea, Label and Badge. Used by the mobile /m views and the desktop thread and calls views.
// Owner 2026-10-02 14:32/15:34: every number is a person in the registry (a name from a contact export, or a
// placeholder). Owner 20:41: the owner must be able to SEE THE SOURCE before deciding, say "this is my
// number" in one tap, pick an existing person from a searchable list, type a name, or confirm the mapping
// that is already there. Every save goes through the governed case-identity API (edit person / merge) and
// lands in registry.identity_change; this component never writes a registry table itself.
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FileSearch, UserRoundCheck, UserRoundSearch } from "lucide-react";
import { useMemo, useState } from "react";

import { NumberSourceSheet } from "@/components/identity/number-source-sheet";
import { prettyNumber } from "@/components/mobile/mobile-format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { addPlaceholders, editPerson, mergePerson, newIdempotencyKey } from "@/lib/case-identity-client";
import { importedApi } from "@/lib/imported-client";
import { cn } from "@/lib/utils";

export { prettyNumber };

const ROLES: [string, string][] = [
  ["third_party", "Someone else"],
  ["witness", "Witness"],
  ["attorney", "Attorney"],
  ["partner", "Partner"],
  ["neutral", "Neutral"],
  ["child", "Child"],
];

interface WhoIsThisProps {
  /** 10-digit number; null for a person known only by an email. */
  number: string | null;
  /** What to show in the header when there is no number. */
  label?: string;
  /** The name a contact export gave, when the person is already named but not confirmed. */
  currentName?: string | null;
  /** The person already carrying the number (a placeholder or contact-named, unconfirmed), when there is one. */
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
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={(event) => { event.preventDefault(); event.stopPropagation(); setOpen(true); }}
        className={cn("h-9 gap-1.5 rounded-full border-amber-500/60 bg-amber-100 text-xs font-semibold text-amber-950 hover:bg-amber-200 dark:bg-amber-950 dark:text-amber-100", className)}
      >
        <UserRoundSearch className="size-4" aria-hidden="true" /> {currentName ? "Confirm" : "Who is this?"}
      </Button>
      <WhoIsThisSheet open={open} onOpenChange={setOpen} number={number} label={label} currentName={currentName ?? null} entityId={entityId ?? null} candidates={candidates ?? []} context={context} />
    </>
  );
}

export function WhoIsThisSheet({ open, onOpenChange, number, label, currentName, entityId, candidates, context }: {
  open: boolean; onOpenChange: (open: boolean) => void; number: string | null; label?: string; currentName: string | null;
  entityId: string | null; candidates: string[]; context: string;
}) {
  const client = useQueryClient();
  const [mode, setMode] = useState<"name" | "same">("name");
  const [name, setName] = useState(currentName ?? "");
  const [role, setRole] = useState("third_party");
  const [note, setNote] = useState("");
  const [target, setTarget] = useState("");
  const [find, setFind] = useState("");
  const [pending, setPending] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const [sourceOpen, setSourceOpen] = useState(false);
  const people = useQuery({ queryKey: ["identity-people"], queryFn: ({ signal }) => importedApi.identity(signal), staleTime: 30_000, enabled: open });
  const shown = number ? prettyNumber(number) : (label ?? "this person");
  const ready = mode === "name" ? name.trim().length > 0 : target.length > 0;
  const me = people.data?.people.find((person) => person.role === "user");
  const matches = useMemo(() => {
    const needle = find.trim().toLowerCase();
    const all = people.data?.people ?? [];
    return needle ? all.filter((person) => `${person.name} ${person.short}`.toLowerCase().includes(needle)) : all;
  }, [find, people.data]);

  /** The person carrying this number: the one already there, or a new placeholder made through the governed API. */
  async function ensurePerson(): Promise<string> {
    if (entityId) return entityId;
    if (!number) throw new Error("This person has no number to start from.");
    const made = await addPlaceholders({ numbers: [number], change_reason: `number seen in ${context}, identified from the Workbench` }, newIdempotencyKey("placeholder"));
    const created = made.detail?.entity_ids?.[number] ?? Object.values((await importedApi.numberStatus([number])).items)[0]?.entity_id ?? null;
    if (!created) throw new Error("The person for this number could not be found. Try again.");
    return created;
  }

  async function run(action: () => Promise<string>) {
    if (pending) return;
    setPending(true);
    setFailure(null);
    try {
      setDone(await action());
      await client.invalidateQueries();
    } catch (error) {
      setFailure(error instanceof Error ? error.message : "Could not save");
    } finally {
      setPending(false);
    }
  }

  const mergeInto = (targetId: string, targetName: string, reason: string) => run(async () => {
    const personId = await ensurePerson();
    await mergePerson(personId, { into_id: targetId, change_reason: `owner: ${shown} ${reason} (seen in ${context})` }, newIdempotencyKey("merge"));
    return `Saved. ${shown} is now ${targetName}.`;
  });

  const saveName = () => run(async () => {
    const personId = await ensurePerson();
    const fields: Record<string, string | null> = {
      display_name: name.trim(), role_in_case: role, connection_to: "third_party",
      verification_state: "confirmed", requires_human_review: "false", review_status: "approved",
    };
    if (note.trim()) fields.relationship_type = note.trim().slice(0, 200);
    await editPerson(personId, { fields, change_reason: `named by the owner (seen in ${context})` }, newIdempotencyKey("name"));
    return `Saved. ${shown} is ${name.trim()}.`;
  });

  // The mapping already there (a name from a contact export) is right: confirm it without retyping.
  const confirmCurrent = () => run(async () => {
    const personId = await ensurePerson();
    await editPerson(personId, {
      fields: { verification_state: "confirmed", requires_human_review: "false", review_status: "approved" },
      change_reason: `owner confirmed the contact-export mapping (seen in ${context})`,
    }, newIdempotencyKey("confirm"));
    return `Confirmed. ${shown} is ${currentName}.`;
  });

  return (
    <>
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="mx-auto max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-2xl pb-[env(safe-area-inset-bottom)]">
        <SheetHeader>
          <SheetTitle className="text-lg">Who is {shown}?</SheetTitle>
          <SheetDescription>
            {currentName ? `A contact export says: ${currentName}. Check the source, then confirm it or change it.` : "Check where it appears, then name this person or say they are someone you already know."}
          </SheetDescription>
        </SheetHeader>

        {done ? (
          <div className="space-y-3 px-4 pb-6">
            <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">{done}</p>
            <p className="text-xs text-muted-foreground">Recorded in the case identity log. Every call and message for this number now shows the name.</p>
            <Button type="button" onClick={() => onOpenChange(false)} className="h-12 w-full">Done</Button>
          </div>
        ) : (
          <div className="space-y-4 px-4 pb-6">
            {number ? (
              <Button type="button" variant="outline" onClick={() => setSourceOpen(true)} className="h-12 w-full justify-start gap-2">
                <FileSearch className="size-5" aria-hidden="true" /> See where this number appears
              </Button>
            ) : null}

            <div className="grid gap-2">
              {currentName ? (
                <Button type="button" onClick={() => void confirmCurrent()} disabled={pending} className="h-12 justify-start gap-2">
                  <UserRoundCheck className="size-5" aria-hidden="true" /> Yes, this is {currentName}
                </Button>
              ) : null}
              {me ? (
                <Button type="button" variant="secondary" onClick={() => void mergeInto(me.entity_id, me.name, "is the owner's own number")} disabled={pending} className="h-12 justify-start gap-2">
                  <UserRoundCheck className="size-5" aria-hidden="true" /> This is my number
                </Button>
              ) : null}
            </div>

            <div className="grid grid-cols-2 gap-2" role="tablist">
              {([["name", "Name them"], ["same", "Someone I know"]] as const).map(([value, text]) => (
                <Button key={value} type="button" role="tab" aria-selected={mode === value} variant={mode === value ? "default" : "outline"} onClick={() => setMode(value)} className="h-12 whitespace-normal text-sm">{text}</Button>
              ))}
            </div>

            {mode === "name" ? (
              <div className="space-y-3">
                {candidates.length > 0 ? (
                  <div>
                    <p className="text-sm font-semibold">Your contacts say</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {candidates.map((candidate) => (
                        <Button key={candidate} type="button" variant="outline" onClick={() => setName(candidate)} className="h-11 rounded-full">{candidate}</Button>
                      ))}
                    </div>
                  </div>
                ) : null}
                <div className="space-y-1.5">
                  <Label htmlFor="who-name">Name</Label>
                  <Input id="who-name" value={name} onChange={(event) => setName(event.target.value)} autoComplete="off" placeholder="First and last name" className="h-12 text-base" />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="who-role">Who are they in the case?</Label>
                  <select id="who-role" value={role} onChange={(event) => setRole(event.target.value)} className="h-12 w-full rounded-md border border-input bg-background px-3 text-base">
                    {ROLES.map(([value, text]) => <option key={value} value={value}>{text}</option>)}
                  </select>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="who-find">Find a person</Label>
                <Input id="who-find" type="search" value={find} onChange={(event) => setFind(event.target.value)} autoComplete="off" placeholder="Type to filter" className="h-12 text-base" />
                {people.isPending ? <p className="text-sm text-muted-foreground">Loading people</p> : (
                  <ul className="max-h-64 space-y-2 overflow-y-auto" role="listbox" aria-label="People">
                    {matches.length === 0 ? <li className="py-3 text-sm text-muted-foreground">No one matches. Use "Name them" to add a new person.</li> : null}
                    {matches.map((person) => (
                      <li key={person.entity_id}>
                        <Button type="button" role="option" aria-selected={target === person.entity_id} variant={target === person.entity_id ? "default" : "outline"}
                          onClick={() => setTarget(person.entity_id)} className="h-12 w-full justify-between">
                          <span className="truncate">{person.name}</span>
                          {person.role === "user" ? <Badge variant="secondary">You</Badge> : null}
                        </Button>
                      </li>
                    ))}
                  </ul>
                )}
                <p className="text-xs text-muted-foreground">Their number joins that person, and every call and message for it moves to them.</p>
              </div>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="who-note">Relationship or note (optional)</Label>
              <Textarea id="who-note" value={note} onChange={(event) => setNote(event.target.value)} rows={2} placeholder="For example: Katrina's sister, babysitter, coworker" />
            </div>

            {failure ? <p className="text-sm font-semibold text-destructive" role="alert">{failure}</p> : null}
            <div className="grid grid-cols-2 gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)} className="h-12">Cancel</Button>
              <Button
                type="button"
                onClick={() => {
                  if (mode === "name") void saveName();
                  else void mergeInto(target, matches.find((person) => person.entity_id === target)?.name ?? people.data?.people.find((person) => person.entity_id === target)?.name ?? "that person",
                    `is this person${note.trim() ? ` (${note.trim()})` : ""}`);
                }}
                disabled={!ready || pending}
                className="h-12"
              >
                {pending ? "Saving" : "Save"}
              </Button>
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
    {number ? <NumberSourceSheet open={sourceOpen} onOpenChange={setSourceOpen} number={number} onNavigate={() => { setSourceOpen(false); onOpenChange(false); }} /> : null}
    </>
  );
}
