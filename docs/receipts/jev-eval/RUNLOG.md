# Jev Tier-1 eval: run log

> _Byline: Claude Code · Opus 5.5 · 2026-09-23. Handoff: `docs/handoffs/HANDOFF-2026-09-23-jev-tier1-eval.md`. Facts: `docs/handoffs/JEV-EVAL-PREP-2026-09-23.md`._
> Work dir with case data: **ovh-files `/data/probata/jev-eval/`** (owner 14:24: run on the server). Message text and raw responses stay there and never go into git. Code: `scripts/jev_eval/`.

| When (EDT) | Step | Result | Evidence |
|---|---|---|---|
| 14:15 | Opus auth | Long-lived `CLAUDE_CODE_OAUTH_TOKEN` via Claude Agent SDK; smoke `claude-opus-5-5` → `TOKEN_OK` | desktop `claude -p` run |
| 14:20–14:23 | Katrina pool | FB 67,377 · SMS 2021–22 (810-295-9303, hers) · 2024 her phone (conv 8102959302) · SMS 2025–26 (810-353-3592); 869 dropped | prep report addendum |
| 14:26 | Sample v1 | 300 rows; 5 Facebook system notices got in | `sample/sample_v1.jsonl` sha256 `f5e9e04b…0d35`; sql `db8b38dc…1a` |
| 14:45 | Sample v2 (current) | 300 rows, 291 unique texts; system notices + tapbacks excluded; 295/300 same as v1; every cell full; source_sha1 271/300 (29 missing in catalog provenance; SHA-256 not in catalog) | `sample/sample_v2.jsonl` sha256 `a07ddc18…6604`; sql `build_sample_v2.sql` sha256 `f936e1ae…2f19` |
| 14:29 | Phase 0 OpenRouter smoke (synthetic state) | HTTP 200, 752 ms; noul + choice shapes match handoff §1; served model `typesafe/jev-1.13-20260917`; cost $0.00002247 | `raw/phase0/smoke_20260923T182933Z.json` |
| 14:33 | Runtime | Reused the ovh-files **devbox** (Claude Code 2.1.263, Python 3.12, Node 22; owner 14:33) instead of a new image; work dir moved to devbox `persist/jev-eval` (host `/data/probata/volumes/devbox/home/jev-eval`); venv with claude-agent-sdk 0.2.158 (bundled CLI), pyarrow 25, duckdb 1.5.5; token passed at run time with `--env-file` from `/data/probata/secrets/jev-eval/claude.env` | `Propria/docs/reference/DEVBOX-ON-OVH-FILES.md` |
| 14:37–14:47 | **Phase 3 Opus labels** | 300/300 labelled + 30/30 repeat, 0 errors, 0 structural problems; `claude-opus-5-5`, prompt `a0dbb42296d7`, ~6–8 s/message at 4 in parallel. Claude Code also makes small Haiku housekeeping calls; every label is Opus. True counts: case_relevant 80, hostility 39, blame_shift 36, third_party 35, child_referenced 34, disparagement 29, financial 25, logistics 23, reframes 21, admission 13, wellbeing 10, legal 4, cooperation 4, **parenting_time_denial 0, info_gatekeeping 0**; 160 with no tag. Self-consistency on the repeat: 476/480 decisions identical (register 2, third_party 1, logistics 1) | `labels_opus.jsonl` sha256 `e0fe705c…74d9`; `labels_opus_repeat.jsonl` `56258c7e…c4`; `raw/opus/`, `raw/opus_repeat/` |
| 14:48 | Opus-label report v1 | For owner review before any Jev run | `reports/opus_labels_report_v1.html` sha256 `27d0de93…0249` |
| 15:15 | Owner review page | Owner 15:06: case_relevant "wrong entirely on every account"; asked for a page where he can type responses. Published a private claude.ai artifact **Opus Label Review** (`https://claude.ai/artifact/CWoTSa3hsUU4XrqQzmU3CD`, capability `db`). The owner flips labels, sets the register, comments, and writes tag definitions. Answers land in the artifact db (`reviews/m001..m300`, `notes/definitions`), read back with read_db | `scripts/jev_eval/review_page.py` |

## Open checkpoints
- [x] Owner approved the Phase 2 sample and said go for Phase 3 (14:32).
- [x] Phase 3 done; report v1 delivered 14:50.
- [ ] **Owner reviews the Opus-label report** before any Jev comparison (owner 14:17).
- [ ] Owner: 2 tags have zero positives in the random sample (parenting_time_denial, info_gatekeeping), so Jev cannot be scored on them. Proposed: an enriched add-on set for the rare tags, labelled by Opus and scored separately.
- [ ] Owner: 9 messages are `case_relevant` only because of their context ("Nope" answering a child-care question). Should the gate count context, or target words only?

## Known deviations from the handoff
- Opus runs through the Agent SDK, which has no temperature setting. Proposed substitute: re-label a random 10% to measure self-consistency.
- `source_sha256` is not in the catalog; `source_sha1` is carried instead (271/300) and the gap is reported, never filled in.
- TypeSafe direct is unavailable (sign-ups paused): cells A/B only.
