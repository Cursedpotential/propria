# Byline: Claude Code · Sonnet · 2026-10-02
"""Independent oracle for the HTML tool bench.

Only the standard library (html.parser + html.unescape) is used, so no tool under
test grades its own output. It yields (a) the visible text of a document and
(b) for a Facebook "Download Your Information" Messenger thread file, the list of
message blocks: sender, timestamp string, body text, reactions, attachment refs.
"""
from __future__ import annotations

import re
from html.parser import HTMLParser

VOID = {"br", "img", "meta", "link", "input", "hr", "base", "area", "col", "source", "track", "wbr"}
SKIP = {"script", "style", "head", "title", "noscript"}
WS = re.compile(r"\s+")

# One emoji "unit": base + optional VS16 / skin tone + ZWJ chains, flags, keycaps.
EMOJI_RE = re.compile(
    r"(?:[\U0001F1E6-\U0001F1FF]{2}|[0-9#*]️?⃣|"
    r"(?:[\U0001F300-\U0001FAFF☀-➿⬀-⯿⌀-⏿©®‼⁉™ℹ←-⇿■-◿]"
    r"[️\U0001F3FB-\U0001F3FF]?"
    r"(?:‍[\U0001F300-\U0001FAFF☀-➿♀♂⚕⚖✈❤][️\U0001F3FB-\U0001F3FF]?)*))"
)


def norm(text: str) -> str:
    return WS.sub(" ", text or "").strip()


def emoji_units(text: str) -> list[str]:
    return EMOJI_RE.findall(text or "")


class _Visible(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.out: list[str] = []
        self.skip = 0

    def handle_starttag(self, tag, attrs):
        if tag in SKIP:
            self.skip += 1
        if tag in {"br", "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "section", "footer", "header"}:
            self.out.append("\n")

    def handle_endtag(self, tag):
        if tag in SKIP and self.skip:
            self.skip -= 1

    def handle_data(self, data):
        if not self.skip:
            self.out.append(data)


RAWHTML = re.compile(r"rawHtml\s*=\s*`(.*?)`\s*;", re.S)


def visible_text(html: str) -> str:
    p = _Visible()
    p.feed(html)
    text = "".join(p.out)
    # iMessage HTML exports carry the whole conversation as an HTML string inside a JS template
    # literal. That text IS the document content, so the oracle counts it.
    m = RAWHTML.search(html)
    if m:
        q = _Visible()
        q.feed(m.group(1))
        text += "\n" + "".join(q.out)
    return text


class _FB(HTMLParser):
    """Block = element whose class list contains _a6-g. Works for div and section variants."""

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.stack: list[tuple[str, set[str]]] = []
        self.blocks: list[dict] = []
        self.cur: dict | None = None
        self.block_depth = -1
        self.field: str | None = None
        self.field_depth = -1
        self.buf: list[str] = []
        self.in_react = 0
        self.react_depth = -1
        self.skipdepth = 0

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        cls = set((a.get("class") or "").split())
        if tag not in VOID:
            self.stack.append((tag, cls))
        depth = len(self.stack)
        if tag in {"script", "style"}:
            self.skipdepth += 1
        if self.cur is None and "_a6-g" in cls:
            self.cur = {"sender": "", "ts": "", "body_parts": [], "reactions": [], "attach": []}
            self.block_depth = depth
            return
        if self.cur is None:
            return
        if "_a6-h" in cls and self.field is None:
            self.field, self.field_depth, self.buf = "sender", depth, []
        elif "_a6-o" in cls and self.field is None:
            self.field, self.field_depth, self.buf = "ts", depth, []
        elif "_a6-p" in cls and self.field is None:
            self.field, self.field_depth, self.buf = "body", depth, []
        if tag == "ul" and "_a6-q" in cls:
            self.in_react, self.react_depth = 1, depth
            self.react_buf = []
        if tag == "li" and self.in_react:
            self.rbuf: list[str] = []
        if tag == "a" and a.get("href"):
            self.cur["attach"].append(a["href"])
        if tag in {"img", "video", "audio", "source"} and (a.get("src")):
            if not a["src"].startswith("data:"):
                self.cur["attach"].append(a["src"])
        if self.field == "body" and tag in {"br", "div", "p", "li"} and not self.in_react:
            self.buf.append("\n")

    def handle_endtag(self, tag):
        if tag in {"script", "style"} and self.skipdepth:
            self.skipdepth -= 1
        if tag in VOID or not self.stack:
            return
        depth = len(self.stack)
        if self.cur is not None:
            if self.in_react and tag == "li" and hasattr(self, "rbuf"):
                self.cur["reactions"].append(norm("".join(self.rbuf)))
                del self.rbuf
            if self.in_react and depth == self.react_depth:
                self.in_react = 0
            if self.field and depth == self.field_depth:
                text = "".join(self.buf)
                if self.field == "body":
                    self.cur["body_parts"].append(text)
                else:
                    self.cur[self.field] = norm(text)
                self.field = None
            if depth == self.block_depth:
                self.blocks.append(self.cur)
                self.cur = None
        self.stack.pop()

    def handle_data(self, data):
        if self.skipdepth:
            return
        if self.cur is not None:
            if hasattr(self, "rbuf") and self.in_react:
                self.rbuf.append(data)
                return
            if self.field and not self.in_react:
                self.buf.append(data)


def fb_blocks(html: str) -> list[dict]:
    p = _FB()
    p.feed(html)
    out = []
    for b in p.blocks:
        body = norm(" ".join(b["body_parts"]))
        out.append({"sender": b["sender"], "ts": b["ts"], "body": body, "reactions": b["reactions"], "attach": b["attach"]})
    return out
