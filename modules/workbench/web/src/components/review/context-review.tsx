// Byline: Claude Code · Opus 5.5 · 2026-09-26
// Context review for one Review message (owner 2026-09-25 19:13): who it is TO
// and ABOUT, whether it is about the child, whether it is relevant — saved as an
// attributed, append-only revision — and the separate FORESHADOWING flag.
//
// Foreshadowing is an internal, hindsight-only system flag ("significant in
// hindsight"). It is written to its own overlay, never to the message, and the
// as-lived read path never returns it. This owner surface reads the hindsight
// horizon explicitly; nothing here defaults to it.
"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plus, X } from "lucide-react";
import { useState, type KeyboardEvent } from "react";

import { SmallFlag } from "@/components/metadata/small-flag";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  getContextReview,
  postContextReview,
  postForeshadowing,
  type AboutChild,
  type ReviewAssertions,
  type ReviewParty,
} from "@/lib/review-overlays-client";
import type { MatterMode } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

const EMPTY: ReviewAssertions = { addressed_to: [], about: [], about_child: null, relevant: null };
const CHILD_CHOICES: Array<{ value: AboutChild; label: string }> = [
  { value: "yes", label: "Yes" },
  { value: "no", label: "No" },
  { value: "unsure", label: "Unsure" },
];

function sameAssertions(left: ReviewAssertions, right: ReviewAssertions) {
  const labels = (parties: ReviewParty[]) => parties.map((party) => party.label).join("\u0000");
  return labels(left.addressed_to) === labels(right.addressed_to) && labels(left.about) === labels(right.about)
    && left.about_child === right.about_child && left.relevant === right.relevant;
}

function PartyEditor({ label, parties, disabled, onChange }: {
  label: string;
  parties: ReviewParty[];
  disabled: boolean;
  onChange: (parties: ReviewParty[]) => void;
}) {
  const [text, setText] = useState("");
  const add = () => {
    const value = text.trim();
    if (!value || parties.some((party) => party.label.toLowerCase() === value.toLowerCase()) || parties.length >= 32) return;
    onChange([...parties, { label: value.slice(0, 200), entity_id: null }]);
    setText("");
  };
  const onKey = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      add();
    }
  };
  return (
    <div className="space-y-1">
      <Label className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</Label>
      <div className="flex flex-wrap gap-1">
        {parties.map((party) => (
          <span key={party.label} className="inline-flex items-center gap-1 border bg-accent/40 px-1.5 py-0.5 text-[11px]">
            {party.label}
            <button type="button" disabled={disabled} aria-label={`Remove ${party.label}`} onClick={() => onChange(parties.filter((item) => item.label !== party.label))}>
              <X className="size-3" />
            </button>
          </span>
        ))}
      </div>
      <div className="flex gap-1">
        <Input className="h-7 text-xs" value={text} disabled={disabled} placeholder="Add a name, then Enter" onChange={(event) => setText(event.target.value)} onKeyDown={onKey} aria-label={`Add to ${label}`} />
        <Button type="button" size="icon" variant="outline" className="size-7" disabled={disabled || !text.trim()} onClick={add} aria-label={`Add to ${label}`}>
          <Plus className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

export function ContextReviewPanel({ previewHandle, mode, messageId }: { previewHandle: string; mode: MatterMode; messageId: string }) {
  const queryClient = useQueryClient();
  const queryKey = ["context-review", mode, previewHandle, messageId] as const;
  // The owner's Review is a hindsight surface: it asks for the hindsight horizon explicitly.
  const query = useQuery({ queryKey, queryFn: () => getContextReview(previewHandle, mode, messageId, "hindsight"), retry: false });
  const current = query.data?.reviews[0];
  const flag = query.data?.foreshadowing?.[0];
  const [draft, setDraft] = useState<ReviewAssertions | null>(null);
  const [reason, setReason] = useState("");
  const [flagNote, setFlagNote] = useState("");
  const [pending, setPending] = useState<"review" | "flag" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const saved = current?.assertions ?? EMPTY;
  const value = draft ?? saved;
  const changed = draft !== null && !sameAssertions(draft, saved);
  const disabled = query.isPending || query.isError || pending !== null;

  const edit = (patch: Partial<ReviewAssertions>) => setDraft({ ...value, ...patch });

  const run = async (kind: "review" | "flag", write: () => Promise<unknown>) => {
    setPending(kind);
    setError(null);
    try {
      await write();
      await queryClient.invalidateQueries({ queryKey });
      return true;
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : "Not saved");
      return false;
    } finally {
      setPending(null);
    }
  };

  const saveReview = async () => {
    const ok = await run("review", () => postContextReview(previewHandle, mode, messageId, {
      ...value,
      supersedes_ref: current?.review_ref ?? "",
      change_reason: reason.trim() || "Context review in Review",
    }));
    if (ok) {
      setDraft(null);
      setReason("");
    }
  };

  const setForeshadowing = (next: boolean) => void run("flag", () => postForeshadowing(previewHandle, mode, messageId, {
    supersedes_ref: flag?.flag_ref ?? "",
    foreshadowing: next,
    note: flagNote.trim(),
    change_reason: next ? "Marked as foreshadowing in hindsight" : "Foreshadowing flag cleared",
  }));

  return (
    <section className="space-y-2 border-t px-4 py-3" aria-label="Context review" data-testid="context-review-panel">
      <header className="flex flex-wrap items-center gap-1.5">
        <h4 className="platform-rule-title text-xs">Context review</h4>
        {current && <SmallFlag tone="info" title={`${current.change_reason} · ${current.actor_username}`}>revision {current.revision}</SmallFlag>}
        {query.isError && <SmallFlag title={query.error instanceof Error ? query.error.message : undefined}>review unavailable</SmallFlag>}
        {error && <SmallFlag title={error}>not saved</SmallFlag>}
        {query.isPending && <Loader2 className="size-3.5 animate-spin text-muted-foreground" aria-label="Loading review" />}
      </header>

      <PartyEditor label="To" parties={value.addressed_to} disabled={disabled} onChange={(parties) => edit({ addressed_to: parties })} />
      <PartyEditor label="About" parties={value.about} disabled={disabled} onChange={(parties) => edit({ about: parties })} />

      <div className="space-y-1">
        <Label className="text-[10px] uppercase tracking-wide text-muted-foreground">About the child</Label>
        <div className="inline-flex rounded-md border p-0.5" role="group" aria-label="About the child">
          {CHILD_CHOICES.map((choice) => (
            <Button
              key={choice.value}
              type="button"
              size="sm"
              variant="ghost"
              disabled={disabled}
              aria-pressed={value.about_child === choice.value}
              className={cn("h-6 px-2 text-xs", value.about_child === choice.value && "bg-accent text-accent-foreground")}
              onClick={() => edit({ about_child: value.about_child === choice.value ? null : choice.value })}
            >
              {choice.label}
            </Button>
          ))}
        </div>
      </div>

      <label className="flex items-center gap-2 text-xs">
        <Checkbox checked={value.relevant === true} disabled={disabled} onCheckedChange={(checked) => edit({ relevant: checked === true })} aria-label="Relevant" />
        Relevant
      </label>

      <div className="flex gap-1">
        <Input className="h-7 text-xs" value={reason} disabled={disabled} maxLength={4000} placeholder="Note (optional)" onChange={(event) => setReason(event.target.value)} aria-label="Why this review" />
        <Button type="button" size="sm" className="h-7" disabled={disabled || !changed} onClick={() => void saveReview()} data-testid="save-context-review">
          {pending === "review" ? <Loader2 className="size-3.5 animate-spin" /> : null} Save review
        </Button>
      </div>

      <div className="space-y-1 border-t pt-2" data-testid="foreshadowing-control">
        <label className="flex flex-wrap items-center gap-2 text-xs">
          <Checkbox checked={flag?.foreshadowing === true} disabled={disabled} onCheckedChange={(checked) => setForeshadowing(checked === true)} aria-label="Foreshadowing" />
          Foreshadowing
          <SmallFlag tone="hindsight" title="Internal flag: significant in hindsight. Kept apart from the message and never shown to the as-lived view.">hindsight only</SmallFlag>
          {pending === "flag" && <Loader2 className="size-3.5 animate-spin" />}
        </label>
        <Input className="h-7 text-xs" value={flagNote} disabled={disabled} maxLength={4000} placeholder="What you know now (optional, saved with the next flag change)" onChange={(event) => setFlagNote(event.target.value)} aria-label="Foreshadowing note" />
        {flag?.note && <p className="text-[10px] text-muted-foreground">Saved note: {flag.note}</p>}
      </div>

      {query.data && query.data.reviews.length > 1 && (
        <details className="text-[10px] text-muted-foreground">
          <summary className="cursor-pointer">History ({query.data.reviews.length} revisions)</summary>
          <ol className="mt-1 space-y-0.5">
            {query.data.reviews.map((item) => (
              <li key={item.review_ref}>r{item.revision} · {item.actor_username} · {new Date(item.recorded_at).toLocaleString()} · {item.change_reason}</li>
            ))}
          </ol>
        </details>
      )}
    </section>
  );
}
