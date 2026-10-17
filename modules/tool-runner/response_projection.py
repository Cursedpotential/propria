"""Use transient DuckDB JSON queries for bounded tool discovery and clean result envelopes.
Inputs: caller-authorized native metadata/results. Outputs: selected rows or compact JSON.
Effects: in-memory query processing only; no persistent database, source edits or shell execution.
"""
import json
import duckdb


def catalog_page(tools,query,offset,limit):
    """Filter and paginate metadata in DuckDB without returning every tool schema to an agent."""
    with duckdb.connect(':memory:') as db:
        values=json.dumps(tools,ensure_ascii=False)
        sql="""SELECT value FROM json_each(CAST(? AS JSON))
          WHERE contains(lower(coalesce(json_extract_string(value,'$.name'),'') || ' ' ||
             coalesce(json_extract_string(value,'$.title'),'') || ' ' ||
             coalesce(json_extract_string(value,'$.description'),'')), lower(?))"""
        total=db.execute('SELECT count(*) FROM ('+sql+')', [values,query]).fetchone()[0]
        rows=db.execute(sql+" ORDER BY json_extract_string(value,'$.name') LIMIT ? OFFSET ?",[values,query,limit,offset]).fetchall()
    return [json.loads(r[0]) for r in rows],total


def normalize(native):
    """Remove only a redundant single JSON-text MCP envelope; retain errors and source-bearing metadata."""
    encoded=json.dumps(native,ensure_ascii=False)
    sql="""WITH input AS (SELECT CAST(? AS JSON) AS doc), fields AS
      (SELECT doc,json_extract_string(doc,'$.content[0].text') AS body FROM input)
      SELECT CASE WHEN json_array_length(doc,'$.content')=1
        AND json_extract_string(doc,'$.content[0].type')='text'
        AND json_valid(body) AND json_extract(doc,'$.structuredContent') IS NULL
        AND json_extract(doc,'$._meta') IS NULL
        THEN CAST(body AS JSON) ELSE doc END FROM fields"""
    with duckdb.connect(':memory:') as db:
        value=db.execute(sql,[encoded]).fetchone()[0]
    return json.loads(value)
