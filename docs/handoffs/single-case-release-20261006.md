# Single-case Dev/Live release receipt

Byline: Codex orchestrator, 2026-10-06. Updated during the owner-authorized resumed release.

## Current outcome

PR #2 merged at **2026-10-06 11:42:06 UTC** (07:42 EDT):
`21b746fb0577b1ba73d118f6ed69418c7aa4247a`.
https://github.com/Cursedpotential/propria/pull/2

The merged source preserves upstream `cf83d7a7`. The concurrently dirty shared
checkout was not staged, reset, stashed or overwritten. Parent integration
was fast-forwarded to the verified merge without touching that checkout.

The narrow Workbench health repair also merged as PR #3 at
**2026-10-06 12:23:14 UTC**, merge `ade5bac9e696cf9d67a550c37e4856ebcf0d9de7`.
https://github.com/Cursedpotential/propria/pull/3

**Core source is merged, all six core manual deployments finished, and the
additional Workbench repair is deployed and strictly healthy. Runtime
acceptance remains incomplete:** the Python worker's authoritative network
check is blocked by the explicit security decision below, and actual browser
clicks remain unverified. Automatic deployment remains disabled. Neutral
matter/court environment values were provisioned for
Workbench, exec API and Python worker; the existing mounted service credential
was reused without rotation or a user-auth/public-ingress change.

## Verified source and independent review

- Workbench API: **729 passed**, zero failures, source-pinned VPS packet
  `d41be78f`; independently repeated in GitHub source run `37456823464`, final
  proof-document run `37457896309`, and merged-main run `37458156760` (green).
- Workbench web: lint zero errors/26 existing warnings; typecheck/build and
  Storybook pass. Browser-free smoke **144 passed/four browser-dependent skips**.
  These do not establish actual user-facing clicks.
- Parent Python scope/promotion/chunk suite: **399 passed/five existing skips**,
  zero failures/errors, 8.03 seconds. JUnit totals 404 collected.
  `modules/Probata/probata/to_be_deleted/single-case-python-raw-final-20261006.xml`.
  Ephemeral uv extras supply Temporal SDK/SQLAlchemy without source or lock edits.
- Engine: all **47 test-bearing packages** pass; `go build ./...` and
  `go vet ./...` also pass, Go 1.27.1, existing VPS devbox, source-only packet
  `engine-52e5240f`. No integration-test DB credentials or live writes.
  Initial whole-suite failure was an upstream stale optional-stage count; the
  test-only repair preserves prior assertions and explicitly verifies the
  twelfth stage. The only later engine difference is its byline comment.
- Legal consumer: freshly revalidated **27 passed in 0.51 seconds**, exact
  `d27e2b5a` source (Legal files unchanged through merge), existing VPS fixture
  `/root/single-case-flags-20261006/legal-consumer-d27e2b5a`. Locked uv project
  dependencies, synthetic upstream only. Local minimal-environment attempts
  failed before collection; they were not counted as a pass or assertions.
- Read-only independent reviewer accepted source `ad9caa46`: no blocking
  finding remains in the reviewed scope. Workbench/Python mandatory approval
  reads reject environment proxies and redirects, own elapsed deadlines and
  raw-response caps, reject encoded/ambiguous JSON responses, and correlate the
  actual court parent. API/CLI/synchronous Activities retain compatible guards.
  Invalid/padded durable receipt IDs deny without panic or pointer replacement.
  Reviewer's bare-environment helper run had 96 pass/four missing-SDK imports;
  parent's complete 399-case run is distinct evidence, not a waiver.

Synthetic HTTPX transport and SQL/replay fixtures are not live network,
production-history, database-commit or concurrency proof.

Engine archive SHA-256:
`6ea2d981555c213686d18a0e72e4ab3fc59ae1078413c1b554f34c3c179accd5`.
Test log SHA-256:
`2facc26346e0805a074940b308ce51bcb4e8bece209a99e2f37055c6d5c14b61`.
Logs are retained in parent `to_be_deleted/engine-proof-52e5240f-20261006/`
and the immutable VPS packet. Build/vet logs are empty with successful exits.

## Runtime rollout receipt

| Application | Coolify UUID | Deployment | Verification |
| --- | --- | --- | --- |
| Identity starter | r1084s1lsm80fsv4ol9ocij0 | t3fm8kmnhlggll1xghryrkv2 | Finished; container healthy; fresh scope/aggregate contract passes |
| Proffer Go worker | d24bb9eoo47qtw9eq1xc6u64 | kqgco76pkmgkho12xprq3xcn | Finished; running, zero restarts; both Temporal poller kinds fresh |
| Workbench | xjbuo6drbwjfby75lalk8bk7 | ois8ffz6ijjewu5e9adph1in (health repair); tqushmbge0l9rulx6f1ql4bz (core) | Both finished; new container strictly healthy; served identity/storage readiness pass |
| Legal | gvghzivfmctev8dloetfssnj | yetzp36gyojpirop354czwfp | Finished; API/office healthy; live read-only records pass |
| Exec API | rz41wqhpjfh1rj796ixvjhfs | thjpzwqwdufgsq5fvrd7lomw | Finished; served health 200; installed approval helper passes |
| Python worker | zjutdbcnru2waxu9uiyptqjb | emyhsccycacaugxey9oyar0r | Finished; installed helper returns safe 503; diagnosis pending |

Verification through 2026-10-06 12:26 UTC:

- Starter private authenticated scope and aggregate reads pass for default,
  Dev and Live, all with the same approved pair and actual court parent;
  default is Live and invalid mode is 422. No source bodies or canonical writes.
- Ordinary enrolled-desktop HTTPS Workbench identity reads return 200 for
  default/Dev/Live, same approved pair, actual parent, same two person IDs and
  available catalog; UNKNOWN returns 422. This is not complete source-ID-set or
  browser-click proof. Repeated after the storage-health deployment: all three
  modes still pass with the same pair and identical two-person ID set.
- Legal's actual tailnet API `/v1/probata/records?kind=entity` returns 200,
  available true, Live, approved pair and two records. `kind=event` returns 200,
  available true, Live, approved pair and zero records. The zero is an observed
  listing, not evidence that events exist. `/v1/auth/whoami` returns 200 through
  the existing enrolled-desktop door with no spoofed headers. An earlier guessed
  web `/api/v1/probata/records` 404 was not the actual route or a service failure.
- Exec API's newly installed helper makes a fresh authenticated upstream GET,
  approves Live for the exact neutral pair in 0.425 seconds and denies Dev
  admission. Installed helper SHA-256 matches local source:
  `68807f06eb33a5c70a151e09b71fd2f04aeb03e6947cd0f4ccfa42e147622fda`.
  This proves the installed helper, not a write route or actual job execution.
- Python worker's installed helper has the same verified source hash and a
  readable mounted token, but its fresh network approval returns safe 503,
  `Proffer authoritative case verification is unavailable`. Cause independently
  confirmed: upstream returns 401 in 0.216 seconds because direct socket peer
  172.25.0.27 is outside the scope route's 100.64.0.0/10 restriction. Tokens are
  byte-equal and read-only mounted. Host equivalent GET succeeds in 0.098 seconds.
  Starter tsnet is disabled; no active verified alternative service hostname
  exists. A narrow read-only internal service door requires owner approval;
  that question has been presented. The proposed route-only exception would
  admit the exact scope GET from the explicitly configured private network
  with the existing token. That is shared service-token possession, not unique
  worker identity, and never permits general write routes, aggregate case reads,
  public access or forwarded-header trust. No auth CIDR, host-network or token
  change has been made. A worker-specific credential/network is a separate
  design if the owner rejects this bounded shared-service trust.
- Workbench degraded readiness is a verified obsolete R2 health-adapter call,
  not an unverified B2 credential failure. The active B2 Casevault prefix lists
  successfully with MaxKeys=1. Narrow repair `c149596b` has 757 full API passes,
  46 focused passes and 28 new cases. Independent reviewer confirms runtime
  has exactly one prefixed B2 root, accepts source with no blocker, independently
  repeats 46 focused passes and verifies original full-suite JUnit has 757 tests,
  zero failures/errors/skips. Integrated b3e2dea1 differs only by attribution
  comments from original tested c149596b. PR #3 and merged-main CI are green
  (runs 37462621456 and 37462881926); 757 API passes and browser-free smoke
  144 passes/four browser-dependent skips. Manual health deployment finished.
  New container `workbench-xjbuo6drbwjfby75lalk8bk7-122408446484` is running,
  Docker healthy, zero restarts and not OOM-killed. Ordinary enrolled-desktop
  HTTPS `/health` returns HTTP 200, status ok, lancedb true, object_store true.
  Installed production/test SHA-256 matches integrated source exactly:
  `33d79ef6d89240308c04cdfa634e0f0b0f04efeaff2bc78e456e9645f2e6faf3`
  and `5b07688c56f98c59825e3b8c71f2f5db3172a853b5787d0633dc874f8b721079`.
  Image `sha256:c4c207aae7215a6a488d88e843693ab0342f49e49e8dcb954a715c69678ebc2c`.
  A guessed host port 8020 was not reachable; the actual served HTTPS door and
  container health are the acceptance evidence. Strict health was not weakened.
  Existing synchronous health/provider elapsed-budget debt is a follow-up.
- Legal/Python deployments waited on the shared Coolify high queue, then
  reserved and finished normally. Historical metadata supports contention with
  other scheduled jobs; no queue cancellation, purge or redispatch occurred.
- Independent live worker check at 12:18:52 UTC, repeated at 12:21:06 UTC:
  namespace default, queues proffer-v1 and evidence-pipeline both have fresh
  workflow and activity pollers matching the new container hostnames. Backlog
  is zero; both containers running with zero restarts and OOMKilled false.
  Neither has a healthcheck; fresh pollers are readiness evidence, not completed
  workflow execution or an alternative to the Python approval failure above.
  Six inspected Python source files hash-match merge21b. Go binary contains
  the operating registration and canonical-write guard symbols, but has no
  embedded commit/revision label; commit linkage relies on the deployment receipt.

Pre-rollout HTTP checks from the enrolled owner's current desktop returned
200 for Workbench, Legal, Legal API health and exec API health. These probe the
old deployment and do not prove the new source. Earlier old starter scope was
404/canonical modes 422, while its compatibility aggregate was 200 in 6.998s.

Actual read-only user-facing clicks remain an explicit owner-choice gate:
approve use of an already-open desktop browser or perform the check personally.
No desktop browser was launched and no boundary was forged or relaxed.

## Working checklist and next authority

- [x] Preserve concurrent upstream work; source integration and independent review.
- [x] Merge core repair; source-pinned Python/Go/Workbench/Legal gates.
- [x] Provision neutral IDs; manually deploy six affected applications.
- [x] Fresh same-case scope and served Dev/Live identity checks.
- [x] Legal live records, exec installed approval, both workers' current pollers.
- [x] Diagnose obsolete Workbench health; reviewed fix, green CI, merge and healthy deployment.
- [ ] Owner security decision for Python worker's read-only case-scope service door.
- [ ] Implement/test/review/deploy the approved resolution; repeat worker approval.
- [ ] Actual Case-page browser interaction check, with owner-approved access.
- [ ] Approve the bounded disposable-Dev design/persistence plan, then implement
      independent flags and frozen baseline/overlay lifecycle in separately owned slices.
- [ ] Future end-to-end write/job acceptance with approved input and custody scope;
      no live job or mutation was submitted as a readiness probe.

The disposable-Dev design is retained in
`docs/handoffs/disposable-dev-workspace-design-20261006.md`. It is proposed,
not implemented. Unchanged source or state is not a reason to repeat deployments.

The read-only architect's service-door proposal is ready but not authorized:
dedicated empty-default CIDR configuration, a transport wrapper on only the exact
scope GET, unchanged tailnet branch and shared overlayAuth, per-request mounted
token loading/constant-time comparison, rejection of broad/unrelated/noncanonical
ranges and forwarded identity. Exact GET-only method check excludes Go mux's
implicit HEAD matching. Verify the live bridge mask before choosing a subnet;
an observed worker /32 is narrower but breaks on address churn. Test denial for
every other method/route, bad/missing token, invalid config and outside peers,
then independently recheck worker scope and existing tailnet/write exclusions.
No registry/schema, worker code, general auth, tsnet or network change is proposed.

## Invariant and remaining product work

Dev and Live share one approved case, people and source identities. Fresh
omitted mode defaults Live; absent historical durable mode remains unknown.
Dev canonical writes stay denied before persistence/dispatch until disposable
data isolation exists. No alternate case is created or selected by mode.

**Disposable Dev workspace, reversible Dev-only data overlay/rollback and the
general independent feature-flag registry remain unimplemented owed features.**
This release does not roll shared Live data backward or erase concurrent writes.

Only this lane's unused generated empty Legal test environment was moved to
`modules/Legal-desktop/to_be_deleted/unused-empty-env-20261006`; it is recoverable.
No file was permanently deleted. Other agents' source/worktrees are retained.

The earlier source checkpoint was independently saved/read back in Docstore:
`document:dc1148waufaphiwra6rr`. The refreshed release receipt supersedes only
that named handoff, not every same-domain record. Governed handoff status is not
source approval or semantic-indexing proof. The critical owner decision remains
separately active and is not superseded by a release-progress handoff.

Current release handoff: `document:yad8l45rww40mteakjn6`, active across the six
release domains, independently retrieved with whitespace-normalized content
equality (the read adapter collapses Markdown whitespace, so this is not a
byte-for-byte formatting claim). The specific
prior source checkpoint above is superseded. Semantic indexing was not triggered.
