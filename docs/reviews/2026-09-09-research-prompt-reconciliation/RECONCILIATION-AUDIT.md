# Probata Research-Prompt Reconciliation Audit

> **Byline:** Codex · GPT-5 · 2026-09-09  
> **Scope:** Reconciliation of the supplied prompt history and supporting files against the Indicia Probata application at `E:\AI_Workspace\Projects\the-platform-workspace\probata`  
> **Verification boundary:** Static inspection of supplied files and repository materials; this audit does not claim deployment or live end-to-end verification.

## Outcome

The reconciled prompt restores the user’s full intent without treating exploratory outputs as canonical design. It directs a research agent to inspect Probata first, study the distributed corpus without relocating it, research current methods and authority, design the best-fit target independently, and then produce a current-to-target diff focused on the cheapest safe adaptation.

The earlier “final” prompt was directionally strong but too compressed and partly stale. Its most serious defect was calling the sample extractions “ground truth.” Those files are better understood as probes: they expose desirable analytical products and missing representational capacity, while also demonstrating unsafe inference and provenance failures.

## Intent recovered from the iteration history

The stable objective is a system that can process heterogeneous case material and support behavior-oriented analysis across messages, windows, events, cycles, and eras. It must reconstruct what was knowable at each point in time, compare that experience with hindsight, and expose the delta created by later-acquired information.

The owner repeatedly required:

- a low-cost triage stage before expensive contextual reasoning;
- positive, neutral, cooperative, affectionate, reactive, ambiguous, and concerning conduct as first-class possibilities;
- analysis of deception/narrative manipulation, coercive control, child-related leverage, escalation, and credibility without diagnosis or mind-reading;
- antecedent reconstruction, cycle detection, documentation-gap discovery, and evidence-procurement leads;
- relational, vector, semantic, and temporal capabilities with disciplined responsibilities;
- strict separation of source evidence from AI work product;
- source provenance, temporal availability, human review, and reversible decisions;
- research into the actual corpus and system before locking an implementation design;
- an interactive diff between the ideal target and current application rather than a blind rewrite.

## How prior iterations failed

### Premature architecture lock-in

Several iterations assigned exact responsibilities to PostgreSQL, DuckDB, Weaviate, Neo4j, and SurrealDB before examining the current application. Some treated design suggestions as owner-approved contracts. The reconciled prompt starts from current repository authority and requires an independent target design followed by a fit-gap comparison.

### Meaning drift

Earlier responses confused “SAT” with unrelated concepts before returning to Structure-Aware Temporal retrieval. Other responses converted a knowledge “pass” into a table, lane, or destination. The corrected prompt defines a pass as a retrieval horizon and requires the meaning of SAT to be verified per source.

### Over-specification

Some versions prescribed exact passes, schemas, fingerprints, categories, and graph ontologies. That would cause a research agent to validate the prompt instead of investigating the problem. The reconciled prompt fixes invariants and acceptance criteria while leaving taxonomy, model choice, retrieval strategy, and graph shape open to evidence.

### Over-compression

The latest prompt recovered the broad arc but lost operational distinctions: the one authored spine, source clocks, realization links, activity atomicity, projection ownership, source/work-product layers, competing explanations, and exact evaluation requirements. These details are restored.

### Evidence laundering

Exploratory outputs were sometimes described as facts or exhaustive extraction. The corrected prompt requires source-bound observations, adverse evidence, uncertainty, human promotion, and a fact-to-proof trace. It prohibits treating generated summaries or graph edges as evidence.

### Repository substitution

The application was incorrectly conflated with other Case Bible locations. The corrected scope names only `E:\AI_Workspace\Projects\the-platform-workspace\probata` as the application repository and makes physical corpus consolidation a separate, excluded task.

## Treatment of each supplied artifact

| Artifact | Useful contribution | Limitation or risk | Treatment in final prompt |
|---|---|---|---|
| `I need you to refine a prompt.md` | Requirements history and owner corrections | Contains conflicting assistant interpretations | Primary source for intent; owner corrections control |
| `research-agent-prompt.md` | Strong research arc, ingest-first framing, temporal products, checkpoints | Stale stack assumptions, excessive compression, calls probes “ground truth” | Adapted, not adopted verbatim |
| `case-extract-01.md` | Candidate fact/proof/exhibit and exposure-oriented outputs | Unverified case assertions and model interpretation | Diagnostic probe |
| `case-extract-02.md` | Missing-source and evidence-procurement orientation | Can blur allegation, inference, and fact | Diagnostic probe |
| `case-extract-03.md` | Antecedent reconstruction, cycle detection, documentation gaps | Output shape not proven compatible with Probata | Target-capability probe |
| `extraction-recipe-v1.md` | Atomic mentions, hedges, source/speaker/turn/verbatim, coverage thinking | Controlled vocabulary and multi-pass design are unvalidated; AI-chat centric | Candidate mechanics to test |
| `FULL CASE EXTRACTION — ALL TIMELINES, EVENTS, STRA.md` | Breadth and categories a human may want surfaced | Overconfident facts, motive/diagnostic language, weak item-level proof, false exhaustiveness | Stress test and anti-pattern source |
| `Copy of Michigan Custody_ Digital Evidence Standards.md` | Legal issue checklist for authentication, hearsay, foundation, summaries, procurement | Derivative and potentially outdated or overconfident | Research seed requiring official primary authority |
| `Passage of Time (8 Years)_ Under Michigan administ.md` | Raises remoteness, current nexus, and changed-condition questions | Mixes legal domains and contains unsupported conclusions | Issue-spotting seed only |
| `I wouldn't even say monitored because for the most.md` | Demonstrates why acquisition language and framing matter | Euphemistic relabeling and premature privacy/spoliation conclusions | Adversarial legal/framing example |
| `SAT-Graph RAG on a Weaviate + Neo4j + SurrealDB Stack...md` | Structure/time/provenance and cross-store questions | Overcommits topology and may contain unverified product claims | Technical research seed |
| `Understanding Multi-Query Retrieval Examples.md` | Query expansion, temporal filtering, workflow durability, HITL | Expansion can inject assumptions; UTC advice can erase raw time; claims need current verification | Retrieval research seed with audit controls |

## Probata-specific corrections incorporated

The final prompt requires the research agent to verify and work within these current application concepts:

- Indicia Probata is the processing application; `proffer` is its import lane and the Workbench is the operator surface.
- The Workbench is the browser-first review surface; `intake` is a distinct desktop ingest client that drives Proffer.
- `indagatio` is a planned analytical split, not something to assume has already happened.
- The distributed Case Bible/vault is not to be reorganized during this task.
- The central mechanism is a forward walk over changing knowledge horizons compared with hindsight; the delta is the deliverable.
- A horizon is a pre-retrieval permission/filter, not a duplicated corpus.
- One authored normalized spine supports governed derived projections.
- Event time, source availability, and realization/discovery are separate concepts.
- Source evidence, derived records, analytical work product, and human decisions are separate layers.
- Current rulings distinguish context-ingest fingerprints from H1/H2/H3 evidentiary custody, which begins at owner-controlled promotion.
- Each parser, hasher, normalizer, chunker, classifier, projection writer, and promotion step must remain an atomic activity-compatible unit.
- Current store and orchestration responsibilities must be reverified rather than copied from the older prompt.
- Current conclusions must distinguish a ruled target, code present in the dirty checkout, local test proof, and live/deployed proof.
- The research must account for the 26-stage Proffer graph, recent D-130 through D-153 owner rulings, snapshot/rebuild database doctrine, and material documentation drift without assuming those items are all fully implemented.
- Retrieval context must be platform-issued and horizon-prefiltered; agents may not invent or widen a horizon or query projection stores directly.
- Native Weaviate retrieval appears to enforce the required prefilters, but it is feature-gated and the latest located live receipt showed an empty evidence projection. The older Agno route appears to post-filter horizon after top-k and must not be used as fallback.
- Existing walk derivation has definitions and tests but no located production research-agent caller; Semantica is candidate-only; Surreal/Neo4j promotion and governed walks remain target capabilities pending live proof.
- Local tests and parser counts do not establish a live end-to-end workflow.
- The current coercive-control analysis prompt is an explicit audit target because it may force clinical framing, examples, intent assumptions, and legal-advice language while omitting positive/neutral controls.

## What the final prompt deliberately does not decide

The research brief does not predetermine:

- the final behavior ontology;
- the number of horizon experiences or model passes;
- the classifier technology;
- an exact graph ontology;
- a new database or framework;
- which exploratory extraction fields become durable contracts;
- whether an analytical capability remains in Probata or moves into Indagatio;
- a corpus relocation or Case Bible consolidation plan.

Those decisions require evidence from the actual application, representative source material, current primary research, measured experiments, and owner review.

## Static implementation conflicts preserved for the research agent

The prompt records—but requires fresh verification of—several material implementation conflicts found during reconciliation:

- the active FastAPI route still appears to use a monolithic Python ingest path with ingest-time evidentiary custody, while newer decisions require context-first ingest and promotion-time custody;
- a separate, substantially implemented Go Proffer path has 26 fail-closed activities, but it has not converged with the Python route;
- parser/accounting work is real, while the ruled decoder subtree and structured-ELT registration remain incomplete;
- normalization is currently generic and one-record-to-one-record;
- classification is split among stale/direct and Temporal/HITL paths, including a local-ONNX path that must not be revived;
- the final-form schema snapshot appears to retain `stored_bytes`, duplicated classification histories, and an orphan target inconsistent with newer rulings;
- current focused tests could not establish a green baseline during this reconciliation.

These findings matter because the remaining bottleneck may be bounded execution and live proof—not another wholesale architecture design. The research brief therefore requires the agent to separate already-decided repair work from genuinely unresolved research questions.

## Coverage check

The reconciled prompt explicitly covers:

- exact application repository and non-scope;
- artifact authority and conflict handling;
- probe—not-ground-truth—treatment;
- knowledge-horizon mechanism and contamination testing;
- one authored spine and derived projections;
- source clocks and realization links;
- evidence/work-product/human-decision boundaries;
- positive and negative behavioral coverage;
- antecedents, cycles, gaps, deltas, adverse facts, and fact-to-proof trace;
- current implementation and live-proof inspection;
- activity atomicity and architecture roles;
- cost-aware model funnel;
- temporal, vector, relational, and graph retrieval;
- Michigan primary-authority verification;
- gated research phases;
- current-to-target diff and bounded proof slice;
- privacy, provenance, uncertainty, and no-delete guardrails.
