# Owner decision — Propria is the monorepo root

Date: 2026-09-12

Authority: owner decision

Status: active

Priority: critical

`E:\AI_Workspace\Projects\Propria` is the top of the Propria monorepo. Owned
applications and product modules belong under `projects/`; shared and reusable
source belongs under `resources/`.

This supersedes only the repository-topology sentence in the earlier naming and
router documents that described Propria as a brand/router and “not a repository.”
It does not change the Propria umbrella name, product names, database identities,
service identities, or historical receipts.

The transition is staged because the current workspace contains multiple dirty,
divergent, and nested Git repositories. No child `.git` directory may simply be
discarded, and no dirty working tree may be flattened in place. Each owned lane
must be imported with history and local-work proof. Third-party forks remain
external only when their upstream lifecycle warrants it; donors and permanently
diverged owned product code should be imported with license/provenance records.

Large corpora, evidence bytes, databases, secrets, generated indexes, caches,
temporary files, worktrees, and runtime state are not monorepo source.
