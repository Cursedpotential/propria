# Adaptive discovery: implementation and qualification handoff

Byline: Codex · GPT-6 · 2026-10-11.

Status: first bounded discovery slice implemented on an isolated branch; live qualification pending. This is a repository handoff, not a verified deployment receipt or a new canonical product decision. The governed Docstore preflight/write/readback cannot run in this cloud session: no Docstore tool, credential or VPN capability is configured. Reconcile this handoff through Docstore when project access is available.

Owner source: [Propria Adaptive Discovery Implementation Plan](https://chatgpt.com/space/page_79edb2a9a89481918fe4d90ef706697f), accepted for implementation by “ok go.” Integration base is `origin/main` at `833ace8f99563ce7a628c40b0584c2ba765b4ef3`, branch `codex/adaptive-discovery-20261011`. Current recovery/source-preview and Workbench viewer branches were inspected; their full divergent trees are not merged into this change. Shared worker, starter, BFF, route and deployment edits are small composition changes for review alongside their owners' branches.

## Implemented slice

Discovery has its own opt-in configuration and workflow. Canonical runs, immutable input snapshots, score revisions, candidates, citations and append-only feedback live under `analysis.discovery_*`. Normalized inputs are read-only. PostgreSQL selects explicitly configured source generations; optional Surreal input selects a scoped subset and verifies its record/generation/revision pins against PostgreSQL before scoring. Missing or mismatched projection records cause an error rather than switching backends.

Existing explicit cross-platform memberships and native source coordinates preserve chronology and provenance. Similar people or nearby timestamps do not create a cross-platform episode. Recorded authors, attributed conduct actors and reported speech remain separate. As-lived input checks both occurrence and availability before limiting inputs; future or unknown clocks are excluded. Compatible historical feedback is bound to case, policy, dataset, selection, generations, hypothesis definitions and source input revisions.

Go assembles external bounded batches, orchestrates the workflow and persists results. Python performs one local CPU scoring operation, using semantic descriptions and explicit human correction examples. It can abstain, including on competing denial/neutral descriptions or missing citations. The scores are uncalibrated retrieval ranks. A wrong-person correction changes attribution independently of label learning; skip and unspecified partial-match feedback do not create label negatives. Corrected labels can raise the desired label and lower replaced labels without altering unrelated labels.

The Workbench `/discovery` page uses the shell's admitted case and policy, offers backend/dataset/generation choices, shows source-linked context and acknowledges corrections only after a canonical receipt. Run and feedback UUIDs provide exact retry identity. Corrections use the current run revision and require refresh on a conflict. Review has no graph-publication or evidence-promotion action.

Bounds are explicit: at most 400 input records, 8 MiB per retained JSON object, 100 candidates and 256 feedback examples. The browser starts with 200 records and 25 candidates. Partial input coverage is exposed; this slice does not claim to search an entire larger dataset in one run. Retained files are immutable and verified on read; workflow history carries references and counts rather than message bodies. Pending dispatches survive outages/restarts in PostgreSQL. Exhausted scoring stages record a safe failure status independently of the existing workflows.

## Deployment prerequisites

All three existing services default `ADAPTIVE_DISCOVERY_ENABLED=false`. No new worker service, paid provider or user-desktop runtime is introduced. Before enabling, qualify the existing Python/Go worker and starter together and add `requirements-discovery.txt` to Coolify's existing worker watch paths. The Python image includes the CPU retrieval dependency and reuses its existing CPU torch stack; model weights are not downloaded by discovery.

1. Review and apply only the additive `analysis.discovery_*` schema from the final bootstrap snapshot through the project database owner. Do not rebuild the live corpus or replay the entire snapshot. Verify runtime grants and constraints against the actual deployed schema.
2. Pre-create `/data/probata/volumes/proffer/derive-scratch/ai-content/analysis/discovery`, owned by the existing shared service UID 10001, with private permissions. Go and Python already share its parent bind mounts; the container path is `/data/proffer/derive-scratch/ai-content/analysis/discovery`.
3. Retain the selected CPU model and tokenizer at an immutable, read-only local directory visible to Python. Pin its actual deployment revision in `ADAPTIVE_DISCOVERY_MODEL_CONFIG_JSON`; configure its absolute `model_path`, backend and bounded settings. Test offline loading, model quality, memory, CPU and latency on the existing VPS before admitting real review traffic. A declared revision alone is not a weights-verification receipt.
4. Configure identical dataset definitions in Go worker/starter via `ADAPTIVE_DISCOVERY_DATASETS_JSON`: admitted case/policy, dataset ID/label, backend, selection revision and genuine normalized generation IDs. For Surreal, additionally configure the analytical URL/namespace/database, allowlisted table and separately mounted read-only token. The source adapter expects source-linked projection IDs and revisions, not arbitrary graph documents. The selected corpus determines whether the user-visible dataset name is accurate.
5. Deploy pinned image digests through existing Coolify applications. Verify discovery registration on `proffer-v1` and the CPU Activity on `evidence-pipeline`, then enable the independent flag. Existing workflows must still run with discovery disabled, enabled, unavailable and failed.

## Required live acceptance evidence

The cloud session has no configured database credentials, VPS access or VPN capability. No live schema apply, model quality evaluation, Coolify deployment, source-navigation check or database restart proof was performed here. Local fixtures and mocks do not establish those facts.

Collect genuine source-backed candidates and follow their recorded source locators through the actual reader. Save a correction, prove it remains after worker/starter/browser restart, and show the next score revision changes relevant ranking or attribution. Verify original normalized bodies and custody state remain unchanged. Test stale revisions, repeated retry keys, unavailable selected Surreal input, empty results, both historical clocks, partial selections and an exhausted model failure. Record deployed commit/image digests, exact run/feedback/score references, retained-file locations, rollback flag behavior and timing/memory measurements.

The plan's later phases remain pending: trained SetFit/small-transformer revisions and rollback, evaluated independent detector families, wider investigation expansion, optional media signals, owner-selected Surreal publication, and full rollout acceptance. This slice supplies the bounded retrieval/feedback foundation; it does not claim those phases are complete.

## Local validation

Record the final merged validation results here before opening the draft PR. Existing frontend lint errors in `read/context-search.tsx` and `sources/sources-screen.tsx` are present on the selected base and remain outside this change. Discovery-specific lint must pass; browser journeys and live acceptance are separate gates.
