"""Repair `client.py sync` in the four installed propria-docstore 0.8.2 client copies.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Defect (found 2026-09-26): `sync()` hard-coded the pre-2026-09-19 folders
(`Probata/probata/docs`, `Consignatio/docs`, `Consignatio/Intake/docs`, `Legal-desktop/docs`),
which moved under `modules/` on 2026-09-19, and it ignored the governed registry's
excluded/blocked patterns, so advocatio's `planning/original-context` reference dump (files over
1 MiB, 32 MB in all) aborted every upload. Nothing reached the VPS source mirror after 2026-09-20.

Repair: folders come from `Propria/docs/docstore-source-registry.json` (its `docs/<project>`
junctions are the routing surface), with the old paths as a fallback; the registry's
excluded_patterns + blocked_patterns apply with the server's glob semantics; a file over the
server's 1 MiB limit and a second byte-identical body (document.content_hash is UNIQUE in the
store) are left out and LISTED in the result under `client.omitted`, never silently.
Everything else in client.py (including the 17:36 uv re-exec fallback) is left byte-for-byte.

Idempotent (marker `0.8.2-sync-r2`); refuses to edit unless each anchor occurs exactly once;
compiles the result before writing; keeps CRLF. Dry run by default, `--apply` writes.

    python3 patch_client_sync.py            # report what would change
    python3 patch_client_sync.py --apply
"""
from __future__ import annotations

import argparse
import hashlib
from pathlib import Path

HOME = Path.home()
TARGETS = [
    HOME / ".claude/local-plugins/plugins/propria-docstore/client.py",
    HOME / ".claude/plugins/cache/casebible-local/propria-docstore/0.8.2/client.py",
    HOME / ".codex/local-marketplaces/propria-docstore/plugins/propria-docstore/client.py",
    HOME / ".codex/plugins/cache/propria-docstore-local/propria-docstore/0.8.2/client.py",
]
MARKER = "0.8.2-sync-r2"
EXPECTED_BEFORE = "a7aa7d250b2a0bf6c54149a4b2d78fde5ba5c232ae246903e8b6d13fee3929fb"
IMPORT_ANCHOR = "import os\n"
START_ANCHOR = "async def sync(client,root,apply=False):\n"
END_ANCHOR = "async def main(args):\n"

NEW_SYNC = r'''# 0.8.2-sync-r2 (Claude Code · Opus 5.5, 2026-09-26). The hosted mirror accepts exactly these five
# projects. Their local folders come from the governed registry (Propria/docs/docstore-source-registry.json,
# whose docs/<project> junctions are the routing surface since the 2026-09-19 move under modules/); the
# pre-move paths are only a fallback for an older checkout. The registry's excluded and blocked patterns now
# apply, so a reference dump such as advocatio's planning/original-context no longer aborts the upload.
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


async def sync(client,root,apply=False):
    sources=sync_sources(root)
    folders={project:folder for project,folder,_,_ in sources}
    # ADR payloads are authoritative; projections are included in the manifest without local writes on dry run.
    projections=await call(client,'docstore_adr',{'action':'projections'})
    projected={p['path'].removeprefix('docs/'):p for p in projections.get('projections',[])}
    files=[]; omitted={'over_1_mib':[],'duplicate_content':[]}; first_seen={}
    for project,folder,included,excluded in sources:
        directory=(root/folder).resolve(strict=True)
        if not directory.is_relative_to(root):
            raise ValueError('Root escapes Propria')
        found={}
        for file in sorted(directory.rglob('*.md')):
            relative=file.relative_to(directory)
            rel=relative.as_posix()
            if (any(p.startswith('.') or p.lower() in {'private','to_be_deleted','_to_be_deleted'} for p in relative.parts)
                or (project=='propria' and relative.parts[0].lower() in PROPRIA_ALIASES)
                or not any(r.match(rel) for r in included) or any(r.match(rel) for r in excluded)):
                continue
            if not file.resolve().is_relative_to(directory) or file.is_symlink():
                raise ValueError('Source symlinks are not supported')
            if file.stat().st_size>MAX_DOCUMENT_BYTES:
                omitted['over_1_mib'].append(project+'/'+rel)  # the server refuses these; listed, never silent
                continue
            found[rel]=decode_markdown(file.read_bytes())
        if project=='probata':
            found.update({p:v['content'] for p,v in projected.items()})
        for rel in sorted(found):
            content=found[rel]
            # document.content_hash is UNIQUE in the store, so a second byte-identical body could never index.
            digest=stored_hash(content)
            if content.strip() and digest in first_seen:
                omitted['duplicate_content'].append({'skipped':project+'/'+rel,'kept':first_seen[digest]})
                continue
            first_seen.setdefault(digest,project+'/'+rel)
            files.append({'project':project,'path':rel,'content':content,'sha256':hashlib.sha256(content.encode()).hexdigest()})
    summary={'folders':folders,'uploaded':len(files),'omitted':omitted}
    plan=await call(client,'docstore_source_plan',{'files':files})
    if not apply:
        return {**plan,'client':summary}
    receipt=await call(client,'docstore_source_apply',{'files':files,'plan_id':plan['plan_id']})
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


def patched(text: str) -> str:
    if text.count(IMPORT_ANCHOR) != 1 or text.count(START_ANCHOR) != 1 or text.count(END_ANCHOR) != 1:
        raise SystemExit("anchor count is not exactly one; refusing to edit")
    start, end = text.index(START_ANCHOR), text.index(END_ANCHOR)
    if end < start:
        raise SystemExit("sync() does not precede main(); refusing to edit")
    text = text[:start] + NEW_SYNC + text[end:]
    return text.replace(IMPORT_ANCHOR, IMPORT_ANCHOR + "import re\n", 1)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    for target in TARGETS:
        raw = target.read_bytes()
        digest = hashlib.sha256(raw).hexdigest()
        crlf = b"\r\n" in raw
        text = raw.decode("utf-8").replace("\r\n", "\n")
        if MARKER in text:
            print(f"already applied  {target}")
            continue
        drift = "" if digest == EXPECTED_BEFORE else f" (before-hash {digest[:16]} differs from the recorded {EXPECTED_BEFORE[:16]})"
        new_text = patched(text)
        compile(new_text, str(target), "exec")
        data = new_text.replace("\n", "\r\n").encode("utf-8") if crlf else new_text.encode("utf-8")
        if not args.apply:
            print(f"would patch      {target}{drift}")
            continue
        target.write_bytes(data)
        print(f"patched          {target} sha256 {hashlib.sha256(data).hexdigest()[:16]}{drift}")


if __name__ == "__main__":
    main()
