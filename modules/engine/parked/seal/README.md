# parked/seal — content-addressed sealing, held for the promotion step

> _Byline: Claude Code · Fable 5.1 · 2026-09-06._

**Status:** PARKED (reuse later). Not quarantine, not deletion.

**Rule (owner, 2026-09-06; D-124 amended, D-145):**
- Sealing an original into the content-addressed store = custody work = promotion-to-evidence step only.
- Ingest-time retain = working copy + SHA-256 row on a table. No seal ceremony.
- SHA is revalidated at promotion.
- This code is reused when the promote activity is built. Do not rewrite it.

| File | Original path | Reuse point |
|---|---|---|
| `sealfile.go` | `modules/engine/acquisition/sealfile.go` (package `acquisition`; wraps unexported `sealStream`) | `promote` activity (plan Stage 5): re-read original → seal → H1/H2/H3 |
| `proffer_seal_main.go` | `modules/engine/cmd/proffer-seal/main.go` | operator CLI for promotion-time sealing |

Both carry `//go:build parked`, so `go build ./...` skips them. Revive = move back to
the original path, drop the tag, build. Same convention as `sql/parked/` (D-121).
