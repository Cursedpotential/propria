# Propria

Propria is the umbrella monorepo for the owner’s evidence, legal, investigation,
vault, filesystem-recovery, and shared application infrastructure.

The target source layout is:

```text
Propria/
├── projects/    owned applications and product modules
├── resources/   shared contracts, libraries, controlled vendors and references
├── docs/        cross-product decisions, architecture and migration receipts
├── AGENTS.md    repository and safety contract
└── CLAUDE.md    Claude entrypoint that imports the repository contract
```

The repository is in a staged migration. Existing `Consignatio/`, `Probata/`,
and nested Git repositories remain authoritative sources until each is imported,
verified, and marked complete in
[`docs/monorepo-migration-manifest.json`](docs/monorepo-migration-manifest.json).
Do not add corpus bytes, databases, secrets, generated indexes, caches, or runtime
state to Git.
