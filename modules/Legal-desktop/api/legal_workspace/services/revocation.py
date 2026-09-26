"""Evidence revocation and quarantine mark dependent work stale.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from legal_workspace.contracts.events import EventEnvelope
from legal_workspace.domain.work_product import WorkProductState, WorkProductVersion

_STALE_TRIGGERS = {
    "evidence.release.revoked.v1",
    "evidence.source.quarantined.v1",
    "evidence.assertion.superseded.v1",
}


def apply_evidence_event(
    event: EventEnvelope,
    products: list[WorkProductVersion],
) -> list[WorkProductVersion]:
    """Fail-closed on unknown schemas. Mark dependents stale. Never mutate released rows in place."""
    event.fail_closed_if_unknown()
    if event.event_type not in _STALE_TRIGGERS:
        return list(products)

    affected_package = event.payload.get("package_id")
    affected_assertion = event.payload.get("assertion_id")
    updated: list[WorkProductVersion] = []
    for product in products:
        depends = False
        if affected_package and str(product.source_package_id) == str(affected_package):
            depends = True
        if affected_assertion and any(
            str(aid) == str(affected_assertion) for aid in product.cited_assertion_ids
        ):
            depends = True
        if not depends:
            updated.append(product)
            continue
        if product.state is WorkProductState.RELEASED:
            updated.append(product.model_copy(update={"state": WorkProductState.STALE}))
        else:
            updated.append(
                product.model_copy(update={"state": WorkProductState.REVALIDATION_REQUIRED})
            )
    return updated
