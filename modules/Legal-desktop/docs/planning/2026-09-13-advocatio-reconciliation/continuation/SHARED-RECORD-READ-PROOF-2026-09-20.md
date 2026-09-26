# Shared-record read proof — result: not met

> _Byline: Claude Code · Fable 5.1 · 2026-09-20 22:55–23:10 EDT. Read-only; no writes to any store, no provider calls._

Acceptance test attempted (SINGLE-WORKDESK-CONVERGENCE.md, "Acceptance for integration", read half):
open one cheat sheet, one legal source and one case document from the toolkit and from Advocatio
and compare identifiers and versions.

## Result

The test cannot be run yet. There is no second surface to open, and the two sides share no record identity.

| Check | Observed |
|---|---|
| Hosted toolkit / mobile surface | None found. `modules/FL-MCP` is a local Tauri desktop app with a loopback-only sidecar (its README says "local desktop tool, not a hosted product"). `workbench.tilapia-skilift.ts.net` is the Probata Workbench ("The Platform — Evidence & Legal Operations"), not the toolkit. |
| Toolkit legal-source records | 193 in `family-court-toolkit/content/toolkit/ledger.json` (plugin 3.2.0), keyed by slug `id` (e.g. `caselaw-custody-standards`) with `content_hash`, `last_verified`. |
| Advocatio legal-source records | 12 from live `GET /api/legal/v1/authorities`, keyed by citation string `identifier` with `snapshot_hash` of the form `packet:M2:…`; `source_path` points at custody-guide packet files, not the ledger. |
| Identifier overlap | 0 of 12. By citation text, 1 of 12 (MCL 722.23) appears inside a ledger record; the other 11 (e.g. Vodvarka, Pierron, MCR 3.210(D)(1)) have no ledger counterpart. |
| Version comparability | None. `content_hash` (sha256 of fetched text) and `snapshot_hash` (packet label) are different constructions. |
| Cheat sheets in Advocatio | None. No "cheat sheet" reference in `api/`, `web/src` or `config/`; the toolkit has `cheatsheet-l0-hearing-day-card`, `cheatsheet-l1-pro-se-guide` and a `content/toolkit/cheatsheet` pack. |
| Case documents in Advocatio | `GET /v1/documents` and `GET /v1/legal-source-packages` return 404 on GET; matter summary is the only read surface for package state. |

## Separate defect found in the toolkit ledger

59 of 193 ledger records carry `content_hash` = `e3b0c442…b855`, the SHA-256 of zero bytes.
Those records have no real content fingerprint; importing them as "versioned" would be false.

## What has to exist before the proof can pass

1. A hosted, phone-reachable toolkit surface (owner rule: nothing runs on the desktop).
2. One identifier and one version construction for legal sources used by both sides
   (options to put to the owner: adopt the ledger slug + sha256 content hash; or mint a shared ID and keep both as aliases).
3. Import/reconcile of the 193 ledger records and the cheat-sheet pack into a store both surfaces read,
   with the 59 empty-hash records re-fetched or flagged unverified.
4. A GET read surface for case documents in Advocatio.
