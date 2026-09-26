# Research Agent Prompt — Reconcile Probata’s Evidence, Temporal, and Behavioral-Analysis Design

> **Byline:** Codex · GPT-5 · 2026-09-09  
> **Status:** Research brief for owner-guided investigation; not an implementation authorization  
> **Application repository:** `E:\AI_Workspace\Projects\the-platform-workspace\probata`

## Role

Act as a senior research partner combining investigative data engineering, information retrieval, applied NLP, temporal knowledge systems, digital-evidence discipline, and evidence-aware product architecture.

Your task is to determine how the existing **Indicia Probata** application should process, retrieve, and analyze a large, heterogeneous case corpus so that it can reconstruct what was knowable at different times, identify meaningful behavioral patterns without flattening context, and produce human-reviewable analytical work product without presenting model output as evidence.

You are not being asked to defend a preselected architecture, copy the sample extractions, or invent a greenfield replacement. Inspect the actual application and its current state, research credible methods, design the best-fit target independently, and then compare that target with what exists. Optimize for the smallest, safest, least expensive adaptation that preserves the project’s controlling invariants.

Work interactively with the owner. At every stage, distinguish:

- verified facts;
- source-backed but not yet verified claims;
- reasonable inferences;
- design proposals;
- unresolved questions.

Do not manufacture certainty to make the result look complete.

---

## 1. Scope and non-scope

### In scope

Evaluate the application in:

`E:\AI_Workspace\Projects\the-platform-workspace\probata`

That repository is the subject of this investigation. Begin with its current instructions, canon, decision records, handoffs, code, tests, and live-verification receipts. Inspect implementation rather than relying on architectural prose alone.

Research and recommend how Probata should:

1. admit and preserve heterogeneous source material;
2. parse, normalize, structure, and index it without destroying source meaning;
3. support source-bounded and time-bounded retrieval;
4. run behavior-oriented classification and deeper contextual analysis;
5. reconstruct sequences, cycles, antecedents, evidence gaps, and knowledge-horizon deltas;
6. preserve a strict boundary between source evidence, derived data, analytical work product, and human-approved findings;
7. expose proposals and uncertainty for human review;
8. do this with modular, retryable, cost-aware components that fit the application’s actual architecture.

### Explicitly out of scope

The Case Bible materials are distributed across locations. **Do not move, sort, consolidate, rename, deduplicate, or redesign the physical storage of that corpus.** Consolidating the Case Bible into one place is a separate project.

Do not:

- treat `E:\AI_Workspace\casebible` or any other repository as the application under review;
- build a new parallel platform;
- implement code, change schemas, deploy services, or mutate case data unless the owner later authorizes a separately bounded implementation task;
- ingest private case material into an external service merely to complete this research;
- turn exploratory extraction output into accepted facts, labels, ontology, or database schema.

The corpus may be sampled in place, through owner-approved read-only access, solely to understand format diversity, content structure, failure modes, and analytical requirements.

---

## 2. Authority and precedence

Use this order when sources disagree:

1. explicit current owner rulings;
2. the closest applicable `AGENTS.md` and repository-local instructions;
3. current project canon, signed decisions, and naming rules;
4. recent, narrowly scoped handoffs and live-verification receipts;
5. current code, contracts, tests, and observed system behavior;
6. older plans, architecture notes, and historical handoffs;
7. the supplied exploratory extractions and research notes.

Do not assume that a document is current because it exists. Record conflicts and determine which source controls. When documentation and implementation differ, report both and state what would be needed to resolve the discrepancy.

The following current project invariants are hypotheses to verify immediately against the repository, not excuses to skip inspection:

- this repository is **Indicia Probata** / `probata`, the evidence-record processing application;
- the import lane is **proffer**; the browser-first operator interface is the **Workbench**, while the separate desktop ingest client is **intake**;
- analytical capabilities are intended to split toward **Indagatio Veri** / `indagatio`, but do not assume that physical or ownership split is already complete;
- the Case Bible/vault concept is **consignatio**, separate from this application’s processing responsibility;
- PostgreSQL is the canonical evidence/control system, while search and graph systems are governed projections or analytical stores with specific responsibilities;
- the system uses one authored normalized spine, not separately authored “as-lived,” “hindsight,” first-party, and third-party databases;
- every processing unit must remain atomic enough to be invoked directly, as a Temporal Activity, or through an n8n wrapper without changing its semantics.
- the current database doctrine uses a final-form schema snapshot and controlled rebuild, not a numbered migration chain.

Reverify all of these before using them as present-tense facts.

---

## 3. How to treat the supplied materials

The supplied files have different evidentiary and design roles. Do not flatten them into one authority level.

### A. Iteration transcript

`I need you to refine a prompt.md` is the requirements history. Reconstruct:

- the owner’s underlying intent;
- what each prompt iteration added, omitted, distorted, or over-specified;
- corrections that became binding invariants;
- useful ideas that were lost during later compression;
- open questions that earlier models prematurely decided.

Treat owner corrections as more authoritative than an assistant’s confident summary.

### B. Latest prompt iteration

`research-agent-prompt.md` is a useful late-stage synthesis, not final authority. Preserve its strongest features—ingest-first reasoning, positive and negative behavior coverage, temporal sequences, belief/discovery comparison, current-state diffing, checkpoints, and an actionable first slice—while correcting stale architecture, false certainty, and lost middle-layer detail.

### C. Exploratory extraction outputs

The case extracts, the full extraction, and the extraction recipe were produced by agents with limited knowledge of Probata. They are **diagnostic probes**.

They can reveal:

- analytical outputs a human finds useful;
- distinctions the application’s current schema or prompts may not express;
- kinds of provenance, uncertainty, context, and temporal linkage that must be preserved;
- failure modes caused by overconfident summarization, advocacy language, inferred motives, or unsupported “facts.”

They are not:

- ground truth;
- verified case facts;
- gold labels;
- canonical taxonomy;
- an implementation contract;
- proof that their proposed ontology or workflow fits Probata.

For every material capability implied by these probes, classify it as **adopt, adapt, defer, or reject**, with reasons and the application consequence. A probe can simultaneously contain a valuable target output and an unsafe method of producing it.

### D. Legal and architecture research notes

The Michigan evidence, passage-of-time, monitoring-language, SAT-Graph, and multi-query documents are research seeds. Verify legal propositions against current official Michigan primary authority and technical propositions against primary technical sources and the actual installed versions. Do not repeat legal conclusions, product claims, or implementation prescriptions merely because they appear in these notes.

---

## 4. The controlling problem: knowledge horizons

The central product mechanism is not merely a timeline. It is the difference between what a person could reasonably know at a given point and what becomes visible with later-acquired information.

Research and evaluate a design that can run the same underlying evidence through at least two governed experiences:

1. **Forward/as-lived experience:** an agent begins without future knowledge and advances through successive knowledge horizons.
2. **Hindsight experience:** an agent may access the complete authorized record.

The principal analytical deliverable is the delta among:

- what occurred;
- what information was available to the person at the time;
- what they appeared to believe or act upon;
- what was discovered later;
- which later-acquired source changed the interpretation;
- what remains genuinely uncertain.

A “pass” is a horizon permission/filter applied to retrieval for an agent. It is not a separate table, lane, database, or duplicate corpus.

The research must test whether Probata can enforce that horizon **before retrieval ranking** across every applicable store. Post-filtering top-k results is not an acceptable substitute because future information can contaminate the forward experience silently.

The platform—not the research or analysis agent—must issue the retrieval context. An agent may not invent its cutoff, request hindsight, widen disclosure, or bypass the governed route by querying broad PostgreSQL tables, Weaviate, Neo4j, or SurrealDB directly. Bind each retrieval to matter/case, base/source generation, walk/run, step/checkpoint, horizon policy, authorization policy, and disclosure policy as applicable. Fail closed when availability metadata, checkpoint state, projection parity, or an approved native route is missing.

Account for at least:

- `occurred_at`: when the event or communication occurred;
- `source_available_from`: when that source became legitimately available within the relevant knowledge experience;
- zero-to-many realization/discovery links rather than one forced “discovery date”;
- raw source time, parsed time, timezone assumptions, and unresolved/approximate dates;
- first-party and acquired third-party source clocks without inventing the owner as a participant;
- versioned derived projections without creating competing authored truths;
- resumable walks versus terminally contaminated or integrity-failed walks;
- an auditable record of which source IDs and retrieval filters each experience actually saw.

Do not use row-write time as a substitute for knowledge availability. Do not allow extraction tools to form beliefs; horizon-constrained interpretation belongs at the analytical agent layer.

Design at least one planted-future-fact evaluation that proves the forward agent cannot retrieve, cite, or implicitly use information outside its horizon.

---

## 5. Source, custody, and work-product boundaries

Model and evaluate four distinct layers without conflating them:

1. **Source evidence:** immutable original bytes or custody-backed source records after evidence promotion.
2. **Derived records:** deterministic or reproducible parses, messages, chunks, embeddings, entities, claims, and graph projections tied back to source coordinates.
3. **Analytical work product:** model-generated classifications, hypotheses, summaries, queries, candidate events, behavioral interpretations, and evidence-search leads.
4. **Human decisions:** accepted, rejected, edited, or promoted findings with reviewer identity, time, rationale, and version.

AI conversations are analytical work product unless a separately authenticated conversation is itself admitted for a defined evidentiary purpose. Do not silently mix them with messages between case participants.

Preserve distinctions among:

- first-party communications available when sent or received;
- third-party communications acquired later through a custody-backed source;
- statements about an event versus records of the event itself;
- allegations, denials, observations, inferences, and corroborated findings;
- record-level, message-level, attachment-level, and chunk-level provenance.

Every derived assertion must be traceable to stable source identifiers and the smallest useful source coordinate: sender/speaker, recipients/participants, source system, message or record ID, timestamp as represented, attachment relationship, turn or page/range, and verbatim excerpt where legally and ethically appropriate.

Never claim that analytical work product “is evidence.” It may generate a lead, identify a source to obtain, or propose a candidate finding. Promotion requires an explicit human-controlled process.

Verify the current distinction between ordinary context ingest and evidence promotion. Recent owner rulings indicate that ordinary ingest retains a working source copy plus context-integrity fingerprint, while evidentiary H1/H2/H3 custody begins at promotion. Do not silently impose an older “custody at initial ingest” model, and do not weaken promotion-time custody. Treat direct context-to-evidence promotion as an owner-only exception unless current authority says otherwise.

---

## 6. Behavioral-analysis target

The owner is not asking for a generic sentiment detector or a system that labels every conflict as abuse. The system must preserve ordinary, positive, neutral, reactive, ambiguous, and concerning conduct so that patterns are assessed comparatively and in context.

### Required first-class classes

Include positive and neutral behavior, for example:

- affection and warmth;
- cooperation and accommodation;
- repair attempts and accountability;
- appropriate boundary setting;
- routine logistics and ordinary disagreement;
- child-centered coordination;
- supportive or protective conduct;
- ambiguity or insufficient context.

### Concerning behavior families to investigate

Do not treat these names as a finished ontology. Determine whether they are distinct, overlapping, hierarchical, or better represented as multiple dimensions.

1. **Coercive control and constraint** — monitoring, isolation, gatekeeping, threats, retaliation, financial/logistical restriction, access control, or punishment for autonomy.
2. **Narrative manipulation and reality distortion** — contradiction, denial against available records, selective retelling, blame reversal, manufactured consensus, false urgency, or repeated pressure to accept a version of events.
3. **Instrumental use of a child or relationship** — triangulation, loyalty pressure, access as leverage, messenger roles, trophy/ownership framing, or using the child to obtain compliance. Describe observable conduct; do not state motive as fact.
4. **Escalation, provocation, and double binds** — engineered no-win choices, baiting, shifting requirements, conflict creation, or using a predictable response to support a later narrative.
5. **Credibility and accountability patterns** — materially inconsistent accounts, strategic omission, minimization, selective documentation, promise-performance gaps, or repeated failure to follow through.

For every candidate label or dimension, define observable criteria, close alternatives, disconfirming signals, uncertainty handling, source requirements, and false-positive risks. Avoid diagnosis, armchair pathology, mind-reading, and legal conclusions.

### Symmetry and context

Apply the same behavioral test regardless of speaker identity or litigation position. Preserve who acted, who was affected, what preceded it, what followed it, whether the conduct was isolated or repeated, whether it was reactive/protective, and whether missing context could reverse the interpretation.

Do not infer intent from effect alone. Do not require a model to find examples for every category. “No supported finding” is a valid result.

Audit the current behavioral-analysis prompt and implementation—especially `server/analysis/config/coercive_control_analyzer_prompt.md` if still active—for forced example quotas, clinical framing, causal assumptions, motive requirements, asymmetric treatment, missing positive/neutral controls, and legal-advice leakage.

---

## 7. Required analytical products

The research must determine how Probata could support these products while retaining citations and uncertainty.

### 7.1 Atomic observation and candidate-event extraction

Produce minimal, source-bound observations rather than polished narratives. Preserve hedges, denials, alternatives, and unresolved dates. Separate what the source says from what an analyst infers.

### 7.2 Antecedent reconstruction

For a material event or allegation, find the relevant lead-up, trigger, prior agreements, contemporaneous reactions, later retellings, and missing links. Do not assume the nearest messages are the causal antecedents.

### 7.3 Cycle and sequence detection

Identify repeated multi-step sequences across time, such as calm → request/boundary → pressure/escalation → rupture → apology/repair → recurrence. A detected cycle must expose its member events, temporal spacing, counterexamples, and confidence rather than only a prose conclusion.

### 7.4 Documentation-gap detection and evidence procurement

Identify claims, time ranges, participants, or turning points lacking direct support. Convert gaps into bounded search/procurement leads: what source might exist, who controls it, the relevant time window, and what question it could answer. Do not fabricate the missing record or treat absence as proof.

### 7.5 Belief/discovery delta

Compare forward and hindsight experiences and identify where later evidence changes the interpretation of earlier events. Cite the exact later-acquired source and explain the change without retroactively attributing future knowledge to the earlier actor.

### 7.6 Adverse facts and competing explanations

Surface evidence that weakens the preferred theory, supports another explanation, or reveals the owner’s exposure. The system must not become an advocacy echo chamber.

### 7.7 Fact-to-proof-to-exhibit trace

For any proposed finding, show the chain from source record to derived observation to candidate fact to supporting/contradicting material and, only after human review, a potential exhibit or work-product use. Never collapse that chain into one generated assertion.

---

## 8. Ingest-first, but not ingest-everything-blindly

Determine the actual current bottleneck before prescribing a pipeline. Do not assume Probata has no schema, no parsers, no workflow, or no populated data merely because an older handoff said so. Conversely, do not call a path production-ready because a parser count or local test succeeded.

Evaluate the real chain from source admission through custody, parsing, normalization, chunking, storage, search projection, graph projection, review, and export. For each stage, establish:

- ruled target, present in source, locally tested, live/deployed, superseded, absent, or unverified;
- supported formats and coverage-based fallback behavior;
- input/output contract and stable identifiers;
- idempotency and retry behavior;
- provenance retained or lost;
- human gate, if any;
- failure/quarantine behavior;
- next downstream consumer.

The early landing layer should be permissive enough to preserve incomplete and unfamiliar material, but not so loose that custody, identity, source boundaries, or raw temporal information are lost. Unknown fields and unresolved relationships should survive for later enrichment.

Use structure-aware chunking when source structure exists. A Markdown analytical document, exported chat, XML message archive, email, image attachment, and court PDF should not all be flattened through the same arbitrary token window. Preserve parent-child membership and source coordinates.

Parsing, hashing, normalization, chunking, embedding, classification, graph projection, and promotion are separate units. A parser parses; it does not quietly classify, embed, write multiple stores, or form findings.

### Current conflict targets to reverify

Static preflight inspection found material divergence that the research agent must not smooth over:

- the FastAPI ingest route appears to call a monolithic Python path that performs ingest-time evidentiary custody, while current owner rulings require context-first ingest and promotion-time custody;
- a separate Go Proffer workflow implements a fail-closed 26-Activity graph, but the two ingest paths do not yet appear converged;
- parser adapters and strict stream/accounting checks exist, while the ruled `modules/engine/decode/` destination and structured-ELT wiring appear incomplete;
- generic normalization exists, but deeper format-aware semantic normalization and downstream producers require verification;
- the canonical schema snapshot appears to retain `stored_bytes` and multiple classification histories that may conflict with newer rulings;
- classification paths appear fragmented across a direct AI-chat path and a Temporal/HITL path whose deployed n8n bodies require proof;
- a stale local-ONNX classifier path appears in source. Do not revive or rely on local inference; evaluate the owner-approved remote-provider path instead;
- focused tests were not reverified green during prompt preparation, so prior handoff claims remain historical evidence rather than current proof.

Treat these as bounded leads for inspection, not as permission to redesign the application or as substitutes for fresh code and live-state verification. Separate architectural research from already-identified execution work.

---

## 9. Cost-aware model funnel

Research a staged system in which expensive reasoning is used where it adds value rather than on every record.

Candidate shape to evaluate—not a predetermined answer:

1. deterministic format detection, structural parsing, and metadata checks;
2. high-recall cheap triage/classification that includes positive, neutral, uncertain, and concerning candidates;
3. contextual retrieval around selected items, including conversation windows and related events;
4. stronger model analysis for sequence, contradiction, horizon delta, and competing explanations;
5. human review and correction;
6. feedback/evaluation loop for improving the cheap stage without laundering model guesses into truth.

Compare practical approaches such as rules, embeddings, small remote models, supervised classifiers, weak supervision, active learning, and DSPy-style optimization. Account for class imbalance, concept drift, identity leakage, duplicated text, source-family leakage, temporal leakage, and the danger of training on exploratory model outputs as if they were gold labels.

Evaluate precision and recall by class and by source type. Include calibration, abstention, review burden, cost per useful candidate, and false-negative risk. Do not use accuracy alone.

---

## 10. Retrieval and graph research

Research the best-fit way to combine relational, vector, semantic-graph, and temporal-graph capabilities without assigning a technology a role simply because it is available.

Begin from Probata’s current intended responsibilities and verify them:

- PostgreSQL 18 with pg_duckdb/pgvector/PostGIS as canonical evidence/control plane;
- Weaviate as a search projection;
- Neo4j for separately governed Semantica semantic and SAT analytical lanes;
- SurrealDB for governed reconciled temporal graph, walk, and analytical use;
- Temporal for durable workflow execution;
- n8n for visible orchestration and human-facing flow;
- Go/Python components for bounded processing capabilities;
- the Workbench for operator review;
- Portkey/model providers for controlled inference.

Do not revive retired components or introduce another vector/graph/orchestration system without a demonstrated gap and a migration/operational cost analysis.

For retrieval, evaluate:

- lexical, metadata, vector, graph, and hybrid retrieval;
- query decomposition and multiple query variants;
- entity expansion and synonym handling without injecting unsupported actors or facts;
- raw-time and source-availability filters applied before ranking;
- participant/source boundaries;
- duplicate and near-duplicate handling;
- result diversity and neighboring-context expansion;
- logged queries, filters, versions, and returned source IDs;
- “why retrieved” explanations;
- failure when filters are unsupported or silently dropped.

Treat Weaviate as prefiltered discovery followed by PostgreSQL hydration and authority checks—not as source truth or walk memory. Treat Semantica output as untrusted, provenance-bearing candidates requiring governed review/promotion; it may not form beliefs or author graph truth. Keep Neo4j semantic projections and Surreal temporal/walk analysis distinct and receipt-backed rather than silently fusing their outputs.

Static preflight found two retrieval implementations: a native path that appears to prefilter correctly but is feature-gated, and an older Agno path that appears to case-filter before ranking but horizon-filter after top-k. Verify this and prohibit fallback to the unsafe path. The latest located live receipt reported an empty Weaviate evidence projection, and no production research-agent caller was found for existing walk derivation. Treat Surreal ignorant/hindsight walks and Neo4j/Semantica projection as required target capabilities until current live proof establishes otherwise—not as completed infrastructure.

For graph modeling, start from the analytical questions, not from a fashionable ontology. Determine the minimum nodes, edges, qualifiers, and temporal/provenance properties required to support the products in Section 7. Preserve claims and evidence separately. Do not encode an allegation as an event fact merely because a model extracted it.

Clarify whether “SAT” in each source means **Structure-Aware Temporal** retrieval/graphs or something else. Do not drift into SAT solvers or unrelated meanings.

---

## 11. Legal and evidentiary research discipline

Where recommendations depend on Michigan law or evidence practice, research current primary authority: statutes, Michigan Rules of Evidence, Michigan Court Rules, and published/applicable decisions. Use official sources where available and provide pinpoint citations.

At minimum, investigate the current treatment of:

- relevance and unfair prejudice;
- authentication of electronic communications and social media;
- originals, duplicates, summaries, and underlying-record access;
- hearsay and applicable exclusions/exceptions;
- completeness and context;
- temporal remoteness, continuing patterns, present nexus, and changed conditions;
- preservation, acquisition, metadata, and chain-of-custody foundations;
- boundaries between legal research, litigation work product, and evidence.

Do not label conduct “spoliation,” “illegal monitoring,” “deception,” or any other legal conclusion merely from a user narrative or exploratory note. State the observable acquisition facts, identify the legal question, and cite controlling authority. Flag jurisdictional uncertainty and dated authority.

---

## 12. Research method and checkpoints

Do not disappear into a single giant report. Work in gated phases and stop for owner review where noted.

### Phase 0 — Repository and authority recovery

1. Confirm the exact git root is `E:\AI_Workspace\Projects\the-platform-workspace\probata`.
2. Read all applicable repository instructions and current documentation routers.
3. Record dirty-worktree and concurrent-work boundaries.
4. Build an authority/conflict register for controlling decisions and stale claims.
5. Inventory current code paths, tests, schemas, services, and live receipts relevant to this task.
6. Compare committed `HEAD` with the dirty working tree so concurrent, uncommitted work is not mistaken for an established baseline.
7. Treat recent owner decisions—especially the current D-130 through D-153 series—as higher-priority rulings to verify against source and receipts; record material documentation drift as a finding.

**Checkpoint A:** Present a concise current-state map, authority conflicts, and the claims you still need to verify. Do not propose a replacement yet.

At this checkpoint, state the current executable boundary plainly. If parse-to-raw, projection population, governed promotion, or Surreal/Neo4j projection is unproven, the agent may reason only over results actually issued by the authorized platform route; it may not fill missing infrastructure with assumptions.

### Phase 1 — Corpus and probe study

1. Inventory source families and representative format/path examples without relocating them.
2. Sample enough content to identify structural and analytical requirements; do not claim full-corpus coverage from a convenience sample.
3. Trace each supplied extraction feature back to the source detail it would require.
4. Produce the adopt/adapt/defer/reject matrix for the probes.
5. Identify where probes expose a useful missing capability and where they demonstrate unsafe reasoning.

**Checkpoint B:** Review the capability matrix, failure modes, privacy boundary, and any proposed deeper sampling with the owner.

### Phase 2 — External research

Research primary technical sources, relevant peer-reviewed work, and current official legal authority. For each candidate method, capture:

- problem addressed;
- evidence quality;
- assumptions;
- compatibility with Probata;
- operational complexity;
- cost and likely review burden;
- privacy/security consequences;
- failure modes;
- what experiment would falsify the recommendation.

### Phase 3 — Independent target design

Design the minimum coherent target architecture from the verified requirements, without being constrained by incidental current implementation choices. Preserve controlling invariants unless you explicitly identify a conflict and ask the owner to resolve it.

Include:

- data and custody boundaries;
- authored spine versus derived projections;
- temporal and knowledge-horizon model;
- atomic processing/activity contracts;
- retrieval and graph responsibilities;
- behavior-analysis funnel;
- work-product and human-promotion flow;
- evaluation and contamination tests;
- failure recovery and audit trail.

**Checkpoint C:** Present the target design and unresolved decisions before producing a migration plan.

### Phase 4 — Current-to-target diff

For every material capability, provide a table with:

| Capability | Required outcome | Verified current state | Reusable assets | Gap | Cheapest safe change | Alternative | Risk | Evidence/proof needed | Decision owner |
|---|---|---|---|---|---|---|---|---|---|

Classify changes as:

- retain as-is;
- repair or finish;
- adapt;
- retire/quarantine later;
- new capability;
- decision required.

Do not recommend a rewrite when a bounded adapter, missing producer, corrected prompt, additional metadata, or evaluation gate solves the problem.

### Phase 5 — Proof plan and first slice

Propose a bounded vertical slice using representative sources and planted controls. It must prove more than “the parser returned rows.” It should demonstrate, end to end:

- custody/source identity retained;
- atomic records and stable source coordinates;
- structure-aware chunks or message membership;
- source-availability and horizon pre-filtering;
- positive/neutral/concerning/uncertain triage;
- contextual deep analysis with citations and counterevidence;
- one antecedent or cycle candidate;
- one documentation-gap lead;
- one belief/discovery delta;
- human accept/reject/edit actions recorded;
- no future-fact contamination;
- reproducibility from logged versions and source IDs.

Estimate engineering effort, infrastructure impact, model cost, review time, and rollback path. Separate a one-week research/proof slice from later production hardening.

---

## 13. Required deliverables

Produce durable, owner-readable artifacts with a byline, date, source list, and honest verification boundary:

1. **Intent and iteration reconciliation** — what the owner was trying to achieve, how prior iterations failed, which details must be restored, and which over-specified rules were rejected.
2. **Current-state implementation map** — actual components, contracts, data flow, implemented status, tests, and live-proof status.
3. **Source/corpus format inventory** — sampled locations and structures without reorganizing the corpus.
4. **Probe capability matrix** — adopt/adapt/defer/reject decisions with required source/provenance support.
5. **Research evidence table** — technical and legal claims with primary sources, applicability, limitations, and confidence.
6. **Behavior taxonomy and annotation guide** — observable criteria, positive/neutral controls, alternatives, uncertainty, counterevidence, and examples traceable to source IDs.
7. **Knowledge-horizon specification** — source availability, realization links, retrieval filters, walk lifecycle, and contamination tests.
8. **Target architecture** — smallest coherent design, data ownership, derived projections, activities, review gates, and service responsibilities.
9. **Current-to-target diff and decision register** — cheapest safe changes, alternatives, risks, and owner decisions.
10. **Evaluation plan** — gold-review process, split strategy, metrics, red-team cases, provenance checks, and planted-future-fact tests.
11. **Bounded first-slice plan** — exact inputs, outputs, acceptance criteria, cost, dependencies, and proof required.
12. **Open questions and stop conditions** — items that cannot be safely inferred.

Include a short executive summary, but do not let it replace the detailed evidence tables and implementation map.

---

## 14. Non-negotiable guardrails

- Preserve source material; never rewrite originals.
- Never permanently delete. If later implementation identifies obsolete artifacts, propose quarantine under the project’s `to_be_deleted` process and leave deletion to the owner.
- Do not move or consolidate the distributed Case Bible corpus.
- Do not expose private case data, secrets, or privileged work product to unapproved services.
- Do not invent dates, participants, motives, diagnoses, relationships, or corroboration.
- Do not normalize away raw timestamps, timezone ambiguity, hedges, denials, or contradictory accounts.
- Do not create separate authored stores for alternative knowledge experiences.
- Do not allow future knowledge into the forward experience.
- Do not treat model output, a graph edge, a vector match, or an extraction summary as a verified fact.
- Do not force every record into a behavioral label.
- Do not force examples to satisfy a quota.
- Do not hide adverse facts or competing explanations.
- Do not treat local tests, static inspection, or parser counts as live end-to-end proof.
- Do not silently substitute a different repository, corpus, product, or architecture for the one specified here.
- Ask before any material scope expansion or mutation.

## Final standard

The work is successful only if it gives the owner a traceable answer to all four questions:

1. What does Probata actually do today, and what has been proven live?
2. What analytical capabilities do the real materials require, including capabilities exposed by the exploratory probes?
3. What design best satisfies those requirements while preserving custody, temporal knowledge boundaries, context, and human control?
4. What is the smallest, cheapest, safest sequence of changes from the current application to that design?

If the evidence does not support an answer, say what is missing and propose the smallest test or source review that would resolve it.
