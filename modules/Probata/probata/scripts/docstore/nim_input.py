"""Make text safe to send to the NVIDIA NIM embeddings API.

NIM (`nvidia/nemotron-3-embed-1b`) rejects the WHOLE request with HTTP 400 when any input:
- introduces a data: URI anywhere (`data:image/`, base64 or not, any case) -- it reads an inline
  image and answers "image inputs require VLM serving to be enabled on this server";
- is blank -- "must not be blank or empty";
- is longer than 65,536 characters.

One such input fails its whole batch, and with it the index run: run 5356f93f (2026-09-26) died on
ADR prose that quoted "data:image/" while describing this very bug, the 2026-09-28 sync died after
269 s on two documents that mention it mid-chunk, and a 1 MiB file with nothing to split on took
down an earlier run. Every Docstore embedding call goes through `embed_input()`: document chunks
(flow_docs.py), and memory claims and recall queries (recall.py). One space after the colon is
enough for NIM to stop reading a URI. The rewrite applies to the embedding input only; stored text
is never changed. The memsearch fork carries the same guard (src/memsearch/nim.py).

Byline: Claude Code · Opus 5.5 · 2026-09-27; merged with the 2026-09-28 flow_docs guard (any MIME
type, anywhere in the text) by Claude Code · Fable 5.1 · 2026-09-30.
"""
from __future__ import annotations

import re
import sys

BLANK_PLACEHOLDER = "[empty chunk]"
EMBED_MAX_CHARS = 65_536

# Any data: URI introducer, anywhere, base64 or not.
_DATA_URI_INTRO_RE = re.compile(r"data:(?=[a-zA-Z0-9.+-]+/)", re.I)


def embed_input(text: str) -> str:
    """The text NIM will accept for `text`; the caller stores the original."""
    text = _DATA_URI_INTRO_RE.sub("data: ", text)
    if not text.strip():
        return BLANK_PLACEHOLDER
    if len(text) > EMBED_MAX_CHARS:
        # A chunk this size means the splitter had nothing to split on: a source problem, so say so.
        print(
            f"docstore: embedding input capped at {EMBED_MAX_CHARS} of {len(text)} characters; "
            f"the text is stored in full. Check whether the source belongs in the index at all.",
            file=sys.stderr, flush=True,
        )
        text = text[:EMBED_MAX_CHARS]
    return text
