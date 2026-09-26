#!/usr/bin/env python3
import sys, os, re, csv, json, gzip, hashlib, math
from datetime import datetime, timezone
from typing import Iterator, Dict, Any, Optional

ET_TZ = "America/Detroit"  # used only for run-folder stamp; timestamps stay UTC
RUN_STAMP_FMT = "%Y%m%d_%H%M%S"

def now_run_stamp():
    # Use UTC for determinism; folder name is informational
    return datetime.utcnow().strftime(RUN_STAMP_FMT)

def ensure_dir(p: str):
    os.makedirs(p, exist_ok=True)

def open_maybe_gzip(path: str, mode: str = "rt", encoding="utf-8"):
    if path.endswith(".gz"):
        return gzip.open(path, mode, encoding=encoding)
    return open(path, mode, encoding=encoding)

def to_utc_iso(ts: Optional[str]) -> Optional[str]:
    if not isinstance(ts, str): return None
    try:
        from datetime import datetime
        import dateutil.parser as dp  # falls back to pandas if available; else simple parser
        t = dp.isoparse(ts)
        return t.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    except Exception:
        try:
            import pandas as pd
            t = pd.to_datetime(ts, utc=True, errors="coerce")
            if str(t) == "NaT": return None
            return t.tz_convert("UTC").strftime("%Y-%m-%dT%H:%M:%SZ")
        except Exception:
            return None

def extract_offset_minutes(ts: Optional[str]) -> Optional[int]:
    if not isinstance(ts, str): return None
    m = re.search(r'([+-])(\d{2}):?(\d{2})$', ts)
    if not m: return None
    sign = 1 if m.group(1)=="+" else -1
    return sign*(int(m.group(2))*60+int(m.group(3)))

def clean_latlng(s: Optional[str]) -> Optional[str]:
    if not isinstance(s, str): return None
    s2 = s.replace("°","").replace(" ", "").strip()
    m = re.match(r"^([+-]?\d+(?:\.\d+)?),([+-]?\d+(?:\.\d+)?)(?:,.*)?$", s2)
    if not m: return None
    return f"{float(m.group(1))},{float(m.group(2))}"

def sha(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()

def write_csv_rowwise(rows_iter: Iterator[Dict[str, Any]], out_csv: str, header: list):
    tmp = out_csv + ".tmp"
    with open(tmp, "w", encoding="utf-8", newline="") as f:
        w = csv.DictWriter(f, fieldnames=header, extrasaction="ignore")
        w.writeheader()
        for r in rows_iter:
            w.writerow(r)
    os.replace(tmp, out_csv)

def iter_segments_stream(path: str):
    """
    Streaming reader that supports either:
      1) NDJSON/JSONL: each line is a semantic segment object
      2) Full JSON with top-level { "semanticSegments": [...] }
    We avoid third-party deps. For case (2) we use a simple bracket counter over the array.
    """
    with open_maybe_gzip(path, "rt") as f:
        head = f.read(4096)
        f.seek(0)
        if head.lstrip().startswith("{"):
            # Try full JSON with "semanticSegments": [ ... ]
            buf = []
            in_array = False
            depth = 0
            key_seen = False
            # naive state machine
            while True:
                ch = f.read(1)
                if not ch:
                    break
                buf.append(ch)
                if not key_seen:
                    if '"semanticSegments"' in "".join(buf):
                        key_seen = True
                        buf = []  # drop data before key
                else:
                    # Wait until we hit '[' that starts the array
                    if not in_array:
                        if ch == '[':
                            in_array = True
                            obj = []
                            brace = 0
                            in_obj = False
                    else:
                        # inside the array; capture each JSON object by matching braces
                        if ch == '{':
                            in_obj = True
                            brace = 1
                            obj = ['{']
                        elif in_obj:
                            obj.append(ch)
                            if ch == '{':
                                brace += 1
                            elif ch == '}':
                                brace -= 1
                                if brace == 0:
                                    # end of object
                                    try:
                                        yield json.loads("".join(obj))
                                    except Exception:
                                        pass
                                    in_obj = False
                        # array end is ']' — stop
                        # we don't rely on commas
        else:
            # NDJSON: yield per line
            for line in f:
                s = line.strip()
                if not s: continue
                try:
                    yield json.loads(s)
                except Exception:
                    continue