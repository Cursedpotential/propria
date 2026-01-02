from __future__ import annotations
from pathlib import Path
from typing import Dict, Any, List
import yaml, re

def _merge_dict_lists(dst: Dict[str, list], src: Dict[str, list]):
    for k, v in (src or {}).items():
        dst.setdefault(k, [])
        if isinstance(v, list):
            dst[k].extend(v)
        else:
            # Treat scalars as a single-item list
            dst[k].append(v)

def load_rules_from_dir(dir_path: Path) -> Dict[str, Any]:
    """Load and merge ALL *.yaml files in dir_path.
    Recognizes known keys: behaviors, mcl, entities, spelling_variants, sequences, severity.
    For files named exactly like those, treat the whole file as that section.
    For any other yaml, merge top-level keys into 'behaviors' by default.
    """
    dir_path = Path(dir_path)
    files = sorted(dir_path.glob("*.yaml"))
    out = {"behaviors": {}, "mcl": {}, "entities": {}, "spelling_variants": {}, "sequences": {}, "severity": {}, "parties": {}}

    for f in files:
        data = yaml.safe_load(f.read_text(encoding="utf-8")) or {}
        name = f.stem.lower()
        if name in ("behaviors", "mcl", "entities", "spelling_variants", "sequences", "severity", "parties") and isinstance(data, dict):
            # merge into named section
            if name in ("behaviors", "mcl"):
                _merge_dict_lists(out[name], data)
            else:
                # shallow merge
                for k, v in data.items():
                    out[name][k] = v
        else:
            # treat as behaviors-extension
            if isinstance(data, dict):
                _merge_dict_lists(out["behaviors"], data)
    # Deduplicate list items per category
    for section in ("behaviors", "mcl"):
        for cat, items in out[section].items():
            # Convert dict items to jsonable tuples for de-dup
            norm = []
            seen = set()
            for it in items:
                if isinstance(it, dict):
                    key = (it.get("pattern",""), bool(it.get("regex")), it.get("weight"), it.get("severity"))
                    if key in seen: continue
                    seen.add(key)
                    norm.append(it)
                else:
                    if it in seen: continue
                    seen.add(it)
                    norm.append(it)
            out[section][cat] = norm
    return out
