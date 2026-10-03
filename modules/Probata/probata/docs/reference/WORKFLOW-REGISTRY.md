# Workflow registry: what runs, where, and whether it's proven

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Owner rule (2026-10-02 20:02 EDT): record every workflow that works, and update this file in the same change whenever a workflow is proven or changed. Proven workflows get packaged as agentic, on-demand tools._

The Go engine orchestrates everything through Temporal. Python steps run as Activities on queue `evidence-pipeline`, and n8n flows run as Activities through `RunFlow`.

## Proven live on real data

| Workflow | Queue | What it does | Started by | Live proof |
|---|---|---|---|---|
| `ProfferWorkflow` | `proffer-v1` | One source file: register, retain (heartbeats), hash, repair assessment, handler selection (DuckDB / derive / decoder), parse, normalize, verify, resolve participants, match-up (same-device dedupe), Weaviate-first publish, preview, clean_checks auto-approval or owner decision, commit messages, threads and call log, seal | `proffer_batch_workflow`, the Workbench, the Temporal CLI | 2026-10-02: 741 runs. SMS, call logs and Facebook JSON for the live case |
| `proffer_batch_workflow` | `proffer-v1` | A folder of casevault keys, run as child ProfferWorkflows; inputs include `key_suffix`, `auto_approval`, perspective and `max_in_flight`; a new batch id skips completed sources | the Temporal CLI | 2026-10-02 |
| `proffer_call_log_backfill_workflow` | `proffer-v1` | Runs only `commit_call_log_activity` for already-completed runs | the Temporal CLI | 2026-10-02: 9,175 calls |
| `build_timeline_generation_activity` (Activity) | `evidence-pipeline` | Timeline projection after an extraction commit | `ExtractionCommitWorkflow` | Worker polling proven 2026-10-02; no extraction commit has run yet |

## Built, not yet proven live

| Workflow | Queue | State |
|---|---|---|
| `RepairPlanWorkflow` | `proffer-v1` | Tools: `find_other_version` (now detects scrambled copies), `salvage_truncated_xml`, `lenient_decode`. An optional `reentry` block (owner, perspective, auto_approval) is committed in 9a370986 and awaits deploy. First run: `sms-002-031.xml` |
| `EntityExtractionWorkflow` / `ExtractionCommitWorkflow` | `proffer-v1` | Go + kimi-k3 extraction into `working.candidate_*`, then commit. Not run on real data yet |
| Conversation-chunk branch (`proffer-conversation-chunks-v1`) | `proffer-v1` + `evidence-pipeline` | Distilbert chunks to `ProfferChunks20261002`, committed in fab0cdc0. Being moved to Weaviate-first before deploy |
| HTML handlers | `proffer-v1` | `facebook_messenger_html_v1` and `generic_html_document_v1` DuckDB templates, plus 7 HTML tool Activities. Committed, awaiting deploy |

## Being built (as Temporal workflows)

- AI-chat path: per-format DuckDB templates, topic chunks (`AiChatChunks20261002`), a legal second vector, and extracts to PG.
- Coco Super Index (`Intake/backend/src/casebible_index`): automatic indexing of the catalog, chunks to `CaseBibleChunks20261002`.
- Image and PDF steps as n8n flows run through `RunFlow`.
- Contacts import, placeholder people and re-link.
- Same-device duplicate removal.
- Re-chunk of existing Proffer threads, then removal of the per-message objects.
