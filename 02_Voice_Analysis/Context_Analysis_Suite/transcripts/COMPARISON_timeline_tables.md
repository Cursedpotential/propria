# Timeline Table Field Comparison

## Core Timeline Tables - Side by Side

| Field Name | timeline_events (SRC) | timeline_master (ARCHIVE) | visits (CONVERSATION) | activities (CONVERSATION) | timeline_paths (CONVERSATION) |
|------------|----------------------|---------------------------|----------------------|--------------------------|------------------------------|
| **Primary Key** | event_id (TEXT) | event_uuid (TEXT) | visit_id (UUIDv7?) | activity_id (UUIDv7?) | path_id (UUIDv7?) |
| **Event Type** | event_type (TEXT) | event_type (TEXT) | N/A (implicit: visit) | N/A (implicit: activity) | N/A (implicit: path) |
| **Segment Index** | segment_index (INT) | ❌ | ✓ (likely) | ✓ (likely) | ✓ (likely) |
| **Start Time** | start_time (TEXT) | start_time (TEXT) | start_time | start_time | start_time |
| **End Time** | end_time (TEXT) | end_time (TEXT) | end_time | end_time | end_time |
| **Point Time** | ❌ | point_time (TEXT) | ❌ | ❌ | ✓ (for waypoints) |
| **Timezone Offset (start)** | start_tz_offset (INT) | ❌ | ✓ (likely) | ✓ (likely) | ✓ (likely) |
| **Timezone Offset (end)** | end_tz_offset (INT) | ❌ | ✓ (likely) | ✓ (likely) | ✓ (likely) |
| **Lat/Lng Format** | Separate columns (REAL) | Combined string (TEXT) | Unknown | Unknown | Unknown |
| **Visit: Hierarchy Level** | visit_hierarchy_level (INT) | ❌ | ✓ (likely) | N/A | N/A |
| **Visit: Probability** | visit_probability (REAL) | ❌ | ✓ (likely) | N/A | N/A |
| **Visit: Place ID** | visit_place_id (TEXT) | ❌ | place_id | N/A | N/A |
| **Visit: Place Name** | visit_place_name (TEXT) | ❌ | place_name | N/A | N/A |
| **Visit: Semantic Type** | visit_semantic_type (TEXT) | ❌ | semantic_type | N/A | N/A |
| **Visit: Place Probability** | visit_place_probability (REAL) | ❌ | ✓ (likely) | N/A | N/A |
| **Visit: Latitude** | visit_latitude (REAL) | ❌ | latitude | N/A | N/A |
| **Visit: Longitude** | visit_longitude (REAL) | ❌ | longitude | N/A | N/A |
| **Activity: Type** | activity_type (TEXT) | ❌ | N/A | activity_type | N/A |
| **Activity: Probability** | activity_probability (REAL) | ❌ | N/A | probability | N/A |
| **Activity: Distance (m)** | activity_distance_meters (REAL) | ❌ | N/A | distance_meters | N/A |
| **Activity: Start Lat** | activity_start_latitude (REAL) | ❌ | N/A | start_latitude | N/A |
| **Activity: Start Lng** | activity_start_longitude (REAL) | ❌ | N/A | start_longitude | N/A |
| **Activity: End Lat** | activity_end_latitude (REAL) | ❌ | N/A | end_latitude | N/A |
| **Activity: End Lng** | activity_end_longitude (REAL) | ❌ | N/A | end_longitude | N/A |
| **Path: Time Fallback** | ❌ | path_time_fallback (INT) | ❌ | ❌ | ✓ (forensic tracking) |
| **Path: Time Source** | ❌ | path_time_source (TEXT) | ❌ | ❌ | ✓ (forensic tracking) |
| **Path: Waypoints** | ❌ (separate waypoints table) | ❌ | ❌ | ❌ | waypoints (array or JSON?) |
| **Multi-Device: Is Split** | is_split (BOOL) | ❌ | ❌ | ❌ | multi_device_split (BOOL) |
| **Multi-Device: Split Reason** | split_reason (TEXT) | ❌ | ❌ | ❌ | split_reason (TEXT) |
| **Multi-Device: Device Index** | ❌ | ❌ | ❌ | ❌ | device_index (0 or 1) |
| **Created At** | created_at (TEXT) | ❌ | created_at | created_at | created_at |

## Column Counts
- **timeline_events (SRC):** 23 columns - Unified table for visits + activities
- **timeline_master (ARCHIVE):** 9 columns - Minimal schema
- **visits (CONVERSATION):** ~17 columns - Separate visit-only table
- **activities (CONVERSATION):** ~18 columns - Separate activity-only table
- **timeline_paths (CONVERSATION):** ~24 columns - Separate paths with waypoints

## Recommendation: **Use CONVERSATION schema (separate tables)**

### Why:
1. **Cleaner separation** - Each event type has its own focused table
2. **Easier queries** - No need to filter by event_type constantly
3. **Better performance** - Smaller tables, focused indexes
4. **Forensic clarity** - Path timestamp tracking (path_time_source) documents source for court
5. **Multi-device support** - Explicit device_index and split tracking in timeline_paths

### What to add from SRC:
- `segment_index` - Track position in semanticSegments array
- `start_tz_offset` / `end_tz_offset` - Timezone offsets for proper local time
- `is_split` / `split_reason` - Multi-device tracking (if not already in conversation schema)
- `created_at` - Processing timestamp

### What to add from ARCHIVE:
- `path_time_fallback` - Boolean flag for forensic documentation
- `path_time_source` - Document where timestamp came from ('points_first_last', 'adjacent_events', 'container_fallback')

---

# Waypoints Table Comparison

| Field Name | waypoints (SRC) | timeline_paths waypoints (CONVERSATION) |
|------------|-----------------|----------------------------------------|
| **Primary Key** | id (INTEGER AUTOINCREMENT) | ✓ (UUIDv7 or part of path array?) |
| **Parent ID** | parent_id (TEXT FK) | Implicit (part of path) |
| **Parent Type** | parent_type (TEXT) | Implicit (always path) |
| **Sequence** | sequence (INTEGER) | ✓ (CRITICAL - must order by this, not timestamp) |
| **Timestamp** | timestamp (TEXT) | point_time |
| **Latitude** | latitude (REAL) | latitude |
| **Longitude** | longitude (REAL) | longitude |
| **Multi-Device: Split** | multi_device_split (BOOL) | ✓ (if path is split) |
| **Multi-Device: Device Index** | device_index (0 or 1) | ✓ (inherited from path) |
| **Split From Segment** | split_from_segment (INT) | ✓ (track original segment) |
| **Segment Index** | segment_index (INT) | ✓ (for traceability) |
| **Created At** | created_at (TEXT) | ✓ |

## Waypoints Architecture Decision:

**Option A: Separate waypoints table (like SRC)**
- Pros: Cleaner schema, easier to query individual points
- Cons: Extra JOIN needed to get path + waypoints

**Option B: Embedded in timeline_paths (JSONB array or TEXT)**
- Pros: Single query gets full path, atomic updates
- Cons: Harder to query individual waypoints, larger row size

**Recommendation:** Use **separate waypoints table** (like SRC) with UUIDv7 primary key

---

# All Tables - Final Recommendation

## Core Timeline Tables (4)
1. **visits** - 17+ columns, UUIDv7 PK
2. **activities** - 18+ columns, UUIDv7 PK
3. **timeline_paths** - 24+ columns, UUIDv7 PK
4. **waypoints** - 12+ columns, UUIDv7 PK, FK to timeline_paths

## Geocoding Tables (8) - from ARCHIVE + enhanced for 3 sources
5. **location_key** - Deduplicated location registry
6. **event_geokey** - Multi-precision coordinate rounding
7. **geocode_request** - Request tracking (all sources)
8. **geocode_result_google** - Google Places cache
9. **geocode_result_radar** - Radar cache
10. **geocode_result_geodata** - NEW: Geodata cache (3rd source)
11. **geocode_resolution** - Multi-source resolution
12. **geocode_audit** - Audit trail

## Enriched/Analysis Tables (6)
13. **timeline_enriched** - 36 columns, human-readable master timeline
14. **home_base** - Auto-detected locations
15. **expected_schedule** - Claims vs reality
16. **problematic_locations_contacts** - Violation tracking
17. **vehicle_trips** - Vehicle time tracking
18. **vehicle_time_summary** - Aggregated stats

## Operational Tables (3)
19. **processing_metadata** - Processing runs
20. **parse_errors** - Error tracking
21. **schema_version** - Version tracking

**Total: 21 tables**

---

# UUID7 Requirements

## Tables that MUST use UUIDv7:
- visits (visit_id)
- activities (activity_id)
- timeline_paths (path_id)
- waypoints (waypoint_id) - **OR** (path_id + sequence) composite key
- timeline_enriched (event_id)
- home_base (homebase_id)
- expected_schedule (schedule_id)
- problematic_locations_contacts (entity_id)
- vehicle_trips (trip_id)
- vehicle_time_summary (summary_id)
- All geocoding tables (location_uuid, request_uuid, result_uuid, etc.)

## Tables that can use INTEGER AUTOINCREMENT:
- processing_metadata (operational)
- parse_errors (operational)
- schema_version (operational)

---

**Next Step:** Get full CREATE TABLE statements from conversation for visits, activities, timeline_paths, and all analysis tables
