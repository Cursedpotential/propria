"""Make text safe to send to the NVIDIA NIM embeddings API.

NIM (`nvidia/nemotron-3-embed-1b`) rejects the WHOLE request with HTTP 400 when any input:
- contains the lowercase text "data:image/" -- it reads an inline image and answers "image inputs
  require VLM serving to be enabled on this server";
- starts with "data:" -- it parses the input as a data URI whatever follows;
- is blank -- "must not be blank or empty";
- is longer than 65,536 characters.

One such input fails its whole batch, and with it the index run: run 5356f93f (2026-09-26) died on
ADR prose that quoted "data:image/" while describing this very bug, and a 1 MiB file with nothing
to split on took down an earlier run. Every Docstore embedding call goes through `embed_input()`:
document chunks (flow_docs.py), and memory claims and recall queries (recall.py). The rewrite
applies to the embedding input only; stored text is never changed.
The memsearch fork carries the same guard (src/memsearch/nim.py).

Byline: Claude Code · Opus 5.5 · 2026-09-27
"""
from __future__ import annotations

import re
import sys

BLANK_PLACEHOLDER = "(empty)"
EMBED_MAX_CHARS = 65_536

_DATA_PREFIX_RE = re.compile(r"^\s*data:", re.I)
_DATA_IMAGE_RE = re.compile(r"data:(?=image/)", re.I)


def embed_input(text: str) -> str:
    """The text NIM will accept for `text`; the caller stores the original."""
    if not text.strip():
        return BLANK_PLACEHOLDER
    if _DATA_PREFIX_RE.match(text):
        text = _DATA_PREFIX_RE.sub("text: data:", text, count=1)
    text = _DATA_IMAGE_RE.sub("data: ", text)
    if len(text) > EMBED_MAX_CHARS:
        # A chunk this size means the splitter had nothing to split on: a source problem, so say so.
        print(
            f"docstore: embedding input capped at {EMBED_MAX_CHARS} of {len(text)} characters; "
            f"the text is stored in full. Check whether the source belongs in the index at all.",
            file=sys.stderr, flush=True,
        )
        text = text[:EMBED_MAX_CHARS]
    return text
