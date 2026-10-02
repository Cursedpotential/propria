#!/usr/bin/env python3
"""Import contact exports into the registry, through the governed case-identity API.

Byline: Claude Code · Sonnet · 2026-10-02

Order of operations (owner 2026-10-02 15:34): when a name is available a number is NOT a placeholder.
This runs FIRST, before any placeholder is made. The alias table IS the contact table:

  1. Every contact becomes an UNCONFIRMED person ('proposed', never confirmed here) whose display name is
     the name from the MOST RECENT export that names it. Every other name any export gave is kept as a
     candidate alias, so disagreeing exports never lose a name and nothing is a silent pick. Phones become
     aliases of the person, emails become candidate aliases, and each alias is marked
     "from contacts: <source file key>". Contacts that share a number or email are one person.
  2. A number that a person already carries is left with that person; the contact's names are added to
     it as candidates (and a bare "Unknown ..." placeholder is titled with the most recent name).
  3. Rows for the new people's numbers are linked in the engine's transaction (only NULL entity columns).
  4. Afterwards tools/backfill_placeholders.py makes a placeholder ONLY for numbers still carried by no
     person.

Nothing here writes a registry table. Every change is an existing governed call and lands in
registry.identity_change with the actor and the reason:
    GET  /api/imported/number-status            who carries these numbers (known / unconfirmed / nobody)
    POST /api/case-identity/contact-people      the new unconfirmed people, aliases and row links
    POST /api/case-identity/identifiers         candidate names for a person that already exists
    POST /api/case-identity/people/{id}         title a bare "Unknown ..." placeholder

INPUT. A manifest (JSON lines) written by tools/contacts_manifest.sh and the folder holding the files:
    {"key": "b2://.../contacts.vcf", "sha1": "...", "path": "files/ab12.vcf", "listed_at": "2026-09-01T00:00:00Z"}
Duplicates are dropped by sha1 (the first key wins). Formats: vCard (.vcf, the 'vobject' library), CSV
(Google / Outlook / generic headers) and Facebook / Instagram contact lists (JSON, any nesting).

    python3 tools/import_contacts.py --manifest manifest.jsonl --root ./out            # dry run
    python3 tools/import_contacts.py --manifest manifest.jsonl --root ./out --apply    # writes

Dry run parses, dedupes, looks the numbers up and asks the engine to build every person and link every
row, then ROLLS BACK; it prints the counts the real run will produce. Requires: pip install vobject
"""

from __future__ import annotations

import argparse
import csv
import io
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

DEFAULT_BASE = "https://workbench.tilapia-skilift.ts.net"
PHONE_RE = re.compile(r"^[2-9][0-9]{9}$")
BATCH = 100


def normalize_phone(raw: str) -> str | None:
    """The same rule as registry.norm_identifier for a US phone number."""
    if re.search(r"[A-Za-z@]", raw or ""):
        return None
    digits = re.sub(r"\D", "", raw or "")
    if len(digits) == 11 and digits[0] == "1":
        digits = digits[1:]
    return digits if PHONE_RE.match(digits) else None


def clean_name(value: str | None) -> str:
    return re.sub(r"\s+", " ", (value or "").strip())


# ----------------------------------------------------------------------------- parsing

class Contact:
    def __init__(self, name: str, phones: list[str], emails: list[str], source: str, listed_at: str = ""):
        self.name = name
        self.phones = sorted({p for p in (normalize_phone(x) for x in phones) if p})
        self.emails = sorted({e.strip().lower() for e in emails if "@" in (e or "")})
        self.source = source
        self.listed_at = listed_at


def parse_vcard(text: str, source: str) -> list[Contact]:
    try:
        import vobject  # off-the-shelf vCard parser
    except ImportError:
        raise SystemExit("vobject is required for vCard files: pip install vobject") from None
    out: list[Contact] = []
    for card in vobject.readComponents(text, ignoreUnreadable=True):
        name = clean_name(getattr(getattr(card, "fn", None), "value", "") or "")
        if not name and hasattr(card, "n"):
            n = card.n.value
            name = clean_name(f"{getattr(n, 'given', '')} {getattr(n, 'family', '')}")
        phones = [t.value for t in card.contents.get("tel", [])]
        emails = [e.value for e in card.contents.get("email", [])]
        out.append(Contact(name, phones, emails, source))
    return out


def parse_csv(text: str, source: str) -> list[Contact]:
    reader = csv.DictReader(io.StringIO(text))
    out: list[Contact] = []
    for row in reader:
        lowered = {(k or "").strip().lower(): (v or "").strip() for k, v in row.items()}
        name = lowered.get("name") or lowered.get("display name") or lowered.get("full name") or ""
        if not name:
            name = " ".join(x for x in (lowered.get("first name") or lowered.get("given name") or "",
                                        lowered.get("middle name", ""), lowered.get("last name") or lowered.get("family name") or "") if x)
        phones = [v for k, v in lowered.items() if v and re.search(r"phone|mobile|cell|tel", k) and "type" not in k and "label" not in k]
        emails = [v for k, v in lowered.items() if v and re.search(r"e-?mail", k) and "type" not in k and "label" not in k]
        phones = [p for value in phones for p in re.split(r"\s*:::\s*|;", value)]
        out.append(Contact(clean_name(name), phones, emails, source))
    return out


def parse_social_json(text: str, source: str) -> list[Contact]:
    """Facebook / Instagram contact lists: any object carrying a name and a phone or email."""
    out: list[Contact] = []

    def walk(node):
        if isinstance(node, dict):
            name = clean_name(node.get("name") or " ".join(x for x in (node.get("first_name"), node.get("last_name")) if x))
            point = node.get("contact_point") or node.get("phone") or node.get("phone_number") or node.get("email") or ""
            if name and point:
                out.append(Contact(name, [str(point)], [str(point)], source))
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(json.loads(text))
    return out


def parse_file(path: Path, source: str) -> list[Contact]:
    text = path.read_bytes().decode("utf-8", errors="replace")
    suffix = path.suffix.lower()
    if suffix in (".vcf", ".vcard"):
        return parse_vcard(text, source)
    if suffix == ".csv":
        return parse_csv(text, source)
    if suffix == ".json":
        return parse_social_json(text, source)
    return []


# ----------------------------------------------------------------------------- grouping into people

def build_people(contacts: list[Contact]) -> list[dict]:
    """Group contacts that share a number or email into one person.

    display_name is the name from the most recent export (listed_at, then key); candidate_names keeps
    every name any export gave; source is that most recent export's key.
    """
    parent: dict[str, str] = {}

    def find(x: str) -> str:
        while parent.setdefault(x, x) != x:
            parent[x] = parent[parent[x]]
            x = parent[x]
        return x

    named = [c for c in contacts if c.name and (c.phones or c.emails)]
    for contact in named:
        identifiers = contact.phones + contact.emails
        for other in identifiers[1:]:
            parent[find(other)] = find(identifiers[0])
        find(identifiers[0])
    groups: dict[str, list[Contact]] = {}
    for contact in named:
        groups.setdefault(find((contact.phones + contact.emails)[0]), []).append(contact)
    people = []
    for cards in groups.values():
        newest = max(cards, key=lambda c: (c.listed_at, c.source))
        names: list[str] = []
        for card in sorted(cards, key=lambda c: (c.listed_at, c.source), reverse=True):
            if card.name not in names:
                names.append(card.name)
        people.append({
            "display_name": newest.name,
            "numbers": sorted({p for c in cards for p in c.phones}),
            "emails": sorted({e for c in cards for e in c.emails}),
            "candidate_names": names,
            "source": newest.source,
            "sources": sorted({c.source for c in cards}),
        })
    people.sort(key=lambda p: (p["display_name"].lower(), p["numbers"]))
    return people


# ----------------------------------------------------------------------------- governed API

class Api:
    def __init__(self, base: str):
        self.base = base.rstrip("/")

    def call(self, method: str, path: str, body: dict | None = None, key: str | None = None) -> tuple[int, dict]:
        headers = {"Content-Type": "application/json"}
        if key:
            headers["Idempotency-Key"] = key
        data = json.dumps(body).encode() if body is not None else None
        request = urllib.request.Request(self.base + path, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(request, timeout=300) as response:
                return response.status, json.load(response)
        except urllib.error.HTTPError as error:
            try:
                payload = json.loads(error.read().decode())
            except ValueError:
                payload = {}
            return error.code, payload

    def status(self, numbers: list[str]) -> dict[str, dict]:
        """number -> {state: known | unconfirmed | unknown, entity_id, label}."""
        found: dict[str, dict] = {}
        for index in range(0, len(numbers), 50):
            query = urllib.parse.urlencode([("numbers", n) for n in numbers[index: index + 50]])
            code, payload = self.call("GET", f"/api/imported/number-status?{query}")
            if code != 200:
                raise SystemExit(f"number-status failed: HTTP {code} {payload}")
            for item in payload.get("items", {}).values():
                found[item["number"]] = item
        return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--manifest", required=True, help="JSON lines: key, sha1, path, listed_at")
    parser.add_argument("--root", required=True, help="folder the manifest paths are relative to")
    parser.add_argument("--base", default=DEFAULT_BASE)
    parser.add_argument("--apply", action="store_true", help="write (default is a dry run that rolls back)")
    args = parser.parse_args()

    root = Path(args.root)
    seen_sha: dict[str, str] = {}
    duplicates = files = 0
    contacts: list[Contact] = []
    for line in Path(args.manifest).read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        entry = json.loads(line)
        sha = (entry.get("sha1") or "").lower()
        if sha and sha in seen_sha:
            duplicates += 1
            continue
        seen_sha[sha] = entry["key"]
        path = root / entry["path"]
        if not path.is_file():
            print(f"missing file for {entry['key']}", file=sys.stderr)
            continue
        files += 1
        parsed = parse_file(path, entry["key"])
        for contact in parsed:
            contact.listed_at = str(entry.get("listed_at") or "")
        contacts += parsed

    people = build_people(contacts)
    no_identifier = sum(1 for c in contacts if c.name and not (c.phones or c.emails))
    all_numbers = sorted({n for p in people for n in p["numbers"]})
    print(f"{files} unique files ({duplicates} duplicates by sha1), {len(contacts)} contacts, {len(people)} people "
          f"({len(all_numbers)} distinct numbers); {no_identifier} contacts with no usable number or email were skipped")

    api = Api(args.base)
    state = api.status(all_numbers)
    carried = {n: v for n, v in state.items() if v["state"] != "unknown"}
    disagree = sum(1 for p in people if len(p["candidate_names"]) > 1)
    print(f"{len(carried)} numbers are already carried by a person; {len(all_numbers) - len(carried)} are not; "
          f"{disagree} people have more than one name across the exports (display name = most recent export, the rest kept as candidates)")

    stamp = time.strftime("%Y%m%dT%H%M%S")
    totals = {"created": 0, "skipped_all_carried": 0, "aliases": 0}
    linked: dict[str, int] = {}
    for index in range(0, len(people), BATCH):
        chunk = people[index: index + BATCH]
        payload = {
            "people": [{k: p[k] for k in ("display_name", "numbers", "emails", "candidate_names", "source")} for p in chunk],
            "change_reason": "contacts import: named by a contact export, unconfirmed (owner 2026-10-02)",
            "dry_run": not args.apply,
        }
        code, receipt = api.call("POST", "/api/case-identity/contact-people", payload, f"contacts-{stamp}-{index // BATCH:05d}")
        if code not in (200, 201):
            raise SystemExit(f"contact-people failed: HTTP {code} {receipt}")
        detail = receipt.get("detail") or {}
        for name in totals:
            totals[name] += int(detail.get(name, 0))
        for name, count in (detail.get("linked") or {}).items():
            linked[name] = linked.get(name, 0) + int(count)
        print(f"  batch {index // BATCH + 1}: created {detail.get('created', 0)}, all carried {detail.get('skipped_all_carried', 0)}")

    # Names for people that already exist: candidates, and a bare "Unknown ..." placeholder gets the newest name.
    attached = titled = already = 0
    if args.apply:
        for person in people:
            for number in person["numbers"]:
                match = carried.get(number)
                if not match:
                    continue
                for name in person["candidate_names"]:
                    code, _ = api.call("POST", "/api/case-identity/identifiers", {
                        "entity_id": match["entity_id"], "raw_value": name, "kind": "name", "status": "candidate", "period": None,
                        "basis": "from contacts: " + person["source"][:360],
                        "change_reason": "contacts import: candidate name from a contact export (unconfirmed)"},
                        f"contacts-{stamp}-alias-{attached + already:07d}")
                    if code in (200, 201):
                        attached += 1
                    elif code == 409:
                        already += 1
                if match["state"] == "unconfirmed" and match["label"].startswith("Unknown "):
                    code, _ = api.call("POST", f"/api/case-identity/people/{match['entity_id']}", {
                        "fields": {"display_name": person["display_name"]},
                        "change_reason": "contacts import: the most recent contact export's name (still unconfirmed)"},
                        f"contacts-{stamp}-title-{titled:07d}")
                    titled += code in (200, 201)
    print(json.dumps({"mode": "apply" if args.apply else "dry_run (rolled back)", **totals, "rows_linked": linked,
                      "names_added_to_existing_people": attached, "existing_names_already_present": already,
                      "placeholders_titled": titled}, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
