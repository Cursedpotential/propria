"""Route AI extraction through credential-scoped remote providers with retained limits.

Inputs: approved request profiles, seven-pin source scope and reservation callback.
Outputs: one completion or a typed deferred/permanent failure. Effects: remote
requests and persistent AI-only cooldown/paid accounting under AI_CONTENT_ROOT.
Choose for AI content extraction, never shared SMS/agent defaults or local inference.
Byline: Codex / GPT-6.1-Sol / 2026-10-07.
"""
from __future__ import annotations

from contextlib import contextmanager
from dataclasses import dataclass, field
from datetime import datetime, timezone
from email.utils import parsedate_to_datetime
import fcntl
import json
import math
import os
from pathlib import Path
import random
import re
import time
from typing import Any, Callable, Iterator
from uuid import uuid4

PRIMARY_MODELS = ("moonshotai/kimi-k3", "nvidia/nemotron-3-super-120b-a12b")
BACKUP_MODEL = "nvidia/nemotron-3.5-lightning-30b-a3b"
NIM_URL = "https://integrate.api.nvidia.com/v1"
CLOUD_URL = "https://ollama.com/v1"
OPENROUTER_URL = "https://openrouter.ai/api/v1"
OPENROUTER_FREE_MODEL = "nvidia/nemotron-3-super-120b-a12b:free"
OPENROUTER_FREE_OPTIONS = {"temperature": 0, "response_format": {"type": "json_object"}, "max_tokens": 2048,
                          "extra_body": {"reasoning": {"effort": "low"},
                                         "provider": {"max_price": {"prompt": 0, "completion": 0},
                                                      "allow_fallbacks": False, "require_parameters": True}}}
MIN_SPACING = 5.0
BACKOFF_BASE = 60.0
BACKOFF_CEILING = 900.0
JITTER_CEILING = 5.0
PAID_CALLS_PER_SOURCE = 4
PAID_TOKENS_PER_DAY = 100000
MAX_OUTPUT_TOKENS = 6000
STATE_VERSION = "ai-provider-state-v1"
PLAN_VERSION = "ai-extraction-balanced-v1"
SYSTEM_PROMPT = "Return one JSON object with exact grounded source quotations."
SAFE_CODES = {"rate_limit_exceeded", "invalid_api_key", "insufficient_quota",
              "billing_hard_limit_reached", "invalid_request_error", "model_not_found",
              "permission_denied", "authentication_error", "server_error", "overloaded_error"}


class ProviderDeferred(RuntimeError):
    """Request a Temporal retry without sleeping or consuming another request slot.

    Inputs: safe reason, remaining delay and optional safe failure metadata.
    Outputs: typed error. Effects: none; choose for cooldown, spacing or busy lock.
    """
    def __init__(self, reason: str, delay: float, metadata: dict[str, Any] | None = None):
        """Initialize retry metadata; inputs reason/delay, outputs error, effects none; choose over untyped transport errors."""
        super().__init__(reason)
        self.delay = max(1.0, delay)
        self.metadata = metadata or {}


class ProviderUnavailable(RuntimeError):
    """Stop an AI route requiring credential, billing or request-profile correction.

    Inputs: safe reason and metadata. Outputs: permanent error. Effects: none;
    choose over blind retries after a hard provider failure.
    """
    def __init__(self, reason: str, metadata: dict[str, Any] | None = None):
        """Initialize permanent metadata; inputs safe fields, outputs error, effects none; choose for explicit operational repair."""
        super().__init__(reason)
        self.metadata = metadata or {}


@dataclass(frozen=True)
class Profile:
    """Pin one remote model and its exact approved completion options.

    Inputs: provider, model, credential scope, key and options. Outputs: immutable
    profile. Effects: none; choose instead of implicit provider or model fallback.
    """
    provider: str
    model_id: str
    base_url: str
    credential_scope: str
    api_key: str = field(repr=False, compare=False)
    options: dict[str, Any] = field(default_factory=dict)
    paid: bool = False
    validation_ref: str | None = None
    disabled_reason: str | None = None

    def identity(self) -> dict[str, Any]:
        """Expose secret-free profile identity; inputs self, outputs exact plan fields, effects none; choose for retained provenance."""
        return {"provider": self.provider, "model_id": self.model_id, "base_url": self.base_url,
                "credential_scope": self.credential_scope, "options": self.options, "paid": self.paid,
                "validation_ref": self.validation_ref, "disabled_reason": self.disabled_reason}


def _key_file(name: str) -> str:
    """Read a mounted API key without putting its value in configuration identity.

    Inputs: key-file environment name. Outputs: key. Effects: credential read;
    choose for this remote AI route, never shell sourcing or secret logging.
    """
    value = os.environ.get(name, "").strip()
    path = Path(value)
    if not value or not path.is_absolute() or not path.is_file():
        raise ProviderUnavailable("AI provider credential file is unavailable")
    try:
        key = path.read_text(encoding="utf-8").strip()
    except (OSError, UnicodeError):
        raise ProviderUnavailable("AI provider credential file is unreadable") from None
    label, separator, content = key.partition("=")
    if separator and "KEY" in label and not any(c.isspace() for c in label):
        key = content.strip().strip("\"'")
    if not key or any(c.isspace() for c in key):
        raise ProviderUnavailable("AI provider credential file is malformed")
    return key


def default_profiles() -> tuple[Profile, ...]:
    """Load the explicit AI-only NVIDIA plan and optional paid cloud route.

    Inputs: mounted ENTITY_MODEL_API_KEY_FILE and explicit AI cloud configuration.
    Outputs: approved remote profiles, including explicitly disabled optional
    branches with safe repair reasons. Effects: configuration/credential reads;
    choose without changing shared LangExtract/SMS or global agent defaults.
    """
    key = _key_file("ENTITY_MODEL_API_KEY_FILE")
    common = {"temperature": 0, "response_format": {"type": "json_object"}, "max_tokens": MAX_OUTPUT_TOKENS}
    profiles = []
    for model in (*PRIMARY_MODELS, BACKUP_MODEL):
        options = dict(common)
        options["extra_body"] = {"chat_template_kwargs": {"thinking": False} if model == PRIMARY_MODELS[0] else {"enable_thinking": False}}
        if model == PRIMARY_MODELS[0]:
            options["n"] = 1
        profiles.append(Profile("nvidia", model, NIM_URL, "ai-nim-primary", key, options))
    if os.environ.get("OPENROUTER_API_KEY_FILE") or os.environ.get("OPENROUTER_API_KEY"):
        try:
            proof = os.environ.get("AI_CONTENT_OPENROUTER_PROOF_REF", "").strip()
            if not re.fullmatch(r"document:[a-zA-Z0-9_-]{1,128}", proof):
                raise ProviderUnavailable("AI free route requires its governed catalog and task-validation reference")
            free_key = (_key_file("OPENROUTER_API_KEY_FILE") if os.environ.get("OPENROUTER_API_KEY_FILE")
                        else os.environ["OPENROUTER_API_KEY"].strip())
            if not free_key or any(c.isspace() for c in free_key):
                raise ProviderUnavailable("AI free route credential is malformed")
            profiles.append(Profile("openrouter_free", OPENROUTER_FREE_MODEL, OPENROUTER_URL,
                                    "ai-openrouter-free", free_key, OPENROUTER_FREE_OPTIONS, False, proof))
        except ProviderUnavailable as error:
            profiles.append(Profile("openrouter_free", OPENROUTER_FREE_MODEL, OPENROUTER_URL,
                                    "ai-openrouter-free", "", OPENROUTER_FREE_OPTIONS, disabled_reason=str(error)))
    model = os.environ.get("AI_CONTENT_CLOUD_MODEL", "").strip()
    if model:
        cloud_options = {**common, "reasoning_effort": "low"}
        try:
            if model != "glm-5.3":
                raise ProviderUnavailable("AI cloud model differs from the approved glm-5.3 profile")
            url = os.environ.get("AI_CONTENT_CLOUD_BASE_URL", CLOUD_URL).rstrip("/")
            if url != CLOUD_URL:
                raise ProviderUnavailable("AI cloud route requires https://ollama.com/v1")
            profiles.append(Profile("ollama_cloud", model, url, "ai-ollama-cloud",
                                    _key_file("AI_CONTENT_CLOUD_API_KEY_FILE"), cloud_options, True))
        except ProviderUnavailable as error:
            profiles.append(Profile("ollama_cloud", "glm-5.3", CLOUD_URL, "ai-ollama-cloud",
                                    "", cloud_options, True, disabled_reason=str(error)))
    return tuple(profiles)


def retry_after(value: str | None, now: float) -> float | None:
    """Convert a valid Retry-After header to an absolute UTC deadline.

    Inputs: delta-seconds or HTTP date and current epoch. Outputs: deadline or
    None. Effects: none; choose over clipping provider-requested cooldowns.
    """
    if not value:
        return None
    try:
        if re.fullmatch(r"[0-9]+", value.strip()):
            return now + int(value.strip())
        date = parsedate_to_datetime(value)
        if date.tzinfo is None:
            return None
        return max(now, date.timestamp())
    except (ValueError, TypeError, OverflowError):
        return None


def safe_failure(error: Exception, now: float) -> dict[str, Any]:
    """Retain useful error coordinates without arbitrary error bodies or source text.

    Inputs: SDK exception and clock. Outputs: allowlisted status/code/request ID,
    cause class and Retry-After deadline. Effects: none; choose over str(error),
    raw response bodies or class-name-only diagnostic receipts.
    """
    response = getattr(error, "response", None)
    headers = getattr(response, "headers", {}) or {}
    status = getattr(error, "status_code", None)
    body = getattr(error, "body", None)
    detail = body.get("error", body) if isinstance(body, dict) else {}
    code = detail.get("code") if isinstance(detail, dict) else None
    request_id = getattr(error, "request_id", None) or headers.get("x-request-id") or headers.get("nvcf-reqid")
    if not isinstance(request_id, str) or not re.fullmatch(r"[A-Za-z0-9_.:-]{1,256}", request_id):
        request_id = None
    cause = getattr(error, "__cause__", None)
    return {"error_type": type(error).__name__, "status": status if isinstance(status, int) else None,
            "error_code": code if isinstance(code, str) and code in SAFE_CODES else None, "request_id": request_id,
            "cause_class": type(cause).__name__ if cause is not None else None,
            "retry_after_until": retry_after(headers.get("retry-after"), now)}


def decode_reply(raw: str | None, finish_reason: str | None) -> dict[str, Any]:
    """Decode one JSON object with an optional whole-response JSON code fence.

    Inputs: exact retained reply and finish reason. Outputs: parsed object.
    Effects: none; choose for the AI cloud route's verified fenced JSON shape,
    never rewrite quotations, remove prose, or accept truncated/ambiguous replies.
    The original raw reply remains the evidence receipt.
    """
    if not isinstance(raw, str) or not raw.strip() or finish_reason == "length":
        raise ValueError("AI completion is empty or truncated")
    text = raw.strip()
    fence = re.fullmatch(r"```(?:json)?[ \t]*\r?\n(.*)\r?\n```[ \t]*", text, re.DOTALL | re.IGNORECASE)
    if fence:
        text = fence.group(1)
    value = json.loads(text)
    if not isinstance(value, dict):
        raise ValueError("AI completion must contain one JSON object")
    return value


def _sync_directories(parent: Path, root: Path) -> None:
    """Sync provider-state directory entries from child through existing AI root.

    Inputs: retained parent/root. Outputs: none. Effects: directory fsync;
    choose before releasing state locks or dispatch, not hardware power-loss proof.
    """
    directory = parent.resolve()
    if not directory.is_relative_to(root):
        raise ProviderUnavailable("AI provider state escaped its configured root")
    while True:
        descriptor = os.open(directory, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
        if directory == root:
            return
        directory = directory.parent


class ProviderRouter:
    """Select and dispatch one eligible remote AI request under a credential lock.

    Inputs: existing retained AI root and approved profiles. Outputs: completion
    with actual provider/model or typed operational failure. Effects: durable
    cooldown/paid state and SDK calls; choose only for AI workers sharing this mount.
    """
    def __init__(self, root: Path, profiles: tuple[Profile, ...], *, clock: Callable[[], float] = time.time,
                 jitter: Callable[[], float] = lambda: random.uniform(1, JITTER_CEILING),
                 sleep: Callable[[float], None] = time.sleep, monotonic: Callable[[], float] = time.monotonic,
                 client_factory: Callable[[Profile], Any] | None = None):
        """Bind route dependencies; inputs root/profiles/clock/transport, outputs router, effects validation only; choose injectable transport for tests."""
        self.root, self.profiles, self.clock, self.jitter = root.resolve(), profiles, clock, jitter
        self.sleep, self.monotonic = sleep, monotonic
        self.client_factory = client_factory or self._client
        if not 3 <= len(profiles) <= 5 or tuple(p.model_id for p in profiles[:2]) != PRIMARY_MODELS or profiles[2].model_id != BACKUP_MODEL:
            raise ProviderUnavailable("AI provider primary/backup plan differs from approved literals")
        optional = tuple(p.provider for p in profiles[3:])
        if optional not in ((), ("openrouter_free",), ("ollama_cloud",), ("openrouter_free", "ollama_cloud")):
            raise ProviderUnavailable("AI optional provider order differs from the approved plan")
        for profile in profiles:
            if profile.disabled_reason:
                if profile not in profiles[3:] or profile.api_key:
                    raise ProviderUnavailable("only optional AI providers may be explicitly disabled")
                continue
            if profile.provider not in {"nvidia", "openrouter_free", "ollama_cloud"} or profile.paid != (profile.provider == "ollama_cloud") or "glm-5.1" in profile.model_id.lower():
                raise ProviderUnavailable("AI provider/model differs from the approved remote route")
            if not re.fullmatch(r"[a-z0-9_-]{1,80}", profile.credential_scope):
                raise ProviderUnavailable("AI credential scope label is invalid")
            approved_url = {"nvidia": NIM_URL, "ollama_cloud": CLOUD_URL, "openrouter_free": OPENROUTER_URL}[profile.provider]
            if profile.base_url != approved_url:
                raise ProviderUnavailable("AI provider endpoint is outside the approved remote route")
            if profile.provider == "openrouter_free" and (profile.model_id != OPENROUTER_FREE_MODEL or
                profile.options != OPENROUTER_FREE_OPTIONS or not profile.validation_ref or profile.credential_scope != "ai-openrouter-free"):
                raise ProviderUnavailable("AI free route lacks its exact zero-price profile and validation")
            maximum = profile.options.get("max_tokens")
            if isinstance(maximum, bool) or not isinstance(maximum, int) or not 0 < maximum <= MAX_OUTPUT_TOKENS:
                raise ProviderUnavailable("AI output token ceiling is outside the approved bound")
            if set(profile.options) - {"temperature", "response_format", "max_tokens", "extra_body", "reasoning_effort", "n"}:
                raise ProviderUnavailable("AI completion profile contains unsupported request fields")

    @staticmethod
    def _client(profile: Profile) -> Any:
        """Construct the existing remote SDK with retries disabled; inputs profile, outputs client, effects none until request; choose without local inference."""
        from openai import OpenAI
        return OpenAI(api_key=profile.api_key, base_url=profile.base_url, max_retries=0)

    def identity(self) -> dict[str, Any]:
        """Expose the full secret-free request plan; inputs router, outputs identity, effects none; choose for immutable checkpoint keys."""
        return {"version": PLAN_VERSION, "profiles": [p.identity() for p in self.profiles],
                "system_prompt": SYSTEM_PROMPT,
                "reply_decoder": "whole_response_json_fence_v1",
                "primary_assignment": "prepared_chunk_ordinal_parity", "minimum_spacing_seconds": MIN_SPACING,
                "paid_calls_per_source": PAID_CALLS_PER_SOURCE, "paid_tokens_per_utc_day": PAID_TOKENS_PER_DAY,
                "paid_reservation": "utf8_request_bytes_plus_1024_plus_max_tokens",
                "cooldown": {"base_seconds": BACKOFF_BASE, "ceiling_seconds": BACKOFF_CEILING,
                             "positive_jitter_max_seconds": JITTER_CEILING, "retry_after": "never_shorten",
                             "rate_limit_scope": "all_models_using_this_credential_scope"},
                "sdk_retries": 0, "timeout": {"connect": 5, "read": 600, "write": 600, "pool": 600}}

    @contextmanager
    def _locked(self, scope: str, beat: Callable[[str], None] = lambda _: None) -> Iterator[tuple[Path, dict[str, Any]]]:
        """Acquire a nonblocking Linux credential lock and load persistent shared state.

        Inputs: nonsecret approved scope. Outputs: state directory/dictionary.
        Effects: retained lock file and flock, at most five seconds of cancellable
        acquisition waiting; choose across AI sources/workers,
        never claim coordination with unrelated platform applications.
        """
        directory = self.root / "_provider_state" / "v1" / scope
        directory.mkdir(parents=True, exist_ok=True)
        if not directory.resolve().is_relative_to(self.root):
            raise ProviderUnavailable("AI provider state escaped its configured root")
        descriptor = os.open(directory / "dispatch.lock", os.O_RDWR | os.O_CREAT, 0o600)
        try:
            started = self.monotonic()
            while True:
                try:
                    fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    break
                except BlockingIOError:
                    remaining = MIN_SPACING - (self.monotonic() - started)
                    if remaining <= 0:
                        raise ProviderDeferred("AI credential dispatch is busy", BACKOFF_BASE) from None
                    beat("waiting briefly for AI credential dispatch")
                    self.sleep(min(0.25, remaining))
            path = directory / "current.json"
            try:
                if path.is_symlink():
                    raise ValueError("state pointer must be a retained file hardlink")
                state = json.loads(path.read_bytes()) if path.exists() else {
                    "version": STATE_VERSION, "credential_scope": scope, "account_until": 0,
                    "last_dispatch": 0, "account_failures": 0, "models": {}, "paid_days": {}, "paid_sources": {}, "paused": None}
            except (ValueError, OSError):
                raise ProviderUnavailable("retained AI provider state is unreadable or invalid") from None
            if (not isinstance(state, dict) or state.get("version") != STATE_VERSION or state.get("credential_scope") != scope or
                any(not isinstance(state.get(name), dict) for name in ("models", "paid_days", "paid_sources")) or
                any(not isinstance(state.get(name), (int, float)) or not math.isfinite(state[name]) for name in ("account_until", "last_dispatch"))):
                raise ProviderUnavailable("retained AI provider state identity differs")
            yield directory, state
        finally:
            os.close(descriptor)

    def _persist(self, directory: Path, state: dict[str, Any]) -> None:
        """Retain a state snapshot and atomically advance its current pointer.

        Inputs: locked directory and state. Outputs: none. Effects: immutable
        snapshot, retained hardlink and atomic current replacement with fsync;
        choose without deleting prior state snapshots or reservation history.
        """
        snapshot = directory / (uuid4().hex + ".json")
        encoded = json.dumps(state, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()
        descriptor = os.open(snapshot, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o400)
        with os.fdopen(descriptor, "wb") as output:
            output.write(encoded)
            output.flush()
            os.fsync(output.fileno())
        pointer = directory / (uuid4().hex + ".pending")
        os.link(snapshot, pointer)
        os.replace(pointer, directory / "current.json")
        _sync_directories(directory, self.root)

    def _deadline(self, state: dict[str, Any], profile: Profile) -> float:
        """Read account/model/spacing eligibility; inputs state/profile, outputs deadline, effects none; choose before any request reservation."""
        model = state["models"].get(profile.model_id, {})
        return max(state["account_until"], model.get("until", 0))

    def _alternative_delay(self, pin: dict[str, str], prompt: str, beat: Callable[[str], None], *, minimum: float = BACKOFF_BASE) -> float | None:
        """Find a permitted nonpaused alternative after a hard provider failure.

        Inputs: source pins, exact prompt and cancellation callback. Outputs:
        retry delay or None. Effects: locked state reads only; choose before
        declaring all configured paths unavailable, never retry the paused path.
        """
        waits = []
        scopes: dict[str, list[Profile]] = {}
        for profile in self.profiles:
            if profile.disabled_reason:
                continue
            scopes.setdefault(profile.credential_scope, []).append(profile)
        for scope, profiles in scopes.items():
            try:
                with self._locked(scope, beat) as (_, state):
                    now = self.clock()
                    day = datetime.fromtimestamp(now, timezone.utc).date().isoformat()
                    if state["paused"]:
                        continue
                    for profile in profiles:
                        if state["models"].get(profile.model_id, {}).get("paused"):
                            continue
                        if profile.paid and state["paid_sources"].get(pin["source_version_id"], 0) >= PAID_CALLS_PER_SOURCE:
                            continue
                        delay = max(minimum, self._deadline(state, profile) - now, state["last_dispatch"] + MIN_SPACING - now)
                        if profile.paid:
                            allowance = len(prompt.encode("utf-8")) + 1024 + profile.options["max_tokens"]
                            if state["paid_days"].get(day, 0) + allowance > PAID_TOKENS_PER_DAY:
                                tomorrow = datetime.fromtimestamp(now, timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0).timestamp() + 86400
                                delay = max(delay, tomorrow - now)
                        waits.append(delay)
            except ProviderDeferred as error:
                waits.append(max(BACKOFF_BASE, error.delay))
        return min(waits) if waits else None

    def complete(self, pin: dict[str, str], source: dict[str, Any], ordinal: int, prompt: str,
                 reserve: Callable[[dict[str, Any], dict[str, Any]], tuple[str, str]],
                 retain: Callable[[dict[str, Any]], None], beat: Callable[[str], None]) -> dict[str, Any]:
        """Dispatch one healthy eligible request and retain its actual provider identity.

        Inputs: exact source pins, ordinal/prompt, durable reserve/receipt callbacks
        and cancellation boundary. Outputs: raw reply/actual profile/claim refs.
        Effects: one remote call at most, persisted cooldown/paid reservations;
        choose per chunk attempt, sharing the caller's two-slot/source512 ledgers.
        """
        order = [self.profiles[ordinal % 2], self.profiles[1 - ordinal % 2], *self.profiles[2:]]
        waits, unavailable, busy_scopes = [], [], set()
        for profile in order:
            if profile.disabled_reason:
                unavailable.append(profile.provider)
                continue
            if profile.credential_scope in busy_scopes:
                continue
            try:
                with self._locked(profile.credential_scope, beat) as (directory, state):
                    now = self.clock()
                    model_state = state["models"].get(profile.model_id, {})
                    if state["paused"] or model_state.get("paused"):
                        unavailable.append(profile.provider)
                        continue
                    deadline = self._deadline(state, profile)
                    if deadline > now:
                        waits.append(deadline - now)
                        continue
                    pacing_started = self.monotonic()
                    while state["last_dispatch"] + MIN_SPACING > self.clock():
                        remaining = MIN_SPACING - (self.monotonic() - pacing_started)
                        if remaining <= 0:
                            raise ProviderDeferred("AI credential clock/pacing requires retry", BACKOFF_BASE)
                        beat("pacing AI credential dispatch")
                        self.sleep(max(0, min(0.25, remaining, state["last_dispatch"] + MIN_SPACING - self.clock())))
                    now = self.clock()
                    messages = [{"role": "system", "content": SYSTEM_PROMPT},
                                {"role": "user", "content": prompt}]
                    request = {"model": profile.model_id, "messages": messages, **profile.options}
                    allowance = len(json.dumps(messages, ensure_ascii=False).encode("utf-8")) + 1024 + request["max_tokens"]
                    day = datetime.fromtimestamp(now, timezone.utc).date().isoformat()
                    source_key = pin["source_version_id"]
                    if profile.paid and (state["paid_sources"].get(source_key, 0) >= PAID_CALLS_PER_SOURCE or
                                         state["paid_days"].get(day, 0) + allowance > PAID_TOKENS_PER_DAY):
                        unavailable.append("paid_cap")
                        continue
                    actual = profile.identity()
                    intent_ref, budget_ref = reserve(request, actual)
                    state["last_dispatch"] = now
                    if profile.paid:
                        state["paid_sources"][source_key] = state["paid_sources"].get(source_key, 0) + 1
                        state["paid_days"][day] = state["paid_days"].get(day, 0) + allowance
                    self._persist(directory, state)
                    common = {"source": source, "pins": pin, "request": request, "actual_provider": actual,
                              "intent_ref": intent_ref, "budget_ref": budget_ref,
                              "paid_reserved_tokens": allowance if profile.paid else 0, "paid_utc_day": day if profile.paid else None}
                    client = self.client_factory(profile)
                    beat("starting reserved AI provider request")
                    try:
                        response = client.chat.completions.create(**request)
                    except Exception as error:
                        failure = safe_failure(error, self.clock())
                        status, code = failure["status"], failure["error_code"]
                        hard = status in (400, 401, 402, 403, 404, 422) or code in {"insufficient_quota", "billing_hard_limit_reached", "invalid_api_key"}
                        model_state = state["models"].setdefault(profile.model_id, {})
                        if hard:
                            if status in (401, 402, 403) or code in {"insufficient_quota", "billing_hard_limit_reached", "invalid_api_key"}:
                                state["paused"] = failure
                            else:
                                model_state["paused"] = failure
                        else:
                            count = (state.get("account_failures", 0) if status == 429 else model_state.get("failures", 0)) + 1
                            if status == 429:
                                state["account_failures"] = count
                            else:
                                model_state["failures"] = count
                            delay = min(BACKOFF_CEILING, BACKOFF_BASE * (2 ** min(count - 1, 20))) + self.jitter()
                            until = max(self.clock() + delay, failure.get("retry_after_until") or 0)
                            if status == 429:
                                state["account_until"] = max(state["account_until"], until)
                            else:
                                model_state["until"] = max(model_state.get("until", 0), until)
                            failure["not_before"] = until
                        failure["retry_decision"] = ("provider_paused" if state["paused"] else "model_profile_paused") if hard else ("credential_cooldown" if status == 429 else "model_cooldown")
                        self._persist(directory, state)
                        retain({**common, "response": failure})
                        if hard:
                            raise ProviderUnavailable("AI provider requires credential, billing or request-profile correction", failure) from None
                        raise ProviderDeferred("AI provider failed; retained cooldown applies", MIN_SPACING, failure) from None
                    data = response.model_dump(mode="json")
                    retain({**common, "response": data})
                    # Ambiguous completions remain retained but cannot become a first-use
                    # checkpoint that the stricter reload path would later reject.
                    choice = response.choices[0] if len(response.choices) == 1 else None
                    raw = choice.message.content if choice else None
                    model_state = state["models"].setdefault(profile.model_id, {})
                    model_state["failures"] = 0
                    state["account_failures"] = 0
                    self._persist(directory, state)
                    return {**common, "raw_reply": raw, "finish_reason": choice.finish_reason if choice else None,
                            "reported_model": data.get("model"), "usage": data.get("usage")}
            except ProviderDeferred as error:
                if error.metadata:
                    delay = self._alternative_delay(pin, prompt, beat, minimum=MIN_SPACING)
                    if delay is None:
                        raise ProviderUnavailable("AI provider routes are paused or paid capacity is unavailable", error.metadata) from None
                    raise ProviderDeferred(str(error), delay, error.metadata) from None
                busy_scopes.add(profile.credential_scope)
                waits.append(error.delay)
            except ProviderUnavailable as error:
                if not error.metadata:
                    raise
                delay = self._alternative_delay(pin, prompt, beat)
                if delay is not None:
                    raise ProviderDeferred("AI provider path paused; configured alternative remains", delay, error.metadata) from None
                raise
        if waits:
            raise ProviderDeferred("AI provider routes are cooling or busy", min(waits))
        raise ProviderUnavailable("AI provider routes are paused or paid capacity is unavailable")
