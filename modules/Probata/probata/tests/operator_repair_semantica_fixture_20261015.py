"""Produce a bounded fixture using the real deterministic Semantica extractor.

Inputs: one output path and four embedded synthetic messages. Outputs: exact
message bundle plus actual extractor page, hash and counts. Effects: fixture
file write only; no database, worker, model, source ingestion or evidence action.
"""
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import sys

from server.tools.extractors.entity_events.pages import Message
from server.tools.extractors.entity_events.semantica_extractor import extract_page


def main() -> None:
    """Extract the synthetic bounded page and retain exact bytes for Go grounding."""
    at = datetime(2025, 6, 1, 12, tzinfo=timezone.utc)
    texts = [
        "Morning Katherine Doe, can you get Emma from Lincoln Elementary School at 3?",
        "Meet at Flint City on June 5, 2025. Call Acme Corp Inc about it.",
        "Katherine Doe sued Acme Corp Inc on June 5, 2025.",
        "ok thanks",
    ]
    window = [Message(f"00000000-0000-4000-8000-{i+1:012d}", i, at, body, "self", ("+15555550101",)) for i, body in enumerate(texts)]
    page = extract_page(window, -1)
    assert page["extractor"] == "semantica" and page["messages"] == 4
    assert any("Katherine Doe" in item["name"] for item in page["people"])
    assert page.get("events"), "Real deterministic fixture must produce at least one event"
    output = {
        "contract": "synthetic-semantic-store-proof/1",
        "messages": [{"record_id": m.record_id, "ordinal": m.ordinal, "occurred_at": m.occurred_at.isoformat(), "body": m.body, "sender": m.sender, "recipients": list(m.recipients)} for m in window],
        "page": page,
    }
    raw = json.dumps(output, ensure_ascii=False, sort_keys=True, indent=2).encode("utf-8")
    path = Path(sys.argv[1])
    path.write_bytes(raw)
    print(json.dumps({"fixture": str(path), "sha256": hashlib.sha256(raw).hexdigest(), "messages": len(window), "people": len(page["people"]), "places": len(page["places"]), "organizations": len(page["organizations"]), "events": len(page["events"]), "extractor_version": page["extractor_version"]}))


if __name__ == "__main__":
    main()
