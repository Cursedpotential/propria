#!/usr/bin/env python3
"""
cb_pilot.py - Case Bible F:\case DRY RUN
=========================================
READ-ONLY on all sources. Writes ONLY inside the pilot output dir.
Never touches PG18, R2, OneDrive, or any source file. Nothing is moved,
renamed, or deleted. Source structure is preserved verbatim in the
proposed layout - no packages are disassembled.

Phases (independently runnable, hashing is resumable):
  1 scan     walk source w/ junk pruning        -> files.jsonl
  2 hash     SHA-256 + MD5 + SHA-1, ONE read    -> hashes.jsonl (resumable)
  3 analyze  load + anti-join + folder metrics  -> pilot.duckdb
  4 report   human-readable findings            -> report.html

Usage:
  python cb_pilot.py --all
  python cb_pilot.py --phase 2            # resume an interrupted hash pass
  python cb_pilot.py --phase 3 --phase 4  # re-analyze without re-hashing
"""
import os, re, sys, json, time, hashlib, argparse
from datetime import datetime

SOURCE   = r"F:\case"
OUTDIR   = r"E:\AI_Workspace\casebible\pilot"
CORPUSDB = r"E:\AI_Workspace\casebible\casebible.duckdb"   # READ-ONLY; has r2_files.md5

FILES_JSONL  = os.path.join(OUTDIR, "files.jsonl")
HASHES_JSONL = os.path.join(OUTDIR, "hashes.jsonl")
PILOT_DB     = os.path.join(OUTDIR, "pilot.duckdb")
REPORT_HTML  = os.path.join(OUTDIR, "report.html")
LOGFILE      = os.path.join(OUTDIR, "pilot.log")

IGNORE_DIR_RE = re.compile(
    r'^(?:__pycache__|\.?venv[-_.]?[\w.]*|env|node_modules|bower_components'
    r'|site-packages|[\w.-]*\.egg-info|\.eggs|\.tox|\.nox|\.pytest_cache'
    r'|\.mypy_cache|\.ruff_cache|\.ipynb_checkpoints|\.[\w-]*cache'
    r'|python\d[.\d]*|\.next|\.nuxt|\.svelte-kit|\.turbo|\.yarn|\.parcel-cache'
    r'|\.git|\.hg|\.svn|\.idea|\.vscode|\.vs|build|dist|target|out|bin|obj'
    r'|\.gradle|vendor|\.bundle|\.m2|\.cargo|\.terraform|\.local'
    r'|\.Trash(?:es)?|\$RECYCLE\.BIN|System Volume Information'
    r'|\.dropbox\.cache|\.stfolder|\.stversions)$', re.I)

IGNORE_FILE_RE = re.compile(
    r'^(?:\.DS_Store|Thumbs\.db|desktop\.ini|\._.*|\.gitkeep)$'
    r'|\.(?:pyc|pyo|pyd|class|o|obj|lock|swp|part|crdownload)$', re.I)
# NOTE 2026-08-23 (owner directive): '.tmp' REMOVED from the junk extension
# list, and 'tmp'/'temp' are deliberately NOT in IGNORE_DIR_RE. Real corpus
# data lives in tmp/temp directories in this environment. Do not re-add them.

CRED_RE = re.compile(
    r'^(?:client_secret.*\.json|rclone\.conf|\.env|.*credentials.*\.json'
    r'|.*token.*\.json|id_rsa|.*\.pem|.*\.pfx'
    r'|chrome[ _-]?passwords.*\.csv|.*passwords?.*\.(?:csv|txt|kdbx))$', re.I)
# 'Chrome Passwords*.csv' added 2026-08-23 - two live browser credential
# exports were found in _backup_import/Documents/tmp/. ISOLATE, never ingest,
# never index, never embed. See ISOLATED_CREDENTIALS.csv.

MARKER_FILES = {
    "archive_browser.html": "google-takeout",
    "_chat.txt":            "whatsapp-export",
}
MARKER_DIRS = {
    ".obsidian": "obsidian-vault",
    ".git":      "git-repo",
}

def log(msg):
    line = "%s  %s" % (datetime.now().strftime("%H:%M:%S"), msg)
    print(line, flush=True)
    try:
        with open(LOGFILE, "a", encoding="utf-8") as f:
            f.write(line + "\n")
    except Exception:
        pass


# ============================================================================
# PHASE 1 - SCAN
# ============================================================================
def phase_scan():
    log("PHASE 1: scanning %s" % SOURCE)
    if not os.path.isdir(SOURCE):
        log("  !! SOURCE not found: %s" % SOURCE); return
    n_keep = n_prune_dir = n_prune_file = n_cred = 0
    pruned_dirs = {}
    t0 = time.time()
    with open(FILES_JSONL, "w", encoding="utf-8") as out:
        for dirpath, dirs, files in os.walk(SOURCE, followlinks=False):
            keep = []
            for d in dirs:
                if IGNORE_DIR_RE.match(d):
                    pruned_dirs[d] = pruned_dirs.get(d, 0) + 1
                    n_prune_dir += 1
                else:
                    keep.append(d)
            dirs[:] = keep                      # in-place = actually prunes
            for fn in files:
                full = os.path.join(dirpath, fn)
                if IGNORE_FILE_RE.search(fn):
                    n_prune_file += 1; continue
                if CRED_RE.match(fn):
                    n_cred += 1
                    log("  !! CREDENTIAL-LIKE FILE SKIPPED: %s" % full); continue
                try:
                    st = os.stat(full)
                except Exception as e:
                    log("  stat failed %s: %s" % (full, e)); continue
                rel = os.path.relpath(full, SOURCE).replace("\\", "/")
                out.write(json.dumps({
                    "rel": rel,
                    "dir": os.path.dirname(rel) or ".",
                    "name": fn,
                    "ext": os.path.splitext(fn)[1].lstrip(".").lower(),
                    "size": st.st_size,
                    "mtime": int(st.st_mtime),
                    "btime": int(getattr(st, "st_birthtime", st.st_ctime)),
                }) + "\n")
                n_keep += 1
                if n_keep % 10000 == 0:
                    log("  ... %d files" % n_keep)
    log("PHASE 1 done in %.1fs: %d kept | %d dirs pruned | %d junk files | %d credential-like"
        % (time.time()-t0, n_keep, n_prune_dir, n_prune_file, n_cred))
    if pruned_dirs:
        top = sorted(pruned_dirs.items(), key=lambda x: -x[1])[:10]
        log("  pruned dir names: " + ", ".join("%s x%d" % (k, v) for k, v in top))


# ============================================================================
# PHASE 2 - HASH  (SHA-256 + MD5 + SHA-1 in ONE read; resumable)
# ============================================================================
def phase_hash():
    log("PHASE 2: hashing (resumable)")
    if not os.path.exists(FILES_JSONL):
        log("  !! run phase 1 first"); return

    done = set()
    if os.path.exists(HASHES_JSONL):
        with open(HASHES_JSONL, "r", encoding="utf-8") as f:
            for line in f:
                try: done.add(json.loads(line)["rel"])
                except Exception: pass
        log("  resuming: %d already hashed" % len(done))

    todo, total_bytes = [], 0
    with open(FILES_JSONL, "r", encoding="utf-8") as f:
        for line in f:
            r = json.loads(line)
            if r["rel"] not in done:
                todo.append(r); total_bytes += r["size"]
    log("  to hash: %d files, %.2f GB" % (len(todo), total_bytes/1e9))
    if not todo:
        log("PHASE 2 done (nothing to do)"); return

    t0 = time.time(); done_bytes = 0; n_err = 0; CHUNK = 4 * 1024 * 1024
    with open(HASHES_JSONL, "a", encoding="utf-8") as out:
        for i, r in enumerate(todo, 1):
            full = os.path.join(SOURCE, r["rel"].replace("/", os.sep))
            h256, hmd5, hsha1 = hashlib.sha256(), hashlib.md5(), hashlib.sha1()
            err = None
            try:
                with open(full, "rb") as fh:
                    while True:
                        b = fh.read(CHUNK)
                        if not b: break
                        h256.update(b); hmd5.update(b); hsha1.update(b)
            except Exception as e:
                err = "%s: %s" % (type(e).__name__, str(e)[:120]); n_err += 1
            rec = dict(r)
            rec.update({
                "sha256": None if err else h256.hexdigest(),
                "md5":    None if err else hmd5.hexdigest(),
                "sha1":   None if err else hsha1.hexdigest(),
                "err":    err,
            })
            out.write(json.dumps(rec) + "\n")
            done_bytes += r["size"]
            if i % 500 == 0:
                out.flush()
                el = time.time() - t0
                rate = done_bytes/el/1e6 if el else 0
                pct = 100.0*done_bytes/total_bytes if total_bytes else 100
                eta = (total_bytes-done_bytes)/(done_bytes/el)/60 if done_bytes and el else 0
                log("  %d/%d  %.1f%%  %.0f MB/s  ETA %.0f min  errors=%d"
                    % (i, len(todo), pct, rate, eta, n_err))
    log("PHASE 2 done in %.1f min: %d hashed, %d errors"
        % ((time.time()-t0)/60, len(todo), n_err))


# ============================================================================
# PHASE 3 - ANALYZE  (local DuckDB; schema mirrors future PG18 layout)
# ============================================================================
def phase_analyze():
    import duckdb
    log("PHASE 3: analyze -> %s" % PILOT_DB)
    if not os.path.exists(HASHES_JSONL):
        log("  !! run phase 2 first"); return
    if os.path.exists(PILOT_DB):
        os.remove(PILOT_DB); log("  removed previous pilot.duckdb (ephemeral)")

    con = duckdb.connect(PILOT_DB)
    con.execute("CREATE SCHEMA IF NOT EXISTS pilot")

    con.execute("""
        CREATE TABLE pilot.files AS
        SELECT rel, dir, name, ext, size, mtime, btime,
               sha256, md5, sha1, err
        FROM read_json('%s', format='newline_delimited')
    """ % HASHES_JSONL.replace("\\", "/"))
    n = con.sql("SELECT count(*) FROM pilot.files").fetchone()[0]
    log("  loaded %d rows" % n)

    # --- corpus anti-join (MD5 is the only key r2_files carries) -------------
    have_corpus = os.path.exists(CORPUSDB)
    if have_corpus:
        con.execute("ATTACH '%s' AS corpus (READ_ONLY)" % CORPUSDB.replace("\\", "/"))
        con.execute("""
            CREATE TABLE pilot.corpus_md5 AS
            SELECT DISTINCT md5 FROM corpus.r2_files WHERE md5 IS NOT NULL
        """)
        log("  corpus md5 keys: %d" %
            con.sql("SELECT count(*) FROM pilot.corpus_md5").fetchone()[0])
    else:
        con.execute("CREATE TABLE pilot.corpus_md5 (md5 VARCHAR)")
        log("  !! corpus db missing - everything will read as NEW")

    con.execute("""
        CREATE TABLE pilot.status AS
        SELECT f.*,
               CASE WHEN f.err IS NOT NULL           THEN 'error'
                    WHEN f.size = 0                  THEN 'zero_byte'
                    WHEN c.md5 IS NOT NULL           THEN 'already_in_corpus'
                    ELSE 'net_new' END AS disposition
        FROM pilot.files f
        LEFT JOIN pilot.corpus_md5 c ON f.md5 = c.md5
    """)

    # --- folder-level metrics ------------------------------------------------
    con.execute("""
        CREATE TABLE pilot.folders AS
        WITH per AS (
          SELECT dir,
                 count(*) AS n_files,
                 sum(size) AS bytes,
                 count(*) FILTER (WHERE disposition='already_in_corpus') AS n_dup,
                 count(*) FILTER (WHERE disposition='net_new')           AS n_new,
                 count(*) FILTER (WHERE disposition='zero_byte')         AS n_zero,
                 count(*) FILTER (WHERE disposition='error')             AS n_err,
                 count(DISTINCT ext) AS n_ext,
                 md5(string_agg(COALESCE(sha256,'~'), '|' ORDER BY sha256)) AS fingerprint
          FROM pilot.status GROUP BY dir
        )
        SELECT *,
               ROUND(100.0*n_dup/NULLIF(n_files,0), 1) AS pct_dup,
               ROUND(100.0*n_new/NULLIF(n_files,0), 1) AS pct_remaining,
               CASE WHEN n_files=0 THEN 'empty'
                    WHEN n_new=0 THEN 'fully_redundant'
                    WHEN n_dup=0 THEN 'fully_unique'
                    WHEN 100.0*n_new/n_files BETWEEN 5 AND 40 THEN 'SHRED_RISK'
                    ELSE 'mixed' END AS cohesion
        FROM per
    """)
    log("  folders: %d" % con.sql("SELECT count(*) FROM pilot.folders").fetchone()[0])
    return con, have_corpus


# ============================================================================
# PHASE 3b - atomic-unit markers + proposed _raw/ layout (NO WRITES)
# ============================================================================
def phase_analyze_markers():
    import duckdb
    con = duckdb.connect(PILOT_DB)

    # marker-based atomic units: any dir containing a marker file/dir name
    marker_rows = []
    with open(FILES_JSONL, "r", encoding="utf-8") as f:
        for line in f:
            r = json.loads(line)
            nm = r["name"].lower()
            if nm in MARKER_FILES:
                marker_rows.append((r["dir"], MARKER_FILES[nm], r["name"]))
            parts = r["dir"].split("/")
            for p in parts:
                if p.lower() in MARKER_DIRS:
                    root = r["dir"].split(p)[0].rstrip("/") or "."
                    marker_rows.append((root, MARKER_DIRS[p.lower()], p))
    con.execute("CREATE OR REPLACE TABLE pilot.markers (dir VARCHAR, kind VARCHAR, evidence VARCHAR)")
    if marker_rows:
        con.executemany("INSERT INTO pilot.markers VALUES (?,?,?)", marker_rows)
    con.execute("""
        CREATE OR REPLACE TABLE pilot.atomic_units AS
        SELECT dir, kind, count(*) AS hits FROM pilot.markers GROUP BY 1,2 ORDER BY 1
    """)

    # duplicate clusters WITHIN the source (same content, multiple paths)
    con.execute("""
        CREATE OR REPLACE TABLE pilot.internal_dupes AS
        SELECT sha256, count(*) AS copies, any_value(size) AS size,
               string_agg(rel, ' || ' ORDER BY rel) AS paths
        FROM pilot.status
        WHERE sha256 IS NOT NULL AND size > 0
        GROUP BY sha256 HAVING count(*) > 1
        ORDER BY copies DESC, size DESC
    """)

    # PROPOSED landing layout - structure preserved verbatim, nothing flattened
    con.execute("""
        CREATE OR REPLACE TABLE pilot.proposed_raw AS
        SELECT rel AS source_rel,
               'casebible-sorted/_raw/f_case/' || rel AS proposed_key,
               size, sha256, md5, sha1, disposition
        FROM pilot.status
        WHERE disposition = 'net_new'
        ORDER BY rel
    """)
    n = con.sql("SELECT count(*) FROM pilot.proposed_raw").fetchone()[0]
    gb = con.sql("SELECT COALESCE(sum(size),0)/1e9 FROM pilot.proposed_raw").fetchone()[0]
    log("  proposed _raw/ landings: %d files, %.2f GB (NOT written)" % (n, gb))
    con.close()


# ============================================================================
# PHASE 4 - REPORT
# ============================================================================
def phase_report():
    import duckdb, html as _h
    log("PHASE 4: report -> %s" % REPORT_HTML)
    con = duckdb.connect(PILOT_DB, read_only=True)

    def one(sql, d=0):
        try: return con.sql(sql).fetchone()[0]
        except Exception: return d
    def rows(sql, lim=50):
        try: return con.sql(sql).fetchall()[:lim]
        except Exception: return []

    tot   = one("SELECT count(*) FROM pilot.status")
    tgb   = one("SELECT COALESCE(sum(size),0)/1e9 FROM pilot.status")
    disp  = rows("""SELECT disposition, count(*), ROUND(COALESCE(sum(size),0)/1e9,2)
                    FROM pilot.status GROUP BY 1 ORDER BY 2 DESC""")
    coh   = rows("""SELECT cohesion, count(*), ROUND(COALESCE(sum(bytes),0)/1e9,2)
                    FROM pilot.folders GROUP BY 1 ORDER BY 2 DESC""")
    shred = rows("""SELECT dir, n_files, n_dup, n_new, pct_remaining,
                    ROUND(bytes/1e6,1) FROM pilot.folders
                    WHERE cohesion='SHRED_RISK' ORDER BY n_files DESC""", 40)
    units = rows("SELECT dir, kind, hits FROM pilot.atomic_units ORDER BY kind, dir", 60)
    dupes = rows("""SELECT copies, ROUND(size/1e6,2), paths
                    FROM pilot.internal_dupes ORDER BY copies DESC LIMIT 25""")
    exts  = rows("""SELECT ext, count(*), ROUND(COALESCE(sum(size),0)/1e9,2)
                    FROM pilot.status GROUP BY 1 ORDER BY 2 DESC LIMIT 20""")
    errs  = rows("SELECT rel, err FROM pilot.status WHERE err IS NOT NULL LIMIT 30")
    newn  = one("SELECT count(*) FROM pilot.proposed_raw")
    newgb = one("SELECT COALESCE(sum(size),0)/1e9 FROM pilot.proposed_raw")

    def tbl(headers, data, cls=""):
        h = "".join("<th>%s</th>" % _h.escape(str(x)) for x in headers)
        b = "".join("<tr>%s</tr>" % "".join(
                "<td>%s</td>" % _h.escape(str(c)) for c in r) for r in data)
        return '<table class="%s"><thead><tr>%s</tr></thead><tbody>%s</tbody></table>' % (cls, h, b)

    css = """body{font:14px/1.55 -apple-system,Segoe UI,Roboto,sans-serif;
    max-width:1180px;margin:2rem auto;padding:0 1.5rem;color:#1a1a1a;background:#fafafa}
    h1{border-bottom:3px solid #222;padding-bottom:.4rem;margin-bottom:.2rem}
    h2{margin-top:2.2rem;border-bottom:1px solid #ccc;padding-bottom:.3rem}
    table{border-collapse:collapse;width:100%;margin:.8rem 0;background:#fff;font-size:13px}
    th{background:#222;color:#fff;text-align:left;padding:7px 10px;font-weight:600}
    td{border-bottom:1px solid #e5e5e5;padding:6px 10px;vertical-align:top}
    tr:hover td{background:#f0f4f8}
    .kpi{display:flex;gap:1rem;flex-wrap:wrap;margin:1.2rem 0}
    .kpi div{background:#fff;border:1px solid #ddd;border-left:4px solid #2563eb;
      padding:.8rem 1.1rem;min-width:150px;border-radius:3px}
    .kpi b{display:block;font-size:1.6rem;line-height:1.1}
    .kpi span{color:#666;font-size:12px;text-transform:uppercase;letter-spacing:.4px}
    .warn{border-left-color:#dc2626!important}
    .ok{border-left-color:#16a34a!important}
    .note{background:#fffbe6;border-left:4px solid #d97706;padding:.8rem 1.1rem;margin:1rem 0}
    code{background:#eee;padding:1px 5px;border-radius:3px;font-size:12px}
    .mono{font-family:ui-monospace,Consolas,monospace;font-size:12px;word-break:break-all}"""

    parts = ["<!doctype html><html><head><meta charset='utf-8'>",
             "<title>Case Bible - F:\\case dry run</title><style>%s</style></head><body>" % css,
             "<h1>Case Bible &mdash; <code>F:\\case</code> dry run</h1>",
             "<p style='color:#666'>Generated %s &middot; READ-ONLY &middot; nothing written to R2, PG18, or any source</p>"
                % datetime.now().strftime("%Y-%m-%d %H:%M"),
             "<div class='kpi'>",
             "<div><span>Files scanned</span><b>{:,}</b></div>".format(tot),
             "<div><span>Total size</span><b>{:.1f} GB</b></div>".format(tgb),
             "<div class='ok'><span>Net-new</span><b>{:,}</b>{:.1f} GB</div>".format(newn, newgb),
             "<div class='warn'><span>Shred-risk folders</span><b>{:,}</b></div>".format(len(shred)),
             "</div>",
             "<div class='note'><b>Structure preserved.</b> Every proposed landing keeps its full "
             "source path under <code>casebible-sorted/_raw/f_case/</code>. No package is "
             "disassembled, nothing is flattened or renamed. Atomic units are reported for "
             "discussion, not acted on.</div>",
             "<h2>Disposition</h2>", tbl(["Disposition","Files","GB"], disp),
             "<h2>Folder cohesion</h2>", tbl(["Cohesion","Folders","GB"], coh),
             "<div class='note'><b>SHRED_RISK</b> = 5&ndash;40% of the folder would survive "
             "file-level dedup. Deduping these leaves orphaned fragments with no context. "
             "Handle as whole units or not at all.</div>",
             tbl(["Folder","Files","Dup","New","% remaining","MB"], shred),
             "<h2>Atomic units detected</h2>", tbl(["Folder","Kind","Hits"], units),
             "<h2>Duplicate clusters inside the source</h2>",
             tbl(["Copies","MB","Paths"], [(a,b,c[:300]) for a,b,c in dupes]),
             "<h2>Extensions</h2>", tbl(["Ext","Files","GB"], exts),
             "<h2>Read errors</h2>", tbl(["Path","Error"], errs) if errs else "<p>None.</p>",
             "</body></html>"]

    with open(REPORT_HTML, "w", encoding="utf-8") as f:
        f.write("".join(parts))
    con.close()
    log("PHASE 4 done: %s" % REPORT_HTML)


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--phase", action="append", type=int, choices=[1,2,3,4])
    ap.add_argument("--all", action="store_true")
    a = ap.parse_args()
    phases = sorted(set(a.phase or [])) or ([1,2,3,4] if a.all else [])
    if not phases:
        ap.print_help(); sys.exit(1)
    os.makedirs(OUTDIR, exist_ok=True)
    log("=" * 64)
    log("cb_pilot  source=%s  phases=%s" % (SOURCE, phases))
    if 1 in phases: phase_scan()
    if 2 in phases: phase_hash()
    if 3 in phases:
        r = phase_analyze()
        if r: r[0].close()
        phase_analyze_markers()
    if 4 in phases: phase_report()
    log("ALL REQUESTED PHASES COMPLETE")
