"""Exercise horizon isolation, bounded state and exact source citations with small inert fixtures.

Inputs: synthetic in-memory account strings. Outputs: assertions. Effects: none;
choose for focused walk contracts, never production data or a provider smoke test.
"""
import asyncio
import copy
import importlib.util
import io
import unittest

from server.analysis.approved_walk import Artifacts, MAX_PROMPT_BYTES, canonical, reason, validate_findings, validate_step


def claim(identifier="early", known="2020-01-01T00:00:00Z"):
    return {"id": identifier, "kind": "event_account", "text": "An attributed account, not established truth.",
            "source_version_id": "version", "source_object_id": "object", "source_object_uri": "b2://fixture/source",
            "source_sha256": "a" * 64, "record_id": identifier, "record_sha256": "b" * 64,
            "candidate_sha256": "c" * 64, "source_available_from": known}


def step(claims=None, perspective="as_lived", horizon="2020-01-02T00:00:00Z", index=0):
    return {"schema": "approved-walk-step-v1", "perspective": perspective, "horizon": horizon,
            "scope_digest": "d" * 64, "step_index": index, "claims": [claim()] if claims is None else claims}


def finding(identifier="early"):
    return {"kind": "uncertainty", "text": "The account alone does not establish event truth.",
            "citations": [{"claim_id": identifier, "quote": "An attributed account"}]}


class Contracts(unittest.TestCase):
    def test_later_and_unknown_dates_rejected_before_reasoning(self):
        for known in (None, "2020-01-03T00:00:00Z"):
            with self.assertRaises(ValueError):
                validate_step(step([claim(known=known)]))
        validate_step(step([claim(known=None)], perspective="hindsight", horizon=None))

    def test_future_and_hindsight_memory_rejected(self):
        for perspective, horizon in (("hindsight", None), ("as_lived", "2020-02-01T00:00:00Z")):
            value = step()
            value["previous"] = {"perspective": perspective, "horizon": horizon, "scope_digest": value["scope_digest"],
                                 "findings": [finding()]}
            with self.assertRaises(ValueError):
                validate_step(value)

    def test_bad_model_ids_and_quotes_fail_closed(self):
        indexed = validate_step(step())
        for bad in (finding("made-up"), {**finding(), "citations": [{"claim_id": "early", "quote": "not in source"}]}):
            with self.assertRaises(ValueError):
                validate_findings([bad], indexed)

    def test_complete_native_locator_survives_without_retained_object(self):
        native = claim()
        native.pop("source_object_id")
        native.update(native_json_pointer="/mapping/native-node/message/content/parts/0", native_span_start=0,
                      native_span_end=48, native_span_unit="unicode_codepoint", native_span_sha256="e" * 64)
        checked = validate_findings([finding()], validate_step(step([native])))
        self.assertEqual(checked[0]["citations"][0]["source_pin"], native)
        root = copy.deepcopy(native)
        root.pop("native_json_pointer")  # Go omitempty encodes the valid root JSON Pointer this way.
        validate_step(step([root]))
        broken = copy.deepcopy(native)
        broken.pop("native_span_sha256")
        with self.assertRaises(ValueError):
            validate_step(step([broken]))

    def test_budgets_and_prior_unknown_citation_rejected(self):
        with self.assertRaises(ValueError):
            validate_step(step([claim(str(i)) for i in range(101)]))
        large = claim()
        large["text"] = "x" * MAX_PROMPT_BYTES
        with self.assertRaises(ValueError):
            validate_step(step([large]))
        value = step()
        value["previous"] = {"perspective": "as_lived", "horizon": "2020-01-01T00:00:00Z",
                             "scope_digest": value["scope_digest"], "findings": [finding("unknown-old-id")]}
        with self.assertRaises(ValueError):
            validate_step(value)

    def test_canonical_identity_is_stable_under_key_order(self):
        value = step()
        self.assertEqual(canonical(value), canonical(dict(reversed(list(value.items())))))

    def test_retry_derivatives_are_hash_stable_and_exact_version_reads_fail_closed(self):
        class Client:
            values = {}
            version = 0

            def put_object(self, **kwargs):
                self.version += 1
                version = str(self.version)
                self.values[(kwargs["Key"], version)] = kwargs["Body"]
                return {"VersionId": version}

            def get_object(self, **kwargs):
                version = kwargs["VersionId"]
                return {"VersionId": version, "Body": io.BytesIO(self.values[(kwargs["Key"], version)])}

        artifacts = Artifacts.__new__(Artifacts)
        artifacts.client = Client()
        descriptor = {"uri": "b2://salem-data/" + artifacts.prefix + "test/input.json?versionId=fixture"}
        value = {"input_digest": "fixture", "findings": []}
        first = artifacts.put(descriptor, value)
        second = artifacts.put(descriptor, value)
        self.assertEqual(first["sha256"], second["sha256"])
        self.assertNotEqual(first["version_id"], second["version_id"])
        self.assertEqual(artifacts.read(first), value)
        bad = dict(first, uri=first["uri"].split("?")[0])
        with self.assertRaises(ValueError):
            artifacts.read(bad)
        artifacts.client.values[(first["uri"].split("salem-data/", 1)[1].split("?")[0], first["version_id"])] = b"changed"
        with self.assertRaises(ValueError):
            artifacts.read(first)


@unittest.skipUnless(importlib.util.find_spec("langgraph"), "LangGraph unavailable in this test interpreter")
class GraphIsolation(unittest.IsolatedAsyncioTestCase):
    async def test_every_model_payload_has_only_eligible_current_and_prior_material(self):
        payloads = []

        class Model:
            identity = {"provider": "fixture", "model_id": "inert"}

            async def __call__(self, value):
                payloads.append(copy.deepcopy(value))
                return {"findings": [finding()], "_usage": {"total_tokens": 1}}

        early = step()
        first = await reason(early, Model())
        later = step([claim(), claim("later", "2020-01-04T00:00:00Z")], horizon="2020-01-05T00:00:00Z", index=1)
        later["previous"] = first
        await reason(later, Model())
        hindsight = step([claim(), claim("unknown", None)], perspective="hindsight", horizon=None, index=2)
        await reason(hindsight, Model())
        self.assertEqual([c["id"] for c in payloads[0]["claims"]], ["early"])
        self.assertNotIn("later", canonical(payloads[0]).decode())
        self.assertNotIn("unknown", canonical(payloads[0]).decode())
        self.assertEqual(payloads[1]["previous"]["findings"][0]["citations"][0]["claim_id"], "early")
        self.assertNotIn("previous", payloads[2])

    async def test_invalid_horizon_never_calls_model_and_bad_model_citation_never_checkpoints(self):
        calls = []

        class Model:
            identity = {}

            async def __call__(self, value):
                calls.append(value)
                return {"findings": [finding("unavailable")]}

        with self.assertRaises(ValueError):
            await reason(step([claim(known=None)]), Model())
        self.assertEqual(calls, [])
        with self.assertRaises(ValueError):
            await reason(step(), Model())
        self.assertEqual(len(calls), 1)


if __name__ == "__main__":
    unittest.main()
