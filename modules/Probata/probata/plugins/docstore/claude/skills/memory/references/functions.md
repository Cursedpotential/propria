# memory — function reference

> _Re-conformed to the live store 2026-09-27 (Claude Code · Opus 5.5)._

Agents write and recall through ctl (`docstore_memory_remember`,
`docstore_memory_recall`; see `../SKILL.md`), never by calling these
functions directly. This page documents what the server runs.

Source: `080_memory.surql` / `085_memory_functions.surql` /
`087_memory_access.surql` (dated snapshot applied 2026-09-16 under
`probata/scripts/docstore/schema/applied-2026-09-16-memory/`), then the
dated migrations `schema/2026-09-19-memory-root-propria.surql` (scope root
`propria`) and `schema/2026-09-27-memory-remember-guard.surql` (duplicate
cutoff, working supersession). Deployed and round-trip verified live
2026-09-16; the 2026-09-27 write path verified live through ctl. Target store: `surreal-case` on
the VPS, reached via the `memory` MCP server
(`${MEMORY_MCP_URL:-http://100.91.190.107:8471/mcp}`), **namespace
`probata_memory`, database `memory`** — the `.mcp.json` entry must send
`surreal-ns`/`surreal-db` headers with those values (added 2026-09-16;
their absence produced "Specify a namespace to use" on every call).

## fn::remember

```
fn::remember($payload: object) -> object
  -- refused:    { written: NONE, conflicts: array<object>, note: string }
  -- ok:         { written: record<memory>, conflicts: [] }
  -- supersede:  { written: record<memory>, superseded: record<memory>, conflicts: [] }
```

`$payload` fields: `kind` (ASSERT `["correction","preference","observation",
"handoff","fact","constraint","decision"]`), `claim` (10–600 chars,
ASSERT'd), `detail?`, `evidence?` (a plain string — doc id + source path,
**never** a live cross-store record link), `scope`, `agent`, `confidence?`
(default 0.6), `observed_at?`, `embedding?`, `force?`, `supersede?`,
`reason?`.

Runs a BM25 (`claim @@ $claim`) + vector conflict check against **active**
claims in the same `scope` before writing. The vector leg takes the 5
nearest rows (`<|5,40|>`), computes cosine distance
(`1 - vector::similarity::cosine`) and word overlap (Jaccard over
`fn::claim_words`: lowercase, split on non-alphanumerics, words over 2
characters). It keeps a row if **dist ≤ 0.10**, or **dist ≤ 0.20 and
overlap ≥ 0.35** (migration `schema/2026-09-28-memory-duplicate-guard-lexical.surql`).
History:
- before 2026-09-27 there was no cutoff, so every write into a non-empty
  scope was refused;
- the 2026-09-27 cosine-only 0.20 cutoff still refused unrelated claims
  at 0.174–0.190. If conflicts exist and neither `force:true` nor
`supersede:<id>` is set, the write is refused with the conflicting rows
attached. The function returns this softly; the API turns it into HTTP 409.
`supersede` may be a string (`"memory:abc"`); it is cast with
`type::record`. The API embeds `claim` server-side and adds `embedding`.

## fn::supersede_memory

```
fn::supersede_memory($old: record<memory>, $new_payload: object, $reason: option<string>) -> object
  -- { written: record<memory>, superseded: record<memory>, conflicts: [] }
```

THROWs if `$old` does not exist or is not `active`. Creates the new row,
`RELATE new->supersedes->old`, flips `old.status` to `"superseded"`, and
writes a `memory_superseded` `decision_log` row carrying `$reason` (the
status EVENT writes its own reason-less row too). Until 2026-09-27 it took
two arguments while `fn::remember` passed three, so every supersede failed.
The new claim text must differ from the old one: `UNIQUE(scope, claim)`
covers every status.

## fn::forget

```
fn::forget($id: record<memory>, $reason: string) -> object   -- the updated memory row
```

Sets `status: "retracted"`. Writes **two** `decision_log` rows for one call
by design: the `memory_status_changed` EVENT fires (reason-less audit row)
and `fn::forget` itself writes a second, reason-bearing row — this is
intentional redundancy in the append-only audit trail, not a bug.

## fn::recall

```
fn::recall($query: string, $vec: option<array<float>>, $scope: string, $k: int) -> array<object>
  -- {id, claim, kind, confidence, observed_at, evidence, status, rrf_score}
```

RRF fusion (k=60) over BM25 (`claim @1@ $query`, 20 candidates) and KNN
(`<|20,40|>` literal, only if `$vec` given) — matches rows whose `scope`
equals `$scope` OR starts with `$scope + "/"` (prefix-descendant match).
Truncated to `$k` after fusion, `status = "active"` only.

## fn::reflect

```
fn::reflect($scope: string, $since: datetime) -> array<object>   -- the episode rows selected
```

Selects unreflected `episode` rows for `$scope` (+ descendants) since
`$since` and marks them `reflected = true`. Does **not** write memory
itself — condensing episodes into `fn::remember` calls is the caller's job.

## fn::memory_stats

```
fn::memory_stats($scope: string) -> object
  -- { scope, total, active, superseded, retracted, by_kind: array<{kind,n}>, episodes_pending_reflection }
```

Point-in-time counts for `$scope` and its descendants.

## Scope

Every `memory`/`episode` row's `scope` must match `^propria(/[a-z0-9_-]+)*$`
— path form `propria/<module>/<agent>`, e.g. `propria/intake` (root moved
from `probata` on 2026-09-19).
A `principal` row (agent identity) carries a `scope_prefix` grant; the
prefix itself is enforced inside the `fn::` functions' `WHERE` clauses, not
by `DEFINE ACCESS` alone.

## Gotchas

1. **Raw MCP calls only:** typed optional/datetime args need the `$ql`
   sentinel, same as record ids. (ctl callers pass plain JSON; the API
   handles it.) `fn::recall`'s `$vec` (`option<array<float>>`) and `fn::reflect`'s
   `$since` (`datetime`) both fail to coerce from bare JSON (`null`, a plain
   ISO string) over MCP `run` — reproduced live 2026-09-16. Pass
   `{"$ql": "NONE"}` for no vector and `{"$ql": "d'2026-01-01T00:00:00Z'"}`
   for a datetime.
2. **K/EF are literal** (`<|5,40|>`, `<|20,40|>`) — same constraint as the
   docs store, never pass `$k` into the KNN operator itself.
3. **`evidence` is a plain string, never a live record link** — memory rows
   citing a docs-store id must re-verify with `fn::docs_get` on the docs
   server; a stale doc turns the memory claim into something the
   `reconcile` skill should flag, not something to trust blindly.
4. No `fn::` deletes anything, ever. `forget` retracts; `supersede_memory`
   supersedes. (A hard `DELETE` is still possible via the generic `query`
   tool for genuinely disposable test data — that is how the 2026-09-16
   verification pass purged its QA rows — but no `fn::` function does this,
   and normal agent use should never reach for it.)
5. **The MCP session must select `probata_memory`/`memory`.** The server
   also hosts unrelated `fct`/`main` namespaces for other apps on the same
   VPS instance; a client without the `surreal-ns`/`surreal-db` headers (or
   without an explicit `use` first) gets "Specify a namespace to use", not
   an empty result — it is not silently reading the right store.
