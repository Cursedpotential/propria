"""Exercise native chunk coverage with synthetic sources; no models, live data or external writes."""
import json
import tempfile
import unittest
from pathlib import Path

from server.context_chunks.native_human import MAX_TEXT_BYTES, plan_native_chunks


class NativeChunkTests(unittest.TestCase):
    """Verify bounded windows/oversize segments retain every native source coordinate."""
    def fixture(self, root, records):
        """Create a synthetic immutable extraction bundle and its reference-only request."""
        source = {"source_ref": "synthetic://sms.xml", "source_version_ref": "synthetic-version"}
        path = root / "records.jsonl"
        with path.open("w") as stream:
            for i, (body, participants) in enumerate(records, 1):
                native = {"record_kind": "message", "body": body, "participants": participants,
                          "source_locator": {**source, "record_ordinal": i, "start": i * 20, "end": i * 20 + 19}}
                stream.write(json.dumps({"ordinal": i, "status": "complete", "native": native}) + "\n")
        manifest = {"contract_version": "human-source-v1", "workflow_id": "synthetic", "stage": "extracted", "source": source,
                    "records_ref": path.as_uri(), "records": len(records), "pending": 0, "framing": {"complete": True}}
        ref = root / "manifest.json"
        ref.write_text(json.dumps(manifest))
        return {"contract_version": "human-source-v1", "workflow_id": "synthetic", "source": source, "manifest_ref": ref.as_uri()}

    def test_bounded_windows_and_unicode_oversize_cover_native_spans(self):
        """A long conversation plus one Unicode oversize message retains complete body coverage."""
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            records = [("body " + str(i), ["peer"]) for i in range(700)] + [("😀漢字" * 10000, ["peer"])]
            request = self.fixture(root, records)
            sizes = []
            def planner(lines, *args, **kwargs):
                sizes.append(len(lines))
                return [(0, len(lines) - 1)]
            result = plan_native_chunks(root, request, span_planner=planner, version="synthetic-neural")
            manifest = json.loads(Path(result["manifest_ref"].removeprefix("file://")).read_text())
            chunks = [json.loads(line) for line in Path(manifest["chunks_ref"].removeprefix("file://")).read_text().splitlines()]
            self.assertGreater(len(sizes), 2)
            self.assertLessEqual(max(sizes), 256)
            self.assertTrue(all(len(c["text"].encode()) <= MAX_TEXT_BYTES for c in chunks))
            members = [m for c in chunks for m in c["members"]]
            self.assertEqual({m["ordinal"] for m in members}, set(range(1, 702)))
            ranges = sorted({(m["body_start"], m["body_end"]) for m in members if m["ordinal"] == 701})
            self.assertEqual(ranges[0][0], 0)
            self.assertEqual(ranges[-1][1], len(records[-1][0]))
            self.assertTrue(all(a[1] == b[0] for a, b in zip(ranges, ranges[1:])))
            self.assertTrue(all("normalized" not in json.dumps(c) for c in chunks))
            # Consecutive windows preserve their actual shared native record segments.
            groups = {}
            for c in chunks:
                groups.setdefault(c["window_index"], set()).update(c["message_refs"])
            for left, right in zip(groups.values(), list(groups.values())[1:]):
                self.assertGreaterEqual(len(left & right), 2)

    def test_unknown_participants_are_not_merged_and_bad_plan_fails(self):
        """Unknown conversations stay distinct and an incomplete Neural plan is rejected."""
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            request = self.fixture(root, [("one", []), ("two", [])])
            result = plan_native_chunks(root, request, span_planner=lambda lines, *a, **k: [(0, 0)], version="test")
            manifest = json.loads(Path(result["manifest_ref"].removeprefix("file://")).read_text())
            chunks = [json.loads(line) for line in Path(manifest["chunks_ref"].removeprefix("file://")).read_text().splitlines()]
            self.assertEqual(len({c["thread_id"] for c in chunks}), 2)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            request = self.fixture(root, [("one", ["peer"]), ("two", ["peer"])])
            with self.assertRaisesRegex(ValueError, "omitted"):
                plan_native_chunks(root, request, span_planner=lambda lines, *a, **k: [(0, 0)], version="test")


if __name__ == "__main__":
    unittest.main()
