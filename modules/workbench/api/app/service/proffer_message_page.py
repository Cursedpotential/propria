"""One page of a Proffer preview's messages, with the server-side search filter.

Split out of app/service/proffer.py on 2026-09-22 (that module was over the
300-line cap). The engine call goes through ``app.service.proffer`` so tests that
patch ``proffer._request`` keep covering it. Byline: Claude Code · Fable 5.1 · 2026-09-22.
"""

from __future__ import annotations

from app.service import proffer
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferPreviewMessagesResponse
from app.types.proffer_search import PreviewMessageFilter


async def preview_messages(
    preview_handle: str,
    *,
    mode: MatterMode,
    cursor: str | None,
    limit: int,
    search: PreviewMessageFilter | None = None,
) -> ProfferPreviewMessagesResponse:
    await proffer._require_mode(preview_handle, mode)
    params: dict[str, str | int] = {"limit": limit}
    if cursor:
        params["cursor"] = cursor
    if search is not None:
        params.update(search.as_query_params())
    response = await proffer._request("GET", f"/reference-import/previews/{preview_handle}/messages", params=params)
    result = proffer._validated(
        ProfferPreviewMessagesResponse,
        proffer._mode_payload(proffer._json_payload(response, "preview message page"), "preview message page", mode),
        "preview message page",
    )
    if result.preview_handle != preview_handle:
        raise ProfferError("Proffer preview message correlation failed", 502)
    return result
