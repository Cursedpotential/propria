# Family Court Workbench â€” shared store operations

> Byline: OpenAI Codex Â· GPT-6-Luna Â· 2026-10-04.

## Store and connection

The Family Court case and reference library lives in the hosted `surreal-case` SurrealDB service,
namespace `fct`, database `case`. The phone surface, native Workbench, Advocatio Workdesk and
hosted Family Court tools use the same versioned records. The Workbench does not own a second
local case database.

The Workbench sidecar loads the built store module from `FAMILY_COURT_PLUGIN_ROOT` (default:
`E:/AI_Workspace/plugins/plugins/family-court-toolkit`) and supplies its shared database
configuration for record reads. Native library mutations go through the authenticated hosted
ContextForge tools; both paths reach the shared store, and the desktop does not spawn a local MCP
server for those mutations. Set `CUSTODY_CASE_DB` in the process environment or in
`~/.secrets/family-court-toolkit.env`. The value must select the hosted SurrealDB endpoint. The
sidecar reads that one assignment with a line parser; it never evaluates the secrets file or
prints its values. Database credentials are resolved by the canonical store code from the
environment or secrets and are never part of browser requests.

Missing or invalid shared-store configuration is an error with a visible reason. There is no
implicit local-file fallback. `FAMILY_COURT_PLUGIN_ROOT` selects the plugin code location; it
does not select a database. Keep endpoint credentials outside Git.

## Private case records and versions

Keep complete private case context in the shared records: full names, aliases, child names,
narratives, custom fields and source context. Initials and ages may be retained as additional
fields; they do not replace names or other authored content. Do not put private case content in
Git or public documentation.

Read an exact record with the shared `case_record` contract before editing it. Its canonical
`table:id`, complete record body and `sha256:` version are the common identity and version seen
across the surfaces. Personal edits use `case_put` with that exact `expected_version`; use
`absent` only when intentionally creating a new explicit record id. A stale version conflicts
without writing. Successful versioned edits retain the before and after records in the shared
revision history. Send only changed fields, retain unknown fields, and use explicit `null` to
clear a value. Personal source documents use `kind: case_document`; legal authorities and
published references cannot use this personal edit path.

The canonical version is computed by the store and excludes the embedding field. SurrealDB may
display identifiers containing hyphens with quoted-identifier brackets; the store removes that
transport formatting so a displayed `table:id` can be supplied back unchanged.

## Legal sources and references

Legal authorities and reference-library changes use the governed proposal and publication
lifecycle, not a direct personal-record write. `library_propose` retains the complete proposed
record, exact target version, rationale, and claim citations while the published record remains
unchanged. Each citation binds a source id and exact source version to a pinpoint and claim.
Uncited retained imports remain available as citation-required drafts rather than being
discarded.

Validation is performed by the trusted server workflow. Its receipt is a separate
`library_validation:<UUID>` record tied to the full `library_proposal:<UUID>` and proposed hash.
A queued or completed dispatch alone does not authorize publication. `library_publish` succeeds
only when the stored receipt still matches the proposal, citation source versions and claim
checks, and reports `VERIFIED_PRIMARY` with currency `cleared`. Publication checks versions
atomically and retains the prior full record as a library revision. Ordinary clients cannot
set trusted validation or currency fields; conflicts leave the proposal available for review.

Current tool names, schemas and selection guidance live in the canonical plugin docstrings and
registered schemas in `mcp-app/src/store-tools.ts` and `mcp-app/src/server.ts`. The transaction
and revision rules are implemented in `mcp-app/src/case-library.ts`. Use those as the API
reference; this README intentionally does not maintain a duplicate tool catalog.

## Permanent legal files and synchronization status

The permanent legal home is
`b2://salem-data/consignatio/casevault/KnowledgeBase/legal/`. Preserve complete files as units,
with their sidecars and working internal links. Keep ZIP recovery originals separately. A
shared-record version is not the same thing as a file-byte hash. A durable file link must identify
the provider, bucket, object key, provider version, source-byte SHA-256 and placement receipt;
legacy `source.path` or `r2_path` values alone do not prove that the B2 original exists or opens.

Two-way synchronization between Case Bible files and shared SurrealDB records is owner-approved
and under implementation, but is not deployed. Do not describe file arrival as a completed store
update, or a published record as proof that the B2 file was updated. Treat propagation as
unverified until the synchronization workflow is deployed and both sides have been read back.

## Optional embeddings

Embeddings are optional and are not required to connect to the store or run text search. The
current vector model is `nvidia/nemotron-3-embed-1b`; its dimension defaults to 2048 and can be
configured with `CUSTODY_EMBED_DIM` when the store first creates its embedding schema. The
`NVIDIA_API_KEY` or `NIM_API_KEY` is read from the process environment or parsed secrets files;
values are never printed. Text mode never requests embeddings. Hybrid search falls back to text
and reports `degraded` when no embedding credential is configured; vector-only search reports
that it is unavailable.

## SDK and native packaging

The canonical `mcp-app/build.mjs` externalizes both `surrealdb` and `@surrealdb/node`. The latter
ships platform-specific `.node` binaries that must be loaded from its real package directory;
bundling those imports into `dist/` breaks native module resolution. A runtime package therefore
needs the actual `node_modules/surrealdb` and `node_modules/@surrealdb/node` directories beside
the built `dist/` files. Preserve the package lock and use the canonical package's install and
runtime-packaging steps rather than assuming the `dist/` bundle is self-contained.

The build also keeps the Workbench's cross-cutting local store imports as runtime imports to the
shared compiled module. Do not change that boundary by independently bundling store-dependent
entry points: they would each create a separate connection cache.

## Isolated store tests only

The store accepts an explicit `mem://` override for isolated tests. Embedded or file-backed
engines belong only in tests that deliberately select a disposable target, such as a path
regression fixture. They are not production defaults or fallback targets. Tests must never
connect to or write the owner's shared case database.

## Source locations

- Canonical plugin source root (outside this repository):
  `E:/AI_Workspace/plugins/plugins/family-court-toolkit/`.
- Workbench integration: `modules/FL-MCP/sidecar/lib/store-client.mjs` and the hosted library
  client under `modules/FL-MCP/sidecar/lib/`.
- Canonical store resolution, record versions and record-id normalization:
  `mcp-app/src/store.ts` in the family-court-toolkit plugin.
- Current operation descriptions and schemas: `mcp-app/src/store-tools.ts` and
  `mcp-app/src/server.ts` in that plugin.
- Personal CAS/history and legal proposal/publication transactions: `mcp-app/src/case-library.ts`.
- SDK runtime externals: `mcp-app/build.mjs` and `mcp-app/package.json`.

This guide records code and owner-approved workflow state. It is not a live database or B2
synchronization receipt; verify actual service availability and read-back before claiming either
has completed an operation.
