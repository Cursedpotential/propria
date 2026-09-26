"""Static provider trust grid from the CAT6 cited table.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Not a privilege legal conclusion. Not PACER. Local Ollama is not a
trust posture — the owner has no local inference. `ollama-cloud` is
the Cloud API via Portkey.
"""

from __future__ import annotations

from enum import Enum

from pydantic import BaseModel, model_validator

GRID_DISCLAIMER = (
    "Cited provider-terms snapshot, not a privilege determination, "
    "work-product claim, confidentiality holding, or other legal "
    "conclusion. A court decides privilege. Consumer Claude/ChatGPT "
    "are blocked in Confidential Mode. No silent fallback. No local "
    "Ollama as a trust posture. No PACER."
)


class ProviderRole(str, Enum):
    PRIMARY = "primary"
    SECONDARY = "secondary"
    EMBED = "embed"
    OPTIONAL = "optional"
    BLOCK = "block"


class ProviderRow(BaseModel):
    id: str
    display_name: str
    train: bool
    retain: str
    confidential_eligible: bool
    role: ProviderRole
    notes: str
    use: str


class ProviderGrid(BaseModel):
    rows: list[ProviderRow]
    eligible_confidential_models: list[str]
    pacer: bool = False
    local_ollama_as_trust_posture: bool = False
    court_safe: bool = False
    legal_conclusion: bool = False
    source: str = "docs/reports/CAT6-privilege-routing.md"
    disclaimer: str = GRID_DISCLAIMER

    @model_validator(mode="after")
    def never_a_legal_conclusion(self) -> ProviderGrid:
        self.pacer = False
        self.local_ollama_as_trust_posture = False
        self.court_safe = False
        self.legal_conclusion = False
        return self


PROVIDER_ROWS: tuple[ProviderRow, ...] = (
    ProviderRow(
        id="ollama-cloud",
        display_name="Ollama Cloud (via Portkey)",
        train=False,
        retain="transient",
        confidential_eligible=True,
        role=ProviderRole.PRIMARY,
        use="Primary Confidential Mode route",
        notes=(
            "Cloud API only — not a consumer chat UI, not local Ollama. "
            "Cited policy: never train; not stored beyond fulfilling the request."
        ),
    ),
    ProviderRow(
        id="openrouter-zdr",
        display_name="OpenRouter ZDR + data_collection=deny (via Portkey)",
        train=False,
        retain="no_if_honoured",
        confidential_eligible=True,
        role=ProviderRole.SECONDARY,
        use="Secondary Confidential Mode route",
        notes=(
            "Eligible only with both ZDR flags and no plugin/web-search tools. "
            "If no ZDR endpoint can serve, fail — do not fall through."
        ),
    ),
    ProviderRow(
        id="nim-hosted",
        display_name="NVIDIA NIM hosted",
        train=False,
        retain="unclear",
        confidential_eligible=False,
        role=ProviderRole.EMBED,
        use="Embed/rerank only; not long privileged prose",
        notes=(
            "Training claim is for hosted NIM inference. Retention is less "
            "clear. Not Confidential-eligible for long privileged prose "
            "until the hosted NIM ToS is snapshotted."
        ),
    ),
    ProviderRow(
        id="venice-private",
        display_name="Venice Private / TEE",
        train=False,
        retain="no",
        confidential_eligible=True,
        role=ProviderRole.OPTIONAL,
        use="Optional signup; never a third-party relay",
        notes=(
            "Eligible only on Venice-hosted / Private / TEE modes. "
            "Relaying to Anthropic/OpenAI leaves the no-log perimeter — block that."
        ),
    ),
    ProviderRow(
        id="claude-consumer",
        display_name="Claude.ai consumer",
        train=True,
        retain="yes",
        confidential_eligible=False,
        role=ProviderRole.BLOCK,
        use="Block in Confidential Mode",
        notes=(
            "Not ZDR-equivalent. Consumer ToS is not a reasonable expectation "
            "of confidentiality (Heppner / ABA 1.6). No silent fallback."
        ),
    ),
    ProviderRow(
        id="chatgpt-consumer",
        display_name="ChatGPT consumer",
        train=True,
        retain="yes",
        confidential_eligible=False,
        role=ProviderRole.BLOCK,
        use="Block in Confidential Mode",
        notes=(
            "Not ZDR-equivalent. Consumer ToS is not a reasonable expectation "
            "of confidentiality (Heppner / ABA 1.6). No silent fallback."
        ),
    ),
)

_ROWS_BY_ID: dict[str, ProviderRow] = {row.id: row for row in PROVIDER_ROWS}


def provider_row(model: str) -> ProviderRow | None:
    return _ROWS_BY_ID.get((model or "").strip())


def eligible_confidential_models() -> list[str]:
    """Verified chat ids, primary then secondary then optional Venice."""
    order = (
        ProviderRole.PRIMARY,
        ProviderRole.SECONDARY,
        ProviderRole.OPTIONAL,
    )
    return [
        row.id
        for role in order
        for row in PROVIDER_ROWS
        if row.role is role and row.confidential_eligible
    ]


def confidential_blocked_reason(model: str) -> str | None:
    """None if `model` may be used for Confidential Mode chat."""
    key = (model or "").strip()
    if key in eligible_confidential_models():
        return None
    row = provider_row(key)
    if row is None:
        label = key or "unknown"
        return (
            f"{label} is not on the verified Confidential Mode list. "
            "No silent fallback. Hard-block."
        )
    if row.role is ProviderRole.BLOCK:
        return (
            f"{row.display_name} is a consumer app, not ZDR-equivalent "
            "(Heppner / ABA 1.6). Blocked in Confidential Mode. No silent fallback."
        )
    if row.role is ProviderRole.EMBED:
        return (
            f"{row.display_name} is embed/rerank only; not eligible for "
            "long privileged prose. No silent fallback."
        )
    return f"{row.display_name} is not Confidential-eligible. No silent fallback."


def provider_grid() -> ProviderGrid:
    return ProviderGrid(
        rows=list(PROVIDER_ROWS),
        eligible_confidential_models=eligible_confidential_models(),
        pacer=False,
        local_ollama_as_trust_posture=False,
        court_safe=False,
        legal_conclusion=False,
    )
