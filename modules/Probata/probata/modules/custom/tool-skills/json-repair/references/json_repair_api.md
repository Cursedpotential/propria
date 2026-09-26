# json-repair — full API reference

> Source: https://github.com/mangiucugna/json_repair (clipper snapshot 2026-07-05). Pin `json_repair==0.*`.

## How the parser thinks

Repairs follow this BNF; when a token doesn't fit, simple heuristics close brackets/braces, quote strings, and trim whitespace:

```
<json>      ::= <primitive> | <container>
<primitive> ::= <number> | <string> | <boolean>   ; boolean = true|false|null (unquoted)
<container> ::= <object> | <array>
<array>     ::= '[' [ <json> *(', ' <json>) ] ']'
<object>    ::= '{' [ <member> *(', ' <member>) ] '}'
<member>    ::= <string> ': ' <json>
```

Heuristics: add missing parentheses/brackets when a container should close; quote unquoted strings or add missing quotes; adjust whitespace / strip line breaks.

## Method signatures

```python
json_repair.loads(
    s,                       # str | bytes
    *,
    skip_json_loads=False,   # skip stdlib json.loads fast path
    strict=False,            # raise instead of repair on structural ambiguity
    schema=None,             # JSON Schema dict OR Pydantic v2 model class
    schema_repair_mode="standard",  # "standard" | "salvage"
    stream_stable=False,     # keep partial JSON stable across stream chunks
) -> object

json_repair.repair_json(
    s,
    *,
    return_objects=False,    # True -> return object (faster); False -> JSON str
    skip_json_loads=False,
    strict=False,
    schema=None,
    schema_repair_mode="standard",
    stream_stable=False,
    **json_dumps_kwargs,     # ensure_ascii, indent, sort_keys, ... -> json.dumps
) -> str | object            # "" (str) if unrepairable and return_objects=False

json_repair.load(fd, **kwargs) -> object       # drop-in for json.load(fd)
json_repair.from_file(path, **kwargs) -> object
```

`json.dumps` kwargs pass through (`indent`, `ensure_ascii`, `sort_keys`, `separators`, etc.).

## Defaults + performance rules of thumb

- Default flow: try `json.loads` → return if success → run repair parser on failure. **This is the fast path; keep it.**
- `return_objects=True` is always faster than re-serializing — prefer it when you want an object.
- `skip_json_loads=True` is faster **only** when the input is 100% known-invalid. On possibly-valid input it bypasses the stdlib success path and the repair parser may mutate valid JSON.
- The library never auto-uses third-party JSON libs (e.g. `orjson`). Use them yourself if you want their semantics.

## orjson-first, json_repair as fallback

```python
import json_repair, orjson
try:
    obj = orjson.loads(s)
except orjson.JSONDecodeError:
    obj = json_repair.loads(s, skip_json_loads=True)
```

## Schema-guided repair (beta — `pip install 'json-repair[schema]'`)

JSON Schema:

```python
from json_repair import repair_json

schema = {"type": "object",
          "properties": {"value": {"type": "integer"}},
          "required": ["value"]}
repair_json('{"value": "1"}', schema=schema, return_objects=True)
# {'value': 1}   -- "1" coerced to int

repair_json(
    '{"items":[{"id":1,"score":85.6},{"id":2,"score":"N/A"}]}',
    schema={"type":"object",
            "properties":{"items":{"type":"array",
                "items":{"type":"object",
                    "properties":{"id":{"type":"integer"},"score":{"type":"number"}},
                    "required":["id","score"]}}}},
            "required":["items"]},
    schema_repair_mode="salvage",
    return_objects=True,
)
# salvage drops/repairs the unrepairable item instead of failing
```

Pydantic v2 model:

```python
from pydantic import BaseModel, Field
from json_repair import repair_json

class Payload(BaseModel):
    value: int
    tags: list[str] = Field(default_factory=list)

repair_json('{"value": "1", "tags": }',
            schema=Payload, skip_json_loads=True, return_objects=True)
# {'value': 1, 'tags': []}
```

Schema guidance is **always applied** when `schema` is set (valid or invalid JSON). It is mutually exclusive with `strict=True`. If the input cannot be repaired to satisfy the schema, `ValueError` is raised.

### `schema_repair_mode`

- `standard` (default): fill missing values (defaults, required), coerce safe scalars, drop disallowed properties/items.
- `salvage`: everything in `standard` **plus** drop unrepairable array items; map array→object by property order when shape is unambiguous; unwrap `[{...}]`→`{...}` when root schema is an object; fill missing required fields only when a safe value can be inferred (`default`, `const`, first `enum`, or empty array/object when allowed).

## Streaming

```python
from json_repair import repair_json
out = repair_json(stream_input, stream_stable=True)
# keeps partial JSON structurally stable as chunks arrive
```

## Strict mode

```python
repair_json(s, strict=True)        # Python
json_repair --strict input.json    # CLI
```

Raises `ValueError` on: duplicate keys, missing `:` separators, empty keys/values introduced by stray commas, multiple top-level elements, other ambiguous constructs. Honors `skip_json_loads=True` (skip stdlib check but still enforce strict rules). Use when you want validation with friendlier errors while keeping json_repair's resilience elsewhere in the stack.

## Non-Latin / Unicode

```python
repair_json("{'k':'统一码'}")                          # -> {"k": "统一码"}
repair_json("{'k':'统一码'}", ensure_ascii=False)      # -> {"k": "统一码"}
```

## Truncation behavior

If the string is "super broken" and cannot be repaired, `repair_json(s)` (no `return_objects`) returns `""` (empty string) — not an exception. With `return_objects=True` you get whatever object the parser could salvage. Plan for both.

## CLI (full)

```
json_repair [-h] [-i] [-o TARGET] [--ensure_ascii] [--indent INDENT]
            [--skip-json-loads] [--schema SCHEMA] [--schema-model MODEL]
            [--strict] [--schema-repair-mode {standard,salvage}] [filename]
```

- `filename` omitted → reads stdin.
- `-i/--inline` → replace file in place.
- `-o TARGET` → write to TARGET.
- `--schema MODEL` → Pydantic v2 model in `module:ClassName` form.
- `--indent INDENT` → default 2.

## Cross-language equivalents (if you need them elsewhere)

- TypeScript: josdejong/jsonrepair
- Go: RealAlexandreAI/json-repair
- Ruby: sashazykov/json-repair-rb
- Rust: oramasearch/llm_json
- R: cgxjdzz/jsonRepair
- Java: du00cs/json-repairj