---
title: Maybe-useful apps and libraries (owner list, 2026-09-22)
date: 2026-09-22
status: reference — owner-supplied list, not evaluated, nothing adopted
tags: [reference, chunking, rag, graph, extraction, weaviate, messenger, takeout, instagram, gui]
---

# Maybe-useful apps — owner list 2026-09-22 09:36

> _Byline: Claude Code · Fable 5.1 · 2026-09-22. Saved as pasted (duplicates dropped). Where each one MIGHT fit is a first reading, not an evaluation; licences and fit must be checked before any borrowing (owner rule: borrow shape, tweak for our requirements)._

| Repo | What it is | Possible place |
|---|---|---|
| [speedyk-005/chunklet-py](https://github.com/speedyk-005/chunklet-py) | sentence / code / doc chunker for RAG | super index chunking (currently a recursive splitter) — compare, don't add a second chunker |
| [KaifAhmad1/RAGHub](https://github.com/KaifAhmad1/RAGHub) | curated list of RAG frameworks | reference only |
| [vitali87/code-graph-rag](https://github.com/vitali87/code-graph-rag) | monorepo RAG on a knowledge graph | ccc / code side, not the corpus |
| [morluto](https://github.com/morluto) | a GitHub user, not a repo | owner to say which project |
| [kuzudb/kuzu](https://github.com/kuzudb/kuzu) | embedded property graph DB, vector + FTS, Cypher | would be a THIRD graph store beside SurrealDB and Neo4j — needs a reason; Surreal is the ruled file/analysis graph |
| [xberg-io/xberg](https://github.com/xberg-io/xberg) | Rust document intelligence: text, metadata, images, tables from 106 formats; CLI, REST, MCP | super index extractors (today: text-only, 19 extensions) and Probata `documents.extract-*` — strongest candidate on the list; check licence and streaming behaviour |
| [Weavit UI](https://xenoraai.github.io/weavit-ui/) | free desktop client for Weaviate | admin/inspection of the Weaviate projection; sits beside the turbopuffer-gui idea |
| [MrPeker/turbopuffer-gui](https://github.com/MrPeker/turbopuffer-gui) | vector DB desktop GUI (MIT) | already assessed: `Probata/docs/transcripts/2026-09-22-chatgpt-turbopuffer-gui-vector-explorer.md` |
| [facebook-messenger-json-viewer topic](https://github.com/topics/facebook-messenger-json-viewer) | GitHub topic | reference for Messenger viewers |
| [serogee/FB-Messages-Archive-Explorer](https://github.com/serogee/FB-Messages-Archive-Explorer#31-offline-use) | local, searchable viewer over a Messenger export | Read screen: Messenger conversation view (as SBV is for SMS) — borrow shape |
| [AminaEmenena/data-extractor](https://github.com/AminaEmenena/data-extractor) | Go CLI: Takeout Gemini conversations → JSON/CSV/Obsidian md | AI-chat lane (KnowledgeBase/ai-chats); Go, fits the engine's decoder family |
| [cifuentesedw/instagram_json_parser](https://github.com/cifuentesedw/instagram_json_parser) | forensic HTML reports from an Instagram takeout | Instagram decoder + report shape; check licence |
