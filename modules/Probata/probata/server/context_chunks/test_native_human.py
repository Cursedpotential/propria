"""Exercise native chunk coverage with synthetic sources; no models, live data or external writes."""
import json
import tempfile
import unittest
from unittest.mock import Mock, patch
from pathlib import Path

from server.context_chunks.native_human import MAX_PREPARED_BYTES, MAX_TEXT_BYTES, plan_native_chunks


class NativeChunkTests(unittest.TestCase):
    """Verify bounded windows/oversize segments retain every native source coordinate."""
    def fixture(self, root, records):
        """Create a synthetic immutable extraction bundle and its reference-only request."""
        source = {"source_ref": "synthetic://sms.xml", "source_version_ref": "synthetic-version",
                  "registration_receipt_ref": "synthetic://registration", "format": "smsbackuprestore_xml",
                  "owner_person_ref": "synthetic-owner", "perspective_person_ref": "synthetic-perspective",
                  "perspective": "source-device", "provider_version_id": "synthetic-provider-version",
                  "byte_length": 100, "etag": "synthetic-etag"}
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

    def test_manifest_source_drift_fails_before_spool_or_planner(self):
        """Reject identity, attribution and known provider drift before derivative work.

        Inputs: synthetic extraction manifests with one altered source pin each.
        Outputs: validation failures and untouched spool/planner assertions. Effects: temporary fixtures only.
        Choose to verify equal source-version IDs cannot hide other source-pin changes.
        """
        changes = {"source_ref": "synthetic://other.xml", "source_version_ref": "other-version",
                   "registration_receipt_ref": "synthetic://other-receipt", "format": "other-format",
                   "owner_person_ref": "other-owner", "perspective_person_ref": "other-perspective",
                   "perspective": "other-view", "provider_version_id": "other-provider-version",
                   "byte_length": 101, "etag": "other-etag"}
        for field, value in changes.items():
            with self.subTest(field=field), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                request = self.fixture(root, [("body", ["peer"])])
                manifest_path = root / "manifest.json"
                manifest = json.loads(manifest_path.read_text())
                manifest["source"][field] = value
                manifest_path.write_text(json.dumps(manifest))
                planner = Mock(side_effect=AssertionError("planner must not run"))
                with patch("server.context_chunks.native_human._spool") as spool:
                    with self.assertRaisesRegex(ValueError, "pin differs"):
                        plan_native_chunks(root, request, span_planner=planner, version="test")
                    spool.assert_not_called()
                planner.assert_not_called()
                self.assertFalse(list(root.glob("chunked-*")))

    def test_unknown_provider_pins_and_http_quoted_etag_are_accepted(self):
        """Accept newly observed provider metadata and HTTP-quoted equivalent ETags.

        Inputs: synthetic registered sources with known or unknown provider pins.
        Outputs: successful plans preserving observed source metadata. Effects: temporary derivatives only.
        Choose to prevent identity validation from treating unknown provider metadata as a new gate.
        """
        for unknown in (False, True):
            with self.subTest(unknown=unknown), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                request = self.fixture(root, [("body", ["peer"])])
                if unknown:
                    request["source"] = {**request["source"], "provider_version_id": "", "byte_length": 0, "etag": ""}
                else:
                    request["source"]["etag"] = '  "synthetic-etag"  '
                result = plan_native_chunks(root, request,
                                            span_planner=lambda lines, *a, **k: [(0, len(lines) - 1)], version="test")
                manifest = json.loads(Path(result["manifest_ref"].removeprefix("file://")).read_text())
                self.assertEqual(manifest["source"]["provider_version_id"], "synthetic-provider-version")
                self.assertEqual(manifest["source"]["byte_length"], 100)
                self.assertEqual(manifest["source"]["etag"], "synthetic-etag")

    def test_dense_image_markers_bound_prepared_text_and_preserve_original_spans(self):
        """Split on NIM expansion while preserving every original body character and locator.

        Inputs: synthetic dense lowercase markers, Unicode and unchanged uppercase markers.
        Outputs: assertions on exact coverage and independently prepared byte counts.
        Effects: temporary derivative bundles only; choose to catch post-preparation overflow.
        """
        bodies = ["data:image/" * 335, "data:image/" * 335,
                  "漢😀data:image/" * 1200, "DATA:IMAGE/" * 335]
        prefix = "unknown sender: "
        original_pair = "\n".join(prefix + body for body in bodies[:2])
        bodies[0] += "x" * (MAX_TEXT_BYTES - len(original_pair.encode("utf-8")))
        original_pair = "\n".join(prefix + body for body in bodies[:2])
        self.assertEqual(len(original_pair.encode("utf-8")), MAX_TEXT_BYTES)
        self.assertGreater(len(original_pair.replace("data:image/", "data: image/").encode("utf-8")), MAX_PREPARED_BYTES)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            request = self.fixture(root, [(body, ["peer"]) for body in bodies])
            result = plan_native_chunks(root, request,
                                        span_planner=lambda lines, *a, **k: [(0, len(lines) - 1)],
                                        version="synthetic-neural")
            manifest = json.loads(Path(result["manifest_ref"].removeprefix("file://")).read_text())
            chunks = [json.loads(line) for line in Path(manifest["chunks_ref"].removeprefix("file://")).read_text().splitlines()]
            for chunk in chunks:
                prepared = chunk["text"].replace("data:image/", "data: image/")
                self.assertLessEqual(len(chunk["text"].encode("utf-8")), MAX_TEXT_BYTES)
                self.assertLessEqual(len(prepared.encode("utf-8")), MAX_PREPARED_BYTES)
                self.assertFalse({1, 2}.issubset({member["ordinal"] for member in chunk["members"]}))
            for ordinal, body in enumerate(bodies, 1):
                members = {(member["body_start"], member["body_end"]): member
                           for chunk in chunks for member in chunk["members"] if member["ordinal"] == ordinal}
                spans = sorted(members)
                self.assertEqual(spans[0][0], 0)
                self.assertEqual(spans[-1][1], len(body))
                self.assertTrue(all(left[1] == right[0] for left, right in zip(spans, spans[1:])))
                self.assertEqual("".join(members[span]["text"][len(prefix):] for span in spans), body)
                for (first, last), member in members.items():
                    self.assertEqual(member["text"], prefix + body[first:last])
                    self.assertEqual(member["locator"], {**request["source"], "record_ordinal": ordinal,
                                                         "start": ordinal * 20, "end": ordinal * 20 + 19})


if __name__ == "__main__":
    unittest.main()
