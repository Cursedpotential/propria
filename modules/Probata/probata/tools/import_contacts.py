#!/usr/bin/env python3
"""Import contact exports into the registry, through the governed case-identity API.

Byline: Claude Code · Sonnet · 2026-10-02

Owner ruling: the alias table IS the contact table. Each contact either becomes a placeholder person
carrying its phone numbers or names the placeholder that already carries them; every name is recorded
as an alias marked "from contacts: <source file key>" and stays a CANDIDATE, and a person that was
only named by a contact stays 'proposed' (unconfirmed) until the owner confirms it. A number that
several exports name differently keeps EVERY name as a candidate and its display name is not changed
(never a silent pick).

Nothing here writes a registry table. Every change is one of the Workbench's existing governed calls:
    GET  /api/imported/number-status            who carries these numbers (known / placeholder / nobody)
    POST /api/case-identity/placeholders        a placeholder person per unknown number (links the rows)
    POST /api/case-identity/identifiers         the name / email as a candidate alias, with its source
    POST /api/case-identity/people/{id}         display name of a placeholder named by exactly one contact
and each lands in registry.identity_change with the actor and the reason.

INPUT. A manifest (JSON lines) of the exports and a folder holding them:
    {"key": "b2://salem-data/.../contacts.vcf", "sha1": "...", "path": "contacts.vcf"}
Produce it from the catalog (casebible raw_duck.b2_objects) and fetch the files on a VPS (rclone copy by
the listed keys) - this script reads local files only. Duplicates are dropped by sha1 (the first key wins
and the others are reported). Inspect the catalog's columns first; names below are the ones to adjust:
    SELECT key, sha1 FROM raw_duck.b2_objects
    WHERE key ~* '\\.(vcf|csv)$' OR key ~* '(facebook|instagram).*(contact)' ;

FORMATS. vCard (.vcf, 'vobject' library), CSV (Google / Outlook / generic headers), Facebook and
Instagram contact lists (JSON, any nesting: objects that carry a name and a phone or email).

    python3 tools/import_contacts.py --manifest manifest.jsonl --root ./files            # dry run
    python3 tools/import_contacts.py --manifest manifest.jsonl --root ./files --apply    # writes

Dry run parses, dedupes and looks every number up, then prints the plan and the counts. It makes no
write call. Requires: pip install vobject
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
from collections import defaultdict
from pathlib import Path

DEFAULT_BASE = "https://workbench.tilapia-skilift.ts.net"
PHONE_RE = re.compile(r"^[2-9][0-9]{9}$")


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
    def __init__(self, name: str, phones: list[str], emails: list[str], source: str):
        self.name = name
        self.phones = sorted({p for p in (normalize_phone(x) for x in phones) if p})
        self.emails = sorted({e.strip().lower() for e in emails if "@" in (e or "")})
        self.source = source


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
        """number -> {state: known | placeholder | unknown, entity_id}, from the Workbench's registry read."""
        found: dict[str, dict] = {}
        for index in range(0, len(numbers), 50):
            query = urllib.parse.urlencode([("numbers", n) for n in numbers[index: index + 50]])
            code, payload = self.call("GET", f"/api/imported/number-status?{query}")
            if code != 200:
                raise SystemExit(f"number-status failed: HTTP {code} {payload}")
            for item in payload.get("items", {}).values():
                found[item["number"]] = item
        return found


# ----------------------------------------------------------------------------- the plan

def build_plan(contacts: list[Contact]) -> dict[str, dict]:
    """number -> {names: {name: [sources]}, emails: {email: [sources]}}"""
    plan: dict[str, dict] = defaultdict(lambda: {"names": defaultdict(set), "emails": defaultdict(set)})
    for contact in contacts:
        for number in contact.phones:
            if contact.name:
                plan[number]["names"][contact.name].add(contact.source)
            for email in contact.emails:
                plan[number]["emails"][email].add(contact.source)
    return plan


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--manifest", required=True, help="JSON lines: key, sha1, path")
    parser.add_argument("--root", required=True, help="folder the manifest paths are relative to")
    parser.add_argument("--base", default=DEFAULT_BASE)
    parser.add_argument("--apply", action="store_true", help="write (default is a dry run)")
    args = parser.parse_args()

    root = Path(args.root)
    seen_sha: dict[str, str] = {}
    duplicates = 0
    contacts: list[Contact] = []
    files = 0
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
        contacts += parse_file(path, entry["key"])

    plan = build_plan(contacts)
    no_phone = sum(1 for c in contacts if not c.phones)
    print(f"{files} unique files ({duplicates} duplicates by sha1), {len(contacts)} contacts, {len(plan)} distinct phone numbers, {no_phone} contacts without a phone (skipped)")

    api = Api(args.base)
    state = api.status(sorted(plan))
    carried = {n: v for n, v in state.items() if v["state"] != "unknown"}
    unknown = [n for n in plan if n not in carried]
    conflicts = {n: p for n, p in plan.items() if len(p["names"]) > 1}
    print(f"{len(carried)} numbers already carried by a registry person, {len(unknown)} not carried yet, {len(conflicts)} numbers with more than one contact name")
    alias_writes = sum(len(p["names"]) + len(p["emails"]) for p in plan.values())
    print(f"plan: {len(unknown)} placeholders, {alias_writes} candidate aliases, "
          f"{sum(1 for p in plan.values() if len(p['names']) == 1)} placeholder display names set from a single contact")
    if not args.apply:
        print("dry run: nothing written. Re-run with --apply to write.")
        return 0

    stamp = time.strftime("%Y%m%dT%H%M%S")
    counter = 0

    def key(kind: str) -> str:
        nonlocal counter
        counter += 1
        return f"contacts-{stamp}-{kind}-{counter:06d}"

    # 1. a placeholder for every number nobody carries (this links the imported rows too)
    for index in range(0, len(unknown), 200):
        status, payload = api.call("POST", "/api/case-identity/placeholders",
                                   {"numbers": unknown[index: index + 200], "change_reason": "contacts import: number named by a contact export, not yet carried"},
                                   key("placeholders"))
        if status not in (200, 201):
            raise SystemExit(f"placeholders failed: HTTP {status} {payload}")
    state = api.status(sorted(plan))
    carried = {n: v for n, v in state.items() if v["state"] != "unknown"}

    # 2. names and emails as candidate aliases; a single unambiguous name also titles a placeholder
    written = skipped = failed = renamed = 0
    for number, data in sorted(plan.items()):
        match = carried.get(number)
        if not match:
            failed += 1
            continue
        entity = match["entity_id"]
        for kind, values in (("name", data["names"]), ("email", data["emails"])):
            for value, sources in sorted(values.items()):
                status, payload = api.call("POST", "/api/case-identity/identifiers", {
                    "entity_id": entity, "raw_value": value, "kind": kind, "status": "candidate", "period": None,
                    "basis": "from contacts: " + "; ".join(sorted(sources))[:380],
                    "change_reason": "contacts import: candidate from a contact export (unconfirmed)",
                }, key("alias"))
                if status in (200, 201):
                    written += 1
                elif status == 409:
                    skipped += 1  # this person already carries that spelling
                else:
                    failed += 1
                    print(f"alias failed for {number}: HTTP {status} {payload}", file=sys.stderr)
        if len(data["names"]) == 1 and match["state"] == "placeholder":
            (only,) = data["names"]
            status, _ = api.call("POST", f"/api/case-identity/people/{entity}",
                                 {"fields": {"display_name": only}, "change_reason": "contacts import: the only contact name for this number (still unconfirmed)"},
                                 key("title"))
            renamed += status in (200, 201)
    print(json.dumps({"aliases_written": written, "aliases_already_present": skipped, "placeholders_titled": renamed, "failed": failed}, indent=2))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
