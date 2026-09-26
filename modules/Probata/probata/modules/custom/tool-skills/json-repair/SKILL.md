---
name: json-repair
description: Repair malformed/invalid JSON from LLMs, APIs, logs, or user input using the `json-repair` Python library (PyPI `json-repair`, repo mangiucugna/json_repair). Use when `json.loads()` raises JSONDecodeError on model output, API responses, or log lines; JSON has missing/trailing commas, unquoted keys, single quotes, comments, stray prose, truncated/unbalanced brackets/braces; a drop-in tolerant fallback for `json.loads()`/`json.load()` is needed; schema-guided or Pydantic v2-guided repair is needed; or streamed partial JSON must be kept stable. Covers install (pip/pipx), Python API (loads/repair_json/load/from_file), CLI, and the skip_json_loads/return_objects/strict/schema/schema_repair_mode/stream_stable/ensure_ascii flags. Also use when deciding whether to repair vs. reject ambiguous JSON (strict mode).
---

# json-repair

Wrap the `json-repair` library (PyPI: `json-repair`, author Stefano Baccianella, repo https://github.com/mangiucugna/json_repair). It repairs the common LLM/API JSON mistakes (missing quotes/commas/brackets, stray prose, truncated values, single-quoted strings, comments) and is a drop-in tolerant fallback for `json.loads()`.

> _Byline: Claude Code · glm-5.2 · 2026-07-05_

## Install

```bash
pip install json-repair            # Python API
pipx install json-repair           # CLI only
pip install 'json-repair[schema]'  # schema/Pydantic-guided repair (beta)
```

Pin only the major version in requirements: `json_repair==0.*` (semver, frequent non-breaking minor/patch updates).

## Core usage

Drop-in replacement for `json.loads()` — this is the **default pattern**; it tries stdlib `json.loads` first and only runs the repair parser if that fails:

```python
import json_repair
obj = json_repair.loads(json_string)
```

Repair to a JSON string:

```python
from json_repair import repair_json
good = repair_json(bad_json_string)            # returns str; "" if unrepairable
obj  = repair_json(bad_json_string, return_objects=True)  # returns object (faster)
```

File helpers:

```python
import json_repair
obj = json_repair.load(open(fname, "rb"))   # drop-in for json.load (fd)
obj = json_repair.from_file(json_file)       # path-based
# IO exceptions (OSError/IOError) are NOT caught — handle them yourself.
```

## Antipattern to avoid

Don't pre-flight with `json.loads` yourself — `json_repair` already does that. Wasteful:

```python
# DON'T
try: obj = json.loads(s)
except json.JSONDecodeError: obj = json_repair.loads(s)
```

Just call `json_repair.loads(s)` directly. Only use `skip_json_loads=True` when you **already know** the input is invalid (see flags below).

## Key flags

| Flag | When | Effect |
|---|---|---|
| `return_objects=True` | You want a Python object, not a string | Faster — skips re-serialization. Prefer this. |
| `skip_json_loads=True` | Input is **known** invalid | Skips the stdlib fast path, goes straight to repair. Do NOT use on possibly-valid input — the repair parser may still mutate valid JSON. |
| `strict=True` | Validate, don't repair | Raises `ValueError` on duplicate keys, missing `:`, empty keys/values, multiple top-level elements. Mutually exclusive with `schema`. |
| `schema={...}` or `schema=PydanticModel` | Need schema-valid output | Fills defaults/required, coerces scalars (`"1"`→`1`, `"yes"`→`True`), drops disallowed props. Requires `pip install 'json-repair[schema]'`. Always applied (even to valid JSON). |
| `schema_repair_mode="salvage"` | Best-effort array/item salvage | Adds: drop unrepairable array items, map array→object by property order, unwrap `[{...}]`→`{...}` when root expects object, fill required fields only when safe. Default is `"standard"`. |
| `stream_stable=True` | Streaming partial JSON | Keeps in-progress JSON stable across incremental chunks. |
| `ensure_ascii=False` | Non-Latin (CJK, etc.) | Preserve non-Latin chars instead of `\uXXXX` escaping. |
| `indent=N` | Pretty output | Passes through to `json.dumps` (any `json.dumps` kwarg works). |

If escaping is misbehaving, pass the input as a **raw string**: `r"...\""`.

## CLI

```bash
json_repair input.json                 # write repaired JSON to stdout
json_repair -i input.json              # in-place
json_repair -o out.json input.json     # to file
json_repair --strict input.json        # validate, exit nonzero on structural issues
json_repair --skip-json-loads in.json  # known-bad input
json_repair --schema schema.json in.json
json_repair --schema-repair-mode salvage in.json
json_repair --schema-model mymod:Payload in.json   # Pydantic v2 model
json_repair --ensure_ascii --indent 4 in.json
# reads stdin if no filename given
```

## When NOT to use this

- You need guarantees about valid JSON being preserved byte-for-byte — use stdlib `json` (or `orjson`) directly; `json_repair` may mutate even valid input when `skip_json_loads=True`.
- You want a different JSON library's semantics (e.g. `orjson`) — call it yourself first, fall back to `json_repair.loads(s, skip_json_loads=True)` only on its `JSONDecodeError`.
- You need full schema validation, not repair — `strict=True` gives friendlier errors but this is a repair tool, not a validator like `jsonschema`.

## Full API + edge cases

See [references/json_repair_api.md](references/json_repair_api.md) for: the BNF the parser follows, every method signature, schema-guided examples (JSON Schema + Pydantic v2), streaming pattern, strict-mode failure modes, and the `orjson`-fallback pattern.