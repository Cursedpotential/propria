"""Retired legacy R2 upload route, retained as an explicit no-I/O refusal.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from fastapi import APIRouter, HTTPException

router = APIRouter(prefix="/api", tags=["upload"])


@router.post("/upload")
async def upload():
    """Refuse legacy R2 staging without parsing, hashing or storing the body.

    Inputs: any legacy upload request. Output: 410 with the canonical route.
    Effects: none; original staged records and provenance remain untouched.
    Pick /api/proffer/upload for fresh bounded content-addressed acquisition.
    """
    raise HTTPException(
        status_code=410,
        detail="Legacy R2 upload is retired; re-upload through the current flow (/api/proffer/upload).",
    )
