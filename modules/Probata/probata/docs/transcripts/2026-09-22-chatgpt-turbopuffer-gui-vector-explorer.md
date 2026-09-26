---
title: turbopuffer-gui as a vector preview / navigation explorer for Probata and Intake (ChatGPT assessment)
date: 2026-09-22
source_service: ChatGPT (owner pasted at 09:15 EDT into the Probata Workbench chat; "might be helpful")
captured_by: Claude Code · Fable 5.1 · session 77aa963a
status: reference input, not a decision; nothing built from it
domains: [probata, consignatio, intake, workbench, vectors]
tags: [transcript, turbopuffer-gui, vector-explorer, weaviate, regl-scatterplot, embedding-map, reference]
---

> _Byline: saved verbatim by Claude Code · Fable 5.1 · 2026-09-22 under the owner's transcript rule. Owner questions in the exchange: "could this be adapted for my stack or app", "more wanted it for vector preview and navigation for probata and intake", "or / and / both an external advanced vector mgmt app". Repo: https://github.com/MrPeker/turbopuffer-gui (MIT). Nothing below the rule is edited; ChatGPT's own thinking/status lines are kept as pasted._

---

https://github.com/MrPeker/turbopuffer-gui    could this be adapated for my stack or app

ThinkingThinkingI’ll inspect the repo’s Turbopuffer dependencies, then map what’s reusable for TraceIQ/Indicia Probata and what would need replacing for your storage stack.

Identified TraceIQ stack, UI, databases, and deployment choices

Finalized the scope

Structured the blueprint

The strongest fit is a provenance-aware Docstore/Probata explorer using existing indexes and schema metadata. I’d also replace its Turbopuffer query layer and insecure Electron networking path.

Assembled the recommendation

more wanted it for vector preview and navigation for probata and intake

Stopped thinking

or / and / both an external advanced vector mgmt app

Worked for 5m 50s

Yes. This could be adapted into a useful data/context explorer for your stack—and there is enough reusable interface here to make it worth considering. I would reuse its browsing interface, not adopt Turbopuffer or reshape your databases to fit it.
For TraceIQ/Indicia Probata, the strongest use would be an inspection console where you can see what was ingested, inspect the resulting records, test retrieval, and follow records back to their sources. It would not replace your ingestion pipeline or application logic.
I inspected the repository’s connection handling, document services, state store, desktop configuration, dependencies, and license. This is a source-level feasibility assessment; I have not built it or connected it to your running services.
What is actually reusable
The project uses Electron, React 19, TypeScript, Tailwind, Radix UI, and Zustand. That gives you a substantial desktop-interface foundation rather than just a mockup. Its existing dark theme is also reasonably close to the direction you prefer.  GitHub
Here is how I would map its existing interface to your application. The right-hand column is proposed adaptation, not functionality it already understands.
Existing piece	Useful adaptation for your system
Connection and namespace browser	Select an approved service, knowledge base, or dataset without mixing their ownership and scope.
Document table and details inspector	Browse source records, normalized messages, chunks, classifications, and persisted processing artifacts.
Visual filters and saved filter history	Filter by source, participants, dates, processing status, classification, or evidence status.
Full-text, vector, and hybrid-query controls	Inspect and test results from your existing retrieval service.
Schema, aggregation, and query-performance interfaces	Inspect record structure, coverage, missing metadata, and retrieval behavior.
Import/export interfaces	Reuse selected presentation components, but route operations through your application’s controlled ingest/export workflows.

The document page already assembles a table, details panel, visual/raw query controls, aggregation results, and query-performance display. Its state store includes saved filters, pagination, and browse/full-text/vector/hybrid modes. Those are meaningful pieces of interface work you would not need to start from scratch.  
Where it fits your stack
I’m mapping this against the more recent architecture we have discussed, including SurrealDB, DuckDB, CocoIndex, and the separation between Docstore, ccc, and Intake—not assuming every database from the older Supabase/Neo4j/Qdrant plan is still deployed.
Docstore: a useful, relatively contained first target
For your documentation-only Docstore, this could become the interface for examining indexed documents and chunks, source paths, metadata, and search results.
I would keep its dataset selection constrained to Docstore’s approved documentation roots. It should not quietly become a whole-vault browser or start treating repository code as Docstore material. ccc remains the separate codebase-indexing tool.
A SurrealDB integration is technically reasonable: SurrealDB supports document, graph, vector, and full-text data, and its JavaScript SDK supports connections to local or remote instances. However, that does not make its API interchangeable with Turbopuffer’s. SurrealDB
Indicia Probata: the more valuable application-specific adaptation
For the actual case/context application, I would extend the record inspector into a source-and-provenance inspector.
For example, selecting a normalized message could show its preserved text, participants, event time, originating export or source file, source hash/version, related screenshot or call-log references, extraction history, and any later observations. The interface should distinguish what the source says, what a process extracted, and what an agent inferred.
Your work-product, messaging, and source-data knowledge bases could be separate views within one interface. That does not require merging their storage or pretending that a screenshot, a normalized message, and an analysis document are the same kind of object.
Your graph relationships, “as-lived” and “hindsight” views, and evidence-promotion workflow would still need application-specific integration. Replacing a database adapter does not supply those semantics.
Intake and the vault catalog: browse the catalog, not the bucket
For your large vault, I would connect this to existing catalog records, manifests, receipts, and processing logs.
Opening a screen should not initiate a remote bucket scan, download the vault, or regenerate embeddings. The interface should display what is already known and clearly distinguish “cataloged,” “selected,” “ingested,” “processed,” and “failed” according to your actual pipeline states.
Where Intake uses a different backend, such as Weaviate, that should remain behind its own service boundary—not become a reason to migrate anything.
The adaptation is more than changing an endpoint
This is the main technical catch.
There is a turbopufferService.ts, which provides a useful starting point for separating backend access. But the connection context also exposes a Turbopuffer SDK client directly, and the document service accepts and returns Turbopuffer-shaped queries and results. Its query calls use fields such as rank_by, top_k, aggregate_by, and consistency; browsing pagination is implemented with an ID-based filter.   
So this is a moderate refactor for a focused explorer, not a configuration change. The main work would be:
1. Replace the connection and query contract. The UI should request operations such as “list datasets,” “browse records,” “search,” and “get record details” through your application service—not receive a database SDK client.
2. Translate backend-specific behavior. Filters, pagination, schema information, ranking scores, and aggregations need deliberate mappings. Unsupported capabilities should be unavailable rather than silently approximated.
3. Add your record relationships and workflow actions. Source lookup, provenance inspection, processing history, and explicit evidence promotion belong in your application layer.
The intended boundary would be:
Adapted interface → your application API/session broker → existing databases, retrieval services, and source store.
That does not require a new microservice. The interface can use an existing service and add narrow endpoints where they are genuinely missing.
For a local DuckDB-backed component, I would preserve your broker approach rather than let every GUI window or agent open the active database file independently. DuckDB’s in-process read/write model supports concurrency within one owning process; it is not simply “only one connection,” but the process/file ownership distinction matters here. DuckDB
Two things I would change before using private data
The desktop networking and credential boundary
The inspected src/main.ts explicitly disables web security, including webSecurity: false, and restricts outgoing destinations with an allowlist. The connection context retrieves decrypted API credentials into the renderer to initialize the SDK. Those are important details—not just theoretical concerns from the README.  
For your adaptation, I would keep privileged database credentials outside the renderer, use narrowly defined IPC or authenticated application endpoints, and restore normal browser security. Electron’s own guidance explicitly recommends against disabling webSecurity in production. Merely adding your servers to the existing allowlist would not be the fix. Electron
The repository also labels itself beta and warns against production-data use, so I would treat it as reusable source, not a trusted production client to point at your originals unchanged. GitHub
Generic editing must not bypass your provenance model
The application exposes document updates, imports, deletes, and full-namespace exports. Its document service checks an application-side write-permission flag before upserts. Those facilities would need to be reconciled with your own authorization and workflow boundaries.  
For your first adaptation, I recommend read-only access enforced by the backend, not merely disabled buttons. Any later write action should preserve your established model:
Context remains context until you explicitly promote it; originals remain immutable; annotations and processing outputs remain traceable to their sources.
Likewise, I would return paginated metadata and excerpts rather than full documents and vector arrays by default. Semantic searches should use the existing query-embedding/retrieval path; browsing should not create a second ingestion or embedding system.
Desktop app or part of your existing frontend?
As a separate Windows inspection tool: retaining Electron is the more direct reuse path.
Inside an existing web application: reuse the React components, but replace Electron-specific connection storage, file dialogs, exports, and IPC. The current connection context depends directly on window.electronAPI, so this is not a web app that can simply be deployed unchanged to your VPS. 
The repository is MIT-licensed, permitting modification and reuse subject to retaining the required copyright and permission notices. 
My recommendation
Worth adapting for your application-specific explorer; less compelling just to obtain a generic database GUI. SurrealDB already offers an official Studio interface for querying, visualizing, and managing its data, so ordinary database administration alone is not a strong reason to maintain a Turbopuffer fork. SurrealDB
The useful first deliverable would be one complete read-only workflow: choose an approved dataset, browse or search records, inspect a result, and follow it back to its source and processing history—without scanning, re-indexing, copying the corpus, or changing your architecture.
That is where this repository could save you real interface work: making your existing system visible and inspectable, rather than becoming another system you have to build and maintain.

Worked for 4m 30s

Yes—both. I would make it one shared vector explorer that can run inside Probata and Intake, with a standalone workbench exposing the advanced management tools. Not two separate forks, and not a reason to migrate your data to Turbopuffer.
The standalone management app is actually closer to this repository’s existing purpose. The embedded version would reuse its browsing and inspection components without bringing along the entire desktop application. It is an Electron/React/TypeScript client, rather than an already-packaged embeddable widget. GitHub
What this repository actually gives you
There is an important distinction between inspecting a vector, searching by vectors, and visually navigating an embedding space.
Capability	What I verified in this repository	Value for your use
Document browsing and inspection	A document table, filtering, detail panel, and structured/JSON views.	Useful foundation for browsing indexed chunks and inspecting their metadata.
Individual vector preview	Shows dimensions, magnitude, and a sample of numeric values.	Useful technical inspector, but not a visual map of related material.
Vector-search controls	Accepts pasted/uploaded vectors, selects a vector field, and exposes ANN/kNN controls.	Reusable interaction patterns; the underlying queries need adaptation.
Search experimentation	The document store includes browse, BM25, vector, and hybrid modes, with client-side reciprocal-rank fusion.	A starting point for a retrieval-debugging workspace.
A navigable 2D/3D embedding map	I have not found a working implementation in the inspected source. The confirmed vector preview is numeric.	Treat projection, point selection, and neighborhood navigation as additional work—not something the repository already solves.

So: it is a useful source of browser, inspector, and query-workbench components. It is not a finished visual vector explorer that we merely point at another database.
Its data layer is also explicitly Turbopuffer-specific: the connection context holds a Turbopuffer SDK client, and the document service directly constructs Turbopuffer queries. Replacing one endpoint URL would not make it a Weaviate or general-purpose vector client.  
How I would use it across your three surfaces
These are proposed adaptations, not features I am claiming are already implemented.
Surface	Main job	What it should let you do
Probata	Explore indexed material in context.	Select a message, chunk, document, or extracted record; find similar material; open the surrounding conversation and preserved source; move between semantic neighbors and explicitly recorded relationships.
Intake	Inspect what ingestion produced.	Review parsing and chunk boundaries, inspect embedding status, compare a selected sample with existing material, and identify possible duplicates or malformed output before authorizing broader processing.
Standalone advanced workbench	Inspect and manage the indexing/retrieval system itself.	Browse collections, inspect vector spaces and index configuration, test searches, compare retrieval methods, investigate missing/stale vectors, and run explicitly selected maintenance operations.

The useful interaction would be:
Select an item in Probata → open its vector neighborhood → inspect another item → open that item’s complete source context → continue from there.

In Intake:
Select an ingestion run → inspect its chunks → check which have embeddings → preview related existing material → review issues without automatically processing the entire corpus.

That fits your earlier Intake requirement to preview approximately ten records, inspect another batch, and adjust mappings or parsing before full processing. The vector view should extend that preview, not bypass it. Claude - chat pipeline for PostgreSQL - Claude.mdMD
An important implementation detail: unembedded input can still have a source/chunk preview. A vector preview would use existing embeddings or an explicitly selected sample embedding run. Opening the explorer should not silently initiate a full embedding job.
One shared core, not another ingestion system
Using your more recent Weaviate, Surreal, and DuckDB-readable artifacts direction as the design basis—not treating the older Qdrant architecture as automatically current—I would separate the explorer into three parts.
1. A reusable explorer interface
This would contain the collection browser, filtered results, vector inspector, source preview, neighborhood/map view, and saved selections.
It should accept a current scope—such as a collection, ingestion run, document, or selected record—from whichever app opened it. It should also provide an “Open in full workbench” action that preserves the selection and filters.
Where the host frontend is compatible, it can be embedded as a component. Otherwise, a separate web route with deep links can provide the same navigation without requiring a frontend rewrite.
2. An adapter to your existing services
For your current direction, I would start with one useful adapter to your actual vector backend, rather than building a universal connector framework for every database you have ever considered.
Weaviate already supports searching from an existing object, searching from an explicit vector, and selecting a named vector target. That provides the underlying operations for “show me neighbors of this item” without re-embedding that item just to navigate from it. Weaviate Documentation
Your surrounding services would supply the rest:
- Source and metadata resolution: turn an index hit into its normalized record, parent conversation/document, source locator, and provenance.
- Graph navigation: retrieve explicitly stored relationships separately from semantic similarity.
- Persistent artifacts: save projection results, sample manifests, comparison reports, and saved exploration sessions through your existing persistence layer.
The explorer should not own a second copy of the corpus or introduce a parallel embedding pipeline.
3. Different permissions for exploration and administration
The embedded view would primarily expose reading, navigation, and review actions. The standalone app could expose advanced operations where the backend supports them.
For Probata-managed data, I would route repairs and re-embedding through the owning pipeline so the source, derived records, and index remain aligned. That still permits real management; it avoids an independent GUI quietly changing index rows behind the pipeline’s back.
The visualization layer I would add
For actual visual navigation, I would add a point map synchronized with the results table and source inspector.
Selecting a point should select its record. Selecting a table row should highlight the point. A lasso selection should produce a reviewable set of records—not merely change the picture.
A concrete reusable option is regl-scatterplot. Its current project documents WebGL rendering, pan/zoom, individual and lasso selection, visual encodings, and point connections. I would evaluate it as the 2D rendering component rather than write a scatterplot engine from scratch. It is a renderer, however—not an embedding generator or projection service. GitHub
For your application, I would distinguish three views:
Semantic neighborhood. Show the selected item and neighbors returned by the vector backend. This would be my first navigation feature because it directly connects a selected record to related material.
Corpus or batch map. Display a saved projection of a defined collection or sample, with visible filters, scope, and projection version.
Relationship view. Show actual source/parent-child/entity relationships from your records or graph. Those connections should be visually distinct from “these embeddings are similar.”
A projected map is exploratory, not proof of a relationship. UMAP’s own documentation warns that it can create apparent separations and does not completely preserve density. I would therefore use backend vector results for neighbor rankings and label the 2D layout as a projection—not calculate semantic relevance from screen distance. UMAP Documentation
What would make the external app genuinely “advanced”
I would prioritize three groups of functionality.
Inspection and quality checks. Show the embedding model/version, vector field, dimensions, metric, source record, ingestion run, and embedding status. Provide scoped checks for missing or invalid vectors, inconsistent model metadata, broken source links, and records whose embeddings no longer match their current derived content. Keep hash-confirmed duplicates separate from similarity-based duplicate candidates.
Retrieval comparison. Run the same query through vector, keyword, hybrid, and reranked retrieval where supported. Show which records each method returned, their order, the filters applied, and the score type. Save the comparison so you can inspect why relevant material was missed or why unrelated material ranked highly.
Controlled maintenance. Support selected-record re-embedding, versioned index rebuilds, configuration inspection, exports, and backend-supported administrative actions. Preview the affected scope and persist the outcome. Collection deletion or bulk mutation can exist in the advanced app, but should be separate from ordinary browsing and enforced by the backend—not merely hidden behind a UI toggle.
I would also partition comparisons by an explicit embedding-space identity: model/version, vector target, dimensions, and relevant preprocessing configuration. Two collections should not be treated as directly comparable merely because their vectors have the same length.
Keep it within your memory constraints
My proposed implementation would send the browser point coordinates, stable IDs, and compact display metadata, then load full text and vectors on demand.
Projection work would use bounded jobs with saved results. Filtering or reopening a view would reuse those artifacts where applicable, rather than fetching every full embedding and recomputing the layout in the UI. The display should always state whether you are viewing the full scoped dataset, a sample, or a neighborhood.
Do not inherit the desktop security shortcut
The inspected Electron entry point disables webSecurity, and the connection context retrieves decrypted credentials into the renderer. I would replace that arrangement with narrowly scoped API or main-process operations before using a derivative with your material. Electron’s own security guidance explicitly advises against disabling webSecurity.   Electron
My recommendation
Build the standalone workbench around a reusable explorer core, and expose that same core inside Probata and Intake. Reuse selected Turbopuffer GUI components rather than carrying the entire application forward unchanged.
The first useful implementation would be collection browsing → record inspection → semantic neighbors → complete source context, with the visual map added to that same selection/navigation model. That provides value in both embedded and standalone use without making a large visualization project a prerequisite for basic navigation.
The repository is MIT-licensed, so component reuse and modification are permitted subject to retaining the required notices. 
It is worth borrowing from—but the biggest value for you would come from connecting vectors back to their records, context, and sources, not from reproducing a generic database dashboard. This assessment is based on source inspection; I have not built the repository or tested it against your live services.

ChatGPT can make mistakes. Check important info.
