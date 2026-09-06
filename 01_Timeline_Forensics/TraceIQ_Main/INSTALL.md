# TraceIQ Timeline Processor - Installation Summary

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


## Quick Start
cd traceiq_timeline_processor
./launch.sh

## Access
Open browser: http://localhost:5000

## Features Included
- ✅ Complete SQLite database with all indexes
- ✅ Place analytics, anomaly detection, forensic views
- ✅ Data quality metrics and validation
- ✅ Real-time processing status and progress
- ✅ Error handling and logging
- ✅ Export capabilities (CSV, JSON planned)

## Directory Structure
traceiq/
├── app.py                 # Flask application
├── launch.sh             # Launcher script
├── requirements.txt      # Python dependencies
├── schema_complete.sql   # Complete SQLite schema
├── templates/            # HTML templates
├── static/js/           # JavaScript files
└── data/
    ├── raw/             # Upload timeline JSON
    ├── processed/       # SQLite databases
    └── logs/            # Application logs

## Logs
- Application log: data/logs/app.log
- Processing errors: Viewed in web UI
- Quality metrics: /api/quality endpoint

## Upgrading to PostgreSQL
See migrate_to_supabase.sh for PostgreSQL migration
