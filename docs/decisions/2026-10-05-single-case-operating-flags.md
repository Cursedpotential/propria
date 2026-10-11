# One case identity; independent operating and feature flags

Byline: Codex, orchestrator, 2026-10-05. Authority: the owner's current corrections
and explicit instruction to fix mode-dependent case identity.

## Owner contract

There is one actual case. Switching Dev / Live must not switch its matter,
court case, people, source identity or source versions. Live is the normal state;
Dev is enabled for a concrete development/test need. The product surface is the
Case page, not a second class of case. Labels alone do not repair this defect.

Operating mode and feature flags are separate controls:

| Control | Responsibility | Must not do |
| --- | --- | --- |
| Dev / Live operating policy | Identify the explicit operation policy; default Live | Select or mint another case identity |
| Authentication/ingress configuration | Preserve the existing trusted tailnet door and public Authentik proxy boundary | Change because a Case page mode changes |
| Evidence/immutability controls | Gate the particular implemented, tested evidence rules | Hide missing machinery behind a flag or change source identity |
| Data isolation/lifetime | Route Dev changes to a verified disposable workspace and end that workspace explicitly | Roll shared Live data back to an old snapshot |
| Individual feature enablement | Enable/disable the named feature with both states tested | Act as an undocumented all-features/all-security switch |

This is not a claim that a general feature-flag registry or disposable data
workspace is already implemented. Existing authentication controls, including
`PLATFORM_DEV_AUTH_BYPASS` and the Workbench tailnet configuration, are not being
broadened or repurposed by this identity repair.

## Supersession

Historical D-126 (2026-09-02) instructed pre-launch placeholder identity selection
behind a development flag. That is the provenance of the defect, not the current
contract. Its identity-selection/reset/cutover instructions are superseded here.
Current owner corrections also supersede inherited TEST/REAL selectors and their
tests. Preserve historical records; do not treat them as present instructions.

D-127/D-128's useful commitments remain: build the gated capability, prove the
strict path, test both flag states, and make the scoped exception observable.
The owner's 2026-10-02 history rejects a parallel deployment/case-table design.
The 2026-10-05 frozen/ephemeral-copy proposal is a data-lifetime requirement, not
authority to replace identities or to roll a shared database backward.

## Bounded repair contract

- One exact approved matter/court-case resolver in both operating modes. No
  sentinel defaults, arbitrary sole-matter fallback or identity-derived mode.
- Canonical wire values `DEV` / `LIVE`, default `LIVE`; unknown explicit values
  fail validation. User-facing labels are Dev / Live.
- Workbench neutral configuration: `PROFFER_MATTER_ID` and
  `PROFFER_COURT_CASE_ID`. Legacy authoritative environment names may be accepted
  at a documented rollout boundary only, never as another selector.
- Explicit `operating_mode` on start requests and durable operation receipts/
  workflow history. Recovery does not guess from the matter ID. Missing or
  unverifiable historical mode cannot authorize mutations.
- Every relevant Case page mutation carries the selected policy through client,
  BFF and engine. Dev canonical writes fail before persistence/dispatch while
  isolated workspace support is unavailable. Live gates remain unchanged.
- SQL CHECK constraints that still require old stored values may use a narrow
  internal storage mapping during this rollout. Stored values do not determine
  identity and are normalized before response/policy comparisons. No schema
  rebuild or numbered migration is part of this repair.

## Disposable Dev workspace: owed, not implemented by this repair

The owner's proposed behavior is a frozen/disposable view of the existing Live
data: Dev uses the same actual case identity, Dev edits remain outside Live,
and leaving Dev returns to Live without those Dev edits. Live writes by another
session must survive. Authentication exceptions, feature enablement, guard
exceptions and workspace lifetime require independent controls.

The lifecycle still needs implementation and proof: snapshot consistency,
workspace routing, concurrent Live writes, operation receipt binding, restart
recovery, exit/retention behavior, and original-source preservation. Until those
exist, a clear Dev write blocker is required. It is not a substitute for doing
the data-isolation work and does not claim automatic rollback works today.

## Verification and rollout boundary

Before repair, read-only probes of the supported Workbench Tailscale surface on
2026-10-05 confirmed that the current primary-mode request returned the approved
pair, the old test-mode request returned no case identity, DEV/LIVE were rejected,
and omitted mode was rejected. No live data was changed by these probes.

Acceptance requires identity equality for Dev, Live and omitted/default mode;
source/people IDs preserved; invalid/sentinel scope rejected; explicit durable
mode recovery; and zero persistence/dispatch for blocked Dev mutations. Manual
Coolify deployment, served-surface verification and a receipt are required before
calling the repair complete. Automatic deploy remains off. The in-progress code
is not yet claimed to be merged, deployed or complete in this decision record.
