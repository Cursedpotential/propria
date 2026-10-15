# TwelveLabs media tool integration - implementation and verification

Byline: Codex, 2026-10-15. Scope: owner's request to build an app or plugin for the platform, following the supplied TwelveLabs quickstart and credential-file location.

## Implemented scope

An import-light plugin for the existing `server.tools` registry. No extra service,
SDK installation, dependency change, database schema, or separate media store is
needed. The tool-runtime image already copies `server/`; its recursive discovery
exposes these functions through `/tools`, `/tools/{id}/run`, and the `atomic_tools`
MCP directory after rebuild. The generated reference comes only from function
docstrings (`scripts/generate_twelvelabs_reference.py`).

This is a tools implementation, not a new Workbench screen. It serves selected
media processing and inspection. It does not implement a new media review queue,
durable Go workflow, Activity registration, result persistence, or evidence
promotion. Existing extraction review is scoped to conversation generations and
cannot be relabeled as media review. Existing attachment streaming remains intact.

## Configuration

The operator supplied Propria's `.secrets/twelvelabs.env`. Set
`TWELVELABS_ENV_FILE` to that file's path as visible **inside the tool-runtime**,
or inject `TWELVELABS_API_KEY` as a service environment secret. The compatibility
name `TWELVE_LABS_API_KEY` is also accepted. Environment values take precedence.
The env file is parsed as literal assignments; it is never sourced or executed.
No secret values belong in source, payloads, generated references, logs or commits.

`TWELVELABS_TIMEOUT_SECONDS` defaults to 60 and accepts finite values from 1 to
300. Transport targets the fixed HTTPS v1.3 provider host, rejects redirects,
bounds requests to 128 KiB and response JSON to 4 MiB, and performs no HTTP retry.
URL upload consumes an externally accessible direct/signed HTTPS media URL; this
plugin does not generate signatures, copy originals, or upload local file bytes.

## Calling and checkpointing

Use the platform's existing authenticated execution surface. The tool ID is
`twelvelabs.<operation>`; payload contracts are in the generated reference.

Every result has `ok`, `provider`, `operation`, `material_class: context`, and
optional `source_ref`. Success preserves provider JSON in `data`, including
`_id`, status, pagination, timestamps, citation annotations and usage. Check
`ok` first, then provider status; an HTTP success with `status: incomplete` or
`failed` is not completed analysis. Failure has `error.code`, `http_status`,
`ambiguous`, and `retry_safe`, without echoing the provider body or signed URL.
Argument validation errors remain ordinary contract errors before provider I/O.

The existing Python/Go MCP transports may wrap this as
`{"ok":true,"result":{"ok":false,"error":{...}}}`. The outer flag means the
tool invocation returned; the inner `result.ok` is the provider outcome. Direct
REST callers inspect the response body's `ok`. No generic retry/failure behavior
is added by this envelope. This requirement is in every registered tool's
catalog description so it reaches callers through discovery.

Reuse an explicitly selected store. Submit one source URL, checkpoint its asset
`_id`, and poll asset status through separate invocations. When ready, add it to
the store, checkpoint the item `_id`, and poll item status separately. Only then
search or request a response. The caller owns deadlines, waits and persistence.
No hashing or custody gate is imposed on context ingestion. Preserve any existing
source locator/hash linkage in caller-owned records; optional `source_ref` is
passed through and is not a newly invented identity.

Create/submit/analyze calls can incur provider costs and are **not proven
idempotent**. Never put them under automatic Activity retries without durable
reconciliation. A lost write response is marked ambiguous; list stores/items or
use the provider console to reconcile before resubmitting. Asset reconciliation
currently requires the provider console if its ID was lost. This implementation
does not supply an exactly-once guarantee.

Jockey can access the entire chosen store. Its selections are prompt guidance,
not an authorization boundary. Separate authorized corpora at the store level.
Public-web search is disabled by `tools: []`. Generated text/structured output
remains unreviewed context; a JSON Schema request is not independent validation.

The generic facade and MCP currently call registry functions directly, without
enforcing `execution_policy`. This plugin does not rely on a manual-approval
declaration or introduce a new approval flow. Existing authentication and caller
authorization remain necessary. No endpoint or authentication settings changed.

## Verification and current limitations

- 21 isolated unit tests passed, covering wire shapes, registry declarations,
  image indexing, pagination, citation/status preservation, literal secrets
  parsing, bounded responses, redirect rejection, redacted errors, and ambiguous
  writes surviving JSON serialization. These do not establish live operation.
- A read-only stores-list attempt loaded the supplied env-file credentials but
  returned sanitized `transport_failed`. No provider authentication, indexing,
  response generation or charges were verified.
- The canonical desktop checkout could be read, but Git access to it reported
  permission denied. A shell clone failed with `getaddrinfo() thread failed to
  start`. Changes were developed against the GitHub-fetched registry baseline
  in an isolated checkout. No concurrent checkout or synced project source was
  modified.
- Full repository tests, deployed catalog discovery, Coolify rebuild, and end-to-end
  media processing remain unverified. Do not report this as deployed production.
- The review branch `codex/twelvelabs-media-tools-20261015` was created on GitHub,
  but publishing the implementation tree was rejected by automatic approval
  review as export of private code/internal documentation to an unverified
  external destination. No implementation commit or pull request was published.
  Explicit owner approval is required before retrying that publish. The tested
  files are retained locally and packaged for owner review.

Before release: run the repository checks; rebuild the tool-runtime from the
reviewed branch/merge; inject the secret server-side; verify the nine catalog IDs;
perform a read-only store listing; then process a disposable owner-selected media
sample and read back provider IDs, terminal states, search timestamp citations,
response annotations, and the caller's persisted receipt. Durable workflow/UI
work is separate from this tools-only implementation and remains unfinished.

## Sources checked

- Current root/module `AGENTS.md`, server/tools registry and SBV transport sibling.
- Existing tool-runtime facade and atomic MCP execution; Workbench attachment
  streaming and conversation-generation extraction review.
- Docstore search: Oct 8 owner flags require Go-orchestrated atomic operations,
  context ingestion without hashing/custody gates, and AI output remaining context.
- `docstore_related_updates(term="TwelveLabs")` returned no matching indexed
  documents, notes, decision-log records or ADRs; absence is scoped coverage.
- Current official API references: [asset creation](https://docs.twelvelabs.io/api-reference/upload-files/direct-uploads/create),
  [knowledge stores](https://docs.twelvelabs.io/api-reference/knowledge-stores/list),
  [item creation](https://docs.twelvelabs.io/api-reference/knowledge-store-items/create),
  [search](https://docs.twelvelabs.io/api-reference/knowledge-store-search/search),
  [responses](https://docs.twelvelabs.io/api-reference/responses/create).

This is a scoped implementation record, not a new global rule. Full multi-store
historical preflight was not established; no broader architecture or policy was
changed.
