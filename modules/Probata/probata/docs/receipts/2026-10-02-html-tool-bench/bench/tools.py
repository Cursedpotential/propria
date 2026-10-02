# Byline: Claude Code · Sonnet · 2026-10-02
"""Tool adapters for the HTML bench. Each adapter takes a file path and returns
either text (generic mode) or a list of record dicts (structured mode).
Imports happen inside the adapter so a missing package fails only that tool."""
from __future__ import annotations

import re


def _read(path):
    with open(path, "rb") as f:
        return f.read().decode("utf-8", "replace")


# ---------------------------------------------------------------- generic text tools
def webbed_text(path):
    import duckdb
    c = duckdb.connect(); c.sql("LOAD webbed")
    c.execute("select html_extract_text(content) from read_text(?)", [path.replace("\\", "/")])
    return c.fetchone()[0] or ""


def webbed_blocks(path):
    import duckdb
    c = duckdb.connect(); c.sql("LOAD webbed")
    rows = c.execute("select content from read_html_blocks(?)", [path.replace("\\", "/")]).fetchall()
    return "\n".join(r[0] or "" for r in rows)


GEN_TEMPLATE = "modules/Probata/probata/modules/engine/postgres/elt_templates/generic_html_document_v1.sql"


def webbed_generic_template(path):
    """The Proffer generic_html_document_v1 template (webbed XPath blocks), text = block texts in order."""
    import duckdb, json
    c = duckdb.connect(); c.sql("LOAD webbed")
    rows = c.execute(open(GEN_TEMPLATE, encoding="utf-8").read().replace("{{SOURCE}}", path.replace("\\", "/"))).fetchall()
    return "\n".join(json.loads(r[1])["doc_text"] for r in rows)


def selectolax_text(path):
    from selectolax.lexbor import LexborHTMLParser
    t = LexborHTMLParser(_read(path))
    t.strip_tags(["script", "style", "head", "noscript"])
    return t.body.text(separator="\n") if t.body else t.text(separator="\n")


def lxml_text(path):
    import lxml.html
    doc = lxml.html.document_fromstring(_read(path).encode("utf-8"), parser=lxml.html.HTMLParser(encoding="utf-8"))
    for bad in doc.xpath("//script|//style|//head|//noscript"):
        bad.drop_tree()
    return "\n".join(doc.itertext())


def bs4_text(path):
    from bs4 import BeautifulSoup
    s = BeautifulSoup(_read(path), "lxml")
    for bad in s(["script", "style", "head", "noscript"]):
        bad.decompose()
    return s.get_text("\n")


def html2text_text(path):
    import html2text
    h = html2text.HTML2Text(); h.body_width = 0; h.ignore_images = False
    return h.handle(_read(path))


def markitdown_text(path):
    from markitdown import MarkItDown
    return MarkItDown().convert(path).text_content


def trafilatura_text(path):
    import trafilatura
    return trafilatura.extract(_read(path), include_comments=True, include_tables=True, include_links=True,
                               include_images=True, favor_recall=True, no_fallback=False) or ""


def readability_text(path):
    from readability import Document
    import lxml.html
    summary = Document(_read(path)).summary(html_partial=True)
    return "\n".join(lxml.html.fromstring(summary).itertext())


def docling_text(path):
    from docling.document_converter import DocumentConverter
    r = DocumentConverter().convert(path)
    return r.document.export_to_markdown()


def unstructured_text(path):
    from unstructured.partition.html import partition_html
    els = partition_html(filename=path)
    return "\n".join(e.text for e in els)


# ---------------------------------------------------------------- FB structured tools -> records
CLS = lambda c: f'contains(concat(" ",normalize-space(@class)," ")," {c} ")'  # noqa: E731


def lxml_fb(path):
    import lxml.html
    doc = lxml.html.document_fromstring(_read(path).encode("utf-8"), parser=lxml.html.HTMLParser(encoding="utf-8"))
    out = []
    for b in doc.xpath(f'//*[{CLS("_a6-g")}]'):
        s = b.xpath(f'.//*[{CLS("_a6-h")}]')
        ts = b.xpath(f'.//*[{CLS("_a6-o")}]')
        reactions = [" ".join(li.text_content().split()) for li in b.xpath(f'.//ul[{CLS("_a6-q")}]/li')]
        attach = b.xpath(".//a/@href") + [x for x in b.xpath(".//*/@src") if not x.startswith("data:")]
        parts = []
        for body in b.xpath(f'.//*[{CLS("_a6-p")}]'):
            for ul in body.xpath(f'.//ul[{CLS("_a6-q")}]'):
                ul.drop_tree()
            parts.append(" ".join("".join(body.itertext()).split()))
        out.append({"sender": " ".join(s[0].text_content().split()) if s else "",
                    "ts": " ".join(ts[0].text_content().split()) if ts else "",
                    "body": " ".join(parts),
                    "reactions": reactions, "attach": attach})
    return out


def selectolax_fb(path):
    from selectolax.lexbor import LexborHTMLParser
    t = LexborHTMLParser(_read(path))
    out = []
    for b in t.css("._a6-g"):
        s = b.css_first("._a6-h"); ts = b.css_first("._a6-o"); body = b.css_first("._a6-p")
        reacts = [" ".join(li.text().split()) for li in b.css("ul._a6-q li")]
        for ul in b.css("ul._a6-q"):
            ul.decompose()
        out.append({"sender": " ".join(s.text().split()) if s else "",
                    "ts": " ".join(ts.text().split()) if ts else "",
                    "body": " ".join(body.text(separator=" ").split()) if body else "",
                    "reactions": reacts,
                    "attach": [a.attributes.get("href") for a in b.css("a[href]")] +
                              [x.attributes.get("src") for x in b.css("[src]") if not (x.attributes.get("src") or "").startswith("data:")]})
    return out


def bs4_fb(path):
    from bs4 import BeautifulSoup
    s = BeautifulSoup(_read(path), "lxml")
    out = []
    for b in s.select("._a6-g"):
        sd = b.select_one("._a6-h"); ts = b.select_one("._a6-o"); body = b.select_one("._a6-p")
        reacts = [" ".join(li.get_text(" ").split()) for li in b.select("ul._a6-q li")]
        for ul in b.select("ul._a6-q"):
            ul.decompose()
        out.append({"sender": " ".join(sd.get_text(" ").split()) if sd else "",
                    "ts": " ".join(ts.get_text(" ").split()) if ts else "",
                    "body": " ".join(body.get_text(" ").split()) if body else "",
                    "reactions": reacts,
                    "attach": [a["href"] for a in b.select("a[href]")] +
                              [x["src"] for x in b.select("[src]") if not x["src"].startswith("data:")]})
    return out


def webbed_fb(path):
    """The DuckDB webbed XPath template (same query text the Go engine runs)."""
    import duckdb
    c = duckdb.connect(); c.sql("LOAD webbed")
    sql = open(__file__.replace("tools.py", "fb_html_webbed.sql"), encoding="utf-8").read()
    rows = c.execute(sql.replace("{{SRC}}", path.replace("\\", "/"))).fetchall()
    return [{"sender": r[0] or "", "ts": r[1] or "", "body": r[2] or "", "reactions": r[3] or [], "attach": r[4] or []} for r in rows]


def regex_fb(path):
    """The existing Case Bible DuckDB regex template elt_fb_messenger_html_v1 (verbatim columns)."""
    import duckdb
    c = duckdb.connect()
    sql = open(__file__.replace("tools.py", "elt_fb_messenger_html_v1.sql"), encoding="utf-8").read()
    rows = c.execute(sql.replace("{{SRC}}", path.replace("\\", "/"))).fetchall()
    names = [d[0] for d in c.description]
    ix = {n: i for i, n in enumerate(names)}
    return [{"sender": r[ix["sender"]] or "", "ts": r[ix["ts_original"]] or "", "body": r[ix["body"]] or "",
             "reactions": [], "attach": (r[ix["attachments"]] or "")} for r in rows]


def pyport_fb(path):
    """The repo's Python parser facebook_messenger_html.py (BeautifulSoup port of dial-stack), called on bytes."""
    import sys, types
    sys.path.insert(0, __file__.replace("tools.py", "pyport_shim"))
    from pyport_shim import parse_html_records
    return parse_html_records(path)


def chm_fb(path):
    """realdeveloperongithub/chat-history-manager MessengerParser selectors (GPL-3.0), reimplemented call: item.select(...)[0]."""
    from bs4 import BeautifulSoup
    soup = BeautifulSoup(open(path, encoding="utf8"), "html.parser")
    out = []
    for item in soup.findAll("div", {"class": ["_3-95 _a6-g"]}):
        try:
            sender = item.select("div._2ph_._a6-h._a6-i")[0].text.strip()
            ts = item.select("div._3-94._a6-o > div")[0].text.strip()
            body = item.select("div._2ph_._a6-p > div > div:nth-child(2)")[0].text.strip()
        except IndexError:
            continue
        out.append({"sender": sender, "ts": ts, "body": body, "reactions": [], "attach": []})
    return out


GENERIC = {"webbed_generic_template": webbed_generic_template, "webbed_html_extract_text": webbed_text, "webbed_read_html_blocks": webbed_blocks,
           "selectolax_lexbor": selectolax_text, "lxml": lxml_text, "beautifulsoup4": bs4_text,
           "html2text": html2text_text, "markitdown": markitdown_text, "trafilatura": trafilatura_text,
           "readability_lxml": readability_text, "docling": docling_text, "unstructured": unstructured_text}
FB = {"webbed_xpath_template": webbed_fb, "lxml_xpath": lxml_fb, "selectolax_css": selectolax_fb,
      "bs4_css": bs4_fb, "casebible_regex_template_v1": regex_fb, "repo_python_port": pyport_fb,
      "chat_history_manager": chm_fb}
