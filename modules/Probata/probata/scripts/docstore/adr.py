"""Authoritative ADR rows with deterministic legacy imports and Markdown projections."""
from __future__ import annotations
import hashlib
import json
import os
import re
from pathlib import Path
import sq
from upgrade import rows, digest

STATUSES = {'proposed','accepted','superseded','deprecated','rejected'}
MAX_SELECTION = 50


def selection(payload):
    """Validate exact ADR selectors and an optional bounded result limit.

    Inputs: a payload with id/ids and/or number/numbers, plus optional limit.
    Outputs: (deduplicated actual record IDs or None, explicit limit or None).
    Effects: none; invalid, empty or oversized selections fail before database I/O.
    Pick this over a table filter when callers know the ADR identity; numeric
    selectors address canonical propria IDs, while explicit IDs also allow legacy
    imports. Mixed selectors form a union and an exact selection is never truncated.
    """
    limit = payload.get('limit')
    if 'limit' in payload and (type(limit) is not int or not 1 <= limit <= MAX_SELECTION):
        raise ValueError('ADR limit must be an integer from 1 to 50')
    selected = []
    present = False
    for key in ('id', 'ids', 'number', 'numbers'):
        if key not in payload:
            continue
        present = True
        values = payload[key] if key in {'ids', 'numbers'} else [payload[key]]
        if not isinstance(values, list) or not values:
            raise ValueError(f'{key} must contain at least one ADR selector')
        if len(selected) + len(values) > MAX_SELECTION:
            raise ValueError('ADR selection must contain at most 50 selectors')
        for value in values:
            if key in {'number', 'numbers'}:
                if type(value) is not int or not 1 <= value <= 999999:
                    raise ValueError('ADR number must be an integer from 1 to 999999')
                rid = f'adr:propria_{value:04d}'
            else:
                if not isinstance(value, str):
                    raise ValueError('Invalid ADR id')
                match = re.fullmatch(r'adr:propria_([0-9]{4,6})', value)
                if match:
                    number = int(match[1])
                    if not 1 <= number <= 999999 or value != f'adr:propria_{number:04d}':
                        raise ValueError('Invalid canonical propria ADR id')
                elif not re.fullmatch(r'adr:legacy_[0-9a-f]{32}', value):
                    raise ValueError('Invalid ADR id')
                rid = value
            selected.append(rid)
    if not present:
        return None, limit
    selected = list(dict.fromkeys(selected))
    if limit is not None and len(selected) > limit:
        raise ValueError('ADR limit cannot truncate an exact selection')
    return selected, limit


async def selected_records(db, ids):
    """Read only selected ADR record IDs and verify their propria identity.

    Inputs: the Docstore connection and validated, bounded actual ADR IDs.
    Outputs: records in selector order and IDs not found in the database.
    Effects: parameterized record reads only, never a table scan or mutation.
    Pick this over legacy browsing when exact selectors were supplied; it uses
    the same type::record($id) lookup as create/update's record readback.
    """
    records, missing = [], []
    for rid in ids:
        found = rows(await db.query('SELECT * FROM type::record($id);', {'id': rid}))
        if not found:
            missing.append(rid)
            continue
        if len(found) != 1:
            raise ValueError('ADR identity lookup returned multiple records')
        record = found[0]
        if str(record.get('id')) != rid or record.get('project') != 'propria':
            raise ValueError('Selected ADR identity or project mismatch')
        number = record.get('number')
        if type(number) is not int or not 1 <= number <= 999999:
            raise ValueError('Selected ADR has an invalid number')
        if rid.startswith('adr:propria_') and rid != f'adr:propria_{number:04d}':
            raise ValueError('Selected ADR number does not match its canonical id')
        records.append(record)
    return records, missing


def render(record):
    return (f"<!-- Generated from {record['id']}; edit through docstore_adr, version {record.get('version',1)}. -->\n"
            f"# ADR-{record['number']:04d}: {record['title']}\n\nStatus: {record['status']}\n\n"
            f"## Context\n\n{record.get('context','')}\n\n## Decision\n\n{record['decision']}\n\n"
            f"## Consequences\n\n{record.get('consequences','')}\n")


def parse_legacy(doc):
    sections = re.split(r'(?mi)^##\s+(Context|Decision|Consequences)\s*$', doc.get('body',''))
    fields = {sections[i].lower():sections[i+1].strip() for i in range(1,len(sections)-1,2)}
    # Active generic documents are not proof of accepted owner decisions.
    status = {'superseded':'superseded','retracted':'deprecated'}.get(doc.get('status'),'proposed')
    return {'title':doc.get('title') or str(doc['id']), 'context':fields.get('context',''),
            'decision':fields.get('decision',doc.get('body','')), 'consequences':fields.get('consequences',''),
            'status':status, 'legacy_status':doc.get('status','unverified')}


async def migration(db):
    docs = rows(await db.query('SELECT * FROM document WHERE doc_type="decision" ORDER BY id LIMIT 5001;'))
    if len(docs)>5000:
        raise ValueError('Legacy migration exceeds bounded plan')
    existing = rows(await db.query('SELECT id, number, migration_key, version FROM adr WHERE project="propria";'))
    keys = {r.get('migration_key') for r in existing}
    number = max([r['number'] for r in existing] or [0])
    imports=[]
    for doc in docs:
        key = 'legacy_'+hashlib.sha256(str(doc['id']).encode()).hexdigest()[:32]
        if key in keys or '/adr/generated/' in doc.get('source_path',''):
            continue
        number += 1
        imports.append({'id':'adr:'+key,'number':number,'project':'propria','source_doc':str(doc['id']),
                        'migration_key':key,'version':1, **parse_legacy(doc)})
    return {'imports':imports,'plan_id':digest({'imports':imports,'existing':existing}), 'preserves_legacy_documents':True}


async def refresh_projections(materialize=False, payload=None):
    """Render selected ADR Markdown projections or the legacy bounded corpus.

    Inputs: optional id/ids/number/numbers and limit; materialize defaults false.
    Outputs: projections, direction and missing_ids for exact selections.
    Effects: reads only by default; explicit materialize updates projection
    metadata and may write registered Markdown files. With no selectors/limit,
    legacy behavior reads up to 5001 rows and rejects corpora above 5000.
    Pick this over list when rendered Markdown and index verification are needed.
    """
    ids, limit = selection(payload or {})
    db = await sq.connect('docs','probata','docs')
    try:
        if ids is not None:
            records, missing = await selected_records(db, ids)
        else:
            records = rows(await db.query(
                'SELECT * FROM adr WHERE project="propria" ORDER BY number LIMIT $limit;',
                {'limit': limit if limit is not None else 5001}))
            if limit is None and len(records)>5000:
                raise ValueError('Projection limit exceeded')
        projections=[]
        for record in records:
            text=render(record); sha=hashlib.sha256(text.encode()).hexdigest()
            path=f"docs/adr/generated/{record['number']:04d}.md"
            if materialize and (record.get('projection_path')!=path or record.get('projection_hash')!=sha):
                await db.query('UPDATE type::record($id) SET projection_path=$path, projection_hash=$hash;', {'id':str(record['id']),'path':path,'hash':sha})
            matches=rows(await db.query('SELECT id FROM document WHERE source_path=$path AND content_hash=$hash LIMIT 1;',{'path':path,'hash':sha}))
            if materialize and matches and str(record.get('projection_doc'))!=str(matches[0]['id']):
                await db.query('UPDATE type::record($id) SET projection_doc=$doc;',{'id':str(record['id']),'doc':matches[0]['id']})
            projections.append({'adr_id':str(record['id']),'path':path,'sha256':sha,'content':text,'indexed':bool(matches)})
        if materialize and os.environ.get('DOCSTORE_PROJECT_REGISTRY'):
            from source_sync import state
            from scope import ROOTS
            root,_=state()
            for projection in projections:
                # Probata's docs root is docs/probata via the Propria/docs junction, not the
                # pre-2026-09-20 module path this used to join. Strip the canonical prefix
                # from the projection path and rebuild under the root scope.py declares.
                source_root,prefix=ROOTS['probata']
                relative=projection['path'][len(prefix):] if projection['path'].startswith(prefix) else projection['path']
                base=(root/source_root).resolve()
                target=base/relative
                if not target.resolve().is_relative_to(base):
                    raise ValueError('ADR projection path escapes docs')
                data=projection['content'].encode()
                target.parent.mkdir(parents=True,exist_ok=True)
                if target.exists() and target.read_bytes()!=data:
                    import uuid
                    backup=root/'to_be_deleted'/('adr-'+uuid.uuid4().hex)/target.name
                    backup.parent.mkdir(parents=True,exist_ok=True); target.replace(backup)
                if not target.exists():
                    pending=target.with_suffix('.pending')
                    pending.write_bytes(data); pending.replace(target)
        result = {'projections':projections,'direction':'adr -> markdown -> CocoIndex document'}
        if ids is not None:
            result['missing_ids'] = missing
        return result
    finally:
        await db.close()


async def operation(action, payload=None):
    """Execute a governed ADR read, revision, migration or projection operation.

    Inputs: action and payload; list/projections/verify accept exact selectors
    id/ids/number/numbers (1..50) and optional integer limit (1..50).
    Outputs: records or projections, with missing_ids for exact selections.
    Effects: list/projections/verify never materialize; create/update/migration-apply
    retain their governed writes and materialization. Unselected list preserves
    the legacy first-200 browse, while unselected projections/verify retain their
    legacy 5000-row safety bound unless a smaller explicit limit is supplied.
    Pick list for authoritative rows, projections for Markdown, verify for index
    status, and mutation actions only when changing authoritative ADRs.
    """
    payload=payload or {}
    if action in {'projections','verify'}:
        result = await refresh_projections(payload=payload)
        result['projection_sync_required'] = any(not p['indexed'] for p in result['projections'])
        return result
    ids, limit = selection(payload) if action == 'list' else (None, None)
    db=await sq.connect('docs','probata','docs')
    try:
        if action in {'migration-plan','migration-apply'}:
            plan=await migration(db)
            if action=='migration-plan':
                return plan
            if payload.get('plan_id')!=plan['plan_id']:
                raise ValueError('Legacy migration plan changed')
            # One transaction avoids partially numbered migrations and allows unique-index conflicts to abort.
            sql=['BEGIN TRANSACTION;']; params={}
            for i, record in enumerate(plan['imports']):
                params[f'r{i}']={k:v for k,v in record.items() if k not in {'id','source_doc'}}
                params[f'id{i}']=record['id']; params[f'src{i}']=record['source_doc']
                sql.append(f'CREATE type::record($id{i}) CONTENT $r{i}; UPDATE type::record($id{i}) SET source_doc=type::record($src{i}); RELATE (type::record($id{i}))->derived_from->(type::record($src{i}));')
            sql.append('COMMIT TRANSACTION;')
            if plan['imports']:
                await db.query('\n'.join(sql),params)
            result={'imported':len(plan['imports']), 'legacy_documents_deleted':0}
        elif action=='list':
            if ids is not None:
                records, missing = await selected_records(db, ids)
                return {'results': sq.norm(records, True), 'missing_ids': missing}
            return {'results':sq.norm(rows(await db.query(
                'SELECT * FROM adr WHERE project="propria" ORDER BY number LIMIT $limit;',
                {'limit': limit if limit is not None else 200})),True)}
        elif action in {'create','update'}:
            fields={k:payload[k] for k in ('title','context','decision','consequences','status') if k in payload}
            if fields.get('status','proposed') not in STATUSES or sum(len(str(v)) for v in fields.values())>100000:
                raise ValueError('Invalid ADR fields')
            if action=='create':
                if not fields.get('title') or not fields.get('decision'):
                    raise ValueError('Title and decision required')
                number=payload.get('number')
                if not isinstance(number,int) or not 1<=number<=999999:
                    raise ValueError('Explicit positive ADR number required')
                rid=f'adr:propria_{number:04d}'
                fields.update(number=number,project='propria',version=1,context=fields.get('context',''),status=fields.get('status','proposed'))
                await db.query('CREATE type::record($id) CONTENT $fields;',{'id':rid,'fields':fields})
            else:
                rid=payload.get('id','')
                if not re.fullmatch(r'adr:[a-zA-Z0-9_]+',rid):
                    raise ValueError('Invalid ADR id')
                expected=payload.get('expected_version')
                if not isinstance(expected,int) or expected<1:
                    raise ValueError('expected_version must be a positive integer')
                fields['version']=expected+1
                await db.query('''BEGIN TRANSACTION;
                    LET $current = SELECT * FROM ONLY type::record($id);
                    IF $current.version != $expected { THROW "revision_conflict"; };
                    UPDATE type::record($id) MERGE $fields;
                    CREATE decision_log SET subject=type::record($id), action="adr_updated", actor=$actor, rationale=$reason;
                    COMMIT TRANSACTION;''',{'id':rid,'expected':payload.get('expected_version'), 'fields':fields,
                                         'actor':payload.get('actor','docstore'),'reason':payload.get('rationale','ADR update')})
            result={'record':sq.norm(rows(await db.query('SELECT * FROM type::record($id);',{'id':rid})),True)}
        else:
            raise ValueError('Unknown ADR action')
    finally:
        await db.close()
    result.update(await refresh_projections(materialize=action in {'create','update','migration-apply'}))
    result['projection_sync_required']=any(not p['indexed'] for p in result['projections'])
    return result
