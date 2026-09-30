"""Portable thin client. All Docstore operations use the mandatory hosted ctl MCP."""
from __future__ import annotations
import argparse
import asyncio
import hashlib
import json
import os
import re
from pathlib import Path
import sys
from urllib.parse import urlsplit
try:
    from fastmcp import Client
    from fastmcp.client.transports import StreamableHttpTransport
except ImportError:
    # Local fallback (Claude Code · Opus 5.5 · 2026-09-26): a desktop Python can carry an incompatible
    # fastmcp/mcp pair (seen: fastmcp 3.4.7 + mcp 2.0.0), which breaks this import. Re-run this client under
    # the plugin's own pinned requirements through uv instead of changing the global Python.
    import shutil as _shutil, subprocess as _subprocess
    if os.environ.get("DOCSTORE_CLIENT_REEXEC") != "1" and _shutil.which("uv"):
        _here = Path(__file__).resolve().parent
        raise SystemExit(_subprocess.call(
            ["uv", "run", "--no-project", "--quiet", "--with-requirements", str(_here / "requirements.txt"),
             "python", str(Path(__file__).resolve()), *sys.argv[1:]],
            env={**os.environ, "DOCSTORE_CLIENT_REEXEC": "1"}))
    raise


from connection_settings import load_connection, error_report


def connection():
    settings = load_connection()
    return Client(StreamableHttpTransport(settings.url, headers=settings.headers), timeout=180)


async def call(client,tool,args):
    names={item.name for item in await client.list_tools()}
    try:
        resolved=resolve_tool(names,tool)
    except ValueError:
        discovery=resolve_tool(names,'docstore_capabilities')
        schema=(await client.call_tool(discovery,{'operation':tool})).data
        read=schema['read_only'] or (tool=='docstore_adr' and args.get('action') in schema.get('read_actions',[]))
        result=await client.call_tool(resolve_tool(names,'docstore_query'),
            {'operation':tool,'arguments':args,'mode':'read' if read else 'write'})
    else:
        result=await client.call_tool(resolved,args)
    return result.data


def resolve_tool(names,tool):
    if tool in names:return tool
    matches=[name for name in names if name.endswith('-'+tool.replace('_','-'))]
    if len(matches)!=1:raise ValueError('Missing or ambiguous hosted ctl tool: '+tool)
    return matches[0]


# 0.8.2-sync-r3 (Claude Code · Opus 5.5, 2026-09-26). The hosted mirror accepts exactly these five projects. Their
# local folders come from the governed registry (Propria/docs/docstore-source-registry.json, whose docs/<project>
# junctions are the routing surface since the 2026-09-19 move under modules/); the pre-move paths are only a fallback.
# The registry's excluded and blocked patterns apply, so a reference dump such as advocatio's
# planning/original-context no longer aborts the upload (r2).
# Retraction guard (r3; owner order 2026-09-26, after 24 docs vanished from the desktop and a sync would have
# retracted them): a sync never retracts on its own. A document the mirror holds but this desktop lacks is restored
# from the mirror, hash-verified, by default; only keys named with --retract are retracted. Keys matching --hold /
# --hold-file keep the mirror's version untouched. Every uploaded doc git does not track is flagged: desktop-only
# docs are the ones that get lost.
SYNC_PROJECTS=('propria','probata','consignatio','consignatio-intake','advocatio')
LEGACY_ROOTS={'propria':'docs','probata':'Probata/probata/docs','consignatio':'Consignatio/docs','consignatio-intake':'Consignatio/Intake/docs','advocatio':'Legal-desktop/docs'}
PROPRIA_ALIASES={'probata','consignatio','consignatio-intake','advocatio','vestigia','family-court'}
MAX_DOCUMENT_BYTES=1024*1024


def glob_re(pattern):
    """Registry glob as the server's cdc_verify._glob_re: `**` spans directories, `*` and `?` stay in one segment."""
    out,i='^',0
    while i<len(pattern):
        if pattern.startswith('**/',i):
            out+='(?:.*/)?'; i+=3; continue
        if pattern.startswith('**',i):
            out+='.*'; i+=2; continue
        ch=pattern[i]
        out+='[^/]*' if ch=='*' else '[^/]' if ch=='?' else re.escape(ch)
        i+=1
    return re.compile(out+'$')


def sync_sources(root):
    """(project, local folder, include regexes, exclude regexes) for the five hosted projects."""
    registry_file=root/'docs'/'docstore-source-registry.json'
    registry=json.loads(registry_file.read_text(encoding='utf-8')) if registry_file.is_file() else {}
    blocked=[p for p in registry.get('blocked_patterns',[]) if isinstance(p,str)]
    entries={e.get('project_id'):e for e in registry.get('projects',[]) if isinstance(e,dict)}
    sources=[]
    for project in SYNC_PROJECTS:
        entry=entries.get(project,{})
        declared=str(entry.get('source_root') or LEGACY_ROOTS[project]).replace('\\','/').strip('/')
        folder=next((c for c in (declared,LEGACY_ROOTS[project]) if (root/c).is_dir()),None)
        if folder is None:
            raise ValueError('Docs folder not found for '+project+': '+declared)
        included=[glob_re(p) for p in (entry.get('included_patterns') or ['**/*.md'])]
        excluded=[glob_re(p) for p in list(entry.get('excluded_patterns') or [])+blocked]
        sources.append((project,folder,included,excluded))
    return sources


def decode_markdown(raw):
    """As the server's decode_markdown: strict UTF-8, else Windows-1252 with replacement."""
    try:
        return raw.decode('utf-8')
    except UnicodeDecodeError:
        return raw.decode('cp1252',errors='replace')


def stored_hash(text):
    """The store's document.content_hash: sha256 of the body with non-BMP characters folded to names."""
    import unicodedata
    folded=[]
    for char in text:
        if ord(char)<=0xFFFF:
            folded.append(char); continue
        try:
            folded.append(':'+unicodedata.name(char).lower().replace(' ','_')+':')
        except ValueError:
            folded.append(':u%04x:'%ord(char))
    return hashlib.sha256(''.join(folded).encode('utf-8')).hexdigest()


def hold_patterns(patterns,hold_file):
    """--hold patterns plus the non-comment lines of --hold-file."""
    found=list(patterns or [])
    if hold_file:
        found+=[line.strip() for line in Path(hold_file).read_text(encoding='utf-8').splitlines()
                if line.strip() and not line.lstrip().startswith('#')]
    return found


def tracked_files(directory):
    """Paths under directory that git tracks, relative to it; None when git cannot answer."""
    import subprocess
    try:
        out=subprocess.run(['git','-C',str(directory),'ls-files','-z','--','.'],capture_output=True,check=True).stdout
    except (OSError,subprocess.CalledProcessError):
        return None
    return {p for p in out.decode('utf-8','replace').split('\0') if p}


async def read_mirror(client,keys):
    """Exact mirror copies for project/path keys; each copy's sha256 is checked on arrival."""
    found={}; missing=[]; pending=list(keys)
    while pending:
        batch=pending[:20]
        result=await call(client,'docstore_source_read',{'paths':batch})
        for item in result.get('files',[]):
            if hashlib.sha256(item['content'].encode('utf-8')).hexdigest()!=item['sha256']:
                raise ValueError('Mirror copy failed its hash check: '+item['key'])
            found[item['key']]=item
        missing+=result.get('missing',[])
        remaining=list(result.get('remaining',[]))
        if len(remaining)==len(batch):
            raise ValueError('Mirror read made no progress')
        pending=remaining+pending[len(batch):]
    return found,missing


def manifest(entries):
    """Upload rows in a fixed order; a second byte-identical body is left out (document.content_hash is UNIQUE)."""
    files=[]; duplicates=[]; first={}
    for key in sorted(entries,key=lambda k:(SYNC_PROJECTS.index(k.split('/',1)[0]),k)):
        content=entries[key]; digest=stored_hash(content)
        if content.strip() and digest in first:
            duplicates.append({'skipped':key,'kept':first[digest]}); continue
        first.setdefault(digest,key)
        project,path=key.split('/',1)
        files.append({'project':project,'path':path,'content':content,'sha256':hashlib.sha256(content.encode()).hexdigest()})
    return files,duplicates


async def sync(client,root,apply=False,retract=(),hold=()):
    sources=sync_sources(root)
    folders={project:folder for project,folder,_,_ in sources}
    holds=[glob_re(p) for p in hold]
    held=lambda key: any(r.match(key) for r in holds)
    retract=set(retract)
    # ADR payloads are authoritative; projections are included in the manifest without local writes on dry run.
    projections=await call(client,'docstore_adr',{'action':'projections'})
    projected={p['path'].removeprefix('docs/'):p for p in projections.get('projections',[])}
    entries={}; present=set(); omitted={'over_1_mib':[],'held_local':[]}; untracked=[]
    for project,folder,included,excluded in sources:
        directory=(root/folder).resolve(strict=True)
        if not directory.is_relative_to(root):
            raise ValueError('Root escapes Propria')
        tracked=tracked_files(directory)
        for file in sorted(directory.rglob('*.md')):
            relative=file.relative_to(directory)
            rel=relative.as_posix(); key=project+'/'+rel
            if (any(p.startswith('.') or p.lower() in {'private','to_be_deleted','_to_be_deleted'} for p in relative.parts)
                or (project=='propria' and relative.parts[0].lower() in PROPRIA_ALIASES)
                or not any(r.match(rel) for r in included) or any(r.match(rel) for r in excluded)):
                continue
            if not file.resolve().is_relative_to(directory) or file.is_symlink():
                raise ValueError('Source symlinks are not supported')
            present.add(key)
            if tracked is not None and rel not in tracked and not (project=='probata' and rel in projected):
                untracked.append(key)
            if held(key):
                omitted['held_local'].append(key); continue
            if file.stat().st_size>MAX_DOCUMENT_BYTES:
                omitted['over_1_mib'].append(key); continue  # the server refuses these; listed, never silent
            entries[key]=decode_markdown(file.read_bytes())
        if project=='probata':
            entries.update({'probata/'+p:v['content'] for p,v in projected.items() if not held('probata/'+p)})
    for key in untracked:
        print('FLAG untracked doc (desktop only, not in git): '+key,file=sys.stderr)
    files,duplicates=manifest(entries)
    plan=await call(client,'docstore_source_plan',{'files':files})
    # Retraction guard: nothing the mirror holds leaves it unless --retract names it.
    named=sorted(retract & set(plan.get('retracted_sources',[])))
    candidates=[k for k in plan.get('retracted_sources',[]) if k not in retract]
    keep=[k for k in candidates if held(k) or k in present]   # held, or here but excluded/oversized
    restore=[k for k in candidates if k not in keep]          # absent from this desktop
    guard={'restore_from_mirror':restore,'keep_mirror_version':keep,'retract_named':named,
           'default_action':'restore missing documents from the mirror (hash-verified) and keep held or excluded ones; '
                            'retract only keys named with --retract'}
    summary={'folders':folders,'uploaded':len(files),'omitted':{**omitted,'duplicate_content':duplicates},
             'untracked':{'count':len(untracked),'paths':untracked},'retraction_guard':guard}
    if not apply:
        return {**plan,'client':summary}
    if candidates:
        copies,missing=await read_mirror(client,candidates)
        if missing:
            raise ValueError('Mirror no longer holds: '+', '.join(missing[:10]))
        hashes=plan.get('retracted_hashes',{}); restored=[]
        for key in restore:
            item=copies[key]; project,rel=key.split('/',1)
            target=root/folders[project]/rel
            if target.exists():
                raise ValueError('Refusing to overwrite '+str(target))
            if hashes.get(key) not in (None,item['sha256']):
                raise ValueError('Mirror copy changed since the plan: '+key)
            target.parent.mkdir(parents=True,exist_ok=True)
            target.write_bytes(item['content'].encode('utf-8'))
            if hashlib.sha256(target.read_bytes()).hexdigest()!=item['sha256']:
                raise ValueError('Restored copy failed its hash check: '+key)
            restored.append({'key':key,'sha256':item['sha256']})
            print('FLAG restored from mirror (hash-verified): '+key,file=sys.stderr)
        entries.update({key:copies[key]['content'] for key in candidates})
        files,duplicates=manifest(entries)
        plan=await call(client,'docstore_source_plan',{'files':files})
        summary.update(uploaded=len(files),restored=restored)
        summary['omitted']['duplicate_content']=duplicates
    unconfirmed=[k for k in plan.get('retracted_sources',[]) if k not in retract]
    if unconfirmed:
        raise ValueError('Retraction guard: the plan would still retract '+', '.join(unconfirmed[:10]))
    receipt=await call(client,'docstore_source_apply',{'files':files,'plan_id':plan['plan_id'],
                                                       'retract':sorted(retract & set(plan.get('retracted_sources',[])))})
    # Preserve existing local projections before materializing the authoritative version.
    for path,item in projected.items():
        target=root/folders['probata']/path
        target.parent.mkdir(parents=True,exist_ok=True)
        if target.exists() and target.read_bytes()!=item['content'].encode():
            archive=root/'to_be_deleted'/('adr-'+receipt['generation'])/path
            archive.parent.mkdir(parents=True,exist_ok=True); target.replace(archive)
        target.write_bytes(item['content'].encode())
    return {**receipt,'client':summary}


async def main(args):
    async with connection() as client:
        catalog={tool.name for tool in await client.list_tools()}
        required={'docstore_health','docstore_capabilities','docstore_query','coco_docstore_search','docstore_get'}
        for tool in required:
            resolve_tool(catalog,tool)
        if args.command=='sync':
            return await sync(client,Path(args.root).resolve(strict=True),args.apply,
                              args.retract,hold_patterns(args.hold,args.hold_file))
        if args.command=='catalog':
            return {'tools':sorted(catalog)}
        return await call(client,args.tool,json.loads(args.json))


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    sub=parser.add_subparsers(dest='command',required=True)
    sub.add_parser('catalog')
    invoke=sub.add_parser('call'); invoke.add_argument('tool'); invoke.add_argument('--json',default='{}')
    sync_parser=sub.add_parser('sync'); sync_parser.add_argument('--root',required=True); sync_parser.add_argument('--apply',action='store_true')
    sync_parser.add_argument('--retract',action='append',default=[],metavar='PROJECT/PATH',
                             help='retract this exact mirror document (repeatable); nothing is retracted otherwise')
    sync_parser.add_argument('--hold',action='append',default=[],metavar='PATTERN',
                             help='keep the mirror version of matching project/path keys (glob, repeatable)')
    sync_parser.add_argument('--hold-file',metavar='FILE',help='file of --hold patterns, one per line')
    try:
        print(json.dumps(asyncio.run(main(parser.parse_args())),ensure_ascii=False,default=str))
    except Exception as exc:
        print(json.dumps(error_report(exc)),file=sys.stderr)
        raise SystemExit(1)
