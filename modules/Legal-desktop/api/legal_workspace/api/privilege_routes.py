"""Privilege / provider-grid HTTP surface.

> _Byline: Grok · grok-4.6 · 2026-08-18_
GET /v1/providers is static. POST /v1/gateway:invoke is fail-closed
and gated by LEGAL_WORKSPACE_INVOKE_MODELS. Not a privilege legal
conclusion. Parent mounts this router on the FastAPI app.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from legal_workspace.config import get_settings
from legal_workspace.domain.provider_grid import (
    ProviderGrid,
    confidential_blocked_reason,
    provider_grid,
)
from legal_workspace.services.gateway import invoke_chat

router = APIRouter()


class GatewayInvokeRequest(BaseModel):
    prompt: str
    model: str
    confidential: bool = False


class GatewayInvokeResponse(BaseModel):
    ok: bool
    text: str
    model: str
    confidential: bool
    court_safe: bool = False
    fallback: str | None = Field(default=None, description="Always null. No silent fallback.")


@router.get("/v1/providers", response_model=ProviderGrid)
def list_providers() -> ProviderGrid:
    """Cited CAT6 trust grid. Not a privilege legal conclusion."""
    return provider_grid()


@router.post("/v1/gateway:invoke", response_model=GatewayInvokeResponse)
def gateway_invoke(body: GatewayInvokeRequest) -> GatewayInvokeResponse:
    settings = get_settings()
    if not settings.invoke_models:
        raise HTTPException(status_code=409, detail="invoke-disabled")
    confidential = body.confidential
    try:
        from legal_workspace.services.workspace import get_workspace

        confidential = confidential or bool(get_workspace().load().confidential_mode)
    except Exception:
        pass
    if confidential:
        reason = confidential_blocked_reason(body.model)
        if reason is not None:
            raise HTTPException(status_code=409, detail="confidential_blocked")
    result = invoke_chat(body.prompt, body.model, confidential=confidential)
    if not result.ok and result.text == "confidential_blocked":
        raise HTTPException(status_code=409, detail="confidential_blocked")
    if not result.ok:
        raise HTTPException(status_code=503, detail=result.text)
    return GatewayInvokeResponse(
        ok=True,
        text=result.text,
        model=result.model,
        confidential=confidential,
        court_safe=False,
        fallback=None,
    )
