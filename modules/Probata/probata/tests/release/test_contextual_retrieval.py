"""Focused offline contracts for optional contextual descriptions and full-source routing."""
from __future__ import annotations

import asyncio
import json
import sys
import unittest
from unittest.mock import AsyncMock, patch
from dataclasses import replace
from pathlib import Path, PurePosixPath
from types import SimpleNamespace

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "scripts/docstore"))
from context_sources import ContextualSourceItems, PolicyFile, contextual_items
from contextual_retrieval import ContextPolicy, build_request, cache_identity, digest, provider_input, search_metadata, selected_policy
from nim_input import EMBED_MAX_CHARS, embed_input


class FakeFile:
    """Supply a tiny unchanged FileLike-shaped resource for routing checks."""

    def __init__(self, name):
        self.file_path = SimpleNamespace(path=PurePosixPath(name))
        self.body = b"original\r\nsource"

    async def read(self, size=-1):
        return self.body if size < 0 else self.body[:size]

    def __coco_memo_key__(self):
        return self.file_path.path.as_posix()


class FakeItems:
    """Supply full scan plus live lifecycle callbacks without a target store."""

    def __init__(self):
        self.files = [(name, FakeFile(name)) for name in ("a.md", "b.md", "c.md")]

    async def __aiter__(self):
        for item in self.files:
            yield item

    async def watch(self, subscriber):
        await subscriber.update_all()
        await subscriber.update(*self.files[0])
        await subscriber.delete("retired.md")
        await subscriber.mark_ready()


class ContextContracts(unittest.TestCase):
    """Check source preservation, input identity and bounded prefix behavior."""

    def setUp(self):
        self.policy = ContextPolicy("configured/model", "https://example.invalid/v1")
        self.body = "# Original\r\nDocument context.\r\nThe chunk stays unchanged."
        self.chunk = "The chunk stays unchanged."
        self.request = build_request(source_path="docs/a.md", source_hash=digest(self.body),
                                     title="Original", heading="Original", ordinal=1,
                                     offset=self.body.index(self.chunk), chunk_text=self.chunk,
                                     normalized_document=self.body, policy=self.policy)

    def test_original_and_hash_unchanged(self):
        description, search, raw = search_metadata(self.request, self.policy, "Document section.")
        provenance = json.loads(raw)
        self.assertEqual(self.request.chunk_text, self.chunk)
        self.assertEqual(provenance["source_hash"], digest(self.body))
        self.assertEqual(provenance["chunk_hash"], digest(self.chunk))
        self.assertEqual(search, description + "\n\n" + self.chunk)
        self.assertEqual(provenance["embedding_input_hash"], digest(embed_input(search)))

    def test_identity_deterministic_and_all_context_inputs_change_it(self):
        base = cache_identity(self.request, self.policy)
        self.assertEqual(base, cache_identity(self.request, self.policy))
        for field, value in {"source_hash": "a" * 64, "document_context": "changed context",
                             "chunk_text": "changed chunk", "heading": "changed", "ordinal": 2,
                             "source_path": "docs/b.md", "offset": 0, "rendering": "changed"}.items():
            self.assertNotEqual(base, cache_identity(replace(self.request, **{field: value}), self.policy), field)
        for field, value in {"model": "other/model", "api_base": "https://other.invalid/v1",
                             "embedding_model": "other/embed", "disable_thinking": True}.items():
            self.assertNotEqual(base, cache_identity(self.request, replace(self.policy, **{field: value})), field)

    def test_exact_ceiling_drops_prefix_without_truncating_chunk(self):
        request = replace(self.request, chunk_text="x" * EMBED_MAX_CHARS)
        description, search, _ = search_metadata(request, self.policy, "Context")
        self.assertEqual(description, "")
        self.assertEqual(search, request.chunk_text)

    def test_prefix_budget_shortens_only_description(self):
        request = replace(self.request, chunk_text="x" * (EMBED_MAX_CHARS - 8))
        description, search, _ = search_metadata(request, self.policy, "long context")
        self.assertEqual(description, "long c")
        self.assertEqual(len(embed_input(search, allow_truncation=False)), EMBED_MAX_CHARS)
        self.assertTrue(search.endswith(request.chunk_text))

    def test_uri_sanitation_expansion_still_fits(self):
        request = replace(self.request, chunk_text="data:text/plain," + "x" * (EMBED_MAX_CHARS - 24))
        description, search, _ = search_metadata(request, self.policy, "data:text/plain context")
        self.assertLessEqual(len(embed_input(search, allow_truncation=False)), EMBED_MAX_CHARS)
        self.assertTrue(search.endswith(request.chunk_text))
        self.assertTrue(description)

    def test_oversized_original_is_rejected(self):
        with self.assertRaises(ValueError):
            search_metadata(replace(self.request, chunk_text="x" * (EMBED_MAX_CHARS + 1)), self.policy, "Context")

    def test_default_embedding_behavior_retained(self):
        self.assertEqual(embed_input("original"), "original")
        self.assertEqual(len(embed_input("x" * (EMBED_MAX_CHARS + 1))), EMBED_MAX_CHARS)
        with self.assertRaises(ValueError):
            embed_input("x" * (EMBED_MAX_CHARS + 1), allow_truncation=False)

    def test_bad_locator_rejected(self):
        with self.assertRaises(ValueError):
            build_request(source_path="docs/a.md", source_hash=digest(self.body), title="", heading="",
                          ordinal=0, offset=0, chunk_text="absent", normalized_document=self.body, policy=self.policy)

    def test_large_document_context_is_bounded_with_coverage(self):
        body = "x" * 100_000 + self.chunk
        request = build_request(source_path="docs/a.md", source_hash=digest(body), title="", heading="",
                                ordinal=0, offset=100_000, chunk_text=self.chunk,
                                normalized_document=body, policy=self.policy)
        self.assertLessEqual(sum(b - a for a, b in request.context_spans), self.policy.max_document_chars)
        self.assertIn(self.chunk, request.document_context)
        self.assertEqual(request.source_chars, len(body))

    def test_provider_output_hash_keeps_pre_normalized_output(self):
        _, _, provenance = search_metadata(self.request, self.policy, ":folded:", provider_output="original output")
        self.assertEqual(json.loads(provenance)["provider_output_hash"], digest("original output"))

    def test_provider_chunk_excerpt_is_bounded_without_changing_embedding_source(self):
        request = replace(self.request, chunk_text="x" * 65_536)
        payload = json.loads(provider_input(request, self.policy))
        self.assertEqual(len(payload["chunk_text"]), 8_000)
        self.assertEqual(payload["chunk_chars"], 65_536)
        self.assertEqual(payload["chunk_context_span"], [0, 8_000])
        self.assertEqual(len(request.chunk_text), 65_536)
        self.assertEqual(search_metadata(request, self.policy, "Context")[1], request.chunk_text)

    def test_provider_refuses_unbounded_direct_request(self):
        with self.assertRaises(ValueError):
            provider_input(replace(self.request, document_context="x" * 30_000), self.policy)


class SourceContracts(unittest.TestCase):
    """Check full membership, default identity and selected policy invalidation."""

    def setUp(self):
        self.env = {"DOCSTORE_CONTEXT_ENABLED": "1", "DOCSTORE_CONTEXT_SOURCES": '["docs/a.md"]',
                    "DOCSTORE_LLM_MODEL": "configured/model", "DOCSTORE_LLM_BASE_URL": "https://example.invalid/v1"}

    def test_default_source_object_is_identical(self):
        items = FakeItems()
        self.assertIs(contextual_items(items, "docs/", {}), items)
        self.assertIsNone(selected_policy("docs/a.md", {}))

    def test_source_adapter_retains_no_provider_credentials(self):
        source = ContextualSourceItems(FakeItems(), "docs/", dict(self.env, DOCSTORE_LLM_API_KEY="fake-key"))
        self.assertNotIn("DOCSTORE_LLM_API_KEY", source.env)

    def test_full_membership_and_unchanged_sibling_objects(self):
        items = FakeItems()
        async def scan():
            return [entry async for entry in ContextualSourceItems(items, "docs/", self.env)]
        result = asyncio.run(scan())
        self.assertEqual([key for key, _ in result], [key for key, _ in items.files])
        self.assertIsInstance(result[0][1], PolicyFile)
        self.assertIs(result[1][1], items.files[1][1])
        self.assertIs(result[2][1], items.files[2][1])
        self.assertEqual(asyncio.run(result[0][1].read()), items.files[0][1].body)

    def test_live_membership_lifecycle_forwarded(self):
        events = []
        class Subscriber:
            async def update_all(self): events.append("all")
            async def update(self, key, file): events.append((key, isinstance(file, PolicyFile)))
            async def delete(self, key): events.append(("delete", key))
            async def mark_ready(self): events.append("ready")
        asyncio.run(ContextualSourceItems(FakeItems(), "docs/", self.env).watch(Subscriber()))
        self.assertEqual(events, ["all", ("a.md", True), ("delete", "retired.md"), "ready"])

    def test_policy_input_changes_only_selected_file_sdk_fingerprint(self):
        try:
            from cocoindex._internal.memo_fingerprint import memo_fingerprint
        except ImportError:
            self.skipTest("CocoIndex SDK is not available in this interpreter")
        items = FakeItems()
        source = ContextualSourceItems(items, "docs/", self.env)
        before = [memo_fingerprint(file).as_bytes() for _, file in items.files]
        after = [memo_fingerprint(source.transform(file)).as_bytes() for _, file in items.files]
        self.assertNotEqual(before[0], after[0])
        self.assertEqual(before[1:], after[1:])

    def test_canonical_other_root_selection(self):
        env = dict(self.env, DOCSTORE_CONTEXT_SOURCES='["advocatio/docs/a.md"]')
        self.assertIsNotNone(selected_policy("advocatio/docs/a.md", env))
        self.assertIsNone(selected_policy("docs/a.md", env))


class UpgradeContracts(unittest.TestCase):
    """Check contextual schema delivery through the existing checksum-bound plan."""

    def test_context_migration_is_pending_after_existing_release(self):
        import upgrade
        checksum = digest((upgrade.SCHEMA / "100_release_080.surql").read_bytes().decode())
        class Database:
            async def query(self, query):
                if query == "INFO FOR DB;":
                    return [{"tables": {name: "present" for name in upgrade.REQUIRED}}]
                if "docstore_migration" in query:
                    return [{"name": "100_release_080.surql", "checksum": checksum, "state": "applied"}]
                return [{"version": upgrade.VERSION}]
        plan = asyncio.run(upgrade.plan(Database()))
        self.assertEqual([item["name"] for item in plan["pending"]], ["101_contextual_chunks.surql"])
        self.assertFalse(plan["automatic_indexing"])
        self.assertFalse(plan["destructive"])
        self.assertEqual(plan["plan_id"], asyncio.run(upgrade.plan(Database()))["plan_id"])

    def test_applied_release_checksum_drift_still_refused(self):
        import upgrade
        class Database:
            async def query(self, query):
                if query == "INFO FOR DB;":
                    return [{"tables": {name: "present" for name in upgrade.REQUIRED}}]
                if "docstore_migration" in query:
                    return [{"name": "100_release_080.surql", "checksum": "wrong", "state": "applied"}]
                return [{"version": upgrade.VERSION}]
        with self.assertRaisesRegex(ValueError, "checksum drift"):
            asyncio.run(upgrade.plan(Database()))

    def test_verify_requires_ready_index_even_with_applied_ledger(self):
        import upgrade
        state = dict(pending=[], missing_tables=[], current_version=upgrade.VERSION)
        for status in ("cleaning", "indexing", "error", None, "ready"):
            with self.subTest(status=status):
                database = SimpleNamespace(query=AsyncMock(return_value=[{"building": {"status": status}}]),
                                           close=AsyncMock())
                with patch.object(upgrade.sq, "connect", AsyncMock(return_value=database)), \
                     patch.object(upgrade, "plan", AsyncMock(return_value=dict(state))):
                    result = asyncio.run(upgrade.operation("verify"))
                self.assertEqual(result["verified"], status == "ready")
                database.close.assert_awaited_once()

    def test_pending_migration_does_not_query_absent_index(self):
        import upgrade
        state = dict(pending=[{"name": "101_contextual_chunks.surql"}], missing_tables=[],
                     current_version=upgrade.VERSION)
        database = SimpleNamespace(query=AsyncMock(), close=AsyncMock())
        with patch.object(upgrade.sq, "connect", AsyncMock(return_value=database)), \
             patch.object(upgrade, "plan", AsyncMock(return_value=state)):
            result = asyncio.run(upgrade.operation("verify"))
        self.assertFalse(result["verified"])
        self.assertEqual(result["contextual_index"]["status"], "migration-pending")
        database.query.assert_not_awaited()

    def test_indexing_admission_refuses_unready_index(self):
        import upgrade
        with patch.object(upgrade, "operation", AsyncMock(return_value={"verified": False})):
            with self.assertRaisesRegex(RuntimeError, "ready contextual index"):
                asyncio.run(upgrade.verify_required())


if __name__ == "__main__":
    unittest.main()
