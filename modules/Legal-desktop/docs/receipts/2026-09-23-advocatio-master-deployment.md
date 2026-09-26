# Advocatio adoption authorization merge and deployment receipt

**Byline:** Codex · GPT-6 · independent integration reviewer
**Date:** 2026-09-23 (America/New_York)
**Repository:** `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`
**Remote:** `https://github.com/Cursedpotential/Legal-Workspace.git`
**Result:** merged and deployed with a public Authentik ingress hold

## Reviewed source and integration

- Reviewed the adoption-auth commit `7e3fcd68e3351be91647c883020744a06ce92c23` and the independent fail-closed remediation commit `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34` against D01, D08, and D09's 2026-09-23 packets and the adoption-auth change receipt.
- Fresh `origin/master` was `2c6c0007ed6328d94ad4f7b2cc9bec222317c1af`, an ancestor of the remediation head. Local `master` fast-forwarded through both commits. A normal push set `refs/heads/master` to exactly `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34`; no divergence, reset, force push, stash, clean, broad staging, or deletion occurred.
- The pre-existing untracked `.cnf/` directory and `docs/receipts/2026-09-23-advocatio-adoption-auth.md.backup_20260923_075106` were preserved and excluded from the integration.

## Independent verification before merge

- Full `.venv/Scripts/python.exe -m pytest -q`: **181 passed, 1 xfailed, 1 pytest-asyncio deprecation warning**.
- Targeted Ruff passed on the changed authentication/package production modules and focused fixtures/tests. Scoped system mypy passed on `api/legal_workspace/api/auth.py` and `api/legal_workspace/services/source_package.py`. `git diff --check 2c6c000..7a26f8a` passed.
- Official Gitleaks **8.30.0** ran with `git --log-opts='2c6c000..7a26f8a' --redact=100`: two commits scanned, **no leaks found**.
- Source inspection and tests show that a presented bearer is evaluated before Tailnet fallback; a bad bearer does not become a device principal. Only a verified Authentik principal in `AUTHENTIK_REVIEW_GROUPS` can record review, with the actor derived from its subject. Direct Tailnet, signed BFF, service, agent, test-bypass, and forged identity headers remain review-ineligible. A package with approved items is held before import persistence because D08 has not supplied an authoritative issuer, manifest recipe, and current status proof.

## Coolify application and live result

- ContextForge's Coolify control plane identified `legal-workspace`, UUID `gvghzivfmctev8dloetfssnj`, on `ovh-app`, from `Cursedpotential/Legal-Workspace`, Git branch `master`, compose location `/compose.yaml`. The prior finished deployment was `rpp3j5kacklwvj615x995rr3` at commit `6c937bd214ae324beaf82ce9b8c540ed80bae907`.
- Normal Coolify deployment `haj8fv8uvbg696a0h184fr22` finished at **2026-09-23 19:58:35 UTC**. Per-application deployment readback names the exact new commit `7a26f8a33d1f3151d9d2d8f9d2b9b152a49fcc34`. The API image install and Next production build completed, and the new web container started.
- Live Tailnet `https://legal.tilapia-skilift.ts.net/` and `/review` returned **200**. Direct Tailnet API `/health` returned **200**. A review POST with a nonexistent section and no credential returned **403**; the same request with forged `Tailscale-User-Login` and `X-Forwarded-For` headers returned **403**. A presented invalid bearer returned **503**, with detail `AUTHENTIK_ISSUER is required`; it did not downgrade to a Tailnet principal. These probes used a nonexistent section and did not submit a real review or package.
- Coolify's resource state was `running:unknown` because its application-level health check is disabled. The endpoint checks above establish the observed web/API availability; they do not certify every service or workflow.

## Holds and rollback

- The planned public `legal.mitechconsult.com` hostname did not resolve during this review. The older `legal.int.mitechconsult.com` path is recorded in the 2026-09-20 portal preview as an Authentik 404. Coolify's Legal application has no public FQDN/router in its application record, and its current API lacks `AUTHENTIK_ISSUER`. Public Authentik login, redirect, and human review therefore remain **unverified/unavailable**. The portal/Authentik/Tailscale-services ingress owner must supply the route and verified JWT settings before a public review claim. No public routing was changed here.
- Tailnet-only human review remains intentionally held pending a verified Serve-to-bridge human identity chain and forged-header/direct-access negative proof. D08-approved package import remains intentionally held pending producer authenticity and current-status proof. Existing stored legal material was not migrated, released, or filed.
- If this deployment causes a separate operational regression, inspect the exact failing service and use Coolify's prior deployment `rpp3j5kacklwvj615x995rr3` only with an explicit compensating hold on review/import: that image predates the security fix. A durable source rollback would use ordinary `git revert` commits on `master`, normal push, and a Coolify redeploy; never reset or force push. Reverting these security commits alone would restore the known forged-review and unverified-package exposure, so the affected routes must be disabled or replaced with a safe fix first. No rollback was performed.
