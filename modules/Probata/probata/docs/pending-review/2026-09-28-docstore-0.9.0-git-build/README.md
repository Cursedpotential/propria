---
tags: [docstore, propria, release, deployment, git-build]
---

# Docstore 0.9.0 — the deployment comes from git

> _Byline: Claude Code · Opus 5 · 2026-09-28. Owner order 08:40: "I want the best newest version…
> properly configured, properly versioned, properly backed up, properly in git, properly deployed,
> and running on the VPS."_

## What was wrong

The live Docstore was built **on the host**, from `/data/propria/releases/docstore-0.8.1`, which is
in no git repository. `deploy/docstore-worker.yaml` already recorded that: *"Its build context is
/data/propria/releases/docstore-0.8.1, which is in no git repository."*

That was true, and it was not the whole story. Building from git exposed three further defects that
had been invisible because nothing in a clone could run.

### 1. The sync could never start in a container

`docs/docstore-source-registry.json` is authored on the desktop and declares

```json
"monorepo_root": "E:/AI_Workspace/Projects/Propria"
```

`source_sync.state()` resolves that value strict. Inside a Linux container a Windows drive letter is
an ordinary relative path segment, so it resolved to `/app/E:` and every run died with
`FileNotFoundError` before reaching any indexing.

**This is what actually failed on 2026-09-27**, not the stale roots below. The roots were a second,
genuine defect found on the way to it.

`build_projection.py` already existed for exactly this problem — it writes a container copy of the
registry with `monorepo_root` replaced by `--container-root`. The image now performs the same
substitution at build time, and the build gate resolves the seven roots *through* the rewritten
value, so it walks the path the worker walks.

### 2. Three files kept their own copy of the source roots

`docstore_health` reported five `allowed_source_roots` on pre-2026-09-20 module paths **even after
the service was rebuilt from git**, because the stale list was committed, not merely stale on the
host:

| File | What it did |
|---|---|
| `scripts/docstore/api.py` | built `INDEX_IDENTITY` from a hard-coded five-root list, so every `/health` and identified response advertised roots the pipeline stopped using on 2026-09-20; its FastAPI version string still read `0.8.1` |
| `plugins/docstore/control/server.py` | carried the same list, so the control MCP told clients the same wrong thing |
| `scripts/docstore/adr.py` | joined `root/'Probata/probata'` and checked containment against `root/'Probata/probata/docs'`, so ADR materialisation wrote outside the junction layout |

All three now derive from `scope.ROOTS`, and a release test rejects any literal source root in
executable code under `scripts/docstore` or the control plugin. Comments and docstrings may still
discuss the old paths; strings may not. Checked against the pre-fix files, that test catches ten
literals.

### 3. The host ran older code than git, and the tests lived only on the host

| File | Host | git |
|---|---|---|
| `scripts/docstore/scope.py` | the **five**-root version | the **seven**-root version, reading the `docs/<project>` junctions |
| `scripts/docstore/flow_docs.py` | no live watch, no preview | live watch (`DOCSTORE_LIVE`), `DOCSTORE_PREVIEW`, full-reprocess options |
| 66 other shared `.py` files | identical | identical |

Six release tests existed only on the host — they were the evidence behind every "339 passed" claim
in r5/r6, and a fresh clone could not run them. They are tracked now, under `tests/release/`.

Running them from git showed four had been quietly wrong: three hard-coded the root count or the
five project ids, and `test_worker_safety.py` read `<probata>/Dockerfile` and `deploy/compose.yaml`
— the flat layout of the *release tree*, not of this repository. Every count now derives from
`scope.ROOTS`.

### 4. Two control servers ran in parallel

Against the owner's one-live-instance rule: the git-built app `probata-docstore-control` (`:8172`,
ContextForge `ctl`) and the release service (`:8175`, ContextForge `ctl08`). Clients used `ctl08`,
so a git push never reached what was serving.

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
  API, the control MCP server, the schema, and the documentation roots. The build context is the
  **monorepo root**, because the documentation it indexes lives in seven modules.
- **The docs are built into the image from git** (owner choice A, 2026-09-28). The host mirror fed
  by `client.py sync` is gone: every docs push rebuilds the image, and the start-up pass is the
  change event — the same contract the retired `docstore-worker` app had.
- **The deployment definition ships inside the image**, at the path it has in the repository, so a
  running container can show what built it and `test_worker_safety` can assert its own build inputs.
- **Secrets leave the compose file.** They were literal values in the service's compose
  (`NVIDIA_API_KEY`, `SURREAL_DOCS_PASS`, `DOCSTORE_API_TOKEN`, `DOCSTORE_CONTROL_TOKEN`,
  `MEMORY_BASIC_AUTH`) and the Coolify API returned them in full. They are Coolify environment
  variables now, referenced by name, and every one was verified by hash against the running 0.8.1
  container before cutover.
- **One control server.** The `:8172` app *became* the 0.9.0 Docstore rather than being replaced by
  a third thing, and the release service is stopped.

## The cutover, as performed

The existing Coolify application `probata-docstore-control`
(`ywo2qvc5catoa79zgdur5o2j`) was **repointed in place** — no new app, no parallel stack:

| Setting | Was | Now |
|---|---|---|
| name | `probata-docstore-control` | `propria-docstore` |
| base_directory | `/modules/Probata/probata` | `/` (the monorepo root) |
| docker_compose_location | `/deploy/docstore-control.yaml` | `/modules/Probata/probata/deploy/docstore.yaml` |
| watch_paths | 3 control paths | the 11 paths that feed this image |
| published ports | `8172` | `8072` (API) and `8175` (control MCP) |

Order: repoint the app (config only, containers untouched) → stop the 0.8.1 service, freeing 8072
and 8175 → deploy. Coolify clones `main` and builds; nothing on the host is edited by hand.

Every Coolify operation went through the `coolify-write` plugin's own tools. Two gaps found while
doing it were closed in the plugin rather than worked around: `list_applications` / `list_services`
/ `list_databases` / `list_projects` advertised summaries but returned whole records — `list_services`
included `docker_compose_raw` with every secret in plain text — and there was no `update_application`
tool, so repointing an app would have required a raw API call. Both are fixed in
`E:/AI_Workspace/plugins/plugins/coolify-write`.

## Backups taken before the cutover

| What | Where |
|---|---|
| The release tree | `/data/propria/releases/_backup/docstore-0.8.1-20260928T124408Z.tar.zst` (sha256 `7ed9a3d4…`) |
| The docs SurrealDB | 840 MB SurrealQL export beside it (sha256 `cf03ecc2…`) |
| The worker state volume | archived in the same set (sha256 `79ba1b0c…`) |

The release tree stays in place, untouched, as the rollback path. The stopped service can be
restarted with `start_service` on `o8obobz576je1fbyygnywl83`.

## Acceptance

1. `docstore_health` returns `ok: true` with the **seven** roots `scope.py` declares —
   `docs`, `docs/probata`, `docs/consignatio`, `docs/consignatio-intake`, `docs/advocatio`,
   `docs/vestigia`, `docs/family-court`.
   _Corrected 2026-09-28: an earlier draft of this criterion expected a `modules/` prefix. That
   contradicted `scope.py`, which is explicit that roots are read through the `Propria/docs`
   junctions and never through module paths._
2. A real `coco_docstore_search` returns results (not the empty list the stale roots produced).
3. One sync completes with `source_digest_after` set and `cdc_verified` true.
4. `atomic_tools` and the other ContextForge gateways still answer.
5. Nothing on the host is edited by hand; the image is rebuilt by Coolify from `main`.

### Status

| # | Criterion | State |
|---|---|---|
| 1 | seven roots, `ok: true` | roots **verified** (seven, junction paths); `ok` awaits criterion 3 |
| 2 | search returns results | pending |
| 3 | sync with digest + cdc | in progress |
| 4 | ContextForge gateways answer | control MCP observed serving `POST /mcp 200` from ovh-app |
| 5 | no hand edits on the host | **held** — every change went through git and the Coolify plugin |
