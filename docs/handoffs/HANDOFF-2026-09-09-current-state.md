# HANDOFF — current platform state, consolidated from every prior handoff (2026-09-09)

> _Byline: Claude Code · Fable 5.1 · 2026-09-09 06:10 EDT_
> _Owner directive 2026-09-09: one handoff, in the handoffs folder, same format as the newest handoff, carrying only what is still live. Sources consolidated: the 8 dated handoffs, the 9 rename-followup files, `HANDOFFS.md`, and the compact summaries 2026-09-02 through 09-08. Verified against `docs/DECISION_LOG.md` through D-153 (newest ruling), a structural code check (Haiku, smart-explore), and a session-log sweep (Sonnet, DuckDB over Claude Code and Codex logs). Items neither source could settle are marked UNVERIFIED, never assumed._
STATUS: PARTIAL
BUILD_STATUS: UNKNOWN (not run this session; the 2026-09-06 handoff recorded Go green in a Linux container and one Python test failure; treat as stale)

## READ FIRST

- This is a custody case. Nothing is "live" until the owner can assemble messages, build the timeline, and prove custody. Ingest first; the evidence lane is out of scope until ingest is through (D-151c).
- Work surface = `modules/workbench/` (probata Workbench). SBV / `intake` is a separate desktop surface (D-123, D-150), never deployed as the platform surface.
- **No migrations, ever** (D-153). The database is `sql/bootstrap/schema_snapshot_20260907.sql` plus `scripts/rebuild_platform_from_snapshot.sh`. Any proposal that says "apply migration NNNN" is a defect.
- Names (D-137..D-142): repo and product `probata`; ingest lane `proffer` (was `uiw`); vault `consignatio`; legal workbench `advocatio`; geo `vestigia`; analysis engine `indagatio` (not built). Old names in older docs are history, not errors.
- The single open-work register is `docs/MASTER-TODO-2026-09-09.md`. Every unresolved item below has a `T-` or `OW-` id there. Do not open a second list.
- Owner rulings 2026-09-09 on tooling: the agent memory system is one shared SurrealDB on the VPS that every agent uses, unrelated to walks or horizon; all documents stay local in an embedded SurrealDB on the desktop, no Docker, no Coolify; **never shut down Neo4j**; searches go through `ccc` and DuckDB before grep.

## Verified-live state (do not re-derive)

| Thing | State |
|---|---|
| Rulings | `docs/DECISION_LOG.md` tops out at **D-153 (2026-09-07)**. The session-log sweep found no owner ruling after it on the identifier list R-1..R-18, the deny-list restore, or the consolidated-folder question; the only owner reply on those (2026-09-06 17:38) declines to rule ("need WAY MORE CONTEXT AND DETAILS"). |
| Platform DB | ovh-files `100.91.190.107:5432/platform`, Coolify app `agentos-db` (rename to `probata-db` is R-3, unruled). Rebuilt 2026-09-07 from the snapshot with keep-set restore: labels 1,918/1,918, `media.enrichment` 15,252, `reference.detection_pattern` 527 confirmed before and after (D-152). `agno_app` role gone. `uiw_*` objects 0, renamed `proffer_*`. pg_duckdb 1.1.0 / DuckDB 1.4.3; `webbed` extension installed, `read_xml` proven (2026-09-06). |
| Ingest mechanism | DuckDB ELT templates plus Go decoders, routed by a signature-keyed registry (D-149 items 9–10). Native Go chunker `modules/engine/chunk/chunk.go` exists (169 lines) and replaced the Python bake-off (D-151a). Chunks are ids-only in PG; Weaviate holds vector, coordinates, and member ids, no text (D-149 items 7–8, target `EvidenceChunkV2`). |
| Decode subtree | **Not started.** `modules/engine/decode/` does not exist; the `go.mod` replace directive still points at the SBV fork (`modules/forks/sbv`). D-131 absorption unexecuted. |
| Parser fix | SBV decoder fixed 2026-09-06 15:38: SMS 11,676/11,676, calls 1,457/1,457, iMessage HTML and TXT 1,918/1,918 on the real exports. Was uncommitted in the SBV fork working tree at handoff time; commit status UNVERIFIED since. |
| Gateway / Stage 2 | `svc:tool-gateway` live since 2026-09-05; worker repointed; rehearsal reached the HITL repair gate (9 stages). D-149 confirms the gateway and decoder-registry model is still the design. |
| Storage tiers (D-149) | vault `r2:casebible-sorted` · block `/data/test_data/<source_type>/<export-folder>/` · `nexus` upload staging (10 objects, 1.24 GiB incl. 4 real-run leftovers the owner deletes). Custody sealing at PROMOTION only (D-152); no hashing at ingest. |
| Directory rename | Checkout folder `Agno-MCP-Platform` to `probata` DONE 2026-09-06/08 (commit `38a3ea3`; junction aliases retired). Infra identifier renames were in progress as hard cuts on 2026-09-07 (phases 4.1–4.2 done, 4.3–4.6 pending at last record). Structural check today: 29-plus occurrences of the old identifiers (`agentos-db`, `AGENTOS_`, `/data/agno`, `uiw`, `universal_import`, the ghcr image name) remain in tracked config and code. Completion UNVERIFIED. |
| Fleet, per Coolify on 2026-09-09 (JSON-parsed, not text-matched) | `surreal-case` running healthy · `data-neo4j` running healthy (**never shut down**) · `data-graphiti-case` and `data-graphiti-files` exited unhealthy since 2026-09-07 (retired by D-070, correctly down) · `data-surreal-phase1-t0-r1` exited since 2026-09-07 · two Weaviate apps running (`data-weaviate-files`, `data-weaviate-native-v1`; T-007 unresolved) · `casebible-pg18` database running healthy with nightly backups succeeding (last 2026-09-09 02:00). |
| Retired, do not resurrect | Graphiti (D-070), LiteLLM docs (ADR-0042; container never torn down, T-086), Windmill (ADR-0029), AgentOS as API host (D-101/D-107), Milvus as platform vector store (ADR-0040; Milvus is still memsearch's backend, never say "Milvus is down"), Traefik forward-auth Authentik (D-133), Next.js Workbench (D-108), `EvidenceChunkV1` (D-149). SurrealDB is not retired: only the legacy Agno operational adapter was (D-061). |
| Docs tree | 2026-09-08 restructure (commit `288591e`): `docs/wiki/` removed (575 files), planning pruned (72 files), handoffs moved to `docs/handoffs/`, `docs/archive/` does not exist, `docs/private/` gitignored. 87 dead intra-docs links, 46 orphans. MinHash census 2026-09-09: 493 text files, 1 near-duplicate pair, 0 exact, 0 shared boilerplate; the corpus is semantically stale, not textually duplicated. |
| Local SurrealDB | 3.2.0 installed via winget (not on PATH); no data directory anywhere; Python SDK absent; Docker absent. Nothing stood up yet. |
| Git | Working tree carried ~522 dirty paths on 2026-09-09, including another session's staged set (`.review_hold/`, `.agents/blueprint/` deletions). Do not broad-stage. |

## Findings / work done (this consolidation, 2026-09-09)

- Eight read-only walkers (Sonnet) covered every file under `docs/` except `private/`; a Haiku structural pass verified the register's code claims (85 percent held; two corrected: the decode subtree does not exist, and `server/evidence/schemas/` is gone while `server/evidence/config/` remains); a Sonnet session-log sweep of 46 keywords across Claude Code and Codex logs settled 13 rows and left about 80 with no log evidence either way.
- The TODO cluster collapsed to 96 still-open items, about 26 done-but-never-closed, 14 dead, now in `docs/MASTER-TODO-2026-09-09.md`. The three `URGENT-TODO` backup copies are strict subsets of the live file. The five TODO-SNAPSHOT JSON files are hook output with nothing not captured elsewhere.
- Five of ten compact summaries (08-18, 08-29, 08-30, half of 08-24, and today's 09-09) contain no summary at all, only raw PostCompact hook payloads. The hook is still emitting empty files.
- The ADR set is mostly well-bannered. Gaps: the five Graphiti ADRs D-070 suspended (0014, 0031, 0037, 0038, 0039) carry no banner; ADR-0027 has no superseding language despite ADR-0040; at least five ADRs (0004, 0006, 0023, 0025, 0046) still assert AgentOS as the live host against D-107. In-place banners, not moves.
- `docs/reviews/2026-09-05-ingest-day-live-chain.md` superseded three same-week reviews within 48 hours: the working example of consolidation done right.
- Taxonomy proposal written: `docs/reviews/2026-09-09-docs-taxonomy-simplification.md` (seven types, domains from the naming canon, status field, hook repointing, ingest-driven execution). Awaiting the five rulings in its §8.

## UNRESOLVED (mandatory)

Each item has its register id. WHY it is open · what was tried · shortcomings.

- **Real-workflow run result (T-010)** — the agent dispatched 2026-09-06 15:40 to run the SMS export through the live proffer workflow; its seven proof queries appear in no later doc and no session log. `docs/reviews/2026-09-06-ingest-session-digest.md` may hold it; not read in full.
- **Decode-subtree-first (T-011)** — not started. `modules/engine/decode/` absent; replace directive targets the SBV fork. D-149 item 10 still names it the Stage 3 head.
- **Infra identifier rulings R-1..R-18 (T-012)** — unruled in DECISION_LOG and in the logs. R-8 (`agno_app` to `probata_app`) is moot because D-152 deleted the role; the owner ordered hard cuts for R-1/R-2 on 2026-09-07; R-14 retire of the Graphiti apps is effectively done (both exited), the Coolify records remain.
- **Permission deny-list never restored (T-013)** — confirmed open in the logs; lives in `.claude/settings.local.json` (local, gitignored); re-check the live file before acting.
- **R2 fixture prefix (T-014)** — the 3.66 MiB sample copy is DONE per session log; the delete ruling on the 1.24 GiB of real exports under the test-fixtures key is still unknown. Verifying needs a billable rclone listing; owner sign-off first.
- **Golden-clone template (T-015)** — D-142 §3 asked for a flag-gated `CREATE DATABASE … TEMPLATE` mechanism; what landed is `scripts/rebuild_platform_from_snapshot.sh`, run once by hand 2026-09-07. The need is met; the reusable, flag-gated form is not built.
- **Devbox v2 and `opencode web` (T-016, T-017)** — confirmed open in the logs; needs current Coolify state, not a doc check.
- **2026-09-08 infra queue (T-018)** — 15 queued items (Filestash, observability stack, Portkey self-host, ContextForge upgrade and federation of 10 local MCP tools into an empty ContextForge, Cloudflare Tunnel ingress, embed-model EOL swap, and more). Filestash confirmed still open in the logs; the ContextForge PG role and database grant was ruled by the owner but execution is unconfirmed.
- **Evidence lane cannot ingest PDF, DOCX, PPTX, XLSX, HTML (T-003)** — `server/proffer/service.py:251` skips `_extract_document` for the evidence lane; the extraction extras are optional dependencies in `pyproject.toml`, not installed. Gated behind D-151c.
- **NUL bytes in raw records (T-005)** — a live parse failed on `0x00`; no substitution strategy ruled.
- **n8n to worker error contract (T-006)** — parser 4xx surfaces to Temporal as an opaque `decode n8n StageResult: EOF`.
- **Two Weaviate instances (T-007)** — both still running per Coolify; the owner ordered a fix on the stray `/data/agno/volumes/weaviate` path on 2026-09-07 but did not name the canonical instance.
- **Postgres persistence redeploy** — `duckdb.allow_community_extensions=on` is in the Dockerfile CMD; takes effect on the next `agentos-db` redeploy, owner-triggered. UNVERIFIED triggered.
- **From the 127 KB derived-document handoff, still open and never contradicted:** `context_thread_id` has a schema (26 references) but no verified producer; `timeline.event_candidate` has no verified bridge from `working.candidate_event` and no HTTP or Temporal surface; trigram extension is enabled in the schema but search code is not confirmed to use it; OCR and office-format extras unbuilt.
- **From the 2026-08-24 handoffs, orphaned open items with no ruling anywhere:** TODO-213 disclosure-tier naming; TODO-210 bitemporal graph; TODO-211 agent-memory bake-off (now superseded in spirit by the 2026-09-09 ruling: one shared SurrealDB agent memory on the VPS); whether the 5 composed n8n workflows were ever imported. **n8n public API key expires 2026-09-22.**
- **Side tracks surfaced only in compact summaries:** Case Bible JSON extraction (about 885k records landed by 2026-09-08); memsearch global-versus-project collection rename got owner pushback in the logs and no ruling.
- **Two DONE-NOT-CLOSED handoffs to mark in place, not move:** `HANDOFF-2026-08-29-agno-role-dissection.md` became D-101/D-107/D-108 and never says so; `HANDOFFS.md` R0 still says "migrations held".

## Pending owner decisions

- **Taxonomy (§8 of the proposal)** — seven types, domain list, `awaiting-verification/` dissolves into status `unverified`, hooks repointed, execution via ingest. WHY: every other decision below lands in a structure that either holds or leaks.
- **Stand up the local SurrealDB** — binary plus `surrealkv://` under a path you name, bound to localhost, MCP served natively; and which VPS instance hosts the shared agent memory (`surreal-case` is the live candidate). Waiting on the expert brief before recommending the exact commands.
- **Dispose of `docs/awaiting-verification/` (T-002, 97 files)** — folds into the taxonomy ruling if status `unverified` is accepted.
- **Evidence-release questions R1–R6 (T-004)**, **identifier renames R-1..R-18 (T-012)**, **Aperture vs Portkey (T-019; owner leaned Portkey on 2026-09-07, no closing ruling)**, **retire `CHANGE-ORDER.md` (T-057)**, **credential-shaped literals in tracked docs (T-001)**.

## Next steps (work in order)

1. Owner rules on the taxonomy proposal §8 and deletes the replaced TODO documents listed at the end of the register.
2. Apply the in-place fixes that need no ruling: 21 path-prefix dead links in `HANDOFFS.md` and `INDEX.md`; ADR supersession banners (Graphiti five, ADR-0027, AgentOS set); the DONE-NOT-CLOSED markers on the two handoffs.
3. Stand up the local docstore per the expert brief; run the docstore ingest with the mapping table; regenerate the mirror; re-run the dead-link gate. _2026-09-09 07:22 EDT update: store live at `127.0.0.1:8462` (RocksDB under `probata/.docstore/`), schema and 17 functions applied, MCP `run` proven with the documented `$ql` sentinel for skipped optionals; ingest run 2 in progress after run 1 crashed at 332/503 on an escaping bug now fixed; mirror `--apply` follows._
4. Chase T-010 and T-011 to pin ingest state, then resume decode-subtree-first per D-149 item 10.
5. Owner triggers the `agentos-db` redeploy so the DuckDB extension setting persists.

## Owner working-style contract

- Structured replies: bullets, labeled blocks, white space, answer-first. Restate the order before acting on it.
- Confirm before changes; never hard-delete (quarantine); byline every artifact; verify before claiming done. Consolidate in place: new version, same directory, same format, stale content dropped. Never an archive or "consolidated" folder. Agents run on Sonnet or below, Opus at most, never the frontier model; every dispatch names its model; search with `ccc` and DuckDB before grep.
