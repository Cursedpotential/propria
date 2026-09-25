---
title: Repair workflow builder — propose, compose, validate, run on Temporal (+ n8n)
date: 2026-09-25
status: RATIFIED 2026-09-25 19:09 EDT — option A (Workbench step list, runs on Temporal, n8n only for steps that need it); V2 'Open in n8n' DEFERRED; build in progress
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

## Owner decision — 2026-09-25 19:09 EDT
**A**, "with V2 option to open in n8n — deferred for now." Build A; do not build the n8n round-trip.

## Shared contract (engine <-> Workbench) — both build to this
Plan JSON: `{ "plan_id": str, "source_ref": str, "preview_handle": str|null, "matter_mode": "TEST"|"REAL", "steps": [ { "step_id": str, "activity": "<registry id>", "params": {…} } ] }`.

Engine (proffer-starter, all under `/reference-import/repair/`, same auth as the other starter routes):
- `GET  tools` → `{ "tools": [ { "id", "description", "input_types": [..], "output_types": [..], "params_schema": {JSON Schema}, "writes": "derived"|"none", "needs_n8n": bool } ] }` — every repair-capable Activity.
- `POST propose` body `{ "source_ref", "preview_handle"?, "detect_report"? }` → `{ "proposals": [ { "signature", "rationale", "steps": [...] } ] }` — signature table first, agent proposer for uncovered signatures (marked `"by":"agent"`).
- `POST validate` body = plan → `{ "ok": bool, "checks": [ { "rule", "status": "pass"|"fail", "reason" } ] }` — fail-closed.
- `POST run` body = plan → 422 unless validate ok; else `{ "workflow_id", "run_id" }` — starts Temporal `RepairPlanWorkflow`; on success it re-enters proffer on the derived artifact (new proffer run, its preview_handle returned in status).
- `GET  runs/{workflow_id}` → `{ "status", "steps": [ { "step_id", "status", "receipt_ref", "output_ref"? } ], "reentry_preview_handle"? }`.

Workbench BFF passthrough: `/api/proffer/repair/{tools,propose,validate,run,runs/{id}}` (mode-checked like the other proffer routes).

Validator rules (engine): registered Activity per step with known types; type-checked chain (source type -> step1 in; stepN out -> stepN+1 in); no step writes the original (every write is a derived, hashed artifact); terminal output is a derived source proffer can ingest (re-entry appended implicitly — no stage skipped); destination resolves and Test/Live matches; bounded (≤ 12 steps, no cycles).

First repair Activities (each ONE Activity, streaming, bounded, retry-safe): `repair.find_other_version` (read-only catalog query: same basename, other hash / larger size -> re-points the plan's source), `repair.salvage_truncated_xml` (stream to the last complete record -> new derived artifact, hashed), `repair.lenient_decode` (SBV decode in lenient mode -> derived NDJSON).

## Build order (any option)
1. Fix the empty tool catalog (monitored-actions capabilities API + non-empty governed catalog) — prerequisite; also fixes the Go tools menu.
2. Real repair-apply tools (find-other-version-by-name via the catalog, salvage-truncated-XML, lenient decode), each an Activity.
3. Signature table + agent proposer.
4. Validator (the rules above), shown as a pass/fail checklist before Run.
5. Plan runner on Temporal (+ n8n binding for n8n steps).
