# Agno CocoIndex notes (territory)

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_
> Queried with `ccc search` from `Agno-MCP-Platform/` (already indexed).

## What Legal Workspace may consume

Promotion is first-class in Agno today:

- `server/case_management/repository.py` `promote_evidence`
- `server/contracts/case_management.py` `PromotionSourcePointer`
  (sha256, quote, chunk_ref, retrieval_ref)
- Inserts go to `analysis.knowledge_evidence_promotion`
- Fresh promotions land `review_status=unreviewed`,
  `hitl_required=true`, `safe_for_legal_use=false`,
  `is_authenticated=false` (`scripts/_matter_validate_0030.py`)
- DB rejects promoting a non-`evidence` knowledge lane

**Implication for `LegalSourcePackage`:** only items with
`safe_for_legal_use` (or equivalent approved review) may be imported.
The Phase 0 gate that drops `candidate` / `revoked` matches this
territory.

Exact quote must already exist in the source content
(`promote_evidence` checks `quote not in source_row["content"]`).
Legal citations should keep that exact span.

## Custody verify (indexed)

`POST /v1/verify/{sha256}` in `server/api/inspect_routes.py`
re-reads the blob, compares H1, and returns
`sha256_match` / `verdict` (`hash-only-ok` | `intact` | `broken`).
Cat 5 metadata reports should call this, not re-hash blindly.

## What does not exist in Agno (confirmed by handoffs + index)

- No LegalSourcePackage type yet — Legal Workspace defines it
- No Michigan motion/strategy personas under `knowledge/legal/`
- Part 3 (AI Legal Team) is documented as “to build” in Agno canon
  and is owned here

## How to search next time

```
Set-Location Agno-MCP-Platform
ccc search --lang python "your concept"
```

Do not `ccc init` inside Legal-desktop; that index belongs to Agno.
