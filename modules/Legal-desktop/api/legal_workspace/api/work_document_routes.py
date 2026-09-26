"""Authenticated whole-document work-product APIs."""

from __future__ import annotations

from typing import Annotated
from urllib.parse import quote

from fastapi import APIRouter, Depends, File, Form, HTTPException, Request, UploadFile
from fastapi.responses import Response
from pydantic import BaseModel, Field

from legal_workspace.services.work_documents import (
    MAX_DOCUMENT_BYTES,
    MEDIA_TYPES,
    DocumentLocked,
    DocumentNotFound,
    RevisionConflict,
    WorkDocument,
    WorkDocumentRevision,
    WorkDocumentService,
)
from legal_workspace.services.workspace import WORKSPACE


def actor(request: Request) -> str:
    principal = getattr(request.state, "auth", None)
    if principal is None:
        raise HTTPException(401, "authentication required")
    return f"{principal.source}:{principal.subject}"


router = APIRouter(dependencies=[Depends(actor)])


def get_document_service() -> WorkDocumentService:
    return WorkDocumentService(WORKSPACE.store_dir, str(WORKSPACE.load().matter.matter_id))


Service = Annotated[WorkDocumentService, Depends(get_document_service)]
Actor = Annotated[str, Depends(actor)]


class BlankDocument(BaseModel):
    title: str = Field(default="Untitled document", min_length=1, max_length=250)


def translate_error(exc: Exception) -> HTTPException:
    if isinstance(exc, DocumentNotFound):
        return HTTPException(404, "Document or revision not found.")
    if isinstance(exc, RevisionConflict):
        return HTTPException(409, {"message": str(exc), "current_revision": exc.current_revision})
    if isinstance(exc, DocumentLocked):
        return HTTPException(423, str(exc))
    return HTTPException(400, str(exc))


async def upload_bytes(file: UploadFile) -> bytes:
    data = await file.read(MAX_DOCUMENT_BYTES + 1)
    if len(data) > MAX_DOCUMENT_BYTES:
        raise HTTPException(413, "Document exceeds 32 MiB.")
    return data


@router.get("/v1/work-documents", response_model=list[WorkDocument])
def list_documents(service: Service):
    return service.list()


@router.post("/v1/work-documents", response_model=WorkDocument, status_code=201)
def create_document(body: BlankDocument, service: Service, identity: Actor):
    try:
        return service.create_blank(body.title, actor=identity)
    except ValueError as exc:
        raise translate_error(exc) from exc


@router.post("/v1/work-documents/import", response_model=WorkDocument, status_code=201)
async def import_document(
    service: Service,
    identity: Actor,
    file: Annotated[UploadFile, File()],
    title: Annotated[str | None, Form()] = None,
):
    try:
        return service.import_document(
            file.filename or "", await upload_bytes(file), title=title, actor=identity
        )
    except ValueError as exc:
        raise translate_error(exc) from exc


@router.get("/v1/work-documents/{document_id}", response_model=WorkDocument)
def get_document(document_id: str, service: Service):
    try:
        return service.get(document_id)
    except DocumentNotFound as exc:
        raise translate_error(exc) from exc


@router.get("/v1/work-documents/{document_id}/revisions", response_model=list[WorkDocumentRevision])
def get_revisions(document_id: str, service: Service):
    try:
        return service.history(document_id)
    except DocumentNotFound as exc:
        raise translate_error(exc) from exc


@router.get("/v1/work-documents/{document_id}/content")
def get_content(document_id: str, service: Service, revision: int | None = None):
    try:
        document = service.get(document_id)
        selected = revision if revision is not None else document.current_revision
        return Response(
            service.read_bytes(document_id, selected),
            media_type=MEDIA_TYPES[document.format],
            headers={
                "Content-Disposition": f"attachment; filename*=UTF-8''{quote(document.filename)}",
                "ETag": f'"{document_id}:{selected}"',
                "Cache-Control": "private, no-store",
            },
        )
    except DocumentNotFound as exc:
        raise translate_error(exc) from exc


@router.put("/v1/work-documents/{document_id}/content", response_model=WorkDocument)
async def save_document(
    document_id: str,
    service: Service,
    identity: Actor,
    file: Annotated[UploadFile, File()],
    expected_revision: Annotated[int, Form(ge=1)],
):
    try:
        return service.save(
            document_id,
            await upload_bytes(file),
            expected_revision,
            actor=identity,
            origin="upload",
        )
    except (DocumentNotFound, ValueError) as exc:
        raise translate_error(exc) from exc
