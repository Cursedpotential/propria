"""Docstore 0.8.1-r3: retraction guard (server side).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Owner order 2026-09-26, after 24 Probata docs were found missing from the desktop and a routine sync
would have retracted them: "make lost docs impossible to miss". A sync or an index run must never
retract a document on its own; missing documents are restored from the mirror (hash-verified) by
default, and only paths an operator names explicitly are retracted.

Runs on ovh-files against the release build context (default /data/propria/releases/docstore-0.8.1),
like apply_r2.py: anchored in-place edits, every other byte preserved, each file compiled before it is
written, backups <name>.bak-<utc>-r3, idempotent (marker 0.8.1-r3), dry run unless --apply.

    python3 apply_r3.py [release_dir] [--apply]

1. scripts/docstore/source_sync.py - new `read` action (exact mirror copies + sha256 for named
   project/path keys, bounded per reply); the plan lists `retracted_hashes`; `apply` refuses any
   retraction not named in `retract`.
2. plugins/docstore/control/release_tools.py - docstore_source_apply(..., retract) and a read-only
   docstore_source_read(paths) operation.
3. scripts/docstore/cdc_verify.py - retire_unexpected_projection(..., allowed) retracts only the
   named paths; every other absent document is held (stays active) and reported as held_paths.
4. scripts/docstore/worker_sync.py - the run passes DOCSTORE_RETRACT_PATHS as `allowed`; held paths
   are recorded and named in the run's reason, so the run cannot pass as clean.
5. scripts/docstore/api.py - POST /runs accepts retract_paths (up to 1000) and hands them to the run.
6. plugins/docstore/control/server.py - docstore_index_full(..., retract_paths).
7. tests/test_release.py - the two retraction tests follow the new contract.
"""
from __future__ import annotations

import sys
import time
from pathlib import Path

MARKER = "0.8.1-r3"

SOURCE_OPERATION_OLD = """def operation(action,payload):
    files=validated(payload.get('files'))
"""
SOURCE_OPERATION_NEW = """READ_BUDGET=1024*1024  # 0.8.1-r3: keeps each read reply well under the ctl client's 2 MiB response cap


def read(payload):
    \"\"\"0.8.1-r3 (Claude Code · Opus 5.5, 2026-09-26): exact mirror copies of named project/path keys, so a
    client can restore a document it lacks instead of retracting it. Bounded; unsent keys come back in
    `remaining`, keys the mirror does not hold in `missing`.\"\"\"
    keys=payload.get('paths')
    if not isinstance(keys,list) or not 1<=len(keys)<=50 or not all(isinstance(k,str) for k in keys):
        raise ValueError('paths must list 1-50 project/path keys')
    lock=Path(os.environ.get('DOCSTORE_SYNC_LOCK','/data/state/sync.lock'))
    with worker_lock(lock):
        root,existing=state()
        files=[]; missing=[]; remaining=[]; used=0
        for key in keys:
            if key not in existing:
                missing.append(key); continue
            project,path=key.split('/',1)
            data=(root/ROOTS[project][0]/path).read_bytes()
            if files and used+len(data)>READ_BUDGET:
                remaining.append(key); continue
            used+=len(data)
            files.append({'key':key,'project':project,'path':path,'content':data.decode('utf-8'),
                          'sha256':hashlib.sha256(data).hexdigest()})
        return {'files':files,'missing':missing,'remaining':remaining}


def operation(action,payload):
    if action=='read':
        return read(payload)
    files=validated(payload.get('files'))
"""

SOURCE_PLAN_OLD = """        plan={'changed':sorted(k for k,v in incoming.items() if existing.get(k)!=v),
              'retracted_sources':sorted(set(existing)-set(incoming)), 'count':len(files),
              'current_digest':digest(existing),'incoming_digest':digest(incoming)}
"""
SOURCE_PLAN_NEW = """        retracted=sorted(set(existing)-set(incoming))
        plan={'changed':sorted(k for k,v in incoming.items() if existing.get(k)!=v),
              'retracted_sources':retracted, 'count':len(files),
              'current_digest':digest(existing),'incoming_digest':digest(incoming),
              # 0.8.1-r3: the mirror's hash of every document this plan would retract, so a client can
              # restore it byte-for-byte instead (retraction guard, owner order 2026-09-26).
              'retracted_hashes':{k:existing[k] for k in retracted}}
"""

SOURCE_GUARD_OLD = """        if action!='apply' or payload.get('plan_id')!=plan['plan_id']:
            raise ValueError('Upload plan changed; re-plan before apply')
"""
SOURCE_GUARD_NEW = """        if action!='apply' or payload.get('plan_id')!=plan['plan_id']:
            raise ValueError('Upload plan changed; re-plan before apply')
        # 0.8.1-r3 retraction guard (owner order 2026-09-26): an apply never retracts a document the caller
        # did not name. A client restores missing documents from the mirror instead.
        named=payload.get('retract') or []
        if not isinstance(named,list) or not all(isinstance(k,str) for k in named):
            raise ValueError('retract must list project/path keys')
        unconfirmed=[k for k in plan['retracted_sources'] if k not in set(named)]
        if unconfirmed:
            raise ValueError(f'Retraction guard: {len(unconfirmed)} document(s) would be retracted without being named: '
                             +', '.join(unconfirmed[:10])+('' if len(unconfirmed)<=10 else ', ...'))
"""

TOOLS_APPLY_OLD = """    async def docstore_source_apply(files:list[dict],plan_id:str) -> dict:
        \"\"\"Apply an exact source plan, quarantining replaced/removed files; no index run.\"\"\"
        return await request('POST','/sources/apply',payload={'files':files,'plan_id':plan_id})
"""
TOOLS_APPLY_NEW = """    async def docstore_source_apply(files:list[dict],plan_id:str,retract:list[str]|None=None) -> dict:
        \"\"\"Apply an exact source plan, quarantining replaced files; no index run. 0.8.1-r3: every document the plan would retract must be named in retract, or the apply is refused.\"\"\"
        return await request('POST','/sources/apply',payload={'files':files,'plan_id':plan_id,'retract':retract or []})

    @mcp.tool(annotations=read)
    async def docstore_source_read(paths:list[str]) -> dict:
        \"\"\"0.8.1-r3: exact mirror copies (content + sha256) of named project/path keys, for hash-verified restores; bounded, unsent keys return in remaining.\"\"\"
        return await request('POST','/sources/read',payload={'paths':paths})
"""

CDC_RETIRE_START = "async def retire_unexpected_projection("
CDC_RETIRE_END = "await db.close()"
CDC_RETIRE_CHECK = "Unexpected projection retirement exceeds safety bound"
CDC_RETIRE_NEW = '''async def retire_unexpected_projection(expected: tuple[SourceDocument, ...], dry_run=False, allowed=frozenset()) -> dict:
    """Retire only stored document/chunk identities absent from the complete source.

    0.8.1-r3 retraction guard (Claude Code · Opus 5.5, 2026-09-26; owner order): a run never retracts a
    document on its own. Only paths named in `allowed` (the run request's retract_paths) are retracted;
    every other absent document is held, stays active, and is reported in held_paths.
    """
    import sq
    expected_paths = {row.source_path for row in expected}
    prefixes = ('docs/','propria/','consignatio/','advocatio/','vestigia/','family-court-workbench/')
    db = await sq.connect("docs", "probata", "docs")
    try:
        result = await db.query("SELECT id, source_path FROM document WHERE status != \\"retracted\\";")
        while isinstance(result, list) and len(result) == 1 and isinstance(result[0], list):
            result = result[0]
        managed=[row for row in (result if isinstance(result,list) else []) if isinstance(row,dict) and pipeline_owned(row)]
        observed = {row['source_path'] for row in managed if row['source_path'].startswith(prefixes)}
        unexpected = sorted(observed - expected_paths)
        if len(unexpected) > 1000:
            raise RuntimeError("Unexpected projection retirement exceeds safety bound")
        confirmed = [path for path in unexpected if path in allowed]
        held = [path for path in unexpected if path not in allowed]
        if confirmed and not dry_run:
            await db.query("""
BEGIN TRANSACTION;
UPDATE document SET status = "retracted" WHERE id IN array::map($ids, |$id| type::record($id));
UPDATE chunk SET status = "retracted" WHERE source_path IN $paths;
COMMIT TRANSACTION;
""", {"paths": confirmed,"ids":[str(row['id']) for row in managed if row['source_path'] in confirmed]})
        return {"retired_count": len(confirmed) if not dry_run else 0, "retraction_count":len(unexpected),
                "retired_paths": unexpected if dry_run else confirmed,
                "held_paths": [] if dry_run else held, "held_count": 0 if dry_run else len(held), "dry_run":dry_run}
    finally:
        await db.close()'''

WORKER_RETIRE_OLD = """        if True:
            summary['projection_retirement'] = asyncio.run(retire_unexpected_projection(source_snapshot))
            record()
"""
WORKER_RETIRE_NEW = """        if True:
            # 0.8.1-r3 retraction guard: only the paths this run request named in retract_paths are retracted.
            allowed = frozenset(filter(None, os.environ.get('DOCSTORE_RETRACT_PATHS', '').split('\\n')))
            summary['projection_retirement'] = asyncio.run(retire_unexpected_projection(source_snapshot, allowed=allowed))
            held = (summary['projection_retirement'] or {}).get('held_count') or 0
            if held:
                summary['retraction_held'] = summary['projection_retirement']['held_paths'][:50]
                summary['reason'] = (f'Retraction held: {held} stored document(s) are missing from the source; '
                                     'restore them, or name them in retract_paths to retract')
            record()
"""

API_PARAM_OLD = """    tracking_rebuild = payload.get("tracking_rebuild", False)
"""
API_PARAM_NEW = """    tracking_rebuild = payload.get("tracking_rebuild", False)
    retract_paths = payload.get("retract_paths") or []  # 0.8.1-r3 retraction guard
"""
API_VALIDATE_OLD = """    run_id = uuid.uuid4().hex
    env = dict(os.environ)
"""
API_VALIDATE_NEW = """    if (not isinstance(retract_paths, list) or len(retract_paths) > 1000
            or any(not isinstance(path, str) or not path or "\\n" in path for path in retract_paths)):
        raise HTTPException(status_code=400, detail="retract_paths must list up to 1000 stored source paths")
    run_id = uuid.uuid4().hex
    env = dict(os.environ)
    env.pop("DOCSTORE_RETRACT_PATHS", None)
    if retract_paths:
        env["DOCSTORE_RETRACT_PATHS"] = "\\n".join(retract_paths)
"""
API_RESPONSE_OLD = """            "selected_paths_are_verification_targets": bool(paths)})
"""
API_RESPONSE_NEW = """            "selected_paths_are_verification_targets": bool(paths),
            "retract_paths": retract_paths})
"""

SERVER_INDEX_OLD = """    async def docstore_index_full(full_reprocess: bool = False,
                                  tracking_rebuild: bool = False,
                                  index_kind: Literal["docs"] = "docs") -> dict:
        \"\"\"Start one governed full-source CocoIndex reconciliation.\"\"\"
        return await request("POST", "/runs", payload={
            "scope": "full", "paths": [], "full_reprocess": full_reprocess,
            "tracking_rebuild": tracking_rebuild,
            "index_kind": index_kind})
"""
SERVER_INDEX_NEW = """    async def docstore_index_full(full_reprocess: bool = False,
                                  tracking_rebuild: bool = False,
                                  index_kind: Literal["docs"] = "docs",
                                  retract_paths: Annotated[list[str] | None, Field(max_length=1000)] = None) -> dict:
        \"\"\"Start one governed full-source CocoIndex reconciliation. 0.8.1-r3: stored documents missing from the source are held, not retracted, unless named in retract_paths.\"\"\"
        payload = {"scope": "full", "paths": [], "full_reprocess": full_reprocess,
                   "tracking_rebuild": tracking_rebuild, "index_kind": index_kind}
        if retract_paths:
            payload["retract_paths"] = retract_paths
        return await request("POST", "/runs", payload=payload)
"""

# A first r3 build always sent "retract_paths": [] and broke test_governed_run_tools_use_live_job_api, which
# pins the unchanged run payload. Files that already carry the marker get this correction instead.
SERVER_INDEX_FIXUP_OLD = """        return await request("POST", "/runs", payload={
            "scope": "full", "paths": [], "full_reprocess": full_reprocess,
            "tracking_rebuild": tracking_rebuild,
            "index_kind": index_kind, "retract_paths": retract_paths or []})
"""
SERVER_INDEX_FIXUP_NEW = """        payload = {"scope": "full", "paths": [], "full_reprocess": full_reprocess,
                   "tracking_rebuild": tracking_rebuild, "index_kind": index_kind}
        if retract_paths:
            payload["retract_paths"] = retract_paths
        return await request("POST", "/runs", payload=payload)
"""
FIXUPS = {"plugins/docstore/control/server.py": [(SERVER_INDEX_FIXUP_OLD, SERVER_INDEX_FIXUP_NEW)]}

TEST_SYNC_OLD = """    assert plan['retracted_sources']
    sync('apply',{'files':files,'plan_id':plan['plan_id']})
    assert list((tmp_path/'to_be_deleted').rglob('note.md'))
"""
TEST_SYNC_NEW = """    assert plan['retracted_sources'] and set(plan['retracted_hashes'])==set(plan['retracted_sources'])
    copy=sync('read',{'paths':plan['retracted_sources']})  # 0.8.1-r3: restore source for a guarded retraction
    assert not copy['missing'] and copy['files'][0]['sha256']==plan['retracted_hashes'][copy['files'][0]['key']]
    with pytest.raises(ValueError): sync('apply',{'files':files,'plan_id':plan['plan_id']})
    assert not list((tmp_path/'to_be_deleted').rglob('note.md'))
    sync('apply',{'files':files,'plan_id':plan['plan_id'],'retract':plan['retracted_sources']})
    assert list((tmp_path/'to_be_deleted').rglob('note.md'))
"""
TEST_RETIRE_OLD = """    await retire_unexpected_projection(())
    assert upgrade.rows(await database.query('SELECT * FROM document:authored;'))[0]['status']=='active'
"""
TEST_RETIRE_NEW = """    held=await retire_unexpected_projection(())  # 0.8.1-r3: nothing named, nothing retracted
    assert held['held_paths']==[path] and held['retired_count']==0
    assert upgrade.rows(await database.query('SELECT * FROM type::record($id);',{'id':'document:'+stable_id(path)}))[0]['status']=='active'
    await retire_unexpected_projection((),allowed={path})
    assert upgrade.rows(await database.query('SELECT * FROM document:authored;'))[0]['status']=='active'
"""

ANCHORED = {
    "scripts/docstore/source_sync.py": [(SOURCE_OPERATION_OLD, SOURCE_OPERATION_NEW), (SOURCE_PLAN_OLD, SOURCE_PLAN_NEW),
                                        (SOURCE_GUARD_OLD, SOURCE_GUARD_NEW)],
    "plugins/docstore/control/release_tools.py": [(TOOLS_APPLY_OLD, TOOLS_APPLY_NEW)],
    "scripts/docstore/worker_sync.py": [(WORKER_RETIRE_OLD, WORKER_RETIRE_NEW)],
    "scripts/docstore/api.py": [(API_PARAM_OLD, API_PARAM_NEW), (API_VALIDATE_OLD, API_VALIDATE_NEW),
                                (API_RESPONSE_OLD, API_RESPONSE_NEW)],
    "plugins/docstore/control/server.py": [(SERVER_INDEX_OLD, SERVER_INDEX_NEW)],
    "tests/test_release.py": [(TEST_SYNC_OLD, TEST_SYNC_NEW), (TEST_RETIRE_OLD, TEST_RETIRE_NEW)],
}


def anchored(text: str, old: str, new: str) -> str:
    """Replace one anchor, matching it with CRLF or LF endings (never a mix); exactly one match required."""
    for newline in ("\r\n", "\n"):
        old_n = old.replace("\n", newline)
        count = text.count(old_n)
        if count == 1:
            return text.replace(old_n, new.replace("\n", newline), 1)
        if count > 1:
            break
    raise SystemExit(f"anchor not found exactly once; refusing to edit: {old.splitlines()[0][:80]!r}")


def retire_region(text: str) -> str:
    start = text.index(CDC_RETIRE_START)
    end = text.index(CDC_RETIRE_END, start) + len(CDC_RETIRE_END)
    if text.count(CDC_RETIRE_START) != 1 or CDC_RETIRE_CHECK not in text[start:end]:
        raise SystemExit("retire_unexpected_projection region is not the expected one; refusing to edit")
    return text[:start] + CDC_RETIRE_NEW + text[end:]


def main() -> None:
    args = [a for a in sys.argv[1:] if a != "--apply"]
    apply = "--apply" in sys.argv[1:]
    root = Path(args[0] if args else "/data/propria/releases/docstore-0.8.1")
    stamp = time.strftime("%Y%m%dT%H%M%SZ", time.gmtime())
    planned = []
    for relative in [*ANCHORED, "scripts/docstore/cdc_verify.py"]:
        target = root / relative
        raw = target.read_bytes()
        text = raw.decode("utf-8")
        if MARKER in text:
            pending = [(old, new) for old, new in FIXUPS.get(relative, [])
                       if any(old.replace("\n", nl) in text for nl in ("\r\n", "\n"))]
            if not pending:
                print(f"already applied  {relative}")
                continue
            new_text = text
            for old, new in pending:
                new_text = anchored(new_text, old, new)
            compile(new_text, str(target), "exec")
            planned.append((target, raw, new_text.encode("utf-8")))
            print(f"{'fixing' if apply else 'would fix'}  {relative} (r3 correction)")
            continue
        if relative.endswith("cdc_verify.py"):
            new_text = retire_region(text)
        else:
            new_text = text
            for old, new in ANCHORED[relative]:
                new_text = anchored(new_text, old, new)
        compile(new_text, str(target), "exec")
        planned.append((target, raw, new_text.encode("utf-8")))
        print(f"{'patching' if apply else 'would patch'}  {relative} (+{len(new_text.encode()) - len(raw)} bytes)")
    if not apply:
        return
    for target, raw, data in planned:
        target.with_name(f"{target.name}.bak-{stamp}-r3").write_bytes(raw)
        target.write_bytes(data)
    print(f"done; backups end in .bak-{stamp}-r3")


if __name__ == "__main__":
    main()
