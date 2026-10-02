"""Durable claim revisions and actions in the workspace's existing SQLite file.

Source locators refer to the currently imported package. No evidence bytes or
acceptance receipts are created here. Every mutation and revision is one commit.
"""

import sqlite3
from collections.abc import Callable
from contextlib import contextmanager
from datetime import UTC, datetime
from pathlib import Path
from uuid import UUID, uuid4

from pydantic import ValidationError

from legal_workspace.contracts.claims import (
    ClaimCreate,
    ClaimFollowup,
    ClaimGap,
    ClaimPatch,
    ClaimRecord,
    ClaimRevision,
    EvidenceLink,
    FollowupCreate,
    FollowupPatch,
    FromProbataCreate,
    GapCreate,
    GapPatch,
    GapReportRow,
    LinkCreate,
    LinkPatch,
    OriginRecord,
    OriginReference,
    SourceOptions,
    SourceReference,
)
from legal_workspace.contracts.source_package import LegalSourcePackage, ReviewState


class ClaimNotFound(KeyError):
    pass


class ClaimRevisionConflict(ValueError):
    def __init__(self, current_revision: int):
        self.current_revision = current_revision
        super().__init__("The claim has changed. Reload before saving.")


class ClaimService:
    def __init__(
        self,
        store_dir: Path,
        matter_id: UUID | str,
        package_loader: Callable[[], LegalSourcePackage | None],
        record_loader: Callable[[str, str], dict | None] | None = None,
    ):
        self.matter_id = UUID(str(matter_id))
        self.package_loader = package_loader
        self.record_loader = record_loader
        Path(store_dir).mkdir(parents=True, exist_ok=True)
        self.db_path = Path(store_dir) / "legal.sqlite"
        with self._connect() as conn:
            conn.executescript("""
                CREATE TABLE IF NOT EXISTS legal_claim (
                    claim_id TEXT PRIMARY KEY, matter_id TEXT NOT NULL,
                    revision INTEGER NOT NULL, record_json TEXT NOT NULL);
                CREATE INDEX IF NOT EXISTS legal_claim_matter ON legal_claim(matter_id);
                CREATE TABLE IF NOT EXISTS legal_claim_revision (
                    claim_id TEXT NOT NULL REFERENCES legal_claim(claim_id),
                    revision INTEGER NOT NULL, actor TEXT NOT NULL,
                    action TEXT NOT NULL, occurred_at TEXT NOT NULL,
                    record_json TEXT NOT NULL, PRIMARY KEY(claim_id, revision));
            """)
            conn.execute("BEGIN IMMEDIATE")
            columns = {row["name"] for row in conn.execute("PRAGMA table_info(legal_claim)")}
            for column in ("origin_system", "origin_kind", "origin_id"):
                if column not in columns:
                    conn.execute(f"ALTER TABLE legal_claim ADD COLUMN {column} TEXT")
            conn.execute("""CREATE UNIQUE INDEX IF NOT EXISTS legal_claim_origin
                ON legal_claim(matter_id, origin_system, origin_kind, origin_id)
                WHERE origin_id IS NOT NULL""")

    @contextmanager
    def _connect(self):
        conn = sqlite3.connect(self.db_path, timeout=30)
        conn.row_factory = sqlite3.Row
        conn.execute("PRAGMA foreign_keys=ON")
        conn.execute("PRAGMA synchronous=FULL")
        try:
            with conn:
                yield conn
        finally:
            conn.close()

    def sources(self) -> SourceOptions:
        package = self.package_loader()
        if package is None or package.matter_id != self.matter_id:
            return SourceOptions(
                available=False, reason="No accepted evidence package is imported."
            )
        items = []
        for item in package.items:
            if item.review_state != ReviewState.APPROVED or item.permitted_use != "legal_drafting":
                continue
            try:
                items.append(
                    SourceReference(
                        package_id=package.package_id,
                        manifest_hash=package.manifest_hash,
                        **item.model_dump(exclude={"review_state", "permitted_use"}),
                    )
                )
            except ValidationError:
                # Older imported state may lack a complete locator. It is never a usable link.
                continue
        return SourceOptions(
            available=bool(items),
            items=items,
            reason=None if items else "No accepted evidence spans are available.",
        )

    def _validate_source(self, source: SourceReference):
        if source not in self.sources().items:
            raise ValueError(
                "Source does not match an accepted item and exact span in the current package."
            )

    def _project(self, record: ClaimRecord, *, resolve_origin: bool = True) -> ClaimRecord:
        record.origin_record = None
        record.origin_state = "none" if record.origin is None else "unavailable"
        if record.origin is not None and resolve_origin:
            try:
                record.origin_record = self._origin_record(record.origin)
                record.origin_state = (
                    "unchanged" if record.origin == record.origin_record.origin else "changed"
                )
            except ValueError:
                pass
        options = self.sources().items
        for link in record.links:
            link.validation_status = (
                "context"
                if link.source is None
                else "accepted"
                if link.source in options
                else "unavailable"
            )
        active = [link for link in record.links if link.active]
        relations = {link.relationship for link in active if link.validation_status == "accepted"}
        if "contradicts" in relations:
            record.evidence_status = "conflicting_evidence"
        elif "partial" in relations:
            record.evidence_status = "partially_supported"
        elif "supports" in relations:
            record.evidence_status = "evidence_linked"
        elif active and all(link.relationship == "context" for link in active):
            record.evidence_status = "context_only"
        else:
            record.evidence_status = "evidence_needed"
        return record

    def _origin_record(self, origin: OriginReference) -> OriginRecord:
        if self.record_loader is None:
            raise ValueError("Probata record reader is unavailable.")
        try:
            descriptor = self.record_loader(origin.kind, origin.record_id)
            if not isinstance(descriptor, dict):
                raise TypeError("Probata record is unavailable.")
            current = OriginRecord(
                origin=descriptor["origin"],
                title=descriptor.get("title", ""),
                payload=descriptor.get("payload", descriptor.get("record", {})),
            )
        except Exception as exc:
            raise ValueError("Probata record is unavailable.") from exc
        if (current.origin.system, current.origin.kind, current.origin.record_id) != (
            origin.system,
            origin.kind,
            origin.record_id,
        ):
            raise ValueError("Probata returned a different record identity.")
        return current

    @staticmethod
    def _persisted(record: ClaimRecord) -> str:
        return record.model_dump_json(exclude={"origin_state", "origin_record"})

    def _get(self, conn, claim_id: UUID | str) -> ClaimRecord:
        row = conn.execute(
            "SELECT record_json FROM legal_claim WHERE claim_id=? AND matter_id=?",
            (str(claim_id), str(self.matter_id)),
        ).fetchone()
        if row is None:
            raise ClaimNotFound(str(claim_id))
        return ClaimRecord.model_validate_json(row["record_json"])

    def get(self, claim_id: UUID | str) -> ClaimRecord:
        with self._connect() as conn:
            return self._project(self._get(conn, claim_id))

    def history(self, claim_id: UUID | str) -> list[ClaimRevision]:
        with self._connect() as conn:
            self._get(conn, claim_id)
            return [
                ClaimRevision(
                    revision=row["revision"],
                    actor=row["actor"],
                    action=row["action"],
                    occurred_at=row["occurred_at"],
                    record=ClaimRecord.model_validate_json(row["record_json"]),
                )
                for row in conn.execute(
                    "SELECT * FROM legal_claim_revision WHERE claim_id=? ORDER BY revision DESC",
                    (str(claim_id),),
                )
            ]

    def list(self) -> list[ClaimRecord]:
        with self._connect() as conn:
            records = [
                self._project(ClaimRecord.model_validate_json(row[0]))
                for row in conn.execute(
                    "SELECT record_json FROM legal_claim WHERE matter_id=? ORDER BY rowid DESC",
                    (str(self.matter_id),),
                )
            ]
        return records

    def _revision(self, conn, record: ClaimRecord, actor: str, action: str):
        conn.execute(
            "INSERT INTO legal_claim_revision VALUES (?,?,?,?,?,?)",
            (
                str(record.claim_id),
                record.revision,
                actor,
                action,
                record.updated_at.isoformat(),
                self._persisted(record),
            ),
        )

    def create(self, body: ClaimCreate, *, actor: str) -> ClaimRecord:
        if body.origin is not None:
            raise ValueError("Use the Probata record route to open its legal review.")
        stamp = datetime.now(UTC)
        record = ClaimRecord(
            **body.model_dump(),
            claim_id=uuid4(),
            matter_id=self.matter_id,
            revision=1,
            created_at=stamp,
            updated_at=stamp,
        )
        with self._connect() as conn:
            conn.execute("BEGIN IMMEDIATE")
            conn.execute(
                "INSERT INTO legal_claim(claim_id,matter_id,revision,record_json) VALUES (?,?,?,?)",
                (str(record.claim_id), str(self.matter_id), 1, self._persisted(record)),
            )
            self._revision(conn, record, actor, "created")
        return record

    def from_probata(self, body: FromProbataCreate, *, actor: str) -> ClaimRecord:
        current = self._origin_record(body.origin)
        with self._connect() as conn:
            conn.execute("BEGIN IMMEDIATE")
            row = conn.execute(
                """SELECT record_json FROM legal_claim WHERE matter_id=?
                AND origin_system=? AND origin_kind=? AND origin_id=?""",
                (
                    str(self.matter_id),
                    body.origin.system,
                    body.origin.kind,
                    body.origin.record_id,
                ),
            ).fetchone()
            if row is not None:
                record = ClaimRecord.model_validate_json(row["record_json"])
            else:
                if current.origin != body.origin:
                    raise ValueError(
                        "Probata record version changed. Refresh before opening its review."
                    )
                stamp = datetime.now(UTC)
                record = ClaimRecord(
                    text="Legal review of this " + body.origin.kind,
                    kind="question",
                    response=body.response,
                    origin=body.origin,
                    claim_id=uuid4(),
                    matter_id=self.matter_id,
                    revision=1,
                    created_at=stamp,
                    updated_at=stamp,
                )
                conn.execute(
                    """INSERT INTO legal_claim(claim_id,matter_id,revision,record_json,
                    origin_system,origin_kind,origin_id) VALUES (?,?,?,?,?,?,?)""",
                    (
                        str(record.claim_id),
                        str(self.matter_id),
                        1,
                        self._persisted(record),
                        body.origin.system,
                        body.origin.kind,
                        body.origin.record_id,
                    ),
                )
                self._revision(conn, record, actor, "probata_review_opened")
        record.origin_record = current
        record.origin_state = "unchanged" if record.origin == current.origin else "changed"
        return record

    def _mutate(
        self, claim_id, expected_revision: int, actor: str, action: str, edit
    ) -> ClaimRecord:
        with self._connect() as conn:
            conn.execute("BEGIN IMMEDIATE")
            record = self._get(conn, claim_id)
            if record.revision != expected_revision:
                raise ClaimRevisionConflict(record.revision)
            edit(record)
            record.revision += 1
            record.updated_at = datetime.now(UTC)
            # Remote reads must not hold the SQLite writer lock. Evidence
            # status still belongs to the atomic revision snapshot.
            self._project(record, resolve_origin=False)
            conn.execute(
                "UPDATE legal_claim SET revision=?,record_json=? WHERE claim_id=?",
                (record.revision, self._persisted(record), str(record.claim_id)),
            )
            self._revision(conn, record, actor, action)
        return self._project(record)

    def patch(self, claim_id, body: ClaimPatch, *, actor: str):
        def edit(record):
            if body.text is not None and body.text != record.text:
                for link in record.links:
                    link.active = False
            for key, value in body.model_dump(
                exclude_unset=True, exclude={"expected_revision"}
            ).items():
                setattr(record, key, value)

        return self._mutate(claim_id, body.expected_revision, actor, "edited", edit)

    def add_link(self, claim_id, body: LinkCreate, *, actor: str):
        def edit(record):
            if body.source is not None:
                self._validate_source(body.source)
            record.links.append(
                EvidenceLink(
                    **body.model_dump(exclude={"expected_revision"}),
                    link_id=uuid4(),
                    validation_status="accepted" if body.source else "context",
                    created_at=datetime.now(UTC),
                )
            )

        return self._mutate(claim_id, body.expected_revision, actor, "link_added", edit)

    def add_gap(self, claim_id, body: GapCreate, *, actor: str):
        return self._mutate(
            claim_id,
            body.expected_revision,
            actor,
            "gap_added",
            lambda record: record.gaps.append(
                ClaimGap(gap_id=uuid4(), description=body.description, created_at=datetime.now(UTC))
            ),
        )

    def patch_link(self, claim_id, link_id, body: LinkPatch, *, actor: str):
        def edit(record):
            for link in record.links:
                if str(link.link_id) == str(link_id):
                    if body.active and link.source is not None:
                        self._validate_source(link.source)
                    link.active = body.active
                    return
            raise ClaimNotFound(str(link_id))

        return self._mutate(claim_id, body.expected_revision, actor, "link_updated", edit)

    def patch_gap(self, claim_id, gap_id, body: GapPatch, *, actor: str):
        return self._mutate(
            claim_id,
            body.expected_revision,
            actor,
            "gap_updated",
            lambda record: self._set_status(record.gaps, "gap_id", gap_id, body.status),
        )

    def add_followup(self, claim_id, body: FollowupCreate, *, actor: str):
        def edit(record):
            if body.gap_id is not None and not any(g.gap_id == body.gap_id for g in record.gaps):
                raise ValueError("The follow-up gap does not belong to this claim.")
            record.followups.append(
                ClaimFollowup(
                    **body.model_dump(exclude={"expected_revision"}),
                    followup_id=uuid4(),
                    created_at=datetime.now(UTC),
                )
            )

        return self._mutate(claim_id, body.expected_revision, actor, "followup_added", edit)

    def patch_followup(self, claim_id, followup_id, body: FollowupPatch, *, actor: str):
        return self._mutate(
            claim_id,
            body.expected_revision,
            actor,
            "followup_updated",
            lambda record: self._set_status(
                record.followups, "followup_id", followup_id, body.status
            ),
        )

    @staticmethod
    def _set_status(records, id_field: str, record_id, status: str):
        for record in records:
            if str(getattr(record, id_field)) == str(record_id):
                record.status = status
                return
        raise ClaimNotFound(str(record_id))

    def gap_report(self) -> list[GapReportRow]:
        rows = []
        for claim in self.list():
            open_gaps = [gap for gap in claim.gaps if gap.status == "open"]
            for gap in open_gaps:
                rows.append(
                    GapReportRow(
                        claim_id=claim.claim_id,
                        claim_text=claim.text,
                        revision=claim.revision,
                        evidence_status=claim.evidence_status,
                        gap_id=gap.gap_id,
                        description=gap.description,
                        followups=[f for f in claim.followups if f.gap_id == gap.gap_id],
                    )
                )
            if (
                claim.evidence_status != "evidence_linked"
                and not open_gaps
                and claim.kind not in {"question", "theory"}
            ):
                rows.append(
                    GapReportRow(
                        claim_id=claim.claim_id,
                        claim_text=claim.text,
                        revision=claim.revision,
                        evidence_status=claim.evidence_status,
                        description={
                            "partially_supported": "Additional evidence needed.",
                            "conflicting_evidence": "Conflicting evidence needs review.",
                        }.get(claim.evidence_status, "Evidence needed."),
                        followups=claim.followups,
                    )
                )
        return rows
