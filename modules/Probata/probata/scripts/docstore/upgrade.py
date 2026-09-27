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
MIGRATIONS = ('100_release_080.surql',)
REQUIRED = {'document', 'chunk', 'adr', 'decision_log', 'entity', 'docstore_meta', 'docstore_migration',
            'docstore_enrichment', 'statement', 'entity_alias', 'about', 'asserts', 'statement_mentions', 'related_to', 'supports', 'contradicts'}


def rows(value):
    while isinstance(value, list) and len(value) == 1 and isinstance(value[0], list):
        value = value[0]
    return value if isinstance(value, list) else [value] if value else []


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, default=str).encode()).hexdigest()


async def plan(db):
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


async def operation(action='status', plan_id=None):
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
        state['verified'] = not state['pending'] and not state['missing_tables'] and state['current_version'] == VERSION
        return state
    finally:
        await db.close()


async def verify_required():
    state = await operation('verify')
    if not state['verified']:
        raise RuntimeError('Schema upgrade/verification required before indexing')
    return state


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['status','plan','dry-run','apply','verify'])
    parser.add_argument('--plan-id')
    args = parser.parse_args()
    print(json.dumps(asyncio.run(operation(args.action,args.plan_id)),default=str))
