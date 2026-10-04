# Family Court console — hosted build source

Byline: Codex, 2026-10-04. Refreshed against Dockerfile.cloud and the retired sync script.

The hosted console is the agent MCP endpoint and mobile browser surface.
Coolify builds it from Cursedpotential/propria, base directory
modules/Probata/probata, compose deploy/family-court-console.yaml.

The canonical development plugin is
E:/AI_Workspace/plugins/plugins/family-court-toolkit.
deploy/docker/family-court-console/src is its tracked deployment mirror.
Review and reconcile bounded runtime changes between those two locations.
Keep recovered source packages and private case material out of commits.

## Build contract

Dockerfile.cloud performs npm ci and node build.mjs in its build stage.
It copies the resulting server and web bundles into the runtime image.
A clean clone must build without a desktop-generated dist directory.

The source mirror includes mcp-app/src, mcp-app/web, mcp-app/widgets,
mcp-app/tests, build.mjs, tsconfig.json and pinned package files.
Preserve the sibling content and skills directories: server tools resolve
those roots relative to mcp-app/dist at runtime.

Do not run scripts/sync_family_court_console.sh. Its retirement notice explains
that it replaces this source build with the obsolete prebuilt-dist layout.
Do not copy a generated dist tree over the committed source build or use the
retired script to refresh content.

## Update and verification

1. Compare the canonical plugin and deployment mirror before updating exact
   source paths. Preserve concurrent changes and original source material.
2. Run targeted behavioral checks for the changed operation. Build the
   deployment image through Coolify on the VPS.
3. Stage only reviewed source paths; commit and push to Propria. Automatic
   deployment remains disabled.
4. Deploy family-court-console through Coolify, then verify its live operation.
   A finished deployment or healthy container alone does not prove the phone
   page, authenticated API, MCP tool call or shared record behavior.

Use the deployed trusted-proxy authentication paths for browser checks.
ContextForge virtual server family-court-console is the agent path; /mcp
keeps its bearer authentication. Never publish authentication material.

## Shared data and source reconciliation

Working records belong to surreal-case, namespace fct, database case.
Phone, desktop, workdesk and platform must use those same records.
A deployment mirror or shipped file library is not a second writable case store.

The owner-approved convergence also requires preserving the final validated
source corrections, durable Case Bible originals and revalidation before
library updates propagate. Track bulk source work as Temporal Activities.
See Docstore note:family_court_convergence_20261004 and
modules/Legal-desktop/AGENTS.md for the current convergence contract.
