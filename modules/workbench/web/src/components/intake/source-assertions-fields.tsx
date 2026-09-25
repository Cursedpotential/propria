// Byline: Claude Code · Opus 5.5 · 2026-09-25
// The operator's source-context fields, extracted unchanged from unified-intake.tsx so
// Intake (first entry) and Review (correct an existing run) edit exactly the same record.
"use client";

import type { ProfferHumanSourceAssertions } from "@/lib/shared/types";
import { cn } from "@/lib/utils";

type Update = <K extends keyof ProfferHumanSourceAssertions>(key: K, value: ProfferHumanSourceAssertions[K]) => void;

export function SourceAssertionsFields({
  value,
  onChange,
  className,
}: {
  value: ProfferHumanSourceAssertions;
  onChange: Update;
  className?: string;
}) {
  return (
    <div className={cn("mt-5 grid gap-4 sm:grid-cols-2", className)}>
      <label className="grid gap-1.5 text-xs font-semibold">Source relationship
        <select className="h-10 border bg-background px-3 font-normal" value={value.source_class} onChange={(event) => onChange("source_class", event.target.value as ProfferHumanSourceAssertions["source_class"])}>
          <option value="unknown">Unknown / not sure</option><option value="first_party">First party / mine</option><option value="acquired_third_party">Acquired third party</option>
        </select>
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Other party
        <input className="h-10 border bg-background px-3 font-normal" value={value.other_party} onChange={(event) => onChange("other_party", event.target.value)} placeholder="Person, account, organization, or opposing party" />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Source principal
        <input className="h-10 border bg-background px-3 font-normal" value={value.source_principal} onChange={(event) => onChange("source_principal", event.target.value)} placeholder="Account, phone, device, or person this came from" />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">How acquired
        <select className="h-10 border bg-background px-3 font-normal" value={value.acquisition_method} onChange={(event) => onChange("acquisition_method", event.target.value as ProfferHumanSourceAssertions["acquisition_method"])}>
          <option value="">Not entered</option><option value="own_device">Own device</option><option value="household_device">Household device</option><option value="voluntary_third_party">Provided voluntarily</option><option value="legal_process">Legal process</option><option value="public_source">Public source</option><option value="unknown">Unknown</option>
        </select>
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">When acquired
        <input type="datetime-local" className="h-10 border bg-background px-3 font-normal" value={value.acquired_at ?? ""} onChange={(event) => onChange("acquired_at", event.target.value || null)} />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Acquisition authority
        <select className="h-10 border bg-background px-3 font-normal" value={value.acquisition_authority} onChange={(event) => onChange("acquisition_authority", event.target.value as ProfferHumanSourceAssertions["acquisition_authority"])}>
          <option value="">Not entered</option><option value="device_owner">Device owner</option><option value="parent_guardian">Parent / guardian</option><option value="account_holder">Account holder</option><option value="consent_given">Consent given</option><option value="court_order">Court order</option><option value="unclear">Unclear</option>
        </select>
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Known date — start
        <input type="date" className="h-10 border bg-background px-3 font-normal" value={value.occurred_start} onChange={(event) => onChange("occurred_start", event.target.value)} />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Known date — end
        <input type="date" className="h-10 border bg-background px-3 font-normal" value={value.occurred_end} onChange={(event) => onChange("occurred_end", event.target.value)} />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Date certainty
        <select className="h-10 border bg-background px-3 font-normal" value={value.date_certainty} onChange={(event) => onChange("date_certainty", event.target.value as ProfferHumanSourceAssertions["date_certainty"])}>
          <option value="">Not entered</option><option value="exact">Exact</option><option value="approximate">Approximate</option><option value="range">Date range</option><option value="unknown">Unknown</option>
        </select>
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Source device
        <input className="h-10 border bg-background px-3 font-normal" value={value.source_device} onChange={(event) => onChange("source_device", event.target.value)} placeholder="Device or storage source" />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold">Device custodian
        <input className="h-10 border bg-background px-3 font-normal" value={value.device_custodian} onChange={(event) => onChange("device_custodian", event.target.value)} placeholder="Who controlled the device" />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold sm:col-span-2">Context
        <textarea className="min-h-24 border bg-background p-3 font-normal" value={value.context} onChange={(event) => onChange("context", event.target.value)} placeholder="What this source is, why it matters, and anything the parser cannot know" />
      </label>
      <label className="grid gap-1.5 text-xs font-semibold sm:col-span-2">Notes
        <textarea className="min-h-20 border bg-background p-3 font-normal" value={value.notes} onChange={(event) => onChange("notes", event.target.value)} placeholder="Collection notes, limitations, or follow-up needed" />
      </label>
    </div>
  );
}
