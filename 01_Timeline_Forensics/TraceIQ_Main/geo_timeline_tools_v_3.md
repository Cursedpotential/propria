# Geo/Timeline Tools v3.4 — Handoff & README

> Radar‑first reverse geocoding + paths, Google fallback (Place IDs only), hashed UUIDs, 15 m location de‑dupe, column mapper, durations, cache+logs, and extractors.

---

## 0) What this does
- **Processes any sheet** with time/coord columns using a **mapping sidebar**.
- Builds **time‑based Event IDs** and **hashed UUIDs** (v5‑style from SHA‑256).
- Computes **durations** (HH:MM) from Start/End.
- **Sanitizes addresses** to **street‑only** (no state/zip/country; MI/apt OK).
- Creates **coordinate map links** _or_ **hidden‑style Google PlaceID links** (clickable but not visibly underlined).
- **Radar is the default** for reverse geocode and path work; **Google is fallback** for PlaceIDs and only when needed.
- **15 m rounding** for cache keys to cut redundant API calls.
- **Caches** results to a sheet + AppCache; **logs** every call; **import/export** cache to/from Drive.
- **Extracts**: optional split sheets for **Visits**, **Paths**, **Activities**; highlights **overnights**.

---

## 1) Files in the Apps Script project
- **Code.gs** – all logic (menus, sidebar bridge, processors, API calls, caching, links, durations, UUIDs).
- **MappingSidebar.html** – sidebar UI that scans headers and lets you map keys to columns.

> The current code in canvas is synced as: **“Geo Tools Apps Script – Mapping Ui, Radar‑first Geocoding, Google Fallback, Uuids, Durations, Caches, And Logs”**.

---

## 2) Quick start (first run)
1. **Paste both files** into a new Apps Script project bound to your target Google Sheet.
2. Reload the sheet → menu **GeoTools** appears.
3. **GeoTools → Set API Keys** and paste:
   - Radar **Publishable** API key (Authorization header).
   - Google Maps API key (for PlaceID fallback only).
4. **GeoTools → Open Mapping Sidebar** → map required keys (see §3).
5. Click **Run Processor** or use **GeoTools → Process Active Sheet**.
6. Switch link style any time with **GeoTools → Switch Link Mode (Coords/PlaceID)**.

---

## 3) Column mapping keys (what to map)
Required:
- `startTime` – start datetime.
- `endTime` – end datetime.
- `lat` – latitude.
- `lng` – longitude.

Optional (outputs can be new or existing columns):
- `address` – a full address column; will be sanitized to street‑only when written to `streetOut`.
- `placeId` – if present and empty, the tool will fill from Google fallback.
- `durationOut` – writes HH:MM from start/end.
- `eventIdOut` – writes `evt_YYYYMMDD_HHMMSS‑YYYYMMDD_HHMMSS_lat,lng`.
- `uuidOut` – writes hashed UUID (v5‑style from SHA‑256 of start|end|lat|lng|placeId|address).
- `coordLinkOut` – writes `https://www.google.com/maps?q=lat,lng`.
- `placeLinkOut` – writes the raw PlaceID but **makes it a clickable link** (underline removed) to `https://www.google.com/maps/search/?api=1&query_place_id=...`.
- `streetOut` – writes street‑only address.

The **MAPPINGS** sheet mirrors your saved mapping for quick reference.

---

## 4) Rounding & de‑dupe (15 m)
- Reverse‑geocode/cache keys are **quantized** to about **15 meters** to minimize duplicate calls.
- Implementation: lat/lng are rounded to a grid (~15 m cell size) before building the cache key: `RADAR_REV:<lat_q>,<lng_q>`.
- Event IDs keep 5‑dp for readability; cache keys use the 15 m quantization.

> Why 15 m? Typical phone accuracy is 5–30 m; 15 m reduces redundant hits while preserving place‑level fidelity.

---

## 5) Address rules (street‑only)
- From Radar response, compose `number + street` when available.
- If we only have a formatted string, strip: **state, ZIP, country**. Keep apt/unit if included on first line.
- Examples:
  - `6904 Yorkshire Dr, Flint, MI 48505 US` → `6904 Yorkshire Dr`
  - `20 Jay St Apt 3, Brooklyn, NY 11201 USA` → `20 Jay St Apt 3`

---

## 6) Durations & IDs
- **Duration**: rounded to minutes, `HH:MM` (floor hours, zero‑padded minutes).
- **Time‑based Event ID**: `evt_<startStamp>_<endStamp>_<lat,lng@5dp>`.
- **Hashed UUID**: SHA‑256 → 16 bytes with v5/variant bits set; stable for the same inputs.

---

## 7) Links
- **Coordinate link**: standard Google maps query.
- **PlaceID link**: cell shows the **PlaceID text** but is **clickable**; underline removed via RichText style.
- Toggle globally with **GeoTools → Switch Link Mode** (stored in Document Properties).

---

## 8) Radar first, Google fallback
- **Radar**: `/v1/geocode/reverse?coordinates=LAT,LNG&layers=address,place` (Publishable key).
- **Google**: Geocoding used **only** to fetch a **PlaceID** when missing; no primary geocoding.

Mismatches:
- The processor can flag rows where **Radar place/address** and **Google PlaceID** disagree. (See §11 Backlog – “Mismatch flag”.)

---

## 9) Caching & logs
- **Fast cache**: `CacheService` (6h TTL) for active session speed.
- **Sheet cache**: `API_CACHE` sheet (never auto‑expires); key → request → response JSON.
- **Call log**: `API_CALL_LOG` (timestamp, service, endpoint, cache‑hit, ok, request, note).
- **Export**: **GeoTools → Export sheet cache to Drive (JSONL)** → folder `GeoTools_API_Logs`.
- **Import**: **GeoTools → Import JSON cache by File ID** (accepts JSON or JSONL; parses Google Timeline chunk arrays too).

Portability:
- All mapping/config is stored in **Document Properties** + headers, so the project works on any copied sheet. Use **File → Make a copy** of the spreadsheet; the bound script and the cache/log sheets travel with it.

---

## 10) Extractors & highlights
(Enable via mapping + menu run; outputs new tabs when requested.)
- **Visits**: one row per stop (dwell), with start/end, duration, street, PlaceID, and lat/lng.
- **Paths**: segments between visits; optional Radar **route/match** to snap to road geometry; distance/duration if requested.
- **Activities**: optional roll‑ups grouped by day/place/activity.
- **Overnights**: any visit crossing local midnight is highlighted.

> Radar endpoints used for path work in the advanced view: `/route/match`, `/route/directions`, `/route/distance`.

---

## 11) Exports (Kepler.gl, Google Earth)
- **Kepler.gl**: Export **CSV/GeoJSON** layers (Visits points, Paths line strings). Ready to drag‑drop into kepler.gl.
- **Google Earth**: Export **KML** with folders (Visits, Paths) and stylized icons/lines.
- Advanced presets (Kepler): day/night coloration, speed buckets, dwell heat, and per‑segment tooltips.

---

## 12) Sidebar & power‑user formulas
**Sidebar** (MappingSidebar.html):
- “Save Mapping” stores to Document Properties + **MAPPINGS** sheet.
- “Run Processor” executes the end‑to‑end pass.

**Custom functions (can be called directly in cells):**
- `=RADAR_REVERSE_GEOCODE("LAT,LNG")` → street‑only via Radar.
- `=GOOGLEMAPS_REVERSE_GEOCODE("LAT,LNG")` → street‑only via Google (fallback/debug only).
- `=PLACEID_LOOKUP(A2)` → Google PlaceID for an address.

**Duration in native sheets (optional)**: if start/end are proper datetimes, `=TEXT([end]-[start],"h:mm")`.

---

## 13) Mismatch flag (Radar vs Google)
- Planned column: **`Radar/Google Mismatch`** with states: `OK`, `DIFF_PLACEID`, `DIFF_STREET`, `NO_DATA`.
- Logic: compare sanitized street from Radar vs. Google reverse; or Radar placeLabel vs. PlaceID lookup result’s name (when available).

---

## 14) Path maps (Radar) – how we build them
- **Default**: use Radar **route/match** to snap raw points to roads; return matched polyline and attributes.
- Optional **directions** or **optimize** endpoints to compute door‑to‑door or multi‑stop legs.
- Stored per path segment in **Paths** sheet; can export to Kepler/GeoJSON.

---

## 15) Sheets created/used
- `CONFIG` – key/value store (e.g., link mode).
- `MAPPINGS` – your column mapping legend.
- `API_CACHE` – persistent cache rows.
- `API_CALL_LOG` – every API hit.
- `VISITS`, `PATHS`, `ACTIVITIES` – created when you run extractors.

---

## 16) Security, quotas, robustness
- Radar **Publishable** key is required (reverse geocode, routing). Respect rate limits (10 rps for reverse, etc.).
- Google key is used **sparingly** for PlaceID fallback.
- All network calls are wrapped; failures log to `API_CALL_LOG` and do not crash the entire run.

---

## 17) Troubleshooting
- **Missing API key** → use **Set API Keys**.
- **No mapping saved** → open sidebar, map required keys, click **Save**.
- **No output columns** → map outputs or let the tool create new columns by naming them in the mapper.
- **PlaceID links not clickable** → ensure the cell text is just the PlaceID; the processor re‑applies RichText linking each run.
- **Cache not hit** → confirm 15 m quantization is consistent (same rounded lat/lng) and check `API_CACHE`.

---

## 18) Backlog / Next
- **JSON Importer UI** (Drive picker + path‑mapper) to create a sheet from raw timelines.
- **Radar/Google mismatch flag** surfaced in UI and extract sheets.
- **Year‑partitioned human‑readable views** (one sheet per year; summary dashboard).
- **One‑click exports**: Kepler presets, KML themes, and per‑activity bundles.
- **SQLite‑style cache on Drive** (emulated): continue with JSONL + sheet cache for Apps Script; offer optional external SQLite via Apps Script WebApp proxy if needed.

---

## 19) Ops runbook (repeatable steps)
1. Open the target spreadsheet.
2. GeoTools → **Open Mapping Sidebar** → verify mapping.
3. GeoTools → **Process Active Sheet**.
4. (Optional) GeoTools → **Export sheet cache to Drive**.
5. (Optional) Re‑run extractors to refresh **Visits/Paths/Activities**.

---

## 20) Owner’s notes
- Project name: **Geo/Timeline Tools (v3.4)**.
- Radar is the **default** for all geospatial work; Google is **fallback** for PlaceID and used for **place links**.
- Keep this README pinned as the first sheet for humans; the mapping sidebar is your friend.

