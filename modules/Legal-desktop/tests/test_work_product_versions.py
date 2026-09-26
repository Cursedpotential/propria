"""Released work products are immutable; edits fork.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from uuid import uuid4

from legal_workspace.domain.work_product import (
    WorkProductState,
    WorkProductVersion,
    edit_creates_new_version,
)


def test_edit_of_released_document_forks_new_id() -> None:
    released = WorkProductVersion(
        work_product_id=uuid4(),
        version=3,
        state=WorkProductState.RELEASED,
        content_hash="sha256:rel",
    )
    next_version = edit_creates_new_version(released)
    assert next_version.work_product_id != released.work_product_id
    assert next_version.version == 4
    assert next_version.state is WorkProductState.PRIVATE_DRAFT
    assert released.state is WorkProductState.RELEASED
    assert released.version == 3
