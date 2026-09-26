# Human-Readable Master CSV - Forensic Export Specification

## Purpose:
Create a **single master CSV file** that attorneys, judges, and forensic examiners can open in Excel/LibreCalc and immediately understand WITHOUT needing to know the database schema.

---

## High-Level Pipeline

```
Google Timeline JSON
     ↓
[1. DATA CLEANING / NORMALIZATION]

* ## Master CSV (flattened, hashed, forensically sound)

* ## This is the basic analysis or really it's just creation of derived fields for point to point durations overnight flags I Anomalies such as the dual device discovery and gap analysis If there's any other really basic derived fields that are in the tables this is where we would do it As well as the creation of human readable fields and the relocation of the machine fields to the end of the tables
↓
[2. DATABASE TABLES (normalized, UUIDv7, relational)] Including all of the basic analysis derived fields and human readable fields maintaining the linked hashes the linked parent IDS all of the chain of custody forensic best practice things
     ↓
[3. ENRICHMENT (geocoding, analysis)] * ## I believe best best practice would be to utilize the geokey table and I think this is where those come in to link UUI DS with Geokeys for lookup locations to the reverse geocoded data as well as the reverse Google Place ID data and that's how it should probably be stored in the database but verify that
     ↓
* ## This is where we would build our databases     

[4. HUMAN-READABLE CSV EXPORT]  * ## Create joined enriched table      ↓

need to discuss the timeline paths and how to handle them
```

---

## STEP 1: Data Cleaning / Normalization (Before Database Insert)

### What We Clean:

#### **1.1 Coordinate Normalization**
**Problem:** Google uses multiple coordinate formats
- Sometimes: `{"latitude": 43.012, "longitude": -83.687}`
- ****  Sometimes: `{"latitudeE7": 430120000, "longitudeE7": -836870000}` (multiplied by 10^7)   **** This is not used in any of my data You should not give yourself away that you didn't do your homework and actually look at the shit

**Cleaning:**
```
IF field contains "latitudeE7":
    latitude = latitudeE7 / 10000000.0
    longitude = longitudeE7 / 10000000.0
ELSE:
    latitude = latitude (as-is)
    longitude = longitude (as-is)

    also wrong And as this has been reiterated in every conversation for the last nine months no less than 5 times per conversation again giving yourself away that you didn't do the homework we're going to remove any special characters lat long goes into the same field separated by a Comma

Round to 6 decimals (forensic precision ±11cm)
Validate: -90 <= latitude <= 90
Validate: -180 <= longitude <= 180
```
Again Incorrect Also discussed 14 times in every conversation for the last nine months,,,,
We maintain full latitude longitude points We are creating rounded points and inserting them into a new field a new which will also be another table will be the lookup table for the reduction of noise and API calls So we want it narrow it down to eleven meters or so That will strictly be for API lookups However we are maintaining the Accurate latitude longitude precisions

You really made a whole lot of assumptions not grounded in any of the research you realize I've been working on this for a year actually over a year 14 months if you would have just read the fucking shit that I gave you we wouldn't be doing this so I'm not going any further We have the same time stamps all the way across the board OK we've addressed how to handle them I'm not doing anymore I you can you can figure the rest out as I asked you to as you said you did as you should have done before you sorted everything Go one topic at a time if you want to do keyword searches or regex one topic at a time search for it in the database ground yourself in the data that I've already went over reiterated 1000 times There's even a timeline folder in one of parent folders I can direct you to it where you can see 14 months of reiterated conversation that goes over literally anything and everything we could possibly run into Any Conclusions that you game came to and any rules that you wrote ad hoc without pulling from existing script and existing conversation you need to undo and stop using your Google search evidently you need to ground yourself in the conversation and the data and the reiterations and the refinement and the debugging that I've sent 14 months doing Every category that I read pisses me off because it's already been done 14 times we're not reinventing a wheel here Everything's been debugged it's just some of it's been debugged in different conversations some of it's been refined some of it's been changed some of the codes been the language has been changed but all of the logic is there somewhere and it's been checked double checked triple checked debug debug rebug debugged again
#### **1.2 Timestamp Normalization**
**Problem:** Google uses inconsistent timestamp formats
- ISO 8601 with timezone: `"2023-09-15T14:30:00.000-04:00"`
- Sometimes missing milliseconds: `"2023-09-15T14:30:00-04:00"`
- Sometimes UTC: `"2023-09-15T18:30:00.000Z"`

**Cleaning:**
```
Parse using dateutil.parser (handles all ISO 8601 variants)
Extract timezone offset in minutes: -240 for EDT, 0 for UTC
Store in database as UTC timestamp (TEXT, ISO 8601 format)
Store timezone_offset separately (INTEGER, minutes)
```

#### **1.3 Duration Cleaning**
**Problem:** Google sometimes gives duration in seconds, sometimes missing

**Cleaning:**
```
IF duration field exists:
    duration_seconds = duration (as-is)
ELSE:
    duration_seconds = (end_time - start_time).total_seconds()

Validate: duration_seconds >= 0 (end can't be before start)
Validate: duration_seconds < 31536000 (no single event > 1 year)

Flag as "calculated" if not provided by Google
```

#### **1.4 Place ID Cleaning**
**Problem:** Sometimes missing, sometimes malformed

**Cleaning:**
```
IF place_id exists AND starts with "ChIJ":
    place_id = place_id (valid Google Place ID)
ELSE IF place_id exists but invalid format:
    place_id = NULL
    Flag: "malformed_place_id"
ELSE:
    place_id = NULL
    Flag: "missing_place_id"
```

#### **1.5 Semantic Type Normalization**
**Problem:** Google uses verbose, inconsistent semantic types

**Cleaning:**
```
Mapping:
  "IN_PASSENGER_VEHICLE" → "Driving"
  "IN_VEHICLE" → "Driving"
  "WALKING" → "Walking"
  "CYCLING" → "Biking"
  "FLYING" → "Flying"
  "INFERRED_HOME" → "Home"
  "INFERRED_WORK" → "Work"
  "UNKNOWN_ACTIVITY_TYPE" → "Unknown"
  NULL → "Not Specified"
```

#### **1.6 Missing Data Handling**
**Rule:** NEVER throw away data, even if incomplete

```
IF visit has no place_id:
    → Still insert as visit
    → Flag: "orphaned_visit" (no place linkage)
    → Use coordinates only

IF activity has no distance:
    → Calculate from waypoints (if available)
    → OR flag: "distance_unknown"

IF path has no waypoints:
    → Insert as "orphaned_path"
    → Use container timestamps (flagged as "fallback")
```

---

## STEP 2: Database Storage (Normalized, Relational)

Tables:
- visits, activities, timeline_paths, waypoints
- event_geokey (coordinate rounding)
- geocode_result_* (API responses)
- home_base, expected_schedule, problematic_locations_contacts
- processing_metadata, parse_errors

**All using UUIDv7 primary keys**
**All normalized (no duplicate data)**
**All relational (foreign keys)**

---

## STEP 3: Enrichment (Add Human-Readable Data)

- Geocode: lat/lng → addresses, cities, states
- Analyze: overnight stays, gaps, multi-device
- Cluster: detect home/work from frequency
- Validate: multi-source geocoding resolution

---

## STEP 4: Human-Readable CSV Export

### Core Principle:
**Attorneys should be able to:**
1. Open CSV in Excel
2. Understand every field without a data dictionary
3. Sort/filter by date, location, type
4. See addresses instead of coordinates
5. Verify forensic hashes for chain of custody

---

## Human-Readable Master CSV Schema (100+ columns)

### Section A: Identity & Chain of Custody (10 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **record_uuid** | UUIDv7 | `01924d58-9e7f-72b0-8e6e-3a5f8c2d1b0a` | Unique, time-sortable identifier |
| **record_hash** | SHA-256 | `a3f2c8...` (first 16 chars shown) | SHA-256 of ALL data fields (for forensic verification) |
| **record_type** | TEXT | `Visit` or `Activity` or `Path` | Human-readable event type |
| **source_file** | TEXT | `Timeline_2023-Q3.json` | Original JSON filename |
| **source_segment_index** | INTEGER | `47` | Position in semanticSegments array (0-based) |
| **source_file_hash** | SHA-256 | `b2e4d1...` | SHA-256 of source JSON file |
| **processed_at** | ISO 8601 | `2025-12-15T20:30:00Z` | When this record was processed |
| **processing_version** | TEXT | `v4.2.1` | Software version used |
| **data_quality_score** | REAL | `0.95` | 0.0-1.0 (completeness + accuracy) |
| **forensic_flags** | TEXT | `multi_device_split` | Comma-separated forensic notes |

---

### Section B: Human-Readable Time (15 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **date** | DATE | `2023-09-15` | Calendar date (local time) |
| **day_of_week** | TEXT | `Friday` | Human-readable day |
| **start_time_local** | TIME | `02:17:00 PM` | 12-hour format with AM/PM |
| **end_time_local** | TIME | `02:31:00 PM` | 12-hour format with AM/PM |
| **timezone** | TEXT | `EDT (America/Detroit)` | Timezone name + ID |
| **duration** | TEXT | `0h 14m` | Hours and minutes |
| **duration_seconds** | INTEGER | `840` | For calculations |
| **overnight_stay** | YES/NO | `No` | Flag for overnight (22:00-07:00) |
| **multi_day_span** | YES/NO | `No` | Flag if crosses calendar days |
| **days_duration** | REAL | `0.01` | Decimal days (for long visits) |
| **start_timestamp_utc** | ISO 8601 | `2023-09-15T18:17:00Z` | Forensic UTC timestamp |
| **end_timestamp_utc** | ISO 8601 | `2023-09-15T18:31:00Z` | Forensic UTC timestamp |
| **timestamp_source** | TEXT | `waypoints_first_last` | Where time came from (for paths) |
| **timestamp_fallback** | YES/NO | `No` | TRUE if not from GPS |
| **gap_from_previous_minutes** | REAL | `3.5` | Time gap to previous event |

---

### Section C: Location (Human-Readable) (20 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **location_type** | TEXT | `Residence` or `Business` or `Unknown` | Radar label |
| **place_name** | TEXT | `Matt's House` | Google place name (if exists) |
| **address** | TEXT | `4431 E Mount Morris Rd` | Street address |
| **city** | TEXT | `Mount Morris` | City name |
| **state** | TEXT | `Michigan` | Full state name |
| **state_code** | TEXT | `MI` | Two-letter code |
| **postal_code** | TEXT | `48458` | ZIP code |
| **formatted_address** | TEXT | `4431 E Mount Morris Rd, Mount Morris, MI 48458 US` | Full address |
| **neighborhood** | TEXT | `Downtown` | Neighborhood (if available) |
| **county** | TEXT | `Genesee` | County |
| **latitude** | REAL | `43.120735` | Decimal degrees (6 decimals) |
| **longitude** | REAL | `-83.621357` | Decimal degrees (6 decimals) |
| **google_maps_link** | URL | `https://www.google.com/maps?q=43.120735,-83.621357` | Clickable link |
| **google_place_id** | TEXT | `ChIJAbc123...` | Google Place ID (if exists) |
| **geocode_source** | TEXT | `radar_cache` | Which API provided address |
| **geocode_confidence** | REAL | `0.95` | 0.0-1.0 geocoding confidence |
| **geocode_accuracy_meters** | REAL | `0.4` | Distance from request to result |
| **is_home** | YES/NO | `Yes` | Matched to home_base table |
| **is_work** | YES/NO | `No` | Matched to home_base table |
| **flagged_location** | YES/NO | `No` | In problematic_locations_contacts |

---

### Section D: Activity-Specific (For Activities Only) (10 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **activity_type** | TEXT | `Driving` or `Walking` | Human-readable type |
| **confidence** | REAL | `0.85` | Google's confidence |
| **distance_miles** | REAL | `12.3` | Miles traveled |
| **distance_kilometers** | REAL | `19.8` | Kilometers traveled |
| **distance_source** | TEXT | `google` or `calculated` | Where distance came from |
| **avg_speed_mph** | REAL | `35.2` | Average speed |
| **start_address** | TEXT | `Home` | Starting location |
| **end_address** | TEXT | `Work` | Ending location |
| **waypoint_count** | INTEGER | `47` | Number of GPS breadcrumbs |
| **route_type** | TEXT | `Commute` or `Errand` | Inferred from start/end |

---

### Section E: Visit-Specific (For Visits Only) (8 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **place_category** | TEXT | `Residence` or `Food` | Primary category |
| **place_types** | TEXT | `restaurant, food, establishment` | All Google types |
| **visit_confidence** | REAL | `0.95` | Google's confidence |
| **place_rating** | REAL | `4.2` | Google rating (if business) |
| **semantic_type** | TEXT | `Home` or `Work` | Google's inference |
| **hierarchy_level** | INTEGER | `1` | Visit importance (Google) |
| **is_frequent_location** | YES/NO | `Yes` | Visited 10+ times |
| **total_visits_to_location** | INTEGER | `73` | How many times visited here |

---

### Section F: Path-Specific (For Paths Only) (6 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **path_type** | TEXT | `Driving` or `Walking` | Activity type |
| **waypoints_embedded** | YES/NO | `No` | If waypoints in separate rows |
| **waypoint_count** | INTEGER | `47` | Number of GPS breadcrumbs |
| **multi_device_split** | YES/NO | `Yes` | If split due to multi-device |
| **device_index** | TEXT | `device_0` or `device_1` | Which device (if split) |
| **split_reason** | TEXT | `duplicate_timestamp_>100m` | Why split occurred |

---

### Section G: Analysis & Flags (12 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **schedule_verified** | YES/NO/N/A | `Yes` | If in expected_schedule table |
| **schedule_claimed_location** | TEXT | `Doctor's Office` | What they said |
| **schedule_match** | YES/NO/N/A | `Yes` | If location matches claim |
| **schedule_mismatch_km** | REAL | `0.0` | Distance from claimed location |
| **violation_alert** | YES/NO | `No` | In problematic_locations_contacts |
| **violation_type** | TEXT | N/A | PERSON or LOCATION |
| **violation_severity** | TEXT | N/A | LOW/MEDIUM/HIGH/CRITICAL |
| **overnight_deviation** | YES/NO | `No` | Not at home during overnight hours |
| **impossible_travel** | YES/NO | `No` | >200mph required to reach next location |
| **data_gap_after_minutes** | REAL | `0.0` | Time gap to NEXT event (missing data?) |
| **anomaly_detected** | YES/NO | `No` | Any anomaly flagged |
| **anomaly_type** | TEXT | N/A | Type of anomaly |

---

### Section H: Metadata (Forensic Tracking) (15 columns)

| Column | Type | Example | Notes |
|--------|------|---------|-------|
| **original_json_snippet** | JSON (escaped) | `{"startTime":"2023-09-15T..."}` | First 500 chars of original JSON |
| **parsing_warnings** | TEXT | `missing_place_id` | Comma-separated warnings |
| **parsing_errors** | TEXT | N/A | Critical errors (if any) |
| **enrichment_providers** | TEXT | `radar,google` | Which geocoding providers used |
| **radar_result_uuid** | UUIDv7 | `01924d58...` | Link to geocode_result_radar |
| **google_result_uuid** | UUIDv7 | N/A | Link to geocode_result_google |
| **geocode_disagreement** | YES/NO | `No` | If providers gave different addresses |
| **manual_override** | YES/NO | `No` | If human edited |
| **override_reason** | TEXT | N/A | Why human edited |
| **linked_events** | UUIDs | `01924d59...,01924d5a...` | Related event UUIDs |
| **parent_path_uuid** | UUIDv7 | `01924d58...` | If waypoint, link to path |
| **child_waypoints** | INTEGER | `47` | If path, count of waypoints |
| **created_at** | ISO 8601 | `2025-12-15T20:30:00Z` | When record created |
| **updated_at** | ISO 8601 | `2025-12-15T20:30:00Z` | Last modified |
| **export_version** | TEXT | `master_csv_v2.1` | CSV schema version |

---

## Total: ~100 Columns

**Sections:**
- A: Identity & Chain of Custody (10)
- B: Time (15)
- C: Location (20)
- D: Activity-Specific (10)
- E: Visit-Specific (8)
- F: Path-Specific (6)
- G: Analysis & Flags (12)
- H: Metadata (15)

**Additional columns can be added per case needs**

---

## Presentation Logic: Field Modifications for Human Readability

### 1. **Timestamps → Human Format**
```
Database: "2023-09-15T18:17:00Z" (UTC, ISO 8601)
CSV: "02:17:00 PM EDT" (12-hour, local, with timezone)
PLUS: "2023-09-15" in date column
PLUS: "Friday" in day_of_week column
```

### 2. **Duration → Hours & Minutes**
```
Database: 840 (seconds)
CSV: "0h 14m"
PLUS: Keep duration_seconds for calculations
```

### 3. **Distance → Miles AND Kilometers**
```
Database: 19800 (meters)
CSV: "12.3" (miles) + "19.8" (km) in separate columns
```

### 4. **Coordinates → Address + Link**
```
Database: lat=43.120735, lng=-83.621357
CSV: "4431 E Mount Morris Rd, Mount Morris, MI 48458 US"
PLUS: Google Maps link in separate column
PLUS: Keep lat/lng for forensic verification
```

### 5. **Boolean → YES/NO**
```
Database: TRUE/FALSE/NULL
CSV: "Yes" / "No" / "N/A"
```

### 6. **Semantic Types → Plain English**
```
Database: "IN_PASSENGER_VEHICLE"
CSV: "Driving"
```

### 7. **JSON → Flattened Text**
```
Database: {"types": ["restaurant", "food", "establishment"]}
CSV: "restaurant, food, establishment" (comma-separated string)
```

### 8. **UUIDs → Shortened Display (Optional)**
```
Database: "01924d58-9e7f-72b0-8e6e-3a5f8c2d1b0a"
CSV: "01924d58..." (first 8 chars for display)
PLUS: Full UUID in forensic metadata column
```

---

## Forensic Requirements

### 1. **Record Hash (SHA-256)**
**Purpose:** Prove data integrity, detect tampering

**How it works:**
```
For each row:
1. Concatenate ALL data fields (excluding hash itself)
2. Add source_file_hash + timestamp + processing_version
3. Compute SHA-256 hash
4. Store in record_hash column

To verify:
1. Recompute hash from data fields
2. Compare to stored hash
3. If match → Data unchanged since export
4. If mismatch → Data was altered (tampering detected)
```

**Example:**
```
Data: record_uuid=abc123,date=2023-09-15,start_time=14:30,...
Hash: SHA-256("abc123|2023-09-15|14:30|...")
     = a3f2c8d1e4b5a6f9...
```

### 2. **Link to Original Data**
**Requirements:**
- `source_file` - Which JSON file
- `source_segment_index` - Which array position
- `source_file_hash` - SHA-256 of original JSON
- `original_json_snippet` - First 500 chars of segment

**Verification:**
```
1. Find source_file: "Timeline_2023-Q3.json"
2. Verify file hash matches source_file_hash
3. Go to semanticSegments[source_segment_index]
4. Compare JSON to original_json_snippet
5. Confirms: CSV row came from this exact JSON segment
```

### 3. **Processing Metadata**
**Required fields:**
- `processed_at` - When exported
- `processing_version` - Software version
- `export_version` - CSV schema version
- `enrichment_providers` - Which APIs called
- `parsing_warnings` - Non-fatal issues
- `parsing_errors` - Critical issues

**Why:** Shows exactly HOW the CSV was created (reproducible process)

### 4. **Chain of Custody Trail**
**Full audit:**
```
Timeline_2023-Q3.json (SHA: b2e4d1...)
  ↓ Parsed by: timeline_processor v4.2.1
  ↓ At: 2025-12-15T20:30:00Z
  ↓ Geocoded by: radar (cache) + google (API)
  ↓ Enriched with: home_base, expected_schedule
  ↓ Exported to: master_timeline_2023-Q3.csv
  ↓ At: 2025-12-15T21:00:00Z
  ↓ Export version: master_csv_v2.1
  ↓ Row hash: a3f2c8... (verifiable)
```

**Court use:**
"Your Honor, this CSV was exported from Google Timeline JSON file with SHA-256 hash b2e4d1...
Each row has a verification hash. Opposing counsel can verify data integrity by recomputing hashes.
Original JSON preserved and available for inspection."

---

## Export Generation Logic

### Pseudocode:
```
FOR EACH event IN (visits UNION activities UNION timeline_paths):

    # Section A: Identity
    record_uuid = event.uuid
    record_type = event.type (human-readable)
    source_file = processing_metadata.source_file
    source_segment_index = event.segment_index

    # Section B: Time
    date = event.start_time.date (local timezone)
    day_of_week = event.start_time.strftime("%A")
    start_time_local = event.start_time.strftime("%I:%M:%S %p %Z")
    duration = format_duration(event.duration_seconds)
    overnight_stay = is_overnight(event.start_time, event.end_time)
    timestamp_source = event.path_time_source (if path)

    # Section C: Location
    address = geocode_result.formatted_address
    city = geocode_result.city
    state = geocode_result.state
    google_maps_link = f"https://www.google.com/maps?q={lat},{lng}"
    is_home = (location_uuid IN home_base.location_uuids)
    is_work = (location_uuid IN home_base.location_uuids WHERE base_type='WORK')
    flagged_location = (location_uuid IN problematic_locations_contacts)

    # Section D/E/F: Type-specific
    IF event.type == 'activity':
        distance_miles = event.distance_meters / 1609.34
        avg_speed_mph = (distance_miles / duration_hours)
        waypoint_count = COUNT(waypoints WHERE parent_id=event.uuid)

    # Section G: Analysis
    schedule_verified = (event.uuid IN expected_schedule.actual_visit_ids)
    violation_alert = (location_uuid IN problematic_locations_contacts)

    # Section H: Metadata
    original_json_snippet = event.source_json[:500]
    enrichment_providers = geocode_request.providers_called
    radar_result_uuid = geocode_resolution.radar_result_uuid

    # Compute hash
    all_fields = concatenate(A, B, C, D, E, F, G, H_data)
    record_hash = SHA256(all_fields)

    WRITE TO CSV
```

---

## CSV File Metadata (First Row - Comment)

```
# Master Timeline Export
# Generated: 2025-12-15T21:00:00Z
# Software: Timeline Processor v4.2.1
# Export Schema: master_csv_v2.1
# Source Files: Timeline_2023-Q1.json (SHA256: b2e4d1...), Timeline_2023-Q2.json (SHA256: c3f5e2...)
# Total Records: 12,847
# Date Range: 2023-01-01 to 2023-06-30
# Forensic Hash Algorithm: SHA-256
# Verification: Each row hash can be independently verified
# Chain of Custody: Complete audit trail in processing_metadata table (database)
#
record_uuid,record_hash,record_type,source_file,... (100 columns)
```

---

## Example Row

```csv
record_uuid,record_hash,record_type,date,start_time_local,end_time_local,duration,address,city,state,distance_miles,overnight_stay,is_home,flagged_location,geocode_source,record_type
01924d58-9e7f,a3f2c8d1...,Visit,2023-09-15,02:17:00 PM EDT,02:31:00 PM EDT,0h 14m,"4431 E Mount Morris Rd",Mount Morris,MI,0.0,No,Yes,No,radar_cache,Visit
```

**Human sees:**
- Readable UUID prefix
- Verification hash
- Clear event type
- Date and times in 12-hour format
- Duration in hours/minutes
- Full address
- Miles (not meters)
- Yes/No flags
- Geocode source documented

---

## Summary: Forensic CSV Requirements

✅ **SHA-256 hash per row** - Tamper detection
✅ **Link to original JSON** - source_file + segment_index
✅ **Source file hash** - Verify original unchanged
✅ **Processing metadata** - Software version, timestamp
✅ **JSON snippet** - First 500 chars of source
✅ **Enrichment tracking** - Which APIs called
✅ **Audit trail** - Created_at, updated_at, export_version
✅ **Human-readable** - Dates, times, addresses, YES/NO
✅ **Flattened JSON** - No nested structures
✅ **Click able links** - Google Maps URLs
✅ **Calculation transparency** - Document if value calculated vs provided

**Defensible in court:** Every data point traceable to source, verifiable, documented.
