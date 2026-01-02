#!/usr/bin/env python3
from __future__ import annotations
import os, re, json, zipfile, argparse
from pathlib import Path
DEFAULT_PATTERNS = [
    r"(?i)pass1_parse_json.*\.py$", r"(?i)pass4_analytics.*\.py$", r"(?i)orchestrator.*\.py$",
    r"^README\.md$", r"(?i)timeline.*processor.*\.md$", r"(?i)place[_-]?id.*\.json$", r"(?i)latlng.*\.json$",
    r"(?i)sample.*2024.*\.json$", r"(?i)google.*2024.*\.json$", r"^data\.json$", r"^json\.json$",
    r"(?i)analysis_modules\.json$", r"(?i)schedule_config\.json$", r"(?i)flags_config\.json$",
    r"(?i)raw_api_responses.*\.json$", r"(?i)place_id_db\.json$", r"(?i)geocoding_cache\.json(\.txt)?$",
    r"(?i)scr\.zip$",
]
DEFAULT_EXTRAS=["timeline_etl_skeleton_v0_6.zip","AGENT_PROMPT.md"]
def iter_files(base: Path):
    for root,_,files in os.walk(base):
        for f in files: yield Path(root)/f
def match_patterns(paths, patterns):
    comp=[re.compile(p) for p in patterns]; out={p.pattern:[] for p in comp}
    for p in paths:
        name=p.name
        for r in comp:
            if r.search(name): out[r.pattern].append(str(p))
    return out
def collect(home: Path, downloads: Path, patterns, extras):
    seen=set(); included=[]; missing={}
    paths=list(iter_files(home))+list(iter_files(downloads))
    matches=match_patterns(paths, patterns)
    for pat,files in matches.items():
        if not files: missing[pat]=missing.get(pat,0)+1
        for f in files:
            if f not in seen: seen.add(f); included.append(f)
    for e in extras:
        p=Path(e).resolve()
        if p.exists() and str(p) not in seen: included.append(str(p)); seen.add(str(p))
        else:
            if not p.exists(): missing[str(p)]=missing.get(str(p),0)+1
    return included, missing
def add_to_zip(zip_path: Path, files):
    with zipfile.ZipFile(zip_path,"w",zipfile.ZIP_DEFLATED) as z:
        for f in files:
            p=Path(f); z.write(str(p), f"collected/{p.name}")
def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--home",default=str(Path.home()/ "matt"))
    ap.add_argument("--downloads",default=str(Path.home()/ "matt"/"downloads"))
    ap.add_argument("--out",default=str(Path.home()/ "matt"/"etl_hand_off.zip"))
    ap.add_argument("--extra",action="append",default=[])
    ap.add_argument("--dry-run",action="store_true")
    args=ap.parse_args()
    home=Path(os.path.expanduser(args.home)).resolve()
    dl=Path(os.path.expanduser(args.downloads)).resolve()
    extras=list(DEFAULT_EXTRAS)+args.extra
    included, missing=collect(home, dl, DEFAULT_PATTERNS, extras)
    man=Path(str(args.out)+".manifest.json"); man.parent.mkdir(parents=True,exist_ok=True)
    man.write_text(json.dumps({"out":args.out,"included":included,"missing":missing}, indent=2), encoding="utf-8")
    if args.dry_run:
        print(man.read_text()); print("[dry-run]"); return
    add_to_zip(Path(os.path.expanduser(args.out)).resolve(), included)
    print(f"[done] wrote {args.out}\n[manifest] {man}")
if __name__=="__main__": main()
