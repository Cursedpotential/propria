---
tags: [ccc, cocoindex-code, tooling, watchdog, patch, pending-review]
---

# ccc safety guard kills a healthy re-index at 300 s — patch ready, NOT applied

> _Byline: Claude Code · Fable 5.1 · 2026-09-20_

## What is wrong

- `ccc search` / `ccc index` in the Probata repo hang, then fail with `Connection to daemon lost`.
- The daemon is stopped by the local safety guard
  (`%APPDATA%\uv\tools\cocoindex-code\Lib\site-packages\cocoindex_code\_safety_guard.py`, written by Codex 2026-09-20)
  with `index_progress_stalled`, and every restart is then blocked by `~/.cocoindex_code/safety-fault.json` + `safety-active.json`.
- Happened twice today: 18:29 EDT (pid 14976) and 19:03 EDT (pid 13728).

## Cause (read from the code, confirmed live)

- `project.py:129` reports progress as six per-file counters from cocoindex's `process_file` component.
- `_safety_guard.py` `watch()` declares a stall when that tuple has not changed for 300 s.
- While cocoindex applies the target (one long SQLite transaction) no counter moves. Live, the journal
  `.cocoindex_code/target_sqlite.db-journal` grew 114 → 200 → 250 MB over the same minutes the guard called it stalled.
- Not the embedder: `nvidia/nemotron-3-embed-1b` on NIM returned 8 texts → 8 × 2048-dim vectors in 0.5 s, and it is already the configured model.

## The fix

- `2026-09-20-ccc-safety-guard-target-activity.patch` (beside this file): before the stall test, the guard stats the project's
  `target_sqlite.db`, `-journal` and `-wal`; a changed size or mtime refreshes that job's timer. Memory limits, the heartbeat
  check and the fault latch are untouched. A job whose index files stop changing still trips at 300 s.
- Proof on a throwaway `Guard` built from the patched copy: first sight of the file counts as activity; no writes → still
  stale (guard can trip); file grew → timer refreshed. The patched copy compiles.

## Why it is not applied

- The Claude Code permission classifier denied the edit to `_safety_guard.py` (reason: "Security Weaken"). Not routed around.
- A backup of the current file already exists next to it: `_safety_guard.py.bak-20260920T1905-pre-target-activity`.

## To apply (owner, or any session allowed to)

```bash
cd "$APPDATA/uv/tools/cocoindex-code/Lib/site-packages/cocoindex_code" && patch _safety_guard.py < "E:/AI_Workspace/Projects/Propria/modules/Probata/probata/docs/pending-review/2026-09-20-ccc-safety-guard-target-activity.patch"
```

Then archive the two latch files and re-index:

```bash
cd ~/.cocoindex_code && mkdir -p faults-archive && mv safety-fault.json safety-active.json faults-archive/
```

```bash
cd E:/AI_Workspace/Projects/Propria/modules/Probata/probata && ccc index
```

A `uv tool upgrade cocoindex-code` overwrites this patch along with the rest of Codex's local guard.
