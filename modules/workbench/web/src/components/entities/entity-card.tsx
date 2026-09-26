// Byline: Claude Code · Opus 5.5 · 2026-09-25 (one proposed entity: aliases, mentions, owner corrections)
"use client";

import { Flag, GitMerge, Pencil, Plus, RotateCcw, Scissors, Search, X } from "lucide-react";
import { useMemo, useState } from "react";

import { RecordPeek, formatWhen } from "@/components/entities/record-peek";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { useEntityActions } from "@/hooks/use-entity-extraction";
import { searchCommittedEntities, type EntityMention, type EntityProposal, type RegistryEntity } from "@/lib/entity-extraction-client";
import type { MatterMode } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

type Actions = ReturnType<typeof useEntityActions>;

const selectClass = "h-7 border bg-background px-1 text-xs";

export function EntityCard({
  proposal, previewHandle, mode, entityTypes, aliasKinds, selected, onSelect, actions,
}: {
  proposal: EntityProposal;
  previewHandle: string;
  mode: MatterMode;
  entityTypes: string[];
  aliasKinds: string[];
  selected: boolean;
  onSelect: (selected: boolean) => void;
  actions: Actions;
}) {
  const [renaming, setRenaming] = useState(false);
  const [name, setName] = useState(proposal.name);
  const [aliasText, setAliasText] = useState("");
  const [aliasKind, setAliasKind] = useState("nickname");
  const [splitting, setSplitting] = useState(false);
  const [splitAliases, setSplitAliases] = useState<string[]>([]);
  const [splitName, setSplitName] = useState("");
  const [matching, setMatching] = useState(false);
  const [matchQuery, setMatchQuery] = useState(proposal.name);
  const [matches, setMatches] = useState<RegistryEntity[] | null>(null);
  const [showMentions, setShowMentions] = useState(false);
  const [peek, setPeek] = useState<EntityMention | null>(null);
  const [error, setError] = useState<string | null>(null);
  const editable = proposal.review_state === "pending";
  const id = proposal.candidate_id;

  const mentions = useMemo(() => {
    const seen = new Set<string>();
    return [...(proposal.model_mentions ?? []), ...(proposal.mention_sample ?? [])]
      .filter((mention) => {
        const key = `${mention.record_id}|${mention.start ?? ""}|${mention.role}`;
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      })
      .sort((left, right) => left.ordinal - right.ordinal)
      .slice(0, 30);
  }, [proposal.model_mentions, proposal.mention_sample]);

  const run = async (action: () => Promise<unknown>) => {
    setError(null);
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    }
  };

  return (
    <li className={cn("border bg-card p-2 text-xs", proposal.review_state === "rejected" && "opacity-60")} data-testid="entity-card">
      <div className="flex flex-wrap items-center gap-2">
        {editable && <input type="checkbox" aria-label={`Select ${proposal.name} to merge`} checked={selected} onChange={(event) => onSelect(event.target.checked)} />}
        {renaming ? (
          <form className="flex items-center gap-1" onSubmit={(event) => { event.preventDefault(); void run(async () => { await actions.correctEntity({ op: "rename", candidate_ids: [id], name: name.trim() }); setRenaming(false); }); }}>
            <Input aria-label="Entity name" className="h-7 w-56 text-xs" value={name} maxLength={200} onChange={(event) => setName(event.target.value)} autoFocus />
            <Button type="submit" size="xs" disabled={!name.trim()}>Save</Button>
            <Button type="button" size="xs" variant="ghost" onClick={() => setRenaming(false)}>Cancel</Button>
          </form>
        ) : (
          <strong className="text-sm">{proposal.name}</strong>
        )}
        {editable && !renaming && <Button type="button" size="icon-xs" variant="ghost" aria-label="Rename" onClick={() => { setName(proposal.name); setRenaming(true); }}><Pencil /></Button>}
        {editable ? (
          <select aria-label="Entity type" className={selectClass} value={proposal.registry_type} onChange={(event) => void run(() => actions.correctEntity({ op: "retype", candidate_ids: [id], registry_type: event.target.value }))}>
            {entityTypes.map((type) => <option key={type} value={type}>{type.replaceAll("_", " ")}</option>)}
          </select>
        ) : <Badge variant="outline">{proposal.registry_type}</Badge>}
        {proposal.review_state === "approved" && <Badge>committed</Badge>}
        {proposal.review_state === "rejected" && <Badge variant="outline">rejected</Badge>}
        {proposal.source_owner && <Badge variant="outline" title="The owner of the device this source was saved from">device owner</Badge>}
        {proposal.match && (
          <Badge variant="secondary" title={`matched by ${proposal.match.via}`}>
            <GitMerge className="size-3" /> joins committed “{proposal.match.display_name || proposal.match.entity_id}”
          </Badge>
        )}
        {(proposal.flags ?? []).map((flag) => (
          <span key={flag.code} title={flag.detail} className="inline-flex items-center text-[#8a5a00] dark:text-[#ffd48a]"><Flag className="size-3.5" aria-label={flag.detail} /></span>
        ))}
        <span className="ml-auto text-muted-foreground" title={proposal.first_occurred_at ? `${formatWhen(proposal.first_occurred_at)} – ${formatWhen(proposal.last_occurred_at)}` : undefined}>
          {proposal.mention_count.toLocaleString()} mentions
        </span>
      </div>

      <ul className="mt-2 flex flex-wrap gap-1" aria-label="Aliases">
        {proposal.aliases.map((alias) => (
          <li key={`${alias.kind}:${alias.text}`} className="inline-flex items-center gap-1 border px-1.5 py-0.5" title={alias.scope ? "Only within this source; never a registry alias" : `${alias.kind} · from ${alias.source}`}>
            {splitting && <input type="checkbox" aria-label={`Move ${alias.text}`} checked={splitAliases.includes(alias.text)} onChange={(event) => setSplitAliases((current) => event.target.checked ? [...current, alias.text] : current.filter((value) => value !== alias.text))} />}
            <span className={cn(alias.address_kind && "font-mono")}>{alias.address_kind === "self" ? "self (this source)" : alias.text}</span>
            <span className="text-[10px] text-muted-foreground">{alias.kind}</span>
            {editable && !splitting && <button type="button" aria-label={`Remove alias ${alias.text}`} className="text-muted-foreground hover:text-destructive" onClick={() => void run(() => actions.correctEntity({ op: "remove_alias", candidate_ids: [id], alias_text: alias.text }))}><X className="size-3" /></button>}
          </li>
        ))}
        {editable && !splitting && (
          <li>
            <form className="inline-flex items-center gap-1" onSubmit={(event) => { event.preventDefault(); void run(async () => { await actions.correctEntity({ op: "add_alias", candidate_ids: [id], alias_text: aliasText.trim(), alias_kind: aliasKind }); setAliasText(""); }); }}>
              <Input aria-label="New alias" className="h-6 w-32 text-xs" placeholder="add alias" value={aliasText} maxLength={200} onChange={(event) => setAliasText(event.target.value)} />
              <select aria-label="Alias kind" className={selectClass} value={aliasKind} onChange={(event) => setAliasKind(event.target.value)}>
                {aliasKinds.map((kind) => <option key={kind} value={kind}>{kind}</option>)}
              </select>
              <Button type="submit" size="icon-xs" variant="ghost" aria-label="Add alias" disabled={!aliasText.trim()}><Plus /></Button>
            </form>
          </li>
        )}
      </ul>

      {splitting && (
        <form className="mt-2 flex flex-wrap items-center gap-1" onSubmit={(event) => { event.preventDefault(); void run(async () => { await actions.correctEntity({ op: "split", candidate_ids: [id], name: splitName.trim(), split_aliases: splitAliases }); setSplitting(false); }); }}>
          <span className="text-muted-foreground">Tick the aliases that belong to someone else, and name them:</span>
          <Input aria-label="Name of the new entity" className="h-7 w-48 text-xs" value={splitName} maxLength={200} onChange={(event) => setSplitName(event.target.value)} />
          <Button type="submit" size="xs" disabled={!splitName.trim() || splitAliases.length === 0}>Split</Button>
          <Button type="button" size="xs" variant="ghost" onClick={() => setSplitting(false)}>Cancel</Button>
        </form>
      )}

      {matching && (
        <div className="mt-2 space-y-1">
          <form className="flex items-center gap-1" onSubmit={(event) => { event.preventDefault(); void run(async () => setMatches((await searchCommittedEntities(matchQuery.trim())).entities)); }}>
            <Input aria-label="Search committed entities" className="h-7 w-56 text-xs" value={matchQuery} maxLength={200} onChange={(event) => setMatchQuery(event.target.value)} />
            <Button type="submit" size="xs" variant="outline"><Search /> Find committed</Button>
            <Button type="button" size="xs" variant="ghost" onClick={() => setMatching(false)}>Cancel</Button>
          </form>
          {matches && (matches.length === 0 ? <p className="text-muted-foreground">No committed entity matches.</p> : (
            <ul className="space-y-0.5">
              {matches.map((entity) => (
                <li key={entity.id}>
                  <button type="button" className="underline-offset-2 hover:underline" onClick={() => void run(async () => { await actions.correctEntity({ op: "set_match", candidate_ids: [id], match: { entity_id: entity.id, display_name: entity.display_name } }); setMatching(false); })}>
                    Join “{entity.display_name}” ({entity.registry_type})
                  </button>
                </li>
              ))}
            </ul>
          ))}
        </div>
      )}

      {editable && (proposal.suggestions ?? []).length > 0 && !proposal.match && (
        <p className="mt-1 text-muted-foreground">
          Possibly the same as{" "}
          {(proposal.suggestions ?? []).filter((suggestion) => suggestion.entity_id).map((suggestion) => (
            <button key={suggestion.entity_id} type="button" className="mr-2 text-foreground underline underline-offset-2" title={suggestion.reason} onClick={() => void run(() => actions.correctEntity({ op: "set_match", candidate_ids: [id], match: { entity_id: suggestion.entity_id as string } }))}>
              {suggestion.display_name}
            </button>
          ))}
        </p>
      )}

      <div className="mt-2 flex flex-wrap items-center gap-1">
        <Button type="button" size="xs" variant="ghost" onClick={() => setShowMentions((value) => !value)} aria-expanded={showMentions}>
          {showMentions ? "Hide mentions" : `Show mentions (${mentions.length}${proposal.mention_count > mentions.length ? ` of ${proposal.mention_count.toLocaleString()}` : ""})`}
        </Button>
        {editable && (
          <>
            <Button type="button" size="xs" variant="ghost" onClick={() => { setSplitAliases([]); setSplitName(""); setSplitting(true); }}><Scissors /> Split</Button>
            {proposal.match
              ? <Button type="button" size="xs" variant="ghost" onClick={() => void run(() => actions.correctEntity({ op: "clear_match", candidate_ids: [id] }))}>Keep separate</Button>
              : <Button type="button" size="xs" variant="ghost" onClick={() => setMatching(true)}><GitMerge /> Join a committed entity</Button>}
            <Button type="button" size="xs" variant="ghost" className="text-destructive" onClick={() => void run(() => actions.correctEntity({ op: "reject", candidate_ids: [id] }))}><X /> Reject</Button>
          </>
        )}
        {proposal.review_state === "rejected" && <Button type="button" size="xs" variant="ghost" onClick={() => void run(() => actions.correctEntity({ op: "restore", candidate_ids: [id] }))}><RotateCcw /> Restore</Button>}
        {proposal.correction && <span className="ml-auto text-[10px] text-muted-foreground">{proposal.correction.op.replaceAll("_", " ")} by {proposal.correction.actor.username} · {formatWhen(proposal.correction.at)}</span>}
      </div>
      {error && <p className="mt-1 text-destructive" role="alert">{error}</p>}

      {showMentions && (
        <ol className="mt-1 space-y-1" aria-label="Supporting mentions">
          {mentions.length === 0 && <li className="text-muted-foreground">Only participant headers; open the message list to read them.</li>}
          {mentions.map((mention) => (
            <li key={`${mention.record_id}|${mention.start ?? ""}|${mention.role}`}>
              <button type="button" className="w-full text-left hover:bg-accent/40" onClick={() => setPeek(peek?.record_id === mention.record_id && peek.start === mention.start ? null : mention)}>
                <span className="font-mono text-muted-foreground">#{mention.ordinal}</span> {formatWhen(mention.occurred_at)} · {mention.role === "body" ? `“${mention.surface}”` : `${mention.role} ${mention.surface}`}
                {mention.snippet && <span className="block text-muted-foreground">{mention.snippet}</span>}
              </button>
              {peek && peek.record_id === mention.record_id && peek.start === mention.start && (
                <RecordPeek previewHandle={previewHandle} mode={mode} recordId={mention.record_id} highlight={{ start: mention.start, end: mention.end }} onClose={() => setPeek(null)} />
              )}
            </li>
          ))}
        </ol>
      )}
    </li>
  );
}
