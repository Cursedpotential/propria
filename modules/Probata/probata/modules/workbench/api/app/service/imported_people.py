"""Resolve registry identities and format export and participant labels.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

The facade owns shared cache, configuration and upstream injection seams.
Inputs/outputs remain the existing Imported read contract; no writes occur here.
"""

from __future__ import annotations

import re
from typing import Any

from app.service import imported


class _People:
    """Phone numbers and names from the registry, the one identity store."""

    def __init__(self, rows: list[dict[str, Any]]) -> None:
        """Index supplied registry identifiers without performing I/O."""
        self.by_phone: dict[str, dict[str, str]] = {}
        self.by_person: dict[str, dict[str, str]] = {}
        self.candidates: dict[str, list[str]] = {}
        # Every person still marked unconfirmed (verification_state 'proposed'): a placeholder for a number
        # nobody named, or a person only a contact export named. Keyed by entity.
        self.unconfirmed: dict[str, dict[str, Any]] = {}
        self.names: list[tuple[str, dict[str, str]]] = []
        for row in rows:
            unconfirmed = row.get("verification_state") == "proposed"
            who = {
                "person": row["person"], "name": row["display_name"], "role": row["role_in_case"] or "",
                "entity_id": row.get("entity_id"), "unconfirmed": unconfirmed,
            }
            if unconfirmed:
                entry = self.unconfirmed.setdefault(
                    who["entity_id"], {"entity_id": who["entity_id"], "name": who["name"], "numbers": [], "emails": []})
                if row["kind"] == "phone" and imported._digits(row["identifier"] or "") not in entry["numbers"]:
                    entry["numbers"].append(imported._digits(row["identifier"] or ""))
                elif row["kind"] == "email" and row["identifier"] not in entry["emails"]:
                    entry["emails"].append(row["identifier"])
            else:
                self.by_person.setdefault(who["person"], who)
            if row["kind"] == "name" and row.get("alias_status") == "candidate" and row.get("entity_id"):
                # Names a contact export gave this number, kept unconfirmed for the owner to pick from.
                names = self.candidates.setdefault(row["entity_id"], [])
                if row.get("alias_text_raw") and row["alias_text_raw"] not in names:
                    names.append(row["alias_text_raw"])
            ident = (row["identifier"] or "").strip().lower()
            if row["kind"] == "phone":
                self.by_phone[imported._digits(ident)] = who
            elif row["kind"] == "name" and len(ident) >= 5 and ident not in {"owner"}:
                self.names.append((ident, who))

    def phone(self, value: str | None) -> dict[str, str] | None:
        """Resolve a supplied phone value against the indexed registry."""
        return self.by_phone.get(imported._digits(value or "")) if value else None

    def in_path(self, path: str) -> dict[str, str] | None:
        """Find the first indexed name embedded in an export path."""
        lowered = path.lower()
        for ident, who in self.names:
            if re.sub(r"[^a-z]", "", ident) and re.sub(r"[^a-z]", "", ident) in lowered:
                return who
        return None


def _digits(value: str) -> str:
    """Normalize a phone-like string to its final ten digits."""
    digits = re.sub(r"\D", "", value)
    return digits[-10:] if len(digits) >= 10 else digits


def looks_like_phone_value(value: str) -> bool:
    """Check the supported phone-number shape of a participant value.

    Input: participant text. Output: boolean; no I/O or registry mutation.
    Pick before interpreting participant text as a North American phone.
    """
    digits = re.sub(r"\D", "", value)
    return bool(re.fullmatch(r"\+?[\d\s().-]{10,16}", value)) and (len(digits) == 10 or (len(digits) == 11 and digits[0] == "1"))


def _pretty_phone(value: str) -> str:
    """Format a ten-digit phone value, leaving other values unchanged."""
    digits = imported._digits(value)
    if len(digits) == 10:
        return f"({digits[:3]}) {digits[3:6]}-{digits[6:]}"
    return value


def describe_export(export_key: str, people: imported._People) -> dict[str, Any]:
    """Format, device phone and owner, read from the casevault key of an export."""
    lowered = export_key.lower()
    if "sms-backup-restore-calls" in lowered:
        label = "Calls"
    elif "sms-backup-restore" in lowered:
        label = "SMS"
    elif "facebook" in lowered:
        label = "Facebook"
    else:
        label = "Other"
    device = imported._PATH_DEVICE.search(export_key)
    device_phone = device.group(2) if device else None
    owner = people.phone(device_phone) if device_phone else people.in_path(export_key.split("/messaging/")[-1])
    return {
        "format": label,
        "device": imported._pretty_phone(device_phone) if device_phone else None,
        "owner": owner["person"] if owner else None,
        "owner_name": owner["name"] if owner else None,
        "file_name": export_key.rsplit("/", 1)[-1],
        "casevault_key": export_key.split("casevault/", 1)[-1],
    }


def _participant(identifier: str, people: imported._People, owner: str | None) -> dict[str, Any]:
    """One participant: a name when the registry knows the number, else the number itself.

    `self` in an export is the phone's owner. `mine` marks the user's own side of a conversation
    (the registry person whose role is `user`), so a chat reads the same whichever phone it came from.
    """
    if identifier == "self":
        who = people.by_person.get(owner or "")
        label = who["name"] if who else "This phone"
    else:
        who = people.phone(identifier)
        looks_like_phone = imported.looks_like_phone_value(identifier)
        label = who["name"] if who else (imported._pretty_phone(identifier) if looks_like_phone else identifier)
    number = imported._digits(identifier) if identifier != "self" and imported.looks_like_phone_value(identifier) else None
    return {
        "id": identifier, "label": label, "mine": bool(who and who["role"] == "user"),
        "person": who["person"] if who else None,
        # The registry person this number belongs to, whether it is still a placeholder, and the
        # 10-digit number when nobody carries it yet (the "Who is this?" control starts from these).
        "entity_id": who.get("entity_id") if who else None,
        "unconfirmed": bool(who and who.get("unconfirmed")),
        "number": number if (number and (not who or who.get("unconfirmed"))) else None,
    }
