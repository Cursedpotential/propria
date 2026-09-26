# Monorepo cutover — Coolify repointing plan

> _Byline: Claude Code · Opus 5.5 · 2026-09-26_

**Status: plan only; nothing in Coolify was changed.** Written for the session running
`integration/monorepo-import-20260926` ([decision](decisions/2026-09-26-monorepo-import.md)), which owns the
cutover. Live data: Coolify v4.1.2 at `http://100.98.98.38:8000`, read through the API on 2026-09-26.
Machine-readable diff with the API bodies: [`monorepo-coolify-cutover-2026-09-26.json`](monorepo-coolify-cutover-2026-09-26.json).

## What changes

All 35 applications are Docker Compose apps built from four GitHub repositories. 32 build from
`Cursedpotential/probata` with base directory `/`; one each from `Consignatio`, `Legal-Workspace` and
`Intake-desktop`. After the cutover every one builds from `Cursedpotential/propria` (repository id
`1367548570`), with the module folder as its base directory.

## How Coolify resolves paths (checked in the v4.1.2 source)

- **Base directory and compose file.** The deploy job sets `workdir = <clone>` + `base_directory` and runs
  `docker compose --project-directory <workdir> -f <workdir><docker_compose_location>`
  (`app/Jobs/ApplicationDeploymentJob.php`). Setting the base directory to the module folder therefore keeps
  every compose file, and the relative build contexts and bind mounts inside it, working unchanged.
  `docker_compose_location` stays as it is.
- **Watch paths are not relative to the base directory.** The push webhook collects the changed files from
  GitHub's payload (`commits.*.added/removed/modified`, repository-root paths) and matches the watch patterns
  against them as written (`app/Http/Controllers/Webhook/Github.php`, `Application::matchPaths`). So every
  pattern needs the module prefix; `deploy/workbench.yaml` becomes `modules/Probata/probata/deploy/workbench.yaml`.
  Globs support `*`, `**`, `?`, `[abc]`; a leading `!` excludes; the last matching pattern wins.
- **An empty watch path means "deploy on every push"** (`isWatchPathsTriggered(...) || blank(watch_paths)`).
  Six apps have none today; inside the monorepo they would redeploy on every commit anywhere in Propria.
  The plan gives each a pattern covering its own module, which keeps today's behaviour (every push to its
  own code).
- **The GitHub App finds apps by `repository_project_id`**, the numeric GitHub repository id
  (`Github.php`: `Application::where('repository_project_id', $id)`). The public API cannot set that field
  (it is not in the update allow-list of `ApplicationsController`), so each app needs its repository picked
  again in the Coolify UI (Source tab). Without it, pushes to Propria trigger nothing.

## Before the first app moves

1. Give the Coolify GitHub App installation access to `Cursedpotential/propria`.
2. Re-create Probata's 8 Actions secrets on `propria` (`SUBMODULE_TOKEN`, `TS_OAUTH_CLIENT_ID`,
   `TS_OAUTH_CLIENT_SECRET`, `INTEGRATION_DB_USER`, `INTEGRATION_DB_PASS`, `INTEGRATION_SBV_BASE_URL`,
   `INTEGRATION_SBV_SERVICE_USER`, `INTEGRATION_SBV_SERVICE_PASS`); GitHub does not reveal their values.
3. Decide the two apps that deploy a non-default branch: `superindex` (`feat/superindex-service`, not merged
   into Consignatio `main`) and `probata-docstore-control` (`docstore-control-host-20260919`). Either merge the
   branch into its module first, or import it as a Propria branch and keep deploying that branch.
4. Confirm which Xplorer branch the import put into Propria `main` (the plan assumes
   `feat/hosted-intake-engine`, which `intake-engine` deploys today).
5. `family-court-console` has been `exited:unhealthy` since 2026-06-13: repoint it or retire it.
6. Push the monorepo `main` only after the history gates clear (TraceIQ secret findings; the Vestigia
   `raw_api_responses/` location data kept out of git).

## Order

1. **Pilot one low-risk app** (`llm-probe-ui`): apply its API patch, pick `Cursedpotential/propria` in the
   Source tab, deploy manually, check it is healthy. Then push a commit that touches only its module path and
   confirm a deploy fires; push a commit elsewhere and confirm none does.
2. **Move the rest in small batches**, least critical first: API patch, Source tab, manual deploy, health check.
3. **When every app deploys from Propria**, archive `probata`, `Consignatio`, `Legal-Workspace` and
   `Intake-desktop` on GitHub as read-only. Archive, do not delete.

**Rollback per app:** apply the `current` values from the JSON with the same API call and pick the old
repository again in the Source tab.

## Per-app diff

| App | UUID | Builds today | Base directory | Compose file (unchanged) | Watch paths today | Watch paths after | Flags |
|---|---|---|---|---|---|---|---|
| `superindex` | `f12skzwshwp85b1k4lbgm0pp` | Consignatio@feat/superindex-service | `/Intake/backend` → `/modules/Consignatio/Intake/backend` | `/deploy/superindex.compose.yml` | `Intake/backend/**` | `modules/Consignatio/Intake/backend/**` | deploys branch feat/superindex-service: merge it into the module's main before cutover, or import it as a propria branch and keep deploying that |
| `intake-engine` | `dbae59tufgs5zqvb7ym9fozk` | Intake-desktop@feat/hosted-intake-engine | `/` → `/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot` | `/docker-compose.intake-engine.yaml` | *(none)* | `modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module; assumes the import brought feat/hosted-intake-engine into propria main; confirm first |
| `legal-workspace` | `gvghzivfmctev8dloetfssnj` | Legal-Workspace@master | `/` → `/modules/Legal-desktop` | `/compose.yaml` | *(none)* | `modules/Legal-desktop/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module; Legal-Workspace master is what the import brought into propria main |
| `authentik` | `ak206exj3esct2x6h8pdjo4g` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/authentik.yaml` | `deploy/authentik.yaml` | `modules/Probata/probata/deploy/authentik.yaml` |  |
| `coolify-mcp` | `oyzznioap03u34xz125l90oq` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/coolify-mcp.yaml` | `compose.coolify-mcp.yaml`<br>`docker/coolify-mcp/**` | `modules/Probata/probata/compose.coolify-mcp.yaml`<br>`modules/Probata/probata/docker/coolify-mcp/**` |  |
| `data-neo4j` | `ksbq02zynhdt63b8b9ba5cpv` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/data-neo4j.yaml` | `compose.data-neo4j.yaml` | `modules/Probata/probata/compose.data-neo4j.yaml` |  |
| `data-pg-files` | `w10gg3an43jvry4y79n6sxi1` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/data-pg.yaml` | `compose.data-pg.yaml`<br>`docker/postgres/**` | `modules/Probata/probata/compose.data-pg.yaml`<br>`modules/Probata/probata/docker/postgres/**` |  |
| `data-weaviate-native-v1` | `v43tfq25o7i561n4lnc124p2` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/data-weaviate-native-v1.yaml` | `deploy/data-weaviate-native-v1.yaml` | `modules/Probata/probata/deploy/data-weaviate-native-v1.yaml` |  |
| `devbox` | `pd3xc78ahqkfswq12bpfqgy1` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/devbox.yaml` | `deploy/never-auto-deploy-devbox/**` | `modules/Probata/probata/deploy/never-auto-deploy-devbox/**` |  |
| `exec-contextforge` | `k272znxpa4gh6drmolut723w` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/contextforge.yaml` | `deploy/contextforge.yaml` | `modules/Probata/probata/deploy/contextforge.yaml` |  |
| `exec-desktop` | `t130q2xn4r1tux3huee9gal1` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/desktop.yaml` | `compose.desktop.yaml` | `modules/Probata/probata/compose.desktop.yaml` |  |
| `exec-gateway` | `f29166r47gro6fjiq4d8ya92` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/gateway.yaml` | `compose.gateway.yaml`<br>`docker/gateway/**` | `modules/Probata/probata/compose.gateway.yaml`<br>`modules/Probata/probata/docker/gateway/**` |  |
| `exec-sandbox` | `mn2autapl223gmgcpjqy7def` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/sandbox.yaml` | `compose.sandbox.yaml`<br>`docker/sandbox/**` | `modules/Probata/probata/compose.sandbox.yaml`<br>`modules/Probata/probata/docker/sandbox/**` |  |
| `exec-tier` | `rz41wqhpjfh1rj796ixvjhfs` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/exec.yaml` | `Dockerfile`<br>`pyproject.toml`<br>`requirements.txt`<br>`server/**`<br>`scripts/entrypoint.sh`<br>`deploy/exec.yaml` | `modules/Probata/probata/Dockerfile`<br>`modules/Probata/probata/pyproject.toml`<br>`modules/Probata/probata/requirements.txt`<br>`modules/Probata/probata/server/**`<br>`modules/Probata/probata/scripts/entrypoint.sh`<br>`modules/Probata/probata/deploy/exec.yaml` |  |
| `family-court-console` | `sokv65ibdq2y8xdaqmd6p4rq` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/family-court-console.yaml` | *(none)* | `modules/Probata/probata/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module; status exited:unhealthy: dead app, repoint or retire |
| `fileflows` | `bbmf0b7he1k14ivftiu4stry` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/fileflows.yaml` | `deploy/fileflows.yaml` | `modules/Probata/probata/deploy/fileflows.yaml` |  |
| `infisical` | `sp2ueak6dpul09ao8ul5k2z4` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/infisical.yaml` | `deploy/infisical.yaml` | `modules/Probata/probata/deploy/infisical.yaml` |  |
| `llm-probe` | `c26fveiswdhb77b6dpd7zgdp` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/llm-probe.yaml` | *(none)* | `modules/Probata/probata/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module |
| `llm-probe-ui` | `z5rjgzn1agph1qo2jm55tr40` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/llm-probe-ui.yaml` | *(none)* | `modules/Probata/probata/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module |
| `memsearch-milvus` | `d725i1io2o1dwlfjdz09lo87` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/memsearch-milvus.yaml` | `deploy/memsearch-milvus.yaml` | `modules/Probata/probata/deploy/memsearch-milvus.yaml` |  |
| `metabase` | `zb0hi2bi26vnndyw9eozn737` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/metabase.yaml` | `deploy/metabase.yaml` | `modules/Probata/probata/deploy/metabase.yaml` |  |
| `octopoda` | `gwsmgd0sbqd9aheysa9g7xh4` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/octopoda.yaml` | *(none)* | `modules/Probata/probata/**` | no watch paths today: redeploys on every push; new pattern keeps it to its own module |
| `opencode-server` | `w8n50kzvgn3uzqiivvozcmbn` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/opencode-server.yaml` | `deploy/opencode-server.yaml` | `modules/Probata/probata/deploy/opencode-server.yaml` |  |
| `openlist` | `pn6t3nsdrhnxnueuwe7756g5` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/openlist.yaml` | `deploy/openlist.yaml` | `modules/Probata/probata/deploy/openlist.yaml` |  |
| `parser-activity-runtime` | `o11nxvzqwskxrqmtbvup7iet` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/parser-activity-runtime.yaml` | `modules/engine/**`<br>`deploy/docker/parser-activity-runtime/**`<br>`deploy/parser-activity-runtime.yaml` | `modules/Probata/probata/modules/engine/**`<br>`modules/Probata/probata/deploy/docker/parser-activity-runtime/**`<br>`modules/Probata/probata/deploy/parser-activity-runtime.yaml` |  |
| `portkey` | `z5787t1l7gl2zbrya8cxzapf` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/portkey.yaml` | `compose.portkey.yaml`<br>`docker/gateway/portkey/**` | `modules/Probata/probata/compose.portkey.yaml`<br>`modules/Probata/probata/docker/gateway/portkey/**` |  |
| `probata-docstore-control` | `ywo2qvc5catoa79zgdur5o2j` | probata@docstore-control-host-20260919 | `/` → `/modules/Probata/probata` | `/deploy/docstore-control.yaml` | `deploy/docstore-control.yaml`<br>`deploy/docker/docstore-control/**`<br>`plugins/docstore/control/**` | `modules/Probata/probata/deploy/docstore-control.yaml`<br>`modules/Probata/probata/deploy/docker/docstore-control/**`<br>`modules/Probata/probata/plugins/docstore/control/**` | deploys branch docstore-control-host-20260919: merge it into the module's main before cutover, or import it as a propria branch and keep deploying that |
| `probata-tool-runtime` | `e1mshujml6bv8ldtoe8n7je0` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/tool-runtime.yaml` | `deploy/tool-runtime.yaml`<br>`deploy/docker/tool-runtime/**`<br>`server/**` | `modules/Probata/probata/deploy/tool-runtime.yaml`<br>`modules/Probata/probata/deploy/docker/tool-runtime/**`<br>`modules/Probata/probata/server/**` |  |
| `proffer-starter` | `r1084s1lsm80fsv4ol9ocij0` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/proffer-starter.yaml` | `modules/engine/**`<br>`deploy/docker/proffer-starter/**`<br>`deploy/proffer-starter.yaml` | `modules/Probata/probata/modules/engine/**`<br>`modules/Probata/probata/deploy/docker/proffer-starter/**`<br>`modules/Probata/probata/deploy/proffer-starter.yaml` |  |
| `proffer-worker` | `d24bb9eoo47qtw9eq1xc6u64` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/proffer-worker.yaml` | `modules/engine/**`<br>`deploy/docker/proffer-worker/**`<br>`deploy/proffer-worker.yaml` | `modules/Probata/probata/modules/engine/**`<br>`modules/Probata/probata/deploy/docker/proffer-worker/**`<br>`modules/Probata/probata/deploy/proffer-worker.yaml` |  |
| `surreal-case` | `qkcbapa8ozh4ynda2u69055z` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/surreal-case.yaml` | `deploy/surreal-case.yaml` | `modules/Probata/probata/deploy/surreal-case.yaml` |  |
| `surreal-docs` | `r13ehbwuw3xji8x9mibeypp0` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/surreal-docs.yaml` | `deploy/surreal-docs.yaml` | `modules/Probata/probata/deploy/surreal-docs.yaml` |  |
| `temporal-stack` | `llv5zt8phx1xf4devwqugk3y` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/temporal/compose.temporal.yaml` | `deploy/temporal/compose.temporal.yaml` | `modules/Probata/probata/deploy/temporal/compose.temporal.yaml` |  |
| `tool-gateway` | `ws67wgw1qxdgxo956p2k1jvi` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/tool-gateway.yaml` | `modules/engine/**`<br>`deploy/docker/tool-gateway/**`<br>`deploy/tool-gateway.yaml` | `modules/Probata/probata/modules/engine/**`<br>`modules/Probata/probata/deploy/docker/tool-gateway/**`<br>`modules/Probata/probata/deploy/tool-gateway.yaml` |  |
| `workbench` | `xjbuo6drbwjfby75lalk8bk7` | probata@main | `/` → `/modules/Probata/probata` | `/deploy/workbench.yaml` | `modules/workbench/**`<br>`deploy/workbench.yaml` | `modules/Probata/probata/modules/workbench/**`<br>`modules/Probata/probata/deploy/workbench.yaml` |  |
