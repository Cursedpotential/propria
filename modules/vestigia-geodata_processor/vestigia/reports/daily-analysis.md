# TraceIQ — Daily Analysis

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


{% range_calendar id="day_range" /%}

## Activity calendar

```sql daily
select day, events, visits, activities
from analysis.daily_coverage
where day {{day_range.between}}
order by day
```

{% calendar_heatmap data="daily" date="day" value="events" title="Events per day" /%}

## Day-by-day detail

```sql day_events
select serial_display as id, raw_type,
       to_char(start_eastern, 'Dy MM/DD HH12:MI AM') as starts,
       to_char(end_eastern, 'HH12:MI AM') as ends,
       round(extract(epoch from end_utc - start_utc)/60) as minutes,
       case when overnight_simple then 'YES' else '' end as overnight,
       place_id, probability,
       'https://www.google.com/maps?q=' || lat || ',' || lng as map_link
from analysis.latest_events
where start_eastern::date {{day_range.between}}
order by start_utc
```

{% table data="day_events" search=true rows=25 %}
    {% dimension id="raw_type" /%}
{% /table %}

## Overnights in range

```sql overnights
select serial_display, to_char(start_eastern, 'YYYY-MM-DD Dy HH12:MI AM') as start,
       hours, place_id, lat, lng
from analysis.overnight_events
where start_eastern::date {{day_range.between}}
order by start_eastern
```

{% table data="overnights" rows=15 /%}

{% map title="Overnight locations in range" height=420 %}
    {% point_layer data="overnights" lat="lat" lng="lng"
       tooltip=["serial_display", "start", "hours"] /%}
{% /map %}
