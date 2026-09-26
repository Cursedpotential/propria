# Byline: Claude Code · Fable 5 · 2026-07-24
# Ingest v0.2: real-provenance ingest of a Google Timeline export file into raw.
# Handles modern format top-level arrays: semanticSegments, rawSignals, and the
# userLocationProfile object. Verbatim JSONB per record; file archived content-addressed.
# v0.2 hashing = re-serialized JSON (canonical byte-slice hashing lands in v1, ADR-0006 note).
import json, hashlib, sys, os
import psycopg

DSN = "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai"

def ingest(path, subject_name, subject_role, google_account, export_label,
           format_gen, produced_at, chunk_note):
    file_bytes = open(path, "rb").read()
    file_sha = hashlib.sha256(file_bytes).hexdigest()
    archive = rf"E:\TraceIQ_Evidence\{file_sha}.json"
    assert os.path.exists(archive), "archive copy must exist before ingest (ADR-0003)"
    doc = json.loads(file_bytes.decode("utf-8-sig"))

    sections = []
    if "semanticSegments" in doc:
        sections.append(("semanticSegments", doc["semanticSegments"]))
    if "timelineObjects" in doc:
        sections.append(("timelineObjects", doc["timelineObjects"]))
    if "rawSignals" in doc:
        sections.append(("rawSignals", doc["rawSignals"]))
    if "userLocationProfile" in doc:
        sections.append(("userLocationProfile", [doc["userLocationProfile"]]))
    if not sections:
        sys.exit(f"UNRECOGNIZED FORMAT: {list(doc.keys())[:6]}")

    with psycopg.connect(DSN) as conn, conn.cursor() as cur:
        cur.execute("""INSERT INTO raw.subject (display_name, role, notes) VALUES (%s,%s,
                       'respondent - Salem v. Kinzel; attribution per owner 2026-07-24')
                       ON CONFLICT (display_name) DO UPDATE SET role=EXCLUDED.role
                       RETURNING subject_id""", (subject_name, subject_role))
        subject_id = cur.fetchone()[0]
        cur.execute("""INSERT INTO raw.account (subject_id, google_account, label)
                       VALUES (%s,%s,'primary') ON CONFLICT (subject_id, google_account)
                       DO UPDATE SET label=EXCLUDED.label RETURNING account_id""",
                    (subject_id, google_account))
        account_id = cur.fetchone()[0]
        cur.execute("""SELECT export_id FROM raw.export WHERE account_id=%s AND label=%s""",
                    (account_id, export_label))
        row = cur.fetchone()
        if row:
            export_id = row[0]
        else:
            cur.execute("""INSERT INTO raw.export (account_id, label, format_generation,
                           produced_at, ingest_priority, notes)
                           VALUES (%s,%s,%s,%s,1,
                           'PRIORITY EVIDENCE CORPUS. acquisition_method/basis: OWNER TO FILL (admissibility record, ADR-0003 s8)')
                           RETURNING export_id""",
                        (account_id, export_label, format_gen, produced_at))
            export_id = cur.fetchone()[0]
        cur.execute("""INSERT INTO raw.chunk (export_id, file_name, file_sha256, byte_size, archive_path)
                       VALUES (%s,%s,%s,%s,%s)
                       ON CONFLICT (export_id, file_sha256) DO NOTHING
                       RETURNING chunk_id""",
                    (export_id, os.path.basename(path) + " | " + chunk_note,
                     file_sha, len(file_bytes), archive))
        row = cur.fetchone()
        if not row:
            print(f"chunk {file_sha[:12]} already ingested for this export - skipping")
            return
        chunk_id = row[0]

        n = 0
        with cur.copy("COPY raw.record (chunk_id, record_index, record_sha256, google_type, raw_json) FROM STDIN") as copy:
            for section, records in sections:
                for rec in records:
                    blob = json.dumps(rec, separators=(",", ":"), ensure_ascii=False)
                    sha = hashlib.sha256(blob.encode()).hexdigest()
                    if section in ("semanticSegments", "timelineObjects"):
                        gtype = next((k for k in ("visit","activity","timelinePath","timelineMemory",
                                                   "placeVisit","activitySegment","trip") if k in rec), None)
                        gtype = f"{section}.{gtype}" if gtype else section
                    else:
                        gtype = section
                    copy.write_row((chunk_id, n, sha, gtype, blob))
                    n += 1
        cur.execute("UPDATE raw.chunk SET parse_status='parsed' WHERE chunk_id=%s", (chunk_id,))
        conn.commit()
        print(f"chunk {chunk_id} ({file_sha[:12]}): {n} records ingested")

if __name__ == "__main__":
    for path, note in [
        (r"E:\AI_Workspace\Projects\traaceiq_mess\traceiq\20260106033151884\Timeline Tools\Source Files\Copy of K- Timeline.json",
         "variant A - K-named original"),
        (r"E:\AI_Workspace\Projects\traaceiq_mess\traceiq\20260106033151884\Timeline.json",
         "variant B - renamed family (Timeline.json/Records-1.json x4 mirrors)"),
    ]:
        ingest(path, "K (respondent)", "respondent", "k-account-TBD",
               "K Timeline export ~2024-11", "modern_semanticSegments", "2024-11-01", note)
