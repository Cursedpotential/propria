"""User-callable tools forwarded only to the authenticated Docstore service."""
from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field

# 0.8.1-r5 (Claude Code · Opus 5.5, 2026-09-27): the memory write payload is typed, so
# docstore_capabilities(operation="docstore_memory_remember") publishes the real field list instead of
# `payload: object`. Owner, 10:01 EDT: "we have no idea how to write to it". Mirrors the live SurrealDB
# `memory` table (probata_memory/memory) and scripts/docstore/remote_memory.validate_remember.
SCOPE_PATTERN = r'^propria(/[a-z0-9_-]+)*$'


class MemoryWrite(BaseModel):
    """One durable claim for the shared agent memory (SurrealDB probata_memory/memory)."""
    model_config = ConfigDict(extra='forbid')

    kind: Literal['correction', 'preference', 'observation', 'handoff', 'fact', 'constraint', 'decision'] = Field(
        description='What sort of claim: an owner rule is usually constraint, preference or correction.')
    claim: str = Field(min_length=11, max_length=599, description=(
        'One self-contained sentence an agent can act on without the conversation. Unique per scope: the '
        'exact text can never be written twice, even after it is superseded or retracted.'))
    evidence: str = Field(min_length=1, max_length=2000, description=(
        'Where the claim comes from: owner quote with date/time, doc id, file path or session anchor.'))
    agent: str = Field(min_length=1, max_length=200, description='Who writes it, e.g. "Claude Code · Opus 5.5".')
    scope: str = Field('propria', pattern=SCOPE_PATTERN, description=(
        'Hierarchical scope. "propria" is the whole project; "propria/<module>[/<agent>]" narrows it. '
        'Recall of a scope includes its descendants.'))
    detail: str | None = Field(None, max_length=8000, description='Optional longer explanation: why, how to apply.')
    confidence: float | None = Field(None, ge=0, le=1, description='0-1; the store defaults to 0.6. Owner rules: 0.95-1.')
    observed_at: str | None = Field(None, description='ISO-8601 time the claim was observed; defaults to now.')
    force: bool = Field(False, description=(
        'Write even though near-duplicates exist, keeping both. Use only when the claims really differ.'))
    supersede: str | None = Field(None, pattern=r'^memory:[A-Za-z0-9_]+$', description=(
        'Replace this ACTIVE memory id: the new row is written, linked ->supersedes-> the old one, and the old '
        'row becomes status superseded (never deleted). The claim text must differ from the old claim.'))
    reason: str | None = Field(None, max_length=1000, description='Why a supersession happened; stored in decision_log.')


def register(mcp,request,read):
    write={**read,'readOnlyHint':False,'idempotentHint':False}

    @mcp.tool(annotations=read)
    async def docstore_retraction_plan() -> dict:
        """Preview documents outside the current five-root source snapshot; no writes."""
        return await request('GET','/sources/retraction-plan')

    @mcp.tool(annotations=read)
    async def docstore_diagnostics() -> dict:
        """Report release, processing mode and endpoint identity. Does not start indexing."""
        return await request('GET','/release')

    @mcp.tool(annotations=read)
    async def docstore_upgrade_status() -> dict:
        """Read current version, migration checksums and missing schema."""
        return await request('POST','/upgrade/status',payload={})

    @mcp.tool(annotations=read)
    async def docstore_upgrade_plan() -> dict:
        """Dry-run additive migrations and return an exact plan ID."""
        return await request('POST','/upgrade/plan',payload={})

    @mcp.tool(annotations=write)
    async def docstore_upgrade_apply(plan_id:str) -> dict:
        """Apply the checksum-bound plan transactionally; no index run or legacy deletion."""
        return await request('POST','/upgrade/apply',payload={'plan_id':plan_id})

    @mcp.tool(annotations=read)
    async def docstore_upgrade_verify() -> dict:
        """Verify release ledger and live required schema after an upgrade."""
        return await request('POST','/upgrade/verify',payload={})

    @mcp.tool(annotations=write)
    async def docstore_adr(action:Literal['list','create','update','migration-plan','migration-apply','projections','verify'], payload:dict|None=None) -> dict:
        """Manage authoritative ADRs, version-checked edits, legacy imports and generated projections."""
        return await request('POST','/adr/'+action,payload=payload or {})

    @mcp.tool(annotations=read)
    async def docstore_knowledge_graph(start:str,relation:str='about',depth:int=1,limit:int=50) -> dict:
        """Read bounded native SurrealDB relationships across documents, entities and statements."""
        return await request('POST','/knowledge/traverse',payload=locals_payload(start,relation,depth,limit))

    @mcp.tool(annotations=read)
    async def docstore_graph_path(start:str,end:str,relation:str='related_to',depth:int=4,limit:int=100) -> dict:
        """Find a bounded directed graph path; not-found applies only to the requested bounds."""
        return await request('POST','/knowledge/path',payload={**locals_payload(start,relation,depth,limit),'end':end})

    @mcp.tool(annotations=write)
    async def docstore_graph_entity_upsert(name:str,kind:str,aliases:list[str]|None=None) -> dict:
        """Resolve/create one canonical entity using normalized names and explicit aliases."""
        return await request('POST','/knowledge/entity-upsert',payload={'name':name,'kind':kind,'aliases':aliases or []})

    @mcp.tool(annotations=write)
    async def docstore_graph_relate(start:str,end:str,relation:str='related_to') -> dict:
        """Create one native SurrealDB relationship; allowed types and record IDs are validated."""
        return await request('POST','/knowledge/relate',payload={'start':start,'end':end,'relation':relation})

    @mcp.tool(annotations=read)
    async def docstore_context_pack(rows:list[dict],limit:int=20,budget:int=8000) -> dict:
        """Normalize federated candidates through DuckDB with retained provenance and a byte budget."""
        return await request('POST','/context/pack',payload={'rows':rows,'limit':limit,'budget':budget})

    @mcp.tool(annotations=read)
    async def docstore_source_plan(files:list[dict]) -> dict:
        """Dry-run a complete hash-validated five-root source sync; no embedding or writes."""
        return await request('POST','/sources/plan',payload={'files':files})

    @mcp.tool(annotations=write)
    async def docstore_source_apply(files:list[dict],plan_id:str,retract:list[str]|None=None) -> dict:
        """Apply an exact source plan, quarantining replaced files; no index run. 0.8.1-r3: every document the plan would retract must be named in retract, or the apply is refused."""
        return await request('POST','/sources/apply',payload={'files':files,'plan_id':plan_id,'retract':retract or []})

    @mcp.tool(annotations=read)
    async def docstore_source_read(paths:list[str]) -> dict:
        """0.8.1-r3: exact mirror copies (content + sha256) of named project/path keys, for hash-verified restores; bounded, unsent keys return in remaining."""
        return await request('POST','/sources/read',payload={'paths':paths})

    @mcp.tool(annotations=read)
    async def docstore_surrealql_read(query:str) -> dict:
        """Bounded native SELECT fields/traversals FROM table_or_record LIMIT n. Expressions and writes are rejected."""
        return await request('POST','/surrealql/read',payload={'query':query})

    @mcp.tool(annotations=read)
    async def docstore_memory_recall(query:Annotated[str,Field(min_length=1,max_length=2000)],
                                     scope:Annotated[str,Field(pattern=SCOPE_PATTERN)]='propria',
                                     limit:Annotated[int,Field(ge=1,le=50)]=10) -> dict:
        """Recall independent remote shared memory (active rows in scope and its descendants), with
        server-side query embedding, BM25 + vector fusion and DuckDB packing. Scope root is "propria"."""
        return await request('POST','/memory/recall',payload={'query':query,'scope':scope,'limit':limit})

    @mcp.tool(annotations=write)
    async def docstore_memory_remember(payload:MemoryWrite) -> dict:
        """Write one claim through the memory service's governed fn::remember. Call with
        docstore_query(operation="docstore_memory_remember", mode="write", arguments={"payload": {...}}).
        Required: kind, claim, evidence, agent. Optional: scope (default "propria"), detail, confidence,
        observed_at, force, supersede, reason.
        Duplicate guard (0.8.1-r6, 2026-09-28): an active row in the same scope conflicts if BM25 finds every
        claim word in it, or its cosine distance is <= 0.10, or its distance is <= 0.20 AND word overlap
        (Jaccard) is >= 0.35. Then nothing is written and the call fails HTTP 409 listing the conflicting ids
        with dist and overlap; retry with supersede:"<id>" and a reworded claim to replace one, or force:true.
        Success returns {outcome: "written"|"superseded", id, superseded, scope}.
        Errors: 422 invalid payload (every problem listed), 409 near-duplicate or exact claim already stored,
        503/502 memory service unreachable or failed."""
        return await request('POST','/memory/remember',payload=payload.model_dump(exclude_none=True))


def locals_payload(start,relation,depth,limit):
    return {'start':start,'relation':relation,'depth':depth,'limit':limit}
