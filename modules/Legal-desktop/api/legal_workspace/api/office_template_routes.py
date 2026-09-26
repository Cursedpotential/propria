"""HTTP contract for versioned editable office templates.

Document persistence belongs to the document/revision service. These routes
provide catalog metadata and deterministic DOCX bytes for that service.
"""

from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import Response
from pydantic import BaseModel, Field

from legal_workspace.services.office_templates import (
    TEMPLATE_VERSION, build_template_document, template_catalog,
)
from legal_workspace.services.work_documents import WorkDocument, WorkDocumentService
from legal_workspace.services.workspace import WORKSPACE


def actor(request: Request) -> str:
    principal = getattr(request.state, "auth", None)
    if principal is None:
        raise HTTPException(401, "authentication required")
    return f"{principal.source}:{principal.subject}"


def document_service() -> WorkDocumentService:
    return WorkDocumentService(WORKSPACE.store_dir, str(WORKSPACE.load().matter.matter_id))


router = APIRouter(dependencies=[Depends(actor)])


class OfficeTemplateRequest(BaseModel):
    title: str | None = None
    fields: dict[str, str] = Field(default_factory=dict)
    caption: dict[str, str] = Field(default_factory=dict)
    sections: list[str] = Field(default_factory=list)
    signature: dict[str, str] = Field(default_factory=dict)


def _docx_response(artifact: object) -> Response:
    info = artifact.info
    response = Response(
        content=artifact.content,
        media_type="application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    )
    response.headers["content-disposition"] = f'attachment; filename="{info.filename}"'
    response.headers["x-template-version"] = info.version
    response.headers["x-content-sha256"] = artifact.content_hash
    return response


@router.get("/v1/office-templates")
def list_office_templates() -> list[object]:
    return template_catalog()


@router.get("/v1/office-templates/{template_id}/versions/{version}")
def download_office_template(template_id: str, version: str) -> Response:
    if version != TEMPLATE_VERSION:
        raise HTTPException(status_code=404, detail="template version not found")
    try:
        return _docx_response(build_template_document(template_id))
    except (StopIteration, ValueError) as exc:
        raise HTTPException(status_code=404, detail="template not found") from exc


@router.get("/v1/office-templates/{template_id}/download")
def download_current_office_template(template_id: str) -> Response:
    return download_office_template(template_id, TEMPLATE_VERSION)


@router.post("/v1/office-templates/{template_id}/versions/{version}:instantiate", response_model=WorkDocument)
def instantiate_office_template(
    template_id: str, version: str, body: OfficeTemplateRequest,
    request: Request,
) -> WorkDocument:
    if version != TEMPLATE_VERSION:
        raise HTTPException(status_code=404, detail="template version not found")
    try:
        fields = dict(body.fields)
        caption = body.caption or {
            key.removeprefix("caption."): value
            for key, value in fields.items()
            if key.startswith("caption.")
        }
        signature = body.signature or {
            key.removeprefix("signature."): value
            for key, value in fields.items()
            if key.startswith("signature.")
        }
        artifact = build_template_document(
            template_id, title=body.title, caption=caption, sections=body.sections,
            signature=signature,
        )
        service = document_service()
        return service.import_document(
            artifact.info.filename,
            artifact.content,
            title=body.title or artifact.info.title,
            actor=actor(request),
            origin=f"template:{artifact.info.template_id}@{artifact.info.version}:{artifact.content_hash}",
        )
    except (StopIteration, ValueError) as exc:
        raise HTTPException(status_code=404, detail="template not found") from exc


@router.post("/v1/office-templates/{template_id}/instantiate", response_model=WorkDocument)
def instantiate_current_office_template(
    template_id: str, body: OfficeTemplateRequest, request: Request
) -> WorkDocument:
    return instantiate_office_template(template_id, TEMPLATE_VERSION, body, request)
