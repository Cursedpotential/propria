# Documents taxonomy simplification — systems, graph, Socratic, and inversion passes (2026-09-09)

> _Byline: Claude Code · Fable 5.1 · 2026-09-09 06:05 EDT_
> _Owner directive 05:53: "We don't need all of those categories. Blueprint, infrastructure, a set of to-dos, handoffs and compacts in one folder, reviews, maybe references — and categorized by the domains they serve now that there's a database. Use the thinking processes." This is the proposal to rule on. Nothing has been moved._

**STATUS: RULED 2026-09-09 06:27 (owner: "sound good") — logged as D-156. All five §8 items accepted; execution in progress.**

## 0 · What we are simplifying (measured today)

| Today | Count |
|---|---|
| Top-level directories under `docs/` (excluding `private/`) | 21 |
| Files at `docs/` root | 45 |
| Text documents outside `private/` | 493 (1.18 M words; median 1,261 words; largest 61,884) |
| Directories that are really a *status*, not a type (`awaiting-verification/`, `consolidated/`, `recovered/`, the former `pending-review/`, `CLAIMED_COMPLETE_LIKELY_LIES/`) | 5 |
| Directory pairs that are the same type twice (`plans/`+`planning/`, `schema/`+`schemas/`, `design/`+`blueprint/`, `COMPACT-SUMMARY/`+`handoffs/`) | 4 |
| Dead intra-docs links / orphan files (2026-09-09 link walk) | 87 / 46 |
| Byte-level duplication (MinHash census, 493 files, threshold 0.6) | 1 near-duplicate pair; 0 exact; 0 shared boilerplate blocks |

The census result is the important one: **the corpus is not textually duplicated, it is semantically stale.** The same open item is restated in six documents in six wordings. Dedupe tooling cannot see that; only a `supersedes` relationship and a `status` field can. That is why the fix is a schema, not a folder move.

## 1 · Systems pass — the loop, the archetype, the leverage point

The 2026-09-05 audit named the archetype correctly (Shifting the Burden with a Fixes-that-Fail inner loop) and then fed it by adding `docs/consolidated/`. The generator is unchanged:

```
new decision -> new doc written (fast, rewarded)
             -> old doc left in place (retiring is slow, unrewarded, might break a link)
             -> reader cannot tell current from stale
             -> symptomatic fix: a new FOLDER or INDEX that says which is current
             -> the folder/index is itself one more doc -> loop
```

Three machine generators also feed it without any human deciding to: the PreCompact hook writes `TODO-SNAPSHOT-<date>.json`, the PostCompact hook writes `COMPACT-SUMMARY-<date>.md` (five of ten are empty payload dumps), and every session that invents a folder name creates a new category. **21 directories is the count of workarounds, not the count of document kinds.**

Leverage point (Meadows levels 4–5, the rules of the system and its information flows): make *type*, *domain*, and *status* first-class fields that every document must carry and that the store refuses without, and make *supersession* the definition of done for any new version. Folders stop being the classification; they become a mirror of `type`. The generators must write into that schema or be disabled.

## 2 · Graph pass — docs are nodes, folders were the only edges

Today the only relationship the tree encodes is *membership in a folder*. Everything a reader actually needs is an edge the tree cannot hold: **supersedes** (which version is current), **decides** (which ruling closed it), **implements** (which code landed it), **cites** (87 of these are broken). With a database underneath, the folder question dissolves: a document is a node with a `type`, a `domain` set, a `status`, and edges. "Which handoff is current for ingest?" is a one-hop query, not a directory listing.

Consequence for the taxonomy: **types must be few and orthogonal to domains.** A document has exactly one type (what kind of thing it is) and one or more domains (what part of the platform it serves). Mixing them, as `forensic-db-reconciliation/` (a domain used as a folder) or `awaiting-verification/` (a status used as a folder) did, is what produced 21 directories.

## 3 · Socratic pass — what is each document FOR, and does the type survive?

| Question | Answer | Surviving type |
|---|---|---|
| What is this application and how is it structured? | Canon: product invariant, naming, repo structure, conventions, architecture, data model, design | **blueprint** |
| What is it running on, where, and how do I operate it? | Infrastructure facts, runbooks, deploy contracts | **infrastructure** |
| What was irreversibly decided, and why? | ADRs and the decision log. Never archived; superseded in place with a banner | **decision** (the owner's list omits it; it cannot be folded into blueprint because blueprint is amended freely and decisions are append-only) |
| What is still open? | One register. Not five. | **todo** |
| What state did the last session leave, so the next can resume? | Handoffs and compact summaries are the same kind of thing at different granularities | **handoff** (compacts fold in) |
| What was observed at a moment: an audit, a code review, a receipt, a pre-mortem? | Photographs. Evidence, never authority | **review** |
| What stable fact do I look up: a glossary, a schema, a parser list, an external API note, raw research? | Lookup material | **reference** |

Everything else is a *status* or a *domain*, not a type: `awaiting-verification` = status `unverified`; `consolidated` = a review; `recovered` = status `unverified`; `plans/` and `planning/` = blueprint with status `proposed` (a plan is a blueprint of a future state; when it lands it becomes canon, when its premise dies it becomes superseded); `registers/` = todo (`DEBT`, `GUARD-TRIGGER`) or reference (`SETTLED` is a lookup index); `schema/`, `schemas/`, `research/`, `visualizations/` = reference; `runbooks/` = infrastructure; `design/` = blueprint.

Seven types. The owner asked for about six; the seventh, **decision**, is the one the platform's own rules forbid retiring, so it stays as its own kind.

## 4 · Domains — already ruled, just not applied to documents

The naming canon (D-137..D-142, `docs/NAMING.md`) already defines the platform's components. Those are the domains. No new list is needed:

| Domain tag | Covers |
|---|---|
| `probata` | the platform as a whole: canon, conventions, cross-cutting |
| `proffer` | ingest lane: parsers, decoders, ELT, storage tiers, chunks |
| `consignatio` | vault / case bible: sorted sources, custody at promotion |
| `advocatio` | legal workbench |
| `vestigia` | geo |
| `indagatio` | analysis engine (Surreal analytical surface) |
| `intake` | desktop ingest client (SBV) |
| `workbench` | the operator UI |
| `knowledge` | retrieval, search, vectors (Weaviate) |
| `memory` | the agent memory system (VPS) and recall tooling |
| `infra` | hosts, Coolify, Traefik, tailnet, backups |
| `docs` | the documentation system itself |

A document carries one or more. `tags` stays free-form for anything finer.

## 5 · Status — the field that makes retirement real

Align with the docstore schema's ASSERT list, extended by one value the corpus needs:

`active` · `proposed` (a blueprint not yet landed) · `unverified` (claims not proven live; replaces the purgatory folder) · `superseded` (points at its successor) · `retracted` (wrong; kept for provenance)

Rule: a new version of any document is written **in the same type, same domain**, and the store writes `new -supersedes-> old` and flips the old to `superseded`. That is the owner's in-place rule made mechanical. No archive folder exists because `status = superseded` *is* the archive.

## 6 · Inversion / pre-mortem — it is 2026-10-01 and this made things worse. Why?

1. **The hooks kept writing to the old paths.** `TODO-SNAPSHOT-<date>.json` and `COMPACT-SUMMARY-<date>.md` reappeared at `docs/` root within a day. Fix: the same change that adopts the taxonomy repoints or disables both hooks. Non-negotiable.
2. **Moving 493 files broke every relative link again.** Fix: do not move files by hand. Ingest them into the store with `source_path` recorded, let the pipeline assign type, domain, and status from a mapping table, then regenerate the on-disk mirror. Run the dead-link walk as a gate before and after; the count may not rise.
3. **An eighth type appeared.** Fix: the schema ASSERT list rejects it loudly at write time. That is the docstore's design and the reason to keep it.
4. **Status was never maintained.** Fix: the librarian agent refuses to register a new version without a supersession target when one exists, and the retrieval function returns `status` on every hit so a stale document cannot be mistaken for current.
5. **Domains were mis-tagged in bulk.** Fix: derive the initial tag from the current path plus the NAMING glossary, then have the reconciler show the owner the per-domain counts before committing; a domain holding under 1 percent of chunks is the known silent-empty-recall trap.
6. **Decisions got "simplified" into blueprint and lost their append-only guarantee.** Fix: keep `decision` as its own type. ADRs and DECISION_LOG are the only documents whose history is the product.

## 7 · Proposed mapping (counts from today's tree; the per-file mapping is the execution artifact)

| Today | Becomes | Notes |
|---|---|---|
| `PROJECT_CANON`, `NAMING`, `CONVENTIONS`, `REPO_STRUCTURE`, `INDEX`, `MEMORY_ARCHITECTURE`, `BUILD_PLAN`, `blueprint/`, `design/` (18 files + 8 root) | blueprint | `INDEX.md` becomes a generated view of the store, not a hand-maintained file |
| `INFRASTRUCTURE.md`, `runbooks/` (3), deploy contracts | infrastructure | the template twin is the census's one near-duplicate; drop it |
| `adr/` (63), `DECISION_LOG.md` | decision | banners added in place: the five Graphiti ADRs, ADR-0027, the AgentOS set |
| `MASTER-TODO-2026-09-09.md` | todo | the only one; the 14 replaced files are deleted by the owner |
| `handoffs/` (18) + `COMPACT-SUMMARY/` (9) + the root compact + `COORDINATION.md` | handoff | one folder; one current handoff per domain; older ones `superseded` |
| `reviews/` (150), `consolidated/` (2), the reviews and summaries inside `awaiting-verification/` | review | 46 history-only, 76 carry open work already extracted, 20 stale-claim become `superseded` |
| `reference/`, `schemas/`, `schema/`, `research/` (31), `glossary.md`, `registers/SETTLED.md`, `visualizations/` | reference | `schema/catalog.json` regenerated, not audited |
| `plans/` (26), `planning/` (69) | blueprint, status `proposed` or `superseded` per the planning walk's LANDED and PREMISE-DEAD verdicts | the 50-file forensic cluster is one blueprint with domain `consignatio`, status active |
| The R0–R14 packets in `awaiting-verification/`, `recovered/`, the `.bak` copies, the TODO-SNAPSHOT JSON files, `wiki.xxh3`, `semantica`, `semantica-benchmarks` | not ingested; owner deletes | hook output and byte-junk; nothing in them survives that is not already in the register |

On disk after the change: `docs/blueprint`, `docs/infrastructure`, `docs/decision`, `docs/todo`, `docs/handoff`, `docs/review`, `docs/reference`. Seven folders, zero root files except a generated `INDEX.md`.

## 8 · Owner decisions needed before execution

1. Confirm the seven types (your six plus `decision`).
2. Confirm the domain list is the naming canon plus `workbench`, `knowledge`, `memory`, `infra`, `docs`.
3. Confirm `awaiting-verification/` dissolves into status `unverified` rather than being ruled file by file.
4. Confirm the two hooks are repointed or disabled in the same change.
5. Confirm execution runs through the docstore ingest with a mapping table, not by hand-moving files.
