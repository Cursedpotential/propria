PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
CREATE TABLE IF NOT EXISTS timeline_master (
  event_uuid TEXT PRIMARY KEY,
  event_id TEXT,
  event_type TEXT,
  start_time TEXT, end_time TEXT, point_time TEXT,
  start_latlng TEXT, end_latlng TEXT,
  path_time_fallback INTEGER, path_time_source TEXT
);
CREATE TABLE IF NOT EXISTS event_geokey (
  event_uuid TEXT PRIMARY KEY,
  start_latlng_exact TEXT, end_latlng_exact TEXT,
  start_lat_r3 REAL, start_lng_r3 REAL, start_lat_r4 REAL, start_lng_r4 REAL,
  start_geohash8 TEXT, start_geohash9 TEXT,
  end_lat_r3 REAL, end_lng_r3 REAL, end_lat_r4 REAL, end_lng_r4 REAL,
  end_geohash8 TEXT, end_geohash9 TEXT,
  point_lat_r3 REAL, point_lng_r3 REAL, point_lat_r4 REAL, point_lng_r4 REAL,
  point_geohash8 TEXT, point_geohash9 TEXT,
  FOREIGN KEY(event_uuid) REFERENCES timeline_master(event_uuid)
);
CREATE TABLE IF NOT EXISTS location_key (
  location_uuid TEXT PRIMARY KEY,
  latlng_exact TEXT NOT NULL UNIQUE,
  lat_r5 REAL, lng_r5 REAL, lat_r4 REAL, lng_r4 REAL,
  geohash8 TEXT, geohash9 TEXT
);
CREATE TABLE IF NOT EXISTS geocode_request (
  request_uuid TEXT PRIMARY KEY,
  location_uuid TEXT,
  latlng_r4 TEXT, latlng_r5 TEXT, latlng_exact6 TEXT,
  requested_at TEXT DEFAULT CURRENT_TIMESTAMP,
  status TEXT,
  FOREIGN KEY(location_uuid) REFERENCES location_key(location_uuid)
);
CREATE TABLE IF NOT EXISTS geocode_result_google (
  result_uuid TEXT PRIMARY KEY, request_uuid TEXT,
  place_id TEXT, formatted_address TEXT, confidence REAL, bounds TEXT, raw_json TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_google_place ON geocode_result_google(place_id);
CREATE TABLE IF NOT EXISTS geocode_result_radar (
  result_uuid TEXT PRIMARY KEY, request_uuid TEXT,
  place_id TEXT, formatted_address TEXT, confidence REAL, bounds TEXT, raw_json TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_radar_place ON geocode_result_radar(place_id);
CREATE TABLE IF NOT EXISTS geocode_resolution (
  event_uuid TEXT PRIMARY KEY,
  preferred_provider TEXT, result_uuid TEXT, distance_m REAL,
  disagreement_flag INTEGER, tie_break_reason TEXT, resolved_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS geocode_audit (
  audit_id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_uuid TEXT, action TEXT, details TEXT, created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
