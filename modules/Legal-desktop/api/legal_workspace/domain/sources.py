"""Pluggable research-source config and Case identity projection.

> _Byline: Grok · grok-4.6 · 2026-08-18_
CourtListener is a search connector, not a citator. Hits are identity
only — no opinion text, no evidence persist, no subsequent-history flag.
PACER is not a default source.
"""

from __future__ import annotations

import re
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field

_JSONPATH_SEGMENT = re.compile(
    r"""
    \.(?P<dot>[A-Za-z_][A-Za-z0-9_]*)
    |\[(?P<idx>\d+)]
    |\['(?P<sq>[^']+)']
    |\["(?P<dq>[^"]+)"]
    """,
    re.VERBOSE,
)

_SECRET_KEY_FRAGMENTS = ("token", "secret", "password", "api_key", "apikey", "authorization")


class SourceAuth(BaseModel):
    """Auth *shape* only. The token lives in env, never in this JSON."""

    type: str = "api_key_header"
    header: str = "Authorization"
    prefix: str = "Token "


class SourceEndpoint(BaseModel):
    method: str = "GET"
    url: str
    query: dict[str, str] = Field(default_factory=dict)


class CaseFieldMap(BaseModel):
    id: str = "$.cluster_id"
    name: str = "$.caseName"
    citation: str = "$.citation[0]"
    court: str = "$.court"
    date: str = "$.dateFiled"
    url: str = "$.absolute_url"


class SourceMap(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    list_path: str = Field(default="$.results", alias="list")
    case: CaseFieldMap = Field(default_factory=CaseFieldMap, alias="Case")


class SourceConfig(BaseModel):
    id: str
    display_name: str
    may_cost_money: bool = False
    daily_budget: float | None = None
    auth: SourceAuth = Field(default_factory=SourceAuth)
    endpoints: dict[str, SourceEndpoint]
    map: SourceMap = Field(default_factory=SourceMap)


class CaseHit(BaseModel):
    """Canonical Case identity. Extra upstream fields are dropped."""

    model_config = ConfigDict(extra="forbid")

    id: str
    name: str
    citation: str
    court: str
    date: str
    url: str


class SearchResult(BaseModel):
    """Fail-closed search envelope. CourtListener cannot verify subsequent history."""

    model_config = ConfigDict(extra="forbid")

    ok: bool
    hits: tuple[CaseHit, ...] = ()
    reason: str | None = None
    is_citator_verified: Literal[False] = False


def courtlistener_default_config() -> SourceConfig:
    """Baked-in CAT4 CourtListener / RECAP example. Token is not embedded."""
    return SourceConfig.model_validate(
        {
            "id": "courtlistener",
            "display_name": "CourtListener / RECAP",
            "may_cost_money": False,
            "auth": {
                "type": "api_key_header",
                "header": "Authorization",
                "prefix": "Token ",
            },
            "endpoints": {
                "search": {
                    "method": "GET",
                    "url": "https://www.courtlistener.com/api/rest/v4/search/",
                    "query": {"q": "{{query}}", "type": "o"},
                }
            },
            "map": {
                "list": "$.results",
                "Case": {
                    "id": "$.cluster_id",
                    "name": "$.caseName",
                    "citation": "$.citation[0]",
                    "court": "$.court",
                    "date": "$.dateFiled",
                    "url": "$.absolute_url",
                },
            },
        }
    )


def resolve_jsonpath(data: object, path: str) -> object | None:
    """Tiny JSONPath subset: `$.key`, `$.key[0]`, `$.key['x']`. Not a citator."""
    if path in ("", "$"):
        return data
    if not path.startswith("$"):
        path = "$." + path
    current: object = data
    cursor = 1
    while cursor < len(path):
        match = _JSONPATH_SEGMENT.match(path, cursor)
        if match is None or current is None:
            return None
        key = match.group("dot") or match.group("sq") or match.group("dq")
        if key is not None:
            if not isinstance(current, dict):
                return None
            current = current.get(key)
        else:
            idx = int(match.group("idx"))
            if isinstance(current, list):
                current = current[idx] if 0 <= idx < len(current) else None
            elif isinstance(current, str):
                return current if idx == 0 else None
            else:
                return None
        cursor = match.end()
    return current


def project_case_hit(row: object, field_map: CaseFieldMap) -> CaseHit | None:
    """Map one upstream row to identity fields. Missing scalars stay empty."""
    if not isinstance(row, dict):
        return None
    values = {
        field: _as_text(resolve_jsonpath(row, getattr(field_map, field)))
        for field in ("id", "name", "citation", "court", "date", "url")
    }
    if not values["id"] and not values["name"] and not values["citation"]:
        return None
    return CaseHit.model_validate(values)


def redact_source_config(config: SourceConfig) -> dict[str, Any]:
    """Public view. Tokens never leave the process via this dump."""
    return _redact(config.model_dump(mode="json", by_alias=True))


def _as_text(value: object) -> str:
    if value is None:
        return ""
    if isinstance(value, list):
        return _as_text(value[0]) if value else ""
    if isinstance(value, dict):
        return ""
    return str(value)


def _redact(value: object) -> object:
    if isinstance(value, dict):
        redacted: dict[str, object] = {}
        for key, item in value.items():
            lowered = key.lower()
            if any(fragment in lowered for fragment in _SECRET_KEY_FRAGMENTS):
                redacted[key] = "***" if item not in (None, "") else item
            else:
                redacted[key] = _redact(item)
        return redacted
    if isinstance(value, list):
        return [_redact(item) for item in value]
    return value
