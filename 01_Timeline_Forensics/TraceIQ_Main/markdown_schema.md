# TraceIQ Markdown Schema Guide

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` lands with the workspace directory-rename step; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


This file describes the recommended structure for Markdown files that contain SQL schema, indexes, and views to be imported by the TraceIQ app.

Principles
- Only fenced code blocks explicitly labeled with `sql` will be executed by the importer. Example:

```sql
CREATE TABLE example (...);
```

- Non-SQL code (JavaScript, Python, Apps Script, examples) should use other language fences (```js, ```python, ```text) so they are ignored.
- Keep one logical DDL statement per fenced block where possible. Use semicolons (`;`) inside a block as needed.
- Provide sections and headings to organize tables, indexes, and views. Use these recommended headings:
  - `## DATABASE SCHEMAS (DDL)`
  - `## DATABASE INDEXES`
  - `## ANALYTICAL VIEWS`

Recommended file template

````markdown
# Project Schema Export

## DATABASE SCHEMAS (DDL)

### 1. my_table

```sql
CREATE TABLE my_table (
  id TEXT PRIMARY KEY,
  name TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## DATABASE INDEXES

```sql
CREATE INDEX idx_my_table_name ON my_table(name);
```

## ANALYTICAL VIEWS

```sql
CREATE VIEW vw_my_summary AS
SELECT name, COUNT(*) as cnt FROM my_table GROUP BY name;
```
````

Tips
- If you have multiple related CREATE statements, group them in separate `sql` fences in the order you want them executed. The importer will categorize tables, then indexes, then views.
- For safe testing, use the importer `dry` flag to validate without committing changes (POST `dry=true`).
- If you need to include non-executable examples or helper scripts, mark them with their appropriate language fences to avoid accidental execution.

Automated conversions
- If you have an old markdown file with unlabeled fences, consider running a conversion script to add `sql` labels to fences that contain SQL DDL. The TraceIQ app previously supported heuristics but now requires explicit `sql` fences for safety and predictability.

This template and rules should make future markdown schema files consistent and safe to import into TraceIQ.
