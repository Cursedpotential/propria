"""Exercise TwelveLabs wire contracts, credential isolation, and registry discovery."""

from __future__ import annotations

import http.client
import io
import json
import os
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import MagicMock, Mock, patch

from server.tools._twelvelabs_client import (
    MAX_RESPONSE_BYTES,
    TwelveLabsClient,
    TwelveLabsError,
    _api_key,
)
from server.tools.extractors import twelvelabs as tools
from server.tools.registry import registry


class ToolContracts(unittest.TestCase):
    def setUp(self):
        self.patcher = patch.object(tools, "TwelveLabsClient")
        self.client = self.patcher.start().return_value
        self.addCleanup(self.patcher.stop)
        self.client.request.return_value = {"_id": "asset-1", "status": "processing"}

    def test_registered_operations_have_documented_versioned_contracts(self):
        entries = [entry for entry in registry.contract_manifest() if entry["id"].startswith("twelvelabs.")]
        self.assertEqual(len(entries), 9)
        for entry in entries:
            self.assertTrue(entry["description"])
            self.assertEqual(entry["contract_version"], "1")

    def test_upload_uses_multipart_url_and_keeps_source_identity(self):
        result = tools.asset_submit(
            {"url": "https://media.example.org/a.mp4?signature=secret", "source_ref": "source:123"}
        )
        self.client.request.assert_called_once_with(
            "POST", "/assets", form={"method": "url", "url": "https://media.example.org/a.mp4?signature=secret"}
        )
        self.assertEqual(result["material_class"], "context")
        self.assertEqual(result["source_ref"], "source:123")
        self.assertNotIn("signature", json.dumps(result))

    def test_reject_bad_upload_urls_before_external_request(self):
        for url in (
            "file:///a",
            "http://media.example.org/a",
            "https://localhost/a",
            "https://127.0.0.1/a",
            "https://10.0.0.1/a",
            "https://user:pass@example.org/a",
            "https://example.org/a\r\nx",
        ):
            with self.subTest(url=url), self.assertRaises(ValueError):
                tools.asset_submit({"url": url})
        self.client.request.assert_not_called()

    def test_invalid_context_does_not_trigger_billable_call(self):
        with self.assertRaises(ValueError):
            tools.asset_submit({"url": "https://example.org/a.mp4", "source_ref": {"bad": True}})
        self.client.request.assert_not_called()

    def test_status_is_single_call_and_preserves_failed_status(self):
        self.client.request.return_value = {"status": "failed", "error": "unreadable"}
        result = tools.asset_status({"asset_id": "asset-1"})
        self.assertEqual(result["data"]["status"], "failed")
        self.client.request.assert_called_once_with("GET", "/assets/asset-1")

    def test_identifiers_cannot_escape_provider_path(self):
        with self.assertRaises(ValueError):
            tools.item_status({"knowledge_store_id": "../secret", "item_id": "item-1"})
        self.client.request.assert_not_called()

    def test_image_indexing_is_explicit(self):
        tools.item_add({"knowledge_store_id": "ks-1", "asset_id": "asset-1", "asset_type": "image"})
        self.client.request.assert_called_once_with(
            "POST", "/knowledge-stores/ks-1/items", body={"asset_id": "asset-1", "asset_type": "image"}
        )

    def test_invalid_asset_type_fails_before_network(self):
        with self.assertRaises(ValueError):
            tools.item_add({"knowledge_store_id": "ks-1", "asset_id": "a-1", "asset_type": "invalid"})
        self.client.request.assert_not_called()

    def test_pagination_is_bounded(self):
        for value in (True, 0, 51, "10"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                tools.stores_list({"page_limit": value})
        self.client.request.assert_not_called()

    def test_search_preserves_native_timestamps_and_pagination(self):
        expected = {
            "data": [{"item_id": "item-1", "matches": [{"start_sec": 3, "end_sec": 8}]}],
            "next_page_token": "next",
        }
        self.client.request.return_value = expected
        result = tools.search({"knowledge_store_id": "ks-1", "query": "a person enters", "item_ids": ["item-1"]})
        self.assertEqual(result["data"], expected)
        self.assertEqual(self.client.request.call_args.kwargs["body"]["filter"], {"item_id": {"in": ["item-1"]}})

    def test_response_keeps_annotations_usage_and_incomplete_status(self):
        expected = {
            "status": "incomplete",
            "usage": {"input_tokens": 10},
            "output": [
                {
                    "type": "message",
                    "content": [
                        {
                            "text": "Observation [1]",
                            "annotations": [{"type": "video_citation", "item_id": "i-1", "start_sec": 1, "end_sec": 2}],
                        }
                    ],
                }
            ],
        }
        self.client.request.return_value = expected
        result = tools.respond(
            {
                "knowledge_store_id": "ks-1",
                "prompt": "Describe the scene",
                "session_id": "sess-1",
                "json_schema": {"type": "object", "properties": {}},
            }
        )
        self.assertEqual(result["data"], expected)
        body = self.client.request.call_args.kwargs["body"]
        self.assertFalse(body["stream"])
        self.assertEqual(body["tools"], [])
        self.assertEqual(body["session_id"], "sess-1")
        self.assertIn("schema", body["text"]["format"])

    def test_ambiguous_failure_survives_json_transport(self):
        self.client.request.side_effect = TwelveLabsError("transport_failed", ambiguous=True)
        result = json.loads(json.dumps(tools.store_create({"name": "Corpus"})))
        self.assertFalse(result["ok"])
        self.assertTrue(result["error"]["ambiguous"])
        self.assertFalse(result["error"]["retry_safe"])
        self.assertNotIn("data", result)


class TransportContracts(unittest.TestCase):
    def setUp(self):
        self.env = patch.dict(
            os.environ, {"TWELVELABS_API_KEY": "private-key", "TWELVELABS_TIMEOUT_SECONDS": "60"}, clear=True
        )
        self.env.start()
        self.addCleanup(self.env.stop)

    def test_literal_credentials_file_and_environment_precedence(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "twelvelabs.env"
            path.write_text('export TWELVELABS_API_KEY="literal-$(never-execute)"\n', encoding="utf-8")
            os.environ["TWELVELABS_ENV_FILE"] = str(path)
            self.assertEqual(_api_key(), "private-key")
            del os.environ["TWELVELABS_API_KEY"]
            self.assertEqual(_api_key(), "literal-$(never-execute)")

    def test_missing_credentials_error_contains_no_path_or_secret(self):
        del os.environ["TWELVELABS_API_KEY"]
        os.environ["TWELVELABS_ENV_FILE"] = "missing-private-location"
        with self.assertRaisesRegex(TwelveLabsError, "credentials_file_unavailable") as caught:
            _api_key()
        self.assertNotIn("missing-private-location", str(caught.exception))

    def test_timeout_is_finite_and_bounded(self):
        for value in ("nan", "inf", "0", "301"):
            os.environ["TWELVELABS_TIMEOUT_SECONDS"] = value
            with self.subTest(value=value), self.assertRaises(ValueError):
                TwelveLabsClient()

    def test_wire_multipart_and_fixed_host(self):
        client = TwelveLabsClient()
        response = Mock()
        response.read.return_value = b'{"_id":"a-1"}'
        client._opener = MagicMock()
        client._opener.open.return_value.__enter__.return_value = response
        result = client.request("POST", "/assets", form={"method": "url", "url": "https://example.org/a.mp4"})
        request = client._opener.open.call_args.args[0]
        self.assertEqual(request.full_url, "https://api.twelvelabs.io/v1.3/assets")
        self.assertIn(b'name="method"\r\n\r\nurl', request.data)
        self.assertEqual(result["_id"], "a-1")

    def test_provider_error_does_not_echo_body_or_retry(self):
        client = TwelveLabsClient()
        client._opener = Mock()
        client._opener.open.side_effect = urllib.error.HTTPError(
            "https://api.twelvelabs.io", 500, "secret provider text", {}, io.BytesIO(b"private-key")
        )
        with self.assertRaises(TwelveLabsError) as caught:
            client.request("POST", "/responses", body={})
        self.assertTrue(caught.exception.ambiguous)
        self.assertNotIn("private-key", str(caught.exception))
        self.assertNotIn("secret provider text", str(caught.exception))
        self.assertEqual(client._opener.open.call_count, 1)

    def test_transport_failure_is_ambiguous_for_post(self):
        client = TwelveLabsClient()
        client._opener = Mock()
        client._opener.open.side_effect = urllib.error.URLError("signed-url-secret")
        with self.assertRaises(TwelveLabsError) as caught:
            client.request("POST", "/assets", form={"method": "url"})
        self.assertTrue(caught.exception.ambiguous)
        self.assertNotIn("signed-url-secret", str(caught.exception))

    def test_response_size_limit_does_not_truncate_silently(self):
        client = TwelveLabsClient()
        client._opener = MagicMock()
        client._opener.open.return_value.__enter__.return_value.read.return_value = b"x" * (MAX_RESPONSE_BYTES + 1)
        with self.assertRaisesRegex(TwelveLabsError, "response_too_large"):
            client.request("GET", "/assets/a-1")

    def test_truncated_post_response_is_sanitized_and_ambiguous(self):
        client = TwelveLabsClient()
        client._opener = MagicMock()
        client._opener.open.return_value.__enter__.return_value.read.side_effect = http.client.IncompleteRead(
            b"private-key", 100
        )
        with self.assertRaises(TwelveLabsError) as caught:
            client.request("POST", "/responses", body={})
        self.assertTrue(caught.exception.ambiguous)
        self.assertNotIn("private-key", str(caught.exception))

    def test_redirect_is_rejected(self):
        from server.tools._twelvelabs_client import _NoRedirect

        self.assertIsNone(_NoRedirect().redirect_request(None, None, 302, "", {}, "https://elsewhere.example.org"))


if __name__ == "__main__":
    unittest.main()
