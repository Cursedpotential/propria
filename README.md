# Propria

Propria is the umbrella monorepo for the owner’s evidence, legal, investigation,
vault, filesystem-recovery, and shared application infrastructure.

The product and application-fork histories were imported on 2026-09-26. They
share one Git boundary at the Propria root.

| Path | Purpose |
|---|---|
| `modules/` | Products and their imported application forks |
| `docs/` | Cross-product decisions, architecture and dated migration receipts |
| `scripts/`, `plugins/` | Shared source |
| `design-contract/` | Shared design contract |
| `AGENTS.md` | Root contract and current module routing |

Start with [`AGENTS.md`](AGENTS.md), then read the selected module's local
instructions. Current import state is in
[`docs/monorepo-migration-manifest.json`](docs/monorepo-migration-manifest.json);
the [import decision](docs/decisions/2026-09-26-monorepo-import.md) preserves the
dated history. Shared development tools `memsearch`, `claude-context` and
`cocoindex-code` remain outside this repository under
`E:/AI_Workspace/plugins/forks/`.

Do not add corpus bytes, databases, secrets, generated indexes, caches, or runtime
state to Git.

_Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17, from the root
2026-09-26 import contract and Vestigia's 2026-10-02 router at `68f369e`._
