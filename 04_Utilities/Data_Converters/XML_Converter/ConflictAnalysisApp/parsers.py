from __future__ import annotations
from pathlib import Path
from typing import List, Dict, Any, Optional
import re
from datetime import datetime
from bs4 import BeautifulSoup
from PyPDF2 import PdfReader
import xml.etree.ElementTree as ET
from dateutil import parser as dateparser

def parse_date_loose(ts: str) -> Optional[datetime]:
    if not ts:
        return None
    s = ts.strip()
    s = re.sub(r"(\d)(am|pm|AM|PM)\b", r"\1 \2", s)
    try:
        return dateparser.parse(s, fuzzy=True)
    except Exception:
        return None

def parse_sms_xml(path: Path, source_name: str) -> List[Dict[str, Any]]:
    recs = []
    tree = ET.parse(str(path))
    root = tree.getroot()
    for sms in root.findall("sms"):
        ts = sms.attrib.get("date")
        try:
            dt = datetime.fromtimestamp(int(ts)/1000) if ts else None
        except Exception:
            dt = None
        sender = sms.attrib.get("contact_name") or sms.attrib.get("address") or ""
        body = sms.attrib.get("body") or ""
        recs.append({"datetime": dt, "sender": sender, "message": body.strip(), "source": source_name})
    return recs

def parse_html_messages(path: Path, source_name: str) -> List[Dict[str, Any]]:
    html = path.read_text(encoding="utf-8", errors="ignore")
    soup = BeautifulSoup(html, "html.parser")
    recs: List[Dict[str, Any]] = []

    # structure 1
    for msg in soup.find_all("div", class_="message"):
        meta = msg.find("div", class_="meta")
        text_tag = msg.find_next_sibling("p")
        if not meta or not text_tag:
            continue
        meta_text = meta.get_text(" ", strip=True)
        m = re.search(r"^(.*?)\s*-\s*([A-Za-z]+ \d{1,2}, \d{4}\s+\d{1,2}:\d{2}\s(?:AM|PM))", meta_text)
        if not m:
            continue
        sender = (m.group(1) or "").strip()
        dt = parse_date_loose(m.group(2))
        if not dt:
            continue
        text = text_tag.get_text(" ", strip=True)
        if text:
            recs.append({"datetime": dt, "sender": sender, "message": text, "source": source_name})

    # structure 2 (fb cards)
    if not recs:
        for card in soup.find_all("div", class_="_a6-g"):
            header = card.find("div", class_="_a6-h")
            name = header.get_text(strip=True) if header else ""
            body = card.find("div", class_="_a6-p")
            if not name or not body:
                continue
            text_bits = [div.get_text(" ", strip=True) for div in body.find_all("div")]
            text = " ".join([t for t in text_bits if t]).strip() or body.get_text(" ", strip=True)
            nxt = card.find_next_sibling("div", class_="_3-94 _a6-o")
            ts_tag = nxt.find("div", class_="_a72d") if nxt else None
            dt = parse_date_loose(ts_tag.get_text(strip=True) if ts_tag else "")
            if text:
                recs.append({"datetime": dt, "sender": name, "message": text, "source": source_name})
    return recs

def parse_pdf_messages(path: Path, source_name: str, page_limit: Optional[int] = 80) -> List[Dict[str, Any]]:
    recs: List[Dict[str, Any]] = []
    reader = PdfReader(str(path))
    pages = reader.pages
    if page_limit is not None:
        pages = pages[:max(1, min(page_limit, len(reader.pages)))]
    lines: List[str] = []
    for pg in pages:
        txt = pg.extract_text() or ""
        lines.extend(txt.splitlines())
    ts_regex = re.compile(r"([A-Z][a-z]+ \d{1,2}, \d{4}\s+\d{1,2}:\d{2}\s(?:AM|PM))")
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        if ts_regex.fullmatch(line):
            dt = parse_date_loose(line)
            j = i + 1
            sender = ""
            while j < len(lines) and not sender:
                cand = lines[j].strip()
                if cand:
                    sender = cand
                j += 1
            msg_lines = []
            while j < len(lines):
                if ts_regex.fullmatch(lines[j].strip()):
                    break
                msg_lines.append(lines[j])
                j += 1
            text = " ".join([l.strip() for l in msg_lines]).strip()
            if text:
                recs.append({"datetime": dt, "sender": sender, "message": text, "source": source_name})
            i = j
        else:
            i += 1
    return recs


def stream_sms_xml(path: Path, source_name: str):
    """Yield SMS rows using iterparse to avoid loading full XML into memory."""
    import xml.etree.ElementTree as ET
    context = ET.iterparse(str(path), events=("end",))
    for event, elem in context:
        if elem.tag != "sms":
            continue
        ts = elem.attrib.get("date")
        try:
            dt = datetime.fromtimestamp(int(ts)/1000) if ts else None
        except Exception:
            dt = None
        sender = elem.attrib.get("contact_name") or elem.attrib.get("address") or ""
        body = elem.attrib.get("body") or ""
        yield {"datetime": dt, "sender": sender, "message": (body or "").strip(), "source": source_name}
        elem.clear()
