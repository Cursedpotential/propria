#!/usr/bin/env python3
"""List a whole R2 bucket through the casebible-r2-hasher Worker and write rclone-lsjson-shaped lines. Runs ON ovh-files.

Byline: Claude Code · Opus 5.5 · 2026-10-02.
Why: `rclone lsjson --hash` of the R2 buckets needed a HEAD per multipart object (≈8 MB of listing per 30 min); the
Worker's R2 binding lists 1000 objects per call with the ETag. Output feeds `bucket_objects_load.py r2 <bucket> <file>`.
md5 = the stored MD5 checksum, else the ETag when it is a plain MD5 (32 hex, no "-N" multipart suffix), else empty.
Token: HASHER_TOKEN from /data/consignatio/secrets/r2-hasher.env (never printed). Read-only (R2 list calls only).

Usage: r2_list_via_worker.py <bucket> <out.json>
"""
import json
import re
import sys
import time
import urllib.request

URL = "https://casebible-r2-hasher.matt-salem85.workers.dev/list"
TOKEN = re.search(r"^HASHER_TOKEN=(\S+)", open("/data/consignatio/secrets/r2-hasher.env").read(), re.M).group(1)
MD5 = re.compile(r"^[0-9a-f]{32}$")


def page(bucket, cursor):
    body = json.dumps({"bucket": bucket, "cursor": cursor}).encode()
    for attempt in range(6):
        try:
            req = urllib.request.Request(URL, data=body, headers={"Authorization": "Bearer " + TOKEN,
                                         "Content-Type": "application/json", "User-Agent": "casebible-r2-list/1"})
            return json.load(urllib.request.urlopen(req, timeout=120))
        except Exception as e:  # transient Worker/network errors: back off and retry the same cursor
            if attempt == 5:
                raise
            time.sleep(2 ** attempt)
            print(f"retry {attempt + 1}: {e}", file=sys.stderr)


def main():
    bucket, out = sys.argv[1], sys.argv[2]
    n, cursor = 0, None
    with open(out + ".part", "w", encoding="utf-8") as f:
        f.write("[\n")
        while True:
            p = page(bucket, cursor)
            for o in p["objects"]:
                md5 = o.get("md5") or (o["etag"] if MD5.match(o["etag"]) else "")
                hashes = {k: v for k, v in (("md5", md5), ("sha1", o.get("sha1") or "")) if v}
                f.write(json.dumps({"Path": o["key"], "Size": o["size"], "ModTime": o["uploaded"],
                                    "Hashes": hashes, "ETag": o["etag"]}) + ",\n")
                n += 1
            if not p.get("truncated"):
                break
            cursor = p["cursor"]
        f.write("]\n")
    import os
    os.replace(out + ".part", out)
    print(f"{bucket}: {n} objects -> {out}")


if __name__ == "__main__":
    main()
