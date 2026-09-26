# Data Visualization & Ingestion UX

Bar: categorical. Line: trends. Tables: precise/dense values. Scatter: relationships.
Strip decorative chrome — every visual element should carry information.
Tables: monospace numerals, right-aligned numbers, frozen headers, zebra striping.
Ingestion pipelines need a dedicated monitoring surface (source → transform → load) with drill-down.
Status (queued/processing/indexed/failed) must be persistently visible without navigating away.
Pair every anomaly with a direct action (retry/investigate/resolve).
Every chart/table must support CSV/JSON export.
Health should be readable in under 3 seconds via position/size/color alone.
