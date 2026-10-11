"""Assemble deterministic episode context without inventing identity or clocks.

Inputs: immutable normalized record dictionaries and explicit grouping. Outputs:
bounded ordered episodes. Effects: none; choose native conversation fallback over
grouping unrelated messages solely because participants match.
"""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

from .storage import DiscoveryInvalid, fingerprint

MAX_RECORDS = 400
MAX_TEXT_CHARS = 512 * 1024


def timestamp(value: Any) -> datetime | None:
    """Parse an explicitly zoned clock while preserving absent or uncertain time.

    Inputs: ISO clock. Outputs: UTC datetime or None. Effects: none; choose over
    guessing a timezone or converting uncertain clocks to confident chronology.
    """
    if not isinstance(value, str):
        return None
    try:
        parsed = datetime.fromisoformat(value)
        return parsed.astimezone(UTC) if parsed.tzinfo else None
    except ValueError:
        return None


def order_feedback(feedback: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Order human corrections by recorded time independently of per-run revisions.

    Inputs: feedback with zoned created_at, identity and revision. Outputs:
    deterministic chronological corrections. Effects: none; choose cross-run
    timestamps over comparing revision counters from unrelated runs.
    """
    for item in feedback:
        if timestamp(item.get("created_at")) is None:
            raise DiscoveryInvalid("feedback requires a valid zoned created_at")
        if not isinstance(item.get("feedback_id"), str) or not item["feedback_id"]:
            raise DiscoveryInvalid("feedback identity is incomplete")
        revision = item.get("revision", 0)
        if isinstance(revision, bool) or not isinstance(revision, int) or revision < 0:
            raise DiscoveryInvalid("feedback revision is invalid")
    return sorted(
        feedback, key=lambda item: (timestamp(item["created_at"]), item["feedback_id"], item.get("revision", 0))
    )


def assemble_context(batch: dict[str, Any]) -> list[dict[str, Any]]:
    """Group explicit linked episodes and sort bounded native occurrences safely.

    Inputs: batch records/episodes/filters. Outputs: contexts with record identity,
    source locators, speaker/actor distinction and uncertain clocks retained.
    Effects: none; choose for scoring context, never canonical episode mutation.
    """
    records = batch.get("records") or []
    if not isinstance(records, list) or len(records) > MAX_RECORDS:
        raise DiscoveryInvalid("discovery record count exceeds bound")
    if any(not isinstance(record, dict) or not isinstance(record.get("text"), str) for record in records):
        raise DiscoveryInvalid("discovery record requires text")
    if sum(len(record["text"]) for record in records) > MAX_TEXT_CHARS:
        raise DiscoveryInvalid("discovery text exceeds bound")
    filters = batch.get("filters") or {}
    cutoff = timestamp(filters.get("cutoff"))
    if filters.get("perspective") == "as_lived" and cutoff is None:
        raise DiscoveryInvalid("as_lived discovery requires a zoned cutoff")
    explicit: dict[str, str] = {}
    ids = {record.get("record_id") for record in records if isinstance(record, dict)}
    for episode in batch.get("episodes") or []:
        if not isinstance(episode, dict) or not episode.get("episode_id"):
            raise DiscoveryInvalid("explicit episode identity is incomplete")
        for record_id in episode.get("record_ids") or []:
            if record_id not in ids or (record_id in explicit and explicit[record_id] != episode["episode_id"]):
                raise DiscoveryInvalid("explicit episode contains unknown or conflicting membership")
            explicit[record_id] = episode["episode_id"]
    groups: dict[str, list[dict[str, Any]]] = {}
    seen: set[tuple[str, int]] = set()
    for record in records:
        if not isinstance(record, dict) or not isinstance(record.get("text"), str):
            raise DiscoveryInvalid("discovery record requires text")
        for field in ("record_id", "source_version_id", "generation_id"):
            if not isinstance(record.get(field), str) or not record[field]:
                raise DiscoveryInvalid("discovery record identity is incomplete")
        ordinal = record.get("ordinal", 0)
        if isinstance(ordinal, bool) or not isinstance(ordinal, int) or ordinal < 0:
            raise DiscoveryInvalid("discovery record ordinal is invalid")
        occurrence = (record["record_id"], ordinal)
        if occurrence in seen:
            raise DiscoveryInvalid("discovery occurrence identity is duplicated")
        seen.add(occurrence)
        if cutoff is not None and filters.get("perspective") == "as_lived":
            available = timestamp(record.get("source_available_from"))
            occurred = timestamp(record.get("occurred_at"))
            if available is None or available > cutoff or occurred is None or occurred > cutoff:
                raise DiscoveryInvalid("record occurrence or availability falls outside as_lived cutoff")
        episode_id = explicit.get(record["record_id"])
        if not episode_id:
            native = record.get("conversation_id")
            key = "record:" + record["record_id"]
            if native:
                key = "conversation:" + record["source_version_id"] + ":" + native
            if record.get("explicit_episode_id"):
                key = "link:" + record["explicit_episode_id"]
            episode_id = "episode:" + fingerprint(key)[:32]
        groups.setdefault(episode_id, []).append(record)
    result = []
    for episode_id, members in sorted(groups.items()):
        ordered = sorted(
            members,
            key=lambda record: (
                timestamp(record.get("occurred_at")) is None,
                timestamp(record.get("occurred_at")) or datetime.max.replace(tzinfo=UTC),
                record.get("ordinal", 0),
                record["source_version_id"],
                record["record_id"],
            ),
        )
        result.append({"episode_id": episode_id, "records": ordered})
    return result


def attribution(records: list[dict[str, Any]], feedback: list[dict[str, Any]]) -> dict[str, Any]:
    """Keep native speakers separate from explicitly supplied behavioral actors.

    Inputs: episode records and scoped reviewed feedback. Outputs: attribution
    and speech form uncertainty. Effects: none; choose over extracting a person
    confidently from names, quotes or third-party reported threats.
    """
    speakers = sorted(
        {
            p["identifier"]
            for r in records
            for p in r.get("participants") or []
            if p.get("role") in {"speaker", "sender", "author"} and p.get("identifier")
        }
    )
    actors = sorted({actor for r in records for actor in r.get("attributed_actor_ids") or []})
    forms = {r.get("speech_form", "unknown") for r in records}
    for item in order_feedback(feedback):
        if item.get("kind") == "wrong_person":
            actors = sorted(set(item.get("attributed_actor_ids") or []))
        if item.get("kind") == "reported_speech":
            forms.add("reported")
    uncertainty = ["Candidate semantic relevance requires review; similarity is not proof."]
    if not actors:
        uncertainty.append("Behavioral actor is unknown; native speakers are separate.")
    if any(timestamp(r.get("occurred_at")) is None for r in records):
        uncertainty.append("Some source clocks are uncertain; original clocks and native order are retained.")
    if forms & {"reported", "quoted"}:
        uncertainty.append("Quoted or reported speech does not establish the speaker as behavioral actor.")
    return {
        "speaker_ids": speakers,
        "attributed_actor_ids": actors,
        "attribution_basis": "reviewed correction"
        if any(f.get("kind") == "wrong_person" for f in feedback)
        else "explicit native attribution"
        if actors
        else "unknown",
        "speech_forms": sorted(forms),
        "uncertainty": uncertainty,
    }
