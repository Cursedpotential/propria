# Byline: Claude Code · Opus 5 (1M context) · 2026-09-18
"""Chat ELT runner — DuckDB templates over the B2 mount, straight into Weaviate.

Owner direction 2026-09-18 20:52 / 21:05 EDT: extraction is DuckDB ELT (the versioned templates in
elt/), the searchable set is Weaviate `MsgEvents20260918` (messages with people; was ChatEvents20260918, split 2026-09-24) (no Postgres event tables), and the work list
is the chat candidates that the afternoon run did not already load. Type is confirmed by READING the
file (first bytes), not by its folder name.

Per file: sanitize if the format needs it -> template SQL -> event contract -> tag_events_v1.sql (person
tags + the same dedup key as the afternoon build) -> embed (NIM, batched) -> upsert into Weaviate with
uuid5(dedup_key), so an ELT event REPLACES its parser-derived twin instead of duplicating it.

State: /work/elt_state.duckdb (processed files). Progress: one line per file on stdout.
Evidence is read-only; the only writes are Weaviate objects, the state DB and the scratch sanitize file.

2026-09-24 changes (Claude Code · Opus 5.5; bugs found while extracting the owner's 2023-24 side, owner 09:21 "follow
the packing and extraction guidance ... fix any bugs that's part of this process so it works later"):
  * Readers v2 + tag_events_v2 + norm_phone_v1 (one phone-number rule, same as the catalog's raw_duck.norm_phone).
  * dedup_key is now the TRUE-DUPLICATE key (owner 09:30-09:36: same device, same format, same user, same platform).
    Copies from a different device/format/person/platform are corroboration: their own rows and Weaviate objects,
    never an overwrite. The worklist may carry `custodian` (whose device/account: Matt | Katrina) and `source_device`
    (a confirmed device id); without a confirmed device a file's rows get 'unconfirmed:<sha1>' and merge with nothing.
  * Per-attempt proposal bundle (docs/receipts/PIPELINE-HISTORY-2026-09-18.md stage 3): every extracted row of every
    file goes to WORK/proposal/<RUN_ID>/rows/<sha1>.parquet; at the end proposal.duckdb (proposed_records,
    proposed_lineage, proposed_warnings) and manifest.json (row counts, row-set digests, byte length + SHA-256 of
    every artifact). Gates per file: event-contract columns present; rows read back from the parquet equal the rows
    extracted (count + row digest); source markers vs rows recorded as a warning when they differ.
  * Weaviate is written only with PUBLISH=1 (precommit contract: nothing reaches a store before review). The bundle
    is the review artifact and the catalog load source (msg_extract_load_20260924.sh).
  * The runner no longer lists a Cube ACR reader: elt_cube_acr_json_v1.sql never existed.
  * New readers elt_whatsapp_txt_v1 and elt_sms_csv_v1 (no reader existed; owner 09:47 "no tool -> add one").
  * A file where more than 10% of rows have no usable date is reported as suspect_integrity (repair toolkit).
"""
from __future__ import annotations

import csv
import hashlib
import json
import os
import re
import sys
import time
import traceback
import uuid
from pathlib import Path

import duckdb
import httpx

B2 = Path(os.environ.get("B2_ROOT", "/b2")) / "salem-data"
WORK = Path(os.environ.get("WORK", "/work"))
TMP = Path(os.environ.get("TMPD", "/tmpd"))
ELT = Path(os.environ.get("ELT_DIR", "/app/elt"))
RUN_ID = os.environ.get("RUN_ID", time.strftime("%Y%m%dT%H%M%S"))
WV = os.environ.get("WEAVIATE_URL", "http://100.91.190.107:8082").rstrip("/")
COLL = os.environ.get("COLLECTION", "MsgEvents20260918")
MODEL = os.environ.get("NIM_EMBED_MODEL", "nvidia/nemotron-3-embed-1b")
DIM = int(os.environ.get("NIM_EMBED_DIMENSIONS", "2048"))
NIM = os.environ.get("NIM_BASE_URL", "https://integrate.api.nvidia.com/v1").rstrip("/") + "/embeddings"
PUBLISH = os.environ.get("PUBLISH", "0") == "1"
KEY = os.environ.get("NVIDIA_API_KEY", "")
if PUBLISH and not KEY:
    sys.exit("PUBLISH=1 needs NVIDIA_API_KEY")
PROPOSAL = WORK / "proposal" / RUN_ID
BATCH = int(os.environ.get("BATCH", "32"))
PACE = float(os.environ.get("PACE", "0.4"))  # the AI-chat run owns the embedder tonight; stay behind it
LIMIT = int(os.environ.get("LIMIT", "0"))
ONLY = {f for f in os.environ.get("FORMATS", "").split(",") if f}
NS = uuid.UUID("6f1d3a52-1c7e-4b8e-9a51-2d0e3c9b7a18")  # same namespace as the afternoon embed run
TERMS = json.load(open(os.environ["TERMS"]))
# Her confirmed numbers come from the catalog (raw_duck.msg_identity_20260924, exported by
# msg_identity_export_20260924.sql) when that export is given: the terms file's copy was stale on 2026-09-24 (it lacked
# 810-268-9630, her number to 2024). Without the export the terms file's list is used, as before.
IDENTITY = os.environ.get("IDENTITY", "")
if IDENTITY and Path(IDENTITY).exists():
    KATRINA_PHONES = sorted({r["identifier"] for r in csv.DictReader(open(IDENTITY, encoding="utf-8"), delimiter="\t")
                             if r["person"] == "Katrina" and r["kind"] == "phone" and r["status"] == "confirmed"})
else:
    KATRINA_PHONES = TERMS["katrina_phones_confirmed"]

# format -> (template file, blocks, needs_xml_sanitize)
TEMPLATES = {
    "sms_backup_xml": ("elt_smsbackuprestore_v2.sql", ["sms", "mms", "call"], True),
    "calls_backup_xml": ("elt_smsbackuprestore_v2.sql", ["sms", "mms", "call"], True),
    "google_voice_html": ("elt_google_voice_html_v2.sql", ["text", "call"], False),
    "fb_messenger_html": ("elt_fb_messenger_html_v1.sql", ["*"], False),
    "imessage_html": ("elt_imessage_html_v3.sql", ["*"], False),  # v3: the export's clock is UTC (cross-checked)
    "imessage_txt": ("elt_imessage_txt_v2.sql", ["*"], False),
    "mbox": ("elt_mbox_v1.sql", ["*"], False),
    "whatsapp_txt": ("elt_whatsapp_txt_v1.sql", ["*"], False),
    "sms_csv": ("elt_sms_csv_v1.sql", ["*"], False),
}
TAGGER = "tag_events_v2.sql"
# The medium a format records. Part of the true-duplicate key (owner 09:36: "same medium or platform").
PLATFORM = {"sms_backup_xml": "carrier_sms_mms", "calls_backup_xml": "carrier_calls", "google_voice_html": "google_voice",
            "fb_messenger_html": "facebook_messenger", "imessage_html": "apple_messages",
            "imessage_txt": "apple_messages", "mbox": "email", "whatsapp_txt": "whatsapp",
            "sms_csv": "carrier_sms_mms"}
# Source-side markers, counted straight from the file, to reconcile against the rows a reader returns.
MARKERS = {"google_voice_html": r'<div class="message">', "imessage_html": r"""<div class=['"]bubble""",
           "imessage_txt": r"(?m)^\[[0-9]{4}-[0-9]{2}-[0-9]{2}",
           "whatsapp_txt": r"(?m)^\x{200E}?\[?[0-9]{1,2}/[0-9]{1,2}/[0-9]{2,4}, [0-9]{1,2}:[0-9]{2}"}
EVENT_COLS = ["record_index", "event_ts_utc", "sort_ts", "ts_original", "ts_field", "tz_status", "event_kind",
              "conversation_id", "conversation_title", "participants", "sender", "recipients", "direction",
              "counterparty_phone", "contact_name", "body", "attachments", "member_path"]
NEW_PROPS = [("recipients", "text[]"), ("conversation_id", "text"), ("sha1", "text"), ("catalog_path", "text"),
             ("extractor", "text"), ("ingest_run_id", "text"), ("indexed_at", "date"), ("event_kind", "text"),
             ("direction", "text"), ("counterparty_phone", "text"), ("contact_name", "text"),
             ("search_text", "text"), ("attachments", "text"), ("provenance", "text[]"),
             ("record_kind", "text"), ("content_key", "text"), ("custodian", "text"), ("source_device", "text"),
             ("platform", "text"), ("owner_line", "text")]


def sniff(path: Path, guess: str) -> str | None:
    """Confirm the format by reading the file, not by its name. Returns the format to use, or None to skip."""
    try:
        with open(path, "rb") as fh:
            head = fh.read(65536)
    except OSError as e:
        raise RuntimeError(f"unreadable: {e}")
    t = head.decode("utf-8", "replace")
    low = t.lower()
    if "<smses" in low or "<sms " in low or "<mms " in low:
        return "sms_backup_xml"
    if "<calls" in low or "<call " in low:
        return "calls_backup_xml"
    if "hchatlog" in low or ('class="tel"' in low and "<q>" in low):
        return "google_voice_html"
    # Google Voice call / voicemail / missed-call pages carry a.tel + abbr.published but no chat log (2026-09-24:
    # v1 skipped every one of them as "not a supported format").
    if 'class="tel"' in low and 'class="published"' in low:
        return "google_voice_html"
    if "_a6-g" in t or "_a70e" in t:
        return "fb_messenger_html"
    if "bubble from-me" in t or "bubble from-them" in t:
        return "imessage_html"
    if re.search(r"^\[\d{4}-\d{2}-\d{2} [^\]]*\] [^\n]{1,80}:\s*$", t, re.M):
        return "imessage_txt"
    if t.startswith("From ") and re.search(r"(?im)^Date: ", t):
        return "mbox"
    # 2026-09-24: WhatsApp exports and phone SMS CSV exports had no reader and were skipped.
    tw = t.replace("\u200e", "")
    if re.search(r"^\[\d{1,2}/\d{1,2}/\d{2,4}, \d{1,2}:\d{2}[^\]]*\] [^:\n]{1,80}: ", tw, re.M) or \
            re.search(r"^\d{1,2}/\d{1,2}/\d{2,4}, \d{1,2}:\d{2}[^-\n]{0,8} - [^:\n]{1,80}: ", tw, re.M):
        return "whatsapp_txt"
    first = low.lstrip("\ufeff").split("\n", 1)[0].replace('"', "").replace(" ", "")
    if first.startswith("address,readable_date,type,body"):
        return "sms_csv"
    if guess == "cube_acr_json" and t.lstrip().startswith(("{", "[")):
        return "cube_acr_json"
    return None


def load_blocks(name: str) -> dict[str, str]:
    raw = (ELT / name).read_text(encoding="utf-8")
    parts = re.split(r"^-- @block (\w+)\s*$", raw, flags=re.M)
    if len(parts) == 1:
        return {"*": raw.strip().rstrip(";")}
    return {parts[i]: parts[i + 1].strip().rstrip(";") for i in range(1, len(parts), 2)}


def connect() -> duckdb.DuckDBPyConnection:
    con = duckdb.connect()
    con.execute("install webbed from community; load webbed; install zipfs from community; load zipfs")
    con.execute(f"set memory_limit='{os.environ.get('DUCK_MEM', '6GB')}'; set temp_directory='{TMP}'; set threads=4")
    # 2026-09-24: pin the session zone. Unpinned, sort_ts_final (= event_ts_utc::timestamp for zoned sources) was the
    # host's wall clock: UTC in the 09-18 container, US Eastern in devbox.
    con.execute("set TimeZone = 'UTC'")
    for k in ("katrina_strict", "katrina_possible", "catrina_c", "landlord_context", "nickname",
              "daughter_strong", "daughter_weak", "kinship", "custody", "housing"):
        con.execute(f"set variable {k} = ?", [TERMS[k]])
    con.execute("set variable katrina_phones_confirmed = ?", [KATRINA_PHONES])
    con.execute((ELT / "norm_phone_v1.sql").read_text(encoding="utf-8"))
    return con


def embed(client: httpx.Client, texts: list[str]) -> list[list[float]]:
    for attempt in range(5):
        r = client.post(NIM, headers={"Authorization": f"Bearer {KEY}"}, timeout=180,
                        json={"model": MODEL, "input": texts, "input_type": "passage",
                              "encoding_format": "float", "truncate": "END"})
        if r.status_code == 429 or r.status_code >= 500:
            time.sleep(5 * (attempt + 1))
            continue
        r.raise_for_status()
        vs = [d["embedding"] for d in sorted(r.json()["data"], key=lambda d: d["index"])]
        assert len(vs) == len(texts) and all(len(v) == DIM for v in vs)
        return vs
    r.raise_for_status()
    raise RuntimeError("embed failed")


def ensure_schema(client: httpx.Client) -> None:
    r = client.get(f"{WV}/v1/schema/{COLL}")
    r.raise_for_status()
    have = {p["name"] for p in r.json().get("properties", [])}
    for name, dt in NEW_PROPS:
        if name not in have:
            resp = client.post(f"{WV}/v1/schema/{COLL}/properties", json={"name": name, "dataType": [dt]})
            print(f"add property {name}: {resp.status_code}", flush=True)


def publish(client: httpx.Client, rows: list[dict]) -> int:
    """Embed + upsert one batch. Katrina/daughter rows are ordered first by the caller."""
    texts = []
    for r in rows:
        t = (r["body"] or "").strip()
        if not t:
            t = " ".join(x for x in [r.get("conversation_title"), r.get("event_kind"), r.get("direction"),
                                     ("attachments: " + r["attachments"]) if r.get("attachments") else None] if x)
        texts.append(t[:8000] or "(empty)")
    vecs = embed(client, texts)
    objs = []
    for r, v, t in zip(rows, vecs, texts):
        props = {
            "dedup_key": r["dedup_key"], "body": r["body"], "search_text": t, "sender": r["sender"],
            "conversation_title": r["conversation_title"], "conversation_id": r["conversation_id"],
            "source_format": r["source_format"], "event_kind": r["event_kind"], "direction": r["direction"],
            "counterparty_phone": r["counterparty_phone"], "contact_name": r["contact_name"],
            "katrina_ref_type": r["katrina_ref_type"], "katrina_conf": r["katrina_conf"],
            "daughter_conf": r["daughter_conf"], "catrina_class": r["catrina_class"], "tz_status": r["tz_status"],
            "ts_original": r["ts_original"], "vault_key": r["vault_key"], "sha1": r["sha1"],
            "catalog_path": r["catalog_rel"], "extractor": r["extractor"], "ingest_run_id": RUN_ID,
            "indexed_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "embed_model": MODEL,
            # owner glossary 21:08: message transcripts, not AI chats. 2026-09-24: calls were also labelled "message";
            # MsgEvents20260918 holds messages AND calls with people, told apart by record_kind (owner 09:30).
            "record_kind": "call" if r["event_kind"] == "call" else "message",
            "attachments": r["attachments"], "n_sources": 1,
            "provenance": [f"{r['vault_key']}|{r['member_path'] or ''}|{r['record_index']}|{r['extractor']}"],
            "content_key": r["content_key"], "custodian": r["custodian"], "source_device": r["source_device"],
            "platform": r["platform"], "owner_line": r.get("owner_line"),
        }
        if r["participants"]:
            props["participants"] = [p for p in r["participants"] if p]
        if r["recipients"]:
            props["recipients"] = [p for p in r["recipients"] if p]
        if r["sort_ts_final"]:
            props["sort_ts"] = r["sort_ts_final"].isoformat() + "Z"
        objs.append({"class": COLL, "id": str(uuid.uuid5(NS, r["dedup_key"])),
                     "properties": {k: v for k, v in props.items() if v is not None},
                     "vectors": {"text_nim": v}})
    resp = client.post(f"{WV}/v1/batch/objects", json={"objects": objs}, timeout=180)
    resp.raise_for_status()
    errs = [o["result"]["errors"] for o in resp.json() if o.get("result", {}).get("errors")]
    if errs:
        print("  weaviate batch errors:", str(errs[0])[:300], flush=True)
    return len(objs) - len(errs)


def q(v) -> str:
    """SQL string literal (or NULL)."""
    return "NULL" if v in (None, "") else "'" + str(v).replace("'", "''") + "'"


def extract_file(con: duckdb.DuckDBPyConnection, path: Path, fmt: str, row: dict) -> dict:
    """Extract one file into ev_tagged and write its rows to the attempt bundle. Returns the lineage record."""
    tpl, blocks, needs_sanitize = TEMPLATES[fmt]
    src = str(path)
    markers = None
    if fmt in MARKERS:
        markers = con.execute("select len(regexp_extract_all(content, ?)) from read_text(?)",
                              [MARKERS[fmt], src]).fetchone()[0]
        if fmt == "google_voice_html" and markers == 0:
            markers = 1  # a call / voicemail page is one event with no message divs
    if needs_sanitize:
        dst = str(TMP / "sanitized.xml")
        sql = (ELT / "elt_xml_sanitize_v1.sql").read_text(encoding="utf-8")
        con.execute(sql.replace("{{SRC}}", src.replace("'", "''")).replace("{{DST}}", dst))
        src = dst
    avail = load_blocks(tpl)
    con.execute("drop table if exists ev_in")
    device = row.get("source_device") or f"unconfirmed:{row['sha1']}"
    first = True
    for name in (blocks if blocks != ["*"] else list(avail)):
        if name not in avail:
            continue
        body = avail[name].replace("{{SRC}}", src.replace("'", "''"))
        select = f"""select * exclude (record_index), record_index,
                       {q(row['vault_key'])} as vault_key, {q(row['sha1'])} as sha1,
                       {q(row.get('catalog_rel_example'))} as catalog_rel,
                       '{fmt}' as source_format, '{tpl[:-4]}' as extractor, {q(name)}::varchar as block,
                       {q(row.get('custodian'))}::varchar as custodian, {q(device)}::varchar as source_device,
                       {q(PLATFORM.get(fmt))}::varchar as platform
                     from ({body})"""
        try:
            con.execute(f"{'create table ev_in as' if first else 'insert into ev_in by name'} {select}")
            first = False
        except duckdb.Error as e:
            if "does not contain" in str(e) or "no records" in str(e).lower():
                continue
            raise
    lin = {"vault_key": row["vault_key"], "sha1": row["sha1"], "source_format": fmt, "extractor": tpl[:-4],
           "custodian": row.get("custodian"), "source_device": device, "also_at": row.get("also_at") or "",
           "source_markers": markers, "rows_out": 0, "row_digest": None, "bundle_file": None}
    if first:
        return lin
    # Gate 1: the reader returned the whole event contract.
    have = {r[0] for r in con.execute("describe ev_in").fetchall()}
    missing = [c for c in EVENT_COLS if c not in have]
    if missing:
        raise RuntimeError(f"event contract columns missing from {tpl}: {missing}")
    con.execute((ELT / TAGGER).read_text(encoding="utf-8"))
    digest_sql = ("select count(*), md5(string_agg(md5(t::varchar), '' order by t.block, t.record_index, t.dedup_key)) "
                  "from {src} t")
    n, digest = con.execute(digest_sql.format(src="ev_tagged")).fetchone()
    out = PROPOSAL / "rows" / f"{row['sha1']}.parquet"
    out.parent.mkdir(parents=True, exist_ok=True)
    con.execute(f"copy (select * from ev_tagged) to {q(str(out))} (format parquet)")
    # Gate 4: the bundle holds exactly what was extracted (count + row digest read back from the file).
    n2, digest2 = con.execute(digest_sql.format(src=f"read_parquet({q(str(out))})")).fetchone()
    if (n, digest) != (n2, digest2):
        raise RuntimeError(f"bundle read-back mismatch: extracted {n}/{digest}, parquet {n2}/{digest2}")
    lin.update(rows_out=n, row_digest=digest, bundle_file=str(out.relative_to(PROPOSAL)),
               unparsed=con.execute("select count(*) filter (where tz_status in ('unparsed', 'missing')) "
                                    "from ev_tagged").fetchone()[0])
    return lin


def sha256_file(p: Path) -> tuple[int, str]:
    h = hashlib.sha256()
    with open(p, "rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            h.update(chunk)
    return p.stat().st_size, h.hexdigest()


def finalize_bundle(lineage: list[dict], warnings: list[dict]) -> None:
    """proposal.duckdb + manifest.json for this attempt (PIPELINE-HISTORY stage 3, cheap version)."""
    db_path = PROPOSAL / "proposal.duckdb"
    b = duckdb.connect(str(db_path))
    rows_glob = str(PROPOSAL / "rows" / "*.parquet")
    if any((PROPOSAL / "rows").glob("*.parquet")):
        b.execute(f"create or replace table proposed_records as select * from read_parquet({q(rows_glob)}, union_by_name = true)")
    else:
        b.execute("create or replace table proposed_records (dedup_key varchar)")
    b.execute("create or replace table proposed_lineage (vault_key varchar, sha1 varchar, source_format varchar, "
              "extractor varchar, custodian varchar, source_device varchar, also_at varchar, source_markers bigint, "
              "rows_out bigint, row_digest varchar, bundle_file varchar)")
    for lin in lineage:
        b.execute("insert into proposed_lineage values (?,?,?,?,?,?,?,?,?,?,?)",
                  [lin[k] for k in ("vault_key", "sha1", "source_format", "extractor", "custodian", "source_device",
                                    "also_at", "source_markers", "rows_out", "row_digest", "bundle_file")])
    b.execute("create or replace table proposed_warnings (vault_key varchar, sha1 varchar, kind varchar, detail varchar)")
    for w in warnings:
        b.execute("insert into proposed_warnings values (?,?,?,?)", [w["vault_key"], w["sha1"], w["kind"], w["detail"]])
    rel = {}
    for t in ("proposed_records", "proposed_lineage", "proposed_warnings"):
        n, d = b.execute(f"select count(*), md5(coalesce(string_agg(md5(t::varchar), '' order by t::varchar), '')) "
                         f"from {t} t").fetchone()
        rel[t] = {"rows": n, "row_set_md5": d}
    b.execute("checkpoint")
    b.close()
    artifacts = {}
    for p in sorted([db_path] + list((PROPOSAL / "rows").glob("*.parquet"))):
        size, sha = sha256_file(p)
        artifacts[str(p.relative_to(PROPOSAL))] = {"bytes": size, "sha256": sha}
    manifest = {"attempt_id": RUN_ID, "created_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                "runner": "elt_run.py", "tagger": TAGGER, "number_rule": "norm_phone_v1.sql",
                "templates": sorted({lin["extractor"] for lin in lineage if lin.get("extractor")}),
                "published_to_weaviate": PUBLISH, "relations": rel, "artifacts": artifacts}
    (PROPOSAL / "manifest.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    print(f"BUNDLE {PROPOSAL} records={rel['proposed_records']['rows']} files={len(lineage)} "
          f"warnings={len(warnings)}", flush=True)


def main() -> int:
    state = duckdb.connect(str(WORK / "elt_state.duckdb"))
    state.execute("create table if not exists processed(vault_key varchar, sha1 varchar, fmt varchar, "
                  "template varchar, n_events bigint, n_published bigint, status varchar, error varchar, "
                  "seconds double, run_id varchar, done_at timestamptz)")
    done = {r[0] for r in state.execute("select sha1 from processed where status in ('ok','skip')").fetchall()}
    work = list(csv.DictReader(open(WORK / "elt_worklist.tsv", encoding="utf-8"), delimiter="\t"))
    if ONLY:
        work = [w for w in work if w["format_guess"] in ONLY]
    client = httpx.Client(timeout=180)
    t0 = time.time()
    if PUBLISH:
        ensure_schema(client)
        probe = embed(client, ["batching probe a", "batching probe b", "batching probe c", "batching probe d"])
        print(f"embed batch probe: 4 texts -> {len(probe)} vectors of {len(probe[0])} dims in one call", flush=True)
    PROPOSAL.mkdir(parents=True, exist_ok=True)
    lineage: list[dict] = []
    warnings: list[dict] = []
    n_done = n_pub = 0
    for i, row in enumerate(work, 1):
        if row["sha1"] in done:
            continue
        if LIMIT and n_done >= LIMIT:
            break
        path = B2 / row["vault_key"]
        t1 = time.time()
        status = "ok"
        err = None
        n_ev = n_ok = 0
        try:
            fmt = sniff(path, row["format_guess"])
            if fmt is None or fmt not in TEMPLATES:
                status, err = "skip", f"content not a supported chat format (guess={row['format_guess']})"
                warnings.append({"vault_key": row["vault_key"], "sha1": row["sha1"], "kind": "no_reader", "detail": err})
            else:
                con = connect()
                lin = extract_file(con, path, fmt, row)
                lineage.append(lin)
                n_ev = lin["rows_out"]
                if lin["source_markers"] is not None and lin["source_markers"] != n_ev:
                    warnings.append({"vault_key": row["vault_key"], "sha1": row["sha1"], "kind": "count_differs",
                                     "detail": f"{lin['source_markers']} source markers, {n_ev} rows"})
                if n_ev and lin.get("unparsed", 0) > 0.1 * n_ev:
                    warnings.append({"vault_key": row["vault_key"], "sha1": row["sha1"], "kind": "suspect_integrity",
                                     "detail": f"{lin['unparsed']} of {n_ev} rows have no usable date: "
                                               "check with the repair toolkit"})
                if n_ev == 0:
                    warnings.append({"vault_key": row["vault_key"], "sha1": row["sha1"], "kind": "no_rows",
                                     "detail": f"{fmt} reader returned no rows"})
                if n_ev and PUBLISH:
                    cur = con.execute(
                        "select dedup_key, content_key, record_index, event_ts_utc, sort_ts_final, ts_original, "
                        "tz_status, event_kind, conversation_id, conversation_title, participants, sender, recipients, "
                        "direction, counterparty_phone, contact_name, body, attachments, member_path, vault_key, "
                        "sha1, catalog_rel, source_format, extractor, katrina_ref_type, katrina_conf, "
                        "catrina_class, daughter_conf, custodian, source_device, platform, owner_line from ev_tagged "
                        "order by (katrina_conf = 'strong') desc, (daughter_conf is not null) desc, sort_ts_final")
                    cols = [d[0] for d in cur.description]
                    while batch := cur.fetchmany(BATCH):
                        n_ok += publish(client, [dict(zip(cols, b)) for b in batch])
                        time.sleep(PACE)
                con.close()
        except Exception as e:  # recorded per file, never silent
            status, err = "error", f"{type(e).__name__}: {e}"[:1500]
            warnings.append({"vault_key": row["vault_key"], "sha1": row["sha1"], "kind": "error", "detail": err})
            traceback.print_exc()
        dt = time.time() - t1
        state.execute("insert into processed values (?,?,?,?,?,?,?,?,?,?,now())",
                      [row["vault_key"], row["sha1"], row["format_guess"], TEMPLATES.get(row["format_guess"], ("", ))[0],
                       n_ev, n_ok, status, err, dt, RUN_ID])
        n_done += 1
        n_pub += n_ok
        print(f"[{i}/{len(work)}] {status} {row['format_guess']} events={n_ev} published={n_ok} {dt:.1f}s "
              f"size={row['size']} key=...{row['vault_key'][-70:]}", flush=True)
    finalize_bundle(lineage, warnings)
    print(f"RUN DONE run_id={RUN_ID} files={n_done} published={n_pub} elapsed={time.time() - t0:.0f}s", flush=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
