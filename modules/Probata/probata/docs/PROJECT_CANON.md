# PROJECT CANON — current architecture and decision routing

> **Reconciled 2026-10-17.** Read the latest scoped owner conversation, current
> Docstore decisions, [the documentation index](INDEX.md), and the selected lane's
> source and acceptance receipts. This file summarizes accepted roles; its title
> and old "locked" wording never override a newer applicable owner decision.
>
> The July/August fleet inventory, obsolete runtime instructions and superseded
> framework/storage assertions have been removed from this active entry point.
> Their dated decisions and execution history remain in [DECISION_LOG.md](DECISION_LOG.md),
> ADRs and Git history. The October 11 source-mirror corrections are retained:
> newer owner decisions control; the legacy Agno adapter's retirement does not
> retire SurrealDB; Context_Files remains stopped.
>
> _Byline: ChatGPT · GPT-6 Astra · 2026-10-17. Documentation reconciliation against
> current repository sources and governed Docstore records; no runtime deployment
> or database-extension installation is certified by this edit._
>
> [Reconciliation sources and validation boundary](../../../../docs/ARCHITECTURE-REFERENCE-RECONCILIATION-2026-10-17.md).

## 0. Document Contract (ONE of each — read this to know what's authoritative)

Use the most recent applicable owner decision for the particular concern. Read
its scope, amendments and validation evidence. An accepted requirement, committed
implementation, indexed document and verified live result are different claims.

| Concern | Entry point |
|---|---|
| Current product purpose and review experience | [PURPOSE.md](PURPOSE.md), with its dated context-processing amendment |
| As-lived, hindsight, foreshadowing and perspective boundaries | [AS-LIVED-HINDSIGHT-USE-CASE.md](AS-LIVED-HINDSIGHT-USE-CASE.md), approved October 17 |
| Accepted architecture and supersession | This file, [DECISION_LOG.md](DECISION_LOG.md), scoped current Docstore records |
| Current case identity and operating flags | [October 5 owner contract](../../../../docs/decisions/2026-10-05-single-case-operating-flags.md) |
| Repository and module ownership | [Root AGENTS.md](../../../../AGENTS.md), [September 26 import](../../../../docs/decisions/2026-09-26-monorepo-import.md), [module AGENTS.md](../AGENTS.md) |
| Code inventory, structure and conventions | [EVIDENCE_MERGE_MAP.md](EVIDENCE_MERGE_MAP.md), [REPO_STRUCTURE.md](REPO_STRUCTURE.md), [CONVENTIONS.md](CONVENTIONS.md), checked against current source |
| Forward work and handoffs | [INDEX.md](INDEX.md), selected lane packet, [BUILD_PLAN.md](BUILD_PLAN.md), [handoffs](handoffs/) |
| Shared indexing boundaries | [SYSTEM-BOUNDARIES.md](../../../../SYSTEM-BOUNDARIES.md) |

ADRs retain their decision history and any unsuperseded scope. A dated build plan
or deployment receipt does not become current merely because this canon links it.
Keep corrections explicit and dated; generated tool catalogs come from the actual
tool docstrings under the October 2 root contract.

Schema source remains the approved final-form snapshot under
`sql/bootstrap/` (D-152/D-153); numbered migration references in historical records
are history. The old pre-launch reset and alternate-case instructions are superseded
by the October 5 contract. A schema-source convention does not authorize wiping
current data, rolling shared Live state backward, or running an old rebuild recipe.

---

## 1. What we are building (the three-part arc)

A personal **evidence-processing + analysis + legal-strategy platform** for the
owner's **pro se** family-law (custody) case. Propria is the umbrella; Probata, Consignatio/Intake,
Indagatio, Advocatio and Vestigia keep their own product/data responsibilities
inside the single repository. Runtime responsibilities are in §4 and §5.

1. **Part 1 — Source processing and eventual evidence.** Open, extract, normalize
   where appropriate, search and review context from messaging, documents, media,
   location records and complete export packages. Current context ingestion has no
   custody, hashing or approval prerequisite. Evidence admission and court-ready
   export are later, separately governed work; AI conversations remain context.
2. **Part 2 — Analysis.** Multi-pass psychological / abuse / toxicity analysis of
   scoped events and conversations (gaslighting, DARVO, coercive control,
   misleading events, corroboration, contradiction, and missed-pattern discovery).
   Runs may be hindsight, as-lived-so-far, or paired to expose the realization delta.
3. **Part 3 — AI Legal Team.** A third agent family that uses the evidence +
   the knowledge base to build legal strategy, documents, and filings. Already
   prototyped by the owner as **Gemini Gems + personas**; to be ported behind
   the platform orchestration contract. A historical framework evaluation is not
   a current framework-selection or deployment decision.
   This is where the imported Michigan legal skills + MCL 722.23 ontology engage.

**The killer mechanism (why the memory is bitemporal):** Part 2 replays how a
person realizes they were abused, in passes with widening knowledge —
**Pass 1 contemporaneous** (agent sees only what was knowable at that moment —
where gaslighting works), **Pass 2/3 assembled hindsight** (all conversations
connected), **Final pass full disclosure** (incl. facts discovered later). A
"pass" is a **knowledge horizon** — a filter over a bitemporal graph. The
**delta between Pass 1 and the final pass for the same event IS the abuse made
legible** ("what you were led to believe vs what was true vs when you found
out"). Every horizon-eligible projection must preserve event time, source
availability, plural realization provenance, and disclosure policy. This is why
the governed SurrealDB analytical role matters (D-073/D-080/D-095).
The owner-approved [as-lived, hindsight and foreshadowing use case](AS-LIVED-HINDSIGHT-USE-CASE.md)
is the detailed experience and acceptance reference. Preserve the eligible original;
retrospective annotations must not alter an as-lived agent's selection, ordering,
context, graph traversal or memory.

**Current clock and projection ruling (ADR-0059, owner 2026-08-18; narrowly
supersedes ADR-0045 §A/A.4):** keep three concepts separate. `occurred_at` is
event time. `source_available_from` is the source-retrieval horizon: occurrence
for a first-party message and the actual receipt/discovery boundary for an
acquired-third-party message. The October 8 context decision removes ingest-time
custody prerequisites; it does not make later import/index time historical knowledge. A message has zero-to-many separate realization links; realization does
not overwrite or backdate source availability. First-party and acquired-third-party
message surfaces are separate **derived projections** from one authored normalized
spine. The acquired thread preserves its real sender, recipients, and participants;
the owner is not one of those participants. Chunks/embeddings are derived and carry
the same source boundary. ADR-0045 §B still sanctions version-pinned derived pass
materializations (single-writer, hash-attested) and still forbids parallel AUTHORED
as-lived/hindsight stores. First-party AI turns preserve native owner-sent/reply times;
unknown native time remains imprecise rather than becoming a later acquisition gate.

**The end-goal frame (the platform itself):** beneath the three-part arc, the
durable product is a **multi-surface tool-platform GATEWAY** — it **serves** its
own tools, **consumes** external tools, and **routes/proxies** between them, exposed
across supported MCP/API clients. The evidence/analysis/legal arc is the first
domain application of that tool platform. ContextForge owns MCP federation;
Portkey owns model routing; Temporal and n8n own their distinct execution roles.
Retired runtimes and donor implementations do not regain authority through this vision.


---

## 2. Agent families (topology)

Agent responsibilities are task contracts within the platform. The old AgentOS
root-router/team roster is no longer the application architecture (D-107).

- Temporal owns durable workflow/run state, retries, timers, signals, waits,
  Activity history and execution receipts.
- n8n supplies the visual business/integration layer.
- The existing Go engine invokes bounded tools as individual Temporal Activities.
  In the current context-processing route it awaits each tool's result before
  selecting the next; Python workers serve individual bounded operations.
- Platform operations, bounded development assistance and legal-analysis tasks
  remain useful roles. A historical agent name is not evidence that it is enabled
  or callable. Read the current registered contract and workflow.

See D-107/D-130 and the October 8 scoped tool-selection decision cited in the
[reconciliation record](../../../../docs/ARCHITECTURE-REFERENCE-RECONCILIATION-2026-10-17.md).

## 3. The knowledge engine — context first, source-preserving

The October 8 owner correction governs current ingestion: every source enters
context without a custody, hashing or approval gate. Evidence promotion is a
separate future phase. AI conversations always remain context.

Preserve complete source exports, packages, originals, attachments, assets, sidecars,
relative paths and native locators. Context storage/search must not wait for
downstream review of an extracted proposal.

| Source or output | Accepted route |
|---|---|
| Human SMS and telephone transcripts | Preserve complete records and their distinct existing normalized shapes; retain source fields and provenance. |
| AI conversations | Keep bodies/messages outside PostgreSQL and outside human-SMS normalization. Extract coherent Chonkie neural topic chunks for Weaviate, with conversation/package identity, native turns, offsets and ordering. |
| Other context | Prefer existing DuckDB extraction/templates where supported; non-messaging content need not be duplicated into a general-purpose PG text table (D-158). |
| Metadata and processing state | PostgreSQL retains source/package identities, provenance, locators, attempt and template references, decisions, projection coordinates and receipts. |
| Complete created works | Retain actual files with their original package and exported data in the existing Case Vault hierarchy; catalog metadata and locators so they can be reopened. |
| Analysis proposals | Preserve source-grounded entities, events, statements, relationships and other candidates. Selected Surreal materialization follows the existing downstream review contract. |

The tool order is Go orchestration, atomic Temporal Activities, DuckDB first where
capable, Chonkie for chunking, and Python for concrete gaps or bounded hybrid
operations. Parsing, chunking and hashing remain separate units. Do not revive
independent processors or let a worker silently sequence the pipeline.

Historical lane designs remain traceable in ADR-0053 and D-158, but their old
"all chat text in PG," capture-time custody and pre-context approval language is
superseded by the October 8 correction. Context publication, analytical inclusion,
and evidence admission are distinct operations. The D-145 lifecycle order is
context, then owner-selected Surreal analysis, then later evidence admission;
the downstream selections do not add a gate before context storage/search.

Case/corpus search stays separate from documentation and development search.
Docstore, CCC and Intake retain independent identities, configuration, state,
locks and write ownership under [SYSTEM-BOUNDARIES.md](../../../../SYSTEM-BOUNDARIES.md).

## 4. Current stack

This is the accepted responsibility map, with source configuration distinguished
from fresh operational verification. Use current Coolify discovery and application
configuration for host placement, names, ports and health; the former July
four-box table is not a deployment inventory.

| Component | Accepted responsibility and source |
|---|---|
| PostgreSQL 18 | Canonical relational/JSON state, normalized messaging, metadata, lineage, governance and receipts; D-080, D-158 and the October 8 context scope. |
| pg_duckdb, PostGIS, pgvector | The PostgreSQL deployment image configures these extensions. pg_duckdb runs DuckDB inside PG; it is not a second canonical database. See ADR-0013, D-080, and [the PostgreSQL Dockerfile](../deploy/docker/postgres/Dockerfile). Source configuration alone does not prove an extension is installed in a particular live database. |
| Weaviate | Governed semantic/hybrid context search and retrieval projections; ADR-0040, D-080/D-158 and current context-processing decisions. |
| Neo4j/DozerDB | Independent Semantica and SAT-RAG graph projections, with `evidence` and `sat-temporal` scope; D-093/D-106. Graphiti retirement does not retire Neo4j. |
| SurrealDB | Governed temporal aggregation, walks and analysis; D-073/D-080/D-107/D-151. Separate Docstore, Intake and legal-library uses retain their own ownership. |
| Temporal + n8n + Go | Durable execution, visual integration and bounded tool invocation as described in §2; D-107/D-130 and October 8. |
| LlamaIndex + LangGraph | Accepted retrieval/indexing, stitching, SAT extraction and bounded assembly/state-machine roles; D-143/D-144. They consume canonical truth and produce derived results. |
| ContextForge + Portkey | MCP federation and model/embedding/reranking routing respectively; D-107 and ADR-0042. |
| Coolify | Existing VPS application build/deployment plane, from the monorepo/module configuration; root deployment contract. |
| Object storage / Case Vault | Complete retained originals and created works at their current governed locations; use current source locators and configuration. |
| Docstore | CocoIndex/NIM/SurrealDB documentation retrieval; its DuckDB presentation role is separate from PostgreSQL's pg_duckdb role. |

Milvus has a separate current memsearch use. Retiring its old platform-wide vector
role is not authority to remove the memsearch store. Likewise, local/provisional
SQLite use in a particular app is not a platform-wide storage decision.

## 5. Locked decisions

"Locked" does not freeze a superseded decision. The current scoped rulings below
replace the old contradictory assertions in this section.

| Concern | Current ruling | Authority |
|---|---|---|
| Graphiti | Retired; no live lane, required baseline, bake-off gate or replacement-store prerequisite. | D-070/D-095 |
| AgentOS / Agno | AgentOS retired. Agno may remain only as the disabled bounded atomic-agent library adapter until a task contract enables it. | D-107 |
| SurrealDB | Accepted analytical/temporal engine. Only the legacy Agno operational adapter was retired. | D-073/D-080/D-107/D-151 |
| Memgraph | Discussed graph candidate; not selected, scheduled or a blocking dependency. | D-095 |
| LangGraph / LlamaIndex | Included with their defined bounded roles; the old benched/Agno-native-only assertions are superseded. | D-143/D-144 |
| Context ingestion | No hashing, custody or pre-storage/search approval gate; AI bodies stay outside PG and AI conversations remain context. | October 8 Docstore owner correction, revision 9 |
| Evidence and analytical inclusion | Context ingestion, selected Surreal materialization and eventual evidence admission keep their distinct existing contracts. | D-107/D-145 and October 8 amendment |
| Model routing | Portkey is the model gateway; old LiteLLM routes are retired. Read current provider/model configuration rather than a copied historical priority list. | ADR-0042/D-107; current settings |
| Case identity | One approved matter/court case in Dev and Live; independent operating, auth, evidence, feature and lifetime controls. | October 5 root decision |
| Source ownership | One Propria Git root with imported modules; per-product remotes are archives. | September 26 root decision |

The source-grounded analytical requirements survive this infrastructure cleanup:
claim-centered fact assembly, source-family corroboration, bounded investigations,
immutable scope/receipts, source availability versus event time versus realization,
and perspective-correct as-lived/hindsight analysis. See ADR-0057–0059 and the
[October 17 use case](AS-LIVED-HINDSIGHT-USE-CASE.md).

The following standing contracts also remain:

- Prefer proven open-source components and minimize custom code; custom work is
  for the owner's domain-specific requirements (June 13 ruling).
- Every tool, agent and workflow remains atomically addressable through an
  internal API and an external MCP wrapper. Preserve progressive disclosure and
  bounded result retrieval rather than sending whole catalogs in context; see
  [ADR-0023](adr/0023-universal-api-mcp-exposure.md).
- ADR-0054 requires durable itemized run reports: what ran or skipped, why,
  outputs, remediation and attributable decisions, with correlated diagnostics.
- Under ADR-0059, a healthy reconciled checkpoint resumes the same walk identity.
  A terminal integrity failure seals a non-resumable snapshot and starts an
  explicitly linked rewalk; healthy continuity must not become a needless reset.
- Source traceability and the applicable approval/receipt contracts remain.
  Historical blanket approval text does not add a new prompt before routine
  context storage or search.

## 6. Roadmap

The current lane packet, current source and dated live receipts determine the next
work. [BUILD_PLAN.md](BUILD_PLAN.md), [handoffs](handoffs/) and historical TODO
ledgers provide provenance; June–August P0/P1/P2 completion labels are not today's
implementation or deployment status.

The product direction remains context ingestion and readable review, downstream
perspective-correct analysis, and legal work products. The retired Graphiti image,
AgentOS-native UI, old migration cutovers and independent-repository publishing
steps are no longer roadmap items.

## 7. Access & credentials (POINTERS — no secrets in this file)

Use the current [root access contract](../../../../AGENTS.md), selected module
configuration and live service discovery. The canonical application addresses use
the current Tailscale service DNS contract. Historical public IPs, host/port lists,
tunnel recipes, bootstrap-superuser login guidance and retired MCP/model doors
have been removed from this entry point.

The root database-access note records per-client logins for `probata-db` and the
October 2 disabling of password login for the bootstrap `ai` superuser. Read that
current contract and authorized secret references; do not copy credentials into
documentation or resurrect the old pending-app-role-cutover instructions.

## 8. Gotchas (hard-won, do not relearn)

- Accepted design, source configuration and live behavior require separate
  evidence. Do not convert an old successful probe into a current health claim.
- Retired adapter code and generic vendor support do not describe the deployed
  product stack.
- Read current dependency locks, build files and tool docstrings for versions,
  provider input contracts and available operations.
- Keep source objects distinct from derived projections, annotations and
  memories; preserve locators and the applicable source/version.
- An as-lived boundary applies before retrieval, ranking and agent context/memory
  assembly. Hiding a retrospective annotation only in the UI does not enforce it.
- A final-form schema snapshot is not blanket reset permission.

Retired Graphiti/AgentOS image workarounds remain in dated history, not current
setup instructions.

## 9. Where things live

The single Git root is `E:/AI_Workspace/Projects/Propria`; this module is
`modules/Probata/probata/`. All application edits commit through that root.
[Root AGENTS.md](../../../../AGENTS.md) lists the current product and fork paths.

Documentation under this module routes through [INDEX.md](INDEX.md).
[DECISION_LOG.md](DECISION_LOG.md) and ADRs retain their dated decisions; plan and
handoff status must be read with later amendments. Donor archives, preserved
corpora, credentials and generated indexes retain their existing exclusions.
Do not reconstruct old workspace paths or create new checkouts from a historical
memory filename.

[Docstore source boundaries](../../../../SYSTEM-BOUNDARIES.md) determine indexing
eligibility. Documentation search, memsearch/CNF agent memory and case/corpus
search are separate surfaces; correcting one does not refresh the others.

## 10. Open threads / parking lot

Current open work must have a current lane owner and verification boundary. The
former August activation holds, Graphiti replacement proposal and framework
bake-offs have been removed from this active list because later rulings supersede
their premises. Their historical records remain available for provenance.

The full as-lived/hindsight/foreshadowing behavior is an accepted use case, not a
completion claim. Use its [six acceptance checks](AS-LIVED-HINDSIGHT-USE-CASE.md#6-what-successful-behavior-must-demonstrate)
against the actual data paths when that work is implemented.
