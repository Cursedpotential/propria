---
title: Repair workflow builder — propose, compose, validate, run on Temporal (+ n8n)
date: 2026-09-25
status: PROPOSAL — owner has not picked an option; nothing built
domains: [probata, workbench, proffer, engine]
tags: [repair, workflow-builder, temporal, n8n, validation, tool-catalog, design, pending-review]
---

# Repair workflow builder

> _Byline: Claude Code · Opus 5.5 · 2026-09-25._

Owner 2026-09-25 18:25: "Apply proposed repair … should propose options based on file name and then allow me to build out the workflow; it should validate that the workflow will follow the designated rules of the space and of the data, that it's going to be properly extracted and make it to the right places, and there's no gaps; allow me to run that workflow, run against Temporal, and if need be run n8n nodes if that's the best tool."

Earlier requirements it satisfies: bulk-intake 2026-09-20 req 3 ("options when more than one way forward exists"), req 5 ("no parser → call the agent"), and 2026-09-20 "find another version based on filename, or wait for more detail after parsing, offer to look in other sources."

## What exists (verified in source 2026-09-25)
- Temporal proffer workflow (26 stages) with human gates; repair re-entry rule: an approved derived repair keeps the original, writes and hashes a separate artifact, then re-enters validation → routing → handler selection (proffer_operator.py reentry_rule).
- **n8n-as-Temporal-activity already built:** `engine/temporal/flowactivity.go`, `flowbinding.go`, registered in `profferworker` as `run_n8n_flow_activity`; worker logs `n8n_flow_binding_count=0` (plumbing present, no flows bound).
- Tool gateway: registry-keyed tools (`repair.detect`, `repair.preview`, `documents.extract-text`, `messages.sms-xml-sbv`, …). **No repair-apply tool exists.**
- **Tool catalog is empty in the live Workbench** (Docstore critical flag 2026-09-12 `note:probata_function_access_broken_20260912`): `GET /api/tools` = `[]` (MCP_SERVERS=[]), `GET /api/monitored-actions/capabilities` = 404, no monitored-actions backend. Same defect behind the owner's "Go tools menu lists nothing / locks up".

## The design (all options share this)
1. **Propose** — a signature table (file name/format + `repair.detect` report → candidate steps), per the signature-router rule; an agent proposer for signatures the table doesn't cover. Example, truncated SMS XML: find another copy by name in the catalog (same basename, other hash, larger size) · salvage to the last complete record · lenient SBV decode · wait for more detail after parse.
2. **Compose** — the owner orders/edits steps into a plan; each step is ONE registered Activity (atomicity rule).
3. **Validate** before run, fail-closed, each rule a named check with the reason shown:
   - every step is a registered Activity with known input/output types, and each step's output type is accepted by the next (type-checked chain);
   - the original is never written (repair outputs a separate, hashed derived artifact);
   - the plan ends by re-entering validation → routing → handler → extraction → storage → completeness — no path that skips a stage (no gaps);
   - every destination resolves (D-158 storage destination) and matches the Test/Live mode;
   - bounded: no unbounded fan-out; memory-safe (stream-everything).
4. **Run** — as a Temporal workflow; a step that is best done in n8n runs through `run_n8n_flow_activity`. Progress and receipts show in Review/Activity.

## The fork — owner picks one
- **A (default): Temporal plan, built in the Workbench.** A simple step list in the Review Actions panel (proposals pre-filled, drag to reorder, add/remove). Compiles to a generic Temporal "repair plan" workflow that runs registered Activities in order. n8n used only for a step that needs an n8n node (external API, AI node). Fastest to useful; one surface.
- **B: n8n canvas as the builder.** Each tool is an n8n node; the owner builds visually in n8n; the flow is bound and run via `run_n8n_flow_activity` as a Temporal activity. Real visual builder for free; validation runs over the n8n flow JSON. Two surfaces (Workbench + n8n).
- **C: Hybrid.** Build in the Workbench (A), with an "Open in n8n" to edit visually and round-trip.

## Build order (any option)
1. Fix the empty tool catalog (monitored-actions capabilities API + non-empty governed catalog) — prerequisite; also fixes the Go tools menu.
2. Real repair-apply tools (find-other-version-by-name via the catalog, salvage-truncated-XML, lenient decode), each an Activity.
3. Signature table + agent proposer.
4. Validator (the rules above), shown as a pass/fail checklist before Run.
5. Plan runner on Temporal (+ n8n binding for n8n steps).
