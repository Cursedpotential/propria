# Database Viewing, Integration & Productivity Workspace UX

Dedicated data-explorer panel: schema/table tree + dense results grid, dockable like any other panel.
Support both a visual query builder and a raw query console (SQL/SurrealQL), togglable without losing state.
Results grids follow the same dense-table conventions: monospace numerals, frozen headers, CSV/JSON export.
Vector/semantic search results show similarity scores and let users drill back to the source document.
Show query execution metadata (row count, latency, source database) in a persistent status strip.
Label per-field/per-panel data source when blending multiple backends.
Provide a connections panel with live per-database health (connected/degraded/offline) + last-sync time.
Let users save cross-database queries as first-class, shareable workspace objects.
Paginate/virtualize large result sets; debounce and visibly indicate searching states.
