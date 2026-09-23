> _Byline: Claude Code (Sonnet subagent jev-eval-prep) · 2026-09-23_

# Jev Tier-1 Eval — Preparation Facts (read-only)

**Scope note:** the parent session (Opus) narrowed this prep to fact-gathering only,
mid-task (09:30 message). Step 4 of the original brief (draft plan, lettered options,
recommendations) was **not** performed. This document is facts with sources only — no
judgment, no recommendation, no plan. No API calls were made to Jev, OpenRouter, or
Anthropic. No writes were made anywhere except this file.

---

## 1. Key inventory

Presence/absence only. No values or lengths printed anywhere in this investigation.

### Desktop `C:\Users\matts\.secrets\*.env`

Checked with `grep -nE '^\s*KEY\s*=' *.env` (never sourced).

| Key | Present? | File(s) |
|---|---|---|
| `OPENROUTER_API_KEY` | **PRESENT** | `Agno-MCP-Platform.env` (line 99), `probata.env` (line 99) |
| `TYPESAFE_API_KEY` | **MISSING** | not found in any `.secrets\*.env` file |
| `ANTHROPIC_API_KEY` | **MISSING** | not found in any `.secrets\*.env` file. `anthropic.env` exists but its only key is `CLAUDE_CODE_OAUTH_TOKEN`, not an API key |

Search for any key name containing `TYPESAFE` or `JEV` (case-insensitive) across every
`.secrets\*.env` file: **zero matches**.

### Process environment (this session)

Checked with `env | cut -d= -f1 | grep -E '^(OPENROUTER_API_KEY|TYPESAFE_API_KEY|ANTHROPIC_API_KEY)$'` and a broader `typesafe|jev|anthropic|openrouter` sweep of all env var names.

| Key | Present? |
|---|---|
| `OPENROUTER_API_KEY` | **PRESENT** |
| `TYPESAFE_API_KEY` | **MISSING** |
| `ANTHROPIC_API_KEY` | **MISSING** (only `ANTHROPIC_BASE_URL` is set — this session authenticates via Claude Code OAuth, not an API key) |

No env var name containing `TYPESAFE` or `JEV` found.

### VPS `ovh-files` (`ssh -i ~/.ssh/ovh root@100.91.190.107`), `/data/probata/secrets/**`

`find /data/probata/secrets -type f` — file names only:

```
casebible-b2.json, casebible-b2.json.bak-20260920T180719-no-delete-capability,
casebible-b2.json.bak-20260920T205159-vault-prefix-only, casebible-r2.json,
dbgate/dbgate.env, intake-engine/cb-agent, intake-engine/gemini-api-key,
intake-engine/metabase-ro, intake-engine/nvidia-api-key, intake-engine/ollama-api-key,
n8n/proffer-auth, pgadmin/default_password, pgadmin/pgpass, platform/database-url,
proffer/preview-cursor-key, proffer/service-token, spacedrive-gate/sd_auth,
spacedrive-gate/sd_auth.bak-20260915T182804Z-pre-owner-login,
to_be_deleted/intake-engine-engine.env-20260917-smoke,
to_be_deleted/surreal-intake-ROOT-copy-20260919-not-wired,
tool-gateway/service-token, tsnet/authkey
```

No file name containing `typesafe` or `jev`. Var names (not values) inside
`dbgate/dbgate.env`: `CONNECTIONS, LABEL_c1, SERVER_c1, PORT_c1, USER_c1, PASSWORD_c1,
ENGINE_c1, DATABASE_c1, LABEL_c2, SERVER_c2, PORT_c2, USER_c2, PASSWORD_c2, ENGINE_c2,
DATABASE_c2` — no TypeSafe/Jev/OpenRouter/Anthropic names. The other files under
`/data/probata/secrets` are single bare-value files (no `KEY=` structure).

Container env var **names** checked via `docker exec <c> env | cut -d= -f1` for
`intake-engine-dbae59tufgs5zqvb7ym9fozk-145023559708` and
`probata-db-w10gg3an43jvry4y79n6sxi1-122959880896`, grepped for
`openrouter|typesafe|anthropic|jev`: **zero matches** in either container.

`docker ps` container names on ovh-files grepped for `typesafe|jev`: **zero matches**.
`grep -riEl 'typesafe|jev-1|jev_1' /data/probata/secrets/ /data/probata/config/`:
**zero matches**.

**Summary: `TYPESAFE_API_KEY` does not exist anywhere checked (desktop, process env, or
VPS). No key name suggesting TypeSafe or Jev exists anywhere checked.**

---

## 2. Jev API facts — verification against live public sources

Method: WebSearch + WebFetch against `docs.typesafe.ai`, `openrouter.ai`, `pypi.org`,
`github.com/tumf/jev-cli`. No API calls made. Each row: confirmed / contradicted /
not found, with source URL.

| Handoff claim (§1) | Status | Source |
|---|---|---|
| TypeSafe direct endpoint `POST https://api.typesafe.ai/v1/systemone` | **Confirmed** | https://docs.typesafe.ai/api.md |
| Auth `Authorization: Bearer $TYPESAFE_API_KEY` | **Confirmed** | https://docs.typesafe.ai/api.md |
| Pinned model ID `jev-1.13.0` valid | **Confirmed** — `jev-1.13.0` is the current stable versioned ID; `jev-latest` and `jev-preview` both currently resolve to it | https://docs.typesafe.ai/models.md |
| OpenRouter model ID `typesafe/jev-1.13` | **Confirmed** | https://openrouter.ai/docs/guides/community/jev (WebFetch); https://openrouter.ai/typesafe/jev-1.13 (WebSearch snippet only — direct WebFetch of this URL returned 404 both times) |
| OpenRouter endpoint `POST https://openrouter.ai/api/v1/systemone` | **Confirmed, but incomplete** — this is one of two documented OpenRouter surfaces. The docs page also documents a second surface, `POST https://openrouter.ai/api/alpha/decisions` ("Decisions API"), which the handoff does not mention | https://openrouter.ai/docs/guides/community/jev |
| TypeSafe direct context "64k/request (state + longest question ≤32k)" | **Confirmed verbatim** — "64k tokens per request"; "32k tokens for `state` plus the longest question" | https://docs.typesafe.ai/models.md |
| OpenRouter context "32K listed" | **Confirmed verbatim** — "32,000 tokens. That's the `state` you send plus the questions." | https://openrouter.ai/docs/guides/community/jev |
| Price $0.042/M input, output free (both providers) | **Confirmed for TypeSafe direct** — docs state "$42 per Btok input" for `jev-1.13.0` (= $0.042/M) | https://docs.typesafe.ai/api.md |
| | **Confirmed for OpenRouter, lower-confidence source** — WebSearch snippet of the OpenRouter model page states "$0.042/M input tokens and $0.00/M output tokens"; direct WebFetch of `openrouter.ai/typesafe/jev-1.13` 404'd twice, so this line is sourced from the search snippet only, not a direct page fetch | WebSearch snippet, https://openrouter.ai/typesafe/jev-1.13 (fetch failed) |
| TypeSafe direct rate limit "1,200 req/min, 250k tok/s" | **Confirmed verbatim** — "250,000 tokens per second"; "1,200 requests per minute." Docs add: "rate limits are subject to dynamic adjustment ... without notice" | https://docs.typesafe.ai/models.md |
| OpenRouter rate limit "per OpenRouter account" | **Not found / not independently checkable** — no specific figure published | — |
| "Not chat completions... Decisions API" | **Confirmed** — OpenRouter docs state Jev requires its dedicated Decisions/System One endpoints, not chat completions | https://openrouter.ai/docs/guides/community/jev |
| Python SDK `pip install typesafe-sdk` | **Confirmed** — PyPI page lists `pip install typesafe-sdk` / `uv add typesafe-sdk`, Python 3.10–3.14, MIT | https://pypi.org/project/typesafe-sdk/ |
| SDK reads `TYPESAFE_BASE_URL` / `TYPESAFE_API_KEY` | **Confirmed** — SDK constants page lists `TYPESAFE_API_KEY`, `TYPESAFE_BASE_URL` (default `https://api.typesafe.ai`), `TYPESAFE_DEFAULT_MODEL` (default `jev-latest`), `TYPESAFE_LOG_LEVEL` | https://docs.typesafe.ai/sdk/python/api/constants.md |
| "For OpenRouter, set `TYPESAFE_BASE_URL=https://openrouter.ai/api` and put the OpenRouter key in `TYPESAFE_API_KEY`" | **Confirmed verbatim** — "set TYPESAFE_BASE_URL=https://openrouter.ai/api and TYPESAFE_API_KEY=<your OpenRouter API key> instead" | https://openrouter.ai/docs/guides/community/typesafe-sdk |
| Smoke-test CLI `pip install jev-cli`, then `jev auth test --provider openrouter`; official provider is default | **Confirmed for command syntax and default-provider behavior**; `jev-cli` 0.6.3 on PyPI, MIT, `jev auth test --provider openrouter` and `jev auth test --provider vercel` both shown as supported syntax, default provider (no `--provider` flag) is the official TypeSafe API | https://pypi.org/project/jev-cli/ ; https://github.com/tumf/jev-cli |
| | **Minor discrepancy** — the GitHub README's documented install command is `uv tool install jev-cli`, not `pip install jev-cli`. The package is on PyPI and pip-installable (confirmed by the PyPI page itself), so `pip install jev-cli` should also work, but it is not the install command the source documents as canonical | https://github.com/tumf/jev-cli |
| Request shape: `model`, `state`, `questions{type, instructions, criteria}` | **Confirmed** | https://docs.typesafe.ai/api.md |
| Response shape: noul `{type:"noul", noul: <float>}`; choice returns pick + per-option probabilities + confidence; usage block with input/output tokens | **Confirmed** | https://docs.typesafe.ai/api.md |
| `state: null` → 422 error | **Not found** — docs confirm `422 Unprocessable Entity` exists as a general "validation failure" error code, but the specific null-state trigger is not stated in the fetched pages | https://docs.typesafe.ai/api.md |
| Choice: 1–255 labels | **Confirmed (max side only)** — docs state "Maximum 255 options" for choice criteria; a documented minimum of 1 was not found in the fetched text | https://docs.typesafe.ai/api.md |
| Score: 2–10 levels | **Confirmed** — "ordered array of 2-10 level descriptions" | https://docs.typesafe.ai/api.md |
| Question ID never sent to the model | **Confirmed verbatim** — "not sent to the underlying model and is not used in inference" | https://docs.typesafe.ai/api.md |
| All questions in one call answered in parallel; extra questions "nearly free" | **Not independently verified** — not found or denied in the fetched pages | — |
| Weak spot: dates, numbers, counting | **Confirmed verbatim** — "jev-1.13 reads dates as text, not as ordered quantities"; "does not count reliably"; "Jev is not a calculator" | https://docs.typesafe.ai/model-jaggedness/jev-1.13 |
| Weak spot: adversarial/manipulative phrasing | **Confirmed verbatim** — "Content written to adversarially steer the model ... can move the answer" | https://docs.typesafe.ai/model-jaggedness/jev-1.13 |
| Weak spot: non-English text | **Confirmed, qualified** — TypeSafe does not publish a negative claim; official guidance is "no published multilingual evaluation yet" and recommends testing on real multilingual traffic "before trusting it with non-English conversations," despite pretraining on 50+ languages | WebSearch snippet (source page not independently re-fetched) |
| Weak spot: overlapping labels | **Confirmed** — the Choice primitive docs discuss exactly this failure mode (example: "urgent" and "important" overlapping, leaving ownership unresolved) and prescribe writing non-overlapping, observable criteria | https://docs.typesafe.ai/primitives/choice (via WebSearch snippet) |
| Weak spot: irrelevant context lowers accuracy | **Confirmed verbatim** — "Accuracy falls as the state grows with content unrelated to the decision... Jev suffers from context rot" | https://docs.typesafe.ai/model-jaggedness/jev-1.13 |
| "It can be wrong at confidence 1.00" | **Confirmed in substance, not the literal figure** — "A result with high confidence can still be wrong" (third-party reference gist); TypeSafe's own jaggedness page separately notes "score levels are weak in numerical calibration" | https://gist.github.com/pjburnhill/adf8d28efcad9df037bfdece178ef965 ; https://docs.typesafe.ai/model-jaggedness/jev-1.13 |

**Net: no claim in handoff §1 was contradicted by a live source.** One claim is
incomplete rather than wrong (the OpenRouter endpoint list omits a second, alpha
Decisions API surface). One claim (pip vs. uv install for `jev-cli`) names a working
but non-canonical install command. Two claims (`state:null` → 422, "nearly free" extra
questions) were not found in the fetched docs either way.

---

## 3. Message data inventory (read-only, `SELECT ... LIMIT` only)

### 3a. Catalog: PostgreSQL `casebible` on ovh-files, port 5475 (container
`fgz1n7useplhk0t91uk7k1aw`), schema `raw_duck`, role `metabase_ro`
(password read from `/data/probata/secrets/intake-engine/metabase-ro`, never printed;
connected via `docker exec -e PGPASSWORD=... fgz1n7useplhk0t91uk7k1aw psql -h 127.0.0.1 -p 5432 -U metabase_ro -d casebible`).

**`raw_duck.chat_events_20260918`** — the deduped, normalized message/call/AI-chat event table.

- Row count: **551,877**
- Date range (`event_ts_utc`): **2014-01-10 02:52:47 UTC → 2026-09-11 21:59:19 UTC**
- Columns: `dedup_key, n_sources, sort_ts, event_ts_utc, tz_status, ts_original, ts_field, source_format, event_kind, conversation_title, conversation_id, sender, recipients, participants, direction, counterparty_phone, contact_name, body, attachments, katrina_ref_type, katrina_conf, catrina_class, daughter_conf, custody_hit, housing_hit, vault_key, catalog_rel`
- Has sender: yes (`sender`, `counterparty_phone`, `contact_name`)
- Has direction: yes (`direction`) — value breakdown: blank `''` 437,965; `unknown` 65,163; `received` 24,109; `sent` 16,217; `outgoing` 3,837; `missed` 1,976; `incoming` 1,344; `rejected` 726; `refused_list` 534; `failed` 6
- Has thread: yes (`conversation_id`, `conversation_title`) — 1,906 distinct `conversation_id`
- Has timestamp: yes (`event_ts_utc` + `sort_ts`/`ts_original`/`ts_field`/`tz_status`)
- `source_format` breakdown: `fb_messenger_json` 434,560; `sms_backup_xml` 105,495; `calls_backup_xml` 8,417; `ai_conversations_json` 2,437; `whatsapp_txt` 625; `google_chat_json` 221; `ai_chat_file` 122
- `event_kind` breakdown: `message` 540,901; `call` 8,417; `ai_chat_turn` 2,437; `document` 122
- Source file path (`catalog_rel`): non-null for 482,075 / 551,877 rows (87.4%)
- Vault key (`vault_key`): non-null for 551,877 / 551,877 rows (100%)
- sha256: no sha256 column on this table directly
- No `iMessage`/`whatsapp_txt`... note: no `imessage` `source_format` value present at all in this table

**`raw_duck.chat_event_provenance_20260918`** — per-source-occurrence provenance, joins to `chat_events_20260918` on `dedup_key`.

- Row count: **1,080,505** (more than 1 row per deduped event — multiple raw occurrences collapse into one event)
- Columns: `dedup_key, event_uid, source_format, extractor, vault_key, sha1, catalog_rel, member_path, record_index, ts_original, ts_field`
- `sha1` non-null: 1,068,694 / 1,080,505 (98.9%)
- Every row's `dedup_key` matches a row in `chat_events_20260918` (inner join count = 1,080,505, i.e. full provenance coverage)

**`raw_duck.chat_candidates_20260918`** — candidate chat-format files discovered but not necessarily parsed into `chat_events`.
- Row count: 6,431; `sha1` non-null: 6,401
- Columns: `format_guess, vault_key, size, sha1, catalog_rel_example, n_catalog_occurrences, discovered_at`

**`raw_duck.backup_message_records_20260920`** — raw parsed XML node tree, not a normalized message table.
- Row count: **11,678** (`mms` 9,543; `sms` 2,135)
- Columns: `fingerprint, kind, payload (jsonb)`
- `payload` jsonb keys (both `sms` and `mms`): `attributes, children, tag, text` — i.e. this is a raw parse tree (XML tag/attrs/children), not sender/direction/timestamp columns. No normalized sender/direction/timestamp fields at this table's top level.

**`raw_duck.backup_message_occurrences_20260920`**
- Row count: 46,207; distinct `backup_sha256`: 4
- Columns: `backup_sha256, ordinal, fingerprint`

**`raw_duck.missing_message_payloads_20260920`**
- Row count: 16, all `status = missing_embedded_payload`
- Date range (`message_timestamp_utc`): 2025-08-29 18:07:21 UTC → 2025-09-02 19:19:54 UTC
- Columns: `payload_id, backup_sha256, message_timestamp_ms, message_timestamp_utc, filename, content_type, status, payload (jsonb)`

**`raw_duck.ai_chat_probe_20260918`** — 122 AI-chat file candidates (sha1 present), columns `vault_key, size, sha1, ext, probe_class, catalog_rel, catalog_path, catalog_modtime_hint, n_catalog_occurrences, discovered_at`.

Every table name in `raw_duck` matching `%message%|%sms%|%mms%|%thread%|%chat%|%whatsapp%|%imessage%|%facebook%|%messenger%`:
`ai_chat_probe_20260918, backup_message_occurrences_20260920, backup_message_records_20260920, chat_candidates_20260918, chat_dir_files_20260918, chat_directories_20260918, chat_event_provenance_20260918, chat_events_20260918, missing_message_payloads_20260920`.

### 3b. B2 raw export locations recorded in the catalog (`raw_duck.b2_objects`, catalog only, no bucket listing/download performed)

- `b2_objects` total rows: **530,070**; `listed_at` (single catalog snapshot timestamp): **2026-09-14 18:11:17 UTC**
- Columns: `key, size, sha1, listed_at`
- Counts of `key` matching platform patterns (ILIKE):

| Pattern | Count |
|---|---|
| `%sms%` or `%mms%` | 960 |
| `%messenger%` or `%facebook%` | 46,489 |
| `%whatsapp%` | 91 |
| `%imessage%` | 613 |
| `%AI_Chat%` | 277 |

- All matching keys share the prefix `consignatio/intake/raw-dedupe/v1/...` (46,489 + 960 = 47,449 of the 48,421-row combined match set fall under `consignatio/intake/raw-dedupe/v1`).
- SMS/MMS message-body export files matching `sms-<14-digit-timestamp>.xml`: **103** files, e.g.
  `consignatio/intake/raw-dedupe/v1/source-buckets/casebible-raw/_backup_import/Documents/Court/Phone Records/Messages with Katrina/SMS backup/sms-20250615033115.xml`
- `calls-*.xml` files (call logs, separate from SMS body) also present under the same directories.
- iMessage: attachment files (png/jpg/heic/etc., 338+79+59+51+... files) live under
  `.../Evidence/Phone Records/Messages with Katrina/imessage exports/+18102689630/attachments/`.
  Only **2** actual message-body export files were found (not attachments):
  - `.../_backup_import/Documents/Court/Phone Records/Messages with Katrina/imessage exports/+18102689630/imessage export 8102689630 2023-2024.html`
  - `.../_backup_import/Documents/Court/Phone Records/Messages with Katrina/imessage exports/+18103533592/index.html`
- WhatsApp matches (91) are almost entirely screenshot images (`Screenshot_*_WhatsApp.png`) under OneDrive Camera Roll paths, plus one Obsidian plugin file (`whatsapp-backup/styles.css`) — no WhatsApp chat-export text file was found in this sample.
- Facebook Messenger JSON files sampled directly (not just counted): e.g.
  `consignatio/intake/raw-dedupe/v1/source-buckets/casebible-raw/court/fb/messenger_active_status_platform_settings.json` (settings file, not message content in this specific sample) and Google-Takeout-derived Messenger screenshot sidecar JSON under `casebible-sorted/EvidenceVault/exports/google-takeout/...`.

### 3c. Probata's own DB (`probata-db-w10gg3an43jvry4y79n6sxi1-122959880896` on ovh-files, published port 5432)

Container env var **names** read (values never printed, used only inline for the connection): `POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD, PG_MAJOR, PG_VERSION, PGDATA`.

Databases present in this Postgres instance (`\l`): `archive, casebible, contextforge, infisical, platform, platform_baseline_test, platform_admin-owned platform_preburn_20260830, postgres, template0, template1, temporal, temporal_visibility, traceiq`.

The `working`/`context` schemas referenced in `AGENTS.md` live in the **`platform`** database (not a database literally named `casebible` or `probata` — the container is named `probata-db-...` but the app database inside it is `platform`).

Schemas in `platform`: `ai, analysis, archive, canon, context, duckdb, evidence, ext, knowledge, media, ops, public, raw, reference, registry, timeline, working` (plus system schemas).

Message-related tables in `working`/`context` (schema only, from `information_schema.tables`):
`working.message, working.chat_message, working.chat_conversation, working.chat_message_event, working.chat_conversation_event, working.third_party_message, working.third_party_conversation, working.third_party_message_participant, working.message_participant, working.message_projection_route, working.normalized_record, working.vw_message_sms, working.vw_message_imessage, working.email, working.call_log`, plus `context.proffer_preview_message, context.raw_ndjson, context.raw_callsbackuprestore_xml, context.source, context.source_metadata, context.hash_receipt`, and related.

**Row counts (`SELECT count(*)`):**

| Table | Row count |
|---|---|
| `working.message` | 0 |
| `working.chat_message` | 0 |
| `working.third_party_message` | 0 |
| `working.normalized_record` | 0 |
| `working.vw_message_sms` | 0 |
| `working.vw_message_imessage` | 0 |
| `working.email` | 0 |
| `working.call_log` | 0 |
| `context.proffer_preview_message` | **2,680** |
| `context.raw_ndjson` | 10,814 |
| `context.raw_callsbackuprestore_xml` | 610 |
| `context.hash_batch_member` | 33,413 |
| `context.hash_receipt` | 16,727 |
| `context.hash_manifest_member` | 16,697 |
| `context.raw_record_identity` | 11,424 |
| `context.normalized_record_identity` | 5,273 |
| `context.normalization_lineage` | 5,273 |
| `context.source` | 22 |
| `context.source_metadata` | 19 |

(Row-count source for the full schema sweep: `pg_stat_user_tables.n_live_tup` for `working`/`context`/`raw`/`evidence` schemas, cross-checked with direct `count(*)` on the tables listed above.)

**`context.proffer_preview_message`** (the only message table in this database with data):
- Columns: `preview_handle, snapshot_seq, message_id, ordinal, sent_at, sender_participant_id, body, participant_ids (array), source_locator_ref`
- Date range (`sent_at`): **2025-06-01 14:46:00 UTC → 2026-06-07 20:59:58 UTC**
- Distinct `source_locator_ref`: 1,753
- No explicit `platform` or `direction` column; `source_locator_ref` values sampled include both `context.normalized_record_identity/<uuid>` references and direct locators.

**`context.source`** (22 rows) — `source_key` values sampled (all 22, `provenance_class` = `unknown` for all):
mix of `upload://<sha256-like-hash>` handles, `r2://nexus/proffer/test-fixtures/probata-test-20260912/{sms-backup.xml, imessage-thread.txt, chatgpt-conversations.json}` (explicitly named as test fixtures), `r2://casebible-sorted/*` (a handful of real-looking files: `.xml`, `.jpg`, `.pdf`, `.csv`), and `b2://salem-data/consignatio/vault/v1/{sms-*.xml, calls-*.xml, sms-*.xml.derived/threads/*.ndjson, moved/Evidence/SMS Backup and Restore/sms-*.xml}`.
- `context.source_metadata` columns: `id, source_version_id, raw_record_id, metadata_class, metadata (jsonb), extractor_id, extractor_version, extraction_activity_receipt_id, generated_at, created_at`.
- `context.hash_receipt` columns (hash-coverage table, schema only — not joined/counted per-message): `id, activity_receipt_id, hash_kind, algorithm, digest (bytea), construction, hash_manifest_id, source_version_id, raw_record_id, raw_generation_id, normalized_record_id, normalized_generation_id, computed_at, computed_by, created_at`.

---

## Sources consulted (URLs)

- https://docs.typesafe.ai/api.md
- https://docs.typesafe.ai/models.md
- https://docs.typesafe.ai/model-jaggedness/jev-1.13
- https://docs.typesafe.ai/sdk/python/api/constants.md
- https://docs.typesafe.ai/concepts/system-one
- https://docs.typesafe.ai/llms.txt
- https://openrouter.ai/docs/guides/community/jev
- https://openrouter.ai/docs/guides/community/typesafe-sdk
- https://openrouter.ai/typesafe/jev-1.13 (WebFetch 404 both attempts; used via WebSearch snippet only)
- https://pypi.org/project/typesafe-sdk/
- https://pypi.org/project/jev-cli/
- https://github.com/tumf/jev-cli
- https://gist.github.com/pjburnhill/adf8d28efcad9df037bfdece178ef965

## SQL/commands used (catalog and Probata DB)

- `casebible` (raw_duck): `docker exec -e PGPASSWORD=<from metabase-ro file> fgz1n7useplhk0t91uk7k1aw psql -h 127.0.0.1 -p 5432 -U metabase_ro -d casebible -At -c "<SELECT ... LIMIT ...>"`
- `platform` (working/context, on `probata-db-...`): container env var values read once via `docker exec ... env` and piped directly into `psql`, never echoed; `psql -h 127.0.0.1 -p 5432 -U <POSTGRES_USER> -d platform -At -c "<SELECT ...>"`
- All queries were `SELECT`/`information_schema`/`pg_stat_user_tables` reads with `LIMIT` where rows (not aggregates) were returned. No `INSERT`/`UPDATE`/`DELETE`/`CREATE` statements were issued.

## Addendum 2026-09-23 14:25 EDT: Katrina message pool (Claude Code · Opus 5.5)

Owner 14:20–14:22: sample messages between the owner and Katrina. Her numbers were 810-268-9630 (until 2024) and 810-353-3592 (2024–2025); an 810-869 number (about 2018 to early 2019) is dropped by owner order. Pull Facebook too. The 2018–19 texts would be an iMessage export and are not in this table.

Catalog `raw_duck.chat_events_20260918` (read-only queries on the casebible PG container):

| Pool | Rows | Dates | Notes |
|---|---:|---|---|
| fb_messenger_json, title "Katrina Kinzel" | 67,377 | 2018-08-05 → 2025-08-16 | the largest pool |
| sms_backup_xml, title "Katrina Kinzel", counterparty **810-295-9303** | 40,704 | 2021-02 → 2022-12 | a number the owner did not list; sender values "Katrina Kinzel" / "owner". **Confirmed 2026-09-23 14:23 (owner, after one swap):** 810-295-9303 was Katrina's number; 810-295-9302 was the owner's. `counterparty_phone` is correct here; "owner" = Matt on this device. Owner: "there's a lot to work with there." |
| sms_backup_xml, 2024 backup where **810-268-9630 is the recipient on every thread** | 8,014 in thread "Matthew" (28,166 rows in that backup overall) | 2024 | 268-9630 looks like **the device's own number**, i.e. Katrina's phone. "Matthew" = the owner, and `sender='owner'` means Katrina in this backup. **Confirmed by the owner 2026-09-23 14:23 ("yes it was").** Direction must be computed per device, never from the literal `owner` label. |
| sms_backup_xml, counterparty 810-353-3592 ("Katrina" / "Katrina Kinzel") | 4,259 | 2025-06-01 → 2026-01-27 | plus 489 calls |
| whatsapp_txt "WhatsApp Chat - Katrina Kinzel" | 599 | timestamps not parsed | Jev can read it, but the date strata can't use it |
| 810-869-5919 | (2,859 SMS) | 2024–2026 | contact "Jesse Baker"; **not Katrina**; excluded |

Other facts:
- `direction` is empty for every SMS row; only calls carry it. Sender is populated.
- The table has earlier Katrina columns (`katrina_ref_type`: direct 114,801, name_mention 686, …); these are not reference labels.
- SMS coverage by year: 2021 17,867 · 2022 23,150 · 2023 26 · 2024 28,166 · 2025 12,420 · 2026 23,866.
