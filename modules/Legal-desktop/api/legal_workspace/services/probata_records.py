"""Live read-through of Probata-owned context; never another authored record store."""

from __future__ import annotations

import re
from datetime import UTC, datetime
from pathlib import Path
from typing import Literal

import httpx
from pydantic import BaseModel, ConfigDict, Field

from legal_workspace.config import get_settings

Kind = Literal["entity", "event"]
_TOKEN = re.compile(r"[A-Za-z0-9\-._~+/]+={0,}")
_MAX_BODY = 2 * 1024 * 1024


class ProbataUnavailable(RuntimeError):
    pass


class Origin(BaseModel):
    model_config = ConfigDict(extra="forbid")
    system: Literal["probata"]
    kind: Kind
    record_id: str = Field(min_length=1, max_length=250)
    record_version: str = Field(min_length=1, max_length=300)


class Record(BaseModel):
    origin: Origin
    title: str
    record: dict


class Listing(BaseModel):
    available: bool
    reason: str | None = None
    records: list[Record] = Field(default_factory=list)
    mode: Literal["DEV", "LIVE"] = "LIVE"
    refreshed_at: str = Field(default_factory=lambda: datetime.now(UTC).isoformat())
    truncated: bool = False
    matter_id: str | None = None
    court_case_id: str | None = None


def _authorization() -> dict[str, str]:
    path = get_settings().probata_records_token_file
    if not path:
        raise ProbataUnavailable("Probata connection credentials are not configured.")
    token_file = Path(path)
    try:
        if not token_file.is_absolute() or not token_file.is_file() or token_file.stat().st_size > 4098:
            raise ValueError
        token = token_file.read_text(encoding="utf-8").strip()
        if not 32 <= len(token) <= 4096 or not _TOKEN.fullmatch(token):
            raise ValueError
    except (OSError, UnicodeError, ValueError):
        raise ProbataUnavailable("Probata connection credentials are unavailable.") from None
    return {"Authorization": f"Bearer {token}"}


def list_records(kind: Kind, query: str = "", *, record_id: str | None = None,
                 client: httpx.Client | None = None) -> Listing:
    settings = get_settings()
    mode = settings.probata_records_mode
    if not settings.probata_records_base_url:
        return Listing(available=False, reason="Probata records connection is not configured.", mode=mode)
    if kind not in {"entity", "event"} or len(query) > 200:
        raise ValueError("Invalid record type or search.")
    if record_id is not None and not 1 <= len(record_id) <= 250:
        raise ValueError("Invalid Probata record identity.")
    params = {"mode": mode, "kind": kind, "q": query, "limit": "50"}
    if record_id is not None:
        params["record_id"] = record_id
    http = client or httpx.Client(timeout=httpx.Timeout(8, connect=2), follow_redirects=False)
    try:
        headers = _authorization()
        # Bounded streaming keeps the read adapter from retaining an upstream corpus.
        with http.stream("GET", settings.probata_records_base_url.rstrip("/") + "/legal-context/records",
                         params=params, headers=headers) as response:
            if response.status_code != 200:
                raise ProbataUnavailable("Probata records could not be read. Try refreshing.")
            chunks = bytearray()
            for chunk in response.iter_bytes():
                chunks.extend(chunk)
                if len(chunks) > _MAX_BODY:
                    raise ProbataUnavailable("Probata returned too much record data.")
            import json
            data = json.loads(chunks)
        if not isinstance(data, dict) or data.get("mode") != mode or data.get("available") is not True:
            raise ProbataUnavailable("Probata returned an unexpected case scope.")
        raw = data.get("records")
        if not isinstance(raw, list) or len(raw) > 50:
            raise ProbataUnavailable("Probata returned an unexpected record list.")
        records = [Record.model_validate(item) for item in raw]
        if any(item.origin.kind != kind or
               (record_id is not None and item.origin.record_id != record_id) for item in records):
            raise ProbataUnavailable("Probata returned a different record identity.")
        identities = [item.origin.record_id for item in records]
        if len(identities) != len(set(identities)) or (record_id is not None and len(records) > 1):
            raise ProbataUnavailable("Probata returned ambiguous record identities.")
        return Listing(available=True, records=records, mode=mode, truncated=bool(data.get("truncated")),
                       matter_id=data.get("matter_id"), court_case_id=data.get("court_case_id"))
    except (ProbataUnavailable, httpx.HTTPError, ValueError, TypeError):
        return Listing(available=False, reason="Probata records are unavailable. Saved legal responses remain accessible.", mode=mode)
    finally:
        if client is None:
            http.close()


def get_record(kind: Kind, record_id: str) -> dict | None:
    result = list_records(kind, record_id=record_id)
    if not result.available:
        raise ProbataUnavailable(result.reason or "Probata records are unavailable.")
    return result.records[0].model_dump(mode="json") if result.records else None
