# Frontend/backend split and draft contract

Applies: **yes**. The browser/desktop/reference clients and the persistent legal API/jobs are separate workstreams. They remain inside the existing independent Advocatio repository (`web/`, `api/legal_workspace/`, `tests/`); no replacement repository or fresh application scaffold is created. Sibling Probata and Family Court work require separately owned adapters.

This is a **proposed contract** for the reconciled scope. Existing APIs are inventoried in the current audit; the endpoint names below are not claims that these routes exist. Implementation must reconcile them into versioned OpenAPI schemas in Phase B1 before dependent production client work.

## Shared shapes and access

All identifiers are opaque strings, all times are ISO-8601 with zone, all list responses are bounded `{items, next_cursor, source_revision}`. Unknown/approximate occurrence dates are represented explicitly, not fabricated as midnight. Immutable `VersionRef` is `{id, version, content_hash?}`; missing hashes are explicitly absent, never packet labels impersonating digests.

`SourceRef = {system, package_id?, assertion_id?, assertion_version?, resource_version?, locator, acceptance_status}`. `locator` is tagged `{kind: page|text_span|message|timecode|record, value}`; legal authority and factual evidence use distinct source kinds. External source references must resolve through the owning service.

`ResourceVersion = {id, version, kind, title, jurisdiction?, origin, source_url?, retrieved_at?, effective_from?, effective_to?, content_hash?, citation?, authority_type?, verification_status, supersedes?, access_scope}`.

`Claim = {id, version, text, status: reported|alleged|inferred|accepted_source_reference, source_refs[], issue_ids[]}`. `SupportLink = {id, claim_version, source_ref, relation: supports|contradicts|context, review_status, rationale?, reviewed_by?, reviewed_at?}`. Aggregate completeness is computed from per-claim findings, not section citation count.

`EventView = {id, version, origin_system, origin_ref, event_kind, occurred_at?, occurred_range?, time_precision, recorded_at?, knowledge_as_of?, description, claim_refs[], track_ids[], scheduled_status?, links[], access_scope}`. Legal events authored here are procedural; factual event changes are requests to the evidence owner.

`DocumentVersion = {id, version, parent_version?, content, format, content_hash, state: draft|review|approved|released, method_refs[], source_refs[]}`.

`Proposal = {id, document_version, base_content_hash, method_ref?, run_ref?, changes: [{id, anchor, old_text, new_text, explanation, check_kind}], state}`. `Decision = {proposal_id, change_ids[], action: accept|reject|edit, replacement_text?, expected_document_version, rationale?}`.

`ReviewFinding = {id, subject_ref, dimension: structure|legal_source|factual_support|substantive, criterion, status: not_checked|pass|finding|needs_review|stale, detail, source_refs[], reviewer, method_ref?, reviewed_at}`. A pass applies only to its criterion and subject version.

`PrivateRecord = {id, version, kind: context|strategy|risk|accusation|decision|playbook|analysis_followup, title, content, status, basis_refs[], contrary_refs[], action_refs[], access_scope}`.

`Job = {id, kind, input_refs[], tool_version, parameters, state: queued|running|succeeded|failed|cancelled, output_refs[], diagnostics_ref?, started_at?, completed_at?}`. A failed render/import cannot return a success-shaped placeholder.

**Authentication proposal:** retain existing authorization semantics while porting. Browser uses a same-origin session boundary; service/agent clients use scoped short-lived bearer credentials. Neither browser JavaScript nor a desktop bundle receives provider secrets. Final issuer/audience/cookie and CSRF details must be captured from the existing auth/proxy implementation in B0; they are not settled by this document. Every case/resource lookup enforces matter and access scope on the server. Client `surface=reference` is a presentation hint, never authorization.

**Mutation contract:** version-sensitive requests carry `If-Match`/expected version plus an idempotency key for repeatable jobs/decisions. A stale base returns HTTP 409 with current revision; no automatic overwrite. Errors use `{error: {code, message, request_id, field_errors?, current_version?}}`, with 401 unauthenticated, 403 scope denied, 404 missing/not visible, 409 conflict, 422 invalid and 503 dependency unavailable. Do not leak private content through error details.

## Proposed API surface

| Operation | Request | Response and ownership |
|---|---|---|
| `GET /v1/resources` | query, kind, jurisdiction, case relevance, cursor, limit | Page of ResourceVersion summaries scoped to caller. |
| `GET /v1/resources/{id}/versions/{version}` | exact version | ResourceVersion + content/authorized retrieval link. |
| `POST /v1/resource-imports` | manifest_ref, baseline_ref?, expected_collection_version | Job + item dispositions; preserve first, verification/promotion separate. |
| `POST /v1/resources/{id}/reviews` | expected_version, criterion, finding, cited evidence | New versioned review record; does not rewrite original source. |
| `GET /v1/methods` | task/document kind, jurisdiction, cursor | Versioned method resources; guidance and implemented checks distinguished. |
| `GET /v1/cases/{id}/events` | track, time bounds, knowledge_as_of?, support filter, cursor | Page of EventView projections with stable source refs. |
| `GET /v1/cases/{id}/claims` | issue/status/support filter, cursor | Page of Claim + SupportLink summaries. |
| `POST /v1/claims/{id}/support-reviews` | expected_claim_version, source_ref, relation, finding | Versioned SupportLink/review; accepted-source authority remains upstream. |
| `GET /v1/cases/{id}/support-gaps` | scope/document/track filters, as_of, cursor | Claim-level gaps, source revision and affected document/event refs. |
| `POST /v1/investigation-requests` | claim_refs, missing question, scope, idempotency key | Versioned request/receipt; no implied upstream acceptance. |
| `GET/POST /v1/cases/{id}/private-records` | filters OR PrivateRecord with expected context version | Scoped records/new revision, never automatic factual promotion. |
| `POST /v1/analysis-followups` | upstream analysis version, questions, legal use proposal | PrivateRecord linked to unchanged original analysis. |
| `POST /v1/documents` | template/method versions, entered text/source refs, format | Initial DocumentVersion. |
| `POST /v1/documents/{id}/proposals` | base version/hash, operation: translate|draft|review, method refs, scoped sources | Proposal or Job; original unchanged. |
| `POST /v1/proposals/{id}/decisions` | Decision | Decision receipt + resulting DocumentVersion; rejected changes preserved. |
| `GET/POST /v1/documents/{id}/reviews` | exact version OR ReviewFinding | Independent dimension records; stale dependency flags. |
| `POST /v1/document-jobs` | input DocumentVersion, operation: render|convert|redact|bates|scrub|export, options | Job, derivative manifest, tool/input/output hashes. No filing. |
| `GET /v1/jobs/{id}` | job ID | Job state/output; authorized event stream may later mirror these states. |
| `GET /v1/firm/roles` | active configuration version | Role definitions and capabilities, independent of model credentials. |
| `POST /v1/firm/runs` | role, task, document/source/method versions, scope | Job/run provenance and reviewable output; no approval authority. |
| `POST /v1/releases` | exact approved document versions, review refs, expected hashes | Immutable manifest after applicable gates; transmission is a separate explicit action. |

The simple surface uses scoped read endpoints for resources, events/calendar, records and relevant receipts. It does not receive blanket write or private-strategy scope. Agent tools expose the same bounded operations and source IDs rather than a second authored corpus.

The added MCP connection, discovery, manual invocation and resource/prompt endpoints are specified in [MCP-CLIENT-AND-PORTS.md](MCP-CLIENT-AND-PORTS.md). They are part of the same contract and F2/B2 acceptance slice, including server registration and human use without an agent. Bring-up uses the new service-port registry and named DNS routes.

## Workstreams and synchronization

Frontend owns `web/`: shared shell migration, navigation, resource/claim/timeline panels, help, document proposal review, report views, role/run displays. Build independently against contract fixtures (success, partial support, conflict, denied, stale, dependency failure).

Backend owns `api/legal_workspace/`, `tests/`, explicit storage migrations and renderer adapter. Own transactional revisions, query projection, source validation, jobs, independent checks and access. Verify with isolated fixtures and contract tests before client integration.

1. B0/F0 agree auth and current route inventory before removing Next server behavior.
2. B1/F1 freeze shared resource/source/revision envelopes before reference UI and migration import.
3. B3/F3 agree claim/event/support and Timesketch adapter boundary before graph/timeline integration.
4. B4/F4 agree document anchors, revisions/proposals, review dimensions and editor round-trip rules before rich editing/release UI.

Job event streams, if added, require monotonic sequence per job, reconnect cursor and idempotent client application. No realtime library or cross-service write protocol is implicitly selected here.
