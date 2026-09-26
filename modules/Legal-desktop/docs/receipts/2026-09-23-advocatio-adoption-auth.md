# Advocatio adoption-auth change receipt

**Date:** 2026-09-23
**Repository:** `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`
**Base:** `master@2c6c0007ed6328d94ad4f7b2cc9bec222317c1af`
**Status:** recovery implementation verified locally on `wip/advocatio-adoption-auth-20260923`; deployment pending integration

## Reproduction

- `POST /v1/reviews` authenticates the request globally but discards
  `request.state.auth`; it forwards only the caller-controlled `ReviewCreate`.
- `Workspace.add_review` authorizes a consequential decision solely with
  `created.reviewer == "owner"`, then persists the supplied string as the actor.
- The same middleware admits Authentik humans, signed BFF requests, the MCP
  gateway service token, direct-tailnet access, and the explicit test bypass.
  Therefore a service/agent-facing credential can submit `reviewer: "owner"`.
- `Workspace.import_package` persists an accepted package without comparing its
  matter to the workspace matter. The consumer also accepts an arbitrary manifest
  hash and schema version. These are package-consumer gaps, not producer proof.
- `Workspace.home` is GET-backed and may persist a first-remote-matter projection.
  This receipt does not authorize changing that separate projection flow unless a
  focused compatibility test proves it is the same adoption mutation.

## Claimed files

- `api/legal_workspace/api/auth.py` — classify the already-authenticated principal
  for action authorization without weakening JWT issuer/audience/signature/lifetime.
- `api/legal_workspace/api/main.py` — bind review/adoption to `request.state.auth`;
  keep package import available to authenticated producer/service delivery.
- `api/legal_workspace/services/workspace.py` — enforce local matter and supported
  package identity/digest/version before import; derive the review actor from the
  verified eligible-human principal passed by the route; deny before state writes.
- `api/legal_workspace/services/source_package.py` and/or
  `api/legal_workspace/contracts/source_package.py` only if the existing consumer
  validation cannot express the required package digest/version checks locally.
- `tests/test_review_release.py` and `tests/test_source_package.py` (or one new
  focused adoption-auth test module) — human/service/agent/forged-body and exact
  package/matter/digest/version positive/negative coverage, including state equality
  after every denial.
- This receipt, updated with the exact diff, tests, and rollback after verification.

No root/shared manifests, D08 producer code, shared schema, migration, credentials,
deployment, or unrelated files are claimed.

## Intended invariant

Package delivery may make a validated exact package available, including through an
authenticated service identity. Availability never means reliance, legal adoption,
approval, release, filing, service, or transmission. Consequential review/adoption
requires an eligible authenticated human principal. The recorded actor is derived
from that principal and never from `reviewer` in the request body. Service, gateway,
BFF, bypass, and agent identities fail closed for this action. Denial occurs before
any workspace, event, review, or work-product mutation.

JWT verification continues to require Authentik RS256 signature, issuer, audience,
`exp`, `iat`, and `sub`. Package import continues to require approved items and will
add consumer-side local matter, supported schema version, package ID, manifest
SHA-256, item version, and content SHA-256 validation without claiming D08 producer
signature or promotion completeness.

## Planned proof

1. Reproduce the forged `reviewer: "owner"` acceptance with a service identity.
2. Add eligible-human positive coverage and service/agent/BFF/body-forgery denials.
3. Add wrong matter, package linkage, manifest digest, schema/item version denials.
4. Snapshot serialized workspace state and event-log bytes before each denied action
   and prove they are unchanged afterward.
5. Run focused auth/package/review tests, then the relevant Python suite.
6. Inspect and commit only the claimed tracked paths; do not push.

## Checkpoint — stopped for Probata/Intake P0

**Stopped:** 2026-09-23 07:51 EDT
**Status:** partial, uncommitted, not deployed
**Byline:** Codex · GPT-5 · independent verification checkpoint

The owner reprioritized Probata and Intake to P0. All Legal-desktop mutation,
testing, staging, commit, push, deployment, and credential work stopped. No
application file was changed by the independent verification pass. The existing
application/test diff and pre-existing `.cnf/` directory remain preserved.

Verification completed before the stop:

- Confirmed Git root `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`,
  branch `master`, base/remote head `2c6c000`.
- Read the repository `AGENTS.md`, this receipt, the safe-operations skill, and
  the handoff protocol. Literal D01/D09 packet paths were not found in the
  repository, Propria tree, or synced ChatGPT project sources; exact paths remain
  required before resumption.
- `git diff --check` reported no whitespace errors (only Git's existing
  LF-to-CRLF worktree warnings).
- Direct interpreter proof: `.venv/Scripts/python.exe`, Python 3.14.6,
  pytest 8.4.2. This avoids treating a broken `uv` trampoline as a test failure.
- Focused command: `python -m pytest -q tests/test_auth.py
  tests/test_source_package.py tests/test_review_release.py
  tests/test_first_slice.py tests/test_draft_edit.py`.
- Result: **22 passed, 8 failed, 1 warning**. One failure is an isolation/schema
  problem: the new auth test touches the global workspace and encounters the
  pre-existing SQLite schema missing `legal_core_matter_ref.last_agno_verify`.
  Seven failures are test-proof defects: model dumps regenerate timestamped
  structural defaults, so they do not prove denied calls changed persisted
  bytes. These are not evidence that the denial path wrote state.

Unresolved correctness risks:

- The stopped diff does not implement the receipt's promised consumer-side
  package import checks. `Workspace.import_package` still accepts an approved-item
  package without first rejecting a wrong local matter, unsupported schema,
  malformed/nil package identity, malformed manifest SHA-256, invalid assertion
  version, or malformed item content SHA-256.
- The denied-call tests must snapshot the actual persistence/event files (and
  the SQLite store where applicable), not compare unstable reconstructed models.
- The Authentik-human positive HTTP test must use an isolated temporary
  `Workspace`; it must not read or mutate the developer's real workspace store.
- The current actor gate correctly rejects non-Authentik sources before review
  writes and derives the stored actor from the Authentik subject, but the full
  focused suite is red and this has not been accepted as release proof.
- No broader suite, Ruff run, commit, push, deployment, or live verification was
  performed. D08 producer schema/signature/promotion and release remain HOLD.

Resume by locating and reading the exact D01/D09 packets, isolating the HTTP
test workspace, replacing model comparisons with persistence-byte snapshots,
implementing only the bounded consumer validation above, then rerunning focused
and broader Python tests. Do not stage or commit until every required test passes.

## Recovery implementation and verification — 2026-09-23

**Byline:** Codex · GPT-6 · adoption-auth recovery lane
**Base:** `master@2c6c0007ed6328d94ad4f7b2cc9bec222317c1af`
**Review branch:** `wip/advocatio-adoption-auth-20260923`

The original ten-file diff was preserved and reviewed against D01's interim
security checkpoint and D09's static reconciliation, receipts, and handoff under
`E:/AI_Workspace/Projects/Propria/.reconciliation/2026-09-23-orchestration-input/agents/`.
D01 requires a verified human action principal. D09 separates package delivery
from legal adoption and identifies the forged-body review gate. The current
branch binds review to an Authentik principal in the configured review group,
derives the persisted reviewer from its subject, and keeps service, agent,
signed-BFF, tailnet, and test-bypass identities ineligible for review. The
caller-provided `reviewer` field is compatibility input only.

The consumer now checks local matter equality, schema `1.0`, non-nil package
UUID, 64-hex SHA-256 manifest and item content-hash syntax, and positive
assertion versions before an import write. A package with no approved items
returns a blocked result without appending a debug event. The import route
maps consumer validation errors to HTTP 409; malformed UUID input is rejected
by request validation with HTTP 422. This is envelope validation only: the
current producer contract does not provide a verified signature or canonical
manifest recomputation algorithm here, and this branch does not claim either.

**Changed paths:** `api/legal_workspace/api/{auth,main}.py`,
`api/legal_workspace/config.py`, `api/legal_workspace/domain/review.py`,
`api/legal_workspace/services/{source_package,workspace}.py`, and
`tests/{test_api,test_auth,test_draft_edit,test_exhibits,test_filing,test_first_slice,test_persist,test_review_release,test_source_package}.py`.
No D08 producer, shared schema, portal, Authentik, Tailscale-services, or
deployment surface changed. The pre-existing `.cnf/` tree and this receipt's
backup were untouched.

**Verification:** the focused auth/package/review/first-slice/edit suite passed
with 36 tests after recovery; the expanded focused set passed with 45 tests.
The full `.venv/Scripts/python.exe -m pytest -q` run passed with **177 passed,
1 xfailed, 1 pytest-asyncio deprecation warning**. Negative tests compare the
actual SQLite/debug/event file bytes before and after denied operations, while
the authenticated HTTP tests use an isolated temporary workspace. Scoped mypy
on `auth.py`, `source_package.py`, and `review.py` reported no issues. A broader
mypy run found three existing errors at unchanged `workspace.py` lines 215,
696, and 1181. Default Ruff on touched files remains red on existing import
ordering, scheduler exception handlers, deferred router imports, and long
lines; the newly added validation and tests pass behavior checks. `git diff
--check` reported no whitespace errors. These local tests do not prove a live
deployment, producer signature, manifest integrity, source bytes, revocation
propagation, or release readiness.

**Deployment source and rollback:** the checked-in deployment history records
the Coolify `legal-workspace` application as sourced from GitHub `master`;
the current live configuration was not independently read back. The review
branch is not an accepted deployment target, so no deployment is performed.
After integration, a faulty commit can be reverted on the owning branch with
an ordinary `git revert` and redeployed from its configured source; no reset,
force push, or deletion is required. The earlier "do not push" checkpoint was
superseded by the owner's later explicit push authorization for this lane.

## Independent HOLD remediation candidate — 2026-09-23

**Byline:** Codex · GPT-6 · Tailnet and package-consumer remediation lane
**Parent candidate:** `wip/advocatio-adoption-auth-20260923@7e3fcd68e3351be91647c883020744a06ce92c23`
**Branch:** `fix/advocatio-tailnet-manifest-20260923`
**Scope:** API auth ordering, fail-closed package inspection, focused tests, and this receipt.

The independent review found that the browser `ReviewForm` calls the same-origin
Next bridge, which sends an ordinary signed BFF request when no Authentik JWT is
present. A BFF signature proves the bridge, not the person using it. The API
therefore still denies review by `signed-bff`, service credentials, direct Tailnet
device access, and caller-supplied Tailscale identity headers. The review route
gives a specific `Tailnet human attestation required` error for a direct Tailnet
device principal. Any presented bearer credential is now processed before the
Tailnet socket fallback. An invalid Authentik JWT on a Tailnet peer returns 401;
a valid, eligible Authentik JWT keeps its verified human subject and can review.
No second login is added to the Tailnet path, but Tailnet-only review remains
held until a verified human principal can be supplied.

### Claude ingress handoff: no-login Tailnet human identity

The current checked-in Next bridge and deployment material do not establish a
trusted identity path from Tailscale Serve into the web container. Tailscale
documents `Tailscale-User-Login` as a Serve-added user identity header, stripped
and replaced by Serve, absent for tagged devices, and also present for shared
external users. It explicitly warns that a backend reachable outside Serve can
accept forged values. See [Tailscale Serve identity headers](https://tailscale.com/docs/features/tailscale-serve).

Before enabling Tailnet review, Claude's portal/Tailscale-services lane must
prove the actual deployed ingress chain and isolate a Serve-only endpoint from
direct LAN, Tailnet IP, public Traefik, container-network, and other local
callers. The human identity must originate at Serve's verified request context,
reach the Next bridge only across a protected trust path, and bind to the exact
request that the BFF forwards to the API. The bridge and API then need an
explicit owner/eligible-user mapping, including treatment of shared users and
tagged devices, and a stable audit subject. Raw `Tailscale-User-Login`,
`X-Forwarded-*`, host, BFF signature, or Tailnet source IP alone must not become
that subject. This candidate intentionally specifies no new header/signature
format or production secret because no deployed ingress contract was verified.
Claude should provide the route/config/readback and a forged-header/direct-access
negative proof before a later implementation can enable Tailnet human review.

### D08 producer evidence hold

The D08 checkpoint under the 2026-09-23 orchestration input labels the producer
issuer, exact schema/digest recipe, signed/current status receipt, and delivery
boundary unresolved. The current `LegalSourcePackage` contract carries an
unverified caller-supplied `manifest_hash`; no authoritative producer algorithm
or issuer was located in the inspected current Probata/Advocatio source. The
consumer retains envelope checks for matter, schema, IDs, and digest syntax, and
the inspection result still reports omitted unapproved item IDs. Every package
containing `APPROVED` items now returns `blocked=true`, zero accepted items, and
an explicit D08 reason before any workspace or event write. A changed payload
with another valid-looking SHA-256 is denied with byte-identical persisted state.
This hold is not a claim that the supplied digest is false; it is a statement
that its value and the claimed approval cannot be independently verified yet.

The D08 owner must supply an authoritative versioned producer contract, exact
canonical bytes and SHA-256 computation, issuer trust root/signature or an
independent authenticated readback, current approval/revocation status, and
fixtures for tampered payload, wrong issuer/matter/version/span, replay, and
revocation. Only then can a consumer verifier replace this block. Synthetic
downstream unit tests seed fixture workspace state directly and never use that
helper in the production import path; HTTP import tests assert the hold.

**Local verification:** `.venv/Scripts/python.exe -m pytest -q` completed with
181 passed, 1 expected failure, and one pytest-asyncio deprecation warning.
Targeted Ruff passed for both production modules and the new fixture/auth/package
tests. System `mypy` passed on both changed production modules; the repository
virtual environment does not include mypy. `git diff --check` passed. The
official local Gitleaks 8.30.0 staged scan with `--redact=100` found no leaks;
the final outbound-commit scan follows the commit. These checks are local and
do not prove the deployed proxy contract or D08 producer authenticity.

**Rollback:** revert this branch commit if integration review rejects it; the
parent candidate and the pre-existing `.cnf/` and receipt backup remain intact.
