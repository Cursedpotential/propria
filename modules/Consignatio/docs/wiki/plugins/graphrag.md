---
title: "graphrag"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# graphrag

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

GraphRAG design, build, deploy and operations toolkit — the full graphrag.com Pattern Catalog (11 retrievers, 14 graph shapes, gram notation), the mid-2026 framework/backend landscape, corpus acquisition, multi-stage retrieval composition, temporal/bitemporal knowledge-base design, deployment topologies with working compose stacks, and evaluation.

Source: `E:/AI_Workspace/plugins/plugins/graphrag`. Version: `0.2.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `graphrag-audit`

Audit an existing GraphRAG implementation against the pattern catalog and find silent failures

```text
/graphrag:graphrag-audit
```

Source: [graphrag-audit.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/commands/graphrag-audit.md:1>) · SHA-256 `d9959551debded027f96499454f3b1053bf59c099762444b960c66c95edfb94e`

### `graphrag-choose`

Recommend a GraphRAG pattern, graph shape, framework and backend for a described situation

```text
/graphrag:graphrag-choose
```

Source: [graphrag-choose.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/commands/graphrag-choose.md:1>) · SHA-256 `ec59bf6b98e29c5fdaed56e3be599c3e8edbb749827f5cbc0b2549d08313e526`

### `graphrag-design`

Design a knowledge graph model for a domain, emitting a gram pattern and Cypher DDL

```text
/graphrag:graphrag-design
```

Source: [graphrag-design.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/commands/graphrag-design.md:1>) · SHA-256 `b0196cd2afbd5a2d5a215fdc670ad89f5599ce07a30735047b62984202e7aa43`

## Skills

### `graphrag-build`

End-to-end GraphRAG design workflow — from requirements to a working retrieval system, including the iterative and multi-stage retrieval composition the pattern catalog deliberately leaves out. Use when designing a GraphRAG system from scratch, deciding whether a graph is warranted at all, composing multiple retrievers, building an agentic retrieval loop, sequencing a build, or when the user says design a GraphRAG, build a knowledge graph RAG, multi-hop retrieval, iterative retrieval, agentic retrieval, or how do I put this together.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-build/SKILL.md:1>) · SHA-256 `27b7dbf1ee0c54b10a6a7e16558a67ec91f50d1fce88c5c7c94273b2d752e654`

### `graphrag-deploy`

Deployment topologies for GraphRAG knowledge bases — from a laptop to a split multi-node tier, with concrete compose files, sizing, and operational gotchas. Covers embedded, single-VPS, Postgres-only, split-tier, managed-cloud and federated zero-ETL architectures, plus backup, migration and capacity planning. Use when standing up or sizing a knowledge base, choosing between one box and a split tier, writing compose for Neo4j/Memgraph/Weaviate/Qdrant, planning RAM for an in-memory graph, or when the user says deploy, self-host, compose, sizing, capacity, backup, or production.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-deploy/SKILL.md:1>) · SHA-256 `b2b536d7ae96db061a17bf142f6cc8613fcaee1e62ac3c4368da7b85d089f745`

### `graphrag-frameworks`

Choosing a GraphRAG implementation and a graph database backend. Covers Microsoft GraphRAG, LightRAG, nano-graphrag, Fast GraphRAG, neo4j-graphrag-python, LlamaIndex PropertyGraphIndex, LangChain GraphCypherQAChain, Graphiti, Cognee, KAG, RAGFlow and the Memgraph AI Toolkit, plus Neo4j, Memgraph, FalkorDB, ArangoDB, NebulaGraph, Amazon Neptune, Postgres+AGE and the archived Kuzu. Use when picking a GraphRAG library or graph database, comparing indexing cost, migrating between frameworks, or when the user says which framework, which graph database, Microsoft GraphRAG, LightRAG, Graphiti, Cognee, Neo4j vs Memgraph, or FalkorDB.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-frameworks/SKILL.md:1>) · SHA-256 `3eb203e0431fab62226f420ee73c06b199abfaf7fa8a77ffc12319aaa95b73cc`

### `graphrag-ingest`

Corpus acquisition and ingestion for GraphRAG — turning sources into a lexical graph. Covers the live-retrieval vs ingested-index decision, source adapters (Nimble, Bright Data, Tavily, Exa, Firecrawl), chunking strategies, entity extraction and resolution, provenance, dedup, and content validation. Use when deciding how to get a corpus, building an ingestion pipeline, choosing chunk sizes, picking an NER approach, wiring a web-search or scraping provider into a RAG system, or when the user says ingest, corpus, chunking, entity extraction, NER, crawl, scrape, or embedding pipeline.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-ingest/SKILL.md:1>) · SHA-256 `11a8ffb52a04a13f318578887ea873f9702947c7c44d84de354e78af15aabdf3`

### `graphrag-operate`

Evaluating, debugging and tuning a GraphRAG system that already exists. Covers retrieval-quality triage, the evaluation metric set (recall@k, groundedness, coverage, latency), silent-failure detection, cost control, and re-indexing strategy. Use when GraphRAG results are poor, empty or hallucinated, when measuring retrieval quality, when indexing costs are too high, when deciding what to re-index, or when the user says my RAG is bad, retrieval quality, evaluation, groundedness, hallucination, or GraphRAG is too expensive.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-operate/SKILL.md:1>) · SHA-256 `d652e65217d66e9a59a30de81b0625ee5f06cb7878fb24022d05cbcababa2f80`

### `graphrag-patterns`

The GraphRAG Pattern Catalog — 11 retrieval patterns, 14 knowledge-graph shapes, and the gram notation they are written in. Use when choosing or explaining a GraphRAG retriever, deciding what shape a knowledge graph should take, mapping a question type to a retrieval strategy, reading a graph pattern written in gram, or when the user says GraphRAG, lexical graph, domain graph, memory graph, community summaries, Text2Cypher, parent-child retriever, hypothetical questions, local/global retriever, or knowledge graph model.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-patterns/SKILL.md:1>) · SHA-256 `6dc80b98097425b9e9715e3c5e3303a98573bbc3a8628f17edd6ad3e243819ef`

### `graphrag-temporal`

Designing temporal knowledge bases — knowledge graphs where facts have validity periods, get superseded, and can be queried as-of a point in time. Covers valid time vs transaction time vs bitemporal modelling, the two graph implementation strategies (timestamped edges vs version chains), contradiction and invalidation handling, as-of query patterns, and system selection across Graphiti, XTDB, Datomic, Dolt, SQL:2011 tables and hand-rolled graph models. Use when facts change over time, when you need point-in-time or audit queries, when building agent memory that supersedes itself, when modelling evidence or case history, or when the user says temporal, bitemporal, as-of, point-in-time, versioning, audit trail, valid time, or time travel.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/graphrag/skills/graphrag-temporal/SKILL.md:1>) · SHA-256 `2dd637d005b48defe4c9335024f7c8dbf29f0ef0c1032b2c6d5d2dae514f1a8d`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

No entries found in the inspected declarations.

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
