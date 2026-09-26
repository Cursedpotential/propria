# Byline: Claude Code · Fable 5 · 2026-07-24
# Smoke-test load of the raw layer (owner-authorized). Registers a clearly-labeled TEST
# subject/account/export, ingests one corpus chunk verbatim into raw.record.
# NOTE: test uses re-serialized-JSON hashing; Phase C proper ingest defines canonical
# byte-slice hashing per ADR-0006 risk note.
import json, hashlib, sys, io
import psycopg

CHUNK = sys.argv[1]
DSN = "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai"

with io.open(CHUNK, "r", encoding="utf-8", errors="strict") as f:
    doc = json.load(f)

if "semanticSegments" in doc:
    gen, records = "modern_semanticSegments", doc["semanticSegments"]
elif "timelineObjects" in doc:
    gen, records = "legacy_timelineObjects", doc["timelineObjects"]
else:
    # fail-loud per ADR-0003 §3
    sys.exit(f"UNRECOGNIZED FORMAT: top-level keys {list(doc.keys())[:5]}")

file_bytes = open(CHUNK, "rb").read()
file_sha = hashlib.sha256(file_bytes).hexdigest()

with psycopg.connect(DSN) as conn, conn.cursor() as cur:
    cur.execute("""INSERT INTO raw.subject (display_name, role, notes)
                   VALUES ('TEST', 'other', 'smoke-test subject - not case data attribution')
                   ON CONFLICT (display_name) DO UPDATE SET notes=EXCLUDED.notes
                   RETURNING subject_id""")
    subject_id = cur.fetchone()[0]
    cur.execute("""INSERT INTO raw.account (subject_id, google_account, label)
                   VALUES (%s, 'test@smoke.local', 'test account')
                   ON CONFLICT (subject_id, google_account) DO UPDATE SET label=EXCLUDED.label
                   RETURNING account_id""", (subject_id,))
    account_id = cur.fetchone()[0]
    cur.execute("""INSERT INTO raw.export (account_id, label, format_generation, notes)
                   VALUES (%s, 'test-load-2026-07-24', %s, 'smoke test - rebuild freely')
                   RETURNING export_id""", (account_id, gen))
    export_id = cur.fetchone()[0]
    cur.execute("""INSERT INTO raw.chunk (export_id, file_name, file_sha256, byte_size, archive_path)
                   VALUES (%s, %s, %s, %s, %s) RETURNING chunk_id""",
                (export_id, CHUNK.rsplit("\\", 1)[-1], file_sha, len(file_bytes), CHUNK))
    chunk_id = cur.fetchone()[0]

    with cur.copy("COPY raw.record (chunk_id, record_index, record_sha256, google_type, raw_json) FROM STDIN") as copy:
        for i, rec in enumerate(records):
            blob = json.dumps(rec, separators=(",", ":"), ensure_ascii=False)
            sha = hashlib.sha256(blob.encode()).hexdigest()
            gtype = next((k for k in ("visit", "activity", "timelinePath", "timelinePathPoint",
                                       "placeVisit", "activitySegment", "trip") if k in rec), None)
            copy.write_row((chunk_id, i, sha, gtype, blob))

    cur.execute("UPDATE raw.chunk SET parse_status='parsed' WHERE chunk_id=%s", (chunk_id,))
    conn.commit()

    cur.execute("""SELECT count(*), count(DISTINCT record_sha256),
                          count(*) FILTER (WHERE google_type IS NULL)
                   FROM raw.record WHERE chunk_id=%s""", (chunk_id,))
    total, uniq, untyped = cur.fetchone()
    cur.execute("SELECT google_type, count(*) FROM raw.record WHERE chunk_id=%s GROUP BY 1 ORDER BY 2 DESC", (chunk_id,))
    types = cur.fetchall()
    print(f"format: {gen}")
    print(f"chunk {chunk_id}: {total} records, {uniq} unique hashes, {untyped} untyped")
    for t, c in types:
        print(f"  {t}: {c}")
