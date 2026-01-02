# Extracted Concepts from Old Iterations

**Purpose:** This document captures useful patterns, algorithms, and design decisions from older code versions before archiving them. These concepts may be reused or adapted in future development.

---

## 1. From `helpers (2).py` - Utility Functions

### **Geohash Encoding (No External Dependencies)**
```python
# Pure Python geohash encoder - precision 8-9 for location clustering
def _geohash_encode(lat: float, lon: float, precision: int=8) -> str
```
**Use Case:** Group nearby locations without exact coordinate matching
- Precision 8: ~38m x 19m cells
- Precision 9: ~4.8m x 4.8m cells

**Application:** Cluster home/work locations, detect frequent visit patterns

### **Coordinate Rounding for Cache Lookups**
```python
def add_rounding_columns(df, col, prefix):
    # Adds lat_r3, lng_r3, lat_r4, lng_r4, geohash8, geohash9
```
**Strategy:** Multiple precision levels for geocoding cache hits
- **r3 (3 decimals):** ~111m precision - neighborhood level
- **r4 (4 decimals):** ~11m precision - building level
- **r5 (5 decimals):** ~1.1m precision - exact location

**Why:** Geocoding APIs charge per request. Rounding increases cache hit rates while maintaining acceptable accuracy for legal purposes.

### **Haversine Distance Calculation**
```python
def haversine_m(lat1, lon1, lat2, lon2) -> float:
    # Returns meters between two lat/lng points
```
**Application:**
- Multi-device detection (same time, >100m apart)
- Impossible travel detection
- Path distance calculations

### **Semantic Type Normalization**
```python
mapping = {
    "IN_PASSENGER_VEHICLE": "driving",
    "INFERRED_WORK": "work",
    "INFERRED_HOME": "home",
    "POINTS_AND_PATH": "points_and_path"
}
```
**Pattern:** Normalize Google's verbose types to human-readable labels for attorney review.

---

## 2. From `orchestrator (6).py` - Multi-Pass Pipeline Architecture

### **6-Pass Processing Workflow**
```
Pass 1: Parse JSON → Flatten segments
Pass 2: Sort & Assign Event IDs
Pass 3: Analytics (duration, overnight flags, gaps)
Pass 4: Link waypoints to parent paths
Pass 4.5: Generate API request orders (Radar/Google)
Pass 5: (MISSING) Merge geocoding results
Pass 6: (MISSING) Human-readable export + analysis
```

**Key Insight:** Separation of concerns allows:
1. **Resume from failure** - Each pass saves intermediate CSVs
2. **Parallel processing** - Pass 4.5 API calls can run async
3. **Testing isolation** - Test each transformation independently
4. **Preview files** - Generate .preview.csv (first 25 rows) for QA

### **Directory Structure (Old System)**
```
project/
├── source_data/           # Input JSON files
├── source_data/completed/ # Processed files (archived)
├── working_data/
│   ├── pass1_flatten_json/
│   ├── pass2_sorted_records/
│   ├── pass3_analytics/
│   └── pass4_linked_points/
├── output_data/
│   ├── master_csvs/
│   ├── api_orders/       # Radar/Google request batches
│   └── human_readable_csvs/
├── raw_api_responses/     # Cache by provider
└── logs/
```

**Pattern:** Immutable intermediate stages → Easy debugging, rollback capability

---

## 3. From `pass4_analytics (3).py` - Advanced Analytics

### **Path Timestamp Inference**
**Problem:** Google Timeline container timestamps are garbage (2-hour windows)

**Solution:** Multi-source timestamp resolution priority:
```
1. First/last waypoint times (highest accuracy)
2. Adjacent visit/activity endpoint times
3. Container times (fallback only)
```

**Algorithm:**
```python
def _infer_path_times(df):
    # For each timeline_path segment:
    # 1. Check waypoints for first/last time
    # 2. Find adjacent visit/activity within 5min tolerance
    # 3. Use adjacent endpoints if times align
    # 4. Flag: path_time_source = 'points_first_last' | 'adjacent_events' | 'container_fallback'
```

**Forensic Value:** Documents timestamp source for court scrutiny.

### **Overnight Detection (Local Clock)**
```python
def _is_overnight_local(ts):
    # 22:00–07:00 window (coarse check)
    hour = ts.hour
    return hour >= 22 or hour < 7
```

**Refinement Needed:**
- Check timezone offset for true local time
- Flag multi-day stays separately
- Detect anomalous overnight locations (not home)

### **Gap Analysis**
```python
# Time gap between consecutive segments
segs["gap_to_prev_sec"] = (segs["__start"] - segs["__prev_end"]).dt.total_seconds()
```

**Application:** Detect missing data windows (phone off, airplane mode, data deletions)

### **Path Distance Calculation**
```python
# Sum haversine distances between consecutive waypoints
for parent_path:
    total_dist = sum(haversine(pt[i], pt[i+1]) for i in range(len(pts)-1))
```

**Forensic Flag:** Compare to Google's `distanceMeters` field
- Large discrepancy → Possible manual route edit or GPS drift
- Missing field → Incomplete data

---

## 4. From `resolve_geocodes (4).py` - Cache-First API Strategy

### **Three-Tier Geocoding Architecture**

**Tier 1: File-Based Cache (Fastest)**
```python
_cache_path(provider, mode, key) -> Path
# Structure: raw_api/google/place_details/{sha256}.json
#            raw_api/radar/reverse_geocode/{sha256}.json
```

**Tier 2: Database Deduplication**
```sql
geocode_request (request_uuid PK, status, latlng_r4, latlng_r5)
geocode_result_google (result_uuid PK, request_uuid FK, ...)
geocode_result_radar (result_uuid PK, request_uuid FK, ...)
```

**Tier 3: Live API Call (Last Resort)**
```python
res = load_cache(provider, mode, key)
if not res and api_key:
    res = call_api(...)
    save_cache(provider, mode, key, res)
```

**Cost Savings:** Matt's 99% cache hit rate = $0 API costs

### **Multi-Precision Lookup Keys**
```python
request = {
    'latlng_exact6': "43.123456,-83.654321",  # 6 decimals
    'latlng_r5': "43.12346,-83.65432",        # 5 decimals
    'latlng_r4': "43.1235,-83.6543",          # 4 decimals
}
```

**Lookup Strategy:**
1. Try exact match first
2. Fallback to r5
3. Fallback to r4
4. Only call API if all miss

**Database Pattern:** Store ALL precision variants to maximize future cache hits.

---

## 5. Design Patterns Worth Keeping

### **UUID Hashing for Deduplication**
```python
location_uuid = hashlib.sha256(latlng.encode()).hexdigest()
request_uuid = hashlib.sha256(f'{provider}:{key}'.encode()).hexdigest()
```

**Benefit:** Idempotent inserts via `ON CONFLICT(...) DO UPDATE`

### **Upsert Helper**
```python
def upsert(conn, table, row, pk):
    # INSERT ... ON CONFLICT(pk) DO UPDATE SET ...
```

**Pattern:** Replay-safe processing → Can rerun passes without corruption

### **Progressive Preview Files**
```python
def preview_csv(df, path):
    df.head(25).to_csv(path, index=False)
```

**UX:** Attorney can review samples without loading 100K+ row files

### **Automatic Directory Creation**
```python
REQUIRED_DIRS = [...]
for d in REQUIRED_DIRS:
    (PROJECT_ROOT / d).mkdir(parents=True, exist_ok=True)
```

**Pattern:** Zero-config setup, works on any machine

---

## 6. Anti-Patterns to Avoid

### ❌ **Timestamp-Based Waypoint Ordering**
```sql
ORDER BY timestamp  -- FAILS on duplicate timestamps!
```
**Correct:**
```sql
ORDER BY sequence  -- Guaranteed unique within parent
```

### ❌ **Hardcoded Paths**
```python
df = pd.read_csv("/home/claude/data/file.csv")  # Breaks portability
```
**Correct:**
```python
PROJECT_ROOT = Path(__file__).resolve().parents[1]
df = pd.read_csv(PROJECT_ROOT / "data/file.csv")
```

### ❌ **Mixing Concerns in Single Pass**
```python
# DON'T: Parse + geocode + analyze in one loop
# DO: Separate passes with intermediate saves
```

---

## 7. Forensic Best Practices

### **Chain of Custody Metadata**
Every output file should include:
```json
{
    "source_file": "Timeline.json",
    "source_sha256": "abc123...",
    "processed_at": "2025-12-15T20:30:00Z",
    "processing_version": "v4.2",
    "operator": "Matt Salem",
    "command": "python parser.py --input=... --output=..."
}
```

### **Immutable Raw Data**
```python
# COPY source to completed/ folder
shutil.copy2(src, completed_dir / src.name)
# NEVER modify original
```

### **Audit Trail**
```python
# Log every transformation
logging.info("Pass 3: Computed analytics for %d segments", len(df))
# Capture warnings
logging.warning("Segment %d: Missing timestamp, using container fallback", idx)
```

---

## 8. Future Enhancements (From Old Code TODOs)

### **Location Clustering (Unsupervised)**
```python
# Group by geohash8, find top K clusters by visit count
# Label as "home", "work", "school", "frequent_location_N"
```

### **Impossible Travel Detection**
```python
# Check if distance / time_gap > 200mph
# Flag: "ANOMALY: Requires flight, but no FLYING activity"
```

### **Multi-Source Validation**
```python
# Cross-reference Radar vs Google addresses
# If mismatch: Flag for manual review
```

### **Snap to Roads API**
```python
# For driving segments, get actual route polyline
# Compare to waypoint scatter → Detect manual edits
```

---

## Summary

**Keep These Patterns:**
1. Multi-pass pipeline with intermediate saves
2. Cache-first API strategy with multi-precision lookups
3. Timestamp inference from waypoints (not containers)
4. Geohash clustering for location analysis
5. Forensic metadata in all outputs

**Migrate to New System:**
1. Haversine distance calculations
2. Overnight detection logic (refine timezone handling)
3. Gap analysis (detect data deletions)
4. Preview file generation
5. Upsert patterns for idempotent processing

**Archived (Obsolete):**
- Old directory structure (replaced by src/data/docs/)
- Hardcoded paths
- UUID v4 (replaced by UUIDv7 in new code)

---

**Document Version:** 1.0
**Created:** 2025-12-15
**Last Updated:** 2025-12-15
