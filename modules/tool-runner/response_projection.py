"""Use transient DuckDB queries for bounded discovery and proven-redundant MCP envelopes.
Inputs: authorized native schemas/results. Outputs: selected metadata or equivalent semantic JSON.
Effects: in-memory projection only. Business records, errors and nonredundant metadata are retained.
"""
import json
import duckdb


def catalog_page(tools,query,offset,limit):
    """Filter and paginate authorized metadata without exposing the whole schema catalog."""
    with duckdb.connect(':memory:') as db:
        values=json.dumps(tools,ensure_ascii=False)
        sql="""SELECT value FROM json_each(CAST(? AS JSON))
          WHERE contains(lower(coalesce(json_extract_string(value,'$.name'),'') || ' ' ||
             coalesce(json_extract_string(value,'$.title'),'') || ' ' ||
             coalesce(json_extract_string(value,'$.description'),'')), lower(?))"""
        total=db.execute('SELECT count(*) FROM ('+sql+')',[values,query]).fetchone()[0]
        rows=db.execute(sql+" ORDER BY json_extract_string(value,'$.name') LIMIT ? OFFSET ?",[values,query,limit,offset]).fetchall()
    return [json.loads(r[0]) for r in rows],total


def normalize(native):
    """Remove a single duplicate protocol envelope, never an actual record containing a content field.

    Caller retains the native isError flag separately. A _meta object, attachments,
    additional source fields or non-equivalent structured/text outputs preserve the envelope.
    """
    if not isinstance(native,dict) or not set(native)<= {'content','structuredContent','isError','is_error','_meta'}:
        return native
    content=native.get('content')
    if not isinstance(content,list) or len(content)!=1 or content[0].get('type')!='text' or native.get('_meta') is not None:
        return native
    try:decoded=json.loads(content[0].get('text',''))
    except (ValueError,TypeError):return native
    structured=native.get('structuredContent')
    if structured is not None and structured!=decoded:return native
    with duckdb.connect(':memory:') as db:
        value=db.execute('SELECT json(CAST(? AS JSON))',[json.dumps(decoded,ensure_ascii=False)]).fetchone()[0]
    return json.loads(value)
