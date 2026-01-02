# ADDENDUM: Case Context & Geocoding Services

**Append this to the main prompt before implementation begins.**

---

## A. CASE CONTEXT & PURPOSE

### Legal Proceeding
**Case:** Salem v. Kinzel (2025-53985-DC)  
**Court:** Genesee County Family Court, Michigan  
**Presiding:** Judge Dawn M. Weier  
**Petitioner:** Matt Salem (pro se)

### What We're Building & Why

This timeline processor supports a custody case where **location history proves or disproves parenting time, overnight custody, and child welfare claims**. The output must be:

1. **Court-admissible** — Chain of custody, verifiable hashes, no data manipulation
2. **Human-readable** — Judges and attorneys need dates/times/addresses, not UUIDs and coordinates
3. **Pattern-revealing** — Overnight stays, travel patterns, presence/absence at key locations

### MCL 722.23 Relevance (Michigan Child Custody Factors)

The timeline evidence primarily supports:

| Factor | What Location Data Proves |
|--------|---------------------------|
| **(G)** Mental/physical health | Travel patterns, erratic movements, overnight locations |
| **(J)** Willingness to facilitate relationship | Presence at exchanges, travel to visitation |
| **(K)** Domestic violence indicators | Locations during incidents, flight patterns, safe house visits |

### What We're Looking For

**Overnight Stays:** Who had the child overnight and where? Home vs. elsewhere?

**Consistency:** Does location history match sworn statements about custody time?

**Anomalies:** Multi-device data (two phones?), gaps in coverage, location spoofing artifacts

**Corroboration:** Do timestamps align with text messages, photos, witness statements?

---

## B. GEOCODING SERVICES & VALIDATION

### Available Services (Priority Order)

#### 1. Radar.com (Primary — FREE TIER)
- **Cache exists:** `/home/claude/timeline_zip2/radar_geocoding_master_good.csv` (6.8MB, ~50K records)
- **API:** Reverse geocode, forward geocode, autocomplete
- **Limit:** 100K calls/month free
- **Use for:** All new lookups, bulk processing

```python
import requests

def radar_reverse_geocode(lat: float, lng: float, api_key: str) -> dict:
    """Radar reverse geocoding — check cache first!"""
    resp = requests.get(
        "https://api.radar.io/v1/geocode/reverse",
        params={"coordinates": f"{lat},{lng}"},
        headers={"Authorization": api_key}
    )
    return resp.json()
```

#### 2. Google Places API (Secondary — PAID)
- **Use for:** `placeId` lookups only (Google's own IDs)
- **Cost:** $17/1000 requests
- **Cache aggressively** — same placeId = same result forever

```python
def google_place_details(place_id: str, api_key: str) -> dict:
    """Only for Google placeIds that Radar can't resolve."""
    resp = requests.get(
        "https://maps.googleapis.com/maps/api/place/details/json",
        params={
            "place_id": place_id,
            "fields": "formatted_address,name,geometry,types",
            "key": api_key
        }
    )
    return resp.json()
```

#### 3. Nominatim/OSM (Fallback — FREE)
- **Use for:** Validation cross-check, not primary
- **Rate limit:** 1 req/sec, no bulk

### Geocoding Validation Protocol

**Every address gets validated against multiple sources when forensically significant:**

```python
def validate_address(lat: float, lng: float) -> dict:
    """
    Multi-source validation for court-critical locations.
    Returns confidence score and discrepancy flags.
    """
    results = {}
    
    # Primary: Radar
    results['radar'] = radar_reverse_geocode(lat, lng)
    
    # Cross-check for important locations
    if is_forensically_significant(lat, lng):  # Home, school, custody exchange
        results['nominatim'] = nominatim_reverse(lat, lng)
        
        # Flag discrepancies
        if results['radar']['address'] != results['nominatim']['address']:
            results['discrepancy_flag'] = True
            results['discrepancy_detail'] = "Address mismatch between providers"
    
    return results

def is_forensically_significant(lat: float, lng: float) -> bool:
    """Check if location is near known significant addresses."""
    SIGNIFICANT_LOCATIONS = [
        (43.0125, -83.6875, "Petitioner Home"),
        (43.0234, -83.7012, "Respondent Home"),
        (43.0456, -83.7234, "School"),
        # Add custody exchange points, etc.
    ]
    
    for slat, slng, label in SIGNIFICANT_LOCATIONS:
        if haversine(lat, lng, slat, slng) < 500:  # Within 500m
            return True
    return False
```

### Cache Schema Addition

```sql
-- Add to geocoding_cache table
ALTER TABLE geocoding_cache ADD COLUMN validation_source TEXT;  -- 'radar_only', 'multi_validated'
ALTER TABLE geocoding_cache ADD COLUMN validation_status TEXT;  -- 'confirmed', 'discrepancy', 'pending'
ALTER TABLE geocoding_cache ADD COLUMN secondary_response_json TEXT;  -- Nominatim/Google cross-check
```

---

## C. EXISTING RADAR CACHE

**File:** `radar_geocoding_master_good.csv`  
**Location:** `/home/claude/timeline_zip2/`  
**Size:** 6.8MB  
**Records:** ~50,000 cached lookups

**Schema:**
```
lat,lng,formatted_address,city,state,country,postal_code,place_name,api_called_at
```

**Import into geocoding_cache:**
```python
import csv

def import_radar_cache(csv_path: str, db_conn):
    """Import existing Radar cache to avoid duplicate API calls."""
    with open(csv_path, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            lookup_key = f"{row['lat']},{row['lng']}"
            db_conn.execute("""
                INSERT OR IGNORE INTO geocoding_cache 
                (id, cache_type, lookup_key, formatted_address, city, state, 
                 country, postal_code, api_source, api_called_at)
                VALUES (?, 'radar_reverse', ?, ?, ?, ?, ?, ?, 'radar', ?)
            """, (
                generate_id()[0],
                lookup_key,
                row['formatted_address'],
                row['city'],
                row['state'],
                row['country'],
                row['postal_code'],
                row.get('api_called_at')
            ))
    db_conn.commit()
```

---

## D. INVITATION FOR IMPROVEMENTS

**To the implementing AI:**

You have latitude to suggest improvements. If you recognize patterns or have ideas about:

1. **Additional validation layers** — timestamp verification, spoofing detection
2. **Better anomaly detection** — impossible travel speeds, location jumps
3. **Schema optimizations** — indexes, denormalization for common queries
4. **Export formats** — what attorneys/judges actually need to see
5. **Integration points** — correlating with SMS exports, photos, other evidence
6. **Michigan-specific requirements** — court filing formats, exhibit standards

**Please recommend them.** Don't silently skip improvements because they weren't explicitly requested.

The goal is forensic-grade evidence that helps a father prove his custody case. Technical elegance matters less than court admissibility and human readability.

---

## E. COST CONSTRAINTS

**Available resources:**
- OpenRouter credits (use Qwen3-235B at $0.10/$0.80 per M tokens)
- Supabase free tier (PostgreSQL + PostGIS)
- Radar.com free tier (100K geocoding calls/month)
- Local SQLite for development

**Avoid:**
- Expensive Claude sessions for routine processing
- Google API calls when Radar works
- Redundant LLM calls — batch and cache everything

---

## F. FILE LOCATIONS

| Asset | Path |
|-------|------|
| Validated SQLite schema | `/home/claude/timeline_zip2/schema.sql` |
| Validation report | `/home/claude/timeline_zip2/VALIDATION_REPORT.md` |
| Radar geocoding cache | `/home/claude/timeline_zip2/radar_geocoding_master_good.csv` |
| Previous agent spec | `/home/claude/timeline_handoff/AGENT_PROMPT.md` |
| Precision utility | `/home/claude/timeline_handoff/precision.py` |
| Main prompt | `/mnt/user-data/outputs/google_timeline_forensic_processor_prompt.md` |

---

*End of Addendum — Append to main prompt before use.*
