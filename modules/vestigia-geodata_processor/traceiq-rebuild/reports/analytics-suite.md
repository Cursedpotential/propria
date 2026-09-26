# TraceIQ — Extended Analytics Suite

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


## Multi-device signals

Same event, same timestamp, positions >100m apart (audited parameter) — physically
impossible for one device. 1,124 signal pairs.

```sql multidevice
select serial_display, to_char(ts_utc at time zone 'America/Detroit', 'YYYY-MM-DD HH12:MI AM') as eastern,
       separation_m
from analysis.multi_device_signals limit 100
```

{% table data="multidevice" rows=15 /%}

## Bouncy trips (travel far out of proportion to displacement)

```sql bouncy
select serial_display, to_char(start_eastern, 'YYYY-MM-DD HH12:MI AM') as start,
       distance_m_google as traveled_m, distance_m_haversine as displaced_m, bounce_ratio
from analysis.bouncy_trips limit 100
```

{% table data="bouncy" rows=15 /%}

## Weekly rhythm

```sql rhythm
select dow, hour, sum(events) as events from analysis.hourly_patterns group by 1,2
```

{% heat_grid data="rhythm" x="hour" y="dow" value="events" title="Events by day-of-week × hour (Eastern)" /%}

## Recurring routes (≥3 trips)

```sql routes
select start_lat_r4, start_lng_r4, end_lat_r4, end_lng_r4, trips, avg_google_m,
       first_trip, last_trip
from analysis.route_patterns order by trips desc limit 50
```

{% table data="routes" rows=15 /%}

## All-touch location frequency

```sql freq
select lat_r4, lng_r4, touches, visits, overnight_touches, first_seen, last_seen
from analysis.location_frequency order by touches desc limit 100
```

{% map title="Every touched location (all event types)" height=500 %}
    {% point_layer data="freq" lat="lat_r4" lng="lng_r4" size_value="touches"
       color_value="overnight_touches" tooltip=["touches","visits","overnight_touches","first_seen","last_seen"] /%}
{% /map %}
