"""Contacts import: parsing, dedupe and the candidate-name plan. Synthetic exports; no network.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

import importlib.util
from pathlib import Path

import pytest

pytest.importorskip("vobject")

_spec = importlib.util.spec_from_file_location("import_contacts", Path(__file__).resolve().parents[1] / "tools" / "import_contacts.py")
ic = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(ic)

VCF = """BEGIN:VCARD
VERSION:3.0
FN:Jordan Reyes
TEL;TYPE=CELL:(810) 555-0142
TEL:+1 313 555 0199
EMAIL:Jordan@Example.com
END:VCARD
BEGIN:VCARD
VERSION:3.0
N:Ng;Sam;;;
TEL:810.555.0142
END:VCARD
"""


def test_phone_rule_matches_the_registry_rule():
    assert ic.normalize_phone("+1 (810) 555-0142") == "8105550142"
    assert ic.normalize_phone("34428") is None
    assert ic.normalize_phone("someone@example.com") is None
    assert ic.normalize_phone("0105550142") is None


def test_vcard_parses_names_phones_and_emails():
    contacts = ic.parse_vcard(VCF, "b2://x/contacts.vcf")
    assert [c.name for c in contacts] == ["Jordan Reyes", "Sam Ng"]
    assert contacts[0].phones == ["3135550199", "8105550142"]
    assert contacts[0].emails == ["jordan@example.com"]


def test_csv_google_and_outlook_headers():
    google = "First Name,Last Name,Phone 1 - Value,E-mail 1 - Value\nJordan,Reyes,(810) 555-0142,j@example.com\n"
    outlook = "First Name,Last Name,Mobile Phone,E-mail Address\nSam,Ng,810-555-0142,\n"
    a = ic.parse_csv(google, "k1")[0]
    b = ic.parse_csv(outlook, "k2")[0]
    assert (a.name, a.phones, a.emails) == ("Jordan Reyes", ["8105550142"], ["j@example.com"])
    assert (b.name, b.phones) == ("Sam Ng", ["8105550142"])


def test_social_json_any_nesting():
    text = '{"phone_contacts":[{"first_name":"Jordan","last_name":"Reyes","contact_point":"+18105550142"}],"x":{"y":[{"name":"Lee","email":"lee@example.com"}]}}'
    contacts = ic.parse_social_json(text, "fb")
    assert [(c.name, c.phones) for c in contacts] == [("Jordan Reyes", ["8105550142"]), ("Lee", [])]


def test_a_number_named_differently_keeps_every_name_as_a_candidate():
    contacts = ic.parse_vcard(VCF, "k1") + ic.parse_csv("Name,Phone\nJ. Reyes,810-555-0142\n", "k2")
    plan = ic.build_plan(contacts)
    names = plan["8105550142"]["names"]
    assert set(names) == {"Jordan Reyes", "Sam Ng", "J. Reyes"}
    assert names["Jordan Reyes"] == {"k1"} and names["J. Reyes"] == {"k2"}
    assert set(plan["3135550199"]["names"]) == {"Jordan Reyes"}
