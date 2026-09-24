# Tool Catalog — casekit

Everything needed to build and run. No cloud services, no API keys, no
accounts required for Phases 0–6.

## Vendored under `D:\case_apps\engines\`

| Tool | Role | Source | Pin |
|---|---|---|---|
| `pdftotext`, `pdfinfo`, `pdffonts`, `pdfimages`, `pdfdetach`, `pdftoppm` | poppler — primary PDF text + profile | poppler-windows release | record version in sidecar |
| `qpdf` | structure, `--check`, `--qdf` raw stream | qpdf releases | 11.9.x |
| `mutool` | raster/structure **only** — never glyph text (GOTCHAS S-2) | mupdf releases | optional |
| `duckdb` | XML ingest + Parquet + query | duckdb.org | 1.5.5, pin |
| `webbed` | DuckDB XML extension | `INSTALL webbed FROM community` | re-verify after upgrades |
| `tesseract` + `eng` traineddata | OCR, Phase 10 only | tessdata_best | deferred |

## Frozen Python venv — `D:\case_apps\engines\python\.venv`

Managed by `uv`. Keep the path shallow (GOTCHAS B-4).

| Package | Role | Pin |
|---|---|---|
| `pypdf` | secondary PDF reader, metadata, attachments | 6.17.0 |
| `pypdfium2` | pdfium reader + rasterization | pin at install |
| `lxml` | XML slim pass — the only XML parser permitted | pin at install |
| `duckdb` | Python binding if needed by an engine | match CLI version |

## Go modules

| Module | Role |
|---|---|
| `tailscale.com/tsnet` | Phase 8 only — embedded tailnet node, v1.102.0, BSD-3 |
| stdlib `os/exec`, `encoding/json` | engine subprocess contract |

Nothing else. Resist adding dependencies.

## Credentials

**Phases 0–6: none.** Everything is local.

Phase 8 adds one: a Tailscale auth key (`TS_AUTHKEY` env var, or OAuth client
secret). Never committed; read from environment at runtime.

## Existing infrastructure (not built here)

| Thing | Where | Relationship |
|---|---|---|
| Case Bible vault | `V:\sorted\_raw\Case Bible\` | source corpus; casekit reads, never writes |
| Evidence dir | `…\Evidence\Phone Records\Messages with Katrina` | first target for inventory |
| Stirling-PDF | OVH box | interactive PDF work; **not** in the extraction path |
| R2 / B2 buckets | Cloudflare / Backblaze | sealed-package destination, Phase 8+ |
| OVH server | via Tailscale | daemon host, Phase 8+ |

## Repair tools that live outside this kit (use them; do not rebuild)

> _Byline: Claude Code · Opus 5.5 · 2026-09-24 — owner 09:46–09:47: "we have repair tools too"; "if you try the toolkit and there's no tool, then we have to find one and add it to the toolkit."_

| Damage | Tool | Where | Behaviour |
|---|---|---|---|
| SMS Backup & Restore XML that stops mid-record (truncated copy; e.g. declares `count="12416"`, object is 688,128 bytes) | DuckDB streaming sanitizer v2 | `Consignatio/casebible/tools/comm_timeline_mvp/elt/elt_xml_sanitize_v2.sql`, run by `elt_run.py` | keeps every complete record up to the cut, closes the root, reports `truncated_salvaged` with declared vs recovered counts. Rows are *reconstructed*, not *recovered* (RULES R9, GOTCHAS A-15). |
| SMS Backup & Restore XML with a bare `&`, illegal control bytes, surrogate character references, or 95 MB base64 lines | same sanitizer (v1 behaviour, kept in v2) | same | escapes/strips per line, swaps base64 for `data_len` + `data_sha256` |

## Reference files — commit these

Both belong in `docs/reference/`. Every phase exit criterion is stated against
them.

- `_18102689630__1___1_.pdf` — sha256 `f18e250e5aca784f…`
- `Copy_of_sms-20221104021809.xml` — sha256 `55515f2a490c9a0a…`
- `sms.xsd` and `Fields in XML backup files.md` — vendor schema, synctech.com.au
