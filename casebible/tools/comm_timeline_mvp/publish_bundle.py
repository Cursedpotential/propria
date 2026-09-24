# Byline: Claude Code · Opus 5.5 · 2026-09-24
"""Publish a reviewed ELT proposal bundle to Weaviate (owner 2026-09-24 ~13:15: publish the extracted messages into
MsgEvents20260918, one object per copy).

Reads BUNDLE/proposal.duckdb (written by elt_run.py), embeds with the runner's own publish() (NIM, batched), and upserts
with uuid5(dedup_key): dedup_key is the true-duplicate key, so a copy from another device/format/person gets its own
object and nothing is overwritten across copies (owner 09:28-09:36).

Older extractions of the SAME file (same vault_key, published by an earlier run: the 09-18 discovery index or the v1
readers) are retired after the file's new objects are in, so one file is not indexed twice by two reader versions.
They are derived index objects, rebuildable from B2; the bundle and the catalog keep the record. MODE=probe only counts
what would be published and retired; MODE=run does it. Each file's retire count is printed and logged.

Env: BUNDLE, MODE (probe|run), plus the runner's env (TERMS, NVIDIA_API_KEY, COLLECTION, WEAVIATE_URL, PUBLISH=1).
Run inside devbox, detached, from the comm_timeline_mvp directory.
"""
from __future__ import annotations

import os
import sys
import time
from pathlib import Path

import duckdb
import httpx

import elt_run as er

BUNDLE = Path(os.environ["BUNDLE"])
MODE = os.environ.get("MODE", "probe")
COLS = ("dedup_key, content_key, record_index, event_ts_utc, sort_ts_final, ts_original, tz_status, event_kind, "
        "conversation_id, conversation_title, participants, sender, recipients, direction, counterparty_phone, "
        "contact_name, body, attachments, member_path, vault_key, sha1, catalog_rel, source_format, extractor, "
        "katrina_ref_type, katrina_conf, catrina_class, daughter_conf, custodian, source_device, platform, owner_line")


def retire_old(client: httpx.Client, vault_key: str, dry: bool) -> int:
    """Objects of this file from any earlier run (ingest_run_id is not this run's). Returns how many match."""
    body = {"match": {"class": er.COLL, "where": {"operator": "And", "operands": [
        {"path": ["vault_key"], "operator": "Equal", "valueText": vault_key},
        {"path": ["ingest_run_id"], "operator": "NotEqual", "valueText": er.RUN_ID}]}},
        "dryRun": dry, "output": "minimal"}
    r = client.request("DELETE", f"{er.WV}/v1/batch/objects", json=body, timeout=300)
    r.raise_for_status()
    res = r.json().get("results", {})
    if res.get("failed"):
        raise RuntimeError(f"retire failed for {vault_key}: {res}")
    return int(res.get("matches", 0))


def main() -> int:
    b = duckdb.connect(str(BUNDLE / "proposal.duckdb"), read_only=True)
    b.execute("set TimeZone = 'UTC'")
    files = b.execute("select vault_key, rows_out from proposed_lineage where rows_out > 0 order by rows_out").fetchall()
    client = httpx.Client(timeout=180)
    print(f"{MODE}: bundle={BUNDLE.name} files={len(files)} rows={sum(n for _, n in files)} run_id={er.RUN_ID} "
          f"collection={er.COLL}", flush=True)
    if MODE == "run":
        er.ensure_schema(client)
        probe = er.embed(client, ["batching probe a", "batching probe b", "batching probe c", "batching probe d"])
        print(f"embed batch probe: 4 texts -> {len(probe)} vectors of {len(probe[0])} dims", flush=True)
    t0 = time.time()
    tot_pub = tot_old = 0
    for i, (vk, n) in enumerate(files, 1):
        old = retire_old(client, vk, dry=True)
        pub = 0
        if MODE == "run":
            cur = b.execute(f"select {COLS} from proposed_records where vault_key = ? "
                            "order by (katrina_conf = 'strong') desc, (daughter_conf is not null) desc, sort_ts_final",
                            [vk])
            cols = [d[0] for d in cur.description]
            while batch := cur.fetchmany(er.BATCH):
                pub += er.publish(client, [dict(zip(cols, x)) for x in batch])
                time.sleep(er.PACE)
            if pub != n:
                print(f"  WARNING published {pub} of {n} for {vk}; older objects kept", flush=True)
                old_done = 0
            else:
                old_done = retire_old(client, vk, dry=False) if old else 0
        else:
            old_done = old
        tot_pub += pub
        tot_old += old_done
        print(f"[{i}/{len(files)}] rows={n} published={pub} older_objects={old} retired={old_done if MODE == 'run' else 0} "
              f"key=...{vk[-70:]}", flush=True)
    print(f"PUBLISH DONE mode={MODE} published={tot_pub} older_retired={tot_old if MODE == 'run' else 0} "
          f"older_matching={tot_old} elapsed={time.time() - t0:.0f}s", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
