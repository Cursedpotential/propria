"""Exercise the installed API interpreter against a retained synthetic store.

Run inside an existing legal-api container. No production workspace is opened.
The /tmp fixture is retained for inspection; this script does not delete it.
"""

import json
import sqlite3
import sys
import tempfile
from datetime import UTC, datetime
from pathlib import Path
from uuid import uuid4

from legal_workspace.contracts.claims import (
    ClaimCreate, ClaimPatch, FollowupCreate, FromProbataCreate, GapCreate, LinkCreate,
    OriginReference,
)
from legal_workspace.contracts.source_package import LegalSourcePackage, LegalSourcePackageItem
from legal_workspace.services.claims import ClaimRevisionConflict, ClaimService

root = Path(tempfile.mkdtemp(prefix="advocatio-claims-runtime-"))
matter_id = uuid4()
package = LegalSourcePackage(
    package_id=uuid4(), matter_id=matter_id, manifest_hash="sha256:" + "a" * 64,
    created_at=datetime.now(UTC), items=[LegalSourcePackageItem(
        item_id=uuid4(), assertion_id=uuid4(), assertion_version=1,
        span_locator="synthetic:paragraph:1", custody_locator="synthetic:fixture-only",
        content_hash="sha256:" + "b" * 64, review_state="approved")],
)
origin = OriginReference(kind="event", record_id=str(uuid4()), record_version="synthetic-v1")
descriptor = {"origin": origin.model_dump(), "title": "Synthetic upstream title",
              "record": {"synthetic_original": "Never persisted"}}
service = ClaimService(root, matter_id, lambda: package, lambda kind, identity: descriptor)
claim = service.create(ClaimCreate(text="Synthetic runtime statement", response="Synthetic response"), actor="synthetic:test")
claim = service.add_link(claim.claim_id, LinkCreate(expected_revision=1, relationship="partial", source=service.sources().items[0]), actor="synthetic:test")
assert claim.evidence_status == "partially_supported"
claim = service.add_gap(claim.claim_id, GapCreate(expected_revision=2, description="Synthetic gap"), actor="synthetic:test")
claim = service.add_followup(claim.claim_id, FollowupCreate(expected_revision=3, kind="investigate", description="Synthetic planned action", gap_id=claim.gaps[0].gap_id), actor="synthetic:test")
assert len(service.gap_report()) == 1
assert len(service.history(claim.claim_id)) == 4
try:
    service.patch(claim.claim_id, ClaimPatch(expected_revision=1, response="Stale"), actor="synthetic:test")
except ClaimRevisionConflict:
    pass
else:
    raise AssertionError("Stale write was accepted")
assert service.by_origin(origin.kind, origin.record_id) is None
linked = service.from_probata(FromProbataCreate(origin=origin, response="Retained legal response"), actor="synthetic:test")
again = service.from_probata(FromProbataCreate(origin=origin), actor="synthetic:test")
assert again.claim_id == linked.claim_id and again.response == "Retained legal response"
assert service.by_origin(origin.kind, origin.record_id).claim_id == linked.claim_id
assert len(service.history(linked.claim_id)) == 1
descriptor["origin"]["record_version"] = "synthetic-v2"
current = service.get(linked.claim_id)
assert current.origin_state == "changed" and current.origin.record_version == "synthetic-v1"
assert current.response == "Retained legal response"
with sqlite3.connect(root / "legal.sqlite") as connection:
    snapshots = connection.execute("SELECT record_json FROM legal_claim_revision").fetchall()
assert all("Synthetic upstream title" not in row[0] and "Never persisted" not in row[0] for row in snapshots)
reopened = ClaimService(root, matter_id, lambda: package)
assert reopened.get(claim.claim_id).followups[0].status == "open"
assert reopened.get(linked.claim_id).origin_state == "unavailable"
assert reopened.by_origin(origin.kind, origin.record_id).response == "Retained legal response"
print(json.dumps({"python": sys.version.split()[0], "passed": True,
                  "synthetic_store": str(root), "production_records_written": 0}))
