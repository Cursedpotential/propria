"""Wire types for the repair workflow builder (engine /reference-import/repair/*).

Byline: Claude Code · Opus 5.5 · 2026-09-26.

These mirror `modules/engine/repairplan/contract.go` (shared contract:
`docs/pending-review/2026-09-25-repair-workflow-builder.md`). The Workbench
passes the engine's shapes through unchanged except for three things: a null
list becomes `[]`, a null `params` or `params_schema` becomes `{}`, and the
BFF adds its own `matter_mode` echo, exactly as the other Proffer routes do.

Request bodies keep only the bounds the BFF itself relies on (the preview
handle it proves TEST/REAL from, the matter mode, size guards). Everything
semantic is the engine's to judge, so its 400 and its named checks reach the
owner verbatim instead of a Workbench-side paraphrase.
"""

from __future__ import annotations

from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator

from app.types.matter_mode import MatterMode
from app.types.proffer import OpaquePreviewHandle

# The engine's WorkflowIDPattern: a repair run's workflow id.
RepairWorkflowID = Annotated[str, StringConstraints(pattern=r"^[A-Za-z0-9_-]{8,160}$")]
RepairCheckStatus = Literal["pass", "fail"]
RepairRunState = Literal["running", "completed", "failed"]
RepairStepState = Literal["pending", "running", "succeeded", "failed", "skipped"]


def _empty_list(value: Any) -> Any:
    return [] if value is None else value


def _empty_object(value: Any) -> Any:
    return {} if value is None else value


def _empty_text(value: Any) -> Any:
    return "" if value is None else value


# --- browser -> BFF ---------------------------------------------------------


class RepairPlanStep(BaseModel):
    """One registered Activity with its parameters, as the owner composed it."""

    model_config = ConfigDict(extra="forbid")

    step_id: str = Field(max_length=64)
    activity: str = Field(max_length=256)
    params: dict[str, Any] = Field(default_factory=dict)

    _params = field_validator("params", mode="before")(_empty_object)


class RepairPlan(BaseModel):
    """The owner's composed plan (engine `repairplan.Plan`).

    The preview handle is required here although the engine accepts null: the
    BFF proves the plan's TEST/REAL ownership from that handle's run, and a
    plan it cannot prove is refused rather than guessed.
    """

    model_config = ConfigDict(extra="forbid")

    plan_id: str = Field(max_length=96)
    source_ref: str = Field(max_length=2048)
    preview_handle: OpaquePreviewHandle
    matter_mode: MatterMode
    steps: list[RepairPlanStep] = Field(max_length=32)


class RepairProposeRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    source_ref: str = Field(min_length=1, max_length=2048)
    preview_handle: OpaquePreviewHandle


# --- engine -> BFF -> browser -------------------------------------------------


class _EngineModel(BaseModel):
    # Additive engine fields must not turn a working page into a 502.
    model_config = ConfigDict(extra="ignore")


class RepairTool(_EngineModel):
    id: str = Field(min_length=1)
    description: str = ""
    input_types: list[str] = Field(default_factory=list)
    output_types: list[str] = Field(default_factory=list)
    params_schema: dict[str, Any] = Field(default_factory=dict)
    writes: Literal["derived", "none"]
    needs_n8n: bool = False

    _lists = field_validator("input_types", "output_types", mode="before")(_empty_list)
    _schema = field_validator("params_schema", mode="before")(_empty_object)


class RepairToolsResponse(_EngineModel):
    tools: list[RepairTool] = Field(default_factory=list)
    matter_mode: MatterMode

    _tools = field_validator("tools", mode="before")(_empty_list)


class RepairProposedStep(_EngineModel):
    step_id: str
    activity: str
    params: dict[str, Any] = Field(default_factory=dict)

    _params = field_validator("params", mode="before")(_empty_object)


class RepairProposal(_EngineModel):
    signature: str = ""
    rationale: str = ""
    # steps == [] is the "wait: re-run after parse" option.
    steps: list[RepairProposedStep] = Field(default_factory=list)
    by: str = "rule"
    agent_available: bool = False

    _steps = field_validator("steps", mode="before")(_empty_list)


class RepairProposeResponse(_EngineModel):
    signature: str = ""
    proposals: list[RepairProposal] = Field(default_factory=list)
    agent_available: bool = False
    matter_mode: MatterMode

    _proposals = field_validator("proposals", mode="before")(_empty_list)


class RepairCheck(_EngineModel):
    rule: str
    status: RepairCheckStatus
    reason: str = ""


class RepairValidateResponse(_EngineModel):
    ok: bool
    checks: list[RepairCheck] = Field(default_factory=list)
    matter_mode: MatterMode

    _checks = field_validator("checks", mode="before")(_empty_list)


class RepairRunResponse(_EngineModel):
    workflow_id: RepairWorkflowID
    run_id: str = Field(min_length=1)
    matter_mode: MatterMode


class RepairRunRefused(BaseModel):
    """The 422 body when the engine refuses to start a plan: its detail and full checklist."""

    model_config = ConfigDict(extra="forbid")

    detail: str
    ok: Literal[False] = False
    checks: list[RepairCheck] = Field(default_factory=list)

    _checks = field_validator("checks", mode="before")(_empty_list)


class RepairStepStatus(_EngineModel):
    step_id: str
    activity: str
    status: RepairStepState
    receipt_ref: str = ""  # "" until the step's receipt is recorded
    output_ref: str | None = None
    output_sha256: str | None = None
    summary: Any = None
    reason: str = ""

    _receipt = field_validator("receipt_ref", mode="before")(_empty_text)


class RepairRunStatus(_EngineModel):
    workflow_id: RepairWorkflowID
    plan_id: str = ""
    preview_handle: str | None = None
    matter_mode: MatterMode
    status: RepairRunState
    reason: str = ""
    steps: list[RepairStepStatus] = Field(default_factory=list)
    reentry_preview_handle: str | None = None
    reentry_batch_id: str | None = None
    reentry_receipt_ref: str | None = None
    checks: list[RepairCheck] = Field(default_factory=list)

    _lists = field_validator("steps", "checks", mode="before")(_empty_list)
