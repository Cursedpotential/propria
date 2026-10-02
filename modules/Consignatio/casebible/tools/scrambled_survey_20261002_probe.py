#!/usr/bin/env python3
# Byline: Claude Code · Sonnet · 2026-10-02
"""Scrambled-object survey, step 2: read the first 4 KiB of every candidate and classify it.

Input  candidates.ndjson  (scrambled_survey_20261002_candidates.sql)
Output heads.ndjson       one JSON object per distinct sha1, resumable (done sha1s are skipped)

Runs ON ovh-files; credentials come from the systemd EnvironmentFile that already exists there,
parsed with a regex (never sourced), and are never printed. Read-only: one ranged GET
(native b2 download by name with a Range header, a Class B transaction) per attempt, at most MAX_TRIES keys per object.

Classification of a head (class):
  marker_ok      the head carries the format marker of its extension, or is clean text for a text type
  text           the head is clean UTF-8/UTF-16 text (no extension rule applied)
  no_marker_high the head has no format marker and entropy >= HIGH_ENTROPY: a scrambled candidate
  no_marker_low  no format marker, but low entropy (sparse binary, zeros): NOT scrambled
  unreadable     none of the tried keys could be read (missing in B2)
A candidate only becomes SCRAMBLED in the SQL step, and only when an intact twin of the same name and
size exists (marker_ok or text) while this one is no_marker_high.
"""
from __future__ import annotations

import base64
import json
import math
import os
import re
import sys
import threading
import urllib.error
import urllib.parse
import urllib.request
from collections import Counter
from concurrent.futures import ThreadPoolExecutor

ENV_FILE = "/data/consignatio/secrets/rclone-b2-intake.env"
BUCKET = "salem-data"
PREFIX = "consignatio/vault/v1/"
HEAD_BYTES = 4096
MAX_TRIES = 8
HIGH_ENTROPY = 7.5
WORKERS = 24


def load_env() -> dict[str, str]:
    env = dict(os.environ)
    for line in open(ENV_FILE, encoding="utf-8"):
        m = re.match(r"^\s*([A-Z0-9_]+)\s*=\s*(.*?)\s*$", line)
        if m:
            env[m.group(1)] = m.group(2).strip("'\"")
    return env


def entropy(b: bytes) -> float:
    if not b:
        return 0.0
    n = len(b)
    return -sum(c / n * math.log2(c / n) for c in Counter(b).values())


def printable_fraction(b: bytes) -> float:
    if not b:
        return 0.0
    return sum(1 for x in b if 32 <= x < 127 or x in (9, 10, 13) or x >= 0xC2) / len(b)


def is_text(b: bytes) -> bool:
    if b[:3] == b"\xef\xbb\xbf" or b[:2] in (b"\xff\xfe", b"\xfe\xff"):
        return True
    try:
        b.decode("utf-8")  # a truncated trailing multibyte character is tolerated below
    except UnicodeDecodeError as e:
        if e.start < len(b) - 4:
            return False
    return printable_fraction(b) >= 0.92 and b.count(0) == 0


# (name, predicate over the head bytes)
MAGICS = [
    ("jpeg", lambda b: b[:3] == b"\xff\xd8\xff"),
    ("png", lambda b: b[:8] == b"\x89PNG\r\n\x1a\n"),
    ("gif", lambda b: b[:4] == b"GIF8"),
    ("webp", lambda b: b[:4] == b"RIFF" and b[8:12] == b"WEBP"),
    ("wav_avi", lambda b: b[:4] == b"RIFF"),
    ("iso_bmff", lambda b: b[4:8] in (b"ftyp", b"moov", b"mdat", b"free", b"wide", b"skip", b"styp")),
    ("mp3", lambda b: b[:3] == b"ID3" or (len(b) > 1 and b[0] == 0xFF and (b[1] & 0xE0) == 0xE0)),
    ("pdf", lambda b: b[:5] == b"%PDF-"),
    ("zip", lambda b: b[:2] == b"PK"),
    ("gzip", lambda b: b[:2] == b"\x1f\x8b"),
    ("7z", lambda b: b[:6] == b"7z\xbc\xaf\x27\x1c"),
    ("rar", lambda b: b[:4] == b"Rar!"),
    ("xz", lambda b: b[:6] == b"\xfd7zXZ\x00"),
    ("bzip2", lambda b: b[:3] == b"BZh"),
    ("zstd", lambda b: b[:4] == b"\x28\xb5\x2f\xfd"),
    ("tar", lambda b: b[257:262] == b"ustar"),
    ("class", lambda b: b[:4] == b"\xca\xfe\xba\xbe"),
    ("sqlite", lambda b: b[:15] == b"SQLite format 3"),
    ("exe", lambda b: b[:2] == b"MZ"),
    ("elf", lambda b: b[:4] == b"\x7fELF"),
    ("ogg", lambda b: b[:4] == b"OggS"),
    ("flac", lambda b: b[:4] == b"fLaC"),
    ("mkv_webm", lambda b: b[:4] == b"\x1a\x45\xdf\xa3"),
    ("tiff", lambda b: b[:4] in (b"II*\x00", b"MM\x00*")),
    ("bmp", lambda b: b[:2] == b"BM"),
    ("ico", lambda b: b[:4] == b"\x00\x00\x01\x00"),
    ("font", lambda b: b[:4] in (b"\x00\x01\x00\x00", b"OTTO", b"wOFF", b"wOF2", b"true", b"ttcf")),
    ("ole2", lambda b: b[:8] == b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"),
    ("heif_brand", lambda b: b[4:8] == b"ftyp"),
    ("rtf", lambda b: b[:5] == b"{\\rtf"),
    ("psd", lambda b: b[:4] == b"8BPS"),
    ("arrow_parquet", lambda b: b[:4] == b"PAR1" or b[:6] == b"ARROW1"),
    ("pgp", lambda b: b[:2] in (b"\x99\x01", b"\x89\x01") or b[:10] == b"-----BEGIN"),
]
TEXT_EXT = {
    "html", "htm", "xhtml", "xml", "json", "ndjson", "jsonl", "txt", "csv", "tsv", "md", "log", "py", "js", "css", "yml", "yaml",
    "ini", "cfg", "conf", "sql", "sh", "bat", "ps1", "svg", "vcf", "ics", "eml", "srt", "vtt", "tex", "rst", "toml", "java", "go", "ts",
    "c", "h", "cpp", "properties", "gradle", "kt", "swift", "m", "r", "rb", "php", "pl", "lua", "scss", "less", "jsx", "tsx", "map",
}


def magic_of(head: bytes) -> str | None:
    for name, pred in MAGICS:
        try:
            if pred(head):
                return name
        except IndexError:
            continue
    return None


def classify(name: str, head: bytes) -> dict:
    ext = name.rsplit(".", 1)[-1].lower() if "." in name else ""
    ent = entropy(head)
    pf = printable_fraction(head)
    text = is_text(head)
    magic = magic_of(head)
    if magic is not None or (ext in TEXT_EXT and text):
        cls = "marker_ok"
    elif text:
        cls = "text"
    elif ent >= HIGH_ENTROPY:
        cls = "no_marker_high"
    else:
        cls = "no_marker_low"
    return {"ext": ext, "entropy": round(ent, 3), "printable": round(pf, 3), "magic": magic, "is_text": text, "class": cls}


lock = threading.Lock()
counters = Counter()


class B2:
    """Native B2 download by name with a Range header: one Class B transaction per attempt, no listing,
    no process start. One authorization is shared by every worker."""

    def __init__(self, env: dict[str, str]) -> None:
        cred = f"{env['RCLONE_CONFIG_B2_ACCOUNT']}:{env['RCLONE_CONFIG_B2_KEY']}"
        req = urllib.request.Request(
            "https://api.backblazeb2.com/b2api/v3/b2_authorize_account",
            headers={"Authorization": "Basic " + base64.b64encode(cred.encode()).decode()},
        )
        auth = json.load(urllib.request.urlopen(req, timeout=60))
        self.download = auth["apiInfo"]["storageApi"]["downloadUrl"]
        self.token = auth["authorizationToken"]

    def head(self, key: str) -> bytes | None:
        url = f"{self.download}/file/{BUCKET}/{urllib.parse.quote(PREFIX + key, safe='/')}"
        req = urllib.request.Request(url, headers={"Authorization": self.token, "Range": f"bytes=0-{HEAD_BYTES - 1}"})
        for attempt in range(4):
            try:
                with urllib.request.urlopen(req, timeout=60) as resp:
                    return resp.read(HEAD_BYTES)
            except urllib.error.HTTPError as e:
                if e.code in (404, 400):
                    return None
                if e.code in (429, 500, 503) and attempt < 3:
                    import time

                    time.sleep(2**attempt)
                    continue
                return None
            except (urllib.error.URLError, TimeoutError):
                if attempt == 3:
                    return None
        return None


def probe(item: dict, b2: B2) -> dict:
    result = {"sha1": item["sha1"], "size": item["size"], "name": item["name"], "copies": item["copies"], "in_nxplel": item["in_nxplel"], "in_multi": item["in_multi"]}
    for i, key in enumerate(item["keys"][:MAX_TRIES]):
        head = b2.head(key)
        with lock:
            counters["class_b_attempts"] += 1
        if head:
            result.update(classify(item["name"], head))
            result["key_used"] = key
            result["head_len"] = len(head)
            result["tries"] = i + 1
            return result
    result.update({"class": "unreadable", "tries": min(MAX_TRIES, len(item["keys"])), "key_used": None})
    return result


def main(candidates: str, out: str) -> None:
    env = load_env()
    b2 = B2(env)
    done = set()
    if os.path.exists(out):
        for line in open(out, encoding="utf-8"):
            done.add(json.loads(line)["sha1"])
    todo = [json.loads(l) for l in open(candidates, encoding="utf-8") if l.strip()]
    todo = [t for t in todo if t["sha1"] not in done]
    print(f"{len(todo)} to probe, {len(done)} already done", flush=True)
    written = 0
    with open(out, "a", encoding="utf-8") as fh, ThreadPoolExecutor(WORKERS) as pool:
        for res in pool.map(lambda it: probe(it, b2), todo):
            fh.write(json.dumps(res, ensure_ascii=False) + "\n")
            written += 1
            counters[res["class"]] += 1
            if written % 500 == 0:
                fh.flush()
                print(written, dict(counters), flush=True)
    print("DONE", written, dict(counters), flush=True)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
