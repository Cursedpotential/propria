# Byline: OpenCode · ui-scaffold · 2026-07-24
# Whole-JSON ingest proof: stream-parse each source archive and compare against
# raw.record counts. Read-only against DB. Prints table to stdout and writes
# ops/INGEST_VALIDATION_REPORT.md.
import hashlib, os, sys, json
import psycopg, ijson

DSN = "host=100.119.96.29 port=5432 dbname=traceiq user=ai password=ai"
REPORT = os.path.join(os.path.dirname(__file__), "INGEST_VALIDATION_REPORT.md")

SECTION_KEYS = [
    ("semanticSegments", "array"),
    ("timelineObjects", "array"),
    ("rawSignals", "array"),
    ("userLocationProfile", "object"),
]

GOOGLE_TYPES = [
    "semanticSegments.visit",
    "semanticSegments.activity",
    "semanticSegments.timelinePath",
    "semanticSegments.timelineMemory",
    "timelineObjects",
    "rawSignals",
    "userLocationProfile",
]


def classify_record(section, rec):
    if section in ("semanticSegments", "timelineObjects"):
        key = next(
            (
                k
                for k in (
                    "visit",
                    "activity",
                    "timelinePath",
                    "timelineMemory",
                    "placeVisit",
                    "activitySegment",
                    "trip",
                )
                if k in rec
            ),
            None,
        )
        return f"{section}.{key}" if key else section
    return section


def _iter_items(f, section, wrapper):
    """Yield (section, item) tuples from an ijson stream."""
    f.seek(0)
    if section == "userLocationProfile":
        item = next(ijson.items(f, section), None)
        if item is not None:
            yield (section, item)
        return
    if wrapper == "array":
        for item in ijson.items(f, f"{section}.item"):
            yield (section, item)
        return
    for item in ijson.items(f, f"{section}.*"):
        yield (section, item)


def stream_counts(path):
    """Return dict of google_type -> count by streaming the file."""
    counts = {}
    with open(path, "rb") as f:
        for section, wrapper in SECTION_KEYS:
            try:
                for sec, item in _iter_items(f, section, wrapper):
                    gtype = classify_record(sec, item)
                    counts[gtype] = counts.get(gtype, 0) + 1
            except Exception:
                pass
    return counts


def stream_total(path):
    """Return total record count across all recognized sections."""
    total = 0
    with open(path, "rb") as f:
        for section, wrapper in SECTION_KEYS:
            try:
                total += sum(1 for _ in _iter_items(f, section, wrapper))
            except Exception:
                pass
    return total


def file_sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        while True:
            chunk = f.read(8192 * 1024)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()


def main():
    with psycopg.connect(DSN) as conn, conn.cursor() as cur:
        cur.execute("""
            SELECT chunk_id, file_name, file_sha256, byte_size, archive_path, export_id
            FROM raw.chunk ORDER BY chunk_id
        """)
        chunks = cur.fetchall()

        # Latest build reconciliation
        cur.execute("""
            SELECT build_id, raw_count, derived_count, exception_count,
                   raw_count - derived_count - exception_count AS unaccounted
            FROM working.build ORDER BY build_id DESC LIMIT 1
        """)
        build = cur.fetchone()

        cur.execute("""
            SELECT chunk_id, google_type, count(*)
            FROM raw.record
            GROUP BY chunk_id, google_type
            ORDER BY chunk_id, google_type
        """)
        db_counts = {}
        for chunk_id, gtype, cnt in cur.fetchall():
            db_counts.setdefault(chunk_id, {})[gtype] = cnt

    lines = ["# Ingest Validation Report", "", f"Generated: 2026-07-24", ""]
    lines += [
        "## Per-chunk ingest proof",
        "",
        "| chunk | source file | size | source records | DB records | match | hash |",
    ]
    lines += ["|---|---|---:|---:|:---:|---:|"]

    overall_ok = True
    for (
        chunk_id,
        file_name,
        db_file_sha256,
        byte_size,
        archive_path,
        export_id,
    ) in chunks:
        if not os.path.exists(archive_path):
            overall_ok = False
            row = f"| {chunk_id} | `{os.path.basename(file_name)}` | {byte_size} | FILE MISSING | - | ✗ | ✗ |"
            lines.append(row)
            print(row)
            continue

        actual_sha = file_sha256(archive_path)
        hash_ok = actual_sha == db_file_sha256
        src_total = stream_total(archive_path)
        src_by_type = stream_counts(archive_path)
        db_by_type = db_counts.get(chunk_id, {})
        db_total = sum(db_by_type.values())
        match_ok = src_total == db_total and all(
            src_by_type.get(t, 0) == db_by_type.get(t, 0) for t in GOOGLE_TYPES
        )
        if not hash_ok or not match_ok:
            overall_ok = False

        display = os.path.basename(file_name)
        lines.append(
            f"| {chunk_id} | `{display}` | {byte_size} | {src_total} | {db_total} | "
            f"{'✓' if match_ok else '✗'} | {'✓' if hash_ok else '✗'} |"
        )
        print(f"chunk {chunk_id}: {display}")
        print(f"  file sha256: {actual_sha}")
        print(f"  db sha256:   {db_file_sha256} -> {'OK' if hash_ok else 'FAIL'}")
        print(
            f"  source records: {src_total}; db records: {db_total} -> {'OK' if match_ok else 'FAIL'}"
        )
        for t in GOOGLE_TYPES:
            s = src_by_type.get(t, 0)
            d = db_by_type.get(t, 0)
            if s or d:
                ok = "✓" if s == d else "✗"
                print(f"    {t}: source={s} db={d} {ok}")

        lines.append("")
        lines.append("| type | source count | DB count | match |")
        lines.append("|---|---|---:|:---:|")
        for t in GOOGLE_TYPES:
            s = src_by_type.get(t, 0)
            d = db_by_type.get(t, 0)
            if s or d:
                lines.append(f"| {t} | {s} | {d} | {'✓' if s == d else '✗'} |")
        lines.append("")

    if build:
        b_id, raw_count, derived_count, exception_count, unaccounted = build
        lines += ["## Working-layer reconciliation", ""]
        lines += [f"- build_id: {b_id}"]
        lines += [f"- raw_count: {raw_count}"]
        lines += [f"- derived_count: {derived_count}"]
        lines += [f"- exception_count: {exception_count}"]
        lines += [
            f"- unaccounted: {unaccounted} ({'✓ PASS' if unaccounted == 0 else '✗ FAIL'})"
        ]
        if unaccounted != 0:
            overall_ok = False
        lines.append("")

    verdict = "PASS" if overall_ok else "FAIL"
    lines += [f"## Summary: {verdict}", ""]
    print(f"\nSUMMARY: {verdict}")

    with open(REPORT, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")
    print(f"report written: {REPORT}")
    sys.exit(0 if overall_ok else 1)


if __name__ == "__main__":
    main()
