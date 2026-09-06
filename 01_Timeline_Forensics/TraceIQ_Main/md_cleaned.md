# TraceIQ Schema Export

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` lands with the workspace directory-rename step; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


## DATABASE SCHEMAS (DDL)

### 1. TIMELINE_ENRICHED TABLE

```sql
CREATE TABLE timeline_enriched (
    event_id TEXT PRIMARY KEY,
    serial_id TEXT NOT NULL UNIQUE,
    device TEXT,
    event_type TEXT NOT NULL,
    label TEXT,
    date_us TEXT,
    time_12h TEXT,
    day_of_week TEXT,
    duration TEXT,
    google_place_id TEXT,
    google_place_name TEXT,
    google_place_types TEXT,
    google_address TEXT,
    radar_place_name TEXT,
    radar_place_type TEXT,
    radar_address TEXT,
    address_display TEXT,
    map_link_coords TEXT,
    map_link_place TEXT,
    distance_miles REAL,
    delta_duration TEXT,
    overnight_flag INTEGER,
    overnight_type TEXT,
    address_street TEXT,
    address_city TEXT,
    location_geopair TEXT,
    location_fuzzy TEXT,
    delta_distance_meters REAL,
    delta_duration_seconds INTEGER,
    semantic_type_probability REAL,
    activity_type_probability REAL,
    probability REAL,
    google_api_cache_id TEXT,
    radar_api_cache_id TEXT,
    source_visit_id TEXT,
    source_activity_id TEXT,
    source_path_id TEXT,
    source_memory_id TEXT,
    original_json TEXT,
    data_source TEXT,
    processed_at TIMESTAMP,
    timestamp_utc TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (google_api_cache_id) REFERENCES google_api_cache(cache_id),
    FOREIGN KEY (radar_api_cache_id) REFERENCES radar_api_cache(cache_id)
);
```

### 2. RADAR_API_CACHE TABLE

```sql
CREATE TABLE radar_api_cache (
    cache_id TEXT PRIMARY KEY,
    request_lat REAL NOT NULL,
    request_lng REAL NOT NULL,
    response_lat REAL,
    response_lng REAL,
    geocode_accuracy_meters REAL,
    label TEXT,
    label_type TEXT,
    layer TEXT,
    top_type TEXT,
    types TEXT,
    street_number TEXT,
    street TEXT,
    city TEXT,
    state TEXT,
    state_code TEXT,
    postal_code TEXT,
    formatted_address TEXT,
    place_label TEXT,
    address_label TEXT,
    distance_from_request REAL,
    timezone_id TEXT,
    timezone_name TEXT,
    timezone_code TEXT,
    google_place_id TEXT,
    google_place_id_found INTEGER,
    problematic_poi INTEGER,
    problematic_notes TEXT,
    manually_verified INTEGER,
    custom_label TEXT,
    batch_file TEXT,
    batch_timestamp TEXT,
    api_metadata TEXT,
    request_coords_fuzzy TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3. GOOGLE_API_CACHE TABLE

```sql
CREATE TABLE google_api_cache (
    cache_id TEXT PRIMARY KEY,
    place_id TEXT UNIQUE NOT NULL,
    name TEXT,
    formatted_address TEXT,
    types TEXT,
    location_lat REAL,
    location_lng REAL,
    location_fuzzy TEXT,
    viewport_ne_lat REAL,
    viewport_ne_lng REAL,
    viewport_sw_lat REAL,
    viewport_sw_lng REAL,
    maps_url TEXT,
    api_response_json TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_validated TIMESTAMP
);
```

### 4. VISITS TABLE

```sql
CREATE TABLE visits (
    visit_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    hierarchy_level INTEGER,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    visit_detection_probability REAL,
    semantic_type TEXT,
    semantic_type_probability REAL,
    visit_place_id TEXT,
    visit_geopair TEXT,
    parent_id TEXT,
    memory_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 5. ACTIVITIES TABLE

```sql
CREATE TABLE activities (
    activity_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    activity_type TEXT NOT NULL,
    activity_type_probability REAL,
    distance_meters REAL,
    activity_start_geopair TEXT NOT NULL,
    activity_end_geopair TEXT NOT NULL,
    activity_place_id_start TEXT,
    activity_place_id_end TEXT,
    parent_id TEXT,
    memory_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 6. TIMELINE_PATHS TABLE

```sql
CREATE TABLE timeline_paths (
    path_id TEXT NOT NULL,
    point_id TEXT PRIMARY KEY,
    parent_id TEXT,
    path_type TEXT,
    start_timestamp_raw TEXT,
    end_timestamp_raw TEXT,
    start_timestamp_utc TEXT,
    end_timestamp_utc TEXT,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    path_start_geopair TEXT,
    path_end_geopair TEXT,
    waypoints_count INTEGER,
    point_sequence INTEGER,
    point_geopair TEXT,
    point_timestamp_raw TEXT,
    point_timestamp_utc TEXT,
    multi_device_split INTEGER DEFAULT 0,
    device_index INTEGER,
    split_from_segment INTEGER,
    aligned_activity_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 7. MEMORIES_TRIPS TABLE

```sql
CREATE TABLE memories_trips (
    memory_id TEXT PRIMARY KEY,
    event_serial_id TEXT NOT NULL UNIQUE,
    start_timestamp_raw TEXT NOT NULL,
    end_timestamp_raw TEXT NOT NULL,
    start_timestamp_utc TEXT NOT NULL,
    end_timestamp_utc TEXT NOT NULL,
    start_timezone_offset_minutes INTEGER,
    end_timezone_offset_minutes INTEGER,
    duration_seconds INTEGER,
    trip_distance_from_origin_km INTEGER,
    trip_destination_place_ids TEXT,
    parent_id TEXT,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_source TEXT DEFAULT 'google_timeline',
    meta_json TEXT
);
```

### 8. ENRICHMENT_QUEUE TABLE

```sql
CREATE TABLE enrichment_queue (
    queue_id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    request_lat REAL NOT NULL,
    request_lng REAL NOT NULL,
    request_coords_fuzzy TEXT NOT NULL,
    needs_radar INTEGER DEFAULT 1,
    needs_google INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 5,
    event_date TEXT,
    event_city TEXT,
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP,
    error_message TEXT,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES timeline_enriched(event_id)
);
```

## DATABASE INDEXES

```sql
CREATE INDEX idx_serial_id ON timeline_enriched(serial_id);
CREATE INDEX idx_event_type_serial_id ON timeline_enriched(event_type, serial_id);
CREATE INDEX idx_date_us_time_12h ON timeline_enriched(date_us, time_12h);
CREATE INDEX idx_location_geopair ON timeline_enriched(location_geopair);
CREATE INDEX idx_location_fuzzy ON timeline_enriched(location_fuzzy);
CREATE INDEX idx_address_city ON timeline_enriched(address_city);
CREATE INDEX idx_address_street ON timeline_enriched(address_street);
CREATE INDEX idx_google_place_id ON timeline_enriched(google_place_id);
CREATE INDEX idx_radar_place_name ON timeline_enriched(radar_place_name);
CREATE INDEX idx_google_api_cache_id ON timeline_enriched(google_api_cache_id);
CREATE INDEX idx_radar_api_cache_id ON timeline_enriched(radar_api_cache_id);
CREATE INDEX idx_source_visit_id ON timeline_enriched(source_visit_id);
CREATE INDEX idx_source_activity_id ON timeline_enriched(source_activity_id);
CREATE INDEX idx_source_path_id ON timeline_enriched(source_path_id);
CREATE INDEX idx_source_memory_id ON timeline_enriched(source_memory_id);
CREATE INDEX idx_overnight_flag ON timeline_enriched(overnight_flag);
CREATE INDEX idx_distance_miles ON timeline_enriched(distance_miles);
CREATE INDEX idx_probability ON timeline_enriched(probability);
CREATE INDEX idx_processed_at ON timeline_enriched(processed_at);
CREATE INDEX idx_created_at ON timeline_enriched(created_at);
CREATE INDEX idx_timestamp_utc ON timeline_enriched(timestamp_utc);
CREATE INDEX idx_date_city ON timeline_enriched(date_us, address_city);
CREATE INDEX idx_city_event_type ON timeline_enriched(address_city, event_type);
CREATE INDEX idx_event_type_date ON timeline_enriched(event_type, date_us);
CREATE INDEX idx_fuzzy_coords_date ON timeline_enriched(location_fuzzy, date_us);
CREATE INDEX idx_overnight_date ON timeline_enriched(overnight_flag, date_us);
CREATE INDEX idx_device_serial ON timeline_enriched(device, serial_id);
CREATE INDEX idx_google_cache_place_id ON google_api_cache(place_id);
CREATE INDEX idx_google_cache_location_fuzzy ON google_api_cache(location_fuzzy);
CREATE INDEX idx_radar_cache_fuzzy_coords ON radar_api_cache(request_coords_fuzzy);
CREATE INDEX idx_radar_cache_city ON radar_api_cache(city);
CREATE INDEX idx_radar_cache_google_place_id ON radar_api_cache(google_place_id);
CREATE INDEX idx_radar_cache_batch_id ON radar_api_cache(batch_file);
CREATE INDEX idx_radar_cache_quality ON radar_api_cache(problematic_poi, manually_verified);
```

