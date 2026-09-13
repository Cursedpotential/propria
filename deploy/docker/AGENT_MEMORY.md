---
scope: docker
status: current
verified_at: 2026-08-27
superseded_by: null
authority:
  - AGENTS.md
  - docker/README.md
  - docs/REPO_STRUCTURE.md
watches:
  - docker/**/Dockerfile
  - "docker/**/*.yaml"
  - docker/README.md
contains_secrets: false
---

# Service Image Memory

> _Byline: Codex · GPT-5 · 2026-08-27._

- One service image lives in one `docker/<service>/` boundary; deployment composition remains in
  `deploy/` or the production-facing root compose definition.
- Keep images purpose-specific and runtime contracts neutral. Do not move canonical authority into
  a convenience container.
- Do not restore retired Milvus or LanceDB paths through an old image or compatibility label.
- Community and vendored tool images require observed output tests; an image merely starting is not
  proof that extraction, metadata, or OCR behavior is correct.
- For n8n, also read `docker/n8n/AGENT_MEMORY.md`.
- `postgres/` (image `agno-postgres:18-duckdb`): pg_duckdb community extensions are enabled at
  image level (`duckdb.allow_community_extensions = on`, `deploy/docker/postgres/Dockerfile` CMD);
  `webbed` is in use for `read_xml`/`read_html`. Byline: Claude Code · Sonnet 5 · 2026-09-06.
  See `docs/reviews/2026-09-06-webbed-install.md`.

<!-- freshness
watches_hash: 3f06efe
last_verified: 2026-08-27
watches:
  - docker/**/Dockerfile
  - docker/**/*.yaml
  - docker/README.md
-->
