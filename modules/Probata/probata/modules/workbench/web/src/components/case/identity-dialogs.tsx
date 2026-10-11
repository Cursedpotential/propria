// Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02
// Edit dialogs for the Case page. An identifier is added, fixed in place or
// deleted; a header or person save updates its registry row. Every save writes
// one registry.identity_change row with before, after, who and why. One
// Idempotency-Key per opened dialog, so a retried click never writes twice.
"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  addIdentifier,
  addPerson,
  deleteIdentifier,
  editCaseHeader,
  editIdentifier,
  editPerson,
  newIdempotencyKey,
  triageIdentifier,
  type CaseCourtCase,
  type CaseIdentifier,
  type CaseMatter,
  type CasePerson,
  type CaseReceipt,
} from "@/lib/case-identity-client";
import type { MatterMode } from "@/lib/shared/types";
import { useFixedCase } from "@/lib/fixed-case-context";

const IDENTIFIER_KINDS = ["phone", "name", "email", "account", "handle", "nickname", "legal", "maiden", "misspelling", "other"];
const ROLE_OPTIONS = ["user", "co_parent", "partner", "child", "witness", "evaluator", "attorney", "third_party", "neutral", "unknown"];
const CONNECTION_OPTIONS = ["plaintiff", "defendant", "petitioner", "respondent", "child", "mutual", "third_party", "unknown"];

const selectClass =
  "h-9 w-full rounded-md border border-input bg-background px-2 text-sm shadow-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

function Field({ label, children, hint }: { label: string; children: ReactNode; hint?: string }) {
  return (
    <div className="grid gap-1.5">
      <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</Label>
      {children}
      {hint && <p className="text-[11px] text-muted-foreground">{hint}</p>}
    </div>
  );
}

function Select({ value, onChange, options, allowEmpty, name }: { value: string; onChange: (v: string) => void; options: string[]; allowEmpty?: boolean; name?: string }) {
  const all = options.includes(value) || value === "" ? options : [value, ...options];
  return (
    <select className={selectClass} name={name} value={value} onChange={(event) => onChange(event.target.value)}>
      {allowEmpty && <option value="">—</option>}
      {all.map((option) => (
        <option key={option} value={option}>
          {option.replaceAll("_", " ")}
        </option>
      ))}
    </select>
  );
}

function useCaseSave<T>(save: (body: T, key: string, mode: MatterMode) => Promise<CaseReceipt>, onDone: () => void) {
  const { mode } = useFixedCase();
  const queryClient = useQueryClient();
  const key = useMemo(() => newIdempotencyKey("case"), []);
  return useMutation({
    mutationFn: (body: T) => save(body, key, mode),
    onSuccess: async (receipt) => {
      toast.success(receipt.replayed ? "Already saved (same click)" : "Saved", { description: `logged in registry.identity_change · ${receipt.kind} ${receipt.ref}` });
      await queryClient.invalidateQueries({ queryKey: ["case-identity"] });
      onDone();
    },
    onError: (error: Error) => toast.error("Not saved", { description: error.message }),
  });
}

function Footer({ pending, disabled, onCancel, label }: { pending: boolean; disabled: boolean; onCancel: () => void; label: string }) {
  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel} disabled={pending}>
        Cancel
      </Button>
      <Button type="submit" disabled={pending || disabled}>
        {pending && <Loader2 className="h-4 w-4 animate-spin" />} {label}
      </Button>
    </DialogFooter>
  );
}

// ---- identifier -------------------------------------------------------------

export type IdentifierDialogTarget =
  | { mode: "edit"; person: CasePerson; people: CasePerson[]; identifier: CaseIdentifier; status?: "confirmed" | "candidate" | "retired" }
  | { mode: "add"; person: CasePerson | null; people: CasePerson[]; raw?: string; kind?: string; basis?: string };

export function IdentifierDialog({ target, onClose }: { target: IdentifierDialogTarget; onClose: () => void }) {
  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="sm:max-w-lg">
        <IdentifierForm target={target} onClose={onClose} />
      </DialogContent>
    </Dialog>
  );
}

function IdentifierForm({ target, onClose }: { target: IdentifierDialogTarget; onClose: () => void }) {
  const existing = target.mode === "edit" ? target.identifier : null;
  const [personId, setPersonId] = useState(target.mode === "edit" ? target.person.id : (target.person?.id ?? ""));
  const [raw, setRaw] = useState(existing?.raw_value ?? (target.mode === "add" ? (target.raw ?? "") : ""));
  const [kind, setKind] = useState(existing?.kind ?? (target.mode === "add" ? (target.kind ?? "phone") : "phone"));
  const [status, setStatus] = useState<string>(target.mode === "edit" ? (target.status ?? existing!.status) : "candidate");
  const [period, setPeriod] = useState(existing?.period ?? "");
  const [basis, setBasis] = useState(existing?.basis ?? (target.mode === "add" ? (target.basis ?? "") : ""));
  const [reason, setReason] = useState("");
  const add = useCaseSave(addIdentifier, onClose);
  const edit = useCaseSave(
    (fields: Record<string, string | null>, key: string, mode: MatterMode) => editIdentifier(existing!.id, { fields, change_reason: reason }, key, mode),
    onClose,
  );
  const editing = target.mode === "edit";
  const changed = useMemo(() => {
    if (!existing) return {};
    const next: Record<string, string | null> = {};
    if (personId !== existing.entity_id) next.entity_id = personId;
    if (raw !== existing.raw_value) next.raw_value = raw;
    if (kind !== existing.kind) next.kind = kind;
    if (status !== existing.status) next.status = status;
    if ((period.trim() === "" ? null : period) !== existing.period) next.period = period.trim() === "" ? null : period;
    if (basis !== (existing.basis ?? "")) next.basis = basis;
    return next;
  }, [existing, personId, raw, kind, status, period, basis]);
  const pending = add.isPending || edit.isPending;
  const ready =
    personId !== "" && raw !== "" && raw.trim() === raw && basis.trim() !== "" &&
    (!editing || (reason.trim() !== "" && Object.keys(changed).length > 0));

  return (
    <form
      className="grid gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        if (editing) {
          edit.mutate(changed);
        } else {
          add.mutate({ entity_id: personId, raw_value: raw, kind, status, period: period.trim() === "" ? null : period, basis, change_reason: reason });
        }
      }}
    >
      <DialogHeader>
        <DialogTitle>{editing ? `Edit ${existing!.raw_value}` : "Add an identifier"}</DialogTitle>
        <DialogDescription>
          {editing
            ? "The row is fixed in place. Its earlier values, you and the time go into the change log."
            : "The spelling is kept exactly as typed; every reader matches on its normalized form."}
        </DialogDescription>
      </DialogHeader>
      <Field label="Person">
        <select className={selectClass} name="entity_id" value={personId} onChange={(event) => setPersonId(event.target.value)}>
          <option value="">Choose a person</option>
          {target.people.map((person) => (
            <option key={person.id} value={person.id}>
              {person.display_name}
            </option>
          ))}
        </select>
      </Field>
      <div className="grid grid-cols-[1fr_9rem] gap-3">
        <Field label="As seen">
          <Input name="raw_value" value={raw} onChange={(event) => setRaw(event.target.value)} />
        </Field>
        <Field label="Kind">
          <Select name="kind" value={kind} onChange={setKind} options={IDENTIFIER_KINDS} />
        </Field>
      </div>
      <div className="grid grid-cols-[9rem_1fr] gap-3">
        <Field label="Status">
          <Select name="status" value={status} onChange={setStatus} options={["confirmed", "candidate", "retired"]} />
        </Field>
        <Field label="In use (period)">
          <Input name="period" value={period} onChange={(event) => setPeriod(event.target.value)} placeholder="e.g. 2021-2024" />
        </Field>
      </div>
      <Field label="Basis — why we believe it">
        <Textarea name="basis" value={basis} onChange={(event) => setBasis(event.target.value)} rows={3} />
      </Field>
      <Field label={editing ? "Why this change" : "Note (optional)"}>
        <Input name="change_reason" value={reason} onChange={(event) => setReason(event.target.value)} />
      </Field>
      <Footer pending={pending} disabled={!ready} onCancel={onClose} label={editing ? "Save" : "Add"} />
    </form>
  );
}

export function DeleteIdentifierDialog({ identifier, onClose }: { identifier: CaseIdentifier; onClose: () => void }) {
  const [reason, setReason] = useState("");
  const save = useCaseSave((body: { change_reason: string }, key: string, mode: MatterMode) => deleteIdentifier(identifier.id, body, key, mode), onClose);
  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="sm:max-w-md">
        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            save.mutate({ change_reason: reason });
          }}
        >
          <DialogHeader>
            <DialogTitle>Delete {identifier.raw_value}</DialogTitle>
            <DialogDescription>
              The row leaves the registry. The change log keeps what it was, with you, the time and the reason. To keep it but mark it
              no longer believed, use Retire instead.
            </DialogDescription>
          </DialogHeader>
          <Field label="Why">
            <Input name="change_reason" value={reason} onChange={(event) => setReason(event.target.value)} placeholder="e.g. typed into the wrong person" />
          </Field>
          <Footer pending={save.isPending} disabled={reason.trim() === ""} onCancel={onClose} label="Delete" />
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---- person -------------------------------------------------------------------

export function PersonDialog({ person, onClose }: { person: CasePerson | null; onClose: () => void }) {
  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="sm:max-w-lg">
        <PersonForm person={person} onClose={onClose} />
      </DialogContent>
    </Dialog>
  );
}

function PersonForm({ person, onClose }: { person: CasePerson | null; onClose: () => void }) {
  const [displayName, setDisplayName] = useState(person?.display_name ?? "");
  const [shortName, setShortName] = useState(person?.short_name ?? "");
  const [role, setRole] = useState(person?.role_in_case ?? "third_party");
  const [connection, setConnection] = useState(person?.connection_to ?? "third_party");
  const [relationship, setRelationship] = useState(person?.relationship_type ?? "");
  const [notes, setNotes] = useState(person?.notes ?? "");
  const [isMinor, setIsMinor] = useState(person?.is_minor ?? false);
  const [reason, setReason] = useState("");
  const edit = useCaseSave(
    (fields: Record<string, string | null>, key: string, mode: MatterMode) => editPerson(person!.id, { fields, change_reason: reason }, key, mode),
    onClose,
  );
  const add = useCaseSave(
    (_: null, key: string, mode: MatterMode) =>
      addPerson(
        {
          display_name: displayName,
          short_name: shortName.trim() ? shortName : null,
          role_in_case: role,
          connection_to: connection,
          is_minor: isMinor,
          notes: notes.trim() ? notes : null,
          change_reason: reason,
        },
        key,
        mode,
      ),
    onClose,
  );
  const changed = useMemo(() => {
    if (!person) return {};
    const next: Record<string, string | null> = {};
    const blank = (value: string) => (value.trim() === "" ? null : value);
    if (displayName !== person.display_name) next.display_name = displayName;
    if (blank(shortName) !== person.short_name) next.short_name = blank(shortName);
    if (role !== (person.role_in_case ?? "")) next.role_in_case = role;
    if (connection !== (person.connection_to ?? "")) next.connection_to = connection;
    if (blank(relationship) !== person.relationship_type) next.relationship_type = blank(relationship);
    if (blank(notes) !== person.notes) next.notes = blank(notes);
    if (isMinor !== person.is_minor) next.is_minor = String(isMinor);
    return next;
  }, [person, displayName, shortName, role, connection, relationship, notes, isMinor]);
  const pending = edit.isPending || add.isPending;
  const ready = displayName.trim() !== "" && reason.trim() !== "" && (!person || Object.keys(changed).length > 0);

  return (
    <form
      className="grid gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        if (person) edit.mutate(changed);
        else add.mutate(null);
      }}
    >
      <DialogHeader>
        <DialogTitle>{person ? `Edit ${person.display_name}` : "Add a person"}</DialogTitle>
        <DialogDescription>The previous profile is kept in the change log with you and the time.</DialogDescription>
      </DialogHeader>
      <div className="grid grid-cols-[1fr_8rem] gap-3">
        <Field label="Name">
          <Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
        </Field>
        <Field label="Short name" hint="Label the catalog tools use">
          <Input value={shortName} onChange={(event) => setShortName(event.target.value)} />
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Role in the case">
          <Select value={role} onChange={setRole} options={ROLE_OPTIONS} />
        </Field>
        <Field label="Side">
          <Select value={connection} onChange={setConnection} options={CONNECTION_OPTIONS} />
        </Field>
      </div>
      <div className="grid grid-cols-[1fr_auto] items-end gap-3">
        <Field label="Relationship">
          <Input value={relationship} onChange={(event) => setRelationship(event.target.value)} placeholder="e.g. mother of the child" />
        </Field>
        <label className="flex h-9 items-center gap-2 text-sm">
          <input type="checkbox" checked={isMinor} onChange={(event) => setIsMinor(event.target.checked)} /> Minor
        </label>
      </div>
      <Field label="Notes">
        <Textarea value={notes} onChange={(event) => setNotes(event.target.value)} rows={2} />
      </Field>
      <Field label="Why this change">
        <Input value={reason} onChange={(event) => setReason(event.target.value)} />
      </Field>
      <Footer pending={pending} disabled={!ready} onCancel={onClose} label={person ? "Save" : "Add person"} />
    </form>
  );
}

// ---- case header ----------------------------------------------------------------

const COURT_FIELDS: { key: keyof CaseCourtCase; label: string; date?: boolean }[] = [
  { key: "caption", label: "Caption" },
  { key: "docket_number", label: "Docket number" },
  { key: "court_name", label: "Court" },
  { key: "presiding_judge", label: "Judge" },
  { key: "jurisdiction", label: "Jurisdiction" },
  { key: "case_type", label: "Case type" },
  { key: "status", label: "Status" },
  { key: "filed_on", label: "Filed on", date: true },
];

export function HeaderDialog({ mode, matter, courtCase, onClose }: { mode: MatterMode; matter: CaseMatter; courtCase: CaseCourtCase; onClose: () => void }) {
  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="sm:max-w-xl">
        <HeaderForm mode={mode} matter={matter} courtCase={courtCase} onClose={onClose} />
      </DialogContent>
    </Dialog>
  );
}

function HeaderForm({ mode, matter, courtCase, onClose }: { mode: MatterMode; matter: CaseMatter; courtCase: CaseCourtCase; onClose: () => void }) {
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(COURT_FIELDS.map((field) => [field.key, String(courtCase[field.key] ?? "")])),
  );
  const [title, setTitle] = useState(matter.title);
  const [reason, setReason] = useState("");
  const queryClient = useQueryClient();
  const courtKey = useMemo(() => newIdempotencyKey("court-case"), []);
  const matterKey = useMemo(() => newIdempotencyKey("matter"), []);
  const changed = useMemo(() => {
    const next: Record<string, string | null> = {};
    for (const field of COURT_FIELDS) {
      const before = String(courtCase[field.key] ?? "");
      if (values[field.key] !== before) next[field.key] = values[field.key].trim() === "" ? null : values[field.key];
    }
    return next;
  }, [values, courtCase]);
  const save = useMutation({
    mutationFn: async () => {
      const receipts: CaseReceipt[] = [];
      if (Object.keys(changed).length) {
        receipts.push(await editCaseHeader(mode, { target: "court_case", id: courtCase.id, fields: changed, change_reason: reason, expected_updated_at: courtCase.updated_at }, courtKey));
      }
      if (title !== matter.title) {
        receipts.push(await editCaseHeader(mode, { target: "matter", id: matter.id, fields: { title }, change_reason: reason, expected_updated_at: matter.updated_at }, matterKey));
      }
      return receipts;
    },
    onSuccess: async (receipts) => {
      toast.success(`Saved ${receipts.length} change${receipts.length === 1 ? "" : "s"}; the earlier values are in the change log`);
      await queryClient.invalidateQueries({ queryKey: ["case-identity"] });
      onClose();
    },
    onError: (error: Error) => toast.error("Not saved", { description: error.message }),
  });
  const ready = reason.trim() !== "" && (Object.keys(changed).length > 0 || title !== matter.title);

  return (
    <form
      className="grid gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate();
      }}
    >
      <DialogHeader>
        <DialogTitle>Edit the case header</DialogTitle>
        <DialogDescription>The registry row is updated; its previous values are logged with you and the time.</DialogDescription>
      </DialogHeader>
      <Field label="Matter">
        <Input value={title} onChange={(event) => setTitle(event.target.value)} />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        {COURT_FIELDS.map((field) => (
          <Field key={field.key} label={field.label}>
            {field.key === "status" ? (
              <Select value={values.status} onChange={(v) => setValues((prev) => ({ ...prev, status: v }))} options={["pre_filing", "active", "stayed", "closed", "appealed", "archived"]} />
            ) : (
              <Input
                type={field.date ? "date" : "text"}
                value={values[field.key]}
                onChange={(event) => setValues((prev) => ({ ...prev, [field.key]: event.target.value }))}
              />
            )}
          </Field>
        ))}
      </div>
      <Field label="Why this change">
        <Input value={reason} onChange={(event) => setReason(event.target.value)} />
      </Field>
      <Footer pending={save.isPending} disabled={!ready} onCancel={onClose} label="Save" />
    </form>
  );
}

// ---- unknowns: dismiss --------------------------------------------------------------

export function DismissDialog({ raw, onClose }: { raw: string; onClose: () => void }) {
  const [basis, setBasis] = useState("");
  const save = useCaseSave((body: { raw_value: string; decision: "dismissed"; basis: string }, key: string, mode: MatterMode) => triageIdentifier(body, key, mode), onClose);
  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="sm:max-w-md">
        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            save.mutate({ raw_value: raw, decision: "dismissed", basis });
          }}
        >
          <DialogHeader>
            <DialogTitle>Set aside {raw}</DialogTitle>
            <DialogDescription>It leaves the unknowns queue. The decision is a row with you and the time; reopening is another row.</DialogDescription>
          </DialogHeader>
          <Field label="Why">
            <Input value={basis} onChange={(event) => setBasis(event.target.value)} placeholder="e.g. carrier short code, not a person" />
          </Field>
          <Footer pending={save.isPending} disabled={basis.trim() === ""} onCancel={onClose} label="Set aside" />
        </form>
      </DialogContent>
    </Dialog>
  );
}
