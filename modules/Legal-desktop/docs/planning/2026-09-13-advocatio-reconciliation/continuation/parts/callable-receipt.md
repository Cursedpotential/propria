# Task A callable section receipt

- Completed: 96 of 96 assigned entries.
- Input: `inputs/stack/toolkit-capabilities.json`
- Input SHA-256: `7da5304dbfd94f2bfea263de4b242933a653457f06e424e57230a4eea110a7da`
- Completed source-ID list SHA-256 (assignment order): `35c4765abe85ab203a56527800c1d9407a1db6f31feec240f29c6f9c60728ebd`
- Output: `continuation/parts/callable.csv`
- Schema: exact `assignment.json` columns; identity, provenance, and observed implementation/wiring copied verbatim, with missing values empty.
- Coverage: every assigned source ID present once; no unassigned IDs included; rows retain original input index.
- Classification: each row has an allowed destination and integration approach plus distinct proposed route, human access, agent access, dependencies, missing information, verification, and next action.
- Checks run: 96 rows emitted; 96 unique source IDs; 96 unique input indices; all required fields nonempty; CSV header matches assignment schema; input SHA and assignment source-ID order checked before writing.
- Scope boundary: static integration planning only. No toolkit operation, app execution, dependency installation, server registration, database mutation, or application modification was performed.
- Ambiguities requiring coordinator review: live MCP/server reachability, protocol/auth parity, current service allocation, and host-specific runtime assumptions remain unresolved where stated per row.
- Next step: coordinator runs the shared merge/verifier and reviews unresolved/ambiguous classifications before accepting the final map.

## Coordinator review amendment

- MCP write/mixed handlers (`case_put`, `case_export`, `case_import`, `case_status`, `case_memo`, `case_evidence_log`, `case_eval`, `case_reference`) now require isolated synthetic mutation fixtures covering expected effect, invalid-scope rejection, stale-version/conflict behavior, before/after hashes, and explicit proof that no unverified source is accepted as Advocatio evidence.
- `court_language_review` now has four separate verification checks: original preservation, proposal labeling, no invented facts/citations, and scope/source/reviewer disposition retention, plus malformed-input and unavailable-dependency handling.
- MCP server inventory rows are classified as `operational_support` with `investigate`; no full connection is treated as proven. Registered-tool routes are explicitly conditional on Task F current-route, authentication, protocol, and capability-discovery proof.
- No operations were executed; these are planning-field corrections only.

Coordinator final reconciliation: court_language_review now explicitly uses individual accept/reject, stale-proposal checks and the four owner-accepted review dimensions. Final CSV SHA-256: `fadf66f18da17d2c9443eef1e78362220b93e9d5258f9bbe6a7f1530742857c4`.
