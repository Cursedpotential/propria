# Category 4 — Research tools

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_
> Source: `Legal-desktop/artifacts (3)/HANDOFF — Category 4  Research Tools.md`

## Hybrid search (question 7)

**Use in-database hybrid on the shared PG18 cluster:** `pgvector` +
`tsvector`/`ts_rank_cd`, fused with Reciprocal Rank Fusion.

Why not a new search engine: Agno already runs Weaviate for *evidence*
vectors. Legal Workspace’s internal store is this case’s drafts,
saved research, and authority snapshots — small, owner-only. A second
Weaviate collection would mix evidence embeddings with work product
unless isolated, and Cat 1 already put legal state on the same PG.

Concrete fusion (standard RRF, k=60):

```
score(d) = Σ 1 / (k + rank_vector(d)) + 1 / (k + rank_fts(d))
```

- Vector: embed query with the existing Portkey/NIM embedder
  (`nvidia/nv-embed-v1` or whatever Agno currently uses — same dim,
  never mix). Store on `legal_research.chunk.embedding`.
- Lexical: `to_tsvector('english', body)` + GIN. Michigan statute
  numbers (`722.23`, `MCR 3.215`) must remain exact tokens — add a
  simple synonym dict (`mcl` ↔ `michigan compiled laws`).
- Filter: `matter_id` always. Never search another Matter.

External sources (CourtListener, web) are **not** hybrid-indexed
here. They are connector calls returning the canonical `Case` /
`Statute` shape. Deep-research multi-hop lives in Category 3 and
only *calls* these functions.

Docs: [Postgres FTS](https://www.postgresql.org/docs/current/textsearch.html),
[pgvector](https://github.com/pgvector/pgvector),
RRF as used in hybrid retrieval literature.

## Source-config schema (questions 1, 3, 4)

Do **not** adopt Airbyte’s full CDK. Adopt a **subset** of its
manifest idea: auth, one or more HTTP endpoints, pagination, JSONPath
field map. Validate uploads with a JSON Schema
([jsonschema](https://python-jsonschema.readthedocs.io/)).

Field mapping: **JSONPath** via `jsonpath-ng` for list extraction
(`$.results[*]`), plus a constrained rename map for scalars. Full
Airbyte stream logic is overkill.

Worked CourtListener example (token stored separately):

```json
{
  "id": "courtlistener",
  "display_name": "CourtListener / RECAP",
  "may_cost_money": false,
  "auth": { "type": "api_key_header", "header": "Authorization", "prefix": "Token " },
  "endpoints": {
    "search": {
      "method": "GET",
      "url": "https://www.courtlistener.com/api/rest/v4/search/",
      "query": { "q": "{{query}}", "type": "o" }
    }
  },
  "map": {
    "list": "$.results",
    "Case": {
      "id": "$.cluster_id",
      "name": "$.caseName",
      "citation": "$.citation[0]",
      "court": "$.court",
      "date": "$.dateFiled",
      "url": "$.absolute_url"
    }
  }
}
```

CourtListener REST: [v4 search](https://www.courtlistener.com/help/api/rest/).

## Tokens (question 2)

Encrypt at rest in Postgres with `pgp_sym_encrypt` (`pgcrypto`);
key in env (`LEGAL_SOURCE_TOKEN_KEY`), never in the JSON file.
Match Agno’s “secrets in env / Coolify, never repo” pattern. Do not
reuse legal-mcp `integrations/config.py` as-is (plain env, fine for
one token, weak once the owner adds Midpage + Brave).

## eyecite (question 5)

Use [eyecite](https://github.com/freelawproject/eyecite)
([docs](https://freelawproject.github.io/eyecite/)). It extracts
reporter/volume/page/court/year. Map onto CITE tabs:

| Tab | eyecite role |
|---|---|
| Validate | parse succeeded + known reporter |
| Normalize | `eyecite.clean_text` / resolved citation string |
| Verify-integrity | **not eyecite** — pin `AuthoritySnapshot` + currentness |

legal-mcp’s `CitationParser` is regex and **must not** be ported.
eyecite does not Shepardize. CourtListener is not a citator
(custody-guide GUARDRAILS §3.1).

## Midpage (question 6)

Ship as a **generic add-a-source**, not a second built-in, unless
the owner later confirms a documented public API + Michigan coverage.
CourtListener stays the only baked-in default (free, no setup).

## Rate / cost (question 8)

Every source config has `may_cost_money: bool` and optional
`daily_budget`. UI copies legal-mcp’s PACER banner. PACER itself
stays **off** unless the owner opts in.

## Module tree

```
api/legal_workspace/integrations/research/
  schema.json
  registry.py
  connectors/http.py
  connectors/courtlistener.py
  hybrid.py
  eyecite_adapter.py
```

Quick-links (content retarget): MCL 722.23 (a)–(l), Child Custody
Act, MCR 3.215, MCR 2.119, Genesee LCR — from the custody packet
primaries, not legal-mcp’s generic statutes.

## Non-goals

No “search all of law.” No code change to add a source. No multi-hop
research in this category.
