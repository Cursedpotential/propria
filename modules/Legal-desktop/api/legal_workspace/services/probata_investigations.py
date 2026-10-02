"""Explicit investigation writes using the existing service connection, never evidence bytes."""

from __future__ import annotations

import json

import httpx

from legal_workspace.config import get_settings
from legal_workspace.contracts.claims import FollowupInvestigation, InvestigationResponse
from legal_workspace.services.probata_records import ProbataUnavailable, _authorization

_MAX_BODY = 2 * 1024 * 1024


def exchange(
    dispatch: FollowupInvestigation, *, write: bool, client: httpx.Client | None = None
) -> InvestigationResponse:
    """The caller must explicitly choose write intent; reads cannot dispatch."""
    base = get_settings().probata_records_base_url.rstrip("/")
    if not base:
        raise ProbataUnavailable("Probata investigation connection is not configured.")
    headers = {
        **_authorization(),
        "X-authentik-uid": dispatch.actor_uid,
        "X-authentik-username": dispatch.actor_username,
    }
    payload = dispatch.request.model_dump(mode="json")
    params = None
    url = base + "/legal-context/investigations"
    if write:
        headers["Idempotency-Key"] = str(dispatch.idempotency_key)
    else:
        if dispatch.request_id is None:
            raise ValueError("Dispatch this investigation before refreshing its status.")
        url += "/" + str(dispatch.request_id)
        params = {key: payload[key] for key in ("mode", "matter_id", "court_case_id")}
    http = client or httpx.Client(timeout=httpx.Timeout(8, connect=2), follow_redirects=False)
    try:
        with http.stream(
            "POST" if write else "GET",
            url,
            headers=headers,
            json=payload if write else None,
            params=params,
        ) as response:
            if response.status_code not in ({200, 201, 202} if write else {200}):
                raise ProbataUnavailable(
                    "Probata investigation request could not be confirmed. Retry or refresh."
                )
            data = bytearray()
            for chunk in response.iter_bytes():
                data.extend(chunk)
                if len(data) > _MAX_BODY:
                    raise ProbataUnavailable("Probata returned too much investigation data.")
            raw = json.loads(data)
            if not isinstance(raw, dict):
                raise TypeError("Unexpected response shape")
            if len(json.dumps(raw.get("results", []), ensure_ascii=False).encode("utf-8")) > 262144:
                raise ProbataUnavailable("Probata returned too much investigation result data.")
            result = InvestigationResponse.model_validate(raw)
        if (
            result.model_dump(
                mode="json", exclude={"request_id", "status", "created_at", "updated_at", "results"}
            )
            != payload
        ):
            raise ProbataUnavailable(
                "Probata returned a different investigation identity or case scope."
            )
        if dispatch.request_id is not None and result.request_id != dispatch.request_id:
            raise ProbataUnavailable("Probata returned a different investigation request.")
        return result
    except (httpx.HTTPError, ValueError, TypeError) as exc:
        raise ProbataUnavailable(
            "Probata investigation response is unavailable or invalid. Retry or refresh."
        ) from exc
    finally:
        if client is None:
            http.close()
