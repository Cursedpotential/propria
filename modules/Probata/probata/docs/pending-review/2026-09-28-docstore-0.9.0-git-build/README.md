---
tags: [docstore, propria, release, deployment, git-build]
---

# Docstore 0.9.0 — the deployment comes from git

> _Byline: Claude Code · Opus 5 · 2026-09-28. Owner order 08:40: "I want the best newest version…
> properly configured, properly versioned, properly backed up, properly in git, properly deployed,
> and running on the VPS."_

## What was wrong

The live Docstore was built **on the host**, from `/data/propria/releases/docstore-0.8.1`, which is
in no git repository. `deploy/docstore-worker.yaml` already records that: *"Its build context is
/data/propria/releases/docstore-0.8.1, which is in no git repository."* Three consequences, all
measured on 2026-09-28:

1. **The host ran older code than git.** Comparing every Python file in the release tree against
   `origin/main`:

   | File | Host | git |
   |---|---|---|
   | `scripts/docstore/scope.py` | the **five**-root version | the **seven**-root version, reading the `docs/<project>` junctions |
   | `scripts/docstore/flow_docs.py` | no live watch, no preview | live watch (`DOCSTORE_LIVE`), `DOCSTORE_PREVIEW`, full-reprocess options |
   | 66 other shared `.py` files | identical | identical |

   The stale `scope.py` is why `docstore_health` reported `allowed_source_roots` as five
   pre-`modules/` paths and why the 2026-09-27 sync failed: the validator rejected the registry.

2. **Six release tests existed only on the host** — `test_release.py`, `test_remote_memory.py`,
   `test_public_surface.py`, `test_federation_adapters.py`, `test_memory_duplicate_guard.py`,
   `coco_retention_probe.py`, plus the release `conftest.py`. They were the evidence for every
   "339 passed" claim in r5/r6, and a fresh clone could not run them.

3. **Two control servers ran in parallel**, against the owner's one-live-instance rule: the
   git-built app `probata-docstore-control` (`:8172`, ContextForge `ctl`, 57 tools) and the
   release service (`:8175`, ContextForge `ctl08`, 5 tools). Clients used `ctl08`, so a git push
   never reached what was serving.

## Versioning

Three sequences looked like one, which is what made "0.8.1-r7" read as a downgrade from the
plugin's 0.8.6:

| Artifact | Was | Now |
|---|---|---|
| Server image | `propria-docstore:0.8.1-r1…r7` | `propria-docstore:0.9.0` |
| Claude/Codex plugin | `0.8.6` (marketplace repo) | `0.9.0` |
| `plugins/docstore/control/pyproject.toml` | `0.6.2`, never bumped | `0.9.0` |

**One version line for the whole Docstore product from here.** The server image, the plugin and
the control package share it. A server-only fix bumps the patch digit; the `-rN` suffix is retired.

## What this release changes

- **`deploy/docker/docstore/Dockerfile`** builds the whole service from the repository: the worker
  API, the control MCP server, the schema, and the documentation roots.
- **The docs are built into the image from git** (owner choice A, 2026-09-28). The host mirror fed
  by `client.py sync` is gone: every docs push rebuilds the image, and the start-up pass is the
  change event — the same contract the retired `docstore-worker` app had.
- **Secrets leave the compose file.** They were literal values in the service's compose
  (`NVIDIA_API_KEY`, `SURREAL_DOCS_PASS`, `DOCSTORE_API_TOKEN`, `DOCSTORE_CONTROL_TOKEN`,
  `MEMORY_BASIC_AUTH`) and were returned in full by the Coolify API. They are Coolify environment
  variables now, referenced by name.
- **The six release tests are tracked**, under `tests/release/`, so a clone can run them.
- **One control server.** The `:8172` app and ContextForge's `ctl` gateway are retired once the
  git build serves `ctl08`'s URL.

## Backups taken before the cutover

| What | Where |
|---|---|
| The release tree | `/data/propria/releases/_backup/docstore-0.8.1-<utc>.tar.zst` |
| The docs SurrealDB | export beside it, hash recorded |
| The worker state volume | `/data/probata/volumes/docstore-worker` archived in the same set |

The release tree stays in place, untouched, as the rollback path until the git build has served a
successful sync and a real query.

## Acceptance

1. `docstore_health` returns `ok: true` with **seven** `allowed_source_roots` carrying the
   `modules/` prefix.
2. A real `coco_docstore_search` returns results (not the empty list the stale roots produced).
3. One sync completes with `source_digest_after` set and `cdc_verified` true.
4. `atomic_tools` and the other ContextForge gateways still answer.
5. Nothing on the host is edited by hand; the image is rebuilt by Coolify from `main`.
