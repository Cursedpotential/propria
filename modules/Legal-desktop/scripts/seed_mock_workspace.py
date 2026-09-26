"""Load labeled MOCK rows so the motion path can be clicked through.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Not clerk-confirmed. Not court-safe. Re-runnable. Does not invent a
docket number or assigned judge.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import uuid4

import httpx

API = "http://127.0.0.1:8010"


def main() -> None:
    client = httpx.Client(base_url=API, timeout=10.0)
    matter = client.get("/v1/matter").raise_for_status().json()
    matter_id = matter["matter"]["matter_id"]
    assertion_id = uuid4()
    package_id = uuid4()
    now = datetime.now(UTC)

    package = {
        "package_id": str(package_id),
        "manifest_hash": "sha256:mock-ui-motion-path",
        "matter_id": matter_id,
        "created_at": now.isoformat(),
        "items": [
            {
                "item_id": str(uuid4()),
                "assertion_id": str(assertion_id),
                "assertion_version": 1,
                "span_locator": "sms:mock:exchange-location",
                "custody_locator": "h1:mock-ui",
                "content_hash": "sha256:mock-span",
                "review_state": "approved",
            },
            {
                "item_id": str(uuid4()),
                "assertion_id": str(uuid4()),
                "assertion_version": 1,
                "span_locator": "sms:mock:unreviewed",
                "custody_locator": "h1:mock-skip",
                "content_hash": "sha256:mock-skip",
                "review_state": "candidate",
            },
        ],
    }
    imported = client.post("/v1/legal-source-packages:import", json=package).raise_for_status().json()
    print("import", imported)

    cite = client.post(
        "/v1/factors/j/citations",
        json={
            "letter": "j",
            "side": "petitioner",
            "package_id": str(package_id),
            "citation": {
                "statement": "MOCK: parenting-time exchange was at the ordered public location.",
                "package_id": str(package_id),
                "assertion_id": str(assertion_id),
                "assertion_version": 1,
                "span_locator": "sms:mock:exchange-location",
                "epistemic_class": "established_fact",
            },
        },
    )
    print("cite", cite.status_code, cite.text[:200])

    drafted = client.post(
        "/v1/drafts",
        json={
            "letter": "j",
            "heading": "MOCK Factor (j) — facilitation (test only)",
            "body": (
                "MOCK TEST ROW. Parenting-time exchange occurred at the ordered "
                "public location. Replace this with clerk-confirmed facts before filing."
            ),
        },
    ).raise_for_status().json()
    print("draft", drafted["section_id"])

    for template_id in (
        "motion-parenting-time-specific",
        "affidavit",
        "proposed-order",
        "notice-of-hearing",
        "proof-of-service",
    ):
        created = client.post(
            "/v1/templates:instantiate",
            json={"template_id": template_id, "heading": f"MOCK {template_id}"},
        )
        print("template", template_id, created.status_code)

    review = client.post(
        "/v1/reviews",
        json={
            "section_id": drafted["section_id"],
            "verdict": "approve",
            "rationale": "MOCK UI path only. Not a real owner approval for filing.",
            "reviewer": "owner",
        },
    )
    print("review", review.status_code, review.text[:200])

    release = client.post(
        "/v1/releases",
        json={"section_ids": [drafted["section_id"]]},
    ).raise_for_status().json()
    print("release", release.get("blocked"), (release.get("manifest") or {}).get("content_hash"))

    events = [
        {
            "occurs_at": (now - timedelta(days=21)).isoformat(),
            "title": "MOCK — last FOC contact (test row, not clerk-confirmed)",
            "kind": "foc",
            "detail": "Placeholder historic event so TIML is not empty.",
            "location": "Genesee FOC — MOCK",
            "source": "mock",
            "confirmed": False,
        },
        {
            "occurs_at": (now + timedelta(days=10)).isoformat(),
            "title": "MOCK — target file date for specific-terms motion (not a hearing)",
            "kind": "deadline",
            "detail": "Owner working date. Change or delete. Not set by the court.",
            "location": "",
            "source": "mock",
            "confirmed": False,
        },
        {
            "occurs_at": (now + timedelta(days=28)).isoformat(),
            "title": "MOCK — placeholder hearing slot (replace with clerk-set date)",
            "kind": "hearing",
            "detail": "Not a real setting. Remove when the clerk gives a date.",
            "location": "900 S. Saginaw St., Flint — MOCK",
            "source": "mock",
            "confirmed": False,
        },
    ]
    for body in events:
        posted = client.post("/v1/docket-events", json=body)
        print("event", posted.status_code, body["title"][:40])

    home = client.get("/v1/matter").raise_for_status().json()
    print(
        "home",
        {
            "package": home["package_imported"],
            "drafts": home["draft_count"],
            "upcoming": home["upcoming_event_count"],
            "reviews": home["review_count"],
            "releases": home["release_count"],
        },
    )


if __name__ == "__main__":
    main()
