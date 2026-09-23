# Hosted Intake engine release preflight — 2026-09-23

Byline: Codex · GPT-6 · 2026-09-23

## Planned change — before modification

Record a bounded Linux/container release check for the three commits on
`feat/hosted-intake-engine`. Verify the exact Coolify source and deployed image,
build the tracked candidate in an isolated Linux build context, exercise health,
name search, and Smart Suggestions without an app bearer token, then record the
deployment and user-facing acceptance evidence if the gates pass. The owner has
authorized a normal push of this branch and deployment through Coolify after the
preflight. No portal, Authentik, Tailscale Services, credential, or live database
configuration edits are in this lane.

The only planned repository edit is this receipt unless a specific release defect
requires an Intake-owned fix. Any such fix will receive a separate planned-change
and checkpoint block before modification. Do not alter the existing quarantine.

## Checkpoint — before modification

- Git root: `E:/AI_Workspace/Projects/Propria/modules/Consignatio/Intake/xplorer-copilot-buildkit/xplorer-copilot`.
- Branch and HEAD: `feat/hosted-intake-engine` at `c5dd8b976e5958398d193c83d87a2068f403fdbc`.
- Remote branch: `private/feat/hosted-intake-engine` at `69773c8decc8fbfe999098b5419bf51eb658bf8f`; local is ahead by three commits, with no remote divergence at this checkpoint.
- Working tree: tracked files clean; only pre-existing untracked `to_be_deleted/`.
- Coolify application: `intake-engine`, UUID `dbae59tufgs5zqvb7ym9fozk`, server `ovh-files`, source `Cursedpotential/Intake-desktop`, branch `feat/hosted-intake-engine`, compose location `/docker-compose.intake-engine.yaml`.
- Last finished deployment: `69773c8decc8fbfe999098b5419bf51eb658bf8f` on 2026-09-20. Running container is healthy and uses image `sha256:149e4199196556060e0f319c405206d55cf8557ac6eaaa4dcefff392b3e7dc15`.
- The candidate Linux build started from a streamed `git archive HEAD` containing only the Dockerfile's tracked context paths. Tag: `intake-preflight:c5dd8b97`. It does not touch the running Coolify service.

## Verification and release decision

Byline: Codex · GPT-6 · 2026-09-23

- Linux release image: passed. Streamed `git archive --format=tar HEAD` with the
  exact Dockerfile context paths to `docker build --progress=plain -t
  intake-preflight:c5dd8b97 -f apps/intake-engine/Dockerfile -` on `ovh-files`.
  The final image ID is
  `sha256:c41fca5f37f15331d200c505deb0a47fd089775f9ecec833847647c91f2707b9`.
  Build output reported ten unused-import/dead-code warnings from reused donor
  code and no error. The running Coolify container was not touched.
- Coolify compose: build path, Tailnet-only host bind `100.91.190.107:8790`,
  state, OpenList and read-only secret mounts, and all thirteen environment keys
  match the tracked compose file after removing comments and blank lines. Coolify
  reports HTTP basic auth disabled. The engine HTTP router has no bearer/app-auth
  middleware; access is governed by the Tailnet bind and the shared portal's
  separate boundary.
- Isolated runtime: passed. Launched the candidate image with a loopback-only
  listener `127.0.0.1:18790`, tmpfs state and synthetic files, and read-only
  mounted catalog/model credential files. No running Coolify container or source
  object was changed. Unauthenticated `GET /healthz` returned `ok=true`, 332
  commands, and `mount_readable=true`.
- Unauthenticated `POST /api/intake_search_names` for `Takeout`, scope
  `everything`, kinds `folders`, limit 2 returned HTTP 200: two shown of 4,121
  matching folder hits, `capped=true`, 1,605 ms, with no search-leg errors.
- Unauthenticated `POST /api/analyze_directory` on two synthetic PDF names
  returned HTTP 200 twice. The first model call timed out at 45 seconds and
  truthfully reported `suggestion_source=fallback`, zero suggestions, and the
  timeout note. A bounded retry returned `suggestion_source=model:portkey:gemini-3.8-flash`,
  one suggestion, no fallback note, in 17,840 ms. This proves the configured
  model route can answer; transient provider latency remains visible to users.
  The test container was then stopped and retained for inspection.

Release preflight decision: **green for a normal branch push and Coolify-managed
deployment**, subject to a fresh remote fast-forward check. This is a container
and API preflight, not user-facing deployment acceptance.

## Push, deployment, rollback and live acceptance

Byline: Codex · GPT-6 · 2026-09-23

1. Fetch `private feat/hosted-intake-engine`, verify the remote remains an
   ancestor and the tracked diff contains only this receipt, then stage and
   commit this exact file in the child Git root. Push
   `feat/hosted-intake-engine:feat/hosted-intake-engine` to `private` without
   force. Do not push to public `origin` or `upstream`.
2. Deploy only Coolify application `dbae59tufgs5zqvb7ym9fozk` through
   `coolify-write-deploy-application`, then inspect that application's deployment
   status/logs and final commit. Do not start, stop, or replace its container
   directly.
3. Live acceptance: Coolify reports `running:healthy`; the image was built from
   the pushed commit; direct `http://100.91.190.107:8790/healthz` returns 200
   without an app token; a search from the actual Tailnet browser UI finds a
   folder beyond the open folder and shows its original/current locations; Smart
   Suggestions returns a model-backed suggestion or an explicit fallback note;
   the user is not asked for a second app login. The shared portal route is
   owned and verified in the separate Claude lane.
4. If the new service fails acceptance, use Coolify's deployment history for
   immediate operational rollback if it offers the previously healthy image.
   For a source rollback, create normal revert commits for `c5dd8b97`,
   `34aa2ba7`, and `a198bfe0` in that order, inspect the diff/tests, push the
   branch normally to `private`, and redeploy the same Coolify application.
   Never force-push, reset, or touch the quarantined folder.

## Deployment checkpoint

Byline: Codex · GPT-6 · 2026-09-23

- Preflight receipt committed as `56d60875ca4022ffe972d94a2cc0eedda7eadbf3`.
  Normal push to `private/feat/hosted-intake-engine` succeeded.
- Coolify deployment `jfo3e301kuvmvll2ncjwaptn` finished at that commit.
  Image `sha256:08ac903924f1d12fc86475a0f5bbe534c34cfbbd4943af1e39012b6a3e4ef797`
  initially reported healthy. Direct unauthenticated `/healthz` and
  `/api/intake_search_names` returned HTTP 200; the latter found two shown of
  4,121 `Takeout` folders in 1,730 ms without search-leg errors.
- The published browser bundle is older: `intake-build/current.json` on
  `ovh-app` points to release `2026-09-19T01-25-33-000Z`, source fingerprint
  `c0ef74a2`. The staged UI artifact on `ovh-files` has the same September 19
  timestamp. The new magnifying-glass UI from `34aa2ba7` therefore awaits a
  fresh UI build and publication in Claude's separate surface lane.

## Incident and correction plan — before code modification

Byline: Codex · GPT-6 · 2026-09-23

An unauthenticated live `analyze_directory` call on `/srv/openlist` exceeded
the 65-second client timeout. Code inspection identified a recursive
`build_duplicate_summary` call at the end of `analyze_directory`; it descends
through the entire selected tree and can hash candidate duplicates. On the
OpenList root this is an unintended corpus scan. The HTTP layer also detaches
the task after client disconnect, so the client timeout did not cancel server
work. Even `/healthz` briefly stopped answering. A call to
`cancel_duplicate_scan` also timed out. No source operation or cleanup was
requested by this diagnostic call.

Coolify-managed restart `hce0huzs4vql7b1genoal935` finished. The old
container and scan process are gone; the replacement container returned
unauthenticated `/healthz` HTTP 200 and Coolify reported `running:healthy`.
This incident supersedes the preflight's green release decision until a fix is
built and verified. Do not run analysis on a corpus root again.

Planned narrow code change: in hosted mode, remove the recursive duplicate
summary from ordinary directory analysis; enforce a bounded direct-child
entry/time check before Smart Suggestions; abort its detached request task on
client disconnect or a finite overall timeout. Add focused tests for the entry
bound and for both cancellation paths releasing task capacity. Preserve the
desktop command's existing duplicate workflow and do not change the portal,
credentials, database, or source objects.

## Checkpoint — before correction

- Child Git root and branch remain as above; HEAD `56d60875`, remote equal.
- Tracked working tree was clean before this receipt update. Existing untracked
  `to_be_deleted/` remains intact.
- Back up the three source files and this receipt under a new isolated
  `to_be_deleted/2026-09-23-hosted-analysis-bounds/` directory before editing
  source. Only the owner may remove quarantined material.
- Fix acceptance: Linux locked tests/build, isolated runtime health and bounded
  fixture analysis, request-cancellation/timeout capacity tests, normal push,
  Coolify redeploy, and live read-only health/name-search checks. Do not test
  Smart Suggestions against a corpus root or modify the UI release pointer.

## Bounded-analysis correction — verification before release

Byline: Codex · GPT-6 · 2026-09-23

- Changed only `apps/src-tauri/src/file_organizer.rs`,
  `apps/intake-engine/src/routing.rs`, and `apps/intake-engine/src/http.rs`.
  Hosted `analyze_directory` no longer invokes the recursive duplicate finder;
  the desktop behavior is unchanged. Hosted Smart Suggestions rejects the
  OpenList/B2 roots, allows at most 400 direct entries, and gives listing five
  seconds. Its request task aborts on client disconnect or an overall 60-second
  HTTP deadline. The explicit duplicate command remains separate.
- Built the modified source in an isolated Linux Docker context on `ovh-files`.
  `cargo build --release --bin intake-engine` passed with the donor's existing
  ten warning class; final runtime image
  `sha256:ebb62ad2fe15424465a3fcf0b8e438a7d2cc0b841d7a55e88bdd9758ec36b0e2`.
  A Linux `cargo test --release --locked --bin intake-engine analysis_` run
  passed all three focused tests: direct-entry cap, client disconnect releasing
  capacity, and command timeout releasing capacity. A test-only unused-permit
  warning was corrected after this run; the final runtime source compiled.
- An isolated loopback container used tmpfs state and a synthetic two-file
  `/srv/openlist/smoke` directory, with secret files mounted read-only. It
  returned unauthenticated `/healthz` HTTP 200. A request to its storage root
  returned HTTP 422 in 3 ms without starting analysis. Name search returned two
  hits and no search-leg errors. The bounded two-file Smart Suggestions request
  returned a model-backed result (`portkey:gemini-3.8-flash`), one suggestion,
  `duplicate_summary=null`, in 12 seconds. A separate request deliberately
  disconnected after one second; `/healthz` immediately remained HTTP 200.
  The isolated container was stopped and retained, not deleted.
- The live Coolify container was not involved in these tests and still reported
  `running:healthy`, unauthenticated `/healthz` HTTP 200. This correction is
  green for a normal branch push and Coolify-managed redeployment, subject to
  exact-file diff and fresh fast-forward checks. It is not yet live acceptance.
