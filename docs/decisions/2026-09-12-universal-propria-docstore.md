# Owner decision — Docstore is the universal Propria documentation plane

Date: 2026-09-12

Authority: owner decision

Status: active

Priority: critical

Probata hosts the current Docstore implementation, but that hosting location does
not limit its scope to Probata. Docstore is the universal semantic-search, recall,
note, decision and documentation-resource plane for the entire Propria monorepo.
Its capabilities must be available to every Propria project and every supported
agent environment.

All project documentation will be migrated into the Docstore index over time.
Migration means registering and indexing the owning source documents with stable
identity, path, provenance, revision history and authority. It does not require
moving every source file into Probata's repository, and it does not transfer
product ownership to Docstore.

The required first-class capability surface includes semantic and structured
search, bounded recall, resource retrieval, governed note and decision writes,
revision-safe updates, priority/authority/status flags, approved-revision state,
relationship and graph inspection, and explicit source-freshness/CocoIndex CDC
verification. Results use the shared compact-cleaning contract while retaining
full provenance and diagnostic drill-down.

All agents query Docstore for related current decisions before modifying project
documentation or adding a note. Notes and decisions are persisted through the
governed Docstore tools and read back; chat-only statements and unindexed local
files are not durable project memory. Tools, skills and resources must be
discoverable without requiring an agent to know an internal script path.

This decision expands Docstore's corpus and consumers, not its runtime identity.
Docstore remains isolated from CCC's project-local code indexes and from Intake's
filesystem reconstruction index. Shared access or federation must not share or
silently repoint credentials, tracking databases, worker locks, target tables or
write authority.
