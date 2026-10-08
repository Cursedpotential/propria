"""Versioned, checksum-bound migration plans. Application never runs implicitly."""
from __future__ import annotations
import argparse
import asyncio
import hashlib
import json
from pathlib import Path
import sq

VERSION = '0.8.1'
SCHEMA = Path(__file__).parent / 'schema'
MIGRATIONS = ('100_release_080.surql', '101_contextual_chunks.surql')
REQUIRED = {'document', 'chunk', 'adr', 'decision_log', 'entity', 'docstore_meta', 'docstore_migration',
            'docstore_enrichment', 'statement', 'entity_alias', 'about', 'asserts', 'statement_mentions', 'related_to', 'supports', 'contradicts'}


def rows(value):
    while isinstance(value, list) and len(value) == 1 and isinstance(value[0], list):
        value = value[0]
    return value if isinstance(value, list) else [value] if value else []


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, default=str).encode()).hexdigest()


async def plan(db):
    """Plan checksum-bound additive upgrades for an existing Docstore database.

    Inputs: connected Docstore client. Output: reviewable migration plan and ID.
    Side effects: database/file reads only. Pick this before explicit apply; adding
    contextual fields does not index sources or enable contextual generation.
    """
    info = rows(await db.query('INFO FOR DB;'))[0]
    tables = set(info.get('tables', {}))
    if not {'document', 'chunk', 'adr', 'decision_log', 'entity'} <= tables:
        raise ValueError('Existing baseline schema required; use explicit bootstrap on an empty dedicated database')
    ledger = rows(await db.query('SELECT * FROM docstore_migration;')) if 'docstore_migration' in tables else []
    meta = rows(await db.query('SELECT * FROM docstore_meta:release;')) if 'docstore_meta' in tables else []
    current = meta[0].get('version') if meta else None
    if current and tuple(map(int, current.split('.'))) > tuple(map(int, VERSION.split('.'))):
        raise ValueError('Downgrade refused')
    applied = {r['name']: r['checksum'] for r in ledger if r.get('state') == 'applied'}
    pending = []
    for name in MIGRATIONS:
        checksum = hashlib.sha256((SCHEMA / name).read_bytes()).hexdigest()
        if name in applied and applied[name] != checksum:
            raise ValueError('Applied migration checksum drift; explicit repair migration required')
        if name not in applied:
            pending.append({'name': name, 'checksum': checksum})
    state = {'version': VERSION, 'current_version': current, 'pending': pending, 'missing_tables': sorted(REQUIRED-tables),
             'ledger': applied, 'destructive': False, 'automatic_indexing': False}
    state['plan_id'] = digest(state)
    return state


async def contextual_index_readiness(db):
    """Read contextual keyword index readiness without waiting for its build.

    Input: connected Docstore client. Output: ready flag, status and index details.
    Side effects: one INFO query only. Use after migration101 is ledger-applied;
    missing, failed or unknown index states must never permit indexing admission.
    """
    result = rows(await db.query('INFO FOR INDEX chunk_context_ft ON chunk;'))
    info = result[0] if result and isinstance(result[0], dict) else {}
    building = info.get('building', {})
    status = building.get('status') if isinstance(building, dict) else None
    return {'ready': status == 'ready', 'status': status or 'unknown', 'details': info}


async def operation(action='status', plan_id=None):
    """Plan, apply or verify a checksum-bound upgrade with index readiness admission.

    Inputs: action and reviewed plan ID for apply. Output: plan and verification
    state. Side effects: apply alone writes the governed migration transaction;
    verification performs bounded reads and never waits for concurrent builds.
    Pick verify before admitting indexing or enabling contextual keyword search.
    """
    db = await sq.connect('docs', 'probata', 'docs')
    try:
        state = await plan(db)
        if action in {'status', 'plan', 'dry-run'}:
            return state
        if action == 'apply':
            if plan_id != state['plan_id']:
                raise ValueError('Migration plan changed; obtain a fresh plan')
            sql = ['BEGIN TRANSACTION;']
            params = {'version': VERSION}
            for i, item in enumerate(state['pending']):
                sql.append((SCHEMA / item['name']).read_text())
                sql.append(f'CREATE type::record("docstore_migration", $key{i}) SET name=$name{i}, checksum=$hash{i}, state="applied", at=time::now();')
                params.update({f'key{i}':item['name'].replace('.','_'),f'name{i}':item['name'],f'hash{i}':item['checksum']})
            sql.extend(['UPSERT docstore_meta:release SET version=$version, updated_at=time::now();', 'COMMIT TRANSACTION;'])
            if state['pending'] or state['current_version'] != VERSION:
                await db.query('\n'.join(sql), params)
            state = await plan(db)
        if action not in {'verify', 'apply'}:
            raise ValueError('Unknown upgrade operation')
        state['contextual_index'] = (await contextual_index_readiness(db) if not state['pending']
                                     else {'ready': False, 'status': 'migration-pending'})
        state['verified'] = (not state['pending'] and not state['missing_tables']
                             and state['current_version'] == VERSION and state['contextual_index']['ready'])
        return state
    finally:
        await db.close()


async def verify_required():
    """Require applied migrations and a ready contextual index before indexing.

    Input: current configured Docstore connection. Output: verified schema state.
    Side effects: database reads only; raises while schema or index is unready.
    Use at the existing indexing admission boundary, never to start a migration.
    """
    state = await operation('verify')
    if not state['verified']:
        raise RuntimeError('Schema upgrade and ready contextual index required before indexing')
    return state


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['status','plan','dry-run','apply','verify'])
    parser.add_argument('--plan-id')
    args = parser.parse_args()
    print(json.dumps(asyncio.run(operation(args.action,args.plan_id)),default=str))
