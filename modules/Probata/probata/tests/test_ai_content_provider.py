"""Verify AI-only provider routing using retained fixtures and fake SDK clients.

Inputs: synthetic exact scopes and deterministic clocks. Outputs: policy/accounting
assertions. Effects: retained proof files only, no inference or source mutation.
Choose for routing regressions beside the existing content/budget suite.
Byline: Codex / GPT-6.1-Sol / 2026-10-07.
"""
import json
import os
from pathlib import Path
from types import SimpleNamespace
from uuid import uuid4

import pytest

from server.analysis import ai_content_provider as p


class FakeError(Exception):
    """Model SDK status errors without printing an arbitrary provider body.

    Inputs: status/headers/body. Outputs: exception. Effects: none; choose without
    any network access or actual provider credentials.
    """
    def __init__(self, status=None, headers=None, body=None):
        """Bind synthetic SDK fields; inputs safe fixture values, outputs error, effects none; choose for classification tests."""
        self.status_code, self.body = status, body
        self.response = SimpleNamespace(headers=headers or {})
        self.request_id = "fixture-request-1"


@pytest.fixture
def harness(monkeypatch):
    """Retain one isolated fake-provider scope for deterministic routing tests.

    Inputs: proof-root environment. Outputs: router/clock/call fixtures. Effects:
    retained directories only; choose instead of temporary-directory cleanup.
    """
    root = Path(os.environ["AI_CONTENT_TEST_ROOT"]) / ("provider-" + uuid4().hex)
    root.mkdir(parents=True)
    clock, calls, reservations, receipts, outcomes = [1000.0], [], [], [], []
    options = {"max_tokens": 6000, "temperature": 0, "response_format": {"type": "json_object"}}
    profiles = tuple(p.Profile("nvidia", model, p.NIM_URL, "ai-nim-primary", "synthetic-key", dict(options))
                     for model in (*p.PRIMARY_MODELS, p.BACKUP_MODEL))
    def factory(profile):
        """Build fake SDK facade; inputs profile, outputs client, effects none; choose for exact request recording."""
        def create(**request):
            """Record one simulated request; inputs arguments, outputs reply/error, effects fixture list; choose without inference."""
            calls.append((profile.model_id, request))
            outcome = outcomes.pop(0) if outcomes else None
            if isinstance(outcome, Exception):
                raise outcome
            if outcome is not None:
                return outcome
            return SimpleNamespace(choices=[SimpleNamespace(message=SimpleNamespace(content='{"candidates":[]}'), finish_reason="stop")],
                                   model_dump=lambda **kwargs: {"model": profile.model_id, "usage": {"total_tokens": 12}, "choices": []})
        return SimpleNamespace(chat=SimpleNamespace(completions=SimpleNamespace(create=create)))
    def reserve(request, actual):
        """Record pre-call reservation; inputs request/profile, outputs distinct refs, effects fixture list; choose for ordering checks."""
        reservations.append((request, actual))
        return "file:///intent/" + str(len(reservations)), "file:///budget/" + str(len(reservations))
    def sleep(seconds):
        """Advance fake wall/monotonic time; inputs delay, outputs none, effects clock only; choose without real pacing waits."""
        clock[0] += seconds
    router = p.ProviderRouter(root, profiles, clock=lambda: clock[0], jitter=lambda: 1.0,
                              sleep=sleep, monotonic=lambda: clock[0], client_factory=factory)
    return SimpleNamespace(root=root, clock=clock, calls=calls, reservations=reservations, receipts=receipts,
                           outcomes=outcomes, profiles=profiles, factory=factory, reserve=reserve, router=router, sleep=sleep,
                           pin={"source_version_id": str(uuid4())}, source={"original_sha256": "a" * 64})


def complete(h, ordinal=0, *, pin=None, beat=lambda _: None):
    """Call one fake route with standard callbacks; inputs harness/ordinal, outputs result, effects retained fake state; choose for concise policy fixtures."""
    return h.router.complete(pin or h.pin, h.source, ordinal, "synthetic grounded window", h.reserve, h.receipts.append, beat)


def test_alternating_primaries_and_spacing_do_not_reserve_early(harness):
    """Alternate healthy assigned primaries and pace before taking another slot.

    Inputs: two ordinals and clock. Outputs: exact models/pre-reservation wait.
    Effects: retained synthetic state; choose for deterministic balanced routing.
    """
    h = harness
    assert complete(h, 0)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[0]
    waits = []
    def beat(label):
        """Inspect pre-reservation pacing; inputs safe label, outputs none, effects assertions; choose for no-scheduler-retry proof."""
        if label.startswith("pacing"):
            assert len(h.reservations) == 1
            waits.append(label)
    assert complete(h, 1, beat=beat)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[1]
    assert h.clock[0] == 1005 and waits
    assert len(h.calls) == len(h.reservations) == 2


@pytest.mark.parametrize("header,deadline", [("3600", 4600.0), ("Thu, 01 Jan 1970 01:16:40 GMT", 4600.0)])
def test_account_429_spans_models_and_sources_and_honors_retry_after(harness, header, deadline):
    """Block every same-credential NIM route across sources until Retry-After.

    Inputs: safe synthetic429 and another source. Outputs: shared deadline and
    zero extra reservations. Effects: retained cooldown; choose over NIM hopping.
    """
    h = harness
    h.outcomes.append(FakeError(429, {"retry-after": header}, {"error": {"code": "rate_limit_exceeded"}}))
    with pytest.raises(p.ProviderDeferred):
        complete(h)
    h.clock[0] = deadline - 1
    other = p.ProviderRouter(h.root, h.profiles, clock=lambda: h.clock[0], client_factory=h.factory)
    with pytest.raises(p.ProviderDeferred) as error:
        other.complete({"source_version_id": str(uuid4())}, h.source, 1, "other window", h.reserve, h.receipts.append, lambda _: None)
    assert error.value.delay == 1 and len(h.calls) == len(h.reservations) == 1
    h.clock[0] = deadline
    assert complete(h, 1)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[1]


def test_model_failure_leaves_other_primary_eligible_and_records_safe_error(harness):
    """Cool only the failed model and preserve safe failure coordinates.

    Inputs: simulated503/body with secrets. Outputs: alternate actual winner and
    redacted receipt. Effects: retained state; choose without assuming outage cause.
    """
    h = harness
    h.outcomes.append(FakeError(503, {"retry-after": "120", "authorization": "Bearer SECRET"},
                                {"error": {"code": "server_error", "message": "PRIVATE CASE TEXT SECRET"}}))
    with pytest.raises(p.ProviderDeferred):
        complete(h)
    h.clock[0] += 5
    assert complete(h)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[1]
    failure = h.receipts[0]["response"]
    assert failure["status"] == 503 and failure["request_id"] == "fixture-request-1"
    assert failure["not_before"] == 1120 and failure["retry_decision"] == "model_cooldown"
    assert "SECRET" not in json.dumps(h.receipts) and "PRIVATE CASE" not in json.dumps(h.receipts)


def test_paid_unknown_outcomes_are_charged_and_per_source_cap_persists(harness):
    """Charge paid unknown outcomes and enforce four calls across router instances.

    Inputs: disabled NIM scopes and fake cloud timeout. Outputs: retained paid
    reservations and cap failure. Effects: synthetic state; choose for paid safety.
    """
    h = harness
    cloud = p.Profile("ollama_cloud", "explicit-fixture-model", p.CLOUD_URL, "ai-ollama-cloud", "synthetic-cloud", h.profiles[0].options, True)
    h.router = p.ProviderRouter(h.root, (*h.profiles, cloud), clock=lambda: h.clock[0], jitter=lambda: 1, client_factory=h.factory)
    with h.router._locked("ai-nim-primary") as (directory, state):
        state["account_until"] = 100000
        h.router._persist(directory, state)
    h.outcomes.append(TimeoutError("unprinted secret"))
    with pytest.raises(p.ProviderDeferred):
        complete(h)
    h.router = p.ProviderRouter(h.root, (*h.profiles, cloud), clock=lambda: h.clock[0],
                               jitter=lambda: 1, client_factory=h.factory)
    for _ in range(3):
        h.clock[0] += 100
        complete(h)
    h.clock[0] += 100
    with pytest.raises(p.ProviderDeferred):
        complete(h)
    assert len(h.calls) == len(h.reservations) == 4
    with h.router._locked("ai-ollama-cloud") as (_, state):
        assert state["paid_sources"][h.pin["source_version_id"]] == 4
        assert sum(state["paid_days"].values()) == sum(r["paid_reserved_tokens"] for r in h.receipts)
        assert sum(state["paid_days"].values()) > 4 * 6000


def test_paid_daily_cap_and_cancellation_never_issue_unreserved_requests(harness):
    """Enforce daily paid reservations and cancellation immediately before SDK.

    Inputs: near-exhausted day/canceled callback. Outputs: no cloud request, then
    one conservatively charged canceled NIM claim. Effects: fixtures only.
    Choose for token-budget and pre-dispatch boundaries.
    """
    h = harness
    cloud = p.Profile("ollama_cloud", "explicit-fixture-model", p.CLOUD_URL, "ai-ollama-cloud", "synthetic-cloud", h.profiles[0].options, True)
    h.router = p.ProviderRouter(h.root, (*h.profiles, cloud), clock=lambda: h.clock[0], client_factory=h.factory)
    with h.router._locked("ai-nim-primary") as (directory, state):
        state["account_until"] = 100000
        h.router._persist(directory, state)
    with h.router._locked("ai-ollama-cloud") as (directory, state):
        state["paid_days"]["1970-01-01"] = p.PAID_TOKENS_PER_DAY - 1
        h.router._persist(directory, state)
    with pytest.raises(p.ProviderDeferred):
        complete(h)
    assert h.calls == h.reservations == []
    with h.router._locked("ai-nim-primary") as (directory, state):
        state["account_until"] = 0
        h.router._persist(directory, state)
    def canceled(_):
        """Stop at the request boundary; inputs label, outputs cancellation, effects none; choose for no-dispatch proof."""
        raise RuntimeError("synthetic cancellation")
    with pytest.raises(RuntimeError, match="synthetic cancellation"):
        complete(h, beat=canceled)
    assert len(h.reservations) == 1 and h.calls == []


def test_busy_lock_and_hard_credentials_are_not_blind_retries(harness):
    """Defer an occupied credential and pause hard authentication failures.

    Inputs: held Linux flock and synthetic401. Outputs: zero busy-slot usage,
    permanent pause across sources. Effects: fixtures; choose for worker coordination.
    """
    h = harness
    with h.router._locked("ai-nim-primary"):
        with pytest.raises(p.ProviderDeferred):
            complete(h)
    assert h.reservations == []
    h.outcomes.append(FakeError(401, body={"error": {"code": "invalid_api_key", "message": "SECRET"}}))
    with pytest.raises(p.ProviderUnavailable):
        complete(h)
    h.clock[0] += 10000
    with pytest.raises(p.ProviderUnavailable):
        complete(h, pin={"source_version_id": str(uuid4())})
    assert len(h.calls) == len(h.reservations) == 1


def test_cloud_endpoint_and_banned_model_are_refused(harness):
    """Reject local cloud aliases and banned models before I/O.

    Inputs: explicit invalid profiles. Outputs: permanent validation errors.
    Effects: none; choose without modifying shared global model policy.
    """
    h = harness
    for url, model in (("http://localhost:11434/v1", "glm-5.3"), (p.CLOUD_URL, "glm-5.1")):
        cloud = p.Profile("ollama_cloud", model, url, "ai-ollama-cloud", "synthetic", h.profiles[0].options, True)
        with pytest.raises(p.ProviderUnavailable):
            p.ProviderRouter(h.root, (*h.profiles, cloud))


def test_default_profiles_pin_parent_verified_request_options(harness, monkeypatch):
    """Pin the reviewed model options independently of fake transport profiles.

    Inputs: synthetic mounted key. Outputs: exact Kimi/Nemotron option assertions.
    Effects: retained fixture key only; choose without treating fake calls as live
    model capability proof or adding any production credential to logs.
    """
    key = harness.root / "synthetic-key.txt"
    key.write_text("synthetic-test-only")
    monkeypatch.setenv("ENTITY_MODEL_API_KEY_FILE", str(key))
    monkeypatch.delenv("AI_CONTENT_CLOUD_MODEL", raising=False)
    monkeypatch.delenv("OPENROUTER_API_KEY", raising=False)
    monkeypatch.delenv("OPENROUTER_API_KEY_FILE", raising=False)
    profiles = p.default_profiles()
    assert tuple(profile.model_id for profile in profiles) == (*p.PRIMARY_MODELS, p.BACKUP_MODEL)
    assert profiles[0].options["extra_body"] == {"chat_template_kwargs": {"thinking": False}}
    assert profiles[0].options["n"] == 1
    assert all(profile.options["extra_body"] == {"chat_template_kwargs": {"enable_thinking": False}} for profile in profiles[1:])
    assert all(profile.options["max_tokens"] == 6000 for profile in profiles)


def test_bad_model_profile_pauses_only_that_model_and_wrapper_defers(harness, monkeypatch):
    """Preserve configured alternatives and Temporal next-retry metadata.

    Inputs: fake404 and wrapper fixture. Outputs: model pause, alternate success,
    retryable typed ApplicationError. Effects: retained fixtures only; choose for
    hard-profile failures without global credential disablement.
    """
    from server.analysis import ai_content as ai
    from server.temporal import ai_content_activities as activities
    from temporalio.exceptions import ApplicationError
    h = harness
    h.outcomes.append(FakeError(404, body={"error": {"code": "model_not_found"}}))
    with pytest.raises(p.ProviderDeferred) as error:
        complete(h)
    assert error.value.delay >= 60
    h.clock[0] += error.value.delay
    assert complete(h)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[1]
    def deferred(*args, **kwargs):
        """Raise safe scheduled cooldown; inputs wrapper args, outputs typed error, effects none; choose for classification proof."""
        raise p.ProviderDeferred("safe cooldown", 3600, {"status": 429})
    monkeypatch.setattr(ai, "extract_candidates", deferred)
    with pytest.raises(ApplicationError) as wrapped:
        activities._run("extract_candidates", activities.AIContentParams())
    assert wrapped.value.type == "AIContentProviderDeferred" and not wrapped.value.non_retryable
    assert wrapped.value.next_retry_delay.total_seconds() == 3600


def test_free_route_pins_zero_price_and_uses_independent_credential_scope(harness):
    """Validate the approved free profile before using its independently cooled scope.

    Inputs: fake unavailable NIM and exact free profile. Outputs: free winner,
    forced zero pricing and no paid reservation. Effects: retained fake SDK state;
    choose without treating this synthetic test as live catalog/task validation.
    """
    h = harness
    free = p.Profile("openrouter_free", p.OPENROUTER_FREE_MODEL, p.OPENROUTER_URL,
                     "ai-openrouter-free", "synthetic-free", p.OPENROUTER_FREE_OPTIONS,
                     validation_ref="document:synthetic-live-validation")
    h.router = p.ProviderRouter(h.root, (*h.profiles, free), clock=lambda: h.clock[0], client_factory=h.factory)
    with h.router._locked("ai-nim-primary") as (directory, state):
        state["account_until"] = 100000
        h.router._persist(directory, state)
    result = complete(h)
    assert result["actual_provider"] == free.identity() and result["paid_reserved_tokens"] == 0
    assert h.calls[0][1]["extra_body"]["provider"] == {"max_price": {"prompt": 0, "completion": 0},
                                                       "allow_fallbacks": False, "require_parameters": True}
    from dataclasses import replace
    for bad in (replace(free, model_id="unvalidated:free"),
                replace(free, options={**p.OPENROUTER_FREE_OPTIONS, "extra_body": {}}),
                replace(free, validation_ref=None)):
        with pytest.raises(p.ProviderUnavailable):
            p.ProviderRouter(h.root, (*h.profiles, bad))


@pytest.mark.parametrize("count", [0, 2])
def test_ambiguous_completion_is_retained_and_unusable_on_first_read(harness, count):
    """Refuse empty or ambiguous choices before they can become a checkpoint.

    Inputs: fake SDK completion. Outputs: retained consumed request and unusable
    reply. Effects: synthetic files only; choose for first-use/reload consistency.
    """
    h = harness
    choices = [SimpleNamespace(message=SimpleNamespace(content='{"candidates":[]}'), finish_reason="stop")] * count
    h.outcomes.append(SimpleNamespace(choices=choices, model_dump=lambda **kwargs: {"choices": [{}] * count}))
    result = complete(h)
    assert len(h.reservations) == len(h.receipts) == 1 and result["raw_reply"] is None
    with pytest.raises(ValueError):
        p.decode_reply(result["raw_reply"], result["finish_reason"])


def test_invalid_optional_configuration_is_retained_without_stopping_nim(harness, monkeypatch):
    """Continue healthy primaries while retaining disabled optional repair reasons.

    Inputs: invalid optional key/reference/URL and valid synthetic NIM key.
    Outputs: explicit disabled plan entries, no optional calls, safe NVIDIA request
    ID. Effects: retained fake files/calls only; choose for independent branch failure.
    """
    h = harness
    key = h.root / "nim-fixture-key"
    key.write_text("synthetic-nim")
    monkeypatch.setenv("ENTITY_MODEL_API_KEY_FILE", str(key))
    monkeypatch.setenv("OPENROUTER_API_KEY_FILE", str(h.root / "missing-key"))
    monkeypatch.setenv("AI_CONTENT_OPENROUTER_PROOF_REF", "not-a-governed-ref")
    monkeypatch.setenv("AI_CONTENT_CLOUD_MODEL", "glm-5.3")
    monkeypatch.setenv("AI_CONTENT_CLOUD_BASE_URL", "http://localhost:11434/v1")
    profiles = p.default_profiles()
    assert len(profiles) == 5 and all(profile.disabled_reason for profile in profiles[3:])
    assert all(not profile.api_key for profile in profiles[3:])
    h.router = p.ProviderRouter(h.root, profiles, clock=lambda: h.clock[0], client_factory=h.factory)
    assert complete(h)["actual_provider"]["model_id"] == p.PRIMARY_MODELS[0]
    assert all(profile["disabled_reason"] for profile in h.router.identity()["profiles"][3:])
    error = FakeError(503, headers={"nvcf-reqid": "nvcf-correlation-123"})
    error.request_id = None
    assert p.safe_failure(error, h.clock[0])["request_id"] == "nvcf-correlation-123"
