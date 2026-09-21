---
tags: [docstore, propria, receipt, contextforge]
---

# Docstore 0.8.1-r1 — what was broken in the five-tool design, and the fixes

> _Byline: Claude Code · Fable 5.1 · 2026-09-20. Owner 22:09: "the new design is right but broken and missing a couple — fix it"._

The 0.8.1 design (Codex, 2026-09-20) is sound: five public tools (`health`, `capabilities`, `query`, `search`, `get`), all 57 operations reachable through `docstore_query`, writes only with `mode=write`. Read from the live server and from `public_surface.py`.

| # | Defect | Cause | Fix |
|---|---|---|---|
| 1 | Discovery uncallable: `docstore_capabilities(group=…)` / `(operation=…)` rejected | ContextForge stored the tool's schema at 15:11 UTC, before the container was rebuilt at 16:09; its copy had NO parameters while the live server advertises `group`, `operation` | `scripts/contextforge_refresh_gateway.py ctl08 --apply` (`POST /gateways/<id>/tools/refresh` → 200); stored schema now `["group","operation"]` |
| 2 | Container permanently `unhealthy`, `/health ok:false` | `api.py` treats every `degraded` sync as not ok, including "Source attribution verified; provider enrichment remains pending" — 2 of 694 documents failing an optional LLM call | `apply_health_fix.py` (12 lines, in place, backup kept): ok stays true when enrichment is the ONLY degradation and attribution is verified; `enrichment_pending` count added |

Not changed: the enrichment code. Reproduced by hand for `docs/adr/generated/0016.md`: the provider (`nvidia/nemotron-3.5-lightning-30b-a3b` on NIM) hit `httpx.ReadTimeout` at 120 s — a provider fault, retried by later runs. Not changed: the old `docstore-control` app (`ctl`, :8172, 57 individual tools, empty API token, dead worker URL) — it is the pre-0.8.1 surface; Codex's 0.5.4 client still calls it, so the remaining work is the client move to the five-tool surface (Codex package `propria-docstore-0.8.2-claude-upload.zip`), not a server change.

`api.py.before` = the server file as found (sha256 prefix `13f37725d2c72e64`, identical in the image, the release dir and Codex's zip). The release directory `/data/propria/releases/docstore-0.8.1/` is not in any git repository.
