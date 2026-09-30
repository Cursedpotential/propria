---
tags: [docstore, propria, receipt, memory, duplicate-guard]
---

# Docstore 0.8.1-r6 — the memory duplicate guard stops refusing unrelated claims

> _Byline: Claude Code · Opus 5.5 · 2026-09-28. Subagent `docstore-memory-fix`, follow-up from the parent session "portal cut over". Follows `../2026-09-27-docstore-0.8.1-r5/README.md`._

**Result:**
- An unrelated claim is written.
- A reworded duplicate is still refused, and the refusal shows why: distance and word overlap.

## What went wrong

The r5 guard treated any active row within cosine distance 0.20 as a near-duplicate. At 04:44 EDT it refused a new, unrelated claim ("when a permission classifier blocks an owner-approved change, give the owner one command in a code block", stored with `force:true` as `memory:vsp0cqtvnnvdzgwxu6i5`) as a duplicate of four unrelated memories:

| Refused as a duplicate of | Distance |
|---|---|
| memsearch watcher | 0.174 |
| credential authority | 0.174 |
| guard hook | 0.180 |
| Claude app workspace folders | 0.190 |

## Measurement (live store, 2026-09-28, `nvidia/nemotron-3-embed-1b`)

| Set | Cosine distance | Word Jaccard |
|---|---|---|
| All 120 pairs of the 16 active embedded rows (all unrelated) | min 0.174, 5th pct 0.216, median 0.295 | max 0.26 |
| Reworded duplicates: owner-rule paraphrase, container paraphrase, subagent paraphrase | 0.184, 0.061, 0.113 | 0.56, 0.54, 0.44 |

The distance ranges overlap: an unrelated pair sits at 0.174 and a duplicate at 0.184. Word overlap does not overlap. Words are lowercased, split on anything but `[a-z0-9]`, and only words over 2 characters are kept.

## New rule

Migration `scripts/docstore/schema/2026-09-28-memory-duplicate-guard-lexical.surql`, adding `fn::claim_words` and redefining `fn::remember`. An active row in the same scope conflicts if:
- BM25 finds every word of the claim in it (unchanged), or
- cosine distance **≤ 0.10** (same meaning in any words; well under the unrelated floor of 0.174), or
- cosine distance **≤ 0.20 and word Jaccard ≥ 0.35**.

The thresholds are calibrated on a small store; revisit them as it grows. The 409 detail now lists `overlap` next to `dist`.

## Deployment

- **Migration:** first run inside a transaction that was cancelled, with both cases checked, then applied live: `has_errors: false`.
- **Release tree:** `apply_r6.py` applied to `/data/propria/releases/docstore-0.8.1` (backups `*.bak-20260928T085157Z-r6`). It installs `remote_memory.py` (409 carries `overlap`), `release_tools.py` (schema text), both memory migrations and the new test.
- **Image:** `propria-docstore:0.8.1-r6` (`sha256:9f177d8c…`): **339 passed, 3 skipped** (r5: 334). New `tests/test_memory_duplicate_guard.py` runs the real `fn::remember` on embedded SurrealDB:
  - an unrelated claim at 0.17 is written;
  - a reworded claim at 0.18 with overlap ≥ 0.35 is refused;
  - different words at 0.08 are refused;
  - `force` keeps both;
  - the tokenizer is checked.
- **Coolify:** service `o8obobz576je1fbyygnywl83`, image tag patched via the API, then `POST /api/v1/deploy` (200). The container has run image `9f177d8c…` since 08:52:51 UTC.

## Live proof

- **Paraphrase of the owner rule, through ctl:** `Docstore refused the request as a conflict (HTTP 409)`. It names `memory:z29uynwp9gdpj34m607t` with `dist 0.184`, `overlap 0.559`.
- **The 04:44 claim, replayed against the committed live function:**
  - Setup: its own row set aside, claim suffixed, inside a cancelled transaction.
  - Result: `written: true`, `conflicts: []`.
  - Its four former false positives measured 0.174 / 0.174 / 0.180 / 0.190, with word overlap 0.05–0.10.
  - After the rollback: 21 rows, `vsp0` still `active`, no `decision_log` rows.
- **No test rows written.**

## Files

| File | What |
|---|---|
| `apply_r6.py` | Whole-file installs checked against the r5 hashes; idempotent; dry run by default. |
| `test_memory_duplicate_guard.py` | Release test, installed as `tests/test_memory_duplicate_guard.py`. |
