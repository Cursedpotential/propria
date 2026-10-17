---
scope: deploy/docker
status: current
verified_at: 2026-10-17
verification_scope: Source paths and owner-ruled roles at 68f369e; no image or service-health verification
superseded_by: null
authority:
  - AGENTS.md
  - deploy/docker/README.md
  - docs/REPO_STRUCTURE.md
watches:
  - deploy/docker/**/Dockerfile
  - "deploy/docker/**/*.yaml"
  - deploy/docker/README.md
contains_secrets: false
---

# Service Image Memory

> _Byline: Codex · GPT-5 · 2026-08-27._
> _Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17; the module/root
> `AGENTS.md` contracts and source tree read at `68f369e`._

Paths below are relative to `modules/Probata/probata/`; commits belong to the
Propria root. The old root `docker/` and compose routes are historical.

- One service image lives in one `deploy/docker/<service>/` boundary;
  deployment composition is under `deploy/`, including `deploy/compose.yaml`.
- Keep images purpose-specific and runtime contracts neutral. Do not move canonical authority into
  a convenience container.
- Milvus belongs to shared memsearch, while Weaviate is the project vector
  projection (2026-09-03 owner role ruling). This does not establish current
  service health or restore the retired platform Milvus/LanceDB roles.
- Community and vendored tool images require observed output tests; an image merely starting is not
  proof that extraction, metadata, or OCR behavior is correct.
- For n8n, also read `deploy/docker/n8n/AGENT_MEMORY.md`.
