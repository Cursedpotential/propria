"""Retraction guard, --hold and the untracked-docs flag for `client.py sync` (four installed 0.8.2 copies).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Owner order 2026-09-26 (relayed by the parent session), after 24 Probata docs were found missing from the
desktop and a routine sync would have retracted them: "make lost docs impossible to miss".
  1. A sync never retracts a document on its own. A document the mirror holds but this desktop lacks is
     restored from the mirror, hash-verified, by default; only keys named with --retract are retracted.
     (The server refuses unnamed retractions too: apply_r3.py.)
  2. Every uploaded doc that git does not track is flagged, one line per file on stderr and as a count +
     list in the result: docs that live only on the desktop are the ones that get lost.
  3. --hold PATTERN / --hold-file FILE: matching project/path keys keep the mirror's version untouched.
Replaces the 0.8.2-sync-r2 block (patch_client_sync.py) and extends main()/argparse; everything else in
client.py stays byte-for-byte. Needs the server's docstore_source_read (0.8.1-r3) for restores.
Idempotent (marker 0.8.2-sync-r3); anchors must occur exactly once; compiles before writing; keeps CRLF.

    python3 patch_client_guard.py            # report
    python3 patch_client_guard.py --apply
"""
from __future__ import annotations

import argparse
import hashlib
from pathlib import Path

from patch_client_sync import TARGETS  # noqa: E402  (same four installed copies)

MARKER = "0.8.2-sync-r3"
BLOCK_START = "# 0.8.2-sync-r2 (Claude Code"
BLOCK_END = "async def main(args):\n"
MAIN_OLD = "            return await sync(client,Path(args.root).resolve(strict=True),args.apply)\n"
MAIN_NEW = ("            return await sync(client,Path(args.root).resolve(strict=True),args.apply,\n"
            "                              args.retract,hold_patterns(args.hold,args.hold_file))\n")
ARGS_OLD = ("    sync_parser=sub.add_parser('sync'); sync_parser.add_argument('--root',required=True); "
            "sync_parser.add_argument('--apply',action='store_true')\n")
ARGS_NEW = ARGS_OLD + (
    "    sync_parser.add_argument('--retract',action='append',default=[],metavar='PROJECT/PATH',\n"
    "                             help='retract this exact mirror document (repeatable); nothing is retracted otherwise')\n"
    "    sync_parser.add_argument('--hold',action='append',default=[],metavar='PATTERN',\n"
    "                             help='keep the mirror version of matching project/path keys (glob, repeatable)')\n"
    "    sync_parser.add_argument('--hold-file',metavar='FILE',help='file of --hold patterns, one per line')\n")

NEW_BLOCK = r'''# 0.8.2-sync-r3 (Claude Code · Opus 5.5, 2026-09-26). The hosted mirror accepts exactly these five projects. Their
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


'''


SKILL_OLD = ("Use python client.py sync --root <Propria> for a dry run and add --apply to apply the exact manifest. "
             "Missing or empty roots are errors. Then docstore_index_full starts one complete incremental CocoIndex run. "
             "Follow docstore_run or docstore_current_run to terminal status; inspect attribution. Complete scope is "
             "required because CocoIndex owns one application. Removed sources are quarantined locally and retracted "
             "remotely. Do not use full_reprocess or tracking_rebuild for ordinary refreshes.")
SKILL_NEW = ("Use bin/docstore-client.sh sync --root <Propria> (or python client.py sync ...) for a dry run and add "
             "--apply to apply the exact manifest. Folders and exclusions come from "
             "Propria/docs/docstore-source-registry.json. Missing or empty roots are errors. Then docstore_index_full "
             "starts one complete incremental CocoIndex run. Follow docstore_run_current to terminal status; inspect "
             "attribution. Complete scope is required because CocoIndex owns one application.\n\n"
             "Retraction guard (0.8.2-sync-r3, owner order 2026-09-26): nothing is retracted unless it is named. A "
             "document the mirror holds but this desktop lacks is restored from the mirror, hash-verified, by default; "
             "--retract PROJECT/PATH retracts one named document (repeatable); --hold PATTERN or --hold-file FILE keeps "
             "the mirror's version of matching keys. The result lists every uploaded doc that git does not track (also "
             "one FLAG line each on stderr): desktop-only docs are the ones that get lost. An index run likewise holds, "
             "rather than retracts, stored documents missing from the source unless docstore_index_full names them in "
             "retract_paths. Do not use full_reprocess or tracking_rebuild for ordinary refreshes.")
CHANGELOG_ENTRY = """
## 0.8.2 sync repair and retraction guard (2026-09-26, Claude Code · Opus 5.5)

- `sync` read the pre-2026-09-19 folders (`Probata/probata/docs` etc., now under `modules/`) and ignored the
  registry's exclusions, so every upload failed and nothing reached the source mirror after 2026-09-20. Folders
  and exclusions now come from `Propria/docs/docstore-source-registry.json` (0.8.2-sync-r2).
- Retraction guard (0.8.2-sync-r3, owner order): a sync never retracts on its own. Missing documents are restored
  from the mirror (hash-verified) by default; `--retract PROJECT/PATH` names one to retract; `--hold` / `--hold-file`
  keep the mirror's version. Docs that git does not track are flagged. Needs the server's 0.8.1-r3
  `docstore_source_read`. Record: Probata `docs/pending-review/2026-09-26-docstore-0.8.1-r2/`.
"""
SKILL_TARGETS = [target.parent / "skills/index/SKILL.md" for target in TARGETS]
CHANGELOG_TARGETS = [target.parent / "CHANGELOG.md" for target in TARGETS if (target.parent / "CHANGELOG.md").exists()]


def patch_docs(apply: bool) -> None:
    """Keep the index skill and the Claude changelog true to the new sync behavior (doc-drift rule)."""
    for target in SKILL_TARGETS:
        raw = target.read_bytes()
        crlf = b"\r\n" in raw
        text = raw.decode("utf-8").replace("\r\n", "\n")
        if MARKER in text:
            print(f"already applied  {target}")
            continue
        if text.count(SKILL_OLD) != 1:
            raise SystemExit(f"skill sentence not found exactly once: {target}")
        text = text.replace(SKILL_OLD, SKILL_NEW, 1)
        if apply:
            target.write_bytes((text.replace("\n", "\r\n") if crlf else text).encode("utf-8"))
        print(f"{'patched' if apply else 'would patch'}      {target}")
    for target in CHANGELOG_TARGETS:
        text = target.read_text(encoding="utf-8")
        if MARKER in text:
            print(f"already applied  {target}")
            continue
        if apply:
            target.write_text(text.rstrip("\n") + "\n" + CHANGELOG_ENTRY, encoding="utf-8")
        print(f"{'patched' if apply else 'would patch'}      {target}")


def patched(text: str) -> str:
    for anchor in (BLOCK_START, BLOCK_END, MAIN_OLD, ARGS_OLD):
        if text.count(anchor) != 1:
            raise SystemExit(f"anchor not found exactly once; refusing to edit: {anchor[:60]!r}")
    start, end = text.index(BLOCK_START), text.index(BLOCK_END)
    if end < start:
        raise SystemExit("the r2 block does not precede main(); refusing to edit")
    text = text[:start] + NEW_BLOCK + text[end:]
    return text.replace(MAIN_OLD, MAIN_NEW, 1).replace(ARGS_OLD, ARGS_NEW, 1)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    for target in TARGETS:
        raw = target.read_bytes()
        crlf = b"\r\n" in raw
        text = raw.decode("utf-8").replace("\r\n", "\n")
        if MARKER in text:
            print(f"already applied  {target}")
            continue
        new_text = patched(text)
        compile(new_text, str(target), "exec")
        data = new_text.replace("\n", "\r\n").encode("utf-8") if crlf else new_text.encode("utf-8")
        if not args.apply:
            print(f"would patch      {target}")
            continue
        target.write_bytes(data)
        print(f"patched          {target} sha256 {hashlib.sha256(data).hexdigest()[:16]}")
    patch_docs(args.apply)


if __name__ == "__main__":
    main()
