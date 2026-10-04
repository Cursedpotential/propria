"""Test path setup. Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The conversation chunker is Probata's server/context_chunks, imported and never copied. In the
monorepo it sits at
modules/Probata/probata; put that on sys.path when it exists so the tests exercise the real shared
module.
"""

import sys
from pathlib import Path

_PROBATA = Path(__file__).resolve().parents[4] / "Probata" / "probata"
if (_PROBATA / "server" / "context_chunks").is_dir() and str(_PROBATA) not in sys.path:
    sys.path.insert(0, str(_PROBATA))
