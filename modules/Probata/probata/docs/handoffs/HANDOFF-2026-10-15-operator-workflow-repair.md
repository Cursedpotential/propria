# Operator workflow repair — 2026-10-15

Byline: Codex · GPT-6 · parent chat `01a13dcc-794a-7c20-a945-52a772e09a7e`.

Status: repair source integrated through `51ec88886`; release verification in progress. No repair deployment or complete browser acceptance is claimed at this checkpoint.

## Authority and scope

The owner asked this chat to review `Probata_Repair_Agent_Handoff1.md` and dispatch subagents to complete it. The handoff SHA-256 is `2ec70507cc9866d288acaa836c8fb1fdca851e817e02222f6ff04a892c0f9fba`; the original remains at `F:/Users/matts/Downloads/Probata_Repair_Agent_Handoff1.md`. Its six packages cover operation identity/results, source inspection, extraction/results, attribution, navigation/search, and accessibility/live acceptance. Preserve its historical audit; this document is an implementation coordination receipt.

Root/module AGENTS, AGENT_MEMORY, `docs/PURPOSE.md`, the October 11 reconciliation and ownership ledger, and the current owner discussions were read. Current owner decisions in `Plan Propria remediation` retain originals and independent search, expose speaker correction from Sources (G16), and retain exact saved results/approvals with a new generation for re-extraction (G19). Context_Files remains stopped. No corpus processing, source repair, inferred classification, evidence promotion or historical mode backfill is authorized by a UI repair.

## Source preservation and deployment baseline

**Controlling owner correction, 2026-10-15:** AI chat content goes directly to Weaviate and must not be normalized or restored into PostgreSQL. PostgreSQL holds extracted events, timelines, entities and artifacts for that lane. Historical AI raw/normalized PostgreSQL receipt references do not establish a present content-storage requirement. Resolve AI searchable content and indexing status through its actual Weaviate/native-content lane, independently of derived result availability. The human-message PostgreSQL reader remains scoped to human messages. This direct owner correction controls the implementation; older architecture descriptions or audit wording cannot widen PostgreSQL content storage.

- Shared checkout: `main`, `3987e8b1f4d453ce0826af248ecf4b3072fc3fa8`; unrelated changes preserved. Its status and worktree inventory were captured in private `.reconciliation/operator-repair-20261015/baseline.json`.
- Fetched `origin/main`: `fdf23125991376de62a94af255e32030e3e8754d`.
- Coolify Workbench: `xjbuo6drbwjfby75lalk8bk7`, ovh-app, branch `codex/messenger-extraction-web-20261012`, pin `bce348897e00f1bae71482fbd31b7b7f3b3b57b8`.
- Coolify starter: `r1084s1lsm80fsv4ol9ocij0`, ovh-files, branch `codex/extraction-recovery-starter-20261012`, pin `58419956e369041a071a0ac7fd1bef95af813ebd`.
- Main did not contain substantial deployed media, analysis and extraction-recovery source. Two conflict-free preservation merges produced integration base `7a5145a52080a139b1f3b45409384e8b4144c689`. These merges preserve existing deployed behavior; they are not new repair acceptance.
- Integration branch: `codex/operator-repair-integration-20261015`, in `_worktrees/operator-repair-integration-20261015`. All builder worktrees were created from fetched main, inspected clean, and fast-forwarded to that preservation base.

## Confirmed causes and competing explanations

| Audit reproduction | Confirmed cause/evidence | Alternative evaluated | Repair boundary |
|---|---|---|---|
| F02 completed legacy import | Activity exposes operation status while Review correctly requires durable mode. Deployed audited `RepairBindingAdmission` appends an actor/reason/source-bound receipt; it is not present on original main. | Browser mode alone cannot establish historical authority. | Expose eligibility before actions; preserve supported audited remediation and mode guard. |
| F03 terminal runs | Snapshot lifecycle can be terminal while phase/checkpoints retain an old wait. BFF/UI actions use phase; decision handlers can persist before validating terminal state. | A stale gate is not a running workflow. | Shared terminal eligibility and pre-persistence rejection. |
| F20 AI content/results | Historical result readers derive generation pins from a PostgreSQL preview snapshot; the frontend gates fetching on later context completion. Those contracts do not represent the current AI content route. | Missing PostgreSQL normalized chat rows is not an AI content failure: the owner directs AI content to Weaviate. Current Weaviate receipt/object availability must be independently read. | Route AI content/status through existing native/Weaviate references. PostgreSQL holds extracted events, timelines, entities and artifacts; do not restore raw/normalized AI chat bodies there. |
| F04 fingerprint | `context.hash_receipt` already records source version/digest/construction/time/receipt; browser operator snapshot hardcodes fingerprint unavailable. | ETag and retained-object SHA are different contracts. | Project exact source fingerprint with provenance. |
| F01 analysis 404 | Exact deployed Workbench has a proxy and reader; exact deployed starter lacks `/reference-import/analysis/projections`. A read-only request from live Workbench returned 404. | No evidence supports an empty snapshot catalog or transient loading. | Return supported capability unavailable; do not construct a new analysis runtime. |
| F21/F25 extraction | Dispatch does not preflight exact target authority. Failure before extractor output rows leaves a Temporal parent but no `working.extraction_run`, and UI loses the parent on unmount. | Zero output does not establish no attempts. | Preflight/recheck at dispatch; bounded durable attempt discovery independent of result rows. |
| F22 zero counts | Missing content defaults to zero; graph/relationship counts are hardcoded zero. | Unqueried/unavailable is not a successful empty query. | Nullable counts and explicit availability. |
| F24 attribution | Imported direction uses person/legal role, and absent sender falls back to first participant. | Recipient order and legal role cannot establish source speaker or direction. | Preserve raw roles and explicit source perspective; unknown stays unknown; expose existing editor scopes. |
| F23 overlapping counts | Sender and counterparty queries independently filter the same catalog rows. | Overlap does not prove duplicate storage. | Explain non-additivity; count unique identity union/intersection where available. |
| F15 media | Five sampled provider objects have signatures inconsistent with their media labels. First JPEG full/range prefix is identical and full MD5 matches ETag. | MIME/range repair cannot make those sampled bytes a JPEG. Corruption origin, earlier good versions, and original-package recovery remain unverified. | Preserve original versions; improve errors and exact-version delivery. Recovery belongs the existing source recovery lane. |
| F17/F18/F13 navigation | Component-only selection is lost on remount; folder label selects processing while chevron browses; content search leaves an old Process target armed. | No backend processing fault is needed to explain these UI target mismatches. | Exact target state, distinct browse/select, navigation restoration, search isolation. |

Source-level findings were checked at the deployed pins above. Only the analysis upstream 404 and provider byte checks were newly exercised live during initial investigation. Other causal mappings are source-confirmed against the supplied browser audit and still require repaired live readback.

## Existing correction scopes

Identifier mapping edits update one global `registry.entity_alias` row and append `registry.identity_change` in the same transaction. They affect readers matching that alias, not just the selected message. Stored period metadata is not enforced by the Imported resolver. Source acquisition context appends a revision bound to request, source and immutable observation; saving does not rewrite historical normalized records. The existing separate processing action must remain explicit. Message context review does not currently provide a one-message speaker override. The repair must not describe any of these scopes inaccurately.

## Write ownership

| Agent | Worktree | Responsibility |
|---|---|---|
| `build_contracts` | `repair-contracts-20261015` | Exact operation result resolver, fingerprint projection, terminal mutation/action gates, checkpoint state and nullable operator counts. Sole writer of shared web types. |
| `build_media_navigation` | `repair-media-navigation-20261015` | Source delivery/version validation, media inspection, Sources target/navigation/search, accessible full names and source API client. |
| `build_results_attribution` | `repair-attribution-results-20261015` | Extraction preflight/history, explicit analysis availability, attribution inspection and existing editor impact/scope, sender/counterparty overlap. |
| Parent | `operator-repair-integration-20261015` | Integration, verification fixtures/receipts, deployment coordination, browser acceptance, this handoff and Docstore readback. |

Builders commit explicit paths in their own worktrees. Parent reviews and integrates. No builder deploys, launches a model job, changes production case data, or reprocesses original sources. Tests and browsers run on the VPS. Do not execute the old `audit.sh` default: it includes a hashing/TEST-processing action and is not a read-only browser check.

## Verification plan and current limits

Required regressions include legacy rejection before dispatch, terminal gates with zero writes, exact saved-generation resolution without preview, source-version fingerprint readback, media HEAD/GET/version/range consistency, unavailable-versus-zero, durable failed attempt reopen, and attribution scope/overlap.

Live mutation verification needs a separately configured disposable database and synthetic fixture set; DEV is not isolation. Synthetic examples must cover sent/received/group/unknown speaker, first-party/acquired-third-party and AI context, plus an unrelated control conversation. Known-decodable media must be separate from the failed case copies. Browser proof must cover 1363×936, 200% zoom, navigation, keyboard, citations, save/reopen and extraction bounds. Real screen-reader acceptance is outstanding and cannot be inferred from DOM selectors or unit tests.

Scoped preflight read Codex owner conversations, supported Docstore related updates, CNF, `.remember` and project memory. Memsearch responded but returned weak/unrelated matches. A bounded Claude-history query succeeded after the schema-inference query failed; its returned material was older and includes explicitly marked model/tool content, not fresh owner authority. No exhaustive history reconstruction is claimed. The legacy Karpathy skill name was not found in installed skill locations; available Think graph/systems/Socratic/debug and Hyperfocus guidance were read.

Implementation commits, focused test results, deployment pins, browser receipts and remaining failures will be appended as separate evidence states. Until the complete journey is demonstrated, this workflow is not accepted as repaired.

## Independent live readback, 2026-10-15

Authenticated HTTP baseline reproduced the canceled handle's `select_handler` action despite `lifecycle=cancelled`. The failed AI and XML handles returned null correlation and pending records. Legacy SMS still refused missing durable mode with HTTP 409. Private HTTP receipts retain the exact request paths and response summaries.

The AI operation has historical raw and normalized persistence receipts, respectively `6f3bf342-f1ce-4cfc-8197-09dec824e543` and `4b60f043-a838-4af5-a52c-e90414001d3f`. They reference raw generation `01a1141f-0490-7289-a35d-c8e22c8bed63` and normalized generation `8bb7c26f-0209-409b-b91c-60530ee9ae6b`. Read-only queries against current `platform` found neither exact generation row, nor generation rows for source version `01a1141c-b03a-7a07-ad05-9dd5632829e8`; the read identity had superuser/bypass-RLS privileges. **The initial inference that this is a PostgreSQL recovery problem is withdrawn following the owner's correction.** AI content belongs in Weaviate; the absence of PostgreSQL chat rows is not an availability failure for that lane. These historical references remain audit evidence only. Actual AI availability must be checked through native-content/indexing receipts and Weaviate. This task has performed no AI PostgreSQL restoration, normalization or corpus processing.

Synthetic media was generated in the VPS devbox: PNG, GIF, one-second H.264 MP4 and one-second AAC audio MP4. Independent ffprobe checks confirmed video and audio stream types/duration. A first shell command's final audio argument contained CRLF and failed; the separate corrected audio probe exited zero. Both receipts are retained. No synthetic media has yet been processed or promoted.

## Integrated implementation and verification

The integration branch now preserves the two deployed application pins and contains coherent repairs for version-checked content delivery, inspection/navigation, extraction admission/history, attribution inspection, and operation result/terminal contracts. Parent review additionally found and fixed exact-version history joining, explicit reselection after Back, and retry attempt retention. AI format discrimination happens before raw/normalized PostgreSQL generation queries. The historical AI handle truthfully reports a Weaviate target with index availability not queried; no native workflow mapping was invented.

Independent PostgreSQL verification used newly created, empty task databases `backend_package_review_operator_results_20261015` and `backend_package_review_operator_contracts_20261015`. Tests run inside rollback transactions, never against the production case. Two results SQL tests passed, including Unicode and long-message byte accounting and unrelated control preservation. The durable admission/operation readback test passed on its second run; the first compile failure is retained.

The deterministic extraction/revision proof also passed, including an independent parent rerun against the combined source. The actual Python Semantica adapter processed four synthetic messages into five people, one place, two organizations and one event (output SHA-256 `e6bf98c76f54cb696bd45a82150863144a65a5f8c9cd8d7b43899ad7494977d4`). Actual Go store methods retained eight entities, eight mentions and one event with exact generation citations. Replay added no duplicates; comparison-only output stayed outside review. Actual alias edits appended one audited revision, source context reopened revision two, an unrelated control and historical normalized records were unchanged, and cross-request writes were refused. The transaction rolled back and the dedicated database remained empty. This is adapter/store proof, not a complete Temporal/browser mutation journey.

At integration commit `9403d28c9`, the parent VPS run passed 245 focused API tests, the production TypeScript/Vite build, full lint with zero errors and 24 existing refresh warnings, and Storybook. The first full smoke run passed 194, failed one outdated source-pattern assertion and skipped four browser tests; a supplemental reader test had four outdated module-boundary failures. Parent commit `96de3a75a` updates those tests for the shared search contract and terminal gate, with all 17 relevant checks passing. These are test results, not browser acceptance.

The current Go worker deployment is `d7ada57f86092193630a3e6e930c30a277f937bc` (finished deployment `94nnaat6ir0qoqd2hwcgnxe8`). Its request contract already accepts pinned `Runs`; additive conversation-history metadata is ignored safely by its JSON decoder. The standalone admission guard now rejects extraction and commit before dispatch when the deployed worker would lack its required published preview; saved artifacts remain readable. No worker deployment is included in this repair.

The combined source through `cf3ac1c59` passed all five scoped Go packages and 251 API tests on the VPS. The API run reported two existing deprecation warnings. Final web verification follows the last media retry and workload-availability changes.

Commit `51ec88886` adds explicit preflight workload coverage to the token-bound selection: `not_queried`, `partial` and `available`, with queried/resolved generation counts. Actual database checks proved four messages with full coverage, AI/no-generation targets unqueried, and four observed messages with one of two generations queried. Partial coverage changes the confirmation token. The UI reports unknown workload or an observed count with unknown remainder; only a successful complete query can establish zero messages. Focused Go, 18 web tests, changed-file lint and production build passed before integration.

Actual repaired source-content routing, service and media component were exercised with a separate read-only synthetic provider. PNG/GIF decoded to 160 by 90; one-second H.264 video and AAC audio reached ready state four and advanced during playback. Full HTTP 200 and byte-range HTTP 206 responses matched all four fixture hashes. An injected provider identity mismatch returned 502; the real rendered Retry handler issued a fresh 200 request with unchanged source/version/ETag/length pins and recovered the image. The browser check found and drove commit `780b067057f5273f553ec78b0fc84df4af0c0529`, which bypasses cached failures on retry. Actual Chrome page zoom at 200 percent changed the CSS viewport from 1363 by 936 to 681 by 468 without horizontal overflow. Keyboard Tab reached Retry. The parent inspected the zoomed screenshot. These are isolated real-browser component checks; production authentication/B2 access, the full live operator journey and screen-reader acceptance are separate.

Known AI declarations are rejected at legacy single/batch starts before workflow or binding writes and directed to native context ingestion. Search links use verified corpus/handle metadata; generic native excerpts do not receive invented human-reader links. Generic JSON filename detection remains non-authoritative: universal automatic recognition of every unknown AI export or legacy decoder is not established, and unknown exports require explicit AI selection.

The owner correction was saved through governed Docstore tools as `document:mu6dmz1rno8c828zod9e` and independently read back. It is a correction receipt, not a claim that all implementation files have been indexed.

## Verification incident

An overly broad devbox process inspection exposed an existing service credential in tool output. The owner was informed immediately. No credential value is retained in this handoff or its private incident note. Treat it as compromised; supported credential rotation remains required. No rotation or devbox redeployment has occurred while verification jobs are active. Further diagnostics are restricted to selected safe fields and task-owned processes.
