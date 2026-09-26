# Byline: Claude Code · Opus 5 · 2026-07-25
# Load the three RAW provider sources into src.* verbatim. Explicit paths only — no corpus
# scanning, no duplicate variants, no SQLite/derivative artifacts (owner ruling).
#
#   python ops/load_src_caches.py --dry-run
#   python ops/load_src_caches.py --execute
import argparse, csv, glob, hashlib, json, os, sys
from datetime import datetime, timezone

import psycopg

DSN = os.environ.get("TRACEIQ_DSN_KV",
                     "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai")

CACHES = r"E:\AI_Workspace\Projects\traaceiq_mess\traceiq\caches"
GOOGLE_PLACE = os.path.join(CACHES, "place_id_db_REPAIRED.json")
RADAR_CSV    = os.path.join(CACHES, "radar_geocoding_master_good.csv")
RADAR_API_DIR = r"E:\AI_Workspace\Projects\traaceiq_mess\Repo\data\raw_api_responses"

CSV_COLS = [
    "request_lat","request_lng","response_lat","response_lng","geocode_accuracy_meters",
    "geocode_variance_flag","is_good_match","is_questionable","is_bad_match","needs_re_enrichment",
    "label","label_type","layer","top_type","types","street_number","street","city","state",
    "state_code","postal_code","formatted_address","place_label","address_label",
    "distance_from_request","timezone_id","timezone_name","timezone_code","google_place_id",
    "google_place_id_found","problematic_poi","problematic_notes","manually_verified",
    "custom_label","batch_file","batch_timestamp","api_metadata",
]


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        for blk in iter(lambda: fh.read(1 << 20), b""):
            h.update(blk)
    return h.hexdigest()


def stat(path):
    return (sha256(path), os.path.getsize(path),
            datetime.fromtimestamp(os.path.getmtime(path), timezone.utc))


def register(cur, digest, kind, path, size, mtime, rows, notes=None):
    cur.execute(
        """INSERT INTO src.source_file
           (source_sha256, kind, canonical_path, byte_size, row_count, file_mtime, notes)
           VALUES (%s,%s,%s,%s,%s,%s,%s)
           ON CONFLICT (source_sha256, kind)
           DO UPDATE SET row_count = EXCLUDED.row_count, notes = EXCLUDED.notes""",
        (digest, kind, path, size, rows, mtime, notes))


def load_google(cur, execute):
    digest, size, mtime = stat(GOOGLE_PLACE)
    with open(GOOGLE_PLACE, encoding="utf-8") as fh:
        d = json.load(fh)
    rows, empties = [], 0
    for pid, v in d.items():
        if not isinstance(v, dict):
            v = {"_scalar": v}
        fields = [v.get(k) for k in ("name", "address", "url", "types", "geometry")]
        is_empty = not any(f is not None for f in fields)
        empties += is_empty
        rows.append((digest, pid, v.get("name"), v.get("address"), v.get("url"),
                     json.dumps(v["types"]) if v.get("types") is not None else None,
                     json.dumps(v["geometry"]) if v.get("geometry") is not None else None,
                     is_empty, json.dumps(v)))
    print(f"google_place_cache : {len(rows):>6,} entries ({empties} empty)  {digest[:12]}")
    if execute:
        register(cur, digest, "google_place_cache", GOOGLE_PLACE, size, mtime, len(rows),
                 f"{empties} entries carry no fields — fillable with free Place Details calls")
        cur.executemany(
            """INSERT INTO src.google_place_cache
               (source_sha256, place_id, name, address, url, types, geometry, is_empty, raw_value)
               VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)
               ON CONFLICT (source_sha256, place_id) DO NOTHING""", rows)
    return len(rows)


def load_radar_csv(cur, execute):
    digest, size, mtime = stat(RADAR_CSV)
    rows = []
    with open(RADAR_CSV, encoding="utf-8", errors="replace", newline="") as fh:
        for i, r in enumerate(csv.DictReader(fh)):
            rows.append((digest, i, *[r.get(c) for c in CSV_COLS]))
    print(f"radar_csv          : {len(rows):>6,} rows  {digest[:12]}")
    if execute:
        register(cur, digest, "radar_csv", RADAR_CSV, size, mtime, len(rows),
                 "pre-filtered is_good_match=true, accuracy capped 100m — NOT a complete record")
        cols = ",".join(CSV_COLS)
        ph = ",".join(["%s"] * (len(CSV_COLS) + 2))
        cur.executemany(
            f"""INSERT INTO src.radar_csv (source_sha256, row_index, {cols})
                VALUES ({ph}) ON CONFLICT (source_sha256,row_index) DO NOTHING""", rows)
    return len(rows)


def load_radar_api(cur, execute):
    files = sorted(glob.glob(os.path.join(RADAR_API_DIR, "*.json")))
    total = 0
    for path in files:
        digest, size, mtime = stat(path)
        with open(path, encoding="utf-8", errors="replace") as fh:
            d = json.load(fh)
        meta = d.get("metadata") or {}
        base = os.path.basename(path)
        rows = []
        for i, e in enumerate(d.get("responses") or []):
            if not isinstance(e, dict):
                continue
            c = e.get("coordinates") or {}
            rows.append((digest, base, meta.get("batch_number"), meta.get("timestamp"), i,
                         e.get("timestamp"), e.get("api_type"), c.get("lat"), c.get("lng"),
                         e.get("response_status"),
                         json.dumps(e["request_params"]) if e.get("request_params") is not None else None,
                         json.dumps(e["response_data"]) if e.get("response_data") is not None else None))
        total += len(rows)
        if execute:
            register(cur, digest, "radar_api_batch", path, size, mtime, len(rows))
            cur.executemany(
                """INSERT INTO src.radar_api_batch
                   (source_sha256, batch_file, batch_number, batch_timestamp, response_index,
                    called_at, api_type, req_lat, req_lng, response_status,
                    request_params, response_data)
                   VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
                   ON CONFLICT (source_sha256,response_index) DO NOTHING""", rows)
    print(f"radar_api_batch    : {total:>6,} responses across {len(files)} batch files")
    return total


def main():
    ap = argparse.ArgumentParser()
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--dry-run", action="store_true")
    g.add_argument("--execute", action="store_true")
    a = ap.parse_args()

    for p in (GOOGLE_PLACE, RADAR_CSV, RADAR_API_DIR):
        if not os.path.exists(p):
            sys.exit(f"missing source: {p}")

    if a.dry_run:
        n = load_google(None, False) + load_radar_csv(None, False) + load_radar_api(None, False)
        print(f"\ntotal: {n:,} rows — DRY-RUN, nothing written")
        return 0

    with psycopg.connect(DSN) as c, c.cursor() as cur:
        n = load_google(cur, True) + load_radar_csv(cur, True) + load_radar_api(cur, True)
        c.commit()
    print(f"\nloaded {n:,} rows")
    return 0


if __name__ == "__main__":
    sys.exit(main())
