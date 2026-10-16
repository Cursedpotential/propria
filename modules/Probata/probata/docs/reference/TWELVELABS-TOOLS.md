<!-- Generated from server/tools/extractors/twelvelabs.py; do not edit manually. -->
# TwelveLabs atomic tools

Available through the existing tool-runtime registry, REST execution, and atomic_tools MCP directory.

## `twelvelabs.asset-submit`

Capability: `media.asset.submit`

Submit a selected media URL to TwelveLabs without waiting; check result.ok inside MCP wrappers.

Inputs: url (direct HTTPS media or signed object URL), optional source_ref.
Output: context envelope with provider asset _id and status. Side effects:
external upload/storage and provider charges. Pick this for a new processing
copy; persist its _id before scheduling asset-status. Never blindly retry an
ambiguous submission. This operation does not hash or alter the original.

## `twelvelabs.asset-status`

Capability: `media.asset.status`

Retrieve a TwelveLabs asset's status and metadata; check result.ok inside MCP wrappers.

Inputs: asset_id, optional source_ref. Output: context envelope with provider
JSON. Side effects: one external read. Use after asset-submit; the caller owns
polling and handles processing, ready, failed and unknown statuses explicitly.

## `twelvelabs.stores-list`

Capability: `media.store.list`

List TwelveLabs knowledge stores for reuse; check result.ok inside MCP wrappers.

Inputs: page (default 1), page_limit (1-50, default 10). Output: provider
stores and page_info. Side effects: one external read. Prefer this over
creating another store on each run; the caller chooses corpus grouping.

## `twelvelabs.store-create`

Capability: `media.store.create`

Create one explicitly named TwelveLabs knowledge store; check result.ok inside MCP wrappers.

Inputs: name (1-255 characters). Output: context envelope with store _id.
Side effects: creates remote storage. Prefer stores-list when reusing an
existing corpus. Persist the returned ID; creation is not retry-idempotent.

## `twelvelabs.item-add`

Capability: `media.item.add`

Add a ready asset to a chosen TwelveLabs store for indexing; check result.ok inside MCP wrappers.

Inputs: knowledge_store_id, asset_id, asset_type (video or image, default
video), optional source_ref. Output: item _id and indexing status in a context
envelope. Side effects: remote indexing and charges. Call only after observing
asset ready; checkpoint the result before item-status. No hidden polling.

## `twelvelabs.items-list`

Capability: `media.item.list`

List a page of TwelveLabs knowledge-store items; check result.ok inside MCP wrappers.

Inputs: knowledge_store_id, page, page_limit. Output: provider item identities
and pagination in a context envelope. Side effects: one external read. Use to
inspect a corpus or reconcile an uncertain item-add before resubmitting it.

## `twelvelabs.item-status`

Capability: `media.item.status`

Retrieve a TwelveLabs item's indexing status; check result.ok inside MCP wrappers.

Inputs: knowledge_store_id, item_id. Output: provider JSON including status
in a context envelope. Side effects: one external read. The caller owns wait
timing/deadlines and handles queued, pending, processing, ready and failed.

## `twelvelabs.search`

Capability: `media.search`

Search TwelveLabs media with timestamped matches; check result.ok inside MCP wrappers.

Inputs: knowledge_store_id, query, optional page_token, item_ids, page_size
(1-50). Output: ranked matches and next_page_token in a context envelope.
Side effects: one provider search and possible charges. Use for finding
moments; use respond for synthesis. Filters narrow search, not authorization.

## `twelvelabs.respond`

Capability: `media.analyze`

Ask TwelveLabs Jockey about a corpus with native citations; check result.ok inside MCP wrappers.

Inputs: knowledge_store_id, prompt, optional session_id and json_schema.
Output: complete provider response (status, usage, messages and annotations)
inside a context envelope. Side effects: provider generation/session state
and charges. No public-web tool is enabled. The entire knowledge store is
accessible to Jockey: choose a properly isolated store, not a selection as
an access boundary. Check completed versus incomplete/failed before review.
Schema output is requested, not treated as validated or verified evidence.
