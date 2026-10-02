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

CHUNKED (owner 2026-10-02, default on): the file is published as Chonkie Neural conversation chunks into
CaseBibleChunks20261002 (chunk_publish.py) and nothing is written to or retired from MsgEvents20260918, whose per-message
objects stay. CHUNKED=0 is the old per-message path above. MODE=probe prints the counts: files, conversations, an
estimate of chunks and embed calls (EXACT=1 runs the chunker for the exact chunk count; no embedding, no writes).

Env: BUNDLE, MODE (probe|run), plus the runner's env (TERMS, NVIDIA_API_KEY, COLLECTION, WEAVIATE_URL, PUBLISH=1).
Run inside devbox, detached, from the comm_timeline_mvp directory.
"""
from __future__ import annotations

import json
import os
import sys
import time
from pathlib import Path

import duckdb
import httpx

import elt_run as er

BUNDLE = Path(os.environ["BUNDLE"])
MODE = os.environ.get("MODE", "probe")
CHUNKED = os.environ.get("CHUNKED", "1") == "1"
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


def published_count(client: httpx.Client, vault_key: str) -> int:
    """Objects of this file already published by THIS run id (so a rerun resumes instead of re-embedding)."""
    q = {"query": "{ Aggregate { %s(where:{operator:And, operands:[{path:[\"vault_key\"],operator:Equal,valueText:%s},"
                  "{path:[\"ingest_run_id\"],operator:Equal,valueText:%s}]}) { meta { count } } } }"
                  % (er.COLL, json.dumps(vault_key), json.dumps(er.RUN_ID))}
    r = client.post(f"{er.WV}/v1/graphql", json=q, timeout=120)
    r.raise_for_status()
    return int(r.json()["data"]["Aggregate"][er.COLL][0]["meta"]["count"])


def publish_retry(client: httpx.Client, rows: list[dict]) -> int:
    """er.publish with retries: a dropped connection (Weaviate or NIM) must not end a long run (2026-09-24)."""
    for attempt in range(5):
        try:
            return er.publish(client, rows)
        except (httpx.TransportError, httpx.HTTPStatusError) as e:
            print(f"  retry {attempt + 1}/5 after {type(e).__name__}: {str(e)[:120]}", flush=True)
            time.sleep(15 * (attempt + 1))
    return er.publish(client, rows)


def main_chunked() -> int:
    import chunk_publish as cp  # noqa: PLC0415  needs the Probata chunk core on PYTHONPATH (its docstring)

    b = duckdb.connect(str(BUNDLE / "proposal.duckdb"), read_only=True)
    b.execute("set TimeZone = 'UTC'")
    files = b.execute("select vault_key, rows_out from proposed_lineage where rows_out > 0 order by rows_out").fetchall()
    convs = b.execute("select count(*) from (select distinct vault_key, coalesce(conversation_id, '') "
                      "from proposed_records)").fetchone()[0]
    msgs = sum(n for _, n in files)
    print(f"{MODE} (chunked): bundle={BUNDLE.name} files={len(files)} rows={msgs} run_id={er.RUN_ID} "
          f"collection={cp.COLLECTION}", flush=True)
    print(f"counts: {cp.estimate(convs, msgs, batch=er.BATCH)}", flush=True)
    client = httpx.Client(timeout=180)
    store = cp.make_store(client, er.WV)
    if MODE != "run":
        if os.environ.get("EXACT") == "1":
            total = 0
            for vk, _ in files:
                cur = b.execute(f"select {COLS} from proposed_records where vault_key = ?", [vk])
                cols = [d[0] for d in cur.description]
                for (v, conv), group in cp.group_rows([dict(zip(cols, x)) for x in cur.fetchall()]).items():
                    total += len(cp.chunk_group(v, conv, group, run_id=er.RUN_ID, model=er.MODEL)[1])
            print(f"exact chunks: {total}; embed calls: at least {-(-total // er.BATCH)}", flush=True)
        return 0
    embedder = cp.make_embedder(client, er.KEY, er.MODEL, er.NIM.rsplit("/embeddings", 1)[0], er.BATCH)
    t0 = time.time()
    tot = {"conversations": 0, "chunks": 0, "skipped_current": 0, "retired": 0}
    for i, (vk, n) in enumerate(files, 1):
        cur = b.execute(f"select {COLS} from proposed_records where vault_key = ?", [vk])
        cols = [d[0] for d in cur.description]
        got = cp.publish_file(store, embedder, [dict(zip(cols, x)) for x in cur.fetchall()], run_id=er.RUN_ID,
                              pace=er.PACE, batch=er.BATCH)
        for k in tot:
            tot[k] += got[k]
        print(f"[{i}/{len(files)}] rows={n} {got} key=...{vk[-70:]}", flush=True)
    print(f"PUBLISH DONE mode=run chunked {tot} embed_requests={embedder.calls} elapsed={time.time() - t0:.0f}s", flush=True)
    return 0


def main() -> int:
    if CHUNKED:
        return main_chunked()
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
        old_done = old if MODE != "run" else 0
        if MODE == "run":
            if published_count(client, vk) == n:
                pub = n  # already fully published by this run id (resume after an interruption)
            else:
                cur = b.execute(f"select {COLS} from proposed_records where vault_key = ? "
                                "order by (katrina_conf = 'strong') desc, (daughter_conf is not null) desc, sort_ts_final",
                                [vk])
                cols = [d[0] for d in cur.description]
                while batch := cur.fetchmany(er.BATCH):
                    pub += publish_retry(client, [dict(zip(cols, x)) for x in batch])
                    time.sleep(er.PACE)
            if pub != n:
                print(f"  WARNING published {pub} of {n} for {vk}; older objects kept", flush=True)
            elif old:
                old_done = retire_old(client, vk, dry=False)
        tot_pub += pub
        tot_old += old_done
        print(f"[{i}/{len(files)}] rows={n} published={pub} older_objects={old} retired={old_done if MODE == 'run' else 0} "
              f"key=...{vk[-70:]}", flush=True)
    print(f"PUBLISH DONE mode={MODE} published={tot_pub} older_retired={tot_old if MODE == 'run' else 0} "
          f"older_matching={tot_old} elapsed={time.time() - t0:.0f}s", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
