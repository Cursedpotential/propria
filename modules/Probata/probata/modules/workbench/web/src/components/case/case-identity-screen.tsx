// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Case — who the case is about and every way they appear in the data.
//
// Owner order 2026-10-01 07:56: "There needs to be a case identity page. Can
// update things. People, profiles, aliases, phone numbers, view what's been
// extracted as far as what's in there." Step 6 of the six (fill in gaps and
// missing context). One viewport, no tabs: the case header on top, the people
// with their identifiers and counts on the left, identifiers tied to nobody on
// the right. Ids, versions and the change log sit in the record drawer.
//
// Registry (Probata) is the one identity store; every count names its store.
"use client";

import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, BookOpen, Check, History, Loader2, Pencil, Plus, UserPlus, X } from "lucide-react";
import { useMemo, useState } from "react";

import { CatalogEventsSheet, type CatalogEventsTarget } from "@/components/case/catalog-events-sheet";
import {
  DismissDialog,
  HeaderDialog,
  IdentifierDialog,
  PersonDialog,
  type IdentifierDialogTarget,
} from "@/components/case/identity-dialogs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import {
  getCaseIdentity,
  type CaseIdentifier,
  type CaseIdentityView,
  type CasePerson,
  type CatalogCount,
  type CatalogUnknown,
} from "@/lib/case-identity-client";
import { useFixedCase } from "@/lib/fixed-case-context";

const ROLE_LABEL: Record<string, string> = {
  user: "Owner (you)",
  co_parent: "Co-parent",
  partner: "Partner",
  child: "Child",
  witness: "Witness",
  evaluator: "Evaluator",
  attorney: "Attorney",
  third_party: "Third party",
  neutral: "Neutral",
  unknown: "Unknown role",
};

function label(value: string | null | undefined, map: Record<string, string> = {}) {
  if (!value) return "—";
  return map[value] ?? value.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
}

function day(value: string | null | undefined) {
  return value ? value.slice(0, 10) : "";
}

function range(first: string | null | undefined, last: string | null | undefined) {
  if (!first && !last) return "";
  return first?.slice(0, 10) === last?.slice(0, 10) ? day(first) : `${day(first)} → ${day(last)}`;
}

const MATCH_LABEL: Record<string, string> = { counterparty_phone: "as the other party", sender: "as sender" };
const KIND_LABEL: Record<string, string> = { message: "messages", call: "calls", ai_chat_turn: "AI chat turns", document: "documents" };

function statusVariant(status: string) {
  return status === "confirmed" ? "default" : status === "candidate" ? "secondary" : "outline";
}

/** Catalog counts for one identifier, grouped by how the identifier matched. */
function CatalogChips({
  counts,
  identifier,
  onOpen,
}: {
  counts: CatalogCount[];
  identifier: CaseIdentifier;
  onOpen: (target: CatalogEventsTarget) => void;
}) {
  const mine = counts.filter((count) => count.identifier === identifier.normalized);
  if (!mine.length) return <span className="text-xs text-muted-foreground">Case Bible: nothing</span>;
  const byMatch = new Map<string, CatalogCount[]>();
  for (const count of mine) byMatch.set(count.match_on, [...(byMatch.get(count.match_on) ?? []), count]);
  return (
    <div className="flex flex-wrap gap-1.5">
      {[...byMatch.entries()].map(([matchOn, rows]) => {
        const first = rows.map((row) => row.first_at).filter(Boolean).sort()[0];
        const last = rows.map((row) => row.last_at).filter(Boolean).sort().at(-1);
        const text = rows.map((row) => `${row.events.toLocaleString()} ${KIND_LABEL[row.event_kind] ?? row.event_kind}`).join(" · ");
        return (
          <button
            key={matchOn}
            type="button"
            className="rounded-md border border-border bg-muted/40 px-2 py-0.5 text-left text-xs hover:bg-accent"
            title={`Case Bible catalog (raw_duck.comm_events_20260918), ${MATCH_LABEL[matchOn]}. Sources: ${[...new Set(rows.flatMap((row) => row.sources))].join(", ")}`}
            onClick={() =>
              onOpen({
                identifier: identifier.normalized,
                matchOn: matchOn as CatalogEventsTarget["matchOn"],
                label: `${identifier.raw_value} — ${text} ${MATCH_LABEL[matchOn]}`,
              })
            }
          >
            <span className="font-semibold">{text}</span> <span className="text-muted-foreground">{MATCH_LABEL[matchOn]} · {range(first, last)}</span>
          </button>
        );
      })}
    </div>
  );
}

function ProbataChips({ view, keyValue }: { view: CaseIdentityView; keyValue: string }) {
  const rows = view.probata_counts.filter((count) => count.key === keyValue);
  if (!rows.length) return <span className="text-xs text-muted-foreground">Probata: nothing ingested yet</span>;
  return (
    <span className="text-xs">
      Probata:{" "}
      {rows.map((row) => `${row.events.toLocaleString()} ${row.source.replaceAll("_", " ")} (${range(row.first_at, row.last_at)})`).join(" · ")}
    </span>
  );
}

function IdentifierRow({
  view,
  person,
  identifier,
  onEdit,
  onEvents,
  onHistory,
}: {
  view: CaseIdentityView;
  person: CasePerson;
  identifier: CaseIdentifier;
  onEdit: (target: IdentifierDialogTarget) => void;
  onEvents: (target: CatalogEventsTarget) => void;
  onHistory: (identifier: CaseIdentifier) => void;
}) {
  const retired = identifier.status === "retired";
  return (
    <li className={`grid grid-cols-[minmax(9rem,13rem)_1fr_auto] gap-3 border-t border-border/60 py-2 first:border-t-0 ${retired ? "opacity-60" : ""}`}>
      <div className="min-w-0">
        <div className={`truncate font-mono text-sm ${retired ? "line-through" : ""}`} title={identifier.raw_value}>
          {identifier.raw_value}
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-1">
          <Badge variant={statusVariant(identifier.status)}>{identifier.status}</Badge>
          <span className="text-[11px] text-muted-foreground">{identifier.kind}</span>
          {identifier.period && <span className="text-[11px] text-muted-foreground">· {identifier.period}</span>}
        </div>
      </div>
      <div className="min-w-0 space-y-1">
        <CatalogChips counts={view.catalog.counts} identifier={identifier} onOpen={onEvents} />
        <ProbataChips view={view} keyValue={identifier.normalized} />
        {identifier.basis && (
          <p className="line-clamp-2 text-[11px] text-muted-foreground" title={identifier.basis}>
            {identifier.basis}
          </p>
        )}
      </div>
      <div className="flex items-start gap-1">
        {identifier.status === "candidate" && (
          <Button size="xs" variant="secondary" onClick={() => onEdit({ mode: "version", person, identifier, status: "confirmed" })}>
            <Check /> Confirm
          </Button>
        )}
        {!retired && (
          <Button size="icon-xs" variant="ghost" title="Retire (kept, marked no longer believed)" onClick={() => onEdit({ mode: "version", person, identifier, status: "retired" })}>
            <X />
          </Button>
        )}
        <Button size="icon-xs" variant="ghost" title="Edit (new version)" onClick={() => onEdit({ mode: "version", person, identifier })}>
          <Pencil />
        </Button>
        <Button size="icon-xs" variant="ghost" title={`${identifier.history.length + 1} version(s)`} onClick={() => onHistory(identifier)}>
          <History />
        </Button>
      </div>
    </li>
  );
}

function PersonCard({
  view,
  person,
  onEditPerson,
  onEditIdentifier,
  onEvents,
  onHistory,
}: {
  view: CaseIdentityView;
  person: CasePerson;
  onEditPerson: (person: CasePerson) => void;
  onEditIdentifier: (target: IdentifierDialogTarget) => void;
  onEvents: (target: CatalogEventsTarget) => void;
  onHistory: (identifier: CaseIdentifier) => void;
}) {
  const phones = person.identifiers.filter((identifier) => identifier.kind === "phone").length;
  const names = person.identifiers.length - phones;
  return (
    <section className="rounded-lg border border-border bg-card p-4" data-testid="case-person" data-person-id={person.id}>
      <header className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">{person.display_name}</h2>
          <p className="text-xs text-muted-foreground">
            {label(person.role_in_case, ROLE_LABEL)} · {label(person.connection_to)}
            {person.short_name ? ` · catalog label "${person.short_name}"` : ""}
            {person.relationship_type ? ` · ${person.relationship_type}` : ""}
            {person.is_minor ? " · minor" : ""}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {phones} phone{phones === 1 ? "" : "s"} · {names} name{names === 1 ? "" : "s"} and accounts · <ProbataChips view={view} keyValue={person.id} />
          </p>
        </div>
        <div className="flex gap-1">
          <Button size="sm" variant="outline" onClick={() => onEditIdentifier({ mode: "add", person, people: [person] })}>
            <Plus /> Identifier
          </Button>
          <Button size="sm" variant="ghost" onClick={() => onEditPerson(person)}>
            <Pencil /> Profile
          </Button>
        </div>
      </header>
      {person.identifiers.length ? (
        <ul className="mt-3">
          {person.identifiers.map((identifier) => (
            <IdentifierRow
              key={identifier.id}
              view={view}
              person={person}
              identifier={identifier}
              onEdit={onEditIdentifier}
              onEvents={onEvents}
              onHistory={onHistory}
            />
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-muted-foreground">No identifiers recorded yet.</p>
      )}
    </section>
  );
}

function UnknownList({
  title,
  items,
  people,
  onAssign,
  onDismiss,
  onEvents,
}: {
  title: string;
  items: CatalogUnknown[];
  people: CasePerson[];
  onAssign: (target: IdentifierDialogTarget) => void;
  onDismiss: (raw: string) => void;
  onEvents: (target: CatalogEventsTarget) => void;
}) {
  return (
    <div>
      <h3 className="platform-kicker mb-1">{title}</h3>
      {items.length === 0 ? (
        <p className="text-xs text-muted-foreground">None.</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {items.map((item) => (
            <li key={item.identifier} className="flex items-center gap-2 py-1.5">
              <button
                type="button"
                className="min-w-0 flex-1 text-left"
                title="Open the catalog events"
                onClick={() =>
                  onEvents({
                    identifier: item.identifier,
                    matchOn: item.match_on.includes("counterparty_phone") ? "counterparty_phone" : "sender",
                    label: `${item.identifier} — tied to nobody`,
                  })
                }
              >
                <div className="truncate font-mono text-xs">{item.identifier}</div>
                <div className="text-[11px] text-muted-foreground">
                  {item.events.toLocaleString()} events · {range(item.first_at, item.last_at)}
                </div>
              </button>
              <Button
                size="xs"
                variant="outline"
                onClick={() =>
                  onAssign({
                    mode: "add",
                    person: null,
                    people,
                    raw: item.identifier,
                    kind: item.kind,
                    basis: `seen ${item.events.toLocaleString()} times in the Case Bible catalog (${range(item.first_at, item.last_at)})`,
                  })
                }
              >
                Assign
              </Button>
              <Button size="icon-xs" variant="ghost" title="Set aside" onClick={() => onDismiss(item.identifier)}>
                <X />
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function RecordDrawer({ view, open, onClose, focus }: { view: CaseIdentityView; open: boolean; onClose: () => void; focus: CaseIdentifier | null }) {
  return (
    <Sheet open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <SheetContent side="right" className="w-[min(720px,96vw)] sm:max-w-none">
        <SheetHeader>
          <SheetTitle>{focus ? `Versions of ${focus.raw_value}` : "Record"}</SheetTitle>
          <SheetDescription>Registry rows behind this page. Nothing is overwritten: every edit is a row here.</SheetDescription>
        </SheetHeader>
        <div className="space-y-5 overflow-auto px-4 pb-6 text-xs">
          {focus && (
            <ol className="space-y-2">
              {[focus, ...focus.history].map((version, index) => (
                <li key={version.id} className="rounded border border-border p-2">
                  <div className="flex items-center gap-2">
                    <Badge variant={statusVariant(version.status)}>{version.status}</Badge>
                    <span>{index === 0 ? "current" : `superseded`}</span>
                    <span className="text-muted-foreground">{version.recorded_by} · {version.recorded_at.replace("T", " ").slice(0, 19)} UTC</span>
                  </div>
                  {version.period && <p>Period: {version.period}</p>}
                  {version.basis && <p>Basis: {version.basis}</p>}
                  {version.change_reason && <p>Why changed: {version.change_reason}</p>}
                  <p className="font-mono text-[10px] text-muted-foreground">registry.entity_alias {version.id}{version.supersedes_id ? ` supersedes ${version.supersedes_id}` : ""}</p>
                </li>
              ))}
            </ol>
          )}
          {!focus && (
            <>
              <section>
                <h3 className="platform-kicker mb-1">Identity</h3>
                <p className="font-mono">matter {view.matter?.id ?? "—"}</p>
                <p className="font-mono">court case {view.court_case?.id ?? "—"}</p>
                {view.people.map((person) => (
                  <p key={person.id} className="font-mono">
                    person {person.display_name}: {person.id}
                  </p>
                ))}
              </section>
              <section>
                <h3 className="platform-kicker mb-1">Stores</h3>
                <p>Identities: Probata registry (registry.entity / person / entity_alias). Counts labeled Probata: working.message_participant, third_party_message_participant, call_log, entity_mention.</p>
                <p>
                  Counts labeled Case Bible: {view.catalog.table} over {view.catalog.events} ({view.catalog.available ? "available" : `unavailable: ${view.catalog.error}`}).
                </p>
              </section>
              <section>
                <h3 className="platform-kicker mb-1">Change log (registry.identity_change)</h3>
                {view.history.length === 0 ? (
                  <p className="text-muted-foreground">No header or profile edits yet.</p>
                ) : (
                  <ol className="space-y-2">
                    {view.history.map((change) => (
                      <li key={change.id} className="rounded border border-border p-2">
                        <p>
                          <span className="font-semibold">{change.subject_table}</span> · {change.recorded_by} · {change.recorded_at.replace("T", " ").slice(0, 19)} UTC
                        </p>
                        <p>{change.change_reason}</p>
                        <pre className="mt-1 overflow-auto whitespace-pre-wrap text-[10px] text-muted-foreground">
                          {Object.keys(change.after)
                            .filter((key) => JSON.stringify(change.before[key]) !== JSON.stringify(change.after[key]))
                            .map((key) => `${key}: ${JSON.stringify(change.before[key] ?? null)} → ${JSON.stringify(change.after[key])}`)
                            .join("\n")}
                        </pre>
                      </li>
                    ))}
                  </ol>
                )}
              </section>
              {view.dismissed.length > 0 && (
                <section>
                  <h3 className="platform-kicker mb-1">Set aside</h3>
                  {view.dismissed.map((item) => (
                    <p key={item.normalized}>
                      <span className="font-mono">{item.raw_value}</span> — {item.basis} ({item.recorded_by})
                    </p>
                  ))}
                </section>
              )}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

export function CaseIdentityScreen() {
  const { mode } = useFixedCase();
  const query = useQuery({ queryKey: ["case-identity", mode], queryFn: () => getCaseIdentity(mode) });
  const [identifierTarget, setIdentifierTarget] = useState<IdentifierDialogTarget | null>(null);
  const [personTarget, setPersonTarget] = useState<CasePerson | null | undefined>(undefined);
  const [headerOpen, setHeaderOpen] = useState(false);
  const [dismissRaw, setDismissRaw] = useState<string | null>(null);
  const [eventsTarget, setEventsTarget] = useState<CatalogEventsTarget | null>(null);
  const [recordOpen, setRecordOpen] = useState(false);
  const [historyFocus, setHistoryFocus] = useState<CaseIdentifier | null>(null);
  const view = query.data;
  const caption = view?.court_case?.caption ?? view?.matter?.title;
  const people = useMemo(() => view?.people ?? [], [view]);

  if (query.isLoading) {
    return (
      <div className="grid h-full place-content-center text-sm text-muted-foreground">
        <Loader2 className="mx-auto h-5 w-5 animate-spin" /> Reading the registry…
      </div>
    );
  }
  if (query.isError || !view) {
    return (
      <div className="grid h-full place-content-center gap-2 text-center text-sm">
        <AlertTriangle className="mx-auto h-5 w-5 text-destructive" />
        <p>{(query.error as Error | null)?.message ?? "The case identity could not be read"}</p>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="case-identity-screen">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-6 py-4" data-testid="case-header">
        <div className="min-w-0">
          <p className="platform-kicker">Case · {view.mode}</p>
          <h1 className="truncate text-2xl font-semibold tracking-tight">{caption ?? "No case has been recorded for this mode"}</h1>
          {view.court_case && (
            <p className="mt-1 text-sm text-muted-foreground">
              {[
                view.court_case.docket_number && `No. ${view.court_case.docket_number}`,
                view.court_case.court_name,
                view.court_case.presiding_judge && `Judge ${view.court_case.presiding_judge}`,
                view.court_case.case_type,
                label(view.court_case.status),
              ]
                .filter(Boolean)
                .join(" · ")}
            </p>
          )}
          {!view.matter && <p className="mt-1 text-sm text-muted-foreground">The registry holds no {view.mode} matter yet.</p>}
        </div>
        <div className="flex gap-2">
          {view.matter && view.court_case && (
            <Button size="sm" variant="outline" onClick={() => setHeaderOpen(true)}>
              <Pencil /> Case header
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={() => setPersonTarget(null)}>
            <UserPlus /> Person
          </Button>
          <Button size="sm" variant="ghost" onClick={() => { setHistoryFocus(null); setRecordOpen(true); }}>
            <BookOpen /> Record
          </Button>
        </div>
      </header>

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden p-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="min-h-0 space-y-4 overflow-auto pr-1" data-testid="case-people">
          {people.length === 0 && <p className="text-sm text-muted-foreground">No people are recorded in the registry yet.</p>}
          {people.map((person) => (
            <PersonCard
              key={person.id}
              view={view}
              person={person}
              onEditPerson={(p) => setPersonTarget(p)}
              onEditIdentifier={setIdentifierTarget}
              onEvents={setEventsTarget}
              onHistory={(identifier) => { setHistoryFocus(identifier); setRecordOpen(true); }}
            />
          ))}
          {!view.catalog.available && (
            <p className="flex items-center gap-2 text-xs text-muted-foreground">
              <AlertTriangle className="h-3.5 w-3.5" /> Case Bible counts unavailable: {view.catalog.error}
            </p>
          )}
        </div>
        <aside className="min-h-0 space-y-4 overflow-auto rounded-lg border border-border bg-card p-4" data-testid="case-unknowns">
          <div>
            <h2 className="text-sm font-semibold">Tied to nobody</h2>
            <p className="text-[11px] text-muted-foreground">Most frequent identifiers in the Case Bible catalog that no person carries. Assign one to a person or set it aside.</p>
          </div>
          {view.probata_unknowns.length > 0 && (
            <div>
              <h3 className="platform-kicker mb-1">Probata participants</h3>
              <ul className="divide-y divide-border/60">
                {view.probata_unknowns.slice(0, 25).map((item) => (
                  <li key={item.normalized} className="flex items-center gap-2 py-1.5 text-xs">
                    <span className="min-w-0 flex-1 truncate font-mono">{item.raw_value}</span>
                    <span className="text-muted-foreground">{item.events}</span>
                    <Button size="xs" variant="outline" onClick={() => setIdentifierTarget({ mode: "add", person: null, people, raw: item.raw_value, kind: /^[0-9]+$/.test(item.normalized) ? "phone" : "name" })}>
                      Assign
                    </Button>
                  </li>
                ))}
              </ul>
            </div>
          )}
          <UnknownList title="Phone numbers" items={view.catalog.unknowns.phone.slice(0, 25)} people={people} onAssign={setIdentifierTarget} onDismiss={setDismissRaw} onEvents={setEventsTarget} />
          <UnknownList title="Names and labels" items={view.catalog.unknowns.name.slice(0, 25)} people={people} onAssign={setIdentifierTarget} onDismiss={setDismissRaw} onEvents={setEventsTarget} />
        </aside>
      </div>

      {identifierTarget && <IdentifierDialog target={identifierTarget} onClose={() => setIdentifierTarget(null)} />}
      {personTarget !== undefined && <PersonDialog person={personTarget} onClose={() => setPersonTarget(undefined)} />}
      {headerOpen && view.matter && view.court_case && (
        <HeaderDialog mode={view.mode} matter={view.matter} courtCase={view.court_case} onClose={() => setHeaderOpen(false)} />
      )}
      {dismissRaw && <DismissDialog raw={dismissRaw} onClose={() => setDismissRaw(null)} />}
      <CatalogEventsSheet target={eventsTarget} onClose={() => setEventsTarget(null)} />
      <RecordDrawer view={view} open={recordOpen} focus={historyFocus} onClose={() => setRecordOpen(false)} />
    </div>
  );
}
