"""Hash-validated complete docs projection upload; removed files are quarantined."""
from __future__ import annotations
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import uuid
from scope import ROOTS, validate_registry
from upgrade import digest
from run_support import worker_lock

ALIASES={'probata','consignatio','consignatio-intake','advocatio','vestigia','family-court'}

def excluded(project,parts):
    return any(p.startswith('.') or p.lower() in {'private','to_be_deleted','_to_be_deleted'} for p in parts) or (project=='propria' and parts[0].lower() in ALIASES)


def validated(files, declared_roots=None):
    """Validate bounded docs and independent root coverage without writing.

    Inputs are hashed file entries and optional approved project IDs; output is
    the validated file list. Like state(), root identity comes from ROOTS, not
    file presence. Legacy files-only callers must still represent every root.
    Byline: Codex / GPT-6.1 / 2026-10-07.
    """
    if declared_roots is not None:
        if (not isinstance(declared_roots,list)
            or not all(isinstance(root,str) for root in declared_roots)
            or len(declared_roots)!=len(ROOTS)
            or set(declared_roots)!=set(ROOTS)):
            raise ValueError('roots must declare exactly the seven approved docs roots')
    minimum = 0 if declared_roots is not None else len(ROOTS)
    if not isinstance(files,list) or not minimum<=len(files)<=5000:
        raise ValueError('A complete bounded seven-root manifest is required')
    seen=set(); file_roots=set(); size=0
    for item in files:
        if not isinstance(item,dict):
            raise ValueError('Source entries must be objects')
        project=item.get('project'); path=item.get('path',''); content=item.get('content','')
        if not isinstance(project,str) or not isinstance(path,str):
            raise ValueError('Source project and path must be text')
        relative=PurePosixPath(path)
        if (project not in ROOTS or not path or '\\' in path or ':' in path or relative.is_absolute()
            or '..' in relative.parts or path!=relative.as_posix() or relative.suffix.lower()!='.md'
            or excluded(project,relative.parts)):
            raise ValueError('Path outside docs-only upload scope')
        if not isinstance(content,str):
            raise ValueError('Content must be UTF-8 text')
        data=content.encode('utf-8'); size+=len(data)
        if len(data)>1024*1024 or size>32*1024*1024 or (project,path) in seen:
            raise ValueError('Duplicate or oversized source')
        if hashlib.sha256(data).hexdigest()!=item.get('sha256'):
            raise ValueError('Source hash mismatch')
        file_roots.add(project); seen.add((project,path))
    if declared_roots is None and file_roots!=set(ROOTS):
        raise ValueError('Every docs root must be represented; empty-root removal needs a separate reviewed migration')
    return files


def state():
    registry=validate_registry(json.loads(Path(os.environ['DOCSTORE_PROJECT_REGISTRY']).read_text()))
    root=Path(registry['monorepo_root']).resolve(strict=True)
    existing={}
    for project,(path,_) in ROOTS.items():
        directory=root/path
        if directory.is_symlink() or not directory.resolve().is_relative_to(root):
            raise ValueError('Source root cannot escape mirror')
        for file in directory.rglob('*.md'):
            relative=file.relative_to(directory)
            if excluded(project,relative.parts):
                continue
            if file.is_symlink() or not file.resolve().is_relative_to(directory.resolve()):
                raise ValueError('Source symlink refused')
            existing[project+'/'+relative.as_posix()]=hashlib.sha256(file.read_bytes()).hexdigest()
    return root,existing


READ_BUDGET=1024*1024  # 0.8.1-r3: keeps each read reply well under the ctl client's 2 MiB response cap


def read(payload):
    """0.8.1-r3 (Claude Code · Opus 5.5, 2026-09-26): exact mirror copies of named project/path keys, so a
    client can restore a document it lacks instead of retracting it. Bounded; unsent keys come back in
    `remaining`, keys the mirror does not hold in `missing`."""
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
    """Plan/apply an exact hashed projection, or read named mirror files.

    Input is a source operation payload; output is its bounded receipt. Plan
    and read are nonmutating; apply quarantines changes after the exact-plan
    and named-retraction guards. roots decouples coverage from file presence.
    Byline: Codex / GPT-6.1 / 2026-10-07.
    """
    if action=='read':
        return read(payload)
    declared_roots=payload.get('roots')
    files=validated(payload.get('files'),declared_roots)
    lock=Path(os.environ.get('DOCSTORE_SYNC_LOCK','/data/state/sync.lock'))
    with worker_lock(lock):
        root,existing=state()
        incoming={f[ 'project']+'/'+f['path']:f['sha256'] for f in files}
        retracted=sorted(set(existing)-set(incoming))
        plan={'changed':sorted(k for k,v in incoming.items() if existing.get(k)!=v),
              'retracted_sources':retracted, 'count':len(files),
              'current_digest':digest(existing),'incoming_digest':digest(incoming),
              # 0.8.1-r3: the mirror's hash of every document this plan would retract, so a client can
              # restore it byte-for-byte instead (retraction guard, owner order 2026-09-26).
              'retracted_hashes':{k:existing[k] for k in retracted}}
        if declared_roots is not None:
            # Bind independent root coverage into the exact plan receipt.
            plan['roots']=sorted(declared_roots)
        plan['plan_id']=digest(plan)
        if action=='plan':
            return plan
        if action!='apply' or payload.get('plan_id')!=plan['plan_id']:
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
        # Validate all final paths before any write. Never follow an existing parent symlink.
        destinations=[]
        for item in files:
            directory=root/ROOTS[item['project']][0]
            target=directory/item['path']
            if not target.resolve().is_relative_to(directory.resolve()) or any(p.is_symlink() for p in [target,*target.parents] if p.is_relative_to(root)):
                raise ValueError('Source destination escapes docs root')
            destinations.append((item,target))
        generation=uuid.uuid4().hex
        journal=lock.parent/'source-sync.json'
        journal.write_text(json.dumps({'state':'applying','generation':generation,'plan':plan}),encoding='utf-8')
        quarantine=root/'to_be_deleted'/('sync-'+generation)
        for identity in plan['retracted_sources']:
            project,path=identity.split('/',1)
            source=root/ROOTS[project][0]/path
            target=quarantine/project/path
            target.parent.mkdir(parents=True,exist_ok=True)
            source.replace(target)
        for item,target in destinations:
            if item['project']+'/'+item['path'] not in plan['changed']:
                continue
            target.parent.mkdir(parents=True,exist_ok=True)
            if target.exists():
                backup=quarantine/item['project']/item['path']
                backup.parent.mkdir(parents=True,exist_ok=True)
                target.replace(backup)
            temporary=target.with_name(target.name+'.'+generation+'.pending')
            temporary.write_bytes(item['content'].encode('utf-8'))
            temporary.replace(target)
        _,actual=state()
        if actual!=incoming:
            raise RuntimeError('Projection verification failed; indexing must not start')
        journal.write_text(json.dumps({'state':'verified','generation':generation,'digest':digest(actual)}),encoding='utf-8')
        return {**plan,'applied':True,'generation':generation,'verified':True,'index_started':False}
