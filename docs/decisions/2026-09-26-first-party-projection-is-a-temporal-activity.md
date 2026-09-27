# First-party message projection becomes a Go Temporal activity — 2026-09-26

> _Byline: Claude Code · Opus 5 · 2026-09-26_

## Decision

First-party message projection is implemented as **Temporal activities in the Go engine**, not as
a repair of the Python `server/evidence/message_projection.py::_write_first_party`.

Owner rulings, 2026-09-26: the process is **extract → confirm → commit**; the **Go engine
orchestrates everything**; **everything has to go through Temporal**. A write path running outside
Temporal is a defect to move into the engine, not a lane to maintain.

## The gap this closes

Entity and event extraction already has the full pattern, and the normalized side has its own
commit activities. First-party message projection never got them — it is still a Python function
writing `working.*` outside Temporal entirely, and it writes to `working.conversation`, a table the
2026-09-07 schema snapshot deleted in favour of the five-table
`working.first_party_context_thread` / `_version` / `_message` / `_source` / `_realization_*` family.

The consequence for the owner is concrete: first-party context cannot be imported.

## Pattern to follow

- `modules/engine/activities/entity_extraction.go` — extract (`ProposeEntitiesRules`,
  `ExtractEntitiesEventsModel`) → confirm (`ReconcileEntityProposals`, `ValidateExtractionCommit`)
  → commit (`CommitEntities`, `CommitEntityAliases`, `CommitEntityMentions`,
  `CommitEventCandidates`, `CommitTimelineMembers`, `FinalizeExtractionCommit`).
- `modules/engine/activities/normalized_pipeline.go` — the package convention, stated in its own
  header: an activity does compute and validation over Store-provided streams, while the PostgreSQL
  Store in `engine/postgres` owns every SQL transaction and idempotency coordinate. **Exactly one
  activity owns each write.** Writes are fail-closed on the guard triggers — never bypassed, no
  partial seal, no publish without a receipt.

## Constraints carried into the implementation

- `working.message.id` must equal the corresponding `working.normalized_record.id`. The current
  Python writer generates a fresh uuid4; that is a real bug independent of where the code lives.
- Pre-launch identity uses the D-126 dev sentinels — matter `deadbeef-dead-beef-dead-beefdeadbeef`,
  court case `cafebabe-cafe-babe-cafe-babecafebabe` — seeded by
  `sql/0069_dev_case_registry_identity.sql` and documented in
  `modules/engine/postgres/proffer_schema_probe.go`. Identity checks stay fully on; only which
  constants they match changes. No column type widens.
- Missing identity fails loudly. An owner, matter, court case or thread identity is never invented,
  defaulted or derived — fabricated provenance on evidence is worse than a failed import.
- Validation is live against a disposable schema on ovh-files built from the current snapshot,
  reading rows back and dropping the schema afterwards. Never mocked; test data never becomes
  canonical.

## How this decision was nearly recorded wrong

The first brief for this work was written from the 2026-09-24 D04 hold receipt and pointed at the
Python writer. That receipt described the code as of its own date and did not establish which lane
owns the work now; the correct pattern was already in the repo one grep away. Recorded as a standing
dispatch rule in `Propria/AGENTS.md` ("Dispatching agents"): find the nearest working sibling
implementation before briefing a fix, name it in the brief, and use a read-only explorer agent for
architectural questions before dispatching a builder.

Supersedes the repair framing in `docs/reviews/2026-09-24-d04-first-party-projection-schema-hold.md`.
