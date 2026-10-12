"""Expose category-first whole extracted-package review through the existing BFF.

Inputs: real package handles, persisted categories and explicit owner actions.
Outputs: bounded lossless pages and durable receipts. Effects: existing readers
or correction writers only; source/chunk/search reads never await review.
"""
from fastapi import APIRouter, Header, HTTPException, Query, Request

from app.repo.imported_pg import ImportedError
from app.service import context_packages as service
from app.service.matter_mode import MatterModeError
from app.service.proffer_errors import ProfferError
from app.types.context_packages import PackageReview, PackageExtractionRetry

router = APIRouter(prefix="/api/context", tags=["context-package-review"])


@router.get("/packages/{handle}/attempts/{workflow_id}")
async def extraction_attempt(handle: str, workflow_id: str, request: Request):
    """Read source-bound retry progress without treating orchestration completion as extraction success.

    Inputs: package handle and actual retry workflow ID. Output: business status
    and retained result refs. Effects: authenticated Temporal read only.
    """
    try:
        return await service.extraction_attempt(handle, workflow_id, request)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)


@router.post("/packages/{handle}/retry", status_code=202)
async def retry_extraction(handle: str, body: PackageExtractionRetry, request: Request,
                           idempotency_key: str = Header(alias="Idempotency-Key")):
    """Start a source-pinned extraction attempt using the existing configured native model.

    Inputs: displayed version, configured model and actor click key. Output:
    actual Temporal attempt/parent references. Effects: bounded extraction/staging,
    with old attempts and durable owner corrections retained independently.
    """
    try:
        return await service.retry_extraction(handle, body, request, idempotency_key)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)


def _fail(error):
    """Translate existing reader/writer errors without exposing SQL or credentials.

    Input: sanitized service error. Output: HTTP exception. Effects: none.
    """
    raise HTTPException(getattr(error, "status_code", getattr(error, "status", 503)),
                        getattr(error, "detail", getattr(error, "message", "Package reader unavailable"))) from None


@router.get("/categories")
async def categories(cursor: str | None = Query(default=None, max_length=4096)):
    """Read persisted category sections before selecting any package.

    Input: bounded cursor. Output: category counts/continuation; effects: SELECT only.
    """
    try:
        return await service.categories(cursor)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)


@router.get("/packages")
async def packages(category: str = Query(min_length=1, max_length=256), domain: str = Query(default="", max_length=256),
                   cursor: str | None = Query(default=None, max_length=4096)):
    """Read actual source packages inside the selected persisted category.

    Inputs: category/domain/cursor. Output: package references and placement basis.
    Effects: bounded SELECT; this endpoint performs no source relocation.
    """
    try:
        return await service.packages(category, domain, cursor)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)


@router.get("/packages/{handle}")
async def package(handle: str, request: Request, section: str = Query(default="allmetadata", max_length=64),
                  cursor: str | None = Query(default=None, max_length=4096)):
    """Read one dynamic extracted-package section with all native fields and actual edges.

    Inputs: real handle/section/cursor. Output: requested snapshot DTO. Effects:
    existing native/versioned readers only; no review or custody prerequisite.
    """
    try:
        return await service.snapshot(handle, section, cursor, request)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)


@router.post("/packages/{handle}/review", status_code=201)
@router.patch("/packages/{handle}/review", status_code=201)
async def review(handle: str, body: PackageReview, request: Request, idempotency_key: str = Header(alias="Idempotency-Key")):
    """Append an explicit metadata/edit/accept/reject/interpretation-commit revision.

    Inputs: displayed revision, source locators, reason and existing actor/key.
    Output: durable correction receipt. Effects: existing Go metadata/candidate
    writers atomically; native candidate Commit records canonical analysis approval,
    and Accept remains held. This route performs no evidence promotion.
    """
    try:
        return await service.review(handle, body, request, idempotency_key)
    except (ImportedError, ProfferError, MatterModeError) as error:
        _fail(error)
