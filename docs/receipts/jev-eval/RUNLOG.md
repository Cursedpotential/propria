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

## Open checkpoints
- [ ] Owner approves the Phase 2 sample (spread table + 10 random rows).
- [ ] Owner go for Phase 3: Opus reference labels on the 300 messages, run on the long-lived token, then the Opus-label report to the owner **before** any Jev comparison.

## Known deviations from the handoff
- Opus runs through the Agent SDK, which has no temperature setting. Proposed substitute: re-label a random 10% to measure self-consistency.
- `source_sha256` is not in the catalog; `source_sha1` is carried instead (271/300) and the gap is reported, never filled in.
- TypeSafe direct is unavailable (sign-ups paused): cells A/B only.
