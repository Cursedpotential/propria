# Byline: Claude Code · Sonnet · 2026-10-02
"""Verbatim logic of server/tools/parsers/messaging/facebook_messenger_html.py (_structure_legacy, _structure_card),
lifted without the platform imports so it can run in the bench. Output reshaped to bench records."""
import re
from bs4 import BeautifulSoup


def _structure_legacy(soup):
    rows = []
    for msg in soup.select("div.message"):
        meta = msg.find("div", class_="meta")
        if not meta:
            continue
        meta_text = meta.get_text(" ", strip=True)
        m = re.match(r"^(.*?)\s*[-–—]\s*(.+)$", meta_text)
        if not m:
            continue
        body_el = msg.find("p")
        body = body_el.get_text(" ", strip=True) if body_el else ""
        if not body:
            continue
        rows.append((m.group(1).strip(), body, m.group(2), meta_text[:200]))
    return rows


def _structure_card(soup):
    rows = []
    for card in soup.select("div._a6-g"):
        header = card.find("div", class_="_a6-h")
        sender = header.get_text(" ", strip=True) if header else ""
        if not sender:
            continue
        body_el = card.find("div", class_="_a6-p")
        body = body_el.get_text(" ", strip=True) if body_el else ""
        if not body:
            continue
        ts = None
        sib = card.find_next_sibling("div", class_="_a6-o")
        if sib:
            tstag = sib.find("div", class_="_a72d")
            if tstag:
                ts = tstag.get_text(" ", strip=True)
        if not ts:
            continue
        rows.append((sender, body, ts, "card"))
    return rows


def parse_html_records(path):
    soup = BeautifulSoup(open(path, encoding="utf-8", errors="replace").read(), "html.parser")
    rows = _structure_legacy(soup) or _structure_card(soup)
    return [{"sender": s, "ts": ts, "body": b, "reactions": [], "attach": []} for s, b, ts, _ in rows]
