"""Entity and event extractors: the shared page, Semantica, LangExtract's NIM rules, and the two Temporal Activities.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Semantica runs the real vendored library. The LangExtract tests run its real data classes and its real provider code
with a recording HTTP client standing in for the NIM endpoint (the wire shape is what is asserted); a live kimi-k3
call is a separate receipt, not a unit test.
"""

from __future__ import annotations

import json
from datetime import datetime, timezone
from types import SimpleNamespace

import pytest
from temporalio.testing import ActivityEnvironment

from server.temporal import entity_event_activities as activities
from server.tools.extractors.entity_events import langextract_extractor as lx_module
from server.tools.extractors.entity_events import pages, semantica_extractor

AT = datetime(2025, 6, 1, 14, 46, tzinfo=timezone.utc)


def message(ordinal: int, body: str, record_id: str | None = None, sender: str = "self", recipients: tuple[str, ...] = ("+18105550101",)):
    return pages.Message(record_id or f"00000000-0000-4000-8000-{ordinal:012d}", ordinal, AT, body, sender, recipients)


# --------------------------------------------------------------------------- the shared page

def test_page_merges_a_name_across_messages_and_reports_where_the_window_ended():
    builder = pages.PageBuilder("semantica", "1")
    builder.add_entity("people", "Katherine Doe", "r1", "Katherine Doe")
    builder.add_entity("people", "katherine  doe", "r2")
    builder.add_entity("places", "Flint", "r2")
    builder.add_event("Court date", "r2", date="2025-07-02", event_type="court", people=["Katherine Doe"])
    builder.add_event("Odd", "r1", date="next Tuesday", event_type="not-a-type", when="next Tuesday")
    window = [message(0, "a"), message(1, "b")]
    page = builder.page(window, -1)
    assert [p["name"] for p in page["people"]] == ["Katherine Doe"]
    assert [m["record_id"] for m in page["people"][0]["mentions"]] == ["r1", "r2"]
    assert page["last_ordinal"] == 1 and page["messages"] == 2 and page["done"] is True
    court, odd = page["events"]
    assert court["date"] == "2025-07-02" and court["type"] == "court"
    assert odd["date"] is None and odd["type"] == "other" and odd["when"] == "next Tuesday"
    assert set(page) >= {"extractor", "extractor_version", "after_ordinal", "last_ordinal", "messages", "done", "people", "places", "organizations", "events"}


def test_a_full_window_is_not_done():
    window = [message(i, "x") for i in range(pages.PAGE_MESSAGES)]
    assert pages.PageBuilder("x", "1").page(window, -1)["done"] is False


def test_read_window_reads_the_same_keyset_page_the_engine_does():
    seen = {}

    class Result:
        def mappings(self):
            return [{"id": "r1", "ordinal": 3, "occurred_at": AT, "body": "hi",
                     "participants": [{"role": "sender", "identifier": "self"}, {"role": "recipient", "identifier": "+1810"}, {"role": "unknown", "identifier": "self"}]}]

    class Conn:
        def execute(self, statement, params):
            seen["sql"], seen["params"] = str(statement), params
            return Result()

    window = pages.read_window(Conn(), "g1", 2, 50)
    assert seen["params"] == {"g": "g1", "after": 2, "limit": 50}
    assert "record_ordinal > :after" in seen["sql"] and "ORDER BY record_ordinal" in seen["sql"] and "record_type = 'message'" in seen["sql"]
    assert window[0].sender == "self" and window[0].recipients == ("+1810",)


# --------------------------------------------------------------------------- semantica

def test_semantica_finds_names_and_ties_each_to_its_message():
    window = [
        message(0, "Morning Katherine Doe, can you get Emma from Lincoln Elementary School at 3?"),
        message(1, "Meet at Flint City on June 5, 2025. Call Acme Corp Inc about it."),
        message(2, "ok thanks"),
        message(3, ""),
    ]
    page = semantica_extractor.extract_page(window, -1)
    assert page["extractor"] == "semantica" and page["extractor_version"].startswith("vendored-")
    names = {p["name"] for p in page["people"]}
    assert any("Katherine Doe" in name for name in names)
    assert any(o["name"].endswith("Acme Corp Inc") for o in page["organizations"])
    by_record = {m["record_id"] for p in page["people"] for m in p["mentions"]}
    assert by_record <= {w.record_id for w in window[:2]}
    for person in page["people"]:
        for mention in person["mentions"]:
            body = next(w.body for w in window if w.record_id == mention["record_id"])
            assert mention["text"] in body, "a mention's text must be copied from its message"
    assert page["messages"] == 4 and page["last_ordinal"] == 3


# --------------------------------------------------------------------------- langextract

def test_reply_problem_names_every_failure_the_default_extractor_retries():
    assert lx_module.reply_problem(None, "stop") == "empty reply"
    assert lx_module.reply_problem("   ", "stop") == "empty reply"
    assert lx_module.reply_problem("!!!!!!!!", "stop").startswith("junk reply")
    assert "cut off" in lx_module.reply_problem('{"extractions": []}', "length")
    assert "not JSON" in lx_module.reply_problem("{nope", "stop")
    assert "one JSON object" in lx_module.reply_problem("[1]", "stop")
    assert lx_module.reply_problem('{"extractions": []}', "stop") is None


def test_config_refuses_glm_and_too_few_tokens_and_skips_without_a_key(tmp_path, monkeypatch):
    for name in (lx_module.ENV_API_KEY_FILE, lx_module.ENV_MODEL_ID, lx_module.ENV_MAX_TOKENS, lx_module.ENV_BASE_URL):
        monkeypatch.delenv(name, raising=False)
    with pytest.raises(lx_module.ExtractorUnavailable):
        lx_module.config_from_env()
    key = tmp_path / "key"
    key.write_text("NVIDIA_API_KEY=nvapi-test-0000000000\n")
    monkeypatch.setenv(lx_module.ENV_API_KEY_FILE, str(tmp_path / "absent"))
    with pytest.raises(lx_module.ExtractorUnavailable):
        lx_module.config_from_env()
    monkeypatch.setenv(lx_module.ENV_API_KEY_FILE, str(key))
    config = lx_module.config_from_env()
    assert (config.model_id, config.base_url, config.max_tokens, config.api_key) == (
        "moonshotai/kimi-k3", "https://integrate.api.nvidia.com/v1", 6000, "nvapi-test-0000000000")
    monkeypatch.setenv(lx_module.ENV_MODEL_ID, "z-ai/GLM-5.1")
    with pytest.raises(ValueError, match="GLM"):
        lx_module.config_from_env()
    monkeypatch.setenv(lx_module.ENV_MODEL_ID, "")
    monkeypatch.setenv(lx_module.ENV_MAX_TOKENS, "1000")
    with pytest.raises(ValueError, match="1500"):
        lx_module.config_from_env()


def test_window_text_maps_a_span_back_to_its_message():
    window = [message(0, "Hi Katherine"), message(1, "Court on 2025-07-02", sender="+1810", recipients=("self",))]
    text, spans = lx_module.window_text(window)
    first = spans[0]
    assert text[first[0]:first[1]] == "Hi Katherine"
    assert lx_module._message_for((spans[1][0] + 9, spans[1][0] + 19), "2025-07-02", text, spans) == window[1].record_id
    # no aligned span: found by where the text occurs
    assert lx_module._message_for(None, "Katherine", text, spans) == window[0].record_id
    assert lx_module._message_for(None, "nowhere", text, spans) is None


def test_extractions_become_a_page_and_unmappable_ones_are_dropped():
    from langextract import data

    window = [message(0, "Hi Katherine, court is on 2025-07-02 at Lincoln Elementary")]
    text, spans = lx_module.window_text(window)
    extractions = [
        data.Extraction(extraction_class="person", extraction_text="Katherine"),
        data.Extraction(extraction_class="organization", extraction_text="Lincoln Elementary"),
        data.Extraction(extraction_class="place", extraction_text="Nowhere Town"),
        data.Extraction(extraction_class="event", extraction_text="court is on 2025-07-02",
                        attributes={"title": "Court date", "type": "court", "date": "2025-07-02"}),
        data.Extraction(extraction_class="mood", extraction_text="Hi"),
    ]
    page = lx_module.page_from_extractions(extractions, window, text, spans, -1, version="1.7.0", model_id="moonshotai/kimi-k3")
    assert page["extractor"] == "langextract" and page["extractor_version"] == "1.7.0" and page["model_id"] == "moonshotai/kimi-k3"
    assert [p["name"] for p in page["people"]] == ["Katherine"]
    assert [o["name"] for o in page["organizations"]] == ["Lincoln Elementary"]
    assert page["places"] == []
    assert page["events"][0]["title"] == "Court date" and page["events"][0]["date"] == "2025-07-02"


class FakeClient:
    """Stands in for the OpenAI SDK client: records every request body and replays scripted replies."""

    def __init__(self, replies):
        self.replies = list(replies)
        self.requests = []
        self.chat = SimpleNamespace(completions=SimpleNamespace(create=self.create))

    def create(self, **params):
        self.requests.append(params)
        content, finish = self.replies.pop(0)
        return SimpleNamespace(choices=[SimpleNamespace(message=SimpleNamespace(content=content, refusal=None), finish_reason=finish)])


def nim_model(replies):
    model = lx_module.build_model(lx_module.Config("https://integrate.api.nvidia.com/v1", "moonshotai/kimi-k3", "nvapi-test-0000000000", 6000))
    model._client = FakeClient(replies)
    return model


GOOD = json.dumps({"extractions": [{"person": "Katherine", "person_attributes": {}}]})


def test_a_bad_reply_is_retried_once_with_thinking_on_and_the_wire_shape_is_kimi_nims():
    model = nim_model([("", "stop"), (GOOD, "stop")])
    output = model._process_single_prompt("short prompt", {"max_output_tokens": 6000})
    assert output.output == GOOD
    first, second = model._client.requests
    assert first["extra_body"] == {"chat_template_kwargs": {"thinking": False}}
    assert second["extra_body"] == {"chat_template_kwargs": {"thinking": True}}
    assert first["response_format"] == {"type": "json_object"}
    assert first["max_tokens"] == 6000 and first["model"] == "moonshotai/kimi-k3"


def test_the_configured_token_budget_reaches_the_request_through_the_providers_own_infer():
    model = nim_model([(GOOD, "stop")])
    outputs = list(model.infer(["short prompt"]))
    assert outputs[0][0].output == GOOD
    assert model._client.requests[0]["max_tokens"] == 6000, "max_tokens must be >= 1500 on every call (a reasoning model returns empty content below)"


def test_a_long_prompt_starts_with_thinking_on():
    model = nim_model([(GOOD, "stop")])
    model._process_single_prompt("x" * 30000, {})
    assert model._client.requests[0]["extra_body"] == {"chat_template_kwargs": {"thinking": True}}


def test_two_bad_replies_fail_loudly_with_the_reason():
    from langextract.core import exceptions

    model = nim_model([("!!!!!!", "stop"), ("not json", "stop")])
    with pytest.raises(exceptions.InferenceRuntimeError, match="after one retry with thinking on"):
        model._process_single_prompt("short prompt", {})
    assert len(model._client.requests) == 2


# --------------------------------------------------------------------------- the Activities

def test_semantica_activity_returns_the_shared_page(monkeypatch):
    monkeypatch.setattr(activities, "_window", lambda params: [message(0, "Call Acme Corp Inc today")])
    page = ActivityEnvironment().run(
        activities.extract_entities_events_semantica_activity, activities.ExternalPageParams(generation_id="g", after_ordinal=-1)
    )
    assert page["extractor"] == "semantica" and page["messages"] == 1 and page["done"] is True


def test_langextract_activity_is_skipped_with_a_reason_when_no_key_is_mounted(monkeypatch):
    monkeypatch.delenv(lx_module.ENV_API_KEY_FILE, raising=False)
    page = ActivityEnvironment().run(
        activities.extract_entities_events_langextract_activity, activities.ExternalPageParams(generation_id="g", after_ordinal=4)
    )
    assert page["skipped"] is True and lx_module.ENV_API_KEY_FILE in page["reason"] and page["last_ordinal"] == 4


def test_langextract_activity_fails_non_retryably_on_a_glm_model(tmp_path, monkeypatch):
    from temporalio.exceptions import ApplicationError

    key = tmp_path / "key"
    key.write_text("nvapi-test-0000000000")
    monkeypatch.setenv(lx_module.ENV_API_KEY_FILE, str(key))
    monkeypatch.setenv(lx_module.ENV_MODEL_ID, "z-ai/glm-5.1")
    with pytest.raises(ApplicationError) as error:
        ActivityEnvironment().run(
            activities.extract_entities_events_langextract_activity, activities.ExternalPageParams(generation_id="g")
        )
    assert error.value.non_retryable


def test_the_activities_are_registered_on_the_evidence_pipeline_worker():
    source = open("server/temporal/worker.py", encoding="utf-8").read()
    assert "extract_entities_events_semantica_activity," in source and "extract_entities_events_langextract_activity," in source


def test_the_python_activity_names_match_the_go_registry():
    go = open("modules/engine/extraction/flow/conversation.go", encoding="utf-8").read()
    assert f'"{activities.SEMANTICA_ACTIVITY}"' in go and f'"{activities.LANGEXTRACT_ACTIVITY}"' in go
