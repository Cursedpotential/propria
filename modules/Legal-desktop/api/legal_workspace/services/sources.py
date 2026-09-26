"""Research-source search. Config-driven HTTP. Fail closed. Not a citator.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Hits are identity projections. They are not evidence, not holdings, and
not subsequent-history. PACER is not loaded. Nothing is persisted.
"""

from __future__ import annotations

import os
from urllib.parse import urljoin

import httpx

from legal_workspace.domain.sources import (
    CaseHit,
    SearchResult,
    SourceConfig,
    courtlistener_default_config,
    project_case_hit,
    resolve_jsonpath,
)

SEARCH_TIMEOUT = 2.0
_TOKEN_ENV = {
    "courtlistener": "COURT_LISTENER_TOKEN",
}


def load_source_configs() -> tuple[SourceConfig, ...]:
    """Baked-in CourtListener only. Midpage / PACER are not defaults."""
    return (courtlistener_default_config(),)


def search_source(
    source_id: str,
    query: str,
    client: httpx.Client | None = None,
) -> SearchResult:
    """GET the source search endpoint. Never raises. Never citator-verifies."""
    try:
        return _search_source(source_id, query, client)
    except Exception as exc:
        return SearchResult(ok=False, hits=(), reason=exc.__class__.__name__)


def _search_source(
    source_id: str,
    query: str,
    client: httpx.Client | None,
) -> SearchResult:
    if not query.strip():
        return SearchResult(ok=False, hits=(), reason="empty-query")
    config = _config_by_id(source_id)
    if config is None:
        return SearchResult(ok=False, hits=(), reason="unknown-source")
    endpoint = config.endpoints.get("search")
    if endpoint is None:
        return SearchResult(ok=False, hits=(), reason="no-search-endpoint")
    if endpoint.method.upper() != "GET":
        return SearchResult(ok=False, hits=(), reason="unsupported-method")

    url = _fill(endpoint.url, query)
    params = {key: _fill(value, query) for key, value in endpoint.query.items()}
    headers = _auth_headers(config)
    http, closer = _client(client)
    try:
        response = http.get(url, params=params, headers=headers, timeout=SEARCH_TIMEOUT)
        if response.status_code in {401, 403}:
            return SearchResult(ok=False, hits=(), reason="auth-required")
        if response.status_code >= 400:
            return SearchResult(ok=False, hits=(), reason="upstream-error")
        return _map_response(config, response.json(), endpoint.url)
    except httpx.TimeoutException:
        return SearchResult(ok=False, hits=(), reason="timeout")
    finally:
        if closer:
            http.close()


def _config_by_id(source_id: str) -> SourceConfig | None:
    for item in load_source_configs():
        if item.id == source_id:
            return item
    return None


def _client(existing: httpx.Client | None) -> tuple[httpx.Client, bool]:
    if existing is not None:
        return existing, False
    return httpx.Client(timeout=SEARCH_TIMEOUT), True


def _fill(template: str, query: str) -> str:
    return template.replace("{{query}}", query)


def _auth_headers(config: SourceConfig) -> dict[str, str]:
    token = _token_for(config.id)
    if not token:
        return {}
    return {config.auth.header: f"{config.auth.prefix}{token}"}


def _token_for(source_id: str) -> str | None:
    env_name = _TOKEN_ENV.get(source_id, f"{source_id.upper().replace('-', '_')}_TOKEN")
    raw = os.getenv(env_name)
    if raw is None:
        return None
    token = raw.strip()
    return token or None


def _map_response(config: SourceConfig, body: object, endpoint_url: str) -> SearchResult:
    rows = resolve_jsonpath(body, config.map.list_path)
    if rows is None:
        return SearchResult(ok=False, hits=(), reason="unexpected-shape")
    if not isinstance(rows, list):
        return SearchResult(ok=False, hits=(), reason="unexpected-shape")
    hits: list[CaseHit] = []
    for row in rows:
        hit = project_case_hit(row, config.map.case)
        if hit is None:
            continue
        hits.append(hit.model_copy(update={"url": _absolutize(hit.url, endpoint_url)}))
    return SearchResult(ok=True, hits=tuple(hits), reason=None)


def _absolutize(url: str, endpoint_url: str) -> str:
    if not url:
        return ""
    if url.startswith("https://") or url.startswith("http://"):
        return url
    return urljoin(endpoint_url, url)
