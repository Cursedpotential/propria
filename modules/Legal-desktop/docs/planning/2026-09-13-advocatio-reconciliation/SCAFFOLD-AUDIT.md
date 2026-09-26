# Existing application integration delta

This is an existing app with substantial concurrent changes. **No application scaffold, AGENTS replacement, package migration, service installation or data import was performed.** Creating a second skeleton would undermine the canonical-directory reconciliation.

| Existing area | Preserve | Planned delta |
|---|---|---|
| `web/src` Next routes/components | Current plain-language shell, source/work-product interactions, assistant context, auth/proxy behavior | Port incrementally into selected shared client architecture. New source/claim/proposal/method views consume versioned contract. F0 determines exact file move map before migration; retain source until parity. |
| `api/legal_workspace/domain` | Accepted-source model, strategy, templates, exhibits and review concepts | Add versioned resources/methods, atomic claims/support, proposals/findings/private record types; reconcile with existing structures rather than parallel duplicates. |
| `api/legal_workspace/services` | Legal workflow and provider boundaries | Atomic revisions, source/store/MCP adapters, method/run handling and derivative jobs. |
| `api/legal_workspace/db` | Existing store adapters as migration evidence | Correct transaction and revision behavior; establish real chosen persistent deployment, explicit migrations and recovery tests. |
| `tests` and `web/smoke` | Existing behavior tests/harnesses | Add isolated meaningful acceptance suites for failures, version conflicts, scope and support truthfulness. No private corpus fixture. |
| `config` | Existing role/tool/provider routing as audit source | Versioned capability/role registry and source settings; no secrets in client/config output. |
| `resources/build-kit` | Original handoffs, catalogs, donor versions and provenance | Remains reference archive; selected modules copied/ported only through an explicit integration manifest. |
| `resources/reference-review/2026-09-13` | Private intake index, source identity and bounded reader reports | Baseline/remote reconciliation inputs. No wholesale promotion, public copying or duplicate source authority. |
| Probata/Family Court sibling projects | Existing evidence, timeline fork and remote Toolkit work | Separately owned contracts/adapters in future phases. Same surface does not authorize this planning task to rewrite siblings. |

Existing root/web AGENTS files contain load-bearing scope rules and some stale stack/persistence descriptions. Do not overwrite them. At implementation preflight, produce a focused proposed instruction patch that records newer owner decisions and current verified behavior, retaining evidence/release boundaries.

Current verification commands observed from manifests: `npm run build` in `web/`; `npx tsc --noEmit` is the proposed TypeScript check; Python project config provides pytest and Ruff through the dev extra. This audit did not execute them. The future frontend `test:acceptance` script in prompts must be established by F0 and is not claimed to exist today.
