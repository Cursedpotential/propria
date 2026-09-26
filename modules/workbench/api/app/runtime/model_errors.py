"""HTTP answers for failed model calls, shared by the classification, sentiment and comparison routes.

Byline: Claude Code · Opus 5.5 · 2026-09-25 (unusable reply → 502 naming the model; provider rate
limit that outlasted the backoff → 429 instead of an opaque 500 "Unknown model error")
"""

from __future__ import annotations

from agno.exceptions import ModelRateLimitError
from fastapi import HTTPException

from app.service.model_replies import ModelReplyError

MODEL_CALL_ERRORS = (ModelReplyError, ModelRateLimitError)


def model_call_http_error(error: Exception) -> HTTPException:
    """Translate a MODEL_CALL_ERRORS failure into the HTTP answer the client should see."""
    if isinstance(error, ModelRateLimitError):
        return HTTPException(
            status_code=429,
            detail=f"Model '{error.model_id}' is rate-limited by its provider after retries with backoff; "
            "try again shortly.",
        )
    return HTTPException(status_code=502, detail=str(error))
