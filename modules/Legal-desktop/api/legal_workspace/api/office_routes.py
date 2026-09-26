"""Authenticated session launch and capability-authenticated WOPI callbacks."""

from __future__ import annotations

import os

from fastapi import APIRouter, HTTPException, Request, Response

from legal_workspace.services.office_editor import (
    OfficeEditor,
    OfficeError,
    discover_editor_url,
    request_token,
)
from legal_workspace.services.work_documents import (
    MAX_DOCUMENT_BYTES,
    DocumentNotFound,
    WorkDocumentService,
)
from legal_workspace.services.workspace import WORKSPACE

router = APIRouter()
MAX_OFFICE_BYTES = MAX_DOCUMENT_BYTES


def editor() -> OfficeEditor:
    state = WORKSPACE.load()
    return OfficeEditor(
        WorkDocumentService(WORKSPACE.store_dir, str(state.matter.matter_id)), WORKSPACE.store_dir
    )


def failure(exc: OfficeError) -> HTTPException:
    headers = {"X-WOPI-Lock": exc.lock} if exc.lock is not None else None
    return HTTPException(exc.status, str(exc), headers=headers)


def token_for(request: Request) -> str:
    return request_token(
        request.query_params.get("access_token"), request.headers.get("authorization")
    )


@router.get("/v1/office/status")
def office_status():
    configured = all(
        os.environ.get(name)
        for name in (
            "OFFICE_EDITOR_PUBLIC_URL",
            "OFFICE_EDITOR_INTERNAL_URL",
            "OFFICE_WOPI_BASE_URL",
        )
    )
    if not configured:
        return {
            "configured": False,
            "available": False,
            "reason": "Interactive office service is not configured",
        }
    try:
        discover_editor_url("odt", "status")
    except OfficeError as exc:
        return {"configured": True, "available": False, "reason": str(exc)}
    return {"configured": True, "available": True, "reason": None}


@router.post("/v1/work-documents/{document_id}/office-session")
def create_office_session(document_id: str, request: Request, response: Response):
    response.headers["Cache-Control"] = "no-store"
    principal = request.state.auth
    if principal.source in {"mcp-gateway", "office-session"}:
        raise HTTPException(403, "An authenticated workspace session is required")
    try:
        service = editor()
        doc = service.documents.get(document_id)
        url = discover_editor_url(doc.format, document_id)
        return service.create_session(document_id, f"{principal.source}:{principal.subject}", url)
    except DocumentNotFound as exc:
        raise HTTPException(404, "Document not found") from exc
    except OfficeError as exc:
        raise failure(exc) from exc


@router.get("/wopi/files/{document_id}")
def check_file_info(document_id: str, request: Request, response: Response):
    response.headers["Cache-Control"] = "no-store"
    try:
        return editor().info(document_id, token_for(request))
    except OfficeError as exc:
        raise failure(exc) from exc
    except DocumentNotFound as exc:
        raise HTTPException(404, "Document not found") from exc


@router.get("/wopi/files/{document_id}/contents")
def get_file(document_id: str, request: Request):
    try:
        data, revision = editor().read(document_id, token_for(request))
        return Response(
            data,
            media_type="application/octet-stream",
            headers={"X-WOPI-ItemVersion": str(revision), "Cache-Control": "no-store"},
        )
    except OfficeError as exc:
        raise failure(exc) from exc
    except DocumentNotFound as exc:
        raise HTTPException(404, "Document not found") from exc


@router.post("/wopi/files/{document_id}")
def file_operation(document_id: str, request: Request):
    operation = request.headers.get("x-wopi-override", "").upper()
    if operation not in {"LOCK", "REFRESH_LOCK", "UNLOCK", "GET_LOCK"}:
        raise HTTPException(501, "Unsupported WOPI operation")
    try:
        value = editor().lock(
            document_id,
            token_for(request),
            operation,
            request.headers.get("x-wopi-lock", ""),
            request.headers.get("x-wopi-oldlock"),
        )
        return Response(
            status_code=200, headers={"X-WOPI-Lock": value} if operation == "GET_LOCK" else {}
        )
    except OfficeError as exc:
        raise failure(exc) from exc
    except DocumentNotFound as exc:
        raise HTTPException(404, "Document not found") from exc


@router.post("/wopi/files/{document_id}/contents")
async def put_file(document_id: str, request: Request):
    if request.headers.get("x-wopi-override", "").upper() != "PUT":
        raise HTTPException(400, "X-WOPI-Override PUT required")
    try:
        service = editor()
        token = token_for(request)
        service.authorize(document_id, token)
        data = bytearray()
        async for chunk in request.stream():
            data.extend(chunk)
            if len(data) > MAX_OFFICE_BYTES:
                raise HTTPException(413, "Document exceeds 32 MiB")
        result = service.save(
            document_id, token, request.headers.get("x-wopi-lock", ""), bytes(data)
        )
        return Response(
            status_code=200, headers={"X-WOPI-ItemVersion": str(result.current_revision)}
        )
    except OfficeError as exc:
        raise failure(exc) from exc
    except DocumentNotFound as exc:
        raise HTTPException(404, "Document not found") from exc
    except ValueError as exc:
        raise HTTPException(400, str(exc)) from exc
