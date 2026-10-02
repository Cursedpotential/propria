"""Database connection string for the ops scripts, read from the environment only.

TRACEIQ_DSN_KV is the libpq key=value form; the UI reads the URL form, TRACEIQ_DSN.
Both live in ~/.secrets/traceiq-db.env; ../.env.example names them.

_Byline: Claude Code · Opus 5.5 · 2026-10-02_
"""
import os
import sys

DSN = os.environ.get("TRACEIQ_DSN_KV")
if not DSN:
    sys.exit("TRACEIQ_DSN_KV is not set. Load it from ~/.secrets/traceiq-db.env "
             "(names in vestigia/.env.example).")
