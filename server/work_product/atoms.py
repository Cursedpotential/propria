"""Source-bound mention atoms and non-lossy observation accounting.

This module is a pure, retry-safe unit. Callers must supply canonical source
occurrence/version and provenance-origin IDs; this unit does not discover them,
read source bytes, or promote candidates to evidence.

Byline: Codex · GPT-6 · 2026-09-23 (D06 bounded atom identity slice).
"""

from __future__ import annotations

from collections import defaultdict
from hashlib import sha256
import json
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


RecordClass = Literal[
    "EVENT",
    "CONDITION",
    "STATEMENT",
    "ARTIFACT",
    "AUTHORITY",
    "STRATEGY",
    "DECISION",
    "EXPOSURE",
    "PERSON",
    "OPEN",
]


class SourceSpan(BaseModel):
    """Character offsets in one exact source-version turn, end exclusive."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    turn_id: str = Field(min_length=1)
    char_start: int = Field(ge=0)
    char_end: int = Field(gt=0)

    @field_validator("turn_id")
    @classmethod
    def reject_blank_turn(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("turn ID must not be blank")
        return value

    @model_validator(mode="after")
    def check_bounds(self) -> SourceSpan:
        if self.char_end <= self.char_start:
            raise ValueError("source span end must exceed start")
        return self


class MentionAtom(BaseModel):
    """One candidate assertion from one source-version occurrence.

    ``mention_key`` disambiguates distinct assertions that share an exact span.
    It must be stable within that source version; a run/window number is invalid.
    The caller must retain the source bytes and verify spans against them.
    """

    model_config = ConfigDict(frozen=True, extra="forbid")

    source_occurrence_id: str = Field(min_length=1)
    source_version_id: str = Field(min_length=1)
    provenance_origin_id: str = Field(min_length=1)
    spans: tuple[SourceSpan, ...] = Field(min_length=1)
    mention_key: str = Field(min_length=1)
    record_class: RecordClass
    statement: str = Field(min_length=1)
    origin_class: Literal["HUMAN_SOURCE", "AI_ORIGIN"]

    @field_validator(
        "source_occurrence_id",
        "source_version_id",
        "provenance_origin_id",
        "mention_key",
        "statement",
    )
    @classmethod
    def reject_blank(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("atom fields must not be blank")
        return value

    @field_validator("spans")
    @classmethod
    def canonicalize_spans(cls, spans: tuple[SourceSpan, ...]) -> tuple[SourceSpan, ...]:
        return tuple(sorted(spans, key=lambda span: (span.turn_id, span.char_start, span.char_end)))

    @model_validator(mode="after")
    def reject_duplicate_spans(self) -> MentionAtom:
        if len(set(self.spans)) != len(self.spans):
            raise ValueError("duplicate source span")
        return self

    @property
    def atom_id(self) -> str:
        """Stable identity independent of run, window, or paraphrased statement."""

        coordinates = {
            "contract": "wp-mention-atom-v1",
            "source_occurrence_id": self.source_occurrence_id,
            "source_version_id": self.source_version_id,
            "spans": [span.model_dump(mode="json") for span in self.spans],
            "mention_key": self.mention_key,
        }
        encoded = json.dumps(coordinates, sort_keys=True, separators=(",", ":")).encode()
        return "wp-atom-v1:" + sha256(encoded).hexdigest()


class AtomObservation(BaseModel):
    """An immutable run/window observation of a candidate atom."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    run_id: str = Field(min_length=1)
    window_id: str = Field(min_length=1)
    atom: MentionAtom

    @field_validator("run_id", "window_id")
    @classmethod
    def reject_blank_coordinate(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("observation coordinate must not be blank")
        return value


class AtomAccounting(BaseModel):
    """One atom with all observations retained and origin scope made explicit."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    atom_id: str
    observation_count: int
    delivery_count: int
    observation_coordinates: tuple[tuple[str, str], ...]
    atom: MentionAtom


class ObservationConflict(BaseModel):
    """A disagreement that requires review, retained with exact run coordinates."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    atom_id: str
    observation_coordinates: tuple[tuple[str, str], ...]
    statements: tuple[str, ...]
    record_classes: tuple[RecordClass, ...]
    provenance_origin_ids: tuple[str, ...]
    observations: tuple[AtomObservation, ...]


class AtomReconciliation(BaseModel):
    """Deterministic accounting; conflicts never disappear into a winner."""

    model_config = ConfigDict(frozen=True, extra="forbid")

    atoms: tuple[AtomAccounting, ...]
    conflicts: tuple[ObservationConflict, ...]
    observations_seen: int


def reconcile_observations(observations: tuple[AtomObservation, ...]) -> AtomReconciliation:
    """Group overlap/retry observations without claiming independent corroboration.

    Each observation is retained in counts. Same-ID disagreements are surfaced as
    conflicts and are excluded from accepted atoms until explicitly resolved.
    """

    grouped: dict[str, list[AtomObservation]] = defaultdict(list)
    for observation in observations:
        grouped[observation.atom.atom_id].append(observation)

    atoms: list[AtomAccounting] = []
    conflicts: list[ObservationConflict] = []
    for atom_id, group in sorted(grouped.items()):
        group.sort(key=lambda item: (item.run_id, item.window_id, item.atom.model_dump_json()))
        coordinates = tuple(sorted({(item.run_id, item.window_id) for item in group}))
        statements = tuple(sorted({item.atom.statement for item in group}))
        record_classes = tuple(sorted({item.atom.record_class for item in group}))
        origins = tuple(sorted({item.atom.provenance_origin_id for item in group}))
        origin_classes = {item.atom.origin_class for item in group}
        if any(len(values) != 1 for values in (statements, record_classes, origins, origin_classes)):
            conflicts.append(
                ObservationConflict(
                    atom_id=atom_id,
                    observation_coordinates=coordinates,
                    statements=statements,
                    record_classes=record_classes,
                    provenance_origin_ids=origins,
                    observations=tuple(group),
                )
            )
            continue
        atoms.append(
            AtomAccounting(
                atom_id=atom_id,
                observation_count=len(coordinates),
                delivery_count=len(group),
                observation_coordinates=coordinates,
                atom=group[0].atom,
            )
        )
    return AtomReconciliation(atoms=tuple(atoms), conflicts=tuple(conflicts), observations_seen=len(observations))
