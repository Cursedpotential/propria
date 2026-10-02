#!/usr/bin/env python3
# Byline: Claude Code · Sonnet · 2026-10-02
"""Scrambled-object survey, step 4: which listed keys really exist in B2 (and still hold the scrambled bytes).

The catalog lists copies that no longer exist, so the dry-run counts only keys B2 confirms. One native
`b2_list_file_names` call (prefix = the key, one file; a Class C transaction) per key, read only, with the file
id and the SHA-1 B2 holds, so the apply can verify it moves exactly the object the survey saw.

Input  keys.txt   one B2 key per line
Output exists.tsv full_key<TAB>1|0<TAB>size<TAB>file_id<TAB>sha1   (resumable)
Runs ON ovh-files at low priority; credentials parsed from the existing env file, never printed.
"""
from __future__ import annotations

import base64
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor

ENV_FILE = "/data/consignatio/secrets/rclone-b2-intake.env"
BUCKET = "salem-data"
WORKERS = 24


def load_env() -> dict[str, str]:
    env = dict(os.environ)
    for line in open(ENV_FILE, encoding="utf-8"):
        m = re.match(r"^\s*([A-Z0-9_]+)\s*=\s*(.*?)\s*$", line)
        if m:
            env[m.group(1)] = m.group(2).strip("'\"")
    return env


class B2:
    def __init__(self, env: dict[str, str]) -> None:
        cred = f"{env['RCLONE_CONFIG_B2_ACCOUNT']}:{env['RCLONE_CONFIG_B2_KEY']}"
        req = urllib.request.Request(
            "https://api.backblazeb2.com/b2api/v3/b2_authorize_account",
            headers={"Authorization": "Basic " + base64.b64encode(cred.encode()).decode()},
        )
        auth = json.load(urllib.request.urlopen(req, timeout=60))
        self.api = auth["apiInfo"]["storageApi"]["apiUrl"]
        self.token = auth["authorizationToken"]
        self.account = auth["accountId"]
        self.bucket_id = self.call("b2_list_buckets", {"accountId": self.account, "bucketName": BUCKET})["buckets"][0]["bucketId"]

    def call(self, name: str, body: dict) -> dict:
        for attempt in range(6):
            req = urllib.request.Request(
                f"{self.api}/b2api/v3/{name}", data=json.dumps(body).encode(),
                headers={"Authorization": self.token, "Content-Type": "application/json"},
            )
            try:
                return json.load(urllib.request.urlopen(req, timeout=120))
            except urllib.error.HTTPError as e:
                if e.code in (429, 500, 503) and attempt < 5:
                    time.sleep(2**attempt)
                    continue
                raise
            except (urllib.error.URLError, TimeoutError):
                if attempt == 5:
                    raise
                time.sleep(2**attempt)
        raise RuntimeError("retries exhausted")

    def visible(self, key: str) -> dict | None:
        r = self.call("b2_list_file_names", {"bucketId": self.bucket_id, "startFileName": key, "maxFileCount": 1, "prefix": key})
        for f in r.get("files", []):
            if f["fileName"] == key and f.get("action") == "upload":
                return f
        return None


def main(keys_file: str, out: str) -> None:
    b2 = B2(load_env())
    done = set()
    if os.path.exists(out):
        done = {l.split("\t", 1)[0] for l in open(out, encoding="utf-8")}
    keys = [k.rstrip("\n") for k in open(keys_file, encoding="utf-8") if k.strip() and k.rstrip("\n") not in done]
    print(len(keys), "to check", flush=True)

    def check(key: str) -> str:
        f = b2.visible(key)
        if f is None:
            return f"{key}\t0\t-1\t\t"
        return f"{key}\t1\t{f['contentLength']}\t{f['fileId']}\t{f.get('contentSha1') or ''}"

    n = 0
    with open(out, "a", encoding="utf-8") as fh, ThreadPoolExecutor(WORKERS) as pool:
        for line in pool.map(check, keys):
            fh.write(line + "\n")
            n += 1
            if n % 1000 == 0:
                fh.flush()
                print(n, flush=True)
    print("DONE", n, flush=True)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
