# TraceIQ — Anomalies & Data Quality

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


## Impossible-speed waypoints

Threshold is the audited `impossible_speed_mph` parameter (currently 120). These are
listed, never silently removed — most are GPS glitches; each is reviewable.

```sql speed
select serial_display, to_char(ts_utc at time zone 'America/Detroit', 'YYYY-MM-DD HH12:MI AM') as eastern,
       mph, gap_prev_s
from analysis.speed_anomalies order by mph desc
```

{% table data="speed" rows=20 /%}

## Unrecorded-stop candidates (waypoint-gap analysis)

GPS quiet ≥ `dwell_threshold_min` (15) while moving <100m — a stop Google never wrote down.

```sql dwell
select serial_display, eastern, lat, lng, gap_minutes
from analysis.dwell_stops order by gap_minutes desc limit 100
```

{% table data="dwell" rows=20 /%}

## Data gaps (missing months)

```sql gap_months
with span as (
  select generate_series(date_trunc('month', min(start_eastern)),
                         date_trunc('month', max(start_eastern)),
                         '1 month')::date as m
  from analysis.latest_events)
select to_char(s.m, 'YYYY-MM') as missing_month
from span s
left join analysis.monthly_coverage mc on to_char(s.m, 'YYYY-MM') = mc.month
where mc.month is null
order by s.m
```

{% table data="gap_months" rows=15 /%}

{% callout status="warning" %}
Every missing month needs a documented explanation (device change, tracking off,
export scope, Google deletion) in `raw.data_loss_event` before exhibits ship.
{% /callout %}

## Google's own confidence

```sql conf
select raw_type, round(avg(probability), 3) as avg_confidence,
       count(*) filter (where probability < 0.5) as below_half
from analysis.latest_events where probability is not null group by 1
```

{% table data="conf" /%}

{% callout status="info" %}
Activity-type confidence is 0.0 in this export vintage (Google's data, verified) —
never present activity *type* (driving vs walking) as attested. Times, distances,
coordinates are unaffected.
{% /callout %}
