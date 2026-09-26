# Google Timeline Forensic Processor - Workspace

**Project:** Google Timeline location data processing for legal proceedings (Salem v. Kinzel custody case)
**Purpose:** Parse, enrich, and export Google Timeline JSON into court-admissible evidence with full chain of custody
**Status:** Active Development
**Last Organized:** 2025-12-15

---

## Directory Structure

```
Folder/
├── src/                          # Active source code (current iteration)
│   ├── parser.py                # Main Timeline JSON parser (Nov 15, 2024)
│   ├── test_parser.py           # Parser test suite
│   ├── schema.sql               # SQLite database schema
│   ├── azure_timeline_processor.py  # Forensic processor with LLM analysis
│   ├── evidence_processor_v4.js # React Evidence Processor (Rev 12 - LATEST)
│   ├── plugins_parsing_salem_v_1.py # Custom plugins
│   ├── timeline_postgresql.sh   # PostgreSQL migration script
│   └── timeline_sqlite.sh       # SQLite processing script
│
├── data/                        # Data files (excluded from git)
│   ├── databases/               # SQLite database files
│   │   ├── timeline_full.sqlite (17MB)
│   │   └── timeline_radar_enriched (1).sqlite (39MB)
│   ├── caches/                  # Geocoding caches
│   │   ├── radar_geocoding_master_good.csv (7MB, 50K+ records)
│   │   └── place_id_db_REPAIRED.json (504KB)
│   ├── enrichment/              # API response caches
│   │   ├── google_enrichment.json
│   │   ├── radar_enrichment.json
│   │   ├── place_id_requests (1).json
│   │   └── geodata_policy.json
│   └── exports/                 # Generated output files
│       ├── overnight_stays_2023.csv
│       ├── overnight_summary_2023.csv
│       ├── overnight_deviations_2023.csv
│       ├── kepler_export.json
│       ├── timeline_export.kml
│       ├── verification_log.jsonl
│       ├── qa_summary.json
│       ├── table_schema.csv
│       └── sample_2024.json
│
├── docs/                        # Documentation & specifications
│   ├── Timeline Processor (Rebuilt).docx.md  # 6-pass pipeline spec
│   ├── google_timeline_forensic_processor_prompt.md  # Main system prompt
│   ├── google_timeline_addendum.md           # Geocoding & case context
│   ├── google_timeline_addendum_part2.md     # Advanced features
│   ├── timeline_parser_prompt.md             # Conversation parser spec
│   ├── Salem_Workspace_Project_Prompt.md     # Workspace configuration
│   ├── QUICK_REFERENCE.md                    # Implementation guide
│   ├── geocoding-usage-guide.md              # Geocoding API guide
│   ├── geo_timeline_tools_v_3.md             # Tool documentation
│   ├── MASTER_TIMELINE_CONSOLIDATED.md       # Consolidated timeline doc
│   └── EXTRACTED_CONCEPTS.md                 # Concepts from old iterations
│
├── utils/                       # Utility scripts
│   ├── logging_config.py        # Logging setup
│   ├── utils_io.py              # I/O utilities
│   ├── overnight_analyzer.py    # Overnight stay analysis
│   ├── schedule_analyzer.py     # Schedule pattern analysis
│   ├── test_db_connection.py    # Database connection test
│   ├── validate_timeline_data.py # Data validation
│   ├── fix_json_manually.py     # JSON repair utility
│   └── fix_place_cache.py       # Place cache repair
│
├── evidence_output/             # Evidence Processor exports
│   ├── _CUSTODY_REPORT.json
│   ├── _CUSTODY_REPORT (2).json
│   ├── evidence_csv_part_1.csv
│   ├── evidence_detailed_csv_part_1.csv
│   ├── evidence_jsonl_part_001.jsonl
│   └── master_evidence_extraction.csv
│
├── reference/                   # Third-party reference projects
│   ├── location-history-json-converter-master/  # Python converter
│   ├── maps-visualizer-master/                  # Map visualization
│   ├── TimelineExtractor-master/                # KML extraction tool
│   ├── google-maps-timeline-viewer-main/        # HTML viewer
│   ├── GoogleMapsTimelineActivityViewer-master/ # Activity viewer
│   └── ai_studio_app/                           # Google AI Studio app (unrelated)
│
├── archive/                     # Old iterations & conversations
│   ├── old_conversations/       # Exported conversation transcripts
│   │   ├── Analyze last conversation.md
│   │   ├── I would like you to pick up Where we left off just (2).md
│   │   ├── 8a152aa7-a106-4f1d-9110-ecf1deaf418f.md
│   │   ├── Google_Gemini_2025-12-09_1634.md
│   │   ├── gemini_building-vertex-ai-app-prompt_2025-12-04T08-15-35-0500.md
│   │   ├── notes - analysis module ..docx
│   │   ├── Based on the information you provided.docx
│   │   ├── Copy of Create a handoff document please and make sure it..-1..docx
│   │   └── Evidence Processor v4 Code Hand-off.docx
│   ├── misc/                    # Miscellaneous archived files
│   │   ├── timeline_etl_scripts.zip
│   │   └── timeline_chunk_2023-Q1.config
│   ├── evidence_processor_v7.js # Older React version (archived)
│   ├── helpers (2).py           # Old utility functions (see EXTRACTED_CONCEPTS.md)
│   ├── orchestrator (6).py      # Old 6-pass orchestrator
│   ├── pass4_analytics (3).py   # Old analytics module
│   ├── pass4_5_generate_api_orders (2).py  # Old API order generator
│   ├── resolve_geocodes (4).py  # Old geocoding resolver
│   ├── sqlite_schema (5).sql    # Old schema version
│   ├── human_readable_view (4).sql  # Old view definition
│   ├── supabase_schema.sql      # Supabase schema (unused)
│   ├── init_local_db.py         # Old DB init script
│   └── google.py                # Old Google API wrapper
│
└── README.md                    # This file
```

---

## Project Overview

### **Three Main Components:**

1. **Google Timeline Forensic Processor (Python)**
   - **Status:** ~40% complete
   - **Current:** `src/azure_timeline_processor.py` (Dec 9, 2024)
   - **Features:**
     - Streaming JSON parser
     - UUIDv7 ID generation
     - Container timestamp fixing
     - Multi-device detection
     - SQLite database with forensic schema
     - Optional LLM analysis (Azure/Qwen)
   - **Missing:**
     - Geocoding integration (cache exists, not integrated)
     - Human-readable CSV export with full schema
     - Passes 3-6 from specification
     - Flet UI

2. **Evidence Processor (React/JavaScript)**
   - **Status:** 95% complete, production-ready
   - **Current:** `src/evidence_processor_v4.js` (Rev 12)
   - **Features:**
     - Browser-based forensic converter
     - Dual-pass processing (analysis → export)
     - UUIDv7 + SHA-256 hashing
     - Dynamic column ordering
     - Split vs combined row modes
     - ZIP archive export with custody report
     - Dark mode UI
   - **Missing:**
     - Geocoding cache integration
     - Backend database integration

3. **Conversation Timeline Parser**
   - **Status:** 0% complete (spec only)
   - **Spec:** `docs/timeline_parser_prompt.md`
   - **Purpose:** Parse FB Messenger, SMS, WhatsApp conversations into unified timeline

---

## Key Data Files

### **Geocoding Cache (CRITICAL)**
- **File:** `data/caches/radar_geocoding_master_good.csv` (7MB)
- **Records:** ~50,000 reverse geocoded locations
- **Coverage:** 99% of timeline locations
- **Purpose:** Avoid API costs by using pre-cached addresses
- **Format:** `lat,lng,formatted_address,city,state,postal_code,place_name,api_called_at`

### **Databases**
- **`timeline_full.sqlite`** (17MB) - Base timeline database
- **`timeline_radar_enriched (1).sqlite`** (39MB) - Enriched with geocoding

### **Enrichment Caches**
- **`google_enrichment.json`** - Google Places API responses
- **`radar_enrichment.json`** - Radar API responses
- **`place_id_db_REPAIRED.json`** - Repaired Place ID database

---

## Quick Start

### **1. Parse Timeline JSON**
```bash
cd src
python parser.py ../data/Timeline.json
```

### **2. Run Tests**
```bash
cd src
python test_parser.py
# Output: ✓ ALL TESTS PASSED
```

### **3. Use Evidence Processor (React)**
- Open `src/evidence_processor_v4.js` in a React app
- Or create standalone HTML file with embedded React
- Load Timeline.json file
- Configure columns and export options
- Download ZIP archive with CSV/JSONL

### **4. Azure Forensic Processor**
```bash
# Set environment variables
export AZURE_ENDPOINT_URL="your_endpoint"
export AZURE_API_KEY="your_key"

cd src
python azure_timeline_processor.py ../data/Timeline.json --analyze
```

---

## Documentation

### **Implementation Specifications**
1. **`docs/Timeline Processor (Rebuilt).docx.md`** - 6-pass pipeline architecture
2. **`docs/google_timeline_forensic_processor_prompt.md`** - System prompt with complete schema
3. **`docs/QUICK_REFERENCE.md`** - Quick implementation guide

### **Concepts & Patterns**
- **`docs/EXTRACTED_CONCEPTS.md`** - Useful patterns from old code iterations

### **Legal Context**
- **`docs/Salem_Workspace_Project_Prompt.md`** - Case background and workspace config

---

## Development Status

### **Completed:**
✅ JSON parsing with timestamp fixing
✅ Multi-device detection
✅ SQLite database schema
✅ React Evidence Processor with full UI
✅ UUIDv7 forensic IDs
✅ SHA-256 hashing
✅ Geocoding cache (7MB CSV)

### **In Progress:**
🔨 Geocoding cache integration
🔨 Human-readable CSV export (93-column schema)
🔨 Overnight stay analysis

### **Not Started:**
❌ Passes 3-6 (analytics, enrichment, merging)
❌ Flet UI for Python processor
❌ Conversation timeline parser
❌ Multi-source geocoding validation (Radar vs Google)
❌ Snap to Roads API integration

---

## File Naming Conventions

### **Active Files (in src/):**
- No version numbers
- Latest working code only

### **Archived Files:**
- Original names preserved
- Version numbers in parentheses: `helpers (2).py`
- Concepts extracted to `docs/EXTRACTED_CONCEPTS.md`

### **Evidence Processor Versions:**
- **v4 (Rev 12):** `src/evidence_processor_v4.js` ← **CURRENT**
- **v7 (Rev 7):** `archive/evidence_processor_v7.js` ← Archived

---

## Important Notes

### **Do NOT Delete:**
- Geocoding cache (`data/caches/radar_geocoding_master_good.csv`) - saves $$$ in API costs
- Database files in `data/databases/` - contain processed timeline data
- Any files in `archive/` - historical reference

### **Git Ignore:**
Add to `.gitignore`:
```
data/databases/*.sqlite
data/caches/*.csv
data/enrichment/*.json
data/exports/*
evidence_output/*
*.docx
```

### **Chain of Custody:**
All exports must include:
- Source file SHA-256 hash
- Processing timestamp
- Operator name
- Processing version
- Command used

---

## Next Steps

### **Immediate Priorities:**
1. **Integrate geocoding cache** into React Evidence Processor
   - Add CSV parser for `radar_geocoding_master_good.csv`
   - Add lookup logic: `{lat,lng} → {address, city, state}`
   - Add columns: `start_address`, `start_city`, `end_address`, etc.

2. **Implement full 93-column CSV schema**
   - Human-readable dates/times (EST)
   - Overnight flags
   - Place names and types
   - Google Maps links
   - Distance in miles

3. **Complete Python processor Passes 3-6**
   - Pass 3: Master cache building
   - Pass 4: Geocoding enrichment
   - Pass 5: Data merging
   - Pass 6: Human-readable export + analysis

---

## Contact & Legal

**Case:** Salem v. Kinzel (Michigan Family Court)
**Operator:** Matt Salem
**Purpose:** MCL 722.23 custody factor evidence (location history proof)
**Data Sensitivity:** HIGH - Contains detailed location history

**⚠️ CONFIDENTIAL - ATTORNEY WORK PRODUCT ⚠️**

---

**Last Updated:** 2025-12-15
**Version:** 1.0
**Organization Date:** 2025-12-15
