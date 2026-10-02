# TraceIQ — Places & Home-Base Candidates

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


```sql places
select place_id, lat_r4, lng_r4, visit_count, overnight_visits, overnight_pct,
       avg_minutes, first_visit, last_visit,
       'https://www.google.com/maps?q=' || lat_r4 || ',' || lng_r4 as map_link
from analysis.place_analytics
order by visit_count desc
```

{% big_value data="places" value="count(*)" title="Distinct places" /%}
{% big_value data="places" value="sum(overnight_visits)" title="Total overnight stays" /%}

## Home-base candidates (high overnight share)

The residence story without any geocoding: places ranked by overnight stays.
Labels pending owner confirmation → `ref.home_base` (ADR-0012).

```sql homebase_candidates
select * from (
  select place_id, lat_r4, lng_r4, visit_count, overnight_visits, first_visit, last_visit
  from analysis.place_analytics
  where overnight_visits >= 10
) t order by overnight_visits desc
```

{% table data="homebase_candidates" rows=12 /%}

{% map title="Overnight-heavy places (candidate residences)" height=500 %}
    {% point_layer data="homebase_candidates" lat="lat_r4" lng="lng_r4"
        size_value="overnight_visits" color_value="overnight_visits"
        color_palette=["#fef3c7", "#fcd34d", "#f59e0b", "#b45309", "#7c2d12"]
        tooltip=["place_id", "visit_count", "overnight_visits", "first_visit", "last_visit"] /%}
{% /map %}

## All places

{% table data="places" search=true rows=25 /%}
